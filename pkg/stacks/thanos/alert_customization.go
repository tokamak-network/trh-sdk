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
		var configKey string
		switch channelType {
		case "email":
			configKey = "email_configs"
		case "telegram":
			configKey = "telegram_configs"
		default:
			continue
		}
		if _, exists := receiver[configKey]; exists {
			return "Enabled"
		}
	}
	return "Disabled"
}

// GetPrometheusRules retrieves all PrometheusRule items in the monitoring namespace
func (a *AlertCustomization) GetPrometheusRules(ctx context.Context) ([]map[string]interface{}, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", "monitoring", "-o", "yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to get PrometheusRules: %w", err)
	}
	var ruleList map[string]interface{}
	if err := yaml.Unmarshal([]byte(output), &ruleList); err != nil {
		return nil, fmt.Errorf("failed to parse PrometheusRule YAML: %w", err)
	}
	items, ok := ruleList["items"].([]interface{})
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("no PrometheusRule items found")
	}
	var allRules []map[string]interface{}
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
			for _, rule := range rules {
				ruleMap, ok := rule.(map[string]interface{})
				if ok {
					allRules = append(allRules, ruleMap)
				}
			}
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
			return nil // Rule already enabled
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
	newRule := a.createRuleWithDefaultValue(ruleName, defaultValue)

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
	var updatedRules []interface{}

	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			updatedRules = append(updatedRules, rule)
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
			// Skip this rule (remove it)
		} else {
			updatedRules = append(updatedRules, rule)
		}
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

	// Default values for configurable rules
	defaultValues := map[string]string{
		"OpBatcherBalanceCritical":  "0.01",
		"OpProposerBalanceCritical": "0.01",
		"BlockProductionStalled":    "1m",
		"ContainerCpuUsageHigh":     "80",
		"ContainerMemoryUsageHigh":  "80",
		"PodCrashLooping":           "2m",
	}

	// Reset configurable rules to default values
	rulesReset := 0
	for ruleName, defaultValue := range defaultValues {
		// Find and update the rule
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}

			if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
				// Update the rule expression with default value
				if err := a.updateRuleExpression(ruleMap, ruleName, defaultValue); err != nil {
					return fmt.Errorf("failed to update rule %s: %w", ruleName, err)
				}
				rulesReset++
				break
			}
		}
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
		return fmt.Errorf("failed to apply reset PrometheusRule: %w", err)
	}

	return nil
}

