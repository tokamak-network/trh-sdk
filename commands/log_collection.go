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
		config, err := utils.ReadConfigFromJSONFile(deploymentPath)
		if err != nil {
			logger.Errorw("Failed to load current config", "err", err)
			return err
		}
		if config == nil {
			return fmt.Errorf("no configuration found. Please run 'trh-sdk deploy' first")
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
				fmt.Println("Download Options:")
				fmt.Println("  --component <name>          Component to download logs from (op-node, op-geth, op-batcher, op-proposer, all)")
				fmt.Println("  --hours <number>            Number of hours to look back for logs")
				fmt.Println("  --minutes <number>          Number of minutes to look back for logs")
				fmt.Println("  --keyword <text>            Keyword to filter logs (case-insensitive)")
				fmt.Println()
				fmt.Println("Examples:")
				fmt.Println("  trh-sdk log-collection --download --component op-node --hours 7")
				fmt.Println("  trh-sdk log-collection --download --component all --hours 24 --keyword error")
				return nil
			}
			return handleLogDownload(ctx, thanosStack, logger, component, hours, minutes, keyword)
		}

		// Handle show command
		if show {
			return handleLogConfigShow()
		}

		// Initialize logging config if it doesn't exist
		if config.LoggingConfig == nil {
			config.LoggingConfig = &types.LoggingConfig{
				Enabled:             false,
				CloudWatchRetention: 30,
				CollectionInterval:  30,
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

		// Save and apply changes
		if hasChanges {
			if err := config.WriteToJSONFile(deploymentPath); err != nil {
				logger.Errorw("Failed to save configuration", "err", err)
				return fmt.Errorf("failed to save configuration: %w", err)
			}
			logger.Info("✅ Configuration saved to settings.json")

			logger.Info("Applying logging configuration changes...")
			if err := UpdateLoggingConfig(ctx, thanosStack, logger, config.LoggingConfig); err != nil {
				logger.Errorw("Failed to update logging configuration", "err", err)
				return fmt.Errorf("failed to update logging configuration: %w", err)
			}
			logger.Info("✅ Logging configuration applied successfully!")
			return nil
		}

		// Show help if no valid command
		fmt.Println("Usage: trh-sdk log-collection [OPTIONS]")
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  --enable                    Enable CloudWatch log collection")
		fmt.Println("  --disable                   Disable CloudWatch log collection")
		fmt.Println("  --retention <days>          Set CloudWatch log retention period in days (e.g., 7, 30, 90)")
		fmt.Println("  --interval <seconds>        Set log collection interval in seconds (e.g., 30, 60, 120)")
		fmt.Println("  --show                      Show current logging configuration")
		fmt.Println()
		fmt.Println("Subcommand Download Options:")
		fmt.Println("  --download                  Download logs from running components")
		fmt.Println("  --component <name>          Component to download logs from (op-node, op-geth, op-batcher, op-proposer, all)")
		fmt.Println("  --hours <number>            Number of hours to look back for logs")
		fmt.Println("  --minutes <number>          Number of minutes to look back for logs")
		fmt.Println("  --keyword <text>            Keyword to filter logs (case-insensitive)")
		return nil
	}
}

