package thanos

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// DeployAAContracts deploys post-genesis Account Abstraction contracts
// (SimplePriceOracle and MultiTokenPaymaster) to L2 when the Gaming or Full
// preset is active. For other presets it is a no-op.
func (t *ThanosStack) DeployAAContracts(ctx context.Context) error {
	preset := t.deployConfig.Preset
	if preset != constants.PresetGaming && preset != constants.PresetFull {
		t.logger.Info("Skipping post-genesis AA deployment (preset is not gaming or full)", "preset", preset)
		return nil
	}

	l2RPCUrl := t.deployConfig.L2RpcUrl
	if l2RPCUrl == "" {
		return fmt.Errorf("L2 RPC URL is not set; cannot deploy AA contracts")
	}

	t.logger.Info("Deploying post-genesis AA contracts (SimplePriceOracle, MultiTokenPaymaster)...")

	scriptsDir := filepath.Join(t.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "scripts")

	// Deploy SimplePriceOracle
	t.logger.Info("Deploying SimplePriceOracle...")
	simplePriceOracleAddr, err := t.deployAAContract(ctx, scriptsDir, "SimplePriceOracle", l2RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to deploy SimplePriceOracle: %w", err)
	}
	t.logger.Infof("✅ SimplePriceOracle deployed at: %s", simplePriceOracleAddr)
	t.deployConfig.SimplePriceOracleAddress = simplePriceOracleAddr

	// Deploy MultiTokenPaymaster (depends on EntryPoint predeploy at 0x4200...0063)
	t.logger.Info("Deploying MultiTokenPaymaster...", "entryPoint", constants.AAEntryPoint)
	multiTokenPaymasterAddr, err := t.deployAAContract(ctx, scriptsDir, "MultiTokenPaymaster", l2RPCUrl)
	if err != nil {
		return fmt.Errorf("failed to deploy MultiTokenPaymaster: %w", err)
	}
	t.logger.Infof("✅ MultiTokenPaymaster deployed at: %s", multiTokenPaymasterAddr)
	t.deployConfig.MultiTokenPaymasterAddress = multiTokenPaymasterAddr

	t.logger.Info("✅ Post-genesis AA contracts deployed successfully")
	return nil
}

// deployAAContract runs the deploy-aa-post-genesis.sh script for the given
// contract name and returns the deployed contract address.
func (t *ThanosStack) deployAAContract(ctx context.Context, scriptsDir, contractName, l2RPCUrl string) (string, error) {
	adminKey := t.deployConfig.AdminPrivateKey
	if adminKey == "" {
		return "", fmt.Errorf("admin private key is not set")
	}

	// Execute the deployment script; the script is expected to print the
	// deployed address as the last line of stdout.
	output, err := utils.ExecuteCommandInDir(
		ctx,
		scriptsDir,
		"bash",
		"./deploy-aa-post-genesis.sh",
		contractName,
		l2RPCUrl,
		adminKey,
	)
	if err != nil {
		return "", fmt.Errorf("deploy script failed for %s: %w", contractName, err)
	}

	addr := extractLastLine(output)
	if addr == "" {
		return "", fmt.Errorf("deploy script for %s produced no address output", contractName)
	}
	return addr, nil
}

// extractLastLine returns the last non-empty line from a multi-line string.
func extractLastLine(s string) string {
	lines := splitLines(s)
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i] != "" {
			return lines[i]
		}
	}
	return ""
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}
