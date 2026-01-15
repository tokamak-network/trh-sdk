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

// ShutdownGenInput holds input for asset generation
type ShutdownGenInput struct {
	L2StartBlock string
	L2EndBlock   string
	Output       string
	SkipVerify   bool
}

// ShutdownGen generates asset snapshots using the new Forge script.
func (s *ThanosStack) ShutdownGen(ctx context.Context, input *ShutdownGenInput) error {
	s.logger.Info("Starting ForceWithdraw Asset Generation (Step 1) via Forge...")

	// Fetch L1 Bridge Proxy from deployments
	contracts, _ := s.readDeploymentContracts()
	l1Bridge := ""
	if contracts != nil {
		l1Bridge = contracts.L1StandardBridgeProxy
	}

	envVars := []string{
		fmt.Sprintf("CONTRACTS_L1BRIDGE_ADDRESS=%s", l1Bridge),
		fmt.Sprintf("CONTRACTS_L2BRIDGE_ADDRESS=%s", "0x4200000000000000000000000000000000000010"),
		fmt.Sprintf("CONTRACTS_NONFUNGIBLE_ADDRESS=%s", "0x0000000000000000000000000000000000000000"), // TODO: Fetch or config
		fmt.Sprintf("L2_WETH_ADDRESS=%s", "0x4200000000000000000000000000000000000006"),
		fmt.Sprintf("L2_START_BLOCK=%s", input.L2StartBlock),
		fmt.Sprintf("L2_END_BLOCK=%s", input.L2EndBlock),
	}

	return s.runForgeScript(ctx, "scripts/shutdown/GenerateAssetSnapshot.s.sol", "run()", nil, true, envVars)
}

// runForgeScript is an internal engine for executing Forge scripts with RPC and Env flexibility.
func (s *ThanosStack) runForgeScript(ctx context.Context, scriptPath string, sig string, args []string, useL2 bool, extraEnv []string) error {
	if s.deployConfig == nil {
		return fmt.Errorf("deployConfig is nil, cannot run forge script")
	}

	// Try multiple possible paths for tokamak-thanos bedrock
	searchPaths := []string{
		filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "tokamak-thanos", "packages", "tokamak", "contracts-bedrock"),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "contracts-bedrock"),
		filepath.Join(s.deploymentPath, "packages", "tokamak", "contracts-bedrock"),
	}

	var bedrockPath string
	found := false
	for _, p := range searchPaths {
		if utils.CheckDirExists(p) {
			bedrockPath = p
			found = true
			break
		}
	}

	if !found {
		// Default fallback
		bedrockPath = filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock")
	}

	rpcUrl := s.deployConfig.L1RPCURL
	if useL2 {
		rpcUrl = s.deployConfig.L2RpcUrl
	}

	envVars := []string{
		fmt.Sprintf("PRIVATE_KEY=%s", s.deployConfig.AdminPrivateKey),
		fmt.Sprintf("L1_RPC_URL=%s", s.deployConfig.L1RPCURL),
		fmt.Sprintf("L2_RPC_URL=%s", s.deployConfig.L2RpcUrl),
	}
	envVars = append(envVars, extraEnv...)

	argStr := ""
	if len(args) > 0 {
		argStr = strings.Join(args, " ")
	}

	// Extract contract name from script path (e.g., scripts/File.s.sol -> File)
	base := filepath.Base(scriptPath)
	contractName := strings.TrimSuffix(base, ".s.sol")

	cmdStr := fmt.Sprintf("cd %s && %s forge script %s:%s --rpc-url %s --broadcast --sig \"%s\" %s",
		bedrockPath,
		strings.Join(envVars, " "),
		scriptPath,
		contractName,
		rpcUrl,
		sig,
		argStr,
	)

	s.logger.Infof("🚀 Bedrock Path: %s", bedrockPath)
	s.logger.Infof("🚀 Running Forge Script: %s (RPC: %s)", scriptPath, rpcUrl)
	s.logger.Debugf("👉 Command: %s", cmdStr)
	return utils.ExecuteCommandStream(ctx, s.logger, "bash", "-c", cmdStr)
}

// readDeploymentContracts reads L1 deployment contract addresses.
func (s *ThanosStack) readDeploymentContracts() (*types.Contracts, error) {
	if s.deployConfig == nil {
		return nil, fmt.Errorf("deployConfig is nil, cannot read deployment contracts")
	}

	// Try multiple possible paths for deployments
	fileName := fmt.Sprintf("%d-deploy.json", s.deployConfig.L1ChainID)

	searchPaths := []string{
		filepath.Join(s.deploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fileName),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments", fileName),
		filepath.Join(filepath.Dir(filepath.Dir(s.deploymentPath)), "contracts-bedrock", "deployments", fileName),
		filepath.Join(s.deploymentPath, "packages", "tokamak", "contracts-bedrock", "deployments", fileName),
		filepath.Join(s.deploymentPath, "deployments", fileName),
	}

	var filePath string
	found := false
	for _, p := range searchPaths {
		if utils.CheckFileExists(p) {
			filePath = p
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("deployment file not found for chain %d", s.deployConfig.L1ChainID)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var contracts types.Contracts
	if err := json.Unmarshal(data, &contracts); err != nil {
		return nil, err
	}

	return &contracts, nil
}

// ShutdownDeployStorage deploys FW storage contracts (Step 2).
func (s *ThanosStack) ShutdownDeployStorage(ctx context.Context, assetsPath string) error {
	return s.runForgeScript(ctx, "scripts/shutdown/DeployGenStorage.s.sol", "run(string)", []string{assetsPath}, false, nil)
}

// ShutdownRegister registers storage positions to the L1 bridge (Step 3).
func (s *ThanosStack) ShutdownRegister(ctx context.Context, bridgeProxy string, positionsPath string) error {
	return s.runForgeScript(ctx, "scripts/shutdown/RegisterForceWithdraw.s.sol", "run(address,string)", []string{bridgeProxy, positionsPath}, false, nil)
}

// ShutdownActivate enables or disables force withdrawal functionality on the bridge (Step 4).
func (s *ThanosStack) ShutdownActivate(ctx context.Context, bridgeProxy string, activate bool) error {
	val := "false"
	if activate {
		val = "true"
	}
	return s.runForgeScript(ctx, "scripts/shutdown/ActivateForceWithdraw.s.sol", "run(address,bool)", []string{bridgeProxy, val}, false, nil)
}

// ShutdownSend executes the actual asset withdrawal claims on L1 (Step 5).
func (s *ThanosStack) ShutdownSend(ctx context.Context, bridgeProxy string, assetsPath string, positionAddr string) error {
	return s.runForgeScript(ctx, "scripts/shutdown/ExecuteForceWithdraw.s.sol", "run(address,string,address)", []string{bridgeProxy, assetsPath, positionAddr}, false, nil)
}
