package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

// MonitoringConfig holds all configuration needed for monitoring installation
type MonitoringConfig struct {
	Namespace         string
	HelmReleaseName   string
	AdminPassword     string
	L1RpcUrl          string
	ServiceNames      map[string]string
	EnablePersistence bool
	EFSFileSystemId   string
	ChartsPath        string
	ValuesFilePath    string
	ChainName         string
	AlertManager      types.AlertManagerConfig
}

// InstallMonitoring installs monitoring stack using Helm
func (t *ThanosStack) InstallMonitoring(ctx context.Context, config *MonitoringConfig) (string, error) {
	fmt.Println("🚀 Starting monitoring installation...")

	// Deploy infrastructure if persistence is enabled
	if config.EnablePersistence {
		if err := t.deployMonitoringInfrastructure(ctx, config); err != nil {
			return "", fmt.Errorf("failed to deploy monitoring infrastructure: %w", err)
		}
	}

	// Generate values file
	if err := t.generateValuesFile(ctx, config); err != nil {
		return "", fmt.Errorf("failed to generate values file: %w", err)
	}

	// Update chart dependencies
	if _, err := utils.ExecuteCommand(ctx, "helm", "dependency", "update", config.ChartsPath); err != nil {
		return "", fmt.Errorf("failed to update chart dependencies: %w", err)
	}

	// Install monitoring stack
	installCmd := []string{
		"upgrade", "--install",
		config.HelmReleaseName,
		config.ChartsPath,
		"--values", config.ValuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
		"--timeout", "15m",
		"--wait",
		"--wait-for-jobs",
	}

	if _, err := utils.ExecuteCommand(ctx, "helm", installCmd...); err != nil {
		return "", fmt.Errorf("failed to install monitoring stack: %w", err)
	}

	// Create additional resources
	t.createAlertManagerSecret(ctx, config)
	t.createPrometheusRule(ctx, config)
	t.createDashboardConfigMaps(ctx, config)

	return t.displayMonitoringInfo(ctx, config), nil
}

// GetMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) GetMonitoringConfig(ctx context.Context, adminPassword string) (*MonitoringConfig, error) {
	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")
	helmReleaseName := fmt.Sprintf("monitoring-%d", time.Now().Unix())

	chartsPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/monitoring", t.deploymentPath)
	if _, err := os.Stat(chartsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chart directory not found: %s", chartsPath)
	}

	serviceNames, err := t.getServiceNames(ctx, t.deployConfig.K8s.Namespace, chainName)
	if err != nil {
		return nil, fmt.Errorf("error getting service names: %w", err)
	}

	efsFileSystemId, err := t.getEFSFileSystemId(ctx, chainName)
	if err != nil {
		return nil, fmt.Errorf("error getting EFS filesystem ID: %w", err)
	}

	// Get AlertManager configuration from user input
	alertManagerConfig := t.getAlertManagerConfigFromUser()

	config := &MonitoringConfig{
		Namespace:         "monitoring",
		HelmReleaseName:   helmReleaseName,
		AdminPassword:     adminPassword,
		L1RpcUrl:          t.deployConfig.L1RPCURL,
		ServiceNames:      serviceNames,
		EnablePersistence: true,
		EFSFileSystemId:   efsFileSystemId,
		ChartsPath:        chartsPath,
		ValuesFilePath:    "",
		ChainName:         chainName,
		AlertManager:      alertManagerConfig,
	}

	return config, nil
}

// UninstallMonitoring removes monitoring stack
func (t *ThanosStack) UninstallMonitoring(ctx context.Context) error {
	monitoringNamespace := "monitoring"
	releases, err := utils.FilterHelmReleases(ctx, monitoringNamespace, "monitoring")
	if err != nil {
		return err
	}

	for _, release := range releases {
		utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", monitoringNamespace)
	}

	utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", monitoringNamespace, "--ignore-not-found=true")
	return nil
}

// displayMonitoringInfo shows access information and checks ALB Ingress
func (t *ThanosStack) displayMonitoringInfo(ctx context.Context, config *MonitoringConfig) string {
	fmt.Println("\n🎉 Monitoring Stack Installation Complete!")

	// Check ALB Ingress status and get URL
	albURL := t.checkALBIngressStatus(ctx, config)

	if albURL != "" {
		fmt.Printf("🌐 ALB Ingress URL: %s\n", albURL)
		fmt.Printf("   Username: admin\n")
		fmt.Printf("   Password: %s\n", config.AdminPassword)
		return albURL
	} else {
		fmt.Printf("⚠️  ALB Ingress not ready, using port-forward as fallback\n")
		fmt.Printf("📊 Grafana: http://localhost:3000 (port-forward)\n")
		fmt.Printf("   Username: admin\n")
		fmt.Printf("   Password: %s\n", config.AdminPassword)

		// Start port-forward in background
		go utils.ExecuteCommand(ctx, "kubectl", "port-forward", "-n", config.Namespace,
			fmt.Sprintf("svc/%s-grafana", config.HelmReleaseName), "3000:80")

		return "http://localhost:3000"
	}
}

