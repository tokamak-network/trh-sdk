package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

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
	// Always derive ThanosRoot from SDKPath to avoid dependency on settings.json
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

// ActionShutdownBlock blocks deposits and withdrawals
func ActionShutdownBlock() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🚫 Blocking L1 Deposits and Withdrawals...")
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}
		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		return client.ShutdownBlock(ctx, cmd.Bool("dry-run"))
	}
}

// ActionShutdownFetch collects L2 asset information
func ActionShutdownFetch() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🔍 Collecting L2 Asset Information...")
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}
		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		return client.ShutdownFetch(ctx, cmd.Bool("dry-run"))
	}
}

// ActionShutdownGen generates force withdrawal assets snapshot
func ActionShutdownGen() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🚀 Generating L2 Asset Snapshot...")

		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		l2StartBlock := cmd.String("l2-start-block")
		if l2StartBlock == "" {
			l2StartBlock = "0"
		}
		l2EndBlock := cmd.String("l2-end-block")
		if l2EndBlock == "" {
			l2EndBlock = "latest"
		}

		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)

		l2StartBlockInt, _ := strconv.ParseUint(l2StartBlock, 10, 64)
		input := types.ShutdownConfig{
			L2StartBlock: l2StartBlockInt,
			L2EndBlock:   l2EndBlock,
		}

		return client.ShutdownGen(ctx, input, cmd.Bool("dry-run"))
	}
}

// ActionShutdownActivate prepares L1 withdrawal (Phase 1)
func ActionShutdownActivate() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("⚙️ Preparing L1 Withdrawal (Phase 1)...")
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}
		assetsPath := cmd.String("input")
		if assetsPath == "" {
			if sc.Config.Shutdown != nil && sc.Config.Shutdown.AssetsDataPath != "" {
				assetsPath = sc.Config.Shutdown.AssetsDataPath
			} else {
				assetsPath = fmt.Sprintf("data/generate-assets-%d.json", sc.Config.L2ChainID)
			}
		}
		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		_, err = client.ShutdownActivate(ctx, assetsPath, cmd.Bool("dry-run"))
		return err
	}
}

// ActionShutdownWithdraw executes liquidity sweep and claims (Phase 2)
func ActionShutdownWithdraw() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("💰 Executing L1 Asset Withdrawal (Phase 2)...")
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}
		assetsPath := cmd.String("input")
		if assetsPath == "" {
			if sc.Config.Shutdown != nil && sc.Config.Shutdown.AssetsDataPath != "" {
				assetsPath = sc.Config.Shutdown.AssetsDataPath
			} else {
				assetsPath = fmt.Sprintf("data/generate-assets-%d.json", sc.Config.L2ChainID)
			}
		}
		client, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		storageAddr := cmd.String("storage-address")
		return client.ShutdownWithdraw(ctx, assetsPath, cmd.Bool("dry-run"), storageAddr)
	}
}

// ActionShutdownRun orchestrates the entire shutdown process sequentially
func ActionShutdownRun() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🏁 Starting Integrated Shutdown Process (Sequential)...")
		sc, err := NewShutdownContext(ctx)
		if err != nil {
			return err
		}

		// 1. Block
		if err := ActionShutdownBlock()(ctx, cmd); err != nil {
			return fmt.Errorf("Step [BLOCK] failed: %w", err)
		}
		fmt.Println("✅ Step [BLOCK] completed successfully.")

		// 2. Fetch
		if cmd.Bool("skip-fetch") {
			fmt.Println("⏭️ Skipping Step [FETCH] as requested (using existing data).")
		} else {
			if err := ActionShutdownFetch()(ctx, cmd); err != nil {
				return fmt.Errorf("Step [FETCH] failed: %w", err)
			}
			fmt.Println("✅ Step [FETCH] completed successfully.")
		}

		// 3. Gen
		err = ActionShutdownGen()(ctx, cmd)
		if err != nil {
			return fmt.Errorf("Step [GEN] failed: %w", err)
		}

		// Persistence: Save the path to settings.json
		if sc.Config.Shutdown == nil {
			sc.Config.Shutdown = &types.ShutdownConfig{}
		}
		sc.Config.Shutdown.AssetsDataPath = fmt.Sprintf("data/generate-assets-%d.json", sc.Config.L2ChainID)
		sc.Config.WriteToJSONFile(sc.DeploymentPath)

		fmt.Println("✅ Step [GEN] completed successfully (Path saved to settings.json).")

		// 4. Activate
		storageAddr := cmd.String("storage-address")
		if storageAddr == "" {
			activateClient, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
			activateAddr, err := activateClient.ShutdownActivate(ctx, sc.Config.Shutdown.AssetsDataPath, cmd.Bool("dry-run"))
			if err != nil {
				return fmt.Errorf("Step [ACTIVATE] failed: %w", err)
			}
			storageAddr = activateAddr
		} else {
			if err := ActionShutdownActivate()(ctx, cmd); err != nil {
				return fmt.Errorf("Step [ACTIVATE] failed: %w", err)
			}
		}
		fmt.Println("✅ Step [ACTIVATE] completed successfully.")

		// 5. Withdraw
		withdrawClient, _ := thanos.NewThanosStack(ctx, sc.Logger, sc.Config.Network, false, sc.DeploymentPath, nil)
		if err := withdrawClient.ShutdownWithdraw(ctx, sc.Config.Shutdown.AssetsDataPath, cmd.Bool("dry-run"), storageAddr); err != nil {
			return fmt.Errorf("Step [WITHDRAW] failed: %w", err)
		}

		fmt.Println("\n🎉 Integrated Shutdown Process Completed Successfully!")
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
		assetsPath := ""
		if sc.Config.Shutdown != nil && sc.Config.Shutdown.AssetsDataPath != "" {
			assetsPath = sc.Config.Shutdown.AssetsDataPath
		} else {
			assetsPath = fmt.Sprintf("data/generate-assets-%d.json", sc.Config.L2ChainID)
		}

		fmt.Printf("\n📁 Assets File:\n")
		if utils.CheckFileExists(assetsPath) {
			info, _ := os.Stat(assetsPath)
			fmt.Printf("   ✅ Found: %s\n", assetsPath)
			fmt.Printf("   Last modified: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Printf("   ❌ Not found (%s). Run 'trh-sdk shutdown gen' first.\n", assetsPath)
		}

		return nil
	}
}
