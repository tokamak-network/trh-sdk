package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

// l1CrossTradeAddresses holds the Tokamak-deployed L1 CrossTrade contract addresses per L1 chain.
// These are shared infrastructure — every L2 chain on a given L1 uses the same L1-side contracts.
var l1CrossTradeAddresses = map[uint64]struct {
	L1CrossTradeProxy  string // L1CrossTrade proxy (L2→L1 bridge)
	L2toL2CrossTradeL1 string // L2toL2CrossTradeProxyL1 (L2→L2 bridge)
}{
	constants.EthereumSepoliaChainID: {
		L1CrossTradeProxy:  "0xf3473E20F1d9EB4468C72454a27aA1C65B67AB35",
		L2toL2CrossTradeL1: "0xDa2CbF69352cB46d9816dF934402b421d93b6BC2",
	},
}

// autoInstallCrossTradeAWS deploys CrossTrade L2 contracts via L1 OptimismPortal depositTransaction
// and then installs the CrossTrade dApp via Helm chart.
//
// This is the fully-automated AWS path — equivalent to what 'trh install cross-trade' does manually.
// It reuses DeployCrossTradeLocal (Deposit Tx) for L2 contract deployment and implements the
// Helm chart deploy inline to avoid the L2ChainConfig slice-indexing bug in DeployCrossTradeApplication.
func (t *ThanosStack) autoInstallCrossTradeAWS(ctx context.Context) error {
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID

	l1CT, ok := l1CrossTradeAddresses[l1ChainID]
	if !ok {
		return fmt.Errorf("CrossTrade L1 contracts not available for L1 chain %d; use 'trh install cross-trade' for manual setup", l1ChainID)
	}

	l2RPC := t.deployConfig.L2RpcUrl
	if l2RPC == "" {
		return fmt.Errorf("autoInstallCrossTradeAWS: L2RpcUrl not set (AWS deployment must complete first)")
	}

	deployedContracts, err := t.readDeploymentContracts()
	if err != nil {
		return fmt.Errorf("autoInstallCrossTradeAWS: failed to read deployed contracts: %w", err)
	}
	if deployedContracts.OptimismPortalProxy == "" {
		return fmt.Errorf("autoInstallCrossTradeAWS: OptimismPortalProxy is empty in deploy-output.json")
	}

	// Deploy L2 CrossTrade contracts via L1 Deposit Tx.
	localInput := &DeployCrossTradeLocalInput{
		L1RPCUrl:             t.deployConfig.L1RPCURL,
		L1ChainID:            l1ChainID,
		DeployerPrivateKey:   t.deployConfig.AdminPrivateKey,
		L2RPCUrl:             l2RPC,
		L2ChainID:            l2ChainID,
		OptimismPortalProxy:  deployedContracts.OptimismPortalProxy,
		CrossDomainMessenger: deployedContracts.L1CrossDomainMessengerProxy,
		L1CrossTradeProxy:    l1CT.L1CrossTradeProxy,
		L2toL2CrossTradeL1:   l1CT.L2toL2CrossTradeL1,
		SupportedTokens:      []TokenPair{},
	}

	t.logger.Info("  ↳ cross-trade: deploying L2 contracts via deposit tx (this takes 20-40 min)...")
	localOutput, err := t.DeployCrossTradeLocal(ctx, localInput)
	if err != nil {
		return fmt.Errorf("autoInstallCrossTradeAWS: L2 contract deployment failed: %w", err)
	}
	t.logger.Infof("L2 CrossTrade contracts deployed: proxy=%s l2l2proxy=%s",
		localOutput.L2CrossTradeProxy, localOutput.L2toL2CrossTradeProxy)

	// Deploy CrossTrade dApp via Helm chart.
	if err := t.installCrossTradeHelmAWS(ctx, l1CT.L1CrossTradeProxy, localOutput.L2CrossTradeProxy, l2RPC); err != nil {
		return fmt.Errorf("autoInstallCrossTradeAWS: Helm dApp deployment failed: %w", err)
	}

	return nil
}

