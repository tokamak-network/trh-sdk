package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
	"go.uber.org/zap"
)

// ActionLogCollection handles the log-collection command
func ActionLogCollection() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get deployment path once
		deploymentPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		// Initialize logger with correct filename
		fileName := fmt.Sprintf("%s/logs/log_collection_%d.log", deploymentPath, time.Now().Unix())
		logger, err := logging.InitLogger(fileName)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		logger.Info("🚀 Starting log-collection command...")

		// Check if monitoring plugin is installed
		if err := utils.CheckMonitoringPluginInstalled(ctx); err != nil {
			logger.Errorw("Failed to check monitoring plugin", "err", err)
			return err
		}

		// Get flags
		enable := cmd.Bool("enable")
		disable := cmd.Bool("disable")
		retention := cmd.String("retention")
		interval := cmd.String("interval")
		show := cmd.Bool("show")
		download := cmd.Bool("download")
		component := cmd.String("component")
		hours := cmd.String("hours")
		minutes := cmd.String("minutes")
		keyword := cmd.String("keyword")

		// Log only relevant flags based on command type
		if download {
			logger.Infow("Download flags", "component", component, "hours", hours, "minutes", minutes, "keyword", keyword)
		} else {
			logger.Infow("Configuration flags", "enable", enable, "disable", disable, "retention", retention, "interval", interval, "show", show)
		}

		// Load current configuration
		config, err := loadCurrentConfig(deploymentPath)
		if err != nil {
			logger.Errorw("Failed to load current config", "err", err)
			return err
		}

		// Initialize ThanosStack
		thanosStack, err := thanos.NewThanosStack(ctx, logger, config.Network, true, deploymentPath, config.AWS)
		if err != nil {
			logger.Errorw("Failed to initialize ThanosStack", "err", err)
			return fmt.Errorf("failed to initialize ThanosStack: %w", err)
		}

		// Handle download command
		if download {
			if component == "" && hours == "" && minutes == "" && keyword == "" {
				return showDownloadHelp()
			}
			return handleLogDownload(ctx, config, thanosStack, logger, component, hours, minutes, keyword)
		}

		// Handle show command
		if show {
			return handleLogConfigShow(ctx)
		}

		// Initialize logging config if it doesn't exist
		if config.LoggingConfig == nil {
			config.LoggingConfig = &types.LoggingConfig{
				Enabled:             false,
				CloudWatchRetention: 30,
				CollectionInterval:  30,
				LogStreamPrefix:     "fluentbit-sidecar",
				Components:          "op-node,op-geth,op-batcher,op-proposer",
			}
		}

		// Track if any changes were made
		hasChanges := false

		// Handle enable/disable
		if enable {
			if config.LoggingConfig.Enabled {
				logger.Info("ℹ️  Log collection is already enabled.")
				return nil
			}
			config.LoggingConfig.Enabled = true
			hasChanges = true
		}
		if disable {
			config.LoggingConfig.Enabled = false
			hasChanges = true
		}

		// Handle retention setting
		if retention != "" {
			retentionDays, err := strconv.Atoi(retention)
			if err != nil {
				logger.Errorw("Invalid retention value", "retention", retention, "err", err)
				return fmt.Errorf("invalid retention value: %s. Please provide a number of days (e.g., 7, 30, 90)", retention)
			}
			config.LoggingConfig.CloudWatchRetention = retentionDays
			hasChanges = true
		}

		// Handle collection interval
		if interval != "" {
			intervalSeconds, err := strconv.Atoi(interval)
			if err != nil {
				logger.Errorw("Invalid interval value", "interval", interval, "err", err)
				return fmt.Errorf("invalid interval value: %s. Please provide a number of seconds (e.g., 30, 60, 120)", interval)
			}
			config.LoggingConfig.CollectionInterval = intervalSeconds
			hasChanges = true
		}

		// Save updated configuration to settings.json
		if hasChanges {
			if err := config.WriteToJSONFile("."); err != nil {
				logger.Errorw("Failed to save configuration", "err", err)
				return fmt.Errorf("failed to save configuration: %w", err)
			}
			logger.Info("✅ Configuration saved to settings.json")
		}

		// Apply changes to running monitoring plugin
		if hasChanges {
			logger.Info("Applying logging configuration changes...")
			return applyLoggingConfig(ctx, config, thanosStack, logger)
		}

		// Show help if no valid command
		return showLogConfigHelp()
	}
}

