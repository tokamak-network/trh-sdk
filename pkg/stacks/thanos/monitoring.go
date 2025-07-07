package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
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
		AlertManager:      t.getDefaultAlertManagerConfig(),
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

// displayMonitoringInfo shows access information
func (t *ThanosStack) displayMonitoringInfo(ctx context.Context, config *MonitoringConfig) string {
	fmt.Println("\n🎉 Monitoring Stack Installation Complete!")
	fmt.Printf("📊 Grafana: http://localhost:3000 (port-forward)\n")
	fmt.Printf("   Username: admin\n")
	fmt.Printf("   Password: %s\n", config.AdminPassword)

	utils.ExecuteCommand(ctx, "kubectl", "port-forward", "-n", config.Namespace,
		fmt.Sprintf("svc/%s-grafana", config.HelmReleaseName), "3000:80")

	return "http://localhost:3000"
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
		"kube-prometheus-stack": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": t.generatePrometheusStorageSpec(config),
			},
			"grafana":      t.generateGrafanaStorageConfig(ctx, config),
			"alertmanager": t.generateAlertManagerConfig(config),
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

// generateAlertManagerConfig creates AlertManager configuration
func (t *ThanosStack) generateAlertManagerConfig(config *MonitoringConfig) map[string]interface{} {
	return map[string]interface{}{
		"enabled": true,
		"config": map[string]interface{}{
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
			"receivers": t.generateAlertManagerReceivers(config),
		},
	}
}

// generateAlertManagerReceivers generates receiver configurations
func (t *ThanosStack) generateAlertManagerReceivers(config *MonitoringConfig) []map[string]interface{} {
	receivers := []map[string]interface{}{
		{
			"name": "telegram-critical",
		},
	}

	if config.AlertManager.Telegram.Enabled {
		telegramConfigs := []map[string]interface{}{}
		for _, receiver := range config.AlertManager.Telegram.CriticalReceivers {
			if receiver.ChatId != "" {
				telegramConfigs = append(telegramConfigs, map[string]interface{}{
					"api_url":    fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.AlertManager.Telegram.ApiToken),
					"chat_id":    receiver.ChatId,
					"parse_mode": "Markdown",
					"message":    "🚨 *Critical Alert*\n\n*Alert:* {{.GroupLabels.alertname}}\n*Description:* {{range .Alerts}}{{.Annotations.description}}{{end}}",
				})
			}
		}
		if len(telegramConfigs) > 0 {
			receivers[0]["telegram_configs"] = telegramConfigs
		}
	}

	return receivers
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

// getTimestampFromExistingPV extracts timestamp from op-geth PV name
func (t *ThanosStack) getTimestampFromExistingPV(ctx context.Context, chainName string) (string, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, chainName) && strings.Contains(line, "thanos-stack-op-geth") {
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find existing op-geth PV to extract timestamp")
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
`, pvcName, config.Namespace, size, pvName)
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
  alertmanager.yml: %s
`, config.Namespace, config.HelmReleaseName, alertManagerYaml)

	return t.applySecretManifest(ctx, secretManifest)
}

// generateAlertManagerSecretConfig generates AlertManager configuration YAML
func (t *ThanosStack) generateAlertManagerSecretConfig(config *MonitoringConfig) (string, error) {
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
		"receivers": t.generateAlertManagerReceivers(config),
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
	manifest := t.generatePrometheusRuleManifest(config)
	return t.applyPrometheusRuleManifest(ctx, manifest)
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
