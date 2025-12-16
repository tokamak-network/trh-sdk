package thanos

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
	// Try multiple methods to find the correct AlertManager secret
	var secretName string

	// Method 1: Try to get from pod volume mount
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", constants.MonitoringNamespace, "-l", "app.kubernetes.io/name=alertmanager", "-o", "jsonpath={.items[0].spec.volumes[?(@.name=='config-volume')].secret.secretName}")
	if err == nil && strings.TrimSpace(podOutput) != "" {
		secretName = strings.TrimSpace(podOutput)
	} else {
		// Method 2: Try to get from AlertManager resource
		amOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "alertmanager", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.configSecret}")
		if err == nil && strings.TrimSpace(amOutput) != "" {
			secretName = strings.TrimSpace(amOutput)
		} else {
			// Method 3: Find the generated secret by prometheus-operator
			secretOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", constants.MonitoringNamespace, "-l", "managed-by=prometheus-operator", "-o", "jsonpath={.items[?(@.metadata.name contains 'alertmanager' && @.metadata.name contains 'generated')].metadata.name}")
			if err == nil && strings.TrimSpace(secretOutput) != "" {
				secretName = strings.TrimSpace(secretOutput)
			} else {
				return "", fmt.Errorf("could not find AlertManager config secret")
			}
		}
	}

	if secretName == "" {
		return "", fmt.Errorf("could not find AlertManager config secret")
	}

	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "secret", "-n", constants.MonitoringNamespace, secretName, "-o", "jsonpath={.data.alertmanager\\.yaml\\.gz}")
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
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "yaml")
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
	// Get the PrometheusRule name first
	ruleNameOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule name: %w", err)
	}

	if ruleNameOutput == "" {
		return fmt.Errorf("no PrometheusRule found in monitoring namespace")
	}

	// Get current rule value if it exists
	currentValue := a.GetCurrentRuleValue(ctx, ruleName)
	if currentValue == "" {
		// Use default value if no current value exists
		defaultValues := map[string]string{
			constants.AlertOpBatcherBalanceCritical:  "0.01",
			constants.AlertOpProposerBalanceCritical: "0.01",
			constants.AlertBlockProductionStalled:    "1m",
			constants.AlertContainerCpuUsageHigh:     "80",
			constants.AlertContainerMemoryUsageHigh:  "80",
			constants.AlertPodCrashLooping:           "2m",
		}
		currentValue = defaultValues[ruleName]
	}

	// Get all alert names to find the correct insertion index
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule alerts: %w", err)
	}

	alertNames := strings.Fields(output)
	insertIndex := len(alertNames) // Insert at the end

	// Create a JSON patch to add the rule
	var patchData string
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"OpBatcherBalanceCritical","expr":"op_batcher_default_balance < %s","for":"10s","labels":{"severity":"critical","component":"op-batcher"},"annotations":{"summary":"OP Batcher ETH balance critically low","description":"OP Batcher balance is {{ $value }} ETH, below %s ETH threshold"}}}]`, insertIndex, currentValue, currentValue)
	case constants.AlertOpProposerBalanceCritical:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"OpProposerBalanceCritical","expr":"op_proposer_default_balance < %s","for":"10s","labels":{"severity":"critical","component":"op-proposer"},"annotations":{"summary":"OP Proposer ETH balance critically low","description":"OP Proposer balance is {{ $value }} ETH, below %s ETH threshold"}}}]`, insertIndex, currentValue, currentValue)
	case constants.AlertBlockProductionStalled:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"BlockProductionStalled","expr":"increase(chain_head_block[%s]) == 0","for":"1m","labels":{"severity":"critical","component":"op-geth"},"annotations":{"summary":"Block production has stalled","description":"No new blocks have been produced for more than 1 minute"}}}]`, insertIndex, currentValue)
	case constants.AlertContainerCpuUsageHigh:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"ContainerCpuUsageHigh","expr":"(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"High CPU usage in Thanos Stack pod","description":"Pod {{ $labels.pod }} CPU usage has been above %s%% for more than 2 minutes"}}}]`, insertIndex, currentValue, currentValue)
	case constants.AlertContainerMemoryUsageHigh:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"ContainerMemoryUsageHigh","expr":"(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"High memory usage in Thanos Stack pod","description":"Pod {{ $labels.pod }} memory usage has been above %s%% for more than 2 minutes"}}}]`, insertIndex, currentValue, currentValue)
	case constants.AlertPodCrashLooping:
		patchData = fmt.Sprintf(`[{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"PodCrashLooping","expr":"rate(kube_pod_container_status_restarts_total[%s]) > 0","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"Pod is crash looping","description":"Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently for more than 2 minutes"}}}]`, insertIndex, currentValue)
	default:
		return fmt.Errorf("unknown rule: %s", ruleName)
	}

	// Apply the JSON patch
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "prometheusrule", ruleNameOutput, "-n", constants.MonitoringNamespace, "--type=json", "-p", patchData); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// DisableRule disables a specific alert rule by name
func (a *AlertCustomization) DisableRule(ctx context.Context, ruleName string) error {
	// Get the PrometheusRule name first
	ruleNameOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule name: %w", err)
	}

	if ruleNameOutput == "" {
		return fmt.Errorf("no PrometheusRule found in monitoring namespace")
	}

	// Get all alert names and find the index of the target rule
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule alerts: %w", err)
	}

	alertNames := strings.Fields(output)
	targetIndex := -1

	for i, alertName := range alertNames {
		if alertName == ruleName {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Errorf("rule %s not found in PrometheusRule", ruleName)
	}

	// Create a JSON patch to remove the rule at the found index
	patchData := fmt.Sprintf(`[{"op":"remove","path":"/spec/groups/0/rules/%d"}]`, targetIndex)

	// Apply the JSON patch
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "prometheusrule", ruleNameOutput, "-n", constants.MonitoringNamespace, "--type=json", "-p", patchData); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// ResetPrometheusRules resets all configurable alert rules to default values
func (a *AlertCustomization) ResetPrometheusRules(ctx context.Context) error {
	// Get the PrometheusRule name first
	ruleNameOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule name: %w", err)
	}

	if ruleNameOutput == "" {
		return fmt.Errorf("no PrometheusRule found in monitoring namespace")
	}

	// Get all alert names to find the indices of configurable rules
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule alerts: %w", err)
	}

	alertNames := strings.Fields(output)

	// Default values for configurable rules
	defaultValues := map[string]string{
		constants.AlertOpBatcherBalanceCritical:  "0.01",
		constants.AlertOpProposerBalanceCritical: "0.01",
		constants.AlertBlockProductionStalled:    "1m",
		constants.AlertContainerCpuUsageHigh:     "80",
		constants.AlertContainerMemoryUsageHigh:  "80",
		constants.AlertPodCrashLooping:           "2m",
	}

	// Create a JSON patch to reset all configurable rules
	patchData := `[`
	first := true

	// First, update existing enabled rules
	for ruleName, defaultValue := range defaultValues {
		// Find the index of the target rule
		targetIndex := -1
		for i, alertName := range alertNames {
			if alertName == ruleName {
				targetIndex = i
				break
			}
		}

		// Update existing rule if found
		if targetIndex != -1 {
			if !first {
				patchData += ","
			}

			switch ruleName {
			case constants.AlertOpBatcherBalanceCritical:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"op_batcher_default_balance < %s"}`, targetIndex, defaultValue)
			case constants.AlertOpProposerBalanceCritical:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"op_proposer_default_balance < %s"}`, targetIndex, defaultValue)
			case constants.AlertBlockProductionStalled:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"increase(chain_head_block[%s]) == 0"}`, targetIndex, defaultValue)
			case constants.AlertContainerCpuUsageHigh:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s"}`, targetIndex, defaultValue)
			case constants.AlertContainerMemoryUsageHigh:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s"}`, targetIndex, defaultValue)
			case constants.AlertPodCrashLooping:
				patchData += fmt.Sprintf(`{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"rate(kube_pod_container_status_restarts_total[%s]) > 0"}`, targetIndex, defaultValue)
			}

			first = false
		}
	}

	// Then, add disabled rules back
	for ruleName, defaultValue := range defaultValues {
		// Check if rule is currently disabled
		isEnabled := false
		for _, alertName := range alertNames {
			if alertName == ruleName {
				isEnabled = true
				break
			}
		}

		// Add disabled rule back
		if !isEnabled {
			if !first {
				patchData += ","
			}

			// Add rule at the end
			insertIndex := len(alertNames)

			switch ruleName {
			case constants.AlertOpBatcherBalanceCritical:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"OpBatcherBalanceCritical","expr":"op_batcher_default_balance < %s","for":"10s","labels":{"severity":"critical","component":"op-batcher"},"annotations":{"summary":"OP Batcher ETH balance critically low","description":"OP Batcher balance is {{ $value }} ETH, below %s ETH threshold"}}}`, insertIndex, defaultValue, defaultValue)
			case constants.AlertOpProposerBalanceCritical:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"OpProposerBalanceCritical","expr":"op_proposer_default_balance < %s","for":"10s","labels":{"severity":"critical","component":"op-proposer"},"annotations":{"summary":"OP Proposer ETH balance critically low","description":"OP Proposer balance is {{ $value }} ETH, below %s ETH threshold"}}}`, insertIndex, defaultValue, defaultValue)
			case constants.AlertBlockProductionStalled:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"BlockProductionStalled","expr":"increase(chain_head_block[%s]) == 0","for":"1m","labels":{"severity":"critical","component":"op-geth"},"annotations":{"summary":"Block production has stalled","description":"No new blocks have been produced for more than 1 minute"}}}`, insertIndex, defaultValue)
			case constants.AlertContainerCpuUsageHigh:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"ContainerCpuUsageHigh","expr":"(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"High CPU usage in Thanos Stack pod","description":"Pod {{ $labels.pod }} CPU usage has been above %s%% for more than 2 minutes"}}}`, insertIndex, defaultValue, defaultValue)
			case constants.AlertContainerMemoryUsageHigh:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"ContainerMemoryUsageHigh","expr":"(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"High memory usage in Thanos Stack pod","description":"Pod {{ $labels.pod }} memory usage has been above %s%% for more than 2 minutes"}}}`, insertIndex, defaultValue, defaultValue)
			case constants.AlertPodCrashLooping:
				patchData += fmt.Sprintf(`{"op":"add","path":"/spec/groups/0/rules/%d","value":{"alert":"PodCrashLooping","expr":"rate(kube_pod_container_status_restarts_total[%s]) > 0","for":"2m","labels":{"severity":"critical","component":"kubernetes"},"annotations":{"summary":"Pod is crash looping","description":"Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently for more than 2 minutes"}}}`, insertIndex, defaultValue)
			}

			first = false
		}
	}

	patchData += `]`

	// Apply the JSON patch
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "prometheusrule", ruleNameOutput, "-n", constants.MonitoringNamespace, "--type=json", "-p", patchData); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
}

