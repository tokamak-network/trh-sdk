package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

// ActionAlertConfig handles alert configuration commands
func ActionAlertConfig() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Check if monitoring plugin is installed
		if err := checkMonitoringPluginInstalled(ctx); err != nil {
			return err
		}

		// Get flags
		status := cmd.Bool("status")
		channel := cmd.String("channel")
		disable := cmd.Bool("disable")
		configure := cmd.Bool("configure")
		rule := cmd.String("rule")

		// Handle status command
		if status {
			return handleAlertStatus(ctx)
		}

		// Handle channel commands
		if channel != "" {
			if disable {
				return handleChannelDisable(ctx, channel)
			}
			if configure {
				return handleChannelConfigure(ctx, channel)
			}
			// If no operation specified, show help
			fmt.Println("❌ Please specify an operation: --disable or --configure")
			return nil
		}

		// Handle rule command
		if rule != "" {
			return handleRuleCommand(ctx, rule)
		}

		// Show help if no valid command
		return showAlertConfigHelp()
	}
}

// handleChannelDisable disables the specified channel
func handleChannelDisable(ctx context.Context, channelType string) error {
	ac := &thanos.AlertCustomization{}

	switch channelType {
	case "email":
		return disableEmailChannel(ctx, ac)
	case "telegram":
		return disableTelegramChannel(ctx, ac)
	default:
		return fmt.Errorf("unknown channel type: %s (must be 'email' or 'telegram')", channelType)
	}
}

// handleChannelConfigure configures the specified channel
func handleChannelConfigure(ctx context.Context, channelType string) error {
	ac := &thanos.AlertCustomization{}

	switch channelType {
	case "email":
		return configureEmailChannel(ctx, ac)
	case "telegram":
		return configureTelegramChannel(ctx, ac)
	default:
		return fmt.Errorf("unknown channel type: %s (must be 'email' or 'telegram')", channelType)
	}
}

// checkMonitoringPluginInstalled checks if monitoring plugin is installed
func checkMonitoringPluginInstalled(ctx context.Context) error {
	// Check if alert namespace exists
	namespaceExists, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check monitoring namespace: %w", err)
	}

	if !namespaceExists {
		fmt.Println("❌ Monitoring plugin is not installed!")
		fmt.Println()
		fmt.Println("To install monitoring plugin, run:")
		fmt.Println("  trh-sdk install monitoring")
		fmt.Println()
		fmt.Println("After installation, you can customize alert settings.")
		return fmt.Errorf("monitoring plugin not installed")
	}

	// Check if alert Helm release exists
	releaseExists, err := checkAlertReleaseExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check monitoring release: %w", err)
	}

	if !releaseExists {
		fmt.Println("❌ Monitoring plugin is not properly installed!")
		fmt.Println()
		fmt.Println("To install monitoring plugin, run:")
		fmt.Println("  trh-sdk install monitoring")
		fmt.Println()
		fmt.Println("After installation, you can customize alert settings.")
		return fmt.Errorf("monitoring plugin not properly installed")
	}
	return nil
}

