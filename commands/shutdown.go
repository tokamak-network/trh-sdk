package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
)

// ShutdownContext holds configuration for shutdown operations
type ShutdownContext struct {
	DeploymentPath string
	Config         *types.Config
	SDKPath        string
	Logger         *zap.SugaredLogger
	State          *types.ShutdownState
}

// NewShutdownContext creates a new shutdown context by reading settings.json
func NewShutdownContext(ctx context.Context) (*ShutdownContext, error) {
	deploymentPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read settings.json: %w", err)
	}

	if config == nil {
		return nil, fmt.Errorf("settings.json not found in current directory")
	}

	// Validate settings.json according to spec
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("settings.json validation failed: %w", err)
	}

	// Find SDK path
	sdkPath := findSDKPath(deploymentPath)
	if sdkPath == "" {
		return nil, fmt.Errorf("could not find tokamak-thanos SDK directory")
	}

	// Initialize logger
	logFileName := fmt.Sprintf("%s/logs/shutdown_%d.log", deploymentPath, config.L2ChainID)
	logger, err := logging.InitLogger(logFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Load or create shutdown state
	state, err := types.LoadShutdownState()
	if err != nil {
		return nil, fmt.Errorf("failed to load shutdown state: %w", err)
	}

	// Update state with current context
	state.ChainID = config.L1ChainID
	state.L2ChainID = config.L2ChainID
	state.ThanosRoot = filepath.Dir(filepath.Dir(sdkPath))
	state.DeploymentsPath = fmt.Sprintf("%d-deploy.json", config.L1ChainID)

	return &ShutdownContext{
		DeploymentPath: deploymentPath,
		Config:         config,
		SDKPath:        sdkPath,
		Logger:         logger,
		State:          state,
	}, nil
}

// validateConfig validates the configuration according to spec requirements
func validateConfig(config *types.Config) error {
	// Validate L1 RPC URL
	if config.L1RPCURL == "" {
		return fmt.Errorf("l1_rpc_url cannot be empty")
	}

	// Validate L2 RPC URL
	if config.L2RpcUrl == "" {
		return fmt.Errorf("l2_rpc_url cannot be empty")
	}

	// Validate L1 Chain ID
	if config.L1ChainID == 0 {
		return fmt.Errorf("l1_chain_id must be greater than 0")
	}

	// Validate L2 Chain ID
	if config.L2ChainID == 0 {
		return fmt.Errorf("l2_chain_id must be greater than 0")
	}

	return nil
}

// ensureWorkspacePackages checks if workspace packages are properly built and symlinked
func (sc *ShutdownContext) ensureWorkspacePackages(ctx context.Context) error {
	coreUtilsSymlink := filepath.Join(sc.SDKPath, "node_modules", "@tokamak-network", "core-utils", "dist", "index.js")

	if _, err := os.Stat(coreUtilsSymlink); os.IsNotExist(err) {
		sc.Logger.Warn("Workspace packages are not properly built or symlinks are broken")
		sc.Logger.Info("Attempting to rebuild and restore workspace symlinks...")

		thanosRoot := filepath.Dir(filepath.Dir(sc.SDKPath))

		// Rebuild core-utils
		coreUtilsPath := filepath.Join(thanosRoot, "core-utils")
		sc.Logger.Info("Rebuilding core-utils...")
		if err := utils.ExecuteCommandStream(ctx, sc.Logger, "bash", "-c", fmt.Sprintf("cd %s && pnpm build", coreUtilsPath)); err != nil {
			sc.Logger.Error("Failed to rebuild core-utils")
			return fmt.Errorf("failed to rebuild core-utils: %v", err)
		}

		// Reinstall SDK dependencies to restore workspace symlinks
		sc.Logger.Info("Restoring workspace symlinks...")
		thanosRootParent := filepath.Dir(filepath.Dir(thanosRoot))
		if err := utils.ExecuteCommandStream(ctx, sc.Logger, "bash", "-c", fmt.Sprintf("cd %s && pnpm install --prefer-offline", thanosRootParent)); err != nil {
			sc.Logger.Error("Failed to restore workspace symlinks")
			return fmt.Errorf("failed to restore workspace symlinks: %v", err)
		}

		// Rebuild SDK
		sc.Logger.Info("Rebuilding SDK...")
		if err := utils.ExecuteCommandStream(ctx, sc.Logger, "bash", "-c", fmt.Sprintf("cd %s && pnpm build", sc.SDKPath)); err != nil {
			sc.Logger.Error("Failed to rebuild SDK")
			return fmt.Errorf("failed to rebuild SDK: %v", err)
		}

		// Verify the fix worked
		if _, err := os.Stat(coreUtilsSymlink); os.IsNotExist(err) {
			sc.Logger.Error("Failed to restore workspace packages after rebuild attempt")
			return fmt.Errorf("workspace packages could not be restored: %v", err)
		}

		sc.Logger.Info("✅ Successfully rebuilt workspace packages and restored symlinks")
	} else {
		sc.Logger.Info("✅ Workspace packages are properly built and symlinked")
	}

	return nil
}