// UpdatePrometheusRule updates a specific rule's expression value
func (a *AlertCustomization) UpdatePrometheusRule(ctx context.Context, ruleName, newValue string) error {
	// Get the PrometheusRule name first
	ruleNameOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule name: %w", err)
	}

	if ruleNameOutput == "" {
		return fmt.Errorf("no PrometheusRule found in monitoring namespace")
	}

	// Get all alert names to find the index of the target rule
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return fmt.Errorf("failed to get PrometheusRule alerts: %w", err)
	}

	alertNames := strings.Fields(output)
	targetIndex := -1

	for i, alertName := range alertNames {
		if alertName == ruleName {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return fmt.Errorf("rule %s not found in PrometheusRule", ruleName)
	}

	// Create a JSON patch for the specific rule
	var patchData string

	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"op_batcher_default_balance < %s"}]`, targetIndex, newValue)
	case constants.AlertOpProposerBalanceCritical:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"op_proposer_default_balance < %s"}]`, targetIndex, newValue)
	case constants.AlertBlockProductionStalled:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"increase(chain_head_block[%s]) == 0"}]`, targetIndex, newValue)
	case constants.AlertContainerCpuUsageHigh:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"(sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > %s"}]`, targetIndex, newValue)
	case constants.AlertContainerMemoryUsageHigh:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"(sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > %s"}]`, targetIndex, newValue)
	case constants.AlertPodCrashLooping:
		patchData = fmt.Sprintf(`[{"op":"replace","path":"/spec/groups/0/rules/%d/expr","value":"rate(kube_pod_container_status_restarts_total[%s]) > 0"}]`, targetIndex, newValue)
	default:
		return fmt.Errorf("unknown rule: %s", ruleName)
	}

	// Apply the JSON patch
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "prometheusrule", ruleNameOutput, "-n", constants.MonitoringNamespace, "--type=json", "-p", patchData); err != nil {
		return fmt.Errorf("failed to apply updated PrometheusRule: %w", err)
	}

	return nil
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
	global["smtp_require_tls"] = true // Required for Gmail and other secure SMTP servers

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

	// Get Grafana URL for templates by finding the actual Helm release
	helmReleaseOutput, _ := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].metadata.name}")

	// Extract Helm release name from ingress name (remove -grafana suffix)
	helmReleaseName := strings.TrimSuffix(helmReleaseOutput, "-grafana")
	grafanaURL := a.Stack.getGrafanaURL(ctx, &types.MonitoringConfig{
		Namespace:       constants.MonitoringNamespace,
		HelmReleaseName: helmReleaseName,
	})

	// Add email_configs to the receiver
	emailConfigs := []interface{}{}

	for _, receiver := range receivers {
		emailConfig := map[string]interface{}{
			"to": receiver,
			"headers": map[string]string{
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
            <strong>Dashboard:</strong> <a href="` + grafanaURL + `">View Details</a>
        </div>
    </div>
</body>
</html>`,
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

	// Verify the configuration was actually applied
	fmt.Println("🔍 Verifying email configuration removal...")
	time.Sleep(5 * time.Second) // Wait for AlertManager to reload config

	// Get updated configuration
	updatedConfig, err := a.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get updated AlertManager config: %w", err)
	}

	// Check if email configuration was actually removed
	emailConfig := a.GetEmailConfiguration(updatedConfig)
	if emailConfig.Enabled {
		return fmt.Errorf("email configuration was not properly removed - still enabled")
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

	// Convert chatID to integer for AlertManager
	chatIDInt, err := strconv.ParseInt(chatID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid chat ID format: %w", err)
	}

	// Get Grafana URL for templates
	grafanaURL := a.Stack.getGrafanaURL(ctx, &types.MonitoringConfig{})
	templates := a.Stack.generateAlertTemplates(grafanaURL)

	// Add telegram_configs to the receiver
	telegramConfig := map[string]interface{}{
		"bot_token":  botToken,
		"chat_id":    chatIDInt,
		"message":    templates["telegram_message"],
		"parse_mode": "Markdown",
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

	// Verify the configuration was actually applied
	fmt.Println("🔍 Verifying telegram configuration removal...")
	time.Sleep(5 * time.Second) // Wait for AlertManager to reload config

	// Get updated configuration
	updatedConfig, err := a.GetAlertManagerConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get updated AlertManager config: %w", err)
	}

	// Check if telegram configuration was actually removed
	telegramConfig := a.GetTelegramConfiguration(updatedConfig)
	if telegramConfig.Enabled {
		return fmt.Errorf("telegram configuration was not properly removed - still enabled")
	}
	return nil
}

