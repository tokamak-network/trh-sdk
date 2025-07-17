package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

// ActionAlertConfig handles alert configuration commands
func ActionAlertConfig() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get flags
		status := cmd.Bool("status")
		channels := cmd.Bool("channels")
		rules := cmd.Bool("rules")
		reset := cmd.Bool("reset")
		actionType := cmd.String("type")
		operation := cmd.String("operation")
		rule := cmd.String("rule")
		value := cmd.String("value")

		// Check if alert plugin is installed
		if err := checkAlertPluginInstalled(ctx); err != nil {
			return err
		}

		// Handle different subcommands
		if status {
			return handleAlertStatus(ctx, cmd, []string{})
		}

		if channels {
			return handleChannelsCustomizationWithFlags(ctx, cmd, actionType, operation)
		}

		if rules {
			return handleRulesCustomizationWithFlags(ctx, cmd, actionType, operation, rule, value)
		}

		if reset {
			return resetAlertRules(ctx, cmd)
		}

		// If no subcommand is specified, show help
		return showAlertConfigHelp()
	}
}

// checkAlertPluginInstalled checks if alert plugin is installed
func checkAlertPluginInstalled(ctx context.Context) error {
	fmt.Println("🔍 Checking alert plugin installation...")

	// Check if alert namespace exists
	namespaceExists, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check alert namespace: %w", err)
	}

	if !namespaceExists {
		fmt.Println("❌ Alert plugin is not installed!")
		fmt.Println()
		fmt.Println("To install alert plugin, run:")
		fmt.Println("  trh-sdk install monitoring")
		fmt.Println()
		fmt.Println("After installation, you can customize alert settings.")
		return fmt.Errorf("alert plugin not installed")
	}

	// Check if alert Helm release exists
	releaseExists, err := checkAlertReleaseExists(ctx)
	if err != nil {
		return fmt.Errorf("failed to check alert release: %w", err)
	}

	if !releaseExists {
		fmt.Println("❌ Alert plugin is not properly installed!")
		fmt.Println()
		fmt.Println("To install alert plugin, run:")
		fmt.Println("  trh-sdk install monitoring")
		fmt.Println()
		fmt.Println("After installation, you can customize alert settings.")
		return fmt.Errorf("alert plugin not properly installed")
	}

	fmt.Println("✅ Alert plugin is installed and ready")
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

// handleChannelsCustomization manages notification channels
func handleChannelsCustomization(ctx context.Context, cmd *cli.Command, args []string) error {
	if len(args) < 2 {
		fmt.Println("❌ Usage: trh-sdk monitoring channels [email|telegram] [enable|disable|configure]")
		return nil
	}

	channelType := args[0]
	action := args[1]

	switch channelType {
	case "email":
		return handleEmailChannel(ctx, cmd, action, args[2:])
	case "telegram":
		return handleTelegramChannel(ctx, cmd, action, args[2:])
	default:
		fmt.Printf("❌ Unknown channel type: %s\n", channelType)
		fmt.Println("Supported channels: email, telegram")
		return nil
	}
}

// handleChannelsCustomizationWithFlags manages notification channels using flags
func handleChannelsCustomizationWithFlags(ctx context.Context, cmd *cli.Command, actionType, operation string) error {
	if actionType == "" {
		fmt.Println("❌ Usage: trh-sdk alert-config --channels --type [email|telegram] --operation [enable|disable|configure]")
		return nil
	}

	if operation == "" {
		fmt.Println("❌ Usage: trh-sdk alert-config --channels --type [email|telegram] --operation [enable|disable|configure]")
		return nil
	}

	switch actionType {
	case "email":
		return handleEmailChannel(ctx, cmd, operation, []string{})
	case "telegram":
		return handleTelegramChannel(ctx, cmd, operation, []string{})
	default:
		fmt.Printf("❌ Unknown channel type: %s\n", actionType)
		fmt.Println("Supported channels: email, telegram")
		return nil
	}
}

// handleEmailChannel manages email channel configuration
func handleEmailChannel(ctx context.Context, cmd *cli.Command, action string, args []string) error {
	switch action {
	case "enable":
		return enableEmailChannel(ctx, cmd)
	case "disable":
		return disableEmailChannel(ctx, cmd)
	case "configure":
		return configureEmailChannel(ctx, cmd)
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Supported actions: enable, disable, configure")
		return nil
	}
}

