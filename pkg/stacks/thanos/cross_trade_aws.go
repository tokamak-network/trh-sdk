package thanos

import (
	"context"
	"encoding/json"
	"errors"
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
		L1CrossTradeProxy:  "0xfea37d39bec823d503ed6fb9d3a6e151190821fb",
		L2toL2CrossTradeL1: "0xd038d89655f106d88c5bd56a9442d9ecee675c1c",
	},
}

// crossTradeReleaseName generates a deterministic Helm release name using the L2 chain ID.
func crossTradeReleaseName(l2ChainID uint64) string {
	return fmt.Sprintf("cross-trade-%d", l2ChainID)
}

// crossTradeL2L2ReleaseName generates a deterministic Helm release name for the L2→L2 bridge.
// Kept for UninstallCrossTradeAWS backward compatibility with clusters that have the old split releases.
func crossTradeL2L2ReleaseName(l2ChainID uint64) string {
	return fmt.Sprintf("cross-trade-l2l2-%d", l2ChainID)
}

// AutoInstallCrossTradeAWSOutput contains deployed contract addresses and the dApp URL
// produced by AutoInstallCrossTradeAWS. Consumed by trh-backend to update stack metadata.
type AutoInstallCrossTradeAWSOutput struct {
	L2CrossTradeProxy     string // L2→L1 bridge proxy on user's L2
	L2toL2CrossTradeProxy string // L2→L2 bridge proxy on user's L2
	L1CrossTradeProxy     string // L1-side L2→L1 bridge proxy (shared Sepolia contract)
	L2toL2CrossTradeL1    string // L1-side L2→L2 bridge proxy (shared Sepolia contract)
	DAppURL               string // CrossTrade dApp ALB URL (single release serving both L2→L1 and L2→L2)
}

// AutoInstallCrossTradeAWS deploys CrossTrade L2 contracts via L1 OptimismPortal depositTransaction
// and then installs both CrossTrade dApp Helm releases (L2→L1 and L2→L2).
//
// This is the fully-automated AWS path — equivalent to what 'trh install cross-trade' does manually.
// It uses DeployCrossTradeLocal (Deposit Tx) for L2 contract deployment so the deployer does not
// need pre-funded L2 ETH (contrast with the forge --broadcast path which requires L2 ETH).
func (t *ThanosStack) AutoInstallCrossTradeAWS(ctx context.Context) (*AutoInstallCrossTradeAWSOutput, error) {
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID

	l1CT, ok := l1CrossTradeAddresses[l1ChainID]
	if !ok {
		return nil, fmt.Errorf("CrossTrade L1 contracts not available for L1 chain %d; use 'trh install cross-trade' for manual setup", l1ChainID)
	}

	l2RPC := t.deployConfig.L2RpcUrl
	if l2RPC == "" {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: L2RpcUrl not set (AWS deployment must complete first)")
	}

	deployedContracts, err := t.readDeploymentContracts()
	if err != nil {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: failed to read deployed contracts: %w", err)
	}
	if deployedContracts.OptimismPortalProxy == "" {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: OptimismPortalProxy is empty in deploy-output.json")
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
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: L2 contract deployment failed: %w", err)
	}
	t.logger.Infof("L2 CrossTrade contracts deployed: proxy=%s l2l2proxy=%s",
		localOutput.L2CrossTradeProxy, localOutput.L2toL2CrossTradeProxy)

	// Register the deployed L2 chain on the shared L1CrossTradeProxy.
	// DeployCrossTradeLocal only sets up the L2 side (setChainInfo on L2CrossTradeProxy).
	// The L1 side must be initialized separately so that L1CrossTradeProxy.chainData(l2ChainId)
	// returns (crossDomainMessenger, l2CrossTradeProxy) — without this, provideCT always reverts.
	if err := setL1CrossTradeChainInfo(
		ctx,
		t.logger,
		t.deployConfig.L1RPCURL,
		t.deployConfig.AdminPrivateKey,
		l1CT.L1CrossTradeProxy,
		deployedContracts.L1CrossDomainMessengerProxy,
		localOutput.L2CrossTradeProxy,
		l2ChainID,
		t.deployConfig.L1ChainID,
	); err != nil {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: failed to set L1 chain info: %w", err)
	}
	t.logger.Info("✅ L1CrossTradeProxy chain info registered")

	// Register the deployed L2 chain on L2toL2CrossTradeL1 (L2→L2 direction).
	// setL1CrossTradeChainInfo above covers L2→L1; this call covers L2→L2.
	// Without this, L2toL2CrossTradeL1.chainData(l2ChainId) returns zero addresses
	// and any L2→L2 bridge request reverts.
	if err := setL2toL2CrossTradeL1ChainInfo(
		ctx,
		t.logger,
		t.deployConfig.L1RPCURL,
		t.deployConfig.AdminPrivateKey,
		l1CT.L2toL2CrossTradeL1,
		deployedContracts.L1CrossDomainMessengerProxy,
		localOutput.L2toL2CrossTradeProxy,
		deployedContracts.L1StandardBridgeProxy,
		l2ChainID,
		l1ChainID,
	); err != nil {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: failed to set L2toL2 chain info: %w", err)
	}
	t.logger.Info("✅ L2toL2CrossTradeL1 chain info registered")

	// Deploy a single CrossTrade dApp Helm release that handles both L2→L1 and L2→L2 modes.
	dAppURL, err := t.installCrossTradeHelmAWS(ctx, l1CT.L1CrossTradeProxy, localOutput.L2CrossTradeProxy, l1CT.L2toL2CrossTradeL1, localOutput.L2toL2CrossTradeProxy, l2RPC)
	if err != nil {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: Helm dApp deployment failed: %w", err)
	}

	return &AutoInstallCrossTradeAWSOutput{
		L2CrossTradeProxy:     localOutput.L2CrossTradeProxy,
		L2toL2CrossTradeProxy: localOutput.L2toL2CrossTradeProxy,
		L1CrossTradeProxy:     l1CT.L1CrossTradeProxy,
		L2toL2CrossTradeL1:    l1CT.L2toL2CrossTradeL1,
		DAppURL:               dAppURL,
	}, nil
}