// readDeploymentContracts reads deployment contract addresses
func (sc *ShutdownContext) readDeploymentContracts() (*types.Contracts, error) {
	deployPath := filepath.Join(sc.DeploymentPath, "tokamak-thanos", "packages", "tokamak", "contracts-bedrock", "deployments")
	fileName := fmt.Sprintf("%d-deploy.json", sc.Config.L1ChainID)
	filePath := filepath.Join(deployPath, fileName)

	if !utils.CheckFileExists(filePath) {
		// Try alternative paths
		altPaths := []string{
			filepath.Join(sc.SDKPath, "..", "..", "contracts-bedrock", "deployments", fileName),
			filepath.Join(sc.DeploymentPath, fileName),
		}

		found := false
		for _, p := range altPaths {
			if utils.CheckFileExists(p) {
				filePath = p
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("deployment file not found")
		}
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

// findSDKPath attempts to locate the tokamak-thanos SDK directory
func findSDKPath(deploymentPath string) string {
	// Check environment variable first
	if path := os.Getenv("TOKAMAK_THANOS_SDK_PATH"); path != "" {
		return path
	}

	// Common paths relative to deployment path
	paths := []string{
		filepath.Join(deploymentPath, "tokamak-thanos", "packages", "tokamak", "sdk"),
		filepath.Join(deploymentPath, "..", "tokamak-thanos", "packages", "tokamak", "sdk"),
		filepath.Join(deploymentPath, "..", "..", "tokamak-thanos", "packages", "tokamak", "sdk"),
	}

	for _, p := range paths {
		if utils.CheckFileExists(filepath.Join(p, "hardhat.config.ts")) {
			absPath, _ := filepath.Abs(p)
			return absPath
		}
	}

	return ""
}

// ActionShutdown returns the shutdown command action
func ActionShutdown() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		return cli.ShowSubcommandHelp(cmd)
	}
}

// ActionShutdownGen generates force withdrawal assets snapshot
func ActionShutdownGen() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🚀 Starting ForceWithdraw Asset Generation...")

		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		l2StartBlock := cmd.String("l2-start-block")
		if l2StartBlock == "" {
			l2StartBlock = "0"
		}
		l2EndBlock := cmd.String("l2-end-block")
		if l2EndBlock == "" {
			l2EndBlock = "latest"
		}

		dataDir := filepath.Join(sc.SDKPath, "data")

		output := cmd.String("output")
		if output == "" {
			output = filepath.Join(dataDir, "generate-assets3.json")
		}

		// Create SDK Client
		client, err := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err != nil {
			return err
		}

		input := &thanos.ShutdownGenInput{
			L2StartBlock: l2StartBlock,
			L2EndBlock:   l2EndBlock,
			Output:       output,
			SkipVerify:   cmd.Bool("skip-verify"),
		}

		if err := client.ShutdownGen(ctx, input); err != nil {
			return err
		}

		sc.State.UpdateAfterGen(output)
		_ = sc.State.Save()

		sc.Logger.Info("✅ Asset generation completed and state updated")
		return nil
	}
}

