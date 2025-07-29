package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

// ActionLogConfig handles the log-config command
func ActionLogConfig() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Check if monitoring plugin is installed
		if err := utils.CheckMonitoringPluginInstalled(ctx); err != nil {
			return err
		}

		// Get flags
		enable := cmd.Bool("enable")
		disable := cmd.Bool("disable")
		retention := cmd.String("retention")
		lokiSize := cmd.String("loki-size")
		promtailSize := cmd.String("promtail-size")
		show := cmd.Bool("show")

		// Handle show command
		if show {
			return handleLogConfigShow(ctx)
		}

		// Load current configuration
		config, err := loadCurrentConfig()
		if err != nil {
			return err
		}

		// Initialize logging config if it doesn't exist
		if config.LoggingConfig == nil {
			config.LoggingConfig = &types.LoggingConfig{
				Enabled: false,
			}
		}

		// Track if any changes were made
		hasChanges := false

		// Handle enable/disable
		if enable {
			config.LoggingConfig.Enabled = true
			hasChanges = true
		}
		if disable {
			config.LoggingConfig.Enabled = false
			hasChanges = true
		}

		// Handle retention setting
		if retention != "" {
			config.LoggingConfig.LokiRetention = retention
			hasChanges = true
		}

		// Handle Loki storage size
		if lokiSize != "" {
			config.LoggingConfig.LokiStorageSize = lokiSize
			hasChanges = true
		}

		// Handle Promtail storage size
		if promtailSize != "" {
			config.LoggingConfig.PromtailStorageSize = promtailSize
			hasChanges = true
		}

		// Save updated configuration to settings.json
		if hasChanges {
			if err := config.WriteToJSONFile("."); err != nil {
				return fmt.Errorf("failed to save configuration: %w", err)
			}
			fmt.Println("✅ Configuration saved to settings.json")
		}

		// Apply changes to running monitoring plugin
		if hasChanges {
			return applyLoggingConfig(ctx, config)
		}

		// Show help if no valid command
		return showLogConfigHelp()
	}
}

// loadCurrentConfig loads the current configuration from settings.json
func loadCurrentConfig() (*types.Config, error) {
	deploymentPath, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	if config == nil {
		return nil, fmt.Errorf("no configuration found. Please run 'trh-sdk deploy' first")
	}

	return config, nil
}

// applyLoggingConfig applies logging configuration to the running monitoring plugin
func applyLoggingConfig(ctx context.Context, config *types.Config) error {
	fmt.Println("Applying logging configuration to monitoring plugin...")

	// Get deployment path
	deploymentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize ThanosStack
	thanosStack, err := thanos.NewThanosStack(ctx, nil, config.Network, true, deploymentPath, config.AWS)
	if err != nil {
		return fmt.Errorf("failed to initialize ThanosStack: %w", err)
	}

	// Create monitoring config with logging settings
	monitoringConfig := &types.MonitoringConfig{
		LoggingEnabled: config.LoggingConfig.Enabled,
		// Add other required fields from config
		Namespace:       config.ChainName,
		HelmReleaseName: "monitoring",
		ChainName:       config.ChainName,
	}

	// Apply logging configuration
	if err := thanosStack.UpdateLoggingConfig(ctx, monitoringConfig); err != nil {
		return fmt.Errorf("failed to update logging configuration: %w", err)
	}

	fmt.Println("✅ Logging configuration applied successfully!")
	return nil
}

// handleLogConfigShow displays current logging configuration
func handleLogConfigShow(ctx context.Context) error {
	config, err := loadCurrentConfig()
	if err != nil {
		return err
	}

	if config.LoggingConfig == nil {
		fmt.Println("No logging configuration found.")
		return nil
	}

	fmt.Println("Current Logging Configuration:")
	fmt.Printf("  Enabled: %t\n", config.LoggingConfig.Enabled)

	if config.LoggingConfig.LokiRetention != "" {
		fmt.Printf("  Loki Retention: %s\n", config.LoggingConfig.LokiRetention)
	} else {
		fmt.Println("  Loki Retention: 30d (default)")
	}

	if config.LoggingConfig.LokiStorageSize != "" {
		fmt.Printf("  Loki Storage Size: %s\n", config.LoggingConfig.LokiStorageSize)
	} else {
		fmt.Println("  Loki Storage Size: 50Gi (default)")
	}

	if config.LoggingConfig.PromtailStorageSize != "" {
		fmt.Printf("  Promtail Storage Size: %s\n", config.LoggingConfig.PromtailStorageSize)
	} else {
		fmt.Println("  Promtail Storage Size: 5Gi (default)")
	}

	return nil
}

// showLogConfigHelp displays help information
func showLogConfigHelp() error {
	fmt.Println("Usage: trh-sdk log-config [OPTIONS]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  --enable                    Enable logging")
	fmt.Println("  --disable                   Disable logging")
	fmt.Println("  --retention <period>        Set log retention period (e.g., 7d, 30d, 90d, 1y)")
	fmt.Println("  --loki-size <size>          Set Loki storage size (e.g., 10Gi, 50Gi, 100Gi)")
	fmt.Println("  --promtail-size <size>      Set Promtail storage size (e.g., 1Gi, 5Gi, 10Gi)")
	fmt.Println("  --show                      Show current logging configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  trh-sdk log-config --enable")
	fmt.Println("  trh-sdk log-config --retention 7d")
	fmt.Println("  trh-sdk log-config --loki-size 100Gi --promtail-size 10Gi")
	fmt.Println("  trh-sdk log-config --show")
	return nil
}