// UpdatePrometheusRule updates a specific rule with a new value
func (a *AlertCustomization) UpdatePrometheusRule(ctx context.Context, ruleName, newValue string) error {
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

	// Find and update the rule
	ruleFound := false
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}

		if alertName, exists := ruleMap["alert"]; exists && alertName == ruleName {
			// Update the rule expression
			if err := a.updateRuleExpression(ruleMap, ruleName, newValue); err != nil {
				return fmt.Errorf("failed to update rule expression: %w", err)
			}

			// Update annotations if needed
			if err := a.updateRuleAnnotations(ruleMap, ruleName, newValue); err != nil {
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
func (a *AlertCustomization) updateRuleExpression(ruleMap map[string]interface{}, ruleName, newValue string) error {
	switch ruleName {
	case "OpBatcherBalanceCritical":
		ruleMap["expr"] = fmt.Sprintf("op_batcher_default_balance < %s", newValue)
	case "OpProposerBalanceCritical":
		ruleMap["expr"] = fmt.Sprintf("op_proposer_default_balance < %s", newValue)
	case "BlockProductionStalled":
		ruleMap["expr"] = fmt.Sprintf("increase(op_node_blocks_produced_total[%s]) == 0", newValue)
	case "ContainerCpuUsageHigh":
		ruleMap["expr"] = fmt.Sprintf("(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m]) * 100) > %s", newValue)
	case "ContainerMemoryUsageHigh":
		ruleMap["expr"] = fmt.Sprintf("(container_memory_usage_bytes{container!=\"\"} / container_spec_memory_limit_bytes{container!=\"\"} * 100) > %s", newValue)
	case "PodCrashLooping":
		ruleMap["expr"] = fmt.Sprintf("increase(kube_pod_container_status_restarts_total[%s]) > 0", newValue)
	default:
		return fmt.Errorf("unknown rule: %s", ruleName)
	}

	return nil
}

func (a *AlertCustomization) updateRuleAnnotations(ruleMap map[string]interface{}, ruleName, newValue string) error {
	annotations, ok := ruleMap["annotations"].(map[string]interface{})
	if !ok {
		annotations = make(map[string]interface{})
		ruleMap["annotations"] = annotations
	}

	switch ruleName {
	case "OpBatcherBalanceCritical", "OpProposerBalanceCritical":
		annotations["current_value"] = fmt.Sprintf("%s ETH", newValue)
	case "BlockProductionStalled":
		annotations["current_value"] = fmt.Sprintf("%s stall detection", newValue)
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh":
		annotations["current_value"] = fmt.Sprintf("%s%% threshold", newValue)
	case "PodCrashLooping":
		annotations["current_value"] = fmt.Sprintf("%s restart detection", newValue)
	}

	return nil
}

func (a *AlertCustomization) createRuleWithDefaultValue(ruleName, defaultValue string) map[string]interface{} {
	rule := map[string]interface{}{
		"alert": ruleName,
		"expr":  "",
		"for":   "1m",
		"labels": map[string]interface{}{
			"severity": "warning",
		},
		"annotations": map[string]interface{}{
			"summary":     fmt.Sprintf("%s alert", ruleName),
			"description": fmt.Sprintf("%s condition detected", ruleName),
		},
	}

	// Set expression based on rule type
	switch ruleName {
	case "OpBatcherBalanceCritical":
		rule["expr"] = fmt.Sprintf("op_batcher_default_balance < %s", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s ETH", defaultValue)
	case "OpProposerBalanceCritical":
		rule["expr"] = fmt.Sprintf("op_proposer_default_balance < %s", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s ETH", defaultValue)
	case "BlockProductionStalled":
		rule["expr"] = fmt.Sprintf("increase(op_node_blocks_produced_total[%s]) == 0", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s stall detection", defaultValue)
	case "ContainerCpuUsageHigh":
		rule["expr"] = fmt.Sprintf("(rate(container_cpu_usage_seconds_total{container!=\"\"}[5m]) * 100) > %s", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s%% threshold", defaultValue)
	case "ContainerMemoryUsageHigh":
		rule["expr"] = fmt.Sprintf("(container_memory_usage_bytes{container!=\"\"} / container_spec_memory_limit_bytes{container!=\"\"} * 100) > %s", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s%% threshold", defaultValue)
	case "PodCrashLooping":
		rule["expr"] = fmt.Sprintf("increase(kube_pod_container_status_restarts_total[%s]) > 0", defaultValue)
		rule["annotations"].(map[string]interface{})["current_value"] = fmt.Sprintf("%s restart detection", defaultValue)
	}

	return rule
}

// ExtractValueFromExpression extracts the current value from a rule expression
func (a *AlertCustomization) ExtractValueFromExpression(ruleName, expr string) string {
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
		// Extract value from "increase(op_node_blocks_produced_total[1m]) == 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[")
			end := strings.Index(expr, "]")
			if start != -1 && end != -1 && start < end {
				return expr[start+1 : end]
			}
		}
	case "ContainerCpuUsageHigh", "ContainerMemoryUsageHigh":
		// Extract value from "> 80"
		if strings.Contains(expr, ">") {
			parts := strings.Split(expr, ">")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case "PodCrashLooping":
		// Extract value from "increase(kube_pod_container_status_restarts_total[2m]) > 0"
		if strings.Contains(expr, "[") && strings.Contains(expr, "]") {
			start := strings.Index(expr, "[")
			end := strings.Index(expr, "]")
			if start != -1 && end != -1 && start < end {
				return expr[start+1 : end]
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
func (a *AlertCustomization) GetEmailConfiguration(config string) map[string]interface{} {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return map[string]interface{}{"enabled": false}
	}

	// Get global SMTP settings
	global, ok := amConfig["global"].(map[string]interface{})
	if !ok {
		return map[string]interface{}{"enabled": false}
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
		return map[string]interface{}{"enabled": false}
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
		return map[string]interface{}{
			"enabled":  true,
			"smtp_url": smtpURL,
			"from":     from,
			"to":       strings.Join(toAddresses, ", "),
		}
	}

	return map[string]interface{}{"enabled": false}
}

// GetTelegramConfiguration extracts telegram configuration from AlertManager config
func (a *AlertCustomization) GetTelegramConfiguration(config string) map[string]interface{} {
	var amConfig map[string]interface{}
	if err := yaml.Unmarshal([]byte(config), &amConfig); err != nil {
		return map[string]interface{}{"enabled": false}
	}

	receivers, ok := amConfig["receivers"].([]interface{})
	if !ok {
		return map[string]interface{}{"enabled": false}
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
		return map[string]interface{}{
			"enabled":   true,
			"bot_token": strings.Join(botTokens, ", "),
			"chat_id":   strings.Join(chatIDs, ", "),
		}
	}

	return map[string]interface{}{"enabled": false}
}