// checkALBIngressStatus checks ALB Ingress status and returns the URL
func (t *ThanosStack) checkALBIngressStatus(ctx context.Context, config *MonitoringConfig) string {
	// Wait for ALB Ingress to be ready (max 5 minutes)
	maxAttempts := 60 // 60 attempts * 5 seconds = 5 minutes
	for i := 0; i < maxAttempts; i++ {
		// Check if ALB Ingress exists and has an address
		output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace,
			fmt.Sprintf("%s-grafana", config.HelmReleaseName), "-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}", "--ignore-not-found=true")

		if err == nil && strings.TrimSpace(output) != "" {
			albHostname := strings.TrimSpace(output)
			if strings.HasPrefix(albHostname, "internal-") || strings.HasPrefix(albHostname, "dualstack.") {
				return fmt.Sprintf("http://%s", albHostname)
			}
		}

		// Check if ALB Ingress is still being created
		output, err = utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace,
			fmt.Sprintf("%s-grafana", config.HelmReleaseName), "-o", "jsonpath={.status.loadBalancer.ingress}", "--ignore-not-found=true")

		if err != nil || strings.TrimSpace(output) == "" {
			fmt.Printf("⏳ Waiting for ALB Ingress to be ready... (%d/%d)\n", i+1, maxAttempts)
			time.Sleep(5 * time.Second)
			continue
		}

		// Try to get the hostname again
		output, err = utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace,
			fmt.Sprintf("%s-grafana", config.HelmReleaseName), "-o", "jsonpath={.status.loadBalancer.ingress[0].hostname}")

		if err == nil && strings.TrimSpace(output) != "" {
			albHostname := strings.TrimSpace(output)
			return fmt.Sprintf("http://%s", albHostname)
		}

		time.Sleep(5 * time.Second)
	}

	// If ALB Ingress is not ready after 5 minutes, return empty string
	fmt.Println("⚠️  ALB Ingress not ready after 5 minutes")
	return ""
}

// generateValuesFile creates the values.yaml file
func (t *ThanosStack) generateValuesFile(ctx context.Context, config *MonitoringConfig) error {
	valuesConfig := map[string]interface{}{
		"global": map[string]interface{}{
			"l1RpcUrl": config.L1RpcUrl,
			"storage": map[string]interface{}{
				"enabled":         config.EnablePersistence,
				"efsFileSystemId": config.EFSFileSystemId,
				"awsRegion":       t.deployConfig.AWS.Region,
			},
		},
		"thanosStack": map[string]interface{}{
			"chainName":   config.ChainName,
			"namespace":   t.deployConfig.K8s.Namespace,
			"releaseName": config.ChainName,
		},
		"alertManager": t.generateAlertManagerValues(config),
		"kube-prometheus-stack": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": t.generatePrometheusStorageSpec(config),
			},
			"grafana": t.generateGrafanaStorageConfig(ctx, config),
			"alertmanager": map[string]interface{}{
				"enabled": true,
				"alertmanagerSpec": map[string]interface{}{
					"configSecret": "alertmanager-config",
				},
			},
			"prometheusRule": map[string]interface{}{
				"enabled": false, // Disable default PrometheusRules, only use thanos-stack-alerts
			},
			// Disable all default rules to prevent conflicts
			"defaultRules": map[string]interface{}{
				"enabled": false,
			},
		},
	}

	yamlContent, err := yaml.Marshal(valuesConfig)
	if err != nil {
		return fmt.Errorf("error marshaling values to YAML: %w", err)
	}

	configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	valuesFilePath := filepath.Join(configFileDir, "monitoring-values.yaml")
	if err := os.WriteFile(valuesFilePath, yamlContent, 0644); err != nil {
		return fmt.Errorf("error writing values file: %w", err)
	}

	config.ValuesFilePath = valuesFilePath
	return nil
}

// generatePrometheusStorageSpec creates Prometheus storage specification
func (t *ThanosStack) generatePrometheusStorageSpec(config *MonitoringConfig) map[string]interface{} {
	if !config.EnablePersistence {
		return map[string]interface{}{}
	}

	timestamp, err := t.getTimestampFromExistingPV(context.Background(), config.ChainName)
	if err != nil {
		timestamp = "static"
	}
	pvName := fmt.Sprintf("%s-%s-thanos-stack-prometheus", config.ChainName, timestamp)

	return map[string]interface{}{
		"volumeClaimTemplate": map[string]interface{}{
			"spec": map[string]interface{}{
				"accessModes":      []string{"ReadWriteMany"},
				"storageClassName": "efs-sc",
				"resources": map[string]interface{}{
					"requests": map[string]interface{}{
						"storage": "20Gi",
					},
				},
				"selector": map[string]interface{}{
					"matchLabels": map[string]string{
						"app": pvName,
					},
				},
			},
		},
		"securityContext": map[string]interface{}{
			"runAsUser":    472,
			"runAsGroup":   472,
			"runAsNonRoot": true,
			"fsGroup":      472,
		},
	}
}

