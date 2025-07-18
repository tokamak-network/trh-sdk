package commands

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
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
			return handleAlertStatus(ctx, cmd, cmd.Args().Slice())
		}

		// Handle channel commands
		if channel != "" {
			if disable {
				return handleChannelDisable(ctx, cmd, channel)
			}
			if configure {
				return handleChannelConfigure(ctx, cmd, channel)
			}
			// If no operation specified, show help
			fmt.Println("❌ Please specify an operation: --disable or --configure")
			return nil
		}

		// Handle rule command
		if rule != "" {
			return handleRuleCommand(ctx, cmd, rule)
		}

		// Show help if no valid command
		return showAlertConfigHelp()
	}
}

// handleChannelDisable disables the specified channel
func handleChannelDisable(ctx context.Context, cmd *cli.Command, channelType string) error {
	switch channelType {
	case "email":
		return disableEmailChannel(ctx, cmd)
	case "telegram":
		return disableTelegramChannel(ctx, cmd)
	default:
		return fmt.Errorf("unknown channel type: %s (must be 'email' or 'telegram')", channelType)
	}
}

// handleChannelConfigure configures the specified channel
func handleChannelConfigure(ctx context.Context, cmd *cli.Command, channelType string) error {
	switch channelType {
	case "email":
		return configureEmailChannel(ctx, cmd)
	case "telegram":
		return configureTelegramChannel(ctx, cmd)
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
func handleAlertStatus(ctx context.Context, cmd *cli.Command, args []string) error {
	// Check monitoring namespace
	_, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check alert namespace: %w", err)
	}

	// Get AlertManager configuration
	alertManagerConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		alertManagerConfig = ""
	}

	fmt.Println("📊 Alert Status Summary:")
	fmt.Println("========================")
	fmt.Printf("   📧 Email channel: %s\n", getEmailChannelStatus(alertManagerConfig))
	fmt.Printf("   📱 Telegram channel: %s\n", getTelegramChannelStatus(alertManagerConfig))

	// Display configuration details
	if alertManagerConfig != "" {
		fmt.Println()
		fmt.Println("📋 Channel Configuration Details:")
		fmt.Println("================================")

		// Email configuration
		emailConfig := getEmailConfiguration(alertManagerConfig)
		if emailConfig.Enabled {
			fmt.Printf("   📧 Email Configuration:\n")
			fmt.Printf("      SMTP URL: %s\n", emailConfig.SMTPURL)
			fmt.Printf("      From: %s\n", emailConfig.From)
			fmt.Printf("      To: %s\n", emailConfig.To)
		}

		// Telegram configuration
		telegramConfig := getTelegramConfiguration(alertManagerConfig)
		if telegramConfig.Enabled {
			fmt.Printf("   📱 Telegram Configuration:\n")
			fmt.Printf("      Bot Token: %s\n", telegramConfig.BotToken)
			fmt.Printf("      Chat ID: %s\n", telegramConfig.ChatID)
		}
	}

	// Display detailed rule status
	fmt.Println()
	fmt.Println("📊 Alert Rules Status:")
	fmt.Println("======================")

	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		fmt.Println("⚠️  Could not get current PrometheusRules")
		return nil
	}

	// Parse the YAML
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		fmt.Println("⚠️  Could not parse PrometheusRule YAML")
		return nil
	}

	// Navigate to rules - process all PrometheusRule items
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		fmt.Println("⚠️  No PrometheusRule items found")
		return nil
	}

	// Collect all rules from all PrometheusRule items
	var allRules []interface{}
	for _, item := range items {
		ruleItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		spec, ok := ruleItem["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		groups, ok := spec["groups"].([]interface{})
		if !ok || len(groups) == 0 {
			continue
		}

		for _, group := range groups {
			groupMap, ok := group.(map[string]interface{})
			if !ok {
				continue
			}

			rules, ok := groupMap["rules"].([]interface{})
			if !ok {
				continue
			}

			allRules = append(allRules, rules...)
		}
	}

	if len(allRules) == 0 {
		fmt.Println("⚠️  No rules found in any PrometheusRule")
		return nil
	}

	// Count active rules
	activeRuleCount := 0
	for _, rule := range allRules {
		if ruleMap, ok := rule.(map[string]interface{}); ok {
			if _, exists := ruleMap["alert"]; exists {
				activeRuleCount++
			}
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
				ruleMap, ok := rule.(map[string]interface{})
				if !ok {
					continue
				}

				if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
					found = true
					// Extract current value from expression
					if expr, ok := ruleMap["expr"].(string); ok {
						currentValue = extractValueFromExpression(ruleName, expr)
					}
					// Extract severity
					if labels, ok := ruleMap["labels"].(map[string]interface{}); ok {
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
func handleRuleCommand(ctx context.Context, cmd *cli.Command, ruleAction string) error {
	switch ruleAction {
	case "reset":
		return resetAlertRules(ctx, cmd)
	case "set":
		return configureAlertRules(ctx, cmd)
	default:
		return fmt.Errorf("unknown rule action: %s (must be 'reset' or 'set')", ruleAction)
	}
}

// configureAlertRules allows users to configure alert rules interactively
func configureAlertRules(ctx context.Context, cmd *cli.Command) error {
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
		return enableRule(ctx, ruleName)
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
		return disableRule(ctx, ruleName)
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
		return configureBalanceThreshold(ctx, ruleName)
	case "BlockProductionStalled":
		return configureBlockProductionStall(ctx)
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh":
		return configureUsageThreshold(ctx, ruleName)
	case "PodCrashLooping":
		return configurePodCrashLoop(ctx)
	default:
		fmt.Printf("❌ Unknown rule: %s\n", ruleName)
		return nil
	}
}

// configureBalanceThreshold configures balance threshold for batcher/proposer
func configureBalanceThreshold(ctx context.Context, ruleName string) error {
	fmt.Printf("💰 Configuring %s balance threshold\n", ruleName)
	fmt.Println("Enter the minimum ETH balance threshold (e.g., 0.01, 0.1, 1.0):")

	fmt.Print("Balance threshold (ETH): ")
	threshold, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read balance threshold: %w", err)
	}

	// Validate input
	if threshold == "" {
		return fmt.Errorf("balance threshold cannot be empty")
	}

	fmt.Printf("🔧 Configuring %s with balance threshold '%s' ETH...\n", ruleName, threshold)
	if err := updatePrometheusRule(ctx, ruleName, threshold); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ %s configured successfully with threshold %s ETH\n", ruleName, threshold)
	return nil
}

// configureBlockProductionStall configures block production stall detection
func configureBlockProductionStall(ctx context.Context) error {
	fmt.Println("⏱️  Configuring BlockProductionStalled detection time")
	fmt.Println("Enter the time duration for block production stall detection:")
	fmt.Println("Examples: 30s, 1m, 2m, 5m")

	fmt.Print("Stall detection time: ")
	stallTime, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read stall time: %w", err)
	}

	// Validate input
	if stallTime == "" {
		return fmt.Errorf("stall detection time cannot be empty")
	}

	fmt.Printf("🔧 Configuring BlockProductionStalled with detection time '%s'...\n", stallTime)
	if err := updatePrometheusRule(ctx, "BlockProductionStalled", stallTime); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ BlockProductionStalled configured successfully with detection time %s\n", stallTime)
	return nil
}

// configureUsageThreshold configures CPU/Memory usage threshold
func configureUsageThreshold(ctx context.Context, ruleName string) error {
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
	if threshold == "" {
		return fmt.Errorf("usage threshold cannot be empty")
	}

	fmt.Printf("🔧 Configuring %s with usage threshold '%s%%'...\n", ruleName, threshold)
	if err := updatePrometheusRule(ctx, ruleName, threshold); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ %s configured successfully with threshold %s%%\n", ruleName, threshold)
	return nil
}

// configurePodCrashLoop configures pod crash loop detection time
func configurePodCrashLoop(ctx context.Context) error {
	fmt.Println("🔄 Configuring PodCrashLooping detection time")
	fmt.Println("Enter the time duration for pod crash loop detection:")
	fmt.Println("Examples: 1m, 2m, 5m, 10m")

	fmt.Print("Crash loop detection time: ")
	detectionTime, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read detection time: %w", err)
	}

	// Validate input
	if detectionTime == "" {
		return fmt.Errorf("detection time cannot be empty")
	}

	fmt.Printf("🔧 Configuring PodCrashLooping with detection time '%s'...\n", detectionTime)
	if err := updatePrometheusRule(ctx, "PodCrashLooping", detectionTime); err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	fmt.Printf("✅ PodCrashLooping configured successfully with detection time %s\n", detectionTime)
	return nil
}