// handleLogConfigShow displays current logging configuration
func handleLogConfigShow() error {
	deploymentPath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		return err
	}
	if config == nil {
		return fmt.Errorf("no configuration found. Please run 'trh-sdk deploy' first")
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

// UpdateLoggingConfig updates the logging configuration for the monitoring stack
func UpdateLoggingConfig(ctx context.Context, thanosStack *thanos.ThanosStack, logger *zap.SugaredLogger, loggingConfig *types.LoggingConfig) error {
	// Resolve namespace from deploy config and validate existence
	deployCfg := thanosStack.GetDeployConfig()
	if deployCfg == nil || deployCfg.K8s == nil || strings.TrimSpace(deployCfg.K8s.Namespace) == "" {
		return fmt.Errorf("k8s namespace is not configured in settings.json")
	}
	namespace := strings.TrimSpace(deployCfg.K8s.Namespace)
	if ok, err := utils.CheckNamespaceExists(ctx, namespace); err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	} else if !ok {
		return fmt.Errorf("namespace does not exist: %s", namespace)
	}

	// Use the provided loggingConfig if available, otherwise fall back to deploy config
	if loggingConfig == nil {
		deployConfig := thanosStack.GetDeployConfig()
		if deployConfig != nil && deployConfig.LoggingConfig != nil {
			loggingConfig = deployConfig.LoggingConfig
			logger.Infof("Using deploy config logging settings: Enabled=%t, Retention=%d, Interval=%d",
				loggingConfig.Enabled, loggingConfig.CloudWatchRetention, loggingConfig.CollectionInterval)
		} else {
			// Use default values if no logging config exists
			loggingConfig = &types.LoggingConfig{
				Enabled:             true,
				CloudWatchRetention: 30,
				CollectionInterval:  30,
			}
			logger.Info("Using default logging config")
		}
	} else {
		logger.Infof("Using provided logging config: Enabled=%t, Retention=%d, Interval=%d",
			loggingConfig.Enabled, loggingConfig.CloudWatchRetention, loggingConfig.CollectionInterval)
	}

	// Check if sidecar is already running (use kubectl util)
	pods, err := utils.GetPodNamesByLabel(ctx, namespace, "app=thanos-logs-sidecar")
	if err != nil {
		logger.Warnw("Failed to list sidecar pods", "err", err)
	}
	sidecarRunning := len(pods) > 0

	if loggingConfig.Enabled {
		if !sidecarRunning {
			// Sidecar is not running, install it
			if err := thanosStack.InstallLogCollectionSidecar(ctx, namespace, loggingConfig); err != nil {
				return fmt.Errorf("failed to install new sidecar deployments: %w", err)
			}
		} else {
			// Sidecar is running, update specific settings without restart

			// Update retention policy if changed
			if loggingConfig.CloudWatchRetention > 0 {
				if err := thanosStack.UpdateRetentionPolicy(ctx, namespace, loggingConfig.CloudWatchRetention); err != nil {
					logger.Warnw("Failed to update retention policy", "err", err)
				} else {
					logger.Info("✅ Retention policy updated successfully")
				}
			}

			// Update collection interval if changed
			if loggingConfig.CollectionInterval > 0 {
				if err := thanosStack.UpdateCollectionInterval(ctx, namespace, loggingConfig.CollectionInterval); err != nil {
					logger.Warnw("Failed to update collection interval", "err", err)
				} else {
					logger.Info("✅ Collection interval updated successfully")
				}
			}
		}
	} else {
		// Disable logging - remove sidecar
		if sidecarRunning {
			if err := thanosStack.CleanupSidecarDeployments(ctx, namespace); err != nil {
				logger.Warnw("Failed to cleanup existing sidecar deployments", "err", err)
			}
			if err := thanosStack.CleanupRBACResources(ctx); err != nil {
				logger.Warnw("Failed to cleanup existing RBAC resources", "err", err)
			}
		}
	}

	// Generate new values file with updated logging configuration
	// Note: config parameter was removed from UpdateLoggingConfig, so we need to create a minimal config
	monitoringConfig := &types.MonitoringConfig{
		Namespace:       "monitoring", // Default namespace
		HelmReleaseName: "monitoring",
		ChainName:       "thanos-stack",
		LoggingEnabled:  loggingConfig.Enabled,
	}

	if err := thanosStack.GenerateValuesFile(monitoringConfig); err != nil {
		return fmt.Errorf("failed to generate values file: %w", err)
	}

	logger.Info("✅ Logging configuration updated successfully")
	return nil
}

// handleLogDownload handles log download functionality
func handleLogDownload(ctx context.Context, thanosStack *thanos.ThanosStack, logger *zap.SugaredLogger, component, hours, minutes, keyword string) error {
	logger.Info("📥 Starting log download...")

	// Resolve namespace from deploy config and validate existence
	deployCfg := thanosStack.GetDeployConfig()
	if deployCfg == nil || deployCfg.K8s == nil || strings.TrimSpace(deployCfg.K8s.Namespace) == "" {
		return fmt.Errorf("k8s namespace is not configured in settings.json")
	}
	namespace := strings.TrimSpace(deployCfg.K8s.Namespace)
	if ok, err := utils.CheckNamespaceExists(ctx, namespace); err != nil {
		logger.Errorw("Failed to check namespace existence", "err", err)
		return fmt.Errorf("failed to check namespace existence: %w", err)
	} else if !ok {
		return fmt.Errorf("namespace does not exist: %s", namespace)
	}

	// Validate component
	validComponents := []string{"op-node", "op-geth", "op-batcher", "op-proposer", "all"}
	valid := false
	for _, validComp := range validComponents {
		if component == validComp {
			valid = true
			break
		}
	}
	if component != "" && !valid {
		logger.Errorw("Invalid component", "component", component, "valid_components", validComponents)
		return fmt.Errorf("invalid component: %s. Valid components are: %v", component, validComponents)
	}

	// Calculate time duration
	duration, err := calculateDuration(hours, minutes)
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
			if err := downloadComponentLogs(ctx, logger, namespace, comp, duration, keyword, downloadDir); err != nil {
				logger.Warnw("Failed to download logs for component", "component", comp, "err", err)
			}
		}
	} else {
		// Download logs for specific component
		if err := downloadComponentLogs(ctx, logger, namespace, component, duration, keyword, downloadDir); err != nil {
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
	// Try label selectors to find the pod name
	labelPatterns := []string{
		fmt.Sprintf("app=%s", component),
		fmt.Sprintf("app=%s-%s", namespace, component),
		fmt.Sprintf("app=%s-thanos-stack-%s", namespace, component),
	}

	for _, pattern := range labelPatterns {
		pods, err := utils.GetPodNamesByLabel(ctx, namespace, pattern)
		if err == nil && len(pods) > 0 {
			return pods[0], nil
		}
	}

	// Fallback: scan all pods and match by substring
	allPods, err := utils.GetK8sPods(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to list pods in namespace %s: %w", namespace, err)
	}
	for _, pod := range allPods {
		if strings.Contains(pod, component) {
			return pod, nil
		}
	}
	return "", fmt.Errorf("no pod found for component: %s", component)
}

// calculateDuration calculates time duration from hours and minutes
func calculateDuration(hours, minutes string) (time.Duration, error) {
	var duration time.Duration

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