// checkNamespaceExists checks if a namespace exists
func checkNamespaceExists(ctx context.Context, namespace string) (bool, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

// checkAlertReleaseExists checks if alert Helm release exists
func checkAlertReleaseExists(ctx context.Context) (bool, error) {
	output, err := utils.ExecuteCommand(ctx, "helm", "list", "-n", "monitoring", "--output", "json")
	if err != nil {
		return false, err
	}

	// Simple check for alert-related releases
	return strings.Contains(output, "monitoring"), nil
}

// showAlertConfigHelp displays help for alert configuration
func showAlertConfigHelp() error {
	fmt.Println("🔧 Alert Configuration Commands")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  trh-sdk alert-config [--status|--channel|--rule] [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  --status                    - Show current alert status and rules")
	fmt.Println("  --channel <type> --disable  - Disable notification channel (email/telegram)")
	fmt.Println("  --channel <type> --configure- Configure notification channel (email/telegram)")
	fmt.Println("  --rule <action>             - Manage alert rules (reset/set)")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Check alert status")
	fmt.Println("  trh-sdk alert-config --status")
	fmt.Println()
	fmt.Println("  # Disable email channel")
	fmt.Println("  trh-sdk alert-config --channel email --disable")
	fmt.Println()
	fmt.Println("  # Configure email channel")
	fmt.Println("  trh-sdk alert-config --channel email --configure")
	fmt.Println()
	fmt.Println("  # Disable telegram channel")
	fmt.Println("  trh-sdk alert-config --channel telegram --disable")
	fmt.Println()
	fmt.Println("  # Configure telegram channel")
	fmt.Println("  trh-sdk alert-config --channel telegram --configure")
	fmt.Println()
	fmt.Println("  # Interactive rule configuration")
	fmt.Println("  trh-sdk alert-config --rule set")
	fmt.Println()

	fmt.Println()
	fmt.Println("💡 Use 'trh-sdk alert-config --status' to see detailed rule status")
	fmt.Println("💡 Use 'trh-sdk alert-config --rule set' for interactive rule management")
	return nil
}

// handleAlertStatus shows current alert configuration
func handleAlertStatus(ctx context.Context) error {
	ac := &thanos.AlertCustomization{}

	// Check monitoring namespace
	_, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check alert namespace: %w", err)
	}

	// Get AlertManager configuration
	alertManagerConfig, err := ac.GetAlertManagerConfig(ctx)
	if err != nil {
		alertManagerConfig = ""
	}

	fmt.Println("📊 Alert Status Summary:")
	fmt.Println("========================")
	fmt.Printf("   📧 Email channel: %s\n", ac.GetChannelStatus(alertManagerConfig, "email"))
	fmt.Printf("   📱 Telegram channel: %s\n", ac.GetChannelStatus(alertManagerConfig, "telegram"))

	// Display configuration details
	if alertManagerConfig != "" {
		fmt.Println()
		fmt.Println("📋 Channel Configuration Details:")
		fmt.Println("================================")

		// Email configuration
		emailConfig := ac.GetEmailConfiguration(alertManagerConfig)
		if enabled, ok := emailConfig["enabled"].(bool); ok && enabled {
			fmt.Printf("   📧 Email Configuration:\n")
			fmt.Printf("      SMTP URL: %s\n", emailConfig["smtp_url"])
			fmt.Printf("      From: %s\n", emailConfig["from"])
			fmt.Printf("      To: %s\n", emailConfig["to"])
		}

		// Telegram configuration
		telegramConfig := ac.GetTelegramConfiguration(alertManagerConfig)
		if enabled, ok := telegramConfig["enabled"].(bool); ok && enabled {
			fmt.Printf("   📱 Telegram Configuration:\n")
			fmt.Printf("      Bot Token: %s\n", telegramConfig["bot_token"])
			fmt.Printf("      Chat ID: %s\n", telegramConfig["chat_id"])
		}
	}

	// Display detailed rule status
	fmt.Println()
	fmt.Println("📊 Alert Rules Status:")
	fmt.Println("======================")

	// Get current PrometheusRule
	allRules, err := ac.GetPrometheusRules(ctx)
	if err != nil {
		fmt.Println("⚠️  Could not get current PrometheusRules")
		return nil
	}

	if len(allRules) == 0 {
		fmt.Println("⚠️  No rules found in any PrometheusRule")
		return nil
	}

	// Count active rules
	activeRuleCount := 0
	for _, rule := range allRules {
		if _, exists := rule["alert"]; exists {
			activeRuleCount++
		}
	}

	fmt.Printf("   📋 Alert Rules: %d active\n", activeRuleCount)

	// Define all rules with their categories
	expectedRules := map[string]map[string]string{
		"Core System Alerts": {
			"OpNodeDown":     "OP Node down detection",
			"OpBatcherDown":  "OP Batcher down detection",
			"OpProposerDown": "OP Proposer down detection",
			"OpGethDown":     "OP Geth down detection",
			"L1RpcDown":      "L1 RPC connection failure",
		},
		"Configurable Alerts": {
			"OpBatcherBalanceCritical":  "OP Batcher balance threshold",
			"OpProposerBalanceCritical": "OP Proposer balance threshold",
			"BlockProductionStalled":    "Block production stall detection",
			"ContainerCpuUsageHigh":     "Container CPU usage threshold",
			"ContainerMemoryUsageHigh":  "Container memory usage threshold",
			"PodCrashLooping":           "Pod crash loop detection",
		},
	}

	// Check each rule category
	for category, ruleMap := range expectedRules {
		fmt.Printf("\n%s:\n", category)
		fmt.Println(strings.Repeat("-", len(category)+1))

		for ruleName, description := range ruleMap {
			found := false
			var currentValue string
			var severity string

			for _, rule := range allRules {
				if alertName, exists := rule["alert"]; exists && alertName == ruleName {
					found = true
					// Extract current value from expression
					if expr, ok := rule["expr"].(string); ok {
						currentValue = ac.ExtractValueFromExpression(ruleName, expr)
					}
					// Extract severity
					if labels, ok := rule["labels"].(map[string]interface{}); ok {
						if sev, exists := labels["severity"]; exists {
							severity = fmt.Sprintf("%v", sev)
						}
					}
					break
				}
			}

			status := "🔴 Disabled"
			if found {
				status = "🟢 Enabled"
			}

			fmt.Printf("   %s: %s", ruleName, status)
			if found && currentValue != "" {
				fmt.Printf(" (Current: %s)", currentValue)
			} else if !found && category == "Configurable Alerts" {
				// Show default value for disabled configurable rules
				defaultValues := map[string]string{
					"OpBatcherBalanceCritical":  "0.01",
					"OpProposerBalanceCritical": "0.01",
					"BlockProductionStalled":    "1m",
					"ContainerCpuUsageHigh":     "80",
					"ContainerMemoryUsageHigh":  "80",
					"PodCrashLooping":           "2m",
				}
				if defaultValue, exists := defaultValues[ruleName]; exists {
					fmt.Printf(" (Default: %s)", defaultValue)
				}
			}
			if severity != "" {
				fmt.Printf(" [%s]", severity)
			}
			fmt.Printf(" - %s\n", description)
		}
	}

	fmt.Println()
	fmt.Println("💡 Use 'trh-sdk alert-config --rule set' to modify configurable rules")
	fmt.Println("💡 Use 'trh-sdk alert-config --channel email --configure' to configure email")
	fmt.Println("💡 Use 'trh-sdk alert-config --channel telegram --configure' to configure telegram")

	return nil
}

// handleRuleCommand handles rule-related commands
func handleRuleCommand(ctx context.Context, ruleAction string) error {
	ac := &thanos.AlertCustomization{}

	switch ruleAction {
	case "reset":
		return resetAlertRules(ctx, ac)
	case "set":
		return configureAlertRules(ctx, ac)
	default:
		return fmt.Errorf("unknown rule action: %s (must be 'reset' or 'set')", ruleAction)
	}
}

// configureAlertRules allows users to configure alert rules interactively
func configureAlertRules(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("🔧 Alert Rules Configuration")
	fmt.Println("============================")

	// Define available rules with numbers
	rules := map[string]string{
		"1": "OpBatcherBalanceCritical",
		"2": "OpProposerBalanceCritical",
		"3": "BlockProductionStalled",
		"4": "ContainerCpuUsageHigh",
		"5": "ContainerMemoryUsageHigh",
		"6": "PodCrashLooping",
	}

	fmt.Println("\nAvailable rules to configure:")
	fmt.Println("1.  OpBatcherBalanceCritical - OP Batcher balance threshold")
	fmt.Println("2.  OpProposerBalanceCritical - OP Proposer balance threshold")
	fmt.Println("3.  BlockProductionStalled - Block production stall detection")
	fmt.Println("4.  ContainerCpuUsageHigh - Container CPU usage threshold")
	fmt.Println("5.  ContainerMemoryUsageHigh - Container memory usage threshold")
	fmt.Println("6.  PodCrashLooping - Pod crash loop detection")
	fmt.Println()
	fmt.Println("⚠️  Note: Core system alerts (OpNodeDown, OpBatcherDown, OpProposerDown, OpGethDown, L1RpcDown)")
	fmt.Println("    are essential and cannot be modified to ensure system stability.")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("- Enter rule number (1-6) to configure threshold/value")
	fmt.Println("- Enter 'enable <number>' to enable a rule")
	fmt.Println("- Enter 'disable <number>' to disable a rule")
	fmt.Println("- Enter 'quit' to exit")
	fmt.Println()
	fmt.Println("💡 Use 'trh-sdk alert-config --status' to see detailed rule status")

	fmt.Print("\nEnter command: ")
	ruleInput, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read rule selection: %w", err)
	}

	if ruleInput == "quit" {
		fmt.Println("Configuration cancelled")
		return nil
	}

	// Handle enable/disable commands
	if strings.HasPrefix(ruleInput, "enable ") {
		parts := strings.Fields(ruleInput)
		if len(parts) != 2 {
			fmt.Println("❌ Invalid command format. Use: enable <number>")
			return nil
		}
		ruleName, exists := rules[parts[1]]
		if !exists {
			fmt.Printf("❌ Invalid rule number: %s\n", parts[1])
			return nil
		}
		return enableRule(ctx, ac, ruleName)
	}

	if strings.HasPrefix(ruleInput, "disable ") {
		parts := strings.Fields(ruleInput)
		if len(parts) != 2 {
			fmt.Println("❌ Invalid command format. Use: disable <number>")
			return nil
		}
		ruleName, exists := rules[parts[1]]
		if !exists {
			fmt.Printf("❌ Invalid rule number: %s\n", parts[1])
			return nil
		}
		return disableRule(ctx, ac, ruleName)
	}

	// Get rule name from number for configuration
	ruleName, exists := rules[ruleInput]
	if !exists {
		fmt.Printf("❌ Invalid command: %s\n", ruleInput)
		fmt.Println("Valid commands: rule number (1-6), enable <number>, disable <number>, quit")
		return nil
	}

	// Configure based on rule type
	switch ruleName {
	case "OpBatcherBalanceCritical", "OpProposerBalanceCritical":
		return configureBalanceThreshold(ctx, ac, ruleName)
	case "BlockProductionStalled":
		return configureBlockProductionStall(ctx, ac)
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh":
		return configureUsageThreshold(ctx, ac, ruleName)
	case "PodCrashLooping":
		return configurePodCrashLoop(ctx, ac)
	default:
		fmt.Printf("❌ Unknown rule: %s\n", ruleName)
		return nil
	}
}