func getAlertManagerConfig(ctx context.Context) (string, error) {
	// Get AlertManager Pod to find the actual secret being used
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return "", fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return "", fmt.Errorf("could not find AlertManager config secret")
	}

	// Get the compressed config from the actual secret
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", "monitoring", secretName, "-o", "jsonpath={.data.alertmanager\\.yaml\\.gz}")
	if err != nil {
		return "", fmt.Errorf("failed to get AlertManager config from secret %s: %w", secretName, err)
	}

	// Remove single quotes and spaces
	output = strings.Trim(output, "' \n\t\r")
	decodedBytes, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return "", fmt.Errorf("failed to decode AlertManager config: %w", err)
	}

	// Decompress using gzip
	reader := bytes.NewReader(decodedBytes)
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	configBytes, err := io.ReadAll(gzReader)
	if err != nil {
		return "", fmt.Errorf("failed to read decompressed config: %w", err)
	}

	return string(configBytes), nil
}

func getEmailChannelStatus(config string) string {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return "Unknown"
	}

	receivers, ok := amConfig["receivers"].([]interface{})
	if !ok {
		return "Disabled"
	}

	for _, r := range receivers {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := receiver["email_configs"]; exists {
			return "Enabled"
		}
	}
	return "Disabled"
}

func getTelegramChannelStatus(config string) string {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return "Unknown"
	}

	receivers, ok := amConfig["receivers"].([]interface{})
	if !ok {
		return "Disabled"
	}

	for _, r := range receivers {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if _, exists := receiver["telegram_configs"]; exists {
			return "Enabled"
		}
	}
	return "Disabled"
}