// buildCrossTradeL2L1Config builds the chain config map for NEXT_PUBLIC_CHAIN_CONFIG_L2_L1.
// L2 tokens include DestinationChains=[l1ChainID] so the dApp renders the L2 as a selectable source.
func buildCrossTradeL2L1Config(
	l1ChainID uint64, l1RPCURL, l1CTProxy string,
	l2ChainID uint64, l2ChainName, l2RPCURL, l2CTProxy string,
) map[string]types.CrossTradeChainConfig {
	l1Config := constants.L1ChainConfigurations[l1ChainID]
	l1Key := fmt.Sprintf("%d", l1ChainID)
	l2Key := fmt.Sprintf("%d", l2ChainID)
	l1CTProxyCopy := l1CTProxy
	l2CTProxyCopy := l2CTProxy
	return map[string]types.CrossTradeChainConfig{
		l1Key: {
			Name:              l1Config.ChainName,
			DisplayName:       l1Config.ChainName,
			Contracts:         types.CrossTradeContracts{L1CrossTrade: &l1CTProxyCopy},
			RPCURL:            l1RPCURL,
			NativeTokenName:   l1Config.NativeTokenName,
			NativeTokenSymbol: l1Config.NativeTokenSymbol,
			Tokens: []*types.RegisterToken{
				{Name: "ETH", Address: "0x0000000000000000000000000000000000000000", DestinationChains: []uint64{}},
				{Name: "USDC", Address: l1Config.USDCAddress, DestinationChains: []uint64{}},
			},
		},
		l2Key: {
			Name:              l2Key,
			DisplayName:       l2ChainName,
			Contracts:         types.CrossTradeContracts{L2CrossTrade: &l2CTProxyCopy},
			RPCURL:            l2RPCURL,
			NativeTokenName:   "Tokamak Network",
			NativeTokenSymbol: "TON",
			Tokens: []*types.RegisterToken{
				{Name: "ETH", Address: "0x0000000000000000000000000000000000000000", DestinationChains: []uint64{l1ChainID}},
				{Name: "USDC", Address: constants.USDCAddress, DestinationChains: []uint64{l1ChainID}},
			},
		},
	}
}

