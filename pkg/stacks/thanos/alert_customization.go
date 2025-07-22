package thanos

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

// AlertCustomization provides alert/notification management for ThanosStack
type AlertCustomization struct {
	Stack *ThanosStack
}

// GetAlertManagerConfig retrieves AlertManager config YAML (decompressed)
func (a *AlertCustomization) GetAlertManagerConfig(ctx context.Context) (string, error) {
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return "", fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}
	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return "", fmt.Errorf("could not find AlertManager config secret")
	}
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", "monitoring", secretName, "-o", "jsonpath={.data.alertmanager\\.yaml\\.gz}")
	if err != nil {
		return "", fmt.Errorf("failed to get AlertManager config from secret %s: %w", secretName, err)
	}
	output = strings.Trim(output, "' \n\t\r")
	decodedBytes, err := base64.StdEncoding.DecodeString(output)
	if err != nil {
		return "", fmt.Errorf("failed to decode AlertManager config: %w", err)
	}
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

// GetChannelStatus checks if a specific channel type is enabled in the configuration
func (a *AlertCustomization) GetChannelStatus(config string, channelType string) string {
	var amConfig types.AlertManagerParsedConfig
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return "Unknown"
	}

	for _, receiver := range amConfig.Receivers {
		switch channelType {
		case constants.ChannelEmail:
			if len(receiver.EmailConfigs) > 0 {
				return "Enabled"
			}
		case constants.ChannelTelegram:
			if len(receiver.TelegramConfigs) > 0 {
				return "Enabled"
			}
		default:
			continue
		}
	}
	return "Disabled"
}