// generateGrafanaStorageConfig creates Grafana storage configuration
func (t *ThanosStack) generateGrafanaStorageConfig(ctx context.Context, config *MonitoringConfig) map[string]interface{} {
	grafanaConfig := map[string]interface{}{
		"adminPassword": config.AdminPassword,
	}

	if config.EnablePersistence {
		pvcName := fmt.Sprintf("%s-grafana", config.HelmReleaseName)
		grafanaConfig["persistence"] = map[string]interface{}{
			"enabled":          true,
			"storageClassName": "efs-sc",
			"accessModes":      []string{"ReadWriteMany"},
			"size":             "10Gi",
			"existingClaim":    pvcName,
		}
	} else {
		grafanaConfig["persistence"] = map[string]interface{}{
			"enabled": false,
		}
	}

	return grafanaConfig
}

// generateAlertManagerValues creates AlertManager values for the chart
func (t *ThanosStack) generateAlertManagerValues(config *MonitoringConfig) map[string]interface{} {
	// Convert Telegram receivers to chat IDs
	var telegramChatIds []string
	if config.AlertManager.Telegram.Enabled {
		for _, receiver := range config.AlertManager.Telegram.CriticalReceivers {
			if receiver.ChatId != "" {
				telegramChatIds = append(telegramChatIds, receiver.ChatId)
			}
		}
	}

	// Convert Email receivers
	var emailReceivers []string
	if config.AlertManager.Email.Enabled {
		emailReceivers = append(emailReceivers, config.AlertManager.Email.CriticalReceivers...)
	}

	return map[string]interface{}{
		"telegram": map[string]interface{}{
			"enabled":  config.AlertManager.Telegram.Enabled,
			"apiToken": config.AlertManager.Telegram.ApiToken,
			"chatIds":  telegramChatIds,
		},
		"email": map[string]interface{}{
			"enabled":      config.AlertManager.Email.Enabled,
			"smtpServer":   config.AlertManager.Email.SmtpSmarthost,
			"smtpFrom":     config.AlertManager.Email.SmtpFrom,
			"smtpUsername": config.AlertManager.Email.SmtpAuthUsername,
			"smtpPassword": config.AlertManager.Email.SmtpAuthPassword,
			"receivers":    emailReceivers,
		},
		"routing": map[string]interface{}{
			"defaultReceiver": "telegram-critical",
			"groupBy":         []string{"alertname", "cluster", "service", "severity"},
			"groupWait":       "30s",
			"groupInterval":   "5m",
			"repeatInterval":  "4h",
		},
	}
}

// getAlertManagerConfigFromUser gets AlertManager configuration from user input
func (t *ThanosStack) getAlertManagerConfigFromUser() types.AlertManagerConfig {
	fmt.Println("\n🔔 AlertManager Configuration")
	fmt.Println("================================")

	// Check if user wants to use default configuration
	fmt.Print("Use default AlertManager configuration? (y/n): ")
	var useDefault string
	fmt.Scanln(&useDefault)

	if strings.ToLower(useDefault) == "y" || strings.ToLower(useDefault) == "yes" {
		fmt.Println("✅ Using default AlertManager configuration")
		return t.getDefaultAlertManagerConfig()
	}

	// Telegram Configuration
	telegramConfig := t.getTelegramConfigFromUser()

	// Email Configuration
	emailConfig := t.getEmailConfigFromUser()

	// Show configuration summary
	fmt.Println("\n📋 AlertManager Configuration Summary")
	fmt.Println("===================================")
	fmt.Printf("Telegram: %s\n", t.getTelegramConfigSummary(telegramConfig))
	fmt.Printf("Email: %s\n", t.getEmailConfigSummary(emailConfig))

	return types.AlertManagerConfig{
		Telegram: telegramConfig,
		Email:    emailConfig,
	}
}

// getTelegramConfigFromUser gets Telegram configuration from user input
func (t *ThanosStack) getTelegramConfigFromUser() types.TelegramConfig {
	fmt.Println("\n📱 Telegram Configuration")
	fmt.Println("-------------------------")

	var enabled bool
	fmt.Print("Enable Telegram alerts? (y/n): ")
	var response string
	fmt.Scanln(&response)
	enabled = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"

	if !enabled {
		return types.TelegramConfig{Enabled: false}
	}

	var apiToken string
	fmt.Print("Enter Telegram Bot API Token: ")
	fmt.Scanln(&apiToken)

	var chatIds []string
	fmt.Print("Enter Telegram Chat IDs (comma-separated): ")
	var chatIdsInput string
	fmt.Scanln(&chatIdsInput)

	if chatIdsInput != "" {
		chatIds = strings.Split(chatIdsInput, ",")
		for i, id := range chatIds {
			chatIds[i] = strings.TrimSpace(id)
		}
	}

	var receivers []types.TelegramReceiver
	for _, chatId := range chatIds {
		if chatId != "" {
			receivers = append(receivers, types.TelegramReceiver{ChatId: chatId})
		}
	}

	return types.TelegramConfig{
		Enabled:           enabled,
		ApiToken:          apiToken,
		CriticalReceivers: receivers,
	}
}