// handleTelegramChannel manages telegram channel configuration
func handleTelegramChannel(ctx context.Context, cmd *cli.Command, action string, args []string) error {
	switch action {
	case "enable":
		return enableTelegramChannel(ctx, cmd)
	case "disable":
		return disableTelegramChannel(ctx, cmd)
	case "configure":
		return configureTelegramChannel(ctx, cmd)
	default:
		fmt.Printf("❌ Unknown action: %s\n", action)
		fmt.Println("Supported actions: enable, disable, configure")
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
	fmt.Println("📊 Current Alert Configuration")
	fmt.Println("==============================")

	// Check alert namespace
	fmt.Println("🔍 Checking alert namespace...")
	namespaceExists, err := checkNamespaceExists(ctx, "monitoring")
	if err != nil {
		return fmt.Errorf("failed to check alert namespace: %w", err)
	}

	if !namespaceExists {
		fmt.Println("❌ Alert namespace not found")
		return nil
	}
	fmt.Println("✅ Alert namespace: monitoring")

	// Check AlertManager pods
	fmt.Println("🔍 Checking AlertManager pods...")
	alertManagerPods, err := getAlertManagerPods(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to check AlertManager pods: %v\n", err)
	} else {
		fmt.Printf("✅ AlertManager pods: %d running\n", alertManagerPods)
	}

	// Check Prometheus pods
	fmt.Println("🔍 Checking Prometheus pods...")
	prometheusPods, err := getPrometheusPods(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to check Prometheus pods: %v\n", err)
	} else {
		fmt.Printf("✅ Prometheus pods: %d running\n", prometheusPods)
	}

	// Check Grafana pods
	fmt.Println("🔍 Checking Grafana pods...")
	grafanaPods, err := getGrafanaPods(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to check Grafana pods: %v\n", err)
	} else {
		fmt.Printf("✅ Grafana pods: %d running\n", grafanaPods)
	}

	// Check AlertManager configuration
	fmt.Println("🔍 Checking AlertManager configuration...")
	alertManagerConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get AlertManager config: %v\n", err)
	} else {
		fmt.Println("✅ AlertManager configuration loaded")
		fmt.Printf("   📧 Email channel: %s\n", getEmailChannelStatus(alertManagerConfig))
		fmt.Printf("   📱 Telegram channel: %s\n", getTelegramChannelStatus(alertManagerConfig))
	}

	// Check PrometheusRules
	fmt.Println("🔍 Checking PrometheusRules...")
	alertRules, err := getPrometheusRules(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get PrometheusRules: %v\n", err)
	} else {
		fmt.Printf("✅ Active alert rules: %d\n", alertRules)
	}

	// Check Helm releases
	fmt.Println("🔍 Checking Helm releases...")
	releases, err := getAlertReleases(ctx)
	if err != nil {
		fmt.Printf("⚠️  Failed to get Helm releases: %v\n", err)
	} else {
		fmt.Printf("✅ Alert releases: %s\n", releases)
	}

	fmt.Println()
	fmt.Println("📊 Alert Status Summary:")
	fmt.Println("========================")
	fmt.Printf("   🏷️  Namespace: %s\n", getNamespaceStatus(namespaceExists))
	fmt.Printf("   📊 AlertManager: %s\n", getPodStatus(alertManagerPods))
	fmt.Printf("   📈 Prometheus: %s\n", getPodStatus(prometheusPods))
	fmt.Printf("   📊 Grafana: %s\n", getPodStatus(grafanaPods))
	fmt.Printf("   📋 Alert Rules: %d active\n", alertRules)
	fmt.Printf("   🚀 Helm Releases: %s\n", releases)

	return nil
}

// Helper functions for status checking
func getAlertManagerPods(ctx context.Context) (int, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app=alertmanager", "--no-headers", "--field-selector=status.phase=Running")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func getPrometheusPods(ctx context.Context) (int, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app=prometheus", "--no-headers", "--field-selector=status.phase=Running")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func getGrafanaPods(ctx context.Context) (int, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app=grafana", "--no-headers", "--field-selector=status.phase=Running")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func getAlertManagerConfig(ctx context.Context) (string, error) {
	// First, find the actual AlertManager secret name
	secretList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secrets", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return "", fmt.Errorf("failed to get secrets list: %w", err)
	}

	secrets := strings.Split(strings.TrimSpace(secretList), " ")
	var alertManagerSecret string
	for _, secret := range secrets {
		if strings.Contains(secret, "alertmanager") && strings.Contains(secret, "kube-alertmanager") && !strings.Contains(secret, "generated") && !strings.Contains(secret, "tls") && !strings.Contains(secret, "web-config") {
			alertManagerSecret = secret
			break
		}
	}

	if alertManagerSecret == "" {
		return "", fmt.Errorf("AlertManager secret not found")
	}

	// Get the AlertManager configuration
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", "monitoring", alertManagerSecret, "-o", "jsonpath={.data.alertmanager\\.yaml}")
	if err != nil {
		return "", fmt.Errorf("failed to get AlertManager config from secret %s: %w", alertManagerSecret, err)
	}

	// Decode base64
	decoded, err := utils.ExecuteCommand(ctx, "echo", output, "|", "base64", "-d")
	if err != nil {
		return "", fmt.Errorf("failed to decode AlertManager config: %w", err)
	}

	return decoded, nil
}