// ActionShutdownDryRun estimates gas without sending transactions
func ActionShutdownDryRun() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🔍 Starting ForceWithdraw Dry-Run...")

		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		dataDir := cmd.String("data-dir")
		if dataDir == "" {
			dataDir = filepath.Join(sc.SDKPath, "data")
		}

		input := cmd.String("input")
		if input == "" {
			input = filepath.Join(dataDir, "generate-assets3.json")
		}

		contracts, _ := sc.readDeploymentContracts()
		bridgeAddr := ""
		if contracts != nil {
			bridgeAddr = contracts.L1StandardBridgeProxy
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := client.ShutdownSend(ctx, bridgeAddr, input, "0x0000000000000000000000000000000000000000"); err != nil {
			return err
		}

		sc.State.UpdateAfterDryRun()
		_ = sc.State.Save()
		return nil
	}
}

// ActionShutdownSend executes force withdrawal on L1
func ActionShutdownSend() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		dataDir := cmd.String("data-dir")
		if dataDir == "" {
			dataDir = filepath.Join(sc.SDKPath, "data")
		}

		input := cmd.String("input")
		if input == "" {
			input = filepath.Join(dataDir, "generate-assets3.json")
		}

		positionAddr := cmd.String("position-contract")
		if positionAddr == "" {
			positionAddr = "0x0000000000000000000000000000000000000000"
		}

		contracts, _ := sc.readDeploymentContracts()
		bridgeAddr := ""
		if contracts != nil {
			bridgeAddr = contracts.L1StandardBridgeProxy
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := client.ShutdownSend(ctx, bridgeAddr, input, positionAddr); err != nil {
			return err
		}

		sc.State.UpdateAfterSend()
		_ = sc.State.Save()
		return nil
	}
}

// ActionShutdownDeployStorage deploys FW storage contracts (Step 2)
func ActionShutdownDeployStorage() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		dataDir := cmd.String("data-dir")
		if dataDir == "" {
			dataDir = filepath.Join(sc.SDKPath, "data")
		}

		input := cmd.String("input")
		if input == "" {
			input = filepath.Join(dataDir, "generate-assets3.json")
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := client.ShutdownDeployStorage(ctx, input); err != nil {
			return err
		}

		sc.Logger.Info("✅ Storage contracts deployed successfully")
		return nil
	}
}

// ActionShutdownRegister registers positions to the bridge (Step 3)
func ActionShutdownRegister() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		dataDir := cmd.String("data-dir")
		if dataDir == "" {
			dataDir = filepath.Join(sc.SDKPath, "data")
		}

		input := cmd.String("input")
		if input == "" {
			input = filepath.Join(dataDir, "genstorage-addresses.json")
		}

		contracts, _ := sc.readDeploymentContracts()
		bridgeAddr := ""
		if contracts != nil {
			bridgeAddr = contracts.L1StandardBridgeProxy
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := client.ShutdownRegister(ctx, bridgeAddr, input); err != nil {
			return err
		}

		sc.Logger.Info("✅ Positions registered successfully")
		return nil
	}
}

// ActionShutdownActivate activates the bridge shutdown functionality (Step 4)
func ActionShutdownActivate() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		contracts, _ := sc.readDeploymentContracts()
		bridgeAddr := ""
		if contracts != nil {
			bridgeAddr = contracts.L1StandardBridgeProxy
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := client.ShutdownActivate(ctx, bridgeAddr, true); err != nil {
			return err
		}

		sc.Logger.Info("✅ Bridge shutdown activated successfully")
		return nil
	}
}