// getEmailConfigFromUser gets Email configuration from user input
func (t *ThanosStack) getEmailConfigFromUser() types.EmailConfig {
	fmt.Println("\n📧 Email Configuration")
	fmt.Println("----------------------")

	var enabled bool
	fmt.Print("Enable Email alerts? (y/n): ")
	var response string
	fmt.Scanln(&response)
	enabled = strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"

	if !enabled {
		return types.EmailConfig{Enabled: false}
	}

	var smtpSmarthost string
	fmt.Print("Enter SMTP Server (e.g., smtp.gmail.com:587): ")
	fmt.Scanln(&smtpSmarthost)

	var smtpFrom string
	fmt.Print("Enter From Email Address: ")
	fmt.Scanln(&smtpFrom)

	var smtpAuthUsername string
	fmt.Print("Enter SMTP Username: ")
	fmt.Scanln(&smtpAuthUsername)

	var smtpAuthPassword string
	fmt.Print("Enter SMTP Password: ")
	fmt.Scanln(&smtpAuthPassword)

	var defaultReceiversInput string
	fmt.Print("Enter Default Email Receivers (comma-separated): ")
	fmt.Scanln(&defaultReceiversInput)

	var criticalReceiversInput string
	fmt.Print("Enter Critical Email Receivers (comma-separated): ")
	fmt.Scanln(&criticalReceiversInput)

	var defaultReceivers []string
	if defaultReceiversInput != "" {
		defaultReceivers = strings.Split(defaultReceiversInput, ",")
		for i, email := range defaultReceivers {
			defaultReceivers[i] = strings.TrimSpace(email)
		}
	}

	var criticalReceivers []string
	if criticalReceiversInput != "" {
		criticalReceivers = strings.Split(criticalReceiversInput, ",")
		for i, email := range criticalReceivers {
			criticalReceivers[i] = strings.TrimSpace(email)
		}
	}

	return types.EmailConfig{
		Enabled:           enabled,
		SmtpSmarthost:     smtpSmarthost,
		SmtpFrom:          smtpFrom,
		SmtpAuthUsername:  smtpAuthUsername,
		SmtpAuthPassword:  smtpAuthPassword,
		DefaultReceivers:  defaultReceivers,
		CriticalReceivers: criticalReceivers,
	}
}

// getDefaultAlertManagerConfig returns default AlertManager configuration
func (t *ThanosStack) getDefaultAlertManagerConfig() types.AlertManagerConfig {
	return types.AlertManagerConfig{
		Telegram: types.TelegramConfig{
			Enabled:  true,
			ApiToken: "7904495507:AAE54gXGoj5X7oLsQHk_xzMFdO1kkn4xME8",
			CriticalReceivers: []types.TelegramReceiver{
				{ChatId: "1266746900"},
			},
		},
		Email: types.EmailConfig{
			Enabled:           true,
			SmtpSmarthost:     "smtp.gmail.com:587",
			SmtpFrom:          "theo@tokamak.network",
			SmtpAuthUsername:  "theo@tokamak.network",
			SmtpAuthPassword:  "myhz wsqg iqcs hwkv",
			DefaultReceivers:  []string{"theo@tokamak.network"},
			CriticalReceivers: []string{"theo@tokamak.network"},
		},
	}
}

// getServiceNames returns a map of component names to their Kubernetes service names
func (t *ThanosStack) getServiceNames(ctx context.Context, namespace, chainName string) (map[string]string, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "services", "-n", namespace, "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("failed to get services in namespace %s: %w", namespace, err)
	}

	serviceNames := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}
		serviceName := strings.TrimSpace(line)
		if strings.Contains(serviceName, "op-node") {
			serviceNames["op-node"] = serviceName
		} else if strings.Contains(serviceName, "op-batcher") {
			serviceNames["op-batcher"] = serviceName
		} else if strings.Contains(serviceName, "op-proposer") {
			serviceNames["op-proposer"] = serviceName
		} else if strings.Contains(serviceName, "op-geth") {
			serviceNames["op-geth"] = serviceName
		}
	}

	return serviceNames, nil
}

// getEFSFileSystemId extracts EFS filesystem ID from existing PV
func (t *ThanosStack) getEFSFileSystemId(ctx context.Context, chainName string) (string, error) {
	// First try to get EFS filesystem ID from existing op-geth PV using a more reliable method
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name,VOLUMEHANDLE:.spec.csi.volumeHandle", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, chainName) && strings.Contains(line, "thanos-stack-op-geth") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				volumeHandle := fields[1]
				if strings.HasPrefix(volumeHandle, "fs-") {
					return volumeHandle, nil
				}
			}
		}
	}

	// Fallback: try to get from AWS EFS directly
	output, err = utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems", "--query", "FileSystems[0].FileSystemId", "--output", "text", "--region", t.deployConfig.AWS.Region)
	if err != nil {
		return "", fmt.Errorf("failed to get EFS filesystem ID from AWS: %w", err)
	}

	efsId := strings.TrimSpace(output)
	if efsId == "" || efsId == "None" {
		return "", fmt.Errorf("no EFS filesystem found in region %s", t.deployConfig.AWS.Region)
	}

	return efsId, nil
}