// validateRuleInput validates common rule input parameters
func validateRuleInput(input, inputType string) error {
	if input == "" {
		return fmt.Errorf("%s cannot be empty", inputType)
	}
	return nil
}

// configureBalanceThreshold configures balance threshold for batcher/proposer
func configureBalanceThreshold(ctx context.Context, ac *thanos.AlertCustomization, ruleName string) error {
	fmt.Printf("💰 Configuring %s balance threshold\n", ruleName)
	fmt.Println("Enter the minimum ETH balance threshold (e.g., 0.01, 0.1, 1.0):")

	fmt.Print("Balance threshold (ETH): ")
	threshold, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read balance threshold: %w", err)
	}

	// Validate input
	if err := validateRuleInput(threshold, "balance threshold"); err != nil {
		return err
	}

	fmt.Printf("🔧 Configuring %s with balance threshold '%s' ETH...\n", ruleName, threshold)
	if err := ac.UpdatePrometheusRule(ctx, ruleName, threshold); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ %s configured successfully with threshold %s ETH\n", ruleName, threshold)
	return nil
}

// configureBlockProductionStall configures block production stall detection
func configureBlockProductionStall(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("⏱️  Configuring BlockProductionStalled detection time")
	fmt.Println("Enter the time duration for block production stall detection:")
	fmt.Println("Examples: 30s, 1m, 2m, 5m")

	fmt.Print("Stall detection time: ")
	stallTime, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read stall time: %w", err)
	}

	// Validate input
	if err := validateRuleInput(stallTime, "stall detection time"); err != nil {
		return err
	}

	fmt.Printf("🔧 Configuring BlockProductionStalled with detection time '%s'...\n", stallTime)
	if err := ac.UpdatePrometheusRule(ctx, "BlockProductionStalled", stallTime); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ BlockProductionStalled configured successfully with detection time %s\n", stallTime)
	return nil
}

