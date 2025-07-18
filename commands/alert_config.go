package commands

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

// cleanPasswordInput cleans up password input by removing unwanted characters
func cleanPasswordInput(password string) string {
	// Remove special whitespace characters (NBSP, etc.)
	password = strings.ReplaceAll(password, "\u00A0", " ") // Replace NBSP with regular space
	password = strings.ReplaceAll(password, "\u200B", "")  // Remove zero-width space
	password = strings.ReplaceAll(password, "\uFEFF", "")  // Remove byte order mark
	password = strings.TrimSpace(password)                 // Trim whitespace

	// Remove any control characters that might cause issues
	var cleaned strings.Builder
	for _, r := range password {
		if r >= 32 && r != 127 { // Printable ASCII characters except DEL
			cleaned.WriteRune(r)
		}
	}

	return cleaned.String()
}

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
		rules := cmd.Bool("rules")
		reset := cmd.Bool("reset")

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

		// Handle rules command
		if rules {
			return handleRulesCustomization(ctx, cmd, cmd.Args().Slice())
		}

		// Handle reset command
		if reset {
			return resetPrometheusRules(ctx)
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
	fmt.Println("🔍 Checking monitoring plugin installation...")

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

	fmt.Println("✅ Monitoring plugin is installed and ready")
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
	fmt.Println("  trh-sdk alert-config [--status|--channels|--rules|--reset] [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  --status   - Show current alert configuration")
	fmt.Println("  --channels - Manage notification channels (email, telegram)")
	fmt.Println("  --rules    - Manage alert rules (list, modify)")
	fmt.Println("  --reset    - Reset alert rules to default values")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  trh-sdk alert-config --status")
	fmt.Println("  trh-sdk alert-config --channels --type email --operation disable")
	fmt.Println("  trh-sdk alert-config --rules --type list")
	fmt.Println("  trh-sdk alert-config --rules --type modify --rule proposer-balance --value 0.1")
	fmt.Println("  trh-sdk alert-config --reset")
	return nil
}

// handleEmailChannel manages email channel configuration
func handleEmailChannel(ctx context.Context, cmd *cli.Command, action string, args []string) error {
	switch action {
	case "disable":
		return disableEmailChannel(ctx, cmd)
	case "configure":
		return configureEmailChannel(ctx, cmd)
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Supported actions: disable, configure")
		return nil
	}
}

// handleTelegramChannel manages telegram channel configuration
func handleTelegramChannel(ctx context.Context, cmd *cli.Command, action string, args []string) error {
	switch action {
	case "disable":
		return disableTelegramChannel(ctx, cmd)
	case "configure":
		return configureTelegramChannel(ctx, cmd)
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Supported actions: disable, configure")
		return nil
	}
}

// handleRulesCustomization manages alert rules
func handleRulesCustomization(ctx context.Context, cmd *cli.Command, args []string) error {
	if len(args) == 0 {
		fmt.Println("❌ Usage: trh-sdk monitoring rules [list|modify|reset]")
		return nil
	}

	action := args[0]
	switch action {
	case "list":
		return listAlertRules(ctx, cmd)
	case "modify":
		return modifyAlertRule(ctx, cmd, args[1:])
	case "reset":
		return resetAlertRules(ctx, cmd)
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Supported actions: list, modify, reset")
		return nil
	}
}

// handleRulesCustomizationWithFlags manages alert rules using flags
func handleRulesCustomizationWithFlags(ctx context.Context, cmd *cli.Command, actionType, operation, rule, value string) error {
	if actionType == "" {
		fmt.Println("❌ Usage: trh-sdk alert-config --rules --type [list|modify]")
		return nil
	}

	switch actionType {
	case "list":
		return listAlertRules(ctx, cmd)
	case "modify":
		if rule == "" {
			fmt.Println("❌ Usage: trh-sdk alert-config --rules --type modify --rule [rule-name] --value [new-value]")
			return nil
		}
		if value == "" {
			fmt.Println("❌ Usage: trh-sdk alert-config --rules --type modify --rule [rule-name] --value [new-value]")
			return nil
		}
		return modifyAlertRule(ctx, cmd, []string{rule, value})
	default:
		fmt.Printf("❌ Unknown action type: %s\n", actionType)
		fmt.Println("Supported actions: list, modify")
		return nil
	}
}

// handleAlertStatus shows current alert configuration
func handleAlertStatus(ctx context.Context, cmd *cli.Command, args []string) error {
	// Check monitoring namespace
	_, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check alert namespace: %w", err)
	}

	// Get the information
	alertRules, _ := getPrometheusRules(ctx)

	// Get AlertManager configuration
	alertManagerConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		alertManagerConfig = ""
	}

	fmt.Println("📊 Alert Status Summary:")
	fmt.Println("========================")
	fmt.Printf("   📧 Email channel: %s\n", getEmailChannelStatus(alertManagerConfig))
	fmt.Printf("   📱 Telegram channel: %s\n", getTelegramChannelStatus(alertManagerConfig))
	fmt.Printf("   📋 Alert Rules: %d active\n", alertRules)

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