// loadCurrentConfig loads the current configuration from settings.json
func loadCurrentConfig(deploymentPath string) (*types.Config, error) {
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
func applyLoggingConfig(ctx context.Context, config *types.Config, thanosStack *thanos.ThanosStack, logger *zap.SugaredLogger) error {
	logger.Info("Applying logging configuration to monitoring plugin...")

	// Create monitoring config with logging settings
	monitoringConfig := &types.MonitoringConfig{
		Namespace:       config.ChainName,
		HelmReleaseName: "monitoring",
		ChainName:       config.ChainName,
		LoggingEnabled:  config.LoggingConfig.Enabled,
	}

	// Apply logging configuration
	if err := UpdateLoggingConfig(ctx, thanosStack, monitoringConfig, logger); err != nil {
		logger.Errorw("Failed to update logging configuration", "err", err)
		return fmt.Errorf("failed to update logging configuration: %w", err)
	}

	logger.Info("✅ Logging configuration applied successfully!")
	return nil
}

// handleLogConfigShow displays current logging configuration
func handleLogConfigShow(ctx context.Context) error {
	deploymentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	config, err := loadCurrentConfig(deploymentPath)
	if err != nil {
		return err
	}

	if config.LoggingConfig == nil {
		fmt.Println("No logging configuration found.")
		return nil
	}

	fmt.Println("Current Logging Configuration:")
	fmt.Printf("  Enabled: %t\n", config.LoggingConfig.Enabled)

	if config.LoggingConfig.CloudWatchRetention > 0 {
		fmt.Printf("  CloudWatch Retention: %d days\n", config.LoggingConfig.CloudWatchRetention)
	} else {
		fmt.Println("  CloudWatch Retention: 30 days (default)")
	}

	if config.LoggingConfig.CollectionInterval > 0 {
		fmt.Printf("  Collection Interval: %d seconds\n", config.LoggingConfig.CollectionInterval)
	} else {
		fmt.Println("  Collection Interval: 30 seconds (default)")
	}

	return nil
}

// showLogConfigHelp displays help information
func showLogConfigHelp() error {
	fmt.Println("Usage: trh-sdk log-collection [OPTIONS]")
	fmt.Println()
	fmt.Println("Log Configuration Options:")
	fmt.Println("  --enable                    Enable CloudWatch log collection")
	fmt.Println("  --disable                   Disable CloudWatch log collection")
	fmt.Println("  --retention <days>          Set CloudWatch log retention period in days (e.g., 7, 30, 90)")
	fmt.Println("  --interval <seconds>        Set log collection interval in seconds (e.g., 30, 60, 120)")
	fmt.Println("  --show                      Show current logging configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Log configuration")
	fmt.Println("  trh-sdk log-collection --enable")
	fmt.Println("  trh-sdk log-collection --retention 30")
	fmt.Println("  trh-sdk log-collection --interval 60")
	fmt.Println("  trh-sdk log-collection --show")
	fmt.Println()
	fmt.Println("  # Log download from running components")
	fmt.Println("  trh-sdk log-collection --download --component op-node --hours 7")
	fmt.Println("  trh-sdk log-collection --download --component all --hours 24 --keyword error")
	fmt.Println("  trh-sdk log-collection --download --component op-geth --minutes 30 --keyword warning")
	fmt.Println()
	fmt.Println("  # Apply all settings at once")
	fmt.Println("  trh-sdk log-collection --enable --retention 90 --interval 60")
	fmt.Println()
	fmt.Println("For download options, use: trh-sdk log-collection --download --help")
	return nil
}

// showDownloadHelp displays help information for download options
func showDownloadHelp() error {
	fmt.Println("Usage: trh-sdk log-collection --download [OPTIONS]")
	fmt.Println()
	fmt.Println("Download Options:")
	fmt.Println("  --component <name>          Component to download logs from (op-node, op-geth, op-batcher, op-proposer, all)")
	fmt.Println("  --hours <number>            Number of hours to look back for logs")
	fmt.Println("  --minutes <number>          Number of minutes to look back for logs")
	fmt.Println("  --keyword <text>            Keyword to filter logs (case-insensitive)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Download logs for specific component")
	fmt.Println("  trh-sdk log-collection --download --component op-node --hours 7")
	fmt.Println("  trh-sdk log-collection --download --component op-geth --minutes 30")
	fmt.Println()
	fmt.Println("  # Download logs for all components with keyword filter")
	fmt.Println("  trh-sdk log-collection --download --component all --hours 24 --keyword error")
	fmt.Println("  trh-sdk log-collection --download --component op-node --hours 1 --keyword warning")
	fmt.Println()
	fmt.Println("  # Download logs with time and keyword filters")
	fmt.Println("  trh-sdk log-collection --download --component op-batcher --hours 12 --keyword failed")
	return nil
}

// UpdateLoggingConfig updates the logging configuration for the monitoring stack
func UpdateLoggingConfig(ctx context.Context, thanosStack *thanos.ThanosStack, config *types.MonitoringConfig, logger *zap.SugaredLogger) error {
	// Get actual namespace where Thanos Stack components are deployed
	actualNamespace, err := thanosStack.GetActualNamespace(ctx)
	if err != nil {
		return fmt.Errorf("failed to get actual namespace: %w", err)
	}

	// Get logging configuration from deploy config
	var loggingConfig *types.LoggingConfig
	deployConfig := thanosStack.GetDeployConfig()
	if deployConfig != nil && deployConfig.LoggingConfig != nil {
		loggingConfig = deployConfig.LoggingConfig
	} else {
		// Use default values if no logging config exists
		loggingConfig = &types.LoggingConfig{
			Enabled:             true,
			CloudWatchRetention: 30,
			CollectionInterval:  30,
			LogStreamPrefix:     "fluentbit-sidecar",
			Components:          "op-node,op-geth,op-batcher,op-proposer",
		}
	}

	// Check if sidecar is already running
	cmd := exec.Command("kubectl", "get", "pods", "-n", actualNamespace,
		"-l", "app=thanos-logs-sidecar",
		"--no-headers", "-o", "custom-columns=NAME:.metadata.name")

	output, err := cmd.CombinedOutput()
	sidecarRunning := err == nil && strings.TrimSpace(string(output)) != ""

	if loggingConfig.Enabled {
		if !sidecarRunning {
			// Sidecar is not running, install it
			if err := thanosStack.InstallFluentBitSidecar(ctx, actualNamespace, loggingConfig); err != nil {
				return fmt.Errorf("failed to install new sidecar deployments: %w", err)
			}
		} else {
			// Sidecar is running, update specific settings without restart

			// Update retention policy if changed
			if loggingConfig.CloudWatchRetention > 0 {
				if err := thanosStack.UpdateRetentionPolicy(ctx, actualNamespace, loggingConfig.CloudWatchRetention); err != nil {
					logger.Warnw("Failed to update retention policy", "err", err)
				} else {
					fmt.Println("✅ Retention policy updated successfully")
				}
			}

			// Update collection interval if changed
			if loggingConfig.CollectionInterval > 0 {
				if err := thanosStack.UpdateCollectionInterval(ctx, actualNamespace, loggingConfig.CollectionInterval); err != nil {
					logger.Warnw("Failed to update collection interval", "err", err)
				} else {
					fmt.Println("✅ Collection interval updated successfully")
				}
			}
		}
	} else {
		// Disable logging - remove sidecar
		if sidecarRunning {
			if err := thanosStack.CleanupSidecarDeployments(ctx, actualNamespace); err != nil {
				logger.Warnw("Failed to cleanup existing sidecar deployments", "err", err)
			}
			if err := thanosStack.CleanupRBACResources(ctx); err != nil {
				logger.Warnw("Failed to cleanup existing RBAC resources", "err", err)
			}
		}
	}

	// Generate new values file with updated logging configuration
	if err := thanosStack.GenerateValuesFile(config); err != nil {
		return fmt.Errorf("failed to generate values file: %w", err)
	}

	// Verify the changes
	if loggingConfig.Enabled && sidecarRunning {
		fmt.Println("\n🔍 Configuration Change Verification:")
		if err := thanosStack.VerifyRetentionPolicy(ctx, actualNamespace); err != nil {
			logger.Warnw("Failed to verify retention policy", "err", err)
		}
		if err := thanosStack.VerifyCollectionInterval(ctx, actualNamespace); err != nil {
			logger.Warnw("Failed to verify collection interval", "err", err)
		}
	}

	fmt.Printf("✅ Logging configuration updated successfully\n")
	return nil
}

// handleLogDownload handles log download functionality
func handleLogDownload(ctx context.Context, config *types.Config, thanosStack *thanos.ThanosStack, logger *zap.SugaredLogger, component, hours, minutes, keyword string) error {
	logger.Info("📥 Starting log download...")

	// Get actual namespace
	actualNamespace, err := thanosStack.GetActualNamespace(ctx)
	if err != nil {
		logger.Errorw("Failed to get actual namespace", "err", err)
		return fmt.Errorf("failed to get actual namespace: %w", err)
	}

	// Validate component
	validComponents := []string{"op-node", "op-geth", "op-batcher", "op-proposer", "all"}
	if component != "" && !contains(validComponents, component) {
		logger.Errorw("Invalid component", "component", component, "valid_components", validComponents)
		return fmt.Errorf("invalid component: %s. Valid components are: %v", component, validComponents)
	}

	// Calculate time duration
	duration, err := calculateDuration("", hours, minutes)
	if err != nil {
		logger.Errorw("Failed to calculate duration", "err", err)
		return err
	}

	// Create download directory
	downloadDir := fmt.Sprintf("./logs/download_%s", time.Now().Format("20060102_150405"))
	if err := os.MkdirAll(downloadDir, 0755); err != nil {
		logger.Errorw("Failed to create download directory", "err", err)
		return fmt.Errorf("failed to create download directory: %w", err)
	}

	logger.Infow("Download configuration", "component", component, "duration", duration, "keyword", keyword, "download_dir", downloadDir)

	// Download logs based on component selection
	if component == "all" || component == "" {
		// Download logs for all components
		for _, comp := range []string{"op-node", "op-geth", "op-batcher", "op-proposer"} {
			if err := downloadComponentLogs(ctx, logger, actualNamespace, comp, duration, keyword, downloadDir); err != nil {
				logger.Warnw("Failed to download logs for component", "component", comp, "err", err)
			}
		}
	} else {
		// Download logs for specific component
		if err := downloadComponentLogs(ctx, logger, actualNamespace, component, duration, keyword, downloadDir); err != nil {
			logger.Errorw("Failed to download logs", "component", component, "err", err)
			return err
		}
	}

	logger.Infow("✅ Log download completed", "download_dir", downloadDir)
	fmt.Printf("✅ Logs downloaded to: %s\n", downloadDir)
	return nil
}

// downloadComponentLogs downloads logs for a specific component
func downloadComponentLogs(ctx context.Context, logger *zap.SugaredLogger, namespace, component string, duration time.Duration, keyword, downloadDir string) error {
	logger.Infow("Downloading logs for component", "component", component, "namespace", namespace)

	// Get pod name for the component
	podName, err := getPodName(ctx, namespace, component)
	if err != nil {
		logger.Errorw("Failed to get pod name", "component", component, "err", err)
		return err
	}

	if podName == "" {
		logger.Warnw("No pod found for component", "component", component)
		return fmt.Errorf("no pod found for component: %s", component)
	}

	// Build kubectl logs command
	cmdArgs := []string{"logs", podName, "-n", namespace}

	// Add time duration if specified
	if duration > 0 {
		cmdArgs = append(cmdArgs, "--since", duration.String())
	}

	// Execute kubectl command
	cmd := exec.Command("kubectl", cmdArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Errorw("Failed to execute kubectl logs", "component", component, "err", err, "output", string(output))
		return fmt.Errorf("failed to get logs for %s: %w", component, err)
	}

	// Apply keyword filter if specified
	if keyword != "" {
		lines := strings.Split(string(output), "\n")
		var filteredLines []string
		for _, line := range lines {
			if strings.Contains(strings.ToLower(line), strings.ToLower(keyword)) {
				filteredLines = append(filteredLines, line)
			}
		}
		output = []byte(strings.Join(filteredLines, "\n"))
	}

	// Write logs to file
	fileName := fmt.Sprintf("%s/%s_%s.log", downloadDir, component, time.Now().Format("20060102_150405"))
	if err := os.WriteFile(fileName, output, 0644); err != nil {
		logger.Errorw("Failed to write log file", "file", fileName, "err", err)
		return fmt.Errorf("failed to write log file: %w", err)
	}

	logger.Infow("✅ Logs downloaded for component", "component", component, "file", fileName, "size", len(output))
	return nil
}

// getPodName gets the pod name for a specific component
func getPodName(ctx context.Context, namespace, component string) (string, error) {
	// Try different label patterns for finding the pod
	labelPatterns := []string{
		fmt.Sprintf("app=%s", component),
		fmt.Sprintf("app=%s-%s", namespace, component),
		fmt.Sprintf("app=%s-thanos-stack-%s", namespace, component),
	}

	for _, pattern := range labelPatterns {
		cmd := exec.Command("kubectl", "get", "pods", "-n", namespace,
			"-l", pattern,
			"--no-headers", "-o", "custom-columns=NAME:.metadata.name")

		output, err := cmd.CombinedOutput()
		if err == nil && strings.TrimSpace(string(output)) != "" {
			podName := strings.TrimSpace(string(output))
			return podName, nil
		}
	}

	// If no pod found with labels, try to find by name pattern
	cmd := exec.Command("kubectl", "get", "pods", "-n", namespace,
		"--no-headers", "-o", "custom-columns=NAME:.metadata.name")

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get pods for %s: %w", component, err)
	}

	podNames := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, podName := range podNames {
		if strings.Contains(podName, component) {
			return strings.TrimSpace(podName), nil
		}
	}

	return "", fmt.Errorf("no pod found for component: %s", component)
}

// calculateDuration calculates time duration from hours and minutes
func calculateDuration(days, hours, minutes string) (time.Duration, error) {
	var duration time.Duration

	if days != "" {
		d, err := strconv.Atoi(days)
		if err != nil {
			return 0, fmt.Errorf("invalid days value: %s", days)
		}
		duration += time.Duration(d) * 24 * time.Hour
	}

	if hours != "" {
		h, err := strconv.Atoi(hours)
		if err != nil {
			return 0, fmt.Errorf("invalid hours value: %s", hours)
		}
		duration += time.Duration(h) * time.Hour
	}

	if minutes != "" {
		m, err := strconv.Atoi(minutes)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes value: %s", minutes)
		}
		duration += time.Duration(m) * time.Minute
	}

	return duration, nil
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