// configureUsageThreshold configures CPU/Memory usage threshold
func configureUsageThreshold(ctx context.Context, ac *thanos.AlertCustomization, ruleName string) error {
	var resourceType string
	if ruleName == "ContainerCpuUsageHigh" {
		resourceType = "CPU"
	} else {
		resourceType = "Memory"
	}

	fmt.Printf("📊 Configuring %s usage threshold\n", resourceType)
	fmt.Println("Enter the usage percentage threshold (e.g., 70, 80, 90):")

	fmt.Print("Usage threshold (%): ")
	threshold, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read usage threshold: %w", err)
	}

	// Validate input
	if err := validateRuleInput(threshold, "usage threshold"); err != nil {
		return err
	}

	fmt.Printf("🔧 Configuring %s with usage threshold '%s%%'...\n", ruleName, threshold)
	if err := ac.UpdatePrometheusRule(ctx, ruleName, threshold); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ %s configured successfully with threshold %s%%\n", ruleName, threshold)
	return nil
}

// configurePodCrashLoop configures pod crash loop detection time
func configurePodCrashLoop(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("🔄 Configuring PodCrashLooping detection time")
	fmt.Println("Enter the time duration for pod crash loop detection:")
	fmt.Println("Examples: 1m, 2m, 5m, 10m")

	fmt.Print("Crash loop detection time: ")
	detectionTime, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read detection time: %w", err)
	}

	// Validate input
	if err := validateRuleInput(detectionTime, "detection time"); err != nil {
		return err
	}

	fmt.Printf("🔧 Configuring PodCrashLooping with detection time '%s'...\n", detectionTime)
	if err := ac.UpdatePrometheusRule(ctx, "PodCrashLooping", detectionTime); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ PodCrashLooping configured successfully with detection time %s\n", detectionTime)
	return nil
}