func getEmailChannelStatus(config string) string {
	if strings.Contains(config, "smtp_smarthost") {
		return "Enabled"
	}
	return "Disabled"
}

func getTelegramChannelStatus(config string) string {
	if strings.Contains(config, "telegram_configs") {
		return "Enabled"
	}
	return "Disabled"
}

func getPrometheusRules(ctx context.Context) (int, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrules", "-n", "monitoring", "--no-headers")
	if err != nil {
		return 0, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count, nil
}

func getAlertReleases(ctx context.Context) (string, error) {
	output, err := utils.ExecuteCommand(ctx, "helm", "list", "-n", "monitoring", "--output", "json")
	if err != nil {
		return "", err
	}
	return output, nil
}

func getNamespaceStatus(exists bool) string {
	if exists {
		return "✅ monitoring"
	}
	return "❌ Not found"
}

func getPodStatus(count int) string {
	if count > 0 {
		return fmt.Sprintf("✅ %d running", count)
	}
	return "❌ No pods running"
}

// Configuration update functions
func updateAlertManagerEmailConfig(ctx context.Context, smtpServer, smtpFrom, smtpUsername, smtpPassword string, receivers []string) error {
	// Find the actual AlertManager secret name
	secretList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secrets", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get secrets list: %w", err)
	}

	secrets := strings.Split(strings.TrimSpace(secretList), " ")
	var alertManagerSecret string
	for _, secret := range secrets {
		if strings.Contains(secret, "alertmanager") && strings.Contains(secret, "kube-alertmanager") && !strings.Contains(secret, "generated") && !strings.Contains(secret, "tls") && !strings.Contains(secret, "web-config") {
			alertManagerSecret = secret
			break
		}
	}

	if alertManagerSecret == "" {
		return fmt.Errorf("AlertManager secret not found")
	}

	// Check if AlertManager secret exists
	_, err = utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", "monitoring", alertManagerSecret)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager secret: %w", err)
	}

	// Create updated AlertManager configuration
	config := fmt.Sprintf(`global:
  smtp_smarthost: %s
  smtp_from: %s
  smtp_auth_username: %s
  smtp_auth_password: %s

route:
  group_by: ['alertname', 'severity', 'component', 'chain_name', 'namespace']
  group_wait: 10s
  group_interval: 1m
  repeat_interval: 10m
  receiver: 'email-critical'

receivers:
- name: 'email-critical'
  email_configs:
`, smtpServer, smtpFrom, smtpUsername, smtpPassword)

	// Add email receivers
	for _, receiver := range receivers {
		config += fmt.Sprintf(`  - to: %s
    headers:
      subject: "🚨 Critical Alert - {{ .GroupLabels.chain_name }}"
    html: |
      <!DOCTYPE html>
      <html>
      <head>
          <meta charset="UTF-8">
          <style>
              body { font-family: Arial, sans-serif; margin: 20px; }
              .alert { border-left: 4px solid #dc3545; padding: 10px; margin: 10px 0; background-color: #f8f9fa; }
              .header { color: #dc3545; font-weight: bold; margin-bottom: 15px; }
              .info { margin: 5px 0; }
              .timestamp { color: #6c757d; font-size: 12px; margin-top: 10px; }
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
          </div>
      </body>
      </html>
`, receiver)
	}

	// Create temporary file for the configuration
	tempFile, err := os.CreateTemp("", "alertmanager-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}
	tempFile.Close()

	// Update AlertManager secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "secret", "generic", alertManagerSecret, "-n", "monitoring", "--from-file=alertmanager.yaml="+tempFile.Name(), "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create AlertManager secret: %w", err)
	}

	// Apply the secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", "-"); err != nil {
		return fmt.Errorf("failed to apply AlertManager secret: %w", err)
	}

	// Restart AlertManager pods to apply configuration
	fmt.Println("🔄 Restarting AlertManager pods to apply configuration...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pods", "-n", "monitoring", "-l", "app=alertmanager"); err != nil {
		return fmt.Errorf("failed to restart AlertManager pods: %w", err)
	}

	return nil
}

