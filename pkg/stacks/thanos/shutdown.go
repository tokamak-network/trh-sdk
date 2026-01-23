package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShutdownBlock blocks L1 deposits and withdrawals (Step 1).
func (s *ThanosStack) ShutdownBlock(ctx context.Context, dryRun bool) error {
	s.logger.Info("Starting ForceWithdraw Shutdown (Step 1: Block) via Forge...")

	contracts, _ := s.readDeploymentContracts()
	l1Bridge := ""
	if contracts != nil {
		l1Bridge = contracts.L1StandardBridgeProxy
	}

	envVars := []string{
		fmt.Sprintf("CONTRACTS_L1BRIDGE_ADDRESS=%s", l1Bridge),
		fmt.Sprintf("CONTRACTS_L2BRIDGE_ADDRESS=%s", "0x4200000000000000000000000000000000000010"),
	}

	return s.runForgeScript(ctx, "scripts/shutdown/L1Withdrawal.s.sol", "run()", nil, false, envVars, dryRun)
}

// ShutdownFetch fetches L2 asset and burn data (Step 2).
func (s *ThanosStack) ShutdownFetch(ctx context.Context, dryRun bool) error {
	s.logger.Info("Starting ForceWithdraw Asset Fetching (Step 2: Fetch) via Python...")
	return nil
}

// ShutdownGen generates asset snapshots (Step 3).
func (s *ThanosStack) ShutdownGen(ctx context.Context, input types.ShutdownConfig, dryRun bool) error {
	s.logger.Info("Starting ForceWithdraw Asset Generation via Forge...")

	contracts, _ := s.readDeploymentContracts()
	l1Bridge := ""
	if contracts != nil {
		l1Bridge = contracts.L1StandardBridgeProxy
	}

	envVars := []string{
		fmt.Sprintf("L1_RPC_URL=%s", s.deployConfig.L1RPCURL),
		fmt.Sprintf("L2_RPC_URL=%s", s.deployConfig.L2RpcUrl),
		fmt.Sprintf("BRIDGE_PROXY=%s", l1Bridge),
		fmt.Sprintf("OPTIMISM_PORTAL_PROXY=%s", contracts.OptimismPortalProxy),
		fmt.Sprintf("CONTRACTS_L1BRIDGE_ADDRESS=%s", l1Bridge),
		fmt.Sprintf("CONTRACTS_L2BRIDGE_ADDRESS=%s", "0x4200000000000000000000000000000000000010"),
		fmt.Sprintf("L2_WETH_ADDRESS=%s", "0x4200000000000000000000000000000000000006"),
		fmt.Sprintf("L2_START_BLOCK=%s", input.L2StartBlock),
		fmt.Sprintf("L2_END_BLOCK=%s", input.L2EndBlock),
		fmt.Sprintf("SKIP_FETCH=%v", true), // Always true since we handled it in Go
	}

	// Read additional bedrock config (e.g. native token)
	bedrockConfig, _ := s.readBedrockDeployConfigTemplate()
	if bedrockConfig != nil {
		envVars = append(envVars, fmt.Sprintf("L1_NATIVE_TOKEN=%s", bedrockConfig.NativeTokenAddress))
	}

	return s.runForgeScript(ctx, "scripts/shutdown/GenerateAssetSnapshot.s.sol", "run()", nil, true, envVars, dryRun)
}

// getBedrockPath returns the path to the contracts-bedrock directory
func (s *ThanosStack) getBedrockPath() (string, error) {
	searchPaths := []string{
		"/Users/theo/workspace_tokamak/tokamak-thanos/packages/tokamak/contracts-bedrock",
		filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"),
		filepath.Join(filepath.Dir(s.deploymentPath), "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"),
	}

	for _, p := range searchPaths {
		if utils.CheckDirExists(p) {
			return p, nil
		}
	}

	return "", fmt.Errorf("contracts-bedrock directory not found")
}

// runForgeScript is an internal engine for executing Forge scripts with RPC and Env flexibility.
func (s *ThanosStack) runForgeScript(ctx context.Context, scriptPath string, sig string, args []string, useL2 bool, envVars []string, dryRun bool) error {
	if s.deployConfig == nil {
		return fmt.Errorf("deployConfig is nil")
	}

	bedrockPath, err := s.getBedrockPath()
	if err != nil {
		bedrockPath = filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")
	}

	rpcUrl := s.deployConfig.L1RPCURL
	if useL2 {
		rpcUrl = s.deployConfig.L2RpcUrl
	}

	// Handle Impersonation during Dry-Run
	senderFlag := ""
	filteredEnv := []string{
		fmt.Sprintf("PRIVATE_KEY=%s", s.deployConfig.AdminPrivateKey),
		fmt.Sprintf("L1_RPC_URL=%s", s.deployConfig.L1RPCURL),
		fmt.Sprintf("L2_RPC_URL=%s", s.deployConfig.L2RpcUrl),
	}

	for _, env := range envVars {
		if strings.HasPrefix(env, "IMPERSONATE_SENDER=") && dryRun {
			senderFlag = fmt.Sprintf("--sender %s", strings.TrimPrefix(env, "IMPERSONATE_SENDER="))
			continue
		}
		filteredEnv = append(filteredEnv, env)
	}

	argStr := ""
	if len(args) > 0 {
		argStr = strings.Join(args, " ")
	}

	base := filepath.Base(scriptPath)
	contractName := strings.TrimSuffix(base, ".s.sol")

	broadcast := "--broadcast"
	if dryRun {
		broadcast = ""
	}

	cmdStr := fmt.Sprintf("cd %s && %s forge script %s:%s --rpc-url %s %s --sig \"%s\" %s %s",
		bedrockPath,
		strings.Join(filteredEnv, " "),
		scriptPath,
		contractName,
		rpcUrl,
		broadcast,
		sig,
		senderFlag,
		argStr,
	)

	s.logger.Infof("🚀 Bedrock Path: %s", bedrockPath)
	s.logger.Infof("🚀 Running Forge Script: %s (RPC: %s, DryRun: %v)", scriptPath, rpcUrl, dryRun)
	return utils.ExecuteCommandStream(ctx, s.logger, "bash", "-c", cmdStr)
}