// installCrossTradeHelmAWS builds the CrossTrade Helm values YAML and runs helm install.
// This reimplements the Helm portion of DeployCrossTradeApplication directly to avoid the
// L2ChainConfig[l2ChainID] slice-indexing bug in the existing function.
func (t *ThanosStack) installCrossTradeHelmAWS(ctx context.Context, l1CTProxy, l2CTProxy, l2RPC string) error {
	namespace := t.deployConfig.K8s.Namespace
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID
	l1Config := constants.L1ChainConfigurations[l1ChainID]

	chainConfig := map[string]types.CrossTradeChainConfig{
		fmt.Sprintf("%d", l1ChainID): {
			Name:        l1Config.ChainName,
			DisplayName: l1Config.ChainName,
			Contracts:   types.CrossTradeContracts{L1CrossTrade: &l1CTProxy},
			RPCURL:      t.deployConfig.L1RPCURL,
			Tokens: types.CrossTradeTokens{
				ETH:  "0x0000000000000000000000000000000000000000",
				USDC: l1Config.USDCAddress,
				USDT: l1Config.USDTAddress,
				TON:  l1Config.TON,
			},
		},
		fmt.Sprintf("%d", l2ChainID): {
			Name:        fmt.Sprintf("%d", l2ChainID),
			DisplayName: t.deployConfig.ChainName,
			Contracts:   types.CrossTradeContracts{L2CrossTrade: &l2CTProxy},
			RPCURL:      l2RPC,
			Tokens: types.CrossTradeTokens{
				ETH:  constants.ETH,
				USDC: constants.USDCAddress,
				TON:  constants.TON,
			},
		},
	}

	chainConfigJSON, err := json.Marshal(chainConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal chain config: %w", err)
	}

	crossTradeConfig := types.CrossTradeConfig{}
	crossTradeConfig.CrossTrade.Env.NextPublicProjectID = "568b8d3d0528e743b0e2c6c92f54d721"
	crossTradeConfig.CrossTrade.Env.NextPublicChainConfig = string(chainConfigJSON)
	crossTradeConfig.CrossTrade.Ingress = types.Ingress{
		Enabled:   true,
		ClassName: "alb",
		Annotations: map[string]string{
			"alb.ingress.kubernetes.io/target-type":  "ip",
			"alb.ingress.kubernetes.io/scheme":       "internet-facing",
			"alb.ingress.kubernetes.io/listen-ports": "[{\"HTTP\": 80}]",
			"alb.ingress.kubernetes.io/group.name":   "cross-trade",
		},
		TLS: types.TLS{Enabled: false},
	}

	data, err := yaml.Marshal(&crossTradeConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal cross-trade values YAML: %w", err)
	}

	configDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	valuesPath := filepath.Join(configDir, "cross-trade-values.yaml")
	if err := os.WriteFile(valuesPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cross-trade-values.yaml: %w", err)
	}

	helmReleaseName := fmt.Sprintf("cross-trade-%d", time.Now().Unix())
	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/cross-trade", t.deploymentPath)

	_, err = utils.ExecuteCommand(ctx, "helm", []string{
		"install",
		helmReleaseName,
		chartPath,
		"--values", valuesPath,
		"--namespace", namespace,
	}...)
	if err != nil {
		return fmt.Errorf("helm install cross-trade failed: %w", err)
	}

	t.logger.Info("  ↳ cross-trade: waiting for ALB ingress...")
	for {
		ingresses, err := utils.GetAddressByIngress(ctx, namespace, helmReleaseName)
		if err != nil {
			return fmt.Errorf("failed to get ingress address: %w", err)
		}
		if len(ingresses) > 0 {
			t.logger.Infof("✅ CrossTrade dApp deployed at: http://%s", ingresses[0])
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}