// Configuration update functions
func updateAlertManagerEmailConfig(ctx context.Context, smtpServer, smtpFrom, smtpUsername, smtpPassword string, receivers []string) error {
	// Get current configuration to preserve existing settings
	currentConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse current YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse current AlertManager config: %w", err)
	}

	// Update global SMTP settings
	global, ok := config["global"].(map[string]interface{})
	if !ok {
		global = make(map[string]interface{})
		config["global"] = global
	}
	global["smtp_smarthost"] = smtpServer
	global["smtp_from"] = smtpFrom
	global["smtp_auth_username"] = smtpUsername
	global["smtp_auth_password"] = smtpPassword

	// Find or create the main receiver
	receiversList, ok := config["receivers"].([]interface{})
	if !ok {
		receiversList = []interface{}{}
	}

	// Find the telegram-critical receiver or create it
	var mainReceiver map[string]interface{}
	for _, r := range receiversList {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if name, exists := receiver["name"]; exists && name == "telegram-critical" {
			mainReceiver = receiver
			break
		}
	}

	if mainReceiver == nil {
		mainReceiver = map[string]interface{}{
			"name": "telegram-critical",
		}
		receiversList = append(receiversList, mainReceiver)
	}

	// Add email_configs to the receiver
	emailConfigs := []interface{}{}
	for _, receiver := range receivers {
		emailConfig := map[string]interface{}{
			"to": receiver,
			"headers": map[string]interface{}{
				"subject": "🚨 Critical Alert - {{ .GroupLabels.chain_name }}",
			},
			"html": `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .alert { border-left: 4px solid #dc3545; padding: 10px; margin: 10px 0; background-color: #f8f9fa; }
        .header { color: #dc3545; font-weight: bold; margin-bottom: 15px; }
        .info { margin: 5px 0; }
        .timestamp { color: #6c757d; font-size: 12px; margin-top: 10px; }
        .dashboard { margin-top: 15px; }
        a { color: #007bff; text-decoration: none; }
        a:hover { text-decoration: underline; }
    </style>
</head>
<body>
    <div class="alert">
        <div class="header">🚨 Critical Alert - {{ .GroupLabels.chain_name }}</div>
        <div class="info"><strong>Alert Name:</strong> {{ .GroupLabels.alertname }}</div>
        <div class="info"><strong>Severity:</strong> {{ .GroupLabels.severity }}</div>
        <div class="info"><strong>Component:</strong> {{ .GroupLabels.component }}</div>
        <div class="info" style="margin-top: 15px;"><strong>Summary:</strong></div>
        <div class="info">{{ .CommonAnnotations.summary }}</div>
        <div class="info" style="margin-top: 10px;"><strong>Description:</strong></div>
        <div class="info">{{ .CommonAnnotations.description }}</div>
        <div class="timestamp">⏰ Alert Time: {{ range .Alerts }}{{ .StartsAt }}{{ end }}</div>
        <div class="dashboard">
            <strong>Dashboard:</strong> <a href="http://k8s-thanosmonitoring-b253b70d4a-1790924322.ap-northeast-2.elb.amazonaws.com/d/thanos-stack-app-v9/thanos-stack-application-monitoring-dashboard?orgId=1&refresh=30s">View Details</a>
        </div>
    </div>
</body>
</html>`,
		}
		emailConfigs = append(emailConfigs, emailConfig)
	}

	mainReceiver["email_configs"] = emailConfigs
	config["receivers"] = receiversList

	// Convert back to YAML
	newConfigBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Get the actual secret name that AlertManager Pod uses
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return fmt.Errorf("could not find AlertManager config secret")
	}

	fmt.Printf("🔧 Updating secret: %s\n", secretName)

	// Compress the new configuration
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(newConfigBytes); err != nil {
		return fmt.Errorf("failed to compress config: %w", err)
	}
	gzWriter.Close()

	// Base64 encode the compressed config
	compressedConfig := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Patch the secret with the new compressed configuration
	patchData := fmt.Sprintf(`{"data":{"alertmanager.yaml.gz":"%s"}}`, compressedConfig)
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", secretName, "-n", "monitoring", "--type=merge", "-p", patchData); err != nil {
		return fmt.Errorf("failed to patch AlertManager secret: %w", err)
	}

	// AlertManager should automatically reload configuration when secret is updated
	fmt.Println("✅ AlertManager configuration updated successfully")
	fmt.Println("💡 AlertManager will automatically reload the configuration")

	// Optional: Wait a moment for configuration to be applied
	fmt.Println("⏳ Waiting for configuration to be applied...")
	time.Sleep(5 * time.Second)

	return nil
}

// Configuration removal functions
func removeEmailConfigFromAlertManager(ctx context.Context) error {
	// Get current configuration from the actual AlertManager secret
	currentConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse YAML to remove email_configs
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse AlertManager config: %w", err)
	}

	// Find and remove email_configs from receivers
	receivers, ok := config["receivers"].([]interface{})
	if !ok {
		return fmt.Errorf("receivers not found in config")
	}

	for i, receiver := range receivers {
		receiverMap, ok := receiver.(map[string]interface{})
		if !ok {
			continue
		}

		// Remove email_configs from this receiver
		if _, exists := receiverMap["email_configs"]; exists {
			delete(receiverMap, "email_configs")
			receivers[i] = receiverMap
			fmt.Println("✅ Removed email_configs from receiver")
		}
	}

	// Convert back to YAML
	newConfigBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Get the actual secret name that AlertManager Pod uses
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return fmt.Errorf("could not find AlertManager config secret")
	}

	fmt.Printf("🔧 Updating secret: %s\n", secretName)

	// Compress the new configuration
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(newConfigBytes); err != nil {
		return fmt.Errorf("failed to compress config: %w", err)
	}
	gzWriter.Close()

	// Base64 encode the compressed config
	compressedConfig := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Patch the secret with the new compressed configuration
	patchData := fmt.Sprintf(`{"data":{"alertmanager.yaml.gz":"%s"}}`, compressedConfig)
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", secretName, "-n", "monitoring", "--type=merge", "-p", patchData); err != nil {
		return fmt.Errorf("failed to patch AlertManager secret: %w", err)
	}

	// AlertManager should automatically reload configuration when secret is updated
	fmt.Println("✅ AlertManager configuration updated successfully")
	fmt.Println("💡 AlertManager will automatically reload the configuration")

	// Optional: Wait a moment for configuration to be applied
	fmt.Println("⏳ Waiting for configuration to be applied...")
	time.Sleep(5 * time.Second)

	return nil
}