// GetPrometheusRules retrieves all PrometheusRule items in the monitoring namespace
func (a *AlertCustomization) GetPrometheusRules(ctx context.Context) ([]types.AlertRule, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	var ruleList types.PrometheusRuleList
	if err := yaml.Unmarshal([]byte(output), &ruleList); err != nil {
		return nil, fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	if len(ruleList.Items) == 0 {
		return nil, fmt.Errorf("no PrometheusRule items found")
	}

	var allRules []types.AlertRule
	for _, prometheusRule := range ruleList.Items {
		for _, group := range prometheusRule.Spec.Groups {
			allRules = append(allRules, group.Rules...)
		}
	}

	return allRules, nil
}

// EnableRule enables a specific alert rule by name
func (a *AlertCustomization) EnableRule(ctx context.Context, ruleName string) error {
	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList types.PrometheusRuleList
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	if len(ruleList.Items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	// Get the first PrometheusRule
	rule := ruleList.Items[0]

	if len(rule.Spec.Groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	// Get the first group
	group := &rule.Spec.Groups[0]

	// Check if rule already exists
	for _, existingRule := range group.Rules {
		if existingRule.Alert == ruleName {
			return nil // Rule already enabled
		}
	}

	// Default values for rules
	defaultValues := map[string]string{
		constants.AlertOpBatcherBalanceCritical:  "0.01",
		constants.AlertOpProposerBalanceCritical: "0.01",
		constants.AlertBlockProductionStalled:    "1m",
		constants.AlertContainerCpuUsageHigh:     "80",
		constants.AlertContainerMemoryUsageHigh:  "80",
		constants.AlertPodCrashLooping:           "2m",
	}

	// Create new rule with default value
	defaultValue := defaultValues[ruleName]
	newRule := a.createRuleWithDefaultValue(ruleName, defaultValue)

	// Add the rule to the rules list
	group.Rules = append(group.Rules, newRule)

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
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// DisableRule disables a specific alert rule by name
func (a *AlertCustomization) DisableRule(ctx context.Context, ruleName string) error {
	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList types.PrometheusRuleList
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	if len(ruleList.Items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	// Get the first PrometheusRule
	rule := ruleList.Items[0]

	if len(rule.Spec.Groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	// Get the first group
	group := &rule.Spec.Groups[0]

	// Find and remove the rule
	var updatedRules []types.AlertRule
	for _, existingRule := range group.Rules {
		if existingRule.Alert != ruleName {
			updatedRules = append(updatedRules, existingRule)
		}
	}

	// Update the rules list
	group.Rules = updatedRules

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
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// ResetPrometheusRules resets all configurable alert rules to default values
func (a *AlertCustomization) ResetPrometheusRules(ctx context.Context) error {
	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList types.PrometheusRuleList
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	if len(ruleList.Items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	// Get the first PrometheusRule
	rule := ruleList.Items[0]

	if len(rule.Spec.Groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	// Get the first group
	group := &rule.Spec.Groups[0]

	// Default values for configurable rules
	defaultValues := map[string]string{
		constants.AlertOpBatcherBalanceCritical:  "0.01",
		constants.AlertOpProposerBalanceCritical: "0.01",
		constants.AlertBlockProductionStalled:    "1m",
		constants.AlertContainerCpuUsageHigh:     "80",
		constants.AlertContainerMemoryUsageHigh:  "80",
		constants.AlertPodCrashLooping:           "2m",
	}

	// Reset configurable rules to default values
	rulesReset := 0
	for ruleName, defaultValue := range defaultValues {
		// Find and update the rule
		for i, existingRule := range group.Rules {
			if existingRule.Alert == ruleName {
				// Update the rule expression with default value
				if err := a.updateRuleExpression(&group.Rules[i], ruleName, defaultValue); err != nil {
					return fmt.Errorf("failed to update rule %s: %w", ruleName, err)
				}
				rulesReset++
				break
			}
		}
	}

	if rulesReset == 0 {
		return fmt.Errorf("no configurable rules found to reset")
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
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// UpdatePrometheusRule updates a specific alert rule with a new value
func (a *AlertCustomization) UpdatePrometheusRule(ctx context.Context, ruleName, newValue string) error {
	// Get current PrometheusRule
	ruleOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRules: %w", err)
	}

	// Parse the YAML
	var ruleList types.PrometheusRuleList
	if err := yaml.Unmarshal([]byte(ruleOutput), &ruleList); err != nil {
		return fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}

	if len(ruleList.Items) == 0 {
		return fmt.Errorf("no PrometheusRule items found")
	}

	// Get the first PrometheusRule
	rule := ruleList.Items[0]

	if len(rule.Spec.Groups) == 0 {
		return fmt.Errorf("no groups found in PrometheusRule")
	}

	// Get the first group
	group := &rule.Spec.Groups[0]

	// Find and update the rule
	ruleFound := false
	for i, existingRule := range group.Rules {
		if existingRule.Alert == ruleName {
			// Update the rule expression
			if err := a.updateRuleExpression(&group.Rules[i], ruleName, newValue); err != nil {
				return fmt.Errorf("failed to update rule expression: %w", err)
			}

			// Update annotations if needed
			if err := a.updateRuleAnnotations(&group.Rules[i], ruleName, newValue); err != nil {
				return fmt.Errorf("failed to update rule annotations: %w", err)
			}

			ruleFound = true
			break
		}
	}

	if !ruleFound {
		return fmt.Errorf("rule '%s' not found", ruleName)
	}

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(ruleList)
	if err != nil {
		return fmt.Errorf("failed to marshal updated YAML: %w", err)
	}

	// Write updated YAML to temporary file
	tempFile := fmt.Sprintf("/tmp/prometheusrule-update-%d.yaml", time.Now().Unix())
	if err := os.WriteFile(tempFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}
	defer os.Remove(tempFile)

	// Apply the updated PrometheusRule
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// Helper methods
func (a *AlertCustomization) updateRuleExpression(rule *types.AlertRule, ruleName, newValue string) error {
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical:
		rule.Expr = fmt.Sprintf("op_batcher_default_balance < %s", newValue)
	case constants.AlertOpProposerBalanceCritical:
		rule.Expr = fmt.Sprintf("op_proposer_default_balance < %s", newValue)
	case constants.AlertBlockProductionStalled:
		rule.Expr = fmt.Sprintf("increase(op_node_blocks_produced_total[%s]) == 0", newValue)
	case constants.AlertContainerCpuUsageHigh:
		rule.Expr = fmt.Sprintf("(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m]) * 100) > %s", newValue)
	case constants.AlertContainerMemoryUsageHigh:
		rule.Expr = fmt.Sprintf("(container_memory_usage_bytes{container!=\"\"} / container_spec_memory_limit_bytes{container!=\"\"} * 100) > %s", newValue)
	case constants.AlertPodCrashLooping:
		rule.Expr = fmt.Sprintf("increase(kube_pod_container_status_restarts_total[%s]) > 0", newValue)
	default:
		return fmt.Errorf("unknown rule: %s", ruleName)
	}

	return nil
}

func (a *AlertCustomization) updateRuleAnnotations(rule *types.AlertRule, ruleName, newValue string) error {
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical, constants.AlertOpProposerBalanceCritical:
		rule.Annotations["current_value"] = fmt.Sprintf("%s ETH", newValue)
	case constants.AlertBlockProductionStalled:
		rule.Annotations["current_value"] = fmt.Sprintf("%s stall detection", newValue)
	case constants.AlertContainerCpuUsageHigh, constants.AlertContainerMemoryUsageHigh:
		rule.Annotations["current_value"] = fmt.Sprintf("%s%% threshold", newValue)
	case constants.AlertPodCrashLooping:
		rule.Annotations["current_value"] = fmt.Sprintf("%s restart detection", newValue)
	}

	return nil
}

// createRuleWithDefaultValue creates a rule with default value
func (a *AlertCustomization) createRuleWithDefaultValue(ruleName, defaultValue string) types.AlertRule {
	rule := types.AlertRule{
		Alert:       ruleName,
		Name:        ruleName,
		Description: fmt.Sprintf("Alert rule for %s", ruleName),
		Severity:    "critical",
		Threshold:   defaultValue,
		Enabled:     true,
		For:         "1m",
		Labels: map[string]string{
			"severity":   "critical",
			"component":  "thanos-stack",
			"chain_name": "thanos-stack",
			"namespace":  "monitoring",
		},
		Annotations: map[string]string{
			"summary":     fmt.Sprintf("Alert for %s", ruleName),
			"description": fmt.Sprintf("This alert is triggered when %s condition is met", ruleName),
		},
	}

	// Set expression based on rule type
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical:
		rule.Expr = fmt.Sprintf("op_batcher_default_balance < %s", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s ETH", defaultValue)
	case constants.AlertOpProposerBalanceCritical:
		rule.Expr = fmt.Sprintf("op_proposer_default_balance < %s", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s ETH", defaultValue)
	case constants.AlertBlockProductionStalled:
		rule.Expr = fmt.Sprintf("increase(op_node_blocks_produced_total[%s]) == 0", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s stall detection", defaultValue)
	case constants.AlertContainerCpuUsageHigh:
		rule.Expr = fmt.Sprintf("(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m]) * 100) > %s", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s%% threshold", defaultValue)
	case constants.AlertContainerMemoryUsageHigh:
		rule.Expr = fmt.Sprintf("(container_memory_usage_bytes{container!=\"\"} / container_spec_memory_limit_bytes{container!=\"\"} * 100) > %s", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s%% threshold", defaultValue)
	case constants.AlertPodCrashLooping:
		rule.Expr = fmt.Sprintf("increase(kube_pod_container_status_restarts_total[%s]) > 0", defaultValue)
		rule.Annotations["current_value"] = fmt.Sprintf("%s restart detection", defaultValue)
	}

	return rule
}

// ExtractValueFromExpression extracts the threshold value from a Prometheus expression
func (a *AlertCustomization) ExtractValueFromExpression(ruleName, expr string) string {
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical, constants.AlertOpProposerBalanceCritical:
		// Extract value from "op_batcher_default_balance < 0.01" or "op_proposer_default_balance < 0.01"
		if strings.Contains(expr, "<") {
			parts := strings.Split(expr, "<")
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				return value
			}
		}
	case constants.AlertBlockProductionStalled:
		// Extract value from "increase(op_node_blocks_produced_total[1m]) == 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[")
			end := strings.Index(expr, "]")
			if start != -1 && end != -1 && end > start {
				value := expr[start+1 : end]
				return value
			}
		}
	case constants.AlertContainerCpuUsageHigh, constants.AlertContainerMemoryUsageHigh:
		// Extract value from "> 80"
		if strings.Contains(expr, ">") {
			parts := strings.Split(expr, ">")
			if len(parts) == 2 {
				value := strings.TrimSpace(parts[1])
				return value
			}
		}
	case constants.AlertPodCrashLooping:
		// Extract value from "increase(kube_pod_container_status_restarts_total[2m]) > 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[")
			end := strings.Index(expr, "]")
			if start != -1 && end != -1 && end > start {
				value := expr[start+1 : end]
				return value
			}
		}
	}
	return ""
}

// AlertManager Channel Management Functions

// UpdateEmailConfig updates AlertManager with email configuration
func (a *AlertCustomization) UpdateEmailConfig(ctx context.Context, smtpServer, smtpFrom, smtpUsername, smtpPassword string, receivers []string) error {
	// Get current configuration to preserve existing settings
	currentConfig, err := a.GetAlertManagerConfig(ctx)
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
		}
		emailConfigs = append(emailConfigs, emailConfig)
	}

	mainReceiver["email_configs"] = emailConfigs

	// Update the receivers list
	config["receivers"] = receiversList

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Apply the updated configuration
	if err := a.applyAlertManagerConfig(ctx, string(updatedYAML)); err != nil {
		return fmt.Errorf("failed to apply AlertManager config: %w", err)
	}

	return nil
}

// RemoveEmailConfig removes email configuration from AlertManager
func (a *AlertCustomization) RemoveEmailConfig(ctx context.Context) error {
	// Get current configuration
	currentConfig, err := a.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse current YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse current AlertManager config: %w", err)
	}

	// Remove global SMTP settings
	global, ok := config["global"].(map[string]interface{})
	if ok {
		delete(global, "smtp_smarthost")
		delete(global, "smtp_from")
		delete(global, "smtp_auth_username")
		delete(global, "smtp_auth_password")
	}

	// Remove email_configs from receivers
	receiversList, ok := config["receivers"].([]interface{})
	if ok {
		for _, r := range receiversList {
			receiver, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			delete(receiver, "email_configs")
		}
		config["receivers"] = receiversList
	}

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Apply the updated configuration
	if err := a.applyAlertManagerConfig(ctx, string(updatedYAML)); err != nil {
		return fmt.Errorf("failed to apply AlertManager config: %w", err)
	}

	return nil
}

// UpdateTelegramConfig updates AlertManager with telegram configuration
func (a *AlertCustomization) UpdateTelegramConfig(ctx context.Context, botToken, chatID string) error {
	// Get current configuration to preserve existing settings
	currentConfig, err := a.GetAlertManagerConfig(ctx)
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
	telegramConfig := map[string]interface{}{
		"bot_token": botToken,
		"chat_id":   chatID,
	}

	mainReceiver["telegram_configs"] = []interface{}{telegramConfig}

	// Update the receivers list
	config["receivers"] = receiversList

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Apply the updated configuration
	if err := a.applyAlertManagerConfig(ctx, string(updatedYAML)); err != nil {
		return fmt.Errorf("failed to apply AlertManager config: %w", err)
	}

	return nil
}

// RemoveTelegramConfig removes telegram configuration from AlertManager
func (a *AlertCustomization) RemoveTelegramConfig(ctx context.Context) error {
	// Get current configuration
	currentConfig, err := a.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current AlertManager config: %w", err)
	}

	// Parse current YAML
	var config map[string]interface{}
	if err := yaml.Unmarshal([]byte(currentConfig), &config); err != nil {
		return fmt.Errorf("failed to parse current AlertManager config: %w", err)
	}

	// Remove telegram_configs from receivers
	receiversList, ok := config["receivers"].([]interface{})
	if ok {
		for _, r := range receiversList {
			receiver, ok := r.(map[string]interface{})
			if !ok {
				continue
			}
			delete(receiver, "telegram_configs")
		}
		config["receivers"] = receiversList
	}

	// Convert back to YAML
	updatedYAML, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Apply the updated configuration
	if err := a.applyAlertManagerConfig(ctx, string(updatedYAML)); err != nil {
		return fmt.Errorf("failed to apply AlertManager config: %w", err)
	}

	return nil
}

// applyAlertManagerConfig applies the updated AlertManager configuration
func (a *AlertCustomization) applyAlertManagerConfig(ctx context.Context, configYAML string) error {
	// Compress the config
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	if _, err := gw.Write([]byte(configYAML)); err != nil {
		return fmt.Errorf("failed to compress config: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Encode to base64
	encodedConfig := base64.StdEncoding.EncodeToString(buf.Bytes())

	// Get AlertManager Pod to find the actual secret being used
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", "monitoring", "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err != nil {
		return fmt.Errorf("failed to get AlertManager pod secret name: %w", err)
	}

	secretName := strings.TrimSpace(podOutput)
	if secretName == "" {
		return fmt.Errorf("could not find AlertManager config secret")
	}

	// Update the secret with the new config
	updateCmd := []string{
		"patch", "secret", secretName,
		"-n", "monitoring",
		"--type", "merge",
		"-p", fmt.Sprintf("{\"data\":{\"alertmanager.yaml.gz\":\"%s\"}}", encodedConfig),
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", updateCmd...); err != nil {
		return fmt.Errorf("failed to update AlertManager secret: %w", err)
	}

	// Restart AlertManager to pick up the new config
	restartCmd := []string{
		"rollout", "restart", "deployment",
		"-n", "monitoring",
		"-l", "app.kubernetes.io/name=alertmanager",
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", restartCmd...); err != nil {
		return fmt.Errorf("failed to restart AlertManager: %w", err)
	}

	return nil
}

// GetEmailConfiguration extracts email configuration from AlertManager config
func (a *AlertCustomization) GetEmailConfiguration(config string) types.EmailConfiguration {
	var amConfig types.AlertManagerParsedConfig

	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return types.EmailConfiguration{Enabled: false}
	}

	var toAddresses []string
	for _, receiver := range amConfig.Receivers {
		for _, emailConfig := range receiver.EmailConfigs {
			if emailConfig.To != "" {
				toAddresses = append(toAddresses, emailConfig.To)
			}
		}
	}

	if len(toAddresses) > 0 {
		return types.EmailConfiguration{
			Enabled: true,
			SmtpURL: amConfig.Global.SmtpSmarthost,
			From:    amConfig.Global.SmtpFrom,
			To:      strings.Join(toAddresses, ", "),
		}
	}

	return types.EmailConfiguration{Enabled: false}
}

// GetTelegramConfiguration extracts telegram configuration from AlertManager config
func (a *AlertCustomization) GetTelegramConfiguration(config string) types.TelegramConfiguration {
	var amConfig types.AlertManagerParsedConfig

	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return types.TelegramConfiguration{Enabled: false}
	}

	var botTokens []string
	var chatIDs []string

	for _, receiver := range amConfig.Receivers {
		for _, telegramConfig := range receiver.TelegramConfigs {
			if telegramConfig.BotToken != "" {
				botTokens = append(botTokens, telegramConfig.BotToken)
			}
			if telegramConfig.ChatID != "" {
				chatIDs = append(chatIDs, telegramConfig.ChatID)
			}
		}
	}

	if len(botTokens) > 0 || len(chatIDs) > 0 {
		return types.TelegramConfiguration{
			Enabled:  true,
			BotToken: strings.Join(botTokens, ", "),
			ChatID:   strings.Join(chatIDs, ", "),
		}
	}

	return types.TelegramConfiguration{Enabled: false}
}