// deployMonitoringInfrastructure creates PVs for Static Provisioning
func (t *ThanosStack) deployMonitoringInfrastructure(ctx context.Context, config *MonitoringConfig) error {
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	// Clean up existing monitoring PVs and PVCs
	if err := t.cleanupExistingMonitoringStorage(ctx, config); err != nil {
		return fmt.Errorf("failed to cleanup existing monitoring storage: %w", err)
	}

	timestamp, err := t.getTimestampFromExistingPV(ctx, config.ChainName)
	if err != nil {
		return fmt.Errorf("failed to get timestamp from existing PV: %w", err)
	}

	// Create Prometheus PV and PVC
	prometheusPV := t.generateStaticPVManifest("prometheus", config, "20Gi", timestamp)
	if err := t.applyPVManifest(ctx, "prometheus", prometheusPV); err != nil {
		return fmt.Errorf("failed to create Prometheus PV: %w", err)
	}

	prometheusPVC := t.generateStaticPVCManifest("prometheus", config, "20Gi", timestamp)
	if err := t.applyPVCManifest(ctx, "prometheus", prometheusPVC); err != nil {
		return fmt.Errorf("failed to create Prometheus PVC: %w", err)
	}

	// Create Grafana PV and PVC
	grafanaPV := t.generateStaticPVManifest("grafana", config, "10Gi", timestamp)
	if err := t.applyPVManifest(ctx, "grafana", grafanaPV); err != nil {
		return fmt.Errorf("failed to create Grafana PV: %w", err)
	}

	grafanaPVC := t.generateStaticPVCManifest("grafana", config, "10Gi", timestamp)
	if err := t.applyPVCManifest(ctx, "grafana", grafanaPVC); err != nil {
		return fmt.Errorf("failed to create Grafana PVC: %w", err)
	}

	return nil
}

// cleanupExistingMonitoringStorage removes existing monitoring PVs and PVCs
func (t *ThanosStack) cleanupExistingMonitoringStorage(ctx context.Context, config *MonitoringConfig) error {
	fmt.Println("🧹 Cleaning up existing monitoring PVs and PVCs...")

	// Get existing monitoring PVCs
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to get existing PVCs: %v\n", err)
	} else {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		deletedPVCs := 0

		for _, line := range lines {
			pvcName := strings.TrimSpace(line)
			if pvcName == "" {
				continue
			}

			// Check if PVC is bound to a pod
			boundOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", pvcName, "-n", config.Namespace, "-o", "jsonpath={.status.phase}")
			if err == nil && strings.TrimSpace(boundOutput) == "Bound" {
				// Check if any pod is using this PVC
				podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", config.Namespace, "-o", "jsonpath={.items[*].spec.volumes[*].persistentVolumeClaim.claimName}")
				if err == nil && strings.Contains(podOutput, pvcName) {
					fmt.Printf("⚠️  Warning: PVC %s is in use by pods, skipping deletion\n", pvcName)
					continue
				}
			}

			// Delete PVC
			_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", pvcName, "-n", config.Namespace, "--ignore-not-found=true")
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete PVC %s: %v\n", pvcName, err)
			} else {
				deletedPVCs++
			}
		}

		if deletedPVCs > 0 {
			fmt.Printf("✅ Deleted %d existing PVCs\n", deletedPVCs)
		}
	}

	// Get existing monitoring PVs
	output, err = utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to get existing PVs: %v\n", err)
	} else {
		lines := strings.Split(strings.TrimSpace(output), "\n")
		deletedPVs := 0

		for _, line := range lines {
			parts := strings.Fields(line)
			if len(parts) < 2 {
				continue
			}

			pvName := parts[0]
			status := parts[1]

			// Only delete Released PVs (not Bound or Available)
			if status == "Released" && (strings.Contains(pvName, "thanos-stack-grafana") || strings.Contains(pvName, "thanos-stack-prometheus")) {
				// Remove claimRef to allow reuse
				_, err = utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", pvName, "-p", `{"spec":{"claimRef":null}}`, "--type=merge")
				if err != nil {
					fmt.Printf("⚠️  Warning: Failed to remove claimRef from PV %s: %v\n", pvName, err)
				} else {
					deletedPVs++
				}
			}
		}

		if deletedPVs > 0 {
			fmt.Printf("✅ Prepared %d existing PVs for reuse\n", deletedPVs)
		}
	}

	fmt.Println("✅ Monitoring storage cleanup completed")
	return nil
}

// ensureNamespaceExists checks if namespace exists and creates it if needed
func (t *ThanosStack) ensureNamespaceExists(ctx context.Context, namespace string) error {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	return nil
}