func removeTelegramConfigFromAlertManager(ctx context.Context) error {
	// Get current configuration from the actual AlertManager secret
	currentConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse YAML to remove telegram_configs
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse AlertManager config: %w", err)
	}

	// Find and remove telegram_configs from receivers
	receivers, ok := config["receivers"].([]interface{})
	if !ok {
		return fmt.Errorf("receivers not found in config")
	}

	for i, receiver := range receivers {
		receiverMap, ok := receiver.(map[string]interface{})
		if !ok {
			continue
		}

		// Remove telegram_configs from this receiver
		if _, exists := receiverMap["telegram_configs"]; exists {
			delete(receiverMap, "telegram_configs")
			receivers[i] = receiverMap
			fmt.Println("✅ Removed telegram_configs from receiver")
		}
	}

	// Convert back to YAML
	newConfigBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Get the actual secret name that AlertManager Pod uses
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return fmt.Errorf("could not find AlertManager config secret")
	}

	fmt.Printf("🔧 Updating secret: %s\n", secretName)

	// Compress the new configuration
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(newConfigBytes); err != nil {
		return fmt.Errorf("failed to compress config: %w", err)
	}
	gzWriter.Close()

	// Base64 encode the compressed config
	compressedConfig := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Patch the secret with the new compressed configuration
	patchData := fmt.Sprintf(`{"data":{"alertmanager.yaml.gz":"%s"}}`, compressedConfig)
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", secretName, "-n", "monitoring", "--type=merge", "-p", patchData); err != nil {
		return fmt.Errorf("failed to patch AlertManager secret: %w", err)
	}

	// AlertManager should automatically reload configuration when secret is updated
	fmt.Println("✅ AlertManager configuration updated successfully")
	fmt.Println("💡 AlertManager will automatically reload the configuration")

	// Optional: Wait a moment for configuration to be applied
	fmt.Println("⏳ Waiting for configuration to be applied...")
	time.Sleep(5 * time.Second)

	return nil
}

func updateAlertManagerTelegramConfig(ctx context.Context, botToken, chatID string) error {
	// Get current configuration to preserve existing settings
	currentConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse current YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse current AlertManager config: %w", err)
	}

	// Find or create the main receiver
	receiversList, ok := config["receivers"].([]interface{})
	if !ok {
		receiversList = []interface{}{}
	}

	// Find the telegram-critical receiver or create it
	var mainReceiver map[string]interface{}
	for _, r := range receiversList {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}
		if name, exists := receiver["name"]; exists && name == "telegram-critical" {
			mainReceiver = receiver
			break
		}
	}

	if mainReceiver == nil {
		mainReceiver = map[string]interface{}{
			"name": "telegram-critical",
		}
		receiversList = append(receiversList, mainReceiver)
	}

	// Add telegram_configs to the receiver
	// Convert chat_id to int64 for Prometheus Operator compatibility
	chatIdInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat_id format: %s", chatID)
	}

	telegramConfigs := []interface{}{
		map[string]interface{}{
			"bot_token":  botToken,
			"chat_id":    chatIdInt,
			"parse_mode": "Markdown",
			"message": `🚨 *Critical Alert - {{ .GroupLabels.chain_name }}*

**Alert Name:** {{ .GroupLabels.alertname }}
**Severity:** {{ .GroupLabels.severity }}
**Component:** {{ .GroupLabels.component }}

**Summary:** {{ .CommonAnnotations.summary }}
**Description:** {{ .CommonAnnotations.description }}

⏰ Alert Time: {{ range .Alerts }}{{ .StartsAt }}{{ end }}`,
		},
	}

	mainReceiver["telegram_configs"] = telegramConfigs
	config["receivers"] = receiversList

	// Convert back to YAML
	newConfigBytes, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Get the actual secret name that AlertManager Pod uses
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return fmt.Errorf("could not find AlertManager config secret")
	}

	fmt.Printf("🔧 Updating secret: %s\n", secretName)

	// Compress the new configuration
	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	if _, err := gzWriter.Write(newConfigBytes); err != nil {
		return fmt.Errorf("failed to compress config: %w", err)
	}
	gzWriter.Close()

	// Base64 encode the compressed config
	compressedConfig := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Patch the secret with the new compressed configuration
	patchData := fmt.Sprintf(`{"data":{"alertmanager.yaml.gz":"%s"}}`, compressedConfig)
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "secret", secretName, "-n", "monitoring", "--type=merge", "-p", patchData); err != nil {
		return fmt.Errorf("failed to patch AlertManager secret: %w", err)
	}

	// AlertManager should automatically reload configuration when secret is updated
	fmt.Println("✅ AlertManager configuration updated successfully")
	fmt.Println("💡 AlertManager will automatically reload the configuration")

	// Optional: Wait a moment for configuration to be applied
	fmt.Println("⏳ Waiting for configuration to be applied...")
	time.Sleep(5 * time.Second)

	return nil
}