// applyAlertManagerConfig applies the updated AlertManager configuration
func (a *AlertCustomization) applyAlertManagerConfig(ctx context.Context, configYAML string) error {
	// Base64 encode the configuration
	encodedConfig := base64.StdEncoding.EncodeToString([]byte(configYAML))

	// Update the AlertManager configuration secret
	patchCmd := []string{
		"patch", "secret", "alertmanager-config",
		"-n", constants.MonitoringNamespace,
		"--type", "merge",
		"-p", fmt.Sprintf("{\"data\":{\"alertmanager.yaml\":\"%s\"}}", encodedConfig),
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", patchCmd...); err != nil {
		return fmt.Errorf("failed to patch AlertManager config secret: %w", err)
	}

	// Restart AlertManager pod to apply the new configuration
	// This ensures the updated config is loaded immediately without waiting for auto-reload
	deleteCmd := []string{
		"delete", "pod",
		"-n", constants.MonitoringNamespace,
		"-l", "app.kubernetes.io/name=alertmanager",
		"--ignore-not-found=true",
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", deleteCmd...); err != nil {
		return fmt.Errorf("failed to restart AlertManager pod: %w", err)
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

// GetCurrentRuleValue gets the current value of a rule from the configuration
func (a *AlertCustomization) GetCurrentRuleValue(ctx context.Context, ruleName string) string {
	// Get all alert names and expressions from the PrometheusRule
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return ""
	}

	alertNames := strings.Fields(output)
	if len(alertNames) == 0 {
		return ""
	}

	// Find the index of the target rule
	targetIndex := -1
	for i, alertName := range alertNames {
		if alertName == ruleName {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		return "" // Rule not found
	}

	// Get the expression for the target rule
	exprOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", fmt.Sprintf("jsonpath={.items[0].spec.groups[0].rules[%d].expr}", targetIndex))
	if err != nil {
		return ""
	}

	expr := strings.TrimSpace(exprOutput)
	if expr == "" {
		return ""
	}

	// Extract the value from the expression based on rule type
	switch ruleName {
	case constants.AlertOpBatcherBalanceCritical:
		// Extract value from "op_batcher_default_balance < VALUE"
		if strings.Contains(expr, "op_batcher_default_balance < ") {
			parts := strings.Split(expr, "op_batcher_default_balance < ")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case constants.AlertOpProposerBalanceCritical:
		// Extract value from "op_proposer_default_balance < VALUE"
		if strings.Contains(expr, "op_proposer_default_balance < ") {
			parts := strings.Split(expr, "op_proposer_default_balance < ")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case constants.AlertBlockProductionStalled:
		// Extract value from "increase(chain_head_block[VALUE]) == 0"
		if strings.Contains(expr, "increase(chain_head_block[") {
			start := strings.Index(expr, "increase(chain_head_block[") + len("increase(chain_head_block[")
			end := strings.Index(expr[start:], "]) == 0")
			if end != -1 {
				return expr[start : start+end]
			}
		}
	case constants.AlertContainerCpuUsageHigh:
		// Extract value from "> VALUE"
		if strings.Contains(expr, ") * 100 > ") {
			parts := strings.Split(expr, ") * 100 > ")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case constants.AlertContainerMemoryUsageHigh:
		// Extract value from "> VALUE"
		if strings.Contains(expr, ") * 100 > ") {
			parts := strings.Split(expr, ") * 100 > ")
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	case constants.AlertPodCrashLooping:
		// Extract value from "rate(kube_pod_container_status_restarts_total[VALUE]) > 0"
		if strings.Contains(expr, "rate(kube_pod_container_status_restarts_total[") {
			start := strings.Index(expr, "rate(kube_pod_container_status_restarts_total[") + len("rate(kube_pod_container_status_restarts_total[")
			end := strings.Index(expr[start:], "]) > 0")
			if end != -1 {
				return expr[start : start+end]
			}
		}
	}

	return ""
}

// IsRuleEnabled checks if a specific rule is currently enabled
func (a *AlertCustomization) IsRuleEnabled(ctx context.Context, ruleName string) (bool, error) {
	// Get all alert names from the PrometheusRule
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", constants.MonitoringNamespace, "-o", "jsonpath={.items[0].spec.groups[0].rules[*].alert}")
	if err != nil {
		return false, fmt.Errorf("failed to get PrometheusRule alerts: %w", err)
	}

	// Split the output and check if the rule name exists
	alertNames := strings.Fields(output)
	for _, alertName := range alertNames {
		if alertName == ruleName {
			return true, nil
		}
	}

	return false, nil
}