// Email channel management functions
func disableEmailChannel(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("📧 Disabling Email Channel...")

	// Get current AlertManager configuration
	config, err := ac.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if email is already disabled
	if ac.GetChannelStatus(config, "email") == "Disabled" {
		fmt.Println("ℹ️  Email channel is already disabled")
		return nil
	}

	// Remove email configuration from AlertManager
	fmt.Println("🔧 Removing email configuration from AlertManager...")
	if err := ac.RemoveEmailConfig(ctx); err != nil {
		return fmt.Errorf("failed to disable email channel: %w", err)
	}

	fmt.Println("✅ Email channel disabled successfully")
	return nil
}

func configureEmailChannel(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("📧 Configuring Email Channel...")

	// Get new email configuration from user
	fmt.Println("Enter email configuration:")

	fmt.Print("SMTP Server (e.g., smtp.gmail.com:587): ")
	smtpServer, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read SMTP server: %w", err)
	}

	fmt.Print("From Email Address: ")
	smtpFrom, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read from email: %w", err)
	}

	// Automatically use the "From Email Address" as SMTP username
	smtpUsername := smtpFrom
	fmt.Printf("✅ SMTP Username automatically set to: %s\n", smtpUsername)

	fmt.Print("SMTP Password: ")
	smtpPassword, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read SMTP password: %w", err)
	}

	// Clean up the password input
	smtpPassword = utils.CleanPasswordInput(smtpPassword)

	// Validate input
	if err := validateEmailInput(smtpServer, smtpFrom, smtpPassword); err != nil {
		return err
	}

	fmt.Print("Default Receivers (comma-separated): ")
	receiversInput, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read receivers: %w", err)
	}

	// Parse receivers
	var receivers []string
	if receiversInput != "" {
		receivers = strings.Split(receiversInput, ",")
		for i, receiver := range receivers {
			receivers[i] = strings.TrimSpace(receiver)
		}
	}

	fmt.Printf("📧 Email Configuration Summary:\n")
	fmt.Printf("   SMTP Server: %s\n", smtpServer)
	fmt.Printf("   From Address: %s\n", smtpFrom)
	fmt.Printf("   Username: %s\n", smtpUsername)
	fmt.Printf("   Receivers: %v\n", receivers)

	// Apply configuration to AlertManager
	fmt.Println("🔧 Applying email configuration to AlertManager...")
	if err := ac.UpdateEmailConfig(ctx, smtpServer, smtpFrom, smtpUsername, smtpPassword, receivers); err != nil {
		return fmt.Errorf("failed to update AlertManager configuration: %w", err)
	}

	fmt.Println("✅ Email channel configured successfully")
	return nil
}

// Telegram channel management functions
func disableTelegramChannel(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("📱 Disabling Telegram Channel...")

	// Get current AlertManager configuration
	config, err := ac.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if telegram is already disabled
	if ac.GetChannelStatus(config, "telegram") == "Disabled" {
		fmt.Println("ℹ️  Telegram channel is already disabled")
		return nil
	}

	// Remove telegram configuration from AlertManager
	fmt.Println("🔧 Removing telegram configuration from AlertManager...")
	if err := ac.RemoveTelegramConfig(ctx); err != nil {
		return fmt.Errorf("failed to disable telegram channel: %w", err)
	}

	fmt.Println("✅ Telegram channel disabled successfully")
	return nil
}

