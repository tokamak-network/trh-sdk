package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShutdownBlock blocks L1 deposits and withdrawals (Step 1).
func (s *ThanosStack) ShutdownBlock(ctx context.Context, dryRun bool) error {
	s.logger.Info("Starting ForceWithdraw Shutdown (Step 1: Block) via Forge...")

	contracts, _ := s.readDeploymentContracts()
	bedrockConfig, _ := s.readBedrockDeployConfigTemplate()

	envVars := []string{
		fmt.Sprintf("PROXY_ADMIN=%s", contracts.ProxyAdmin),
		fmt.Sprintf("OPTIMISM_PORTAL_PROXY=%s", contracts.OptimismPortalProxy),
		fmt.Sprintf("SUPERCHAIN_CONFIG_PROXY=%s", contracts.SuperchainConfigProxy),
		fmt.Sprintf("SYSTEM_OWNER_SAFE=%s", contracts.SystemOwnerSafe),
		fmt.Sprintf("GUARDIAN_SAFE=%s", bedrockConfig.SuperchainConfigGuardian),
		fmt.Sprintf("CONTRACTS_L1BRIDGE_ADDRESS=%s", contracts.L1StandardBridgeProxy),
		fmt.Sprintf("CONTRACTS_L2BRIDGE_ADDRESS=%s", "0x4200000000000000000000000000000000000010"),
	}

	return s.runForgeScript(ctx, "scripts/shutdown/BlockDepositsWithdrawals.s.sol", "run()", nil, false, envVars, dryRun)
}

// ShutdownFetch fetches L2 asset and burn data (Step 2).
func (s *ThanosStack) ShutdownFetch(ctx context.Context, dryRun bool) error {
	s.logger.Info("Starting ForceWithdraw Asset Fetching (Step 2: Fetch) via Python...")

	// Safeguard against nil deployConfig
	if s.deployConfig == nil {
		config, err := utils.ReadConfigFromJSONFile(s.deploymentPath)
		if err != nil || config == nil {
			return fmt.Errorf("failed to load deployConfig: %v", err)
		}
		s.deployConfig = config
	}

	contracts, _ := s.readDeploymentContracts()
	bedrockPath, _ := s.getBedrockPath()
	scriptsDir := filepath.Join(bedrockPath, "scripts", "shutdown")
	dataDir := filepath.Join(bedrockPath, "data")

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %v", err)
	}

	// Determine Explorer URL
	// Try to get dynamic URL from K8s ingress first
	explorerUrl, err := s.GetBlockExplorerURL(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block explorer URL: %v", err)
	}

	// Ensure API path
	if !strings.HasSuffix(explorerUrl, "/api/v2") {
		explorerUrl = explorerUrl + "/api/v2"
	}

	envVars := []string{
		fmt.Sprintf("L1_RPC_URL=%s", s.deployConfig.L1RPCURL),
		fmt.Sprintf("L2_RPC_URL=%s", s.deployConfig.L2RpcUrl),
		fmt.Sprintf("L2_CHAIN_ID=%d", s.deployConfig.L2ChainID),
		fmt.Sprintf("DATA_DIR=%s", dataDir),
		fmt.Sprintf("EXPLORER_URL=%s", explorerUrl),
	}

	// Essential contract addresses for fetching withdrawals
	if contracts != nil {
		// Map SDK contract names to Python script requirements
		envVars = append(envVars, fmt.Sprintf("BRIDGE_PROXY=%s", contracts.L1StandardBridgeProxy))
		envVars = append(envVars, fmt.Sprintf("OPTIMISM_PORTAL_PROXY=%s", contracts.OptimismPortalProxy))
	}

	// 1. Fetch Explorer Assets
	s.logger.Info(" -> Running fetch_explorer_assets.py")
	fetchArgs := []string{fmt.Sprintf("%d", s.deployConfig.L2ChainID)}
	if err := s.runPythonScript(ctx, filepath.Join(scriptsDir, "fetch_explorer_assets.py"), fetchArgs, envVars, false); err != nil {
		return err
	}

	// 2. Compute L2 Burns
	s.logger.Info(" -> Running compute_l2_burns.py")
	l2BurnsArgs := []string{s.deployConfig.L2RpcUrl, fmt.Sprintf("%d", s.deployConfig.L2ChainID)}
	if err := s.runPythonScript(ctx, filepath.Join(scriptsDir, "compute_l2_burns.py"), l2BurnsArgs, envVars, false); err != nil {
		return err
	}

	// 3. Compute Finalized Native Withdrawals
	s.logger.Info(" -> Running compute_finalized_native_withdrawals.py")
	// Pass RPC URL and L1 Bridge Address as arguments
	scriptArgs := []string{s.deployConfig.L1RPCURL, contracts.L1StandardBridgeProxy}

	if err := s.runPythonScript(ctx, filepath.Join(scriptsDir, "compute_finalized_native_withdrawals.py"), scriptArgs, envVars, false); err != nil {
		return err
	}

	return nil
}