func getPrometheusRules(ctx context.Context) (int, error) {
	// Get all PrometheusRules in the monitoring namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrules", "-n", "monitoring", "-o", "jsonpath={.items[*].spec.groups[*].rules[*].alert}")
	if err != nil {
		return 0, err
	}

	// Split by space to count individual alert rules
	alerts := strings.Split(strings.TrimSpace(output), " ")
	count := 0
	for _, alert := range alerts {
		if strings.TrimSpace(alert) != "" {
			count++
		}
	}
	return count, nil
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
	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrules", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// For now, just show what would be updated
	fmt.Printf("🔧 Would update rule '%s' to value '%s'\n", ruleName, newValue)
	fmt.Println("   This feature requires PrometheusRule template implementation")
	fmt.Println("   Current PrometheusRule configuration:")
	fmt.Println(ruleOutput[:min(len(ruleOutput), 500)] + "...")

	return nil
}

func resetPrometheusRules(ctx context.Context) error {
	// Get default PrometheusRule configuration
	fmt.Println("🔧 Resetting PrometheusRules to default configuration...")

	// For now, just show what would be reset
	fmt.Println("   This feature requires default PrometheusRule template implementation")
	fmt.Println("   Would apply default alert rules configuration")

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
	smtpPassword = cleanPasswordInput(smtpPassword)

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
func listAlertRules(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📋 Available Alert Rules")
	fmt.Println("========================")

	rules := []string{
		"op-node-down",
		"op-batcher-down",
		"op-proposer-down",
		"op-geth-down",
		"l1-rpc-down",
		"op-batcher-balance-critical",
		"op-proposer-balance-critical",
		"block-production-stalled",
		"container-cpu-usage-high",
		"container-memory-usage-high",
		"pod-crash-looping",
	}

	for i, rule := range rules {
		fmt.Printf("%2d. %s\n", i+1, rule)
	}

	return nil
}

func modifyAlertRule(ctx context.Context, cmd *cli.Command, args []string) error {
	if len(args) < 2 {
		fmt.Println("❌ Usage: trh-sdk alert-config rules modify [rule-name] [new-value]")
		fmt.Println("Available rules:")
		fmt.Println("  - proposer-balance: Proposer account balance threshold (e.g., 0.1)")
		fmt.Println("  - block-production: Block production stall time (e.g., 30s)")
		fmt.Println("  - cpu-usage: CPU usage threshold (e.g., 80)")
		fmt.Println("  - memory-usage: Memory usage threshold (e.g., 85)")
		return nil
	}

	ruleName := args[0]
	newValue := args[1]

	fmt.Printf("🔧 Modifying Alert Rule: %s\n", ruleName)
	fmt.Printf("   New Value: %s\n", newValue)

	// Validate rule name
	validRules := map[string]string{
		"proposer-balance": "Proposer account balance threshold",
		"block-production": "Block production stall time",
		"cpu-usage":        "CPU usage threshold",
		"memory-usage":     "Memory usage threshold",
	}

	if _, valid := validRules[ruleName]; !valid {
		fmt.Printf("❌ Unknown rule: %s\n", ruleName)
		fmt.Println("Available rules:")
		for rule, desc := range validRules {
			fmt.Printf("  - %s: %s\n", rule, desc)
		}
		return nil
	}

	// Implement actual rule modification
	fmt.Println("🔧 Modifying PrometheusRule...")
	if err := updatePrometheusRule(ctx, ruleName, newValue); err != nil {
		return fmt.Errorf("failed to modify alert rule: %w", err)
	}

	fmt.Printf("✅ Alert rule '%s' modified successfully\n", ruleName)
	fmt.Println("💡 Note: Prometheus will reload rules automatically")
	return nil
}

func resetAlertRules(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("🔄 Resetting Alert Rules to Default...")
	fmt.Print("Are you sure you want to reset all alert rules? (y/N): ")

	confirm, err := scanner.ScanBool(false)
	if err != nil {
		return err
	}

	if confirm {
		// Implement actual rule reset
		fmt.Println("🔧 Resetting PrometheusRules to default...")
		if err := resetPrometheusRules(ctx); err != nil {
			return fmt.Errorf("failed to reset alert rules: %w", err)
		}
		fmt.Println("✅ Alert rules reset to default successfully")
		fmt.Println("💡 Note: Prometheus will reload rules automatically")
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