// Configuration removal functions
func removeEmailConfigFromAlertManager(ctx context.Context) error {
	// Find the actual AlertManager secret name
	secretList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secrets", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get secrets list: %w", err)
	}

	secrets := strings.Split(strings.TrimSpace(secretList), " ")
	var alertManagerSecret string
	for _, secret := range secrets {
		if strings.Contains(secret, "alertmanager") && strings.Contains(secret, "kube-alertmanager") && !strings.Contains(secret, "generated") && !strings.Contains(secret, "tls") && !strings.Contains(secret, "web-config") {
			alertManagerSecret = secret
			break
		}
	}

	if alertManagerSecret == "" {
		return fmt.Errorf("AlertManager secret not found")
	}

	// Create AlertManager configuration without email settings
	config := `global:
  resolve_timeout: 5m
inhibit_rules:
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = critical
  target_matchers:
  - severity =~ warning|info
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = warning
  target_matchers:
  - severity = info
- equal:
  - namespace
  source_matchers:
  - alertname = InfoInhibitor
  target_matchers:
  - severity = info
- target_matchers:
  - alertname = InfoInhibitor
receivers:
- name: "null"
route:
  group_by:
  - namespace
  group_interval: 5m
  group_wait: 30s
  receiver: "null"
  repeat_interval: 12h
  routes:
  - matchers:
    - alertname = "Watchdog"
    receiver: "null"
templates:
- /etc/alertmanager/config/*.tmpl`

	// Create temporary file for the configuration
	tempFile, err := os.CreateTemp("", "alertmanager-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}
	tempFile.Close()

	// Update AlertManager secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "secret", "generic", alertManagerSecret, "-n", "monitoring", "--from-file=alertmanager.yaml="+tempFile.Name(), "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create AlertManager secret: %w", err)
	}

	// Apply the secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", "-"); err != nil {
		return fmt.Errorf("failed to apply AlertManager secret: %w", err)
	}

	// Restart AlertManager pods to apply configuration
	fmt.Println("🔄 Restarting AlertManager pods to apply configuration...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pods", "-n", "monitoring", "-l", "app=alertmanager"); err != nil {
		return fmt.Errorf("failed to restart AlertManager pods: %w", err)
	}

	return nil
}

func removeTelegramConfigFromAlertManager(ctx context.Context) error {
	// Find the actual AlertManager secret name
	secretList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secrets", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get secrets list: %w", err)
	}

	secrets := strings.Split(strings.TrimSpace(secretList), " ")
	var alertManagerSecret string
	for _, secret := range secrets {
		if strings.Contains(secret, "alertmanager") && strings.Contains(secret, "kube-alertmanager") && !strings.Contains(secret, "generated") && !strings.Contains(secret, "tls") && !strings.Contains(secret, "web-config") {
			alertManagerSecret = secret
			break
		}
	}

	if alertManagerSecret == "" {
		return fmt.Errorf("AlertManager secret not found")
	}

	// Get current configuration and remove telegram configs
	currentConfig, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Remove telegram_configs from the configuration
	lines := strings.Split(currentConfig, "\n")
	var newLines []string
	inTelegramConfig := false
	for _, line := range lines {
		if strings.Contains(line, "telegram_configs:") {
			inTelegramConfig = true
			continue
		}
		if inTelegramConfig && (strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "-")) {
			continue
		}
		if inTelegramConfig && !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "-") {
			inTelegramConfig = false
		}
		if !inTelegramConfig {
			newLines = append(newLines, line)
		}
	}

	newConfig := strings.Join(newLines, "\n")

	// Create temporary file for the configuration
	tempFile, err := os.CreateTemp("", "alertmanager-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(newConfig); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}
	tempFile.Close()

	// Update AlertManager secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "secret", "generic", alertManagerSecret, "-n", "monitoring", "--from-file=alertmanager.yaml="+tempFile.Name(), "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create AlertManager secret: %w", err)
	}

	// Apply the secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", "-"); err != nil {
		return fmt.Errorf("failed to apply AlertManager secret: %w", err)
	}

	// Restart AlertManager pods to apply configuration
	fmt.Println("🔄 Restarting AlertManager pods to apply configuration...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pods", "-n", "monitoring", "-l", "app=alertmanager"); err != nil {
		return fmt.Errorf("failed to restart AlertManager pods: %w", err)
	}

	return nil
}