// getTimestampFromExistingPV extracts timestamp from existing monitoring PVs
func (t *ThanosStack) getTimestampFromExistingPV(ctx context.Context, chainName string) (string, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// Look for existing monitoring PVs (grafana or prometheus) that are Released
		if strings.Contains(line, chainName) &&
			(strings.Contains(line, "thanos-stack-grafana") || strings.Contains(line, "thanos-stack-prometheus")) {
			// Extract timestamp from PV name like "theo0707-900hi-thanos-stack-grafana"
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	// Fallback to op-geth PV if no monitoring PVs found
	for _, line := range lines {
		if strings.Contains(line, chainName) && strings.Contains(line, "thanos-stack-op-geth") {
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find existing PV to extract timestamp")
}

// generateStaticPVManifest generates PV manifest
func (t *ThanosStack) generateStaticPVManifest(component string, config *MonitoringConfig, size string, timestamp string) string {
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.ChainName, timestamp, component)
	volumeHandle := config.EFSFileSystemId

	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: %s
  labels:
    app: %s
spec:
  capacity:
    storage: %s
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: efs-sc
  csi:
    driver: efs.csi.aws.com
    volumeHandle: %s
`, pvName, pvName, size, volumeHandle)
}

// applyPVManifest applies PV manifest using kubectl
func (t *ThanosStack) applyPVManifest(ctx context.Context, component string, manifest string) error {
	tempFile := filepath.Join(t.deploymentPath, fmt.Sprintf("monitoring-%s-pv.yaml", component))
	if err := os.WriteFile(tempFile, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write PV manifest: %w", err)
	}
	defer os.Remove(tempFile)

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply PV manifest: %w", err)
	}

	return nil
}

// generateStaticPVCManifest generates PVC manifest
func (t *ThanosStack) generateStaticPVCManifest(component string, config *MonitoringConfig, size string, timestamp string) string {
	pvcName := fmt.Sprintf("%s-%s", config.HelmReleaseName, component)
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.ChainName, timestamp, component)

	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: %s
  selector:
    matchLabels:
      app: %s
  storageClassName: efs-sc
  volumeMode: Filesystem
  volumeName: %s
`, pvcName, config.Namespace, size, pvName, pvName)
}

// applyPVCManifest applies PVC manifest using kubectl
func (t *ThanosStack) applyPVCManifest(ctx context.Context, component string, manifest string) error {
	tempFile := filepath.Join(t.deploymentPath, fmt.Sprintf("monitoring-%s-pvc.yaml", component))
	if err := os.WriteFile(tempFile, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write PVC manifest: %w", err)
	}
	defer os.Remove(tempFile)

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to apply PVC manifest: %w", err)
	}

	return nil
}