// buildCrossTradeL2L2Config builds the chain config map for NEXT_PUBLIC_CHAIN_CONFIG_L2_L2.
// Always includes Thanos Sepolia (111551119090) as the fixed bridge partner.
// Custom L2 tokens point to Thanos Sepolia; Thanos tokens have empty DestinationChains
// because the Thanos Sepolia L2toL2CTProxy has not registered the new L2's chainId.
func buildCrossTradeL2L2Config(
	l1ChainID uint64, l1RPCURL, l2toL2CrossTradeL1 string,
	l2ChainID uint64, l2ChainName, l2RPCURL, l2toL2CTProxy string,
) map[string]types.CrossTradeChainConfig {
	l1Config := constants.L1ChainConfigurations[l1ChainID]
	l1Key := fmt.Sprintf("%d", l1ChainID)
	l2Key := fmt.Sprintf("%d", l2ChainID)
	thanosSepKey := fmt.Sprintf("%d", crossTradeThanosSepolia)
	l2toL2L1Copy := l2toL2CrossTradeL1
	l2toL2CTCopy := l2toL2CTProxy
	thanosCTCopy := crossTradeThanosSepL2CTProxy
	return map[string]types.CrossTradeChainConfig{
		l1Key: {
			Name:              l1Config.ChainName,
			DisplayName:       l1Config.ChainName,
			Contracts:         types.CrossTradeContracts{L1CrossTrade: &l2toL2L1Copy},
			RPCURL:            l1RPCURL,
			NativeTokenName:   l1Config.NativeTokenName,
			NativeTokenSymbol: l1Config.NativeTokenSymbol,
			Tokens: []*types.RegisterToken{
				{Name: "ETH", Address: "0x0000000000000000000000000000000000000000", DestinationChains: []uint64{}},
				{Name: "USDC", Address: l1Config.USDCAddress, DestinationChains: []uint64{}},
			},
		},
		l2Key: {
			Name:              l2Key,
			DisplayName:       l2ChainName,
			Contracts:         types.CrossTradeContracts{L2CrossTrade: &l2toL2CTCopy},
			RPCURL:            l2RPCURL,
			NativeTokenName:   "Tokamak Network",
			NativeTokenSymbol: "TON",
			Tokens: []*types.RegisterToken{
				{Name: "ETH", Address: "0x0000000000000000000000000000000000000000", DestinationChains: []uint64{crossTradeThanosSepolia}},
				{Name: "USDC", Address: constants.USDCAddress, DestinationChains: []uint64{crossTradeThanosSepolia}},
			},
		},
		thanosSepKey: {
			Name:              "Thanos Sepolia",
			DisplayName:       "Thanos Sepolia",
			Contracts:         types.CrossTradeContracts{L2CrossTrade: &thanosCTCopy},
			RPCURL:            crossTradeThanosSepRPCURL,
			NativeTokenName:   "Tokamak Network",
			NativeTokenSymbol: "TON",
			Tokens: []*types.RegisterToken{
				{Name: "ETH", Address: "0x4200000000000000000000000000000000000486", DestinationChains: []uint64{}},
				{Name: "TON", Address: "0xDeadDeAddeAddEAddeadDEaDDEAdDeaDDeAD0000", DestinationChains: []uint64{}},
				{Name: "USDC", Address: "0x4200000000000000000000000000000000000778", DestinationChains: []uint64{}},
			},
		},
	}
}

// installCrossTradeHelmAWS installs a single CrossTrade Helm release that serves both
// L2→L1 and L2→L2 modes. The dApp auto-selects the mode based on the destination chain,
// so a single instance with both chain configs handles all CrossTrade transactions.
func (t *ThanosStack) installCrossTradeHelmAWS(
	ctx context.Context,
	l1CTProxy, l2CTProxy, l2toL2CrossTradeL1, l2toL2CTProxy, l2RPC string,
) (string, error) {
	namespace := t.deployConfig.K8s.Namespace
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID
	configDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/cross-trade", t.deploymentPath)

	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create config dir: %w", err)
	}

	l2l1ChainCfg := buildCrossTradeL2L1Config(l1ChainID, t.deployConfig.L1RPCURL, l1CTProxy, l2ChainID, t.deployConfig.ChainName, l2RPC, l2CTProxy)
	l2l1ConfigJSON, err := json.Marshal(l2l1ChainCfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal L2→L1 chain config: %w", err)
	}

	l2l2ChainCfg := buildCrossTradeL2L2Config(l1ChainID, t.deployConfig.L1RPCURL, l2toL2CrossTradeL1, l2ChainID, t.deployConfig.ChainName, l2RPC, l2toL2CTProxy)
	l2l2ConfigJSON, err := json.Marshal(l2l2ChainCfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal L2→L2 chain config: %w", err)
	}

	helmEnv := types.CrossTradeEnvConfig{
		NextPublicProjectID: "568b8d3d0528e743b0e2c6c92f54d721",
		L2L1Config:          string(l2l1ConfigJSON),
		L2L2Config:          string(l2l2ConfigJSON),
	}

	url, err := t.installOneHelmRelease(ctx, namespace, chartPath,
		crossTradeReleaseName(l2ChainID),
		filepath.Join(configDir, "cross-trade-values.yaml"),
		"cross-trade",
		helmEnv)
	if err != nil {
		return "", fmt.Errorf("CrossTrade Helm install failed: %w", err)
	}

	return url, nil
}