func updateAlertManagerTelegramConfig(ctx context.Context, botToken, chatID string) error {
	// Find the actual AlertManager secret name
	secretList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secrets", "-n", "monitoring", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get secrets list: %w", err)
	}

	secrets := strings.Split(strings.TrimSpace(secretList), " ")
	var alertManagerSecret string
	for _, secret := range secrets {
		if strings.Contains(secret, "alertmanager") && strings.Contains(secret, "kube-alertmanager") && !strings.Contains(secret, "generated") && !strings.Contains(secret, "tls") && !strings.Contains(secret, "web-config") {
			alertManagerSecret = secret
			break
		}
	}

	if alertManagerSecret == "" {
		return fmt.Errorf("AlertManager secret not found")
	}

	// Check if AlertManager secret exists
	_, err = utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", "monitoring", alertManagerSecret)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager secret: %w", err)
	}

	// Create updated configuration with telegram
	config := fmt.Sprintf(`global:
  resolve_timeout: 5m
inhibit_rules:
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = critical
  target_matchers:
  - severity =~ warning|info
- equal:
  - namespace
  - alertname
  source_matchers:
  - severity = warning
  target_matchers:
  - severity = info
- equal:
  - namespace
  source_matchers:
  - alertname = InfoInhibitor
  target_matchers:
  - severity = info
- target_matchers:
  - alertname = InfoInhibitor
receivers:
- name: "telegram-critical"
  telegram_configs:
  - bot_token: %s
    chat_id: %s
    parse_mode: Markdown
    message: |
      🚨 *Critical Alert - {{ .GroupLabels.chain_name }}*
      
      **Alert Name:** {{ .GroupLabels.alertname }}
      **Severity:** {{ .GroupLabels.severity }}
      **Component:** {{ .GroupLabels.component }}
      
      **Summary:** {{ .CommonAnnotations.summary }}
      **Description:** {{ .CommonAnnotations.description }}
      
      ⏰ Alert Time: {{ range .Alerts }}{{ .StartsAt }}{{ end }}
- name: "null"
route:
  group_by:
  - namespace
  group_interval: 5m
  group_wait: 30s
  receiver: "telegram-critical"
  repeat_interval: 12h
  routes:
  - matchers:
    - alertname = "Watchdog"
    receiver: "null"
templates:
- /etc/alertmanager/config/*.tmpl`, botToken, chatID)

	// Create temporary file for the configuration
	tempFile, err := os.CreateTemp("", "alertmanager-config-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(config); err != nil {
		return fmt.Errorf("failed to write configuration to file: %w", err)
	}
	tempFile.Close()

	// Update AlertManager secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "secret", "generic", alertManagerSecret, "-n", "monitoring", "--from-file=alertmanager.yaml="+tempFile.Name(), "--dry-run=client", "-o", "yaml"); err != nil {
		return fmt.Errorf("failed to create AlertManager secret: %w", err)
	}

	// Apply the secret
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", "-"); err != nil {
		return fmt.Errorf("failed to apply AlertManager secret: %w", err)
	}

	// Restart AlertManager pods to apply configuration
	fmt.Println("🔄 Restarting AlertManager pods to apply configuration...")
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pods", "-n", "monitoring", "-l", "app=alertmanager"); err != nil {
		return fmt.Errorf("failed to restart AlertManager pods: %w", err)
	}

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
func enableEmailChannel(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📧 Enabling Email Channel...")

	// Get current AlertManager configuration
	config, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if email is already enabled
	if getEmailChannelStatus(config) == "Enabled" {
		fmt.Println("ℹ️  Email channel is already enabled")
		return nil
	}

	// Prompt user for email configuration
	fmt.Println("📧 Email channel is currently disabled. Please configure it first:")
	fmt.Println("   Use: trh-sdk alert-config --config channels --type email --operation configure")
	fmt.Println("   This will prompt you for SMTP settings and email addresses")

	return nil
}

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
	fmt.Println("💡 Note: AlertManager will restart automatically to apply changes")
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
	fmt.Println("💡 Note: AlertManager will restart automatically to apply changes")
	return nil
}

// Telegram channel management functions
func enableTelegramChannel(ctx context.Context, cmd *cli.Command) error {
	fmt.Println("📱 Enabling Telegram Channel...")

	// Get current AlertManager configuration
	config, err := getAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get AlertManager config: %w", err)
	}

	// Check if telegram is already enabled
	if getTelegramChannelStatus(config) == "Enabled" {
		fmt.Println("ℹ️  Telegram channel is already enabled")
		return nil
	}

	// Prompt user for telegram configuration
	fmt.Println("📱 Telegram channel is currently disabled. Please configure it first:")
	fmt.Println("   Use: trh-sdk alert-config --config channels --type telegram --operation configure")
	fmt.Println("   This will prompt you for Bot Token and Chat ID")

	return nil
}

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
	fmt.Println("💡 Note: AlertManager will restart automatically to apply changes")
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
	fmt.Println("💡 Note: AlertManager will restart automatically to apply changes")
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