// runPythonScript executes a python script with given environment variables and arguments
func (s *ThanosStack) runPythonScript(ctx context.Context, scriptPath string, args []string, envVars []string, dryRun bool) error {
	argStr := ""
	if len(args) > 0 {
		argStr = strings.Join(args, " ")
	}

	if dryRun {
		s.logger.Infof("[DryRun] python3 %s %s", scriptPath, argStr)
		return nil
	}

	cmdStr := fmt.Sprintf("%s python3 %s %s", strings.Join(envVars, " "), scriptPath, argStr)
	return utils.ExecuteCommandStream(ctx, s.logger, "bash", "-c", cmdStr)
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
		fmt.Sprintf("L2_START_BLOCK=%d", input.L2StartBlock),
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

func (s *ThanosStack) getBedrockPath() (string, error) {
	// Directly look in deploymentPath/tokamak-thanos/...
	p := filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")

	if utils.CheckDirExists(p) {
		return p, nil
	}

	return "", fmt.Errorf("contracts-bedrock directory not found at: %s", p)
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

	adminKey := s.deployConfig.AdminPrivateKey
	if !strings.HasPrefix(adminKey, "0x") {
		adminKey = "0x" + adminKey
	}

	filteredEnv := []string{
		fmt.Sprintf("PRIVATE_KEY=%s", adminKey),
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

	// Force use of scripts/deploy-config.json as requested
	filePath := filepath.Join(bedrockPath, "scripts", "deploy-config.json")

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

// ShutdownActivate prepares L1 withdrawal (Step 1~7) and returns the storage address if found.
func (s *ThanosStack) ShutdownActivate(ctx context.Context, assetsPath string, dryRun bool) (string, error) {
	s.logger.Info("Starting L1 Withdrawal Preparation (Phase 1) via Forge...")
	bedrockPath, _ := s.getBedrockPath()

	contracts, _ := s.readDeploymentContracts()

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
		extraEnv = append(extraEnv, fmt.Sprintf("SYSTEM_OWNER_SAFE=%s", contracts.SystemOwnerSafe))
	}

	if dryRun {
		// Impersonate the actual observed ProxyAdmin owner during simulation
		extraEnv = append(extraEnv, fmt.Sprintf("IMPERSONATE_SENDER=%s", "0xC8442F4521bc6C1D39f9A93CC05B548E2cBf4952"))
	}

	output, err := s.runForgeScriptCapture(ctx, "scripts/shutdown/PrepareL1Withdrawal.s.sol", "run()", nil, false, extraEnv, dryRun)
	if err != nil {
		return "", err
	}

	// Extract GenFWStorage address from script output
	storageAddr, err := s.extractStorageAddress(output)
	if err != nil {
		if !dryRun {
			s.logger.Warnf("Failed to extract storage address: %v", err)
		}
		// Don't fail the entire operation if we can't extract the address
		return "", nil
	}

	return storageAddr, nil
}

// ShutdownWithdraw executes liquidity sweep and claims (Step 8~10).
func (s *ThanosStack) ShutdownWithdraw(ctx context.Context, assetsPath string, dryRun bool, storageAddr string) error {
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
		extraEnv = append(extraEnv, fmt.Sprintf("PROXY_ADMIN=%s", contracts.ProxyAdmin))
		extraEnv = append(extraEnv, fmt.Sprintf("SYSTEM_OWNER_SAFE=%s", contracts.SystemOwnerSafe))
	}

	if bedrockConfig != nil {
		extraEnv = append(extraEnv, fmt.Sprintf("L1_NATIVE_TOKEN=%s", bedrockConfig.NativeTokenAddress))
		if dryRun {
			extraEnv = append(extraEnv, fmt.Sprintf("IMPERSONATE_SENDER=%s", bedrockConfig.FinalSystemOwner))
		}
	}

	if storageAddr == "" {
		return fmt.Errorf("storage address is required (run 'shutdown activate' first or pass --storage-address)")
	}
	extraEnv = append(extraEnv, fmt.Sprintf("STORAGE_ADDRESS=%s", storageAddr))

	return s.runForgeScript(ctx, "scripts/shutdown/ExecuteL1Withdrawal.s.sol", "run()", nil, false, extraEnv, dryRun)
}

// runForgeScriptCapture executes a Forge script and returns its combined output.
func (s *ThanosStack) runForgeScriptCapture(ctx context.Context, scriptPath string, sig string, args []string, useL2 bool, envVars []string, dryRun bool) (string, error) {
	if s.deployConfig == nil {
		return "", fmt.Errorf("deployConfig is nil")
	}

	bedrockPath, err := s.getBedrockPath()
	if err != nil {
		bedrockPath = filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")
	}

	rpcUrl := s.deployConfig.L1RPCURL
	if useL2 {
		rpcUrl = s.deployConfig.L2RpcUrl
	}

	adminKey := s.deployConfig.AdminPrivateKey
	if !strings.HasPrefix(adminKey, "0x") {
		adminKey = "0x" + adminKey
	}

	filteredEnv := []string{
		fmt.Sprintf("PRIVATE_KEY=%s", adminKey),
		fmt.Sprintf("L1_RPC_URL=%s", s.deployConfig.L1RPCURL),
		fmt.Sprintf("L2_RPC_URL=%s", s.deployConfig.L2RpcUrl),
	}

	senderFlag := ""
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
	output, err := utils.ExecuteCommand(ctx, "bash", "-c", cmdStr)
	if output != "" {
		for _, line := range strings.Split(output, "\n") {
			if strings.TrimSpace(line) != "" {
				s.logger.Info(line)
			}
		}
	}
	return output, err
}

func (s *ThanosStack) extractStorageAddress(output string) (string, error) {
	re := regexp.MustCompile(`Deployed GenFWStorage at:\s*(0x[0-9a-fA-F]{40})`)
	match := re.FindStringSubmatch(output)
	if len(match) < 2 {
		return "", fmt.Errorf("storage address not found in output")
	}
	return match[1], nil
}

func redactEnvVars(envVars []string) []string {
	redacted := make([]string, 0, len(envVars))
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			redacted = append(redacted, env)
			continue
		}
		key := parts[0]
		val := parts[1]
		switch key {
		case "PRIVATE_KEY", "L1_RPC_URL", "L2_RPC_URL":
			val = "<redacted>"
		}
		redacted = append(redacted, fmt.Sprintf("%s=%s", key, val))
	}
	return redacted
}