func updatePrometheusRule(ctx context.Context, ruleName, newValue string) error {
	// Validate rule name (only configurable rules)
	validRules := map[string]string{
		"OpBatcherBalanceCritical":  "OP Batcher balance threshold",
		"OpProposerBalanceCritical": "OP Proposer balance threshold",
		"BlockProductionStalled":    "Block production stall detection",
		"ContainerCpuUsageHigh":     "Container CPU usage threshold",
		"ContainerMemoryUsageHigh":  "Container memory usage threshold",
		"PodCrashLooping":           "Pod crash loop detection",
	}

	if _, valid := validRules[ruleName]; !valid {
		return fmt.Errorf("unknown rule: %s (valid rules: %v)", ruleName, getKeys(validRules))
	}

	fmt.Printf("🔧 Updating rule '%s' to value '%s'\n", ruleName, newValue)
	fmt.Printf("📋 Rule description: %s\n", validRules[ruleName])

	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	// Find the first PrometheusRule item
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	ruleItem, ok := items[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse PrometheusRule item")
	}

	// Navigate to the rules section
	spec, ok := ruleItem["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find spec in PrometheusRule")
	}

	groups, ok := spec["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	group, ok := groups[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse group")
	}

	rules, ok := group["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("no rules found in group")
	}

	// Find and update the specific rule
	ruleFound := false
	for i, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
			// Update the rule based on its type
			if err := updateRuleExpression(ruleMap, ruleName, newValue); err != nil {
				return fmt.Errorf("failed to update rule expression: %w", err)
			}

			// Update annotations if needed
			if err := updateRuleAnnotations(ruleMap, ruleName, newValue); err != nil {
				return fmt.Errorf("failed to update rule annotations: %w", err)
			}

			ruleFound = true
			fmt.Printf("✅ Found and updated rule '%s' at index %d\n", ruleName, i)
			break
		}
	}

	if !ruleFound {
		return fmt.Errorf("rule '%s' not found in PrometheusRule", ruleName)
	}

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(ruleList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write updated YAML to temporary file
	tempFile := fmt.Sprintf("/tmp/prometheusrule-updated-%d.yaml", time.Now().Unix())
	if err := os.WriteFile(tempFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	// Apply the updated PrometheusRule
	fmt.Println("📝 Applying updated PrometheusRule...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	fmt.Printf("✅ Rule '%s' successfully updated to value '%s'\n", ruleName, newValue)
	return nil
}

// updateRuleExpression updates the expression of a specific rule
func updateRuleExpression(ruleMap map[string]interface{}, ruleName, newValue string) error {
	_, ok := ruleMap["expr"].(string)
	if !ok {
		return fmt.Errorf("failed to get expression for rule %s", ruleName)
	}

	// Update expression based on rule type
	switch ruleName {
	case "OpBatcherBalanceCritical":
		// Update the threshold in the expression
		ruleMap["expr"] = fmt.Sprintf("op_batcher_default_balance < %s", newValue)
	case "OpProposerBalanceCritical":
		// Update the threshold in the expression
		ruleMap["expr"] = fmt.Sprintf("op_proposer_default_balance < %s", newValue)
	case "BlockProductionStalled":
		// Update the time duration in the expression
		ruleMap["expr"] = fmt.Sprintf("increase(chain_head_block[%s]) == 0", newValue)
	case "ContainerCpuUsageHigh":
		// Update the CPU usage threshold
		ruleMap["expr"] = fmt.Sprintf("(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s", newValue)
	case "ContainerMemoryUsageHigh":
		// Update the memory usage threshold
		ruleMap["expr"] = fmt.Sprintf("(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s", newValue)
	case "PodCrashLooping":
		// Update the restart detection time
		ruleMap["expr"] = fmt.Sprintf("rate(kube_pod_container_status_restarts_total[%s]) > 0", newValue)
	default:
		return fmt.Errorf("unsupported rule type: %s", ruleName)
	}

	return nil
}

// updateRuleAnnotations updates the annotations of a specific rule
func updateRuleAnnotations(ruleMap map[string]interface{}, ruleName, newValue string) error {
	annotations, ok := ruleMap["annotations"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to get annotations for rule %s", ruleName)
	}

	// Update description based on rule type
	switch ruleName {
	case "OpBatcherBalanceCritical":
		annotations["description"] = fmt.Sprintf("OP Batcher balance is {{ $value }} ETH, below %s ETH threshold", newValue)
	case "OpProposerBalanceCritical":
		annotations["description"] = fmt.Sprintf("OP Proposer balance is {{ $value }} ETH, below %s ETH threshold", newValue)
	case "BlockProductionStalled":
		annotations["description"] = fmt.Sprintf("No new blocks have been produced for more than %s", newValue)
	case "ContainerCpuUsageHigh":
		annotations["description"] = fmt.Sprintf("Pod {{ $labels.pod }} CPU usage has been above %s%% for more than 2 minutes", newValue)
	case "ContainerMemoryUsageHigh":
		annotations["description"] = fmt.Sprintf("Pod {{ $labels.pod }} memory usage has been above %s%% for more than 2 minutes", newValue)
	case "PodCrashLooping":
		annotations["description"] = fmt.Sprintf("Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently for more than %s", newValue)
	}

	return nil
}

// enableRule enables a specific rule by adding it back to PrometheusRule
func enableRule(ctx context.Context, ruleName string) error {
	fmt.Printf("🟢 Enabling rule '%s'...\n", ruleName)

	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	// Navigate to rules
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	ruleItem, ok := items[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse PrometheusRule item")
	}

	spec, ok := ruleItem["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find spec in PrometheusRule")
	}

	groups, ok := spec["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	group, ok := groups[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse group")
	}

	rules, ok := group["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("no rules found in group")
	}

	// Check if rule already exists
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
			fmt.Printf("ℹ️  Rule '%s' is already enabled\n", ruleName)
			return nil
		}
	}

	// Default values for rules
	defaultValues := map[string]string{
		"OpBatcherBalanceCritical":  "0.01",
		"OpProposerBalanceCritical": "0.01",
		"BlockProductionStalled":    "1m",
		"ContainerCpuUsageHigh":     "80",
		"ContainerMemoryUsageHigh":  "80",
		"PodCrashLooping":           "2m",
	}

	// Create new rule with default value
	defaultValue := defaultValues[ruleName]
	newRule := createRuleWithDefaultValue(ruleName, defaultValue)

	// Add the rule to the rules list
	rules = append(rules, newRule)
	group["rules"] = rules

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(ruleList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write updated YAML to temporary file
	tempFile := fmt.Sprintf("/tmp/prometheusrule-enable-%d.yaml", time.Now().Unix())
	if err := os.WriteFile(tempFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	// Apply the updated PrometheusRule
	fmt.Println("📝 Applying updated PrometheusRule...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	fmt.Printf("✅ Rule '%s' enabled successfully with default value '%s'\n", ruleName, defaultValue)
	return nil
}

// disableRule disables a specific rule by removing it from PrometheusRule
func disableRule(ctx context.Context, ruleName string) error {
	fmt.Printf("🔴 Disabling rule '%s'...\n", ruleName)

	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	// Navigate to rules
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	ruleItem, ok := items[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse PrometheusRule item")
	}

	spec, ok := ruleItem["spec"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to find spec in PrometheusRule")
	}

	groups, ok := spec["groups"].([]interface{})
	if !ok || len(groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	group, ok := groups[0].(map[string]interface{})
	if !ok {
		return fmt.Errorf("failed to parse group")
	}

	rules, ok := group["rules"].([]interface{})
	if !ok {
		return fmt.Errorf("no rules found in group")
	}

	// Find and remove the rule
	ruleFound := false
	var updatedRules []interface{}

	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			updatedRules = append(updatedRules, rule)
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
			ruleFound = true
			fmt.Printf("✅ Found and removing rule '%s'\n", ruleName)
		} else {
			updatedRules = append(updatedRules, rule)
		}
	}

	if !ruleFound {
		fmt.Printf("ℹ️  Rule '%s' is already disabled or not found\n", ruleName)
		return nil
	}

	// Update the rules list
	group["rules"] = updatedRules

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(ruleList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write updated YAML to temporary file
	tempFile := fmt.Sprintf("/tmp/prometheusrule-disable-%d.yaml", time.Now().Unix())
	if err := os.WriteFile(tempFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	// Apply the updated PrometheusRule
	fmt.Println("📝 Applying updated PrometheusRule...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	fmt.Printf("✅ Rule '%s' disabled successfully\n", ruleName)
	return nil
}

// extractValueFromExpression extracts the current value from a rule expression
func extractValueFromExpression(ruleName, expr string) string {
	switch ruleName {
	case "OpBatcherBalanceCritical", "OpProposerBalanceCritical":
		// Extract value from "op_batcher_default_balance < 0.01" or "op_proposer_default_balance < 0.01"
		if strings.Contains(expr, "<") {
			parts := strings.Split(expr, "<")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case "BlockProductionStalled":
		// Extract time from "increase(chain_head_block[1m]) == 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[") + 1
			end := strings.Index(expr, "]")
			if start > 0 && end > start {
				return expr[start:end]
			}
		}
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh":
		// Extract percentage from complex expression like "(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > 80"
		if strings.Contains(expr, ">") {
			// Find the last ">" which should be the threshold comparison
			lastGreaterIndex := strings.LastIndex(expr, ">")
			if lastGreaterIndex != -1 {
				thresholdPart := strings.TrimSpace(expr[lastGreaterIndex+1:])
				// Remove any trailing parts that might be after the number
				if strings.Contains(thresholdPart, " ") {
					thresholdPart = strings.Split(thresholdPart, " ")[0]
				}
				return thresholdPart
			}
		}
	case "PodCrashLooping":
		// Extract time from "rate(kube_pod_container_status_restarts_total[2m]) > 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[") + 1
			end := strings.Index(expr, "]")
			if start > 0 && end > start {
				return expr[start:end]
			}
		}
	}
	return ""
}

// createRuleWithDefaultValue creates a new rule with default value
func createRuleWithDefaultValue(ruleName, defaultValue string) map[string]interface{} {
	rule := map[string]interface{}{
		"alert": ruleName,
		"for":   "10s",
		"labels": map[string]interface{}{
			"chain_name": "theo0715",
			"component":  getComponentForRule(ruleName),
			"namespace":  "monitoring",
			"severity":   "critical",
		},
	}

	// Set expression and annotations based on rule type
	switch ruleName {
	case "OpBatcherBalanceCritical":
		rule["expr"] = fmt.Sprintf("op_batcher_default_balance < %s", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("OP Batcher balance is {{ $value }} ETH, below %s ETH threshold", defaultValue),
			"summary":     "OP Batcher ETH balance critically low",
		}
	case "OpProposerBalanceCritical":
		rule["expr"] = fmt.Sprintf("op_proposer_default_balance < %s", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("OP Proposer balance is {{ $value }} ETH, below %s ETH threshold", defaultValue),
			"summary":     "OP Proposer ETH balance critically low",
		}
	case "BlockProductionStalled":
		rule["expr"] = fmt.Sprintf("increase(chain_head_block[%s]) == 0", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("No new blocks have been produced for more than %s", defaultValue),
			"summary":     "Block production has stalled",
		}
		rule["for"] = "1m"
	case "ContainerCpuUsageHigh":
		rule["expr"] = fmt.Sprintf("(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("Pod {{ $labels.pod }} CPU usage has been above %s%% for more than 2 minutes", defaultValue),
			"summary":     "High CPU usage in Thanos Stack pod",
		}
		rule["for"] = "2m"
	case "ContainerMemoryUsageHigh":
		rule["expr"] = fmt.Sprintf("(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("Pod {{ $labels.pod }} memory usage has been above %s%% for more than 2 minutes", defaultValue),
			"summary":     "High memory usage in Thanos Stack pod",
		}
		rule["for"] = "2m"
	case "PodCrashLooping":
		rule["expr"] = fmt.Sprintf("rate(kube_pod_container_status_restarts_total[%s]) > 0", defaultValue)
		rule["annotations"] = map[string]interface{}{
			"description": fmt.Sprintf("Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently for more than %s", defaultValue),
			"summary":     "Pod is crash looping",
		}
		rule["for"] = "2m"
	}

	return rule
}

// getComponentForRule returns the component name for a rule
func getComponentForRule(ruleName string) string {
	switch ruleName {
	case "OpBatcherBalanceCritical", "OpProposerBalanceCritical":
		return "op-batcher"
	case "BlockProductionStalled":
		return "op-geth"
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh", "PodCrashLooping":
		return "kubernetes"
	default:
		return "unknown"
	}
}

// getKeys returns the keys of a map as a slice
func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func resetPrometheusRules(ctx context.Context) error {
	fmt.Println("🔧 Resetting PrometheusRules to default configuration...")

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

	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	// Process all PrometheusRule items
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	// Collect all rules from all PrometheusRule items
	var allRules []interface{}
	var allRuleItems []map[string]interface{}

	for _, item := range items {
		ruleItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		spec, ok := ruleItem["spec"].(map[string]interface{})
		if !ok {
			continue
		}

		groups, ok := spec["groups"].([]interface{})
		if !ok || len(groups) == 0 {
			continue
		}

		for _, group := range groups {
			groupMap, ok := group.(map[string]interface{})
			if !ok {
				continue
			}

			rules, ok := groupMap["rules"].([]interface{})
			if !ok {
				continue
			}

			allRules = append(allRules, rules...)
			// Store the rule item for later update
			allRuleItems = append(allRuleItems, ruleItem)
		}
	}

	if len(allRules) == 0 {
		return fmt.Errorf("no rules found in any PrometheusRule")
	}

	// Default values for configurable rules
	defaultValues := map[string]string{
		"OpBatcherBalanceCritical":  "0.01",
		"OpProposerBalanceCritical": "0.01",
		"BlockProductionStalled":    "1m",
		"ContainerCpuUsageHigh":     "80",
		"ContainerMemoryUsageHigh":  "80",
		"PodCrashLooping":           "2m",
	}

	// Track which configurable rules exist and which are missing
	existingRules := make(map[string]bool)
	missingRules := make([]string, 0)

	// Check which configurable rules exist
	for _, rule := range allRules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists {
			ruleName := alertName.(string)
			if _, isConfigurable := defaultValues[ruleName]; isConfigurable {
				existingRules[ruleName] = true
			}
		}
	}

	// Find missing configurable rules
	for ruleName := range defaultValues {
		if !existingRules[ruleName] {
			missingRules = append(missingRules, ruleName)
		}
	}

	// Reset existing configurable rules to default values
	rulesReset := 0
	for _, rule := range allRules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists {
			ruleName := alertName.(string)
			if defaultValue, shouldReset := defaultValues[ruleName]; shouldReset {
				// Update the rule to default value
				if err := updateRuleExpression(ruleMap, ruleName, defaultValue); err != nil {
					fmt.Printf("⚠️  Failed to reset rule '%s': %v\n", ruleName, err)
					continue
				}

				// Update annotations
				if err := updateRuleAnnotations(ruleMap, ruleName, defaultValue); err != nil {
					fmt.Printf("⚠️  Failed to reset annotations for rule '%s': %v\n", ruleName, err)
					continue
				}

				fmt.Printf("✅ Reset rule '%s' to default value '%s'\n", ruleName, defaultValue)
				rulesReset++
			}
		}
	}

	// Re-enable missing configurable rules
	for _, ruleName := range missingRules {
		defaultValue := defaultValues[ruleName]
		fmt.Printf("🔄 Re-enabling disabled rule '%s' with default value '%s'\n", ruleName, defaultValue)

		// Find the first PrometheusRule to add the rule back
		if len(allRuleItems) > 0 {
			ruleItem := allRuleItems[0]
			spec, ok := ruleItem["spec"].(map[string]interface{})
			if ok {
				groups, ok := spec["groups"].([]interface{})
				if ok && len(groups) > 0 {
					group, ok := groups[0].(map[string]interface{})
					if ok {
						rules, ok := group["rules"].([]interface{})
						if ok {
							// Create new rule with default value
							newRule := createRuleWithDefaultValue(ruleName, defaultValue)
							rules = append(rules, newRule)
							group["rules"] = rules
							rulesReset++
							fmt.Printf("✅ Re-enabled rule '%s' with default value '%s'\n", ruleName, defaultValue)
						}
					}
				}
			}
		}
	}

	if rulesReset == 0 {
		fmt.Println("ℹ️  No configurable rules found to reset")
		return nil
	}

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(ruleList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write updated YAML to temporary file
	tempFile := fmt.Sprintf("/tmp/prometheusrule-reset-%d.yaml", time.Now().Unix())
	if err := os.WriteFile(tempFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	// Apply the updated PrometheusRule
	fmt.Println("📝 Applying reset PrometheusRule...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply reset PrometheusRule: %w", err)
	}

	fmt.Printf("✅ Successfully reset %d rules to default values\n", rulesReset)
	fmt.Println()
	fmt.Println("📋 Default values applied:")
	fmt.Println("   - OpBatcherBalanceCritical: 0.01 ETH threshold")
	fmt.Println("   - OpProposerBalanceCritical: 0.01 ETH threshold")
	fmt.Println("   - BlockProductionStalled: 1m stall detection")
	fmt.Println("   - ContainerCpuUsageHigh: 80% threshold")
	fmt.Println("   - ContainerMemoryUsageHigh: 80% threshold")
	fmt.Println("   - PodCrashLooping: 2m restart detection")
	fmt.Println()
	fmt.Println("⚠️  Note: Core system alerts (OpNodeDown, OpBatcherDown, OpProposerDown, OpGethDown, L1RpcDown)")
	fmt.Println("    remain unchanged to ensure system stability.")

	return nil
}

// Email channel management functions
func disableEmailChannel(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📧 Disabling Email Channel...")

	// Get current AlertManager configuration
	config, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if email is already disabled
	if getEmailChannelStatus(config) == "Disabled" {
		fmt.Println("ℹ️  Email channel is already disabled")
		return nil
	}

	// Remove email configuration from AlertManager
	fmt.Println("🔧 Removing email configuration from AlertManager...")
	if err := removeEmailConfigFromAlertManager(ctx); err != nil {
		return fmt.Errorf("failed to disable email channel: %w", err)
	}

	fmt.Println("✅ Email channel disabled successfully")
	return nil
}

func configureEmailChannel(ctx context.Context, cmd *cli.Command) error {
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
	if err := updateAlertManagerEmailConfig(ctx, smtpServer, smtpFrom, smtpUsername, smtpPassword, receivers); err != nil {
		return fmt.Errorf("failed to update AlertManager configuration: %w", err)
	}

	fmt.Println("✅ Email channel configured successfully")
	return nil
}

// Telegram channel management functions
func disableTelegramChannel(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📱 Disabling Telegram Channel...")

	// Get current AlertManager configuration
	config, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if telegram is already disabled
	if getTelegramChannelStatus(config) == "Disabled" {
		fmt.Println("ℹ️  Telegram channel is already disabled")
		return nil
	}

	// Remove telegram configuration from AlertManager
	fmt.Println("🔧 Removing telegram configuration from AlertManager...")
	if err := removeTelegramConfigFromAlertManager(ctx); err != nil {
		return fmt.Errorf("failed to disable telegram channel: %w", err)
	}

	fmt.Println("✅ Telegram channel disabled successfully")
	return nil
}

func configureTelegramChannel(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📱 Configuring Telegram Channel...")

	// Get new telegram configuration from user
	fmt.Println("Enter telegram configuration:")

	fmt.Print("Bot API Token: ")
	botToken, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read bot token: %w", err)
	}

	// Validate bot token format (basic check)
	if !strings.Contains(botToken, ":") {
		fmt.Println("❌ Invalid bot token format. Expected format: 123456789:ABCdefGHIjklMNOpqrsTUVwxyz")
		return fmt.Errorf("invalid bot token format")
	}

	fmt.Print("Chat ID: ")
	chatID, err := scanner.ScanString()
	if err != nil {
		return fmt.Errorf("failed to read chat ID: %w", err)
	}

	// Validate chat ID format (basic check)
	if !strings.HasPrefix(chatID, "-") && !strings.HasPrefix(chatID, "1") {
		fmt.Println("❌ Invalid chat ID format. Expected format: -123456789 or 123456789")
		return fmt.Errorf("invalid chat ID format")
	}

	fmt.Printf("📱 Telegram Configuration Summary:\n")
	fmt.Printf("   Bot Token: %s...\n", botToken[:min(len(botToken), 10)])
	fmt.Printf("   Chat ID: %s\n", chatID)

	// Apply configuration to AlertManager
	fmt.Println("🔧 Applying telegram configuration to AlertManager...")
	if err := updateAlertManagerTelegramConfig(ctx, botToken, chatID); err != nil {
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

func resetAlertRules(ctx context.Context, cmd *cli.Command) error {
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

		if err := resetPrometheusRules(ctx); err != nil {
			return fmt.Errorf("failed to reset alert rules: %w", err)
		}

		fmt.Println("✅ Alert rules reset to default successfully")
		fmt.Println("💡 Prometheus will reload rules automatically")
	} else {
		fmt.Println("❌ Reset cancelled")
	}

	return nil
}

// EmailConfiguration holds email channel configuration details
type EmailConfiguration struct {
	Enabled bool
	SMTPURL string
	From    string
	To      string
}

// TelegramConfiguration holds telegram channel configuration details
type TelegramConfiguration struct {
	Enabled  bool
	BotToken string
	ChatID   string
}

func getEmailConfiguration(config string) EmailConfiguration {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return EmailConfiguration{Enabled: false}
	}

	// Get global SMTP settings
	global, ok := amConfig["global"].(map[string]interface{})
	if !ok {
		return EmailConfiguration{Enabled: false}
	}

	smtpURL := ""
	if smtpHost, exists := global["smtp_smarthost"]; exists {
		smtpURL = fmt.Sprintf("%v", smtpHost)
	}

	from := ""
	if smtpFrom, exists := global["smtp_from"]; exists {
		from = fmt.Sprintf("%v", smtpFrom)
	}

	// Get receivers
	receivers, ok := amConfig["receivers"].([]interface{})
	if !ok {
		return EmailConfiguration{Enabled: false}
	}

	var toAddresses []string
	for _, r := range receivers {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		if emailConfigs, exists := receiver["email_configs"]; exists {
			if emailConfigList, ok := emailConfigs.([]interface{}); ok {
				for _, emailConfig := range emailConfigList {
					if emailConfig, ok := emailConfig.(map[string]interface{}); ok {
						if toAddr, exists := emailConfig["to"]; exists {
							toAddresses = append(toAddresses, fmt.Sprintf("%v", toAddr))
						}
					}
				}
			}
		}
	}

	if len(toAddresses) > 0 {
		return EmailConfiguration{
			Enabled: true,
			SMTPURL: smtpURL,
			From:    from,
			To:      strings.Join(toAddresses, ", "),
		}
	}

	return EmailConfiguration{Enabled: false}
}

func getTelegramConfiguration(config string) TelegramConfiguration {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return TelegramConfiguration{Enabled: false}
	}

	receivers, ok := amConfig["receivers"].([]interface{})
	if !ok {
		return TelegramConfiguration{Enabled: false}
	}

	var botTokens []string
	var chatIDs []string

	for _, r := range receivers {
		receiver, ok := r.(map[string]interface{})
		if !ok {
			continue
		}

		if telegramConfigs, exists := receiver["telegram_configs"]; exists {
			if telegramConfigList, ok := telegramConfigs.([]interface{}); ok {
				for _, telegramConfig := range telegramConfigList {
					if telegramConfig, ok := telegramConfig.(map[string]interface{}); ok {
						if token, exists := telegramConfig["bot_token"]; exists {
							botTokens = append(botTokens, fmt.Sprintf("%v", token))
						}

						if chat, exists := telegramConfig["chat_id"]; exists {
							chatIDs = append(chatIDs, fmt.Sprintf("%v", chat))
						}
					}
				}
			}
		}
	}

	if len(botTokens) > 0 || len(chatIDs) > 0 {
		return TelegramConfiguration{
			Enabled:  true,
			BotToken: strings.Join(botTokens, ", "),
			ChatID:   strings.Join(chatIDs, ", "),
		}
	}

	return TelegramConfiguration{Enabled: false}
}