func configureTelegramChannel(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("📱 Configuring Telegram Channel...")

	// Get new telegram configuration from user
	fmt.Println("Enter telegram configuration:")

	fmt.Print("Bot API Token: ")
	botToken, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read bot token: %w", err)
	}

	fmt.Print("Chat ID: ")
	chatID, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read chat ID: %w", err)
	}

	// Validate input
	if err := validateTelegramInput(botToken, chatID); err != nil {
		fmt.Printf("❌ %s\n", err.Error())
		return err
	}

	fmt.Printf("📱 Telegram Configuration Summary:\n")
	fmt.Printf("   Bot Token: %s...\n", botToken[:min(len(botToken), 10)])
	fmt.Printf("   Chat ID: %s\n", chatID)

	// Apply configuration to AlertManager
	fmt.Println("🔧 Applying telegram configuration to AlertManager...")
	if err := ac.UpdateTelegramConfig(ctx, botToken, chatID); err != nil {
		return fmt.Errorf("failed to update AlertManager configuration: %w", err)
	}

	fmt.Println("✅ Telegram channel configured successfully")
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Alert rules management functions
func resetAlertRules(ctx context.Context, ac *thanos.AlertCustomization) error {
	fmt.Println("🔄 Resetting Alert Rules to Default...")
	fmt.Println("⚠️  This will reset all alert rules to their default values.")
	fmt.Print("Are you sure you want to reset all alert rules? (y/N): ")

	confirm, err := scanner.ScanBool(false)
	if err != nil {
		return err
	}

	if confirm {
		fmt.Println("🔧 Resetting PrometheusRules to default...")

		// Get current rules first
		currentRules, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
		if err == nil && strings.TrimSpace(currentRules) != "" {
			ruleNames := strings.Split(strings.TrimSpace(currentRules), " ")
			fmt.Printf("Found %d PrometheusRule(s) to reset:\n", len(ruleNames))
			for _, ruleName := range ruleNames {
				if ruleName != "" {
					fmt.Printf("  - %s\n", ruleName)
				}
			}
		}

		if err := ac.ResetPrometheusRules(ctx); err != nil {
			return fmt.Errorf("failed to reset alert rules: %w", err)
		}

		fmt.Println("✅ Alert rules reset to default successfully")
		fmt.Println("💡 Prometheus will reload rules automatically")
	} else {
		fmt.Println("❌ Reset cancelled")
	}

	return nil
}

// Validation functions
func validateChannelInput(input, inputType string) error {
	if input == "" {
		return fmt.Errorf("%s cannot be empty", inputType)
	}
	return nil
}

func validateEmailInput(smtpServer, smtpFrom, smtpPassword string) error {
	if err := validateChannelInput(smtpServer, "SMTP server"); err != nil {
		return err
	}
	if err := validateChannelInput(smtpFrom, "From email address"); err != nil {
		return err
	}
	if err := validateChannelInput(smtpPassword, "SMTP password"); err != nil {
		return err
	}
	return nil
}

func validateTelegramInput(botToken, chatID string) error {
	if err := validateChannelInput(botToken, "Bot token"); err != nil {
		return err
	}
	if err := validateChannelInput(chatID, "Chat ID"); err != nil {
		return err
	}

	// Validate bot token format (basic check)
	if !strings.Contains(botToken, ":") {
		return fmt.Errorf("invalid bot token format. Expected format: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz")
	}

	// Validate chat ID format (basic check)
	if !strings.HasPrefix(chatID, "-") && !strings.HasPrefix(chatID, "1") {
		return fmt.Errorf("invalid chat ID format. Expected format: -123456789 or 123456789")
	}

	return nil
}

// Helper functions for rule management
func enableRule(ctx context.Context, ac *thanos.AlertCustomization, ruleName string) error {
	fmt.Printf("🟢 Enabling rule '%s'...\n", ruleName)

	if err := ac.EnableRule(ctx, ruleName); err != nil {
		return fmt.Errorf("failed to enable rule: %w", err)
	}

	fmt.Printf("✅ Rule '%s' enabled successfully\n", ruleName)
	return nil
}

func disableRule(ctx context.Context, ac *thanos.AlertCustomization, ruleName string) error {
	fmt.Printf("🔴 Disabling rule '%s'...\n", ruleName)

	if err := ac.DisableRule(ctx, ruleName); err != nil {
		return fmt.Errorf("failed to disable rule: %w", err)
	}

	fmt.Printf("✅ Rule '%s' disabled successfully\n", ruleName)
	return nil
}