// createDashboardConfigMaps creates ConfigMaps for Grafana dashboards
func (t *ThanosStack) createDashboardConfigMaps(ctx context.Context, config *MonitoringConfig) error {
	dashboardsPath := filepath.Join(config.ChartsPath, "dashboards")
	if _, err := os.Stat(dashboardsPath); os.IsNotExist(err) {
		return nil
	}

	files, err := os.ReadDir(dashboardsPath)
	if err != nil {
		return fmt.Errorf("failed to read dashboards directory: %w", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		dashboardPath := filepath.Join(dashboardsPath, file.Name())
		dashboardContent, err := os.ReadFile(dashboardPath)
		if err != nil {
			continue
		}

		configMapName := fmt.Sprintf("dashboard-%s", strings.TrimSuffix(file.Name(), ".json"))
		indentedContent := strings.ReplaceAll(string(dashboardContent), "\n", "\n    ")

		configMapYAML := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
  labels:
    grafana_dashboard: "1"
data:
  %s: |
    %s`, configMapName, config.Namespace, file.Name(), indentedContent)

		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dashboard-%s.yaml", configMapName))
		if err := os.WriteFile(tempFile, []byte(configMapYAML), 0644); err != nil {
			continue
		}

		utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
		os.Remove(tempFile)
	}
	return nil
}

// createAlertManagerSecret creates AlertManager configuration secret
func (t *ThanosStack) createAlertManagerSecret(ctx context.Context, config *MonitoringConfig) error {
	alertManagerYaml, err := t.generateAlertManagerSecretConfig(config)
	if err != nil {
		return fmt.Errorf("failed to generate AlertManager configuration: %w", err)
	}

	secretManifest := fmt.Sprintf(`apiVersion: v1
kind: Secret
metadata:
  name: alertmanager-config
  namespace: %s
  labels:
    app: alertmanager
    release: %s
type: Opaque
data:
  alertmanager.yaml: %s
`, config.Namespace, config.HelmReleaseName, alertManagerYaml)

	return t.applySecretManifest(ctx, secretManifest)
}

// generateAlertManagerSecretConfig generates AlertManager configuration YAML
func (t *ThanosStack) generateAlertManagerSecretConfig(config *MonitoringConfig) (string, error) {
	// Generate receivers configuration
	receivers := []map[string]interface{}{
		{
			"name": "telegram-critical",
		},
	}

	// Add Telegram configurations
	if config.AlertManager.Telegram.Enabled {
		telegramConfigs := []map[string]interface{}{}
		for _, receiver := range config.AlertManager.Telegram.CriticalReceivers {
			if receiver.ChatId != "" {
				// Convert chat_id to int64 for Prometheus Operator compatibility
				chatIdInt, err := strconv.ParseInt(receiver.ChatId, 10, 64)
				if err != nil {
					return "", fmt.Errorf("invalid chat_id format: %s", receiver.ChatId)
				}

				telegramConfigs = append(telegramConfigs, map[string]interface{}{
					"bot_token":  config.AlertManager.Telegram.ApiToken,
					"chat_id":    chatIdInt,
					"parse_mode": "Markdown",
					"message":    "🚨 *Critical Alert*\n\n*Alert:* {{.GroupLabels.alertname}}\n*Description:* {{range .Alerts}}{{.Annotations.description}}{{end}}",
				})
			}
		}
		if len(telegramConfigs) > 0 {
			receivers[0]["telegram_configs"] = telegramConfigs
		}
	}

	// Add Email configurations
	if config.AlertManager.Email.Enabled {
		emailConfigs := []map[string]interface{}{}
		for _, email := range config.AlertManager.Email.CriticalReceivers {
			if email != "" {
				emailConfigs = append(emailConfigs, map[string]interface{}{
					"to": email,
				})
			}
		}
		if len(emailConfigs) > 0 {
			receivers[0]["email_configs"] = emailConfigs
		}
	}

	alertManagerConfig := map[string]interface{}{
		"global": map[string]interface{}{
			"smtp_smarthost":     config.AlertManager.Email.SmtpSmarthost,
			"smtp_from":          config.AlertManager.Email.SmtpFrom,
			"smtp_auth_username": config.AlertManager.Email.SmtpAuthUsername,
			"smtp_auth_password": config.AlertManager.Email.SmtpAuthPassword,
		},
		"route": map[string]interface{}{
			"group_by":        []string{"alertname", "cluster", "service", "severity"},
			"group_wait":      "30s",
			"group_interval":  "5m",
			"repeat_interval": "4h",
			"receiver":        "telegram-critical",
		},
		"receivers": receivers,
	}

	yamlData, err := yaml.Marshal(alertManagerConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AlertManager config to YAML: %w", err)
	}

	return base64.StdEncoding.EncodeToString(yamlData), nil
}

// applySecretManifest applies a Kubernetes Secret manifest
func (t *ThanosStack) applySecretManifest(ctx context.Context, manifest string) error {
	tempFile, err := os.CreateTemp("", "alertmanager-secret-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(manifest); err != nil {
		return fmt.Errorf("failed to write manifest to file: %w", err)
	}
	tempFile.Close()

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile.Name()); err != nil {
		return fmt.Errorf("failed to apply secret manifest: %w", err)
	}

	return nil
}

// createPrometheusRule creates PrometheusRule for alerts
func (t *ThanosStack) createPrometheusRule(ctx context.Context, config *MonitoringConfig) error {
	// Clean up existing PrometheusRules except thanos-stack-alerts
	if err := t.cleanupExistingPrometheusRules(ctx, config); err != nil {
		return fmt.Errorf("failed to cleanup existing PrometheusRules: %w", err)
	}

	manifest := t.generatePrometheusRuleManifest(config)
	return t.applyPrometheusRuleManifest(ctx, manifest)
}

// cleanupExistingPrometheusRules removes all PrometheusRules except thanos-stack-alerts
func (t *ThanosStack) cleanupExistingPrometheusRules(ctx context.Context, config *MonitoringConfig) error {
	fmt.Println("🧹 Cleaning up existing PrometheusRules...")

	// Get all PrometheusRules in the monitoring namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", config.Namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		// If no PrometheusRules exist, that's fine
		return nil
	}

	if strings.TrimSpace(output) == "" {
		fmt.Println("✅ No existing PrometheusRules found")
		return nil
	}

	ruleNames := strings.Split(strings.TrimSpace(output), " ")

	for _, ruleName := range ruleNames {
		if ruleName == "" {
			continue
		}

		// Skip thanos-stack-alerts
		if strings.Contains(ruleName, "thanos-stack-alerts") {
			fmt.Printf("⏭️  Skipping thanos-stack-alerts: %s\n", ruleName)
			continue
		}

		fmt.Printf("🗑️  Deleting PrometheusRule: %s\n", ruleName)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "prometheusrule", ruleName, "-n", config.Namespace); err != nil {
			fmt.Printf("⚠️  Failed to delete PrometheusRule %s: %v\n", ruleName, err)
			// Continue with other rules even if one fails
		}
	}

	fmt.Println("✅ PrometheusRules cleanup completed")
	return nil
}

// generatePrometheusRuleManifest generates the complete PrometheusRule YAML manifest
func (t *ThanosStack) generatePrometheusRuleManifest(config *MonitoringConfig) string {
	manifest := fmt.Sprintf(`apiVersion: monitoring.coreos.com/v1
kind: PrometheusRule
metadata:
  name: %s-thanos-stack-alerts
  namespace: %s
  labels:
    app.kubernetes.io/name: monitoring
    app.kubernetes.io/instance: %s
    prometheus: kube-prometheus
    role: alert-rules
    release: %s
spec:
  groups:
  - name: thanos-stack.critical
    interval: 30s
    rules:
    - alert: OpNodeDown
      expr: absent(up{job="op-node"}) or up{job="op-node"} == 0
      for: 1m
      labels:
        severity: critical
        component: op-node
        chain_name: "%s"
      annotations:
        summary: "OP Node is down"
        description: "OP Node has been down for more than 1 minute"
    
    - alert: OpBatcherDown
      expr: absent(up{job="op-batcher"})
      for: 1m
      labels:
        severity: critical
        component: op-batcher
        chain_name: "%s"
      annotations:
        summary: "OP Batcher is down"
        description: "OP Batcher has been down for more than 1 minute"
    
    - alert: OpProposerDown
      expr: absent(up{job="op-proposer"})
      for: 1m
      labels:
        severity: critical
        component: op-proposer
        chain_name: "%s"
      annotations:
        summary: "OP Proposer is down"
        description: "OP Proposer has been down for more than 1 minute"
    
    - alert: OpGethDown
      expr: absent(up{job="op-geth"}) or up{job="op-geth"} == 0
      for: 1m
      labels:
        severity: critical
        component: op-geth
        chain_name: "%s"
      annotations:
        summary: "OP Geth is down"
        description: "OP Geth has been down for more than 1 minute"
    
    - alert: L1RpcDown
      expr: probe_success{job=~"blackbox-eth.*"} == 0
      for: 10s
      labels:
        severity: critical
        component: l1-rpc
        chain_name: "%s"
      annotations:
        summary: "L1 RPC connection failed"
        description: "L1 RPC endpoint {{ $labels.target }} is unreachable"
    
    - alert: OpBatcherBalanceCritical
      expr: op_batcher_balance < 0.01
      for: 10s
      labels:
        severity: critical
        component: op-batcher
        chain_name: "%s"
      annotations:
        summary: "OP Batcher ETH balance critically low"
        description: "OP Batcher balance is {{ $value }} ETH, below 0.01 ETH threshold"
    
    - alert: OpProposerBalanceCritical
      expr: op_proposer_balance < 0.01
      for: 10s
      labels:
        severity: critical
        component: op-proposer
        chain_name: "%s"
      annotations:
        summary: "OP Proposer ETH balance critically low"
        description: "OP Proposer balance is {{ $value }} ETH, below 0.01 ETH threshold"
    
    - alert: BlockProductionStalled
      expr: increase(geth_chain_head_block[5m]) == 0
      for: 2m
      labels:
        severity: critical
        component: op-geth
        chain_name: "%s"
      annotations:
        summary: "Block production has stalled"
        description: "No new blocks have been produced in the last 5 minutes"
    
    - alert: ContainerCpuUsageHigh
      expr: (sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > 80
      for: 5m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
      annotations:
        summary: "High CPU usage in Thanos Stack pod"
        description: "Pod {{ $labels.pod }} CPU usage is above 80%%"
    
    - alert: ContainerMemoryUsageHigh
      expr: (sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > 80
      for: 5m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
      annotations:
        summary: "High memory usage in Thanos Stack pod"
        description: "Pod {{ $labels.pod }} memory usage is above 80%%"
    
    - alert: PodCrashLooping
      expr: rate(kube_pod_container_status_restarts_total[5m]) > 0
      for: 1m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
      annotations:
        summary: "Pod is crash looping"
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} is restarting frequently"
`,
		config.HelmReleaseName,
		config.Namespace,
		config.HelmReleaseName,
		config.HelmReleaseName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName,
		config.ChainName)

	return manifest
}

// applyPrometheusRuleManifest applies the PrometheusRule manifest using kubectl
func (t *ThanosStack) applyPrometheusRuleManifest(ctx context.Context, manifest string) error {
	tempFile, err := os.CreateTemp("", "prometheus-rule-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.WriteString(manifest); err != nil {
		return fmt.Errorf("failed to write manifest to temp file: %w", err)
	}
	tempFile.Close()

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile.Name()); err != nil {
		return fmt.Errorf("failed to apply PrometheusRule manifest: %w", err)
	}

	return nil
}

// getTelegramConfigSummary returns a summary of Telegram configuration
func (t *ThanosStack) getTelegramConfigSummary(config types.TelegramConfig) string {
	if !config.Enabled {
		return "Disabled"
	}

	chatCount := len(config.CriticalReceivers)
	if chatCount == 0 {
		return "Enabled (no chat IDs configured)"
	}

	return fmt.Sprintf("Enabled (%d chat IDs configured)", chatCount)
}

// getEmailConfigSummary returns a summary of Email configuration
func (t *ThanosStack) getEmailConfigSummary(config types.EmailConfig) string {
	if !config.Enabled {
		return "Disabled"
	}

	receiverCount := len(config.CriticalReceivers)
	if receiverCount == 0 {
		return "Enabled (no receivers configured)"
	}

	return fmt.Sprintf("Enabled (%d receivers configured)", receiverCount)
}
