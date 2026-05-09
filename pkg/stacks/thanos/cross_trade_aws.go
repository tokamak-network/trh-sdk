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
		L1CrossTradeProxy:  "0xf3473E20F1d9EB4468C72454a27aA1C65B67AB35",
		L2toL2CrossTradeL1: "0xDa2CbF69352cB46d9816dF934402b421d93b6BC2",
	},
}

// crossTradeReleaseName generates a deterministic Helm release name using the L2 chain ID.
func crossTradeReleaseName(l2ChainID uint64) string {
	return fmt.Sprintf("cross-trade-%d", l2ChainID)
}

// crossTradeL2L2ReleaseName generates a deterministic Helm release name for the L2→L2 bridge.
func crossTradeL2L2ReleaseName(l2ChainID uint64) string {
	return fmt.Sprintf("cross-trade-l2l2-%d", l2ChainID)
}

// AutoInstallCrossTradeAWSOutput contains deployed contract addresses and dApp URLs
// produced by AutoInstallCrossTradeAWS. Consumed by trh-backend to update stack metadata.
type AutoInstallCrossTradeAWSOutput struct {
	L2CrossTradeProxy     string // L2→L1 bridge proxy on user's L2
	L2toL2CrossTradeProxy string // L2→L2 bridge proxy on user's L2
	L1CrossTradeProxy     string // L1-side L2→L1 bridge proxy (shared Sepolia contract)
	L2toL2CrossTradeL1    string // L1-side L2→L2 bridge proxy (shared Sepolia contract)
	L2L1DAppURL           string // CrossTrade L2→L1 dApp ALB URL
	L2L2DAppURL           string // CrossTrade L2→L2 dApp ALB URL
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

	// Deploy both CrossTrade dApp Helm releases and retrieve their ALB URLs.
	l2l1URL, l2l2URL, err := t.installCrossTradeHelmAWS(ctx, l1CT.L1CrossTradeProxy, localOutput.L2CrossTradeProxy, l1CT.L2toL2CrossTradeL1, localOutput.L2toL2CrossTradeProxy, l2RPC)
	if err != nil {
		return nil, fmt.Errorf("autoInstallCrossTradeAWS: Helm dApp deployment failed: %w", err)
	}

	return &AutoInstallCrossTradeAWSOutput{
		L2CrossTradeProxy:     localOutput.L2CrossTradeProxy,
		L2toL2CrossTradeProxy: localOutput.L2toL2CrossTradeProxy,
		L1CrossTradeProxy:     l1CT.L1CrossTradeProxy,
		L2toL2CrossTradeL1:    l1CT.L2toL2CrossTradeL1,
		L2L1DAppURL:           l2l1URL,
		L2L2DAppURL:           l2l2URL,
	}, nil
}

// installCrossTradeHelmAWS installs both CrossTrade Helm releases (L2→L1 and L2→L2)
// and returns their ALB URLs. This reimplements the Helm portion of DeployCrossTradeApplication
// directly to avoid the L2ChainConfig[l2ChainID] slice-indexing bug in the existing function.
func (t *ThanosStack) installCrossTradeHelmAWS(
	ctx context.Context,
	l1CTProxy, l2CTProxy, l2toL2CrossTradeL1, l2toL2CTProxy, l2RPC string,
) (l2l1URL, l2l2URL string, err error) {
	namespace := t.deployConfig.K8s.Namespace
	l1ChainID := t.deployConfig.L1ChainID
	l2ChainID := t.deployConfig.L2ChainID
	l1Config := constants.L1ChainConfigurations[l1ChainID]
	configDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/cross-trade", t.deploymentPath)

	if err := os.MkdirAll(configDir, os.ModePerm); err != nil {
		return "", "", fmt.Errorf("failed to create config dir: %w", err)
	}

	l1ChainKey := fmt.Sprintf("%d", l1ChainID)
	l2ChainKey := fmt.Sprintf("%d", l2ChainID)

	// --- L2→L1 release ---
	l2l1ChainCfg := map[string]types.CrossTradeChainConfig{
		l1ChainKey: {
			Name:        l1Config.ChainName,
			DisplayName: l1Config.ChainName,
			Contracts:   types.CrossTradeContracts{L1CrossTrade: &l1CTProxy},
			RPCURL:      t.deployConfig.L1RPCURL,
		},
		l2ChainKey: {
			Name:        l2ChainKey,
			DisplayName: t.deployConfig.ChainName,
			Contracts:   types.CrossTradeContracts{L2CrossTrade: &l2CTProxy},
			RPCURL:      l2RPC,
		},
	}
	l2l1ConfigJSON, err := json.Marshal(l2l1ChainCfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal L2→L1 chain config: %w", err)
	}
	l2l1HelmEnv := types.CrossTradeEnvConfig{
		NextPublicProjectID: "568b8d3d0528e743b0e2c6c92f54d721",
		L2L1Config:          string(l2l1ConfigJSON),
	}
	l2l1URL, err = t.installOneHelmRelease(ctx, namespace, chartPath,
		crossTradeReleaseName(l2ChainID),
		filepath.Join(configDir, "cross-trade-values.yaml"),
		"cross-trade",
		l2l1HelmEnv)
	if err != nil {
		return "", "", fmt.Errorf("L2→L1 Helm install failed: %w", err)
	}

	// --- L2→L2 release ---
	l2l2ChainCfg := map[string]types.CrossTradeChainConfig{
		l1ChainKey: {
			Name:        l1Config.ChainName,
			DisplayName: l1Config.ChainName,
			Contracts:   types.CrossTradeContracts{L1CrossTrade: &l2toL2CrossTradeL1},
			RPCURL:      t.deployConfig.L1RPCURL,
		},
		l2ChainKey: {
			Name:        l2ChainKey,
			DisplayName: t.deployConfig.ChainName,
			Contracts:   types.CrossTradeContracts{L2CrossTrade: &l2toL2CTProxy},
			RPCURL:      l2RPC,
		},
	}
	l2l2ConfigJSON, err := json.Marshal(l2l2ChainCfg)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal L2→L2 chain config: %w", err)
	}
	l2l2HelmEnv := types.CrossTradeEnvConfig{
		NextPublicProjectID: "568b8d3d0528e743b0e2c6c92f54d721",
		L2L2Config:          string(l2l2ConfigJSON),
	}
	l2l2URL, err = t.installOneHelmRelease(ctx, namespace, chartPath,
		crossTradeL2L2ReleaseName(l2ChainID),
		filepath.Join(configDir, "cross-trade-l2l2-values.yaml"),
		"cross-trade-l2l2",
		l2l2HelmEnv)
	if err != nil {
		return "", "", fmt.Errorf("L2→L2 Helm install failed: %w", err)
	}

	return l2l1URL, l2l2URL, nil
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