// ActionShutdownRun orchestrates the entire shutdown process or specific steps
func ActionShutdownRun() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🏁 Starting Integrated Shutdown Process...")

		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return fmt.Errorf("initialization failed: %w", err)
		}

		// Check if we should run the integrated shell script or individual Go actions
		if cmd.Bool("use-script") {
			sc.Logger.Info("🚀 Running integrated shell script: e2e-shutdown-test.sh")
			scriptPath := filepath.Join(sc.SDKPath, "scripts", "e2e-shutdown-test.sh")

			// Build environment variables from config
			envVars := []string{
				fmt.Sprintf("L1_RPC_URL=%s", sc.Config.L1RPCURL),
				fmt.Sprintf("L2_RPC_URL=%s", sc.Config.L2RpcUrl),
				fmt.Sprintf("PRIVATE_KEY=%s", sc.Config.AdminPrivateKey),
				fmt.Sprintf("NETWORK=%s", sc.Config.Network),
			}

			cmdStr := fmt.Sprintf("%s %s %s", strings.Join(envVars, " "), scriptPath, sc.Config.Network)
			return utils.ExecuteCommandStream(ctx, sc.Logger, "bash", "-c", cmdStr)
		}

		// Individual steps orchestrated via flags
		all := !cmd.Bool("gen") && !cmd.Bool("deploy") && !cmd.Bool("register") && !cmd.Bool("activate") && !cmd.Bool("send")

		// Ensure defaults for block range if genius is running and flags are missing
		l2Start := cmd.String("l2-start-block")
		if l2Start == "" {
			l2Start = "0"
		}
		l2End := cmd.String("l2-end-block")
		if l2End == "" {
			l2End = "latest"
		}

		if all || cmd.Bool("gen") {
			if err := ActionShutdownGen()(ctx, cmd); err != nil {
				return err
			}
		}

		if all || cmd.Bool("deploy") {
			if err := ActionShutdownDeployStorage()(ctx, cmd); err != nil {
				return err
			}
		}

		if all || cmd.Bool("register") {
			if err := ActionShutdownRegister()(ctx, cmd); err != nil {
				return err
			}
		}

		if all || cmd.Bool("activate") {
			if err := ActionShutdownActivate()(ctx, cmd); err != nil {
				return err
			}
		}

		if all || cmd.Bool("send") {
			if err := ActionShutdownSend()(ctx, cmd); err != nil {
				return err
			}
		}

		fmt.Println("\n✅ Integrated Shutdown Process Completed!")
		return nil
	}
}

// ActionShutdownStatus shows current shutdown status
func ActionShutdownStatus() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("📊 Shutdown Status")
		fmt.Println("==================")

		sc, err := NewShutdownContext(ctx)
		if err != nil {
			fmt.Printf("⚠️  No active deployment found: %s\n", err)
			return nil
		}

		// Display current configuration
		fmt.Printf("\n🔧 Current Configuration:\n")
		fmt.Printf("   Deployment Path: %s\n", sc.DeploymentPath)
		fmt.Printf("   SDK Path: %s\n", sc.SDKPath)
		fmt.Printf("   L1 Chain ID: %d\n", sc.Config.L1ChainID)
		fmt.Printf("   L2 Chain ID: %d\n", sc.Config.L2ChainID)
		fmt.Printf("   Network: %s\n", sc.Config.Network)
		fmt.Printf("   Thanos Root: %s\n", sc.State.ThanosRoot)
		fmt.Printf("   Deployments Path: %s\n", sc.State.DeploymentsPath)
		if sc.State.DataDir != "" {
			fmt.Printf("   Data Directory: %s\n", sc.State.DataDir)
		}

		// Display execution history from state
		fmt.Printf("\n📜 Execution History:\n")
		if sc.State.LastCommand != "" {
			fmt.Printf("   Last Command: %s\n", sc.State.LastCommand)
		} else {
			fmt.Printf("   Last Command: (none)\n")
		}

		if sc.State.LastGenAt != "" {
			fmt.Printf("   Last Gen: %s\n", sc.State.LastGenAt)
			if sc.State.LastSnapshotPath != "" {
				fmt.Printf("   Last Snapshot: %s\n", sc.State.LastSnapshotPath)
			}
		} else {
			fmt.Printf("   Last Gen: (never)\n")
		}

		if sc.State.LastDryRunAt != "" {
			fmt.Printf("   Last Dry-Run: %s\n", sc.State.LastDryRunAt)
		} else {
			fmt.Printf("   Last Dry-Run: (never)\n")
		}

		if sc.State.LastSendAt != "" {
			fmt.Printf("   Last Send: %s\n", sc.State.LastSendAt)
		} else {
			fmt.Printf("   Last Send: (never)\n")
		}

		// Check for generated assets file
		dataDir := sc.State.DataDir
		if dataDir == "" {
			dataDir = filepath.Join(sc.SDKPath, "data")
		}
		assetsPath := filepath.Join(dataDir, "generate-assets3.json")

		fmt.Printf("\n📁 Assets File:\n")
		if utils.CheckFileExists(assetsPath) {
			info, _ := os.Stat(assetsPath)
			fmt.Printf("   ✅ Found: %s\n", assetsPath)
			fmt.Printf("   Last modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   ❌ Not found. Run 'trh-sdk shutdown gen' first.\n")
		}

		return nil
	}
}