// readDeploymentContracts reads L1 deployment contract addresses.
func (s *ThanosStack) readDeploymentContracts() (*types.Contracts, error) {
	bedrockConfig, err := s.readBedrockDeployConfigTemplate()
	if err != nil {
		return nil, err
	}

	fileName := fmt.Sprintf("%d-deploy.json", bedrockConfig.L1ChainID)
	searchPaths := []string{
		filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fileName),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fileName),
		filepath.Join(s.deploymentPath, "deployments", fileName),
	}

	for _, p := range searchPaths {
		if utils.CheckFileExists(p) {
			data, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			var contracts types.Contracts
			if err := json.Unmarshal(data, &contracts); err != nil {
				return nil, err
			}
			return &contracts, nil
		}
	}

	return nil, fmt.Errorf("deployment file not found for chain %d", bedrockConfig.L1ChainID)
}

// readBedrockDeployConfigTemplate reads deployment configuration from monorepo (e.g., thanos-sepolia.json)
func (s *ThanosStack) readBedrockDeployConfigTemplate() (*types.DeployConfigTemplate, error) {
	bedrockPath, err := s.getBedrockPath()
	if err != nil {
		return nil, err
	}

	network := s.network
	if network == "testnet" {
		network = "thanos-sepolia"
	}
	filePath := filepath.Join(bedrockPath, "deploy-config", fmt.Sprintf("%s.json", network))

	if !utils.CheckFileExists(filePath) {
		return nil, fmt.Errorf("deploy config file not found: %s", filePath)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var config types.DeployConfigTemplate
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// ShutdownActivate prepares L1 withdrawal (Step 1~7).
func (s *ThanosStack) ShutdownActivate(ctx context.Context, assetsPath string, dryRun bool) error {
	s.logger.Info("Starting L1 Withdrawal Preparation (Phase 1) via Forge...")
	bedrockPath, _ := s.getBedrockPath()

	contracts, _ := s.readDeploymentContracts()
	bedrockConfig, _ := s.readBedrockDeployConfigTemplate()

	dataPath := assetsPath
	if !filepath.IsAbs(dataPath) {
		dataPath = filepath.Join(bedrockPath, dataPath)
	}

	extraEnv := []string{
		fmt.Sprintf("DATA_PATH=%s", dataPath),
		fmt.Sprintf("DRY_RUN=%v", dryRun),
	}

	if contracts != nil {
		extraEnv = append(extraEnv, fmt.Sprintf("BRIDGE_PROXY=%s", contracts.L1StandardBridgeProxy))
		extraEnv = append(extraEnv, fmt.Sprintf("PROXY_ADMIN=%s", contracts.ProxyAdmin))
		extraEnv = append(extraEnv, fmt.Sprintf("OPTIMISM_PORTAL_PROXY=%s", contracts.OptimismPortalProxy))
		extraEnv = append(extraEnv, fmt.Sprintf("L1_USDC_BRIDGE_PROXY=%s", contracts.L1UsdcBridgeProxy))
	}

	if bedrockConfig != nil {
		extraEnv = append(extraEnv, fmt.Sprintf("SYSTEM_OWNER_SAFE=%s", bedrockConfig.FinalSystemOwner))
		if dryRun {
			// Impersonate the actual observed ProxyAdmin owner during simulation
			extraEnv = append(extraEnv, fmt.Sprintf("IMPERSONATE_SENDER=%s", "0xC8442F4521bc6C1D39f9A93CC05B548E2cBf4952"))
		}
	}

	return s.runForgeScript(ctx, "scripts/shutdown/PrepareL1Withdrawal.s.sol", "run()", nil, false, extraEnv, dryRun)
}

// ShutdownWithdraw executes liquidity sweep and claims (Step 8~10).
func (s *ThanosStack) ShutdownWithdraw(ctx context.Context, assetsPath string, dryRun bool) error {
	s.logger.Info("Starting L1 Asset Withdrawal Execution (Phase 2) via Forge...")
	bedrockPath, _ := s.getBedrockPath()

	contracts, _ := s.readDeploymentContracts()
	bedrockConfig, _ := s.readBedrockDeployConfigTemplate()

	dataPath := assetsPath
	if !filepath.IsAbs(dataPath) {
		dataPath = filepath.Join(bedrockPath, dataPath)
	}

	extraEnv := []string{
		fmt.Sprintf("DATA_PATH=%s", dataPath),
		fmt.Sprintf("DRY_RUN=%v", dryRun),
	}

	if contracts != nil {
		extraEnv = append(extraEnv, fmt.Sprintf("BRIDGE_PROXY=%s", contracts.L1StandardBridgeProxy))
		extraEnv = append(extraEnv, fmt.Sprintf("OPTIMISM_PORTAL_PROXY=%s", contracts.OptimismPortalProxy))
	}

	if bedrockConfig != nil {
		extraEnv = append(extraEnv, fmt.Sprintf("L1_NATIVE_TOKEN=%s", bedrockConfig.NativeTokenAddress))
		if dryRun {
			extraEnv = append(extraEnv, fmt.Sprintf("IMPERSONATE_SENDER=%s", bedrockConfig.FinalSystemOwner))
		}
	}

	return s.runForgeScript(ctx, "scripts/shutdown/ExecuteL1Withdrawal.s.sol", "run()", nil, false, extraEnv, dryRun)
}