// installOneHelmRelease writes the values YAML, runs helm install, and polls for the ALB URL.
func (t *ThanosStack) installOneHelmRelease(
	ctx context.Context,
	namespace, chartPath, releaseName, valuesPath, albGroupName string,
	env types.CrossTradeEnvConfig,
) (string, error) {
	cfg := types.CrossTradeConfig{}
	cfg.CrossTrade.Env = env
	cfg.CrossTrade.Ingress = types.Ingress{
		Enabled:   true,
		ClassName: "alb",
		Annotations: map[string]string{
			"alb.ingress.kubernetes.io/target-type":  "ip",
			"alb.ingress.kubernetes.io/scheme":       "internet-facing",
			"alb.ingress.kubernetes.io/listen-ports": `[{"HTTP": 80}]`,
			"alb.ingress.kubernetes.io/group.name":   albGroupName,
		},
		TLS: types.TLS{Enabled: false},
	}

	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal values YAML for %s: %w", releaseName, err)
	}
	if err := os.WriteFile(valuesPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write values file for %s: %w", releaseName, err)
	}

	_, err = utils.ExecuteCommand(ctx, "helm", []string{
		"install", releaseName, chartPath,
		"--values", valuesPath,
		"--namespace", namespace,
	}...)
	if err != nil {
		return "", fmt.Errorf("helm install %s failed: %w", releaseName, err)
	}

	t.logger.Infof("  ↳ cross-trade: waiting for %s ALB ingress...", releaseName)
	for {
		ingresses, err := utils.GetAddressByIngress(ctx, namespace, releaseName)
		if err != nil {
			return "", fmt.Errorf("failed to get ingress address for %s: %w", releaseName, err)
		}
		if len(ingresses) > 0 {
			url := fmt.Sprintf("http://%s", ingresses[0])
			t.logger.Infof("✅ CrossTrade %s deployed at: %s", releaseName, url)
			return url, nil
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(15 * time.Second):
		}
	}
}

// UninstallCrossTradeAWS removes all CrossTrade Helm releases from the K8s cluster.
// It handles both the L2→L1 release (cross-trade-{l2ChainID}) and the L2→L2 release
// (cross-trade-l2l2-{l2ChainID}), with prefix-scan fallback for legacy timestamp-based names.
// Returns nil if no release is found (not installed = not an error).
func (t *ThanosStack) UninstallCrossTradeAWS(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return nil
	}

	namespace := t.deployConfig.K8s.Namespace
	l2ChainID := t.deployConfig.L2ChainID

	// Collect all CrossTrade releases: try deterministic names first, then prefix scan.
	var allReleases []string
	for _, exactName := range []string{crossTradeReleaseName(l2ChainID), crossTradeL2L2ReleaseName(l2ChainID)} {
		found, err := utils.FilterHelmReleases(ctx, namespace, exactName)
		if err != nil {
			t.logger.Warnf("Could not list CrossTrade helm releases by name %s: %v", exactName, err)
		}
		allReleases = append(allReleases, found...)
	}

	// Fallback: prefix scan for legacy timestamp-based names when no exact matches found.
	if len(allReleases) == 0 {
		found, err := utils.FilterHelmReleases(ctx, namespace, "cross-trade-")
		if err != nil {
			t.logger.Warnf("Could not list CrossTrade helm releases by prefix: %v", err)
			return nil
		}
		allReleases = found
	}

	if len(allReleases) == 0 {
		t.logger.Info("No CrossTrade Helm release found, skipping uninstall")
		return nil
	}

	var errs []error
	for _, release := range allReleases {
		t.logger.Infof("Uninstalling CrossTrade Helm release: %s in namespace: %s", release, namespace)
		if err := utils.UninstallHelmRelease(ctx, namespace, release); err != nil {
			t.logger.Warnf("Failed to uninstall CrossTrade release %s: %v", release, err)
			errs = append(errs, fmt.Errorf("helm uninstall %s: %w", release, err))
		}
	}

	if len(errs) == 0 {
		t.logger.Info("✅ CrossTrade uninstalled successfully")
	}
	return errors.Join(errs...)
}
