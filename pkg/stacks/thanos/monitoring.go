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

	// Ensure monitoring namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return "", fmt.Errorf("failed to ensure monitoring namespace exists: %w", err)
	}

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
	if err := t.createAlertManagerSecret(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create AlertManager secret: %w", err)
	}
	if err := t.createPrometheusRule(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create PrometheusRule: %w", err)
	}
	if err := t.createDashboardConfigMaps(ctx, config); err != nil {
		return "", fmt.Errorf("failed to create dashboard configmaps: %w", err)
	}

	return t.displayMonitoringInfo(ctx, config), nil
}

// GetMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) GetMonitoringConfig(ctx context.Context, adminPassword string, alertManagerConfig types.AlertManagerConfig) (*MonitoringConfig, error) {
	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")
	helmReleaseName := fmt.Sprintf("monitoring-%d", time.Now().Unix())

	chartsPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/monitoring", t.deploymentPath)
	if _, err := os.Stat(chartsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chart directory not found: %s", chartsPath)
	}

	serviceNames, err := t.getServiceNames(ctx, t.deployConfig.K8s.Namespace)
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
		AlertManager:      alertManagerConfig,
	}

	return config, nil
}

// UninstallMonitoring removes monitoring plugin
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
	fmt.Println("🧹 Monitoring plugin uninstalled successfully")
	return nil
}

// displayMonitoringInfo shows access information and checks ALB Ingress
func (t *ThanosStack) displayMonitoringInfo(ctx context.Context, config *MonitoringConfig) string {
	fmt.Println("\n🎉 Monitoring Plugin Installation Complete!")

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

// generateAlertTemplates generates common alert templates
func (t *ThanosStack) generateAlertTemplates(grafanaURL string) map[string]string {
	return map[string]string{
		"email_subject": "🚨 Critical Alert - {{ .GroupLabels.chain_name }}",
		"email_html": `<!DOCTYPE html>
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
        <div class="info"><strong>Alert:</strong> {{ .GroupLabels.alertname }}</div>
        <div class="info"><strong>Severity:</strong> {{ .GroupLabels.severity }}</div>
        <div class="info"><strong>Component:</strong> {{ .GroupLabels.component }}</div>
        <div class="info"><strong>Namespace:</strong> {{ .GroupLabels.namespace }}</div>
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
		"telegram_message": "🚨 Critical Alert - {{ .GroupLabels.chain_name }}\n\nAlert: {{ .GroupLabels.alertname }}\nSeverity: {{ .GroupLabels.severity }}\nComponent: {{ .GroupLabels.component }}\nNamespace: {{ .GroupLabels.namespace }}\n\nSummary: {{ .CommonAnnotations.summary }}\nDescription: {{ .CommonAnnotations.description }}\n\n⏰ Alert Time: {{ range .Alerts }}{{ .StartsAt }}{{ end }}\n\nDashboard: [View Details](" + grafanaURL + ")",
	}
}

// getServiceNames returns a map of component names to their Kubernetes service names
func (t *ThanosStack) getServiceNames(ctx context.Context, namespace string) (map[string]string, error) {
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
	fmt.Println("✅ Created Prometheus PV and PVC")

	// Create Grafana PV and PVC
	grafanaPV := t.generateStaticPVManifest("grafana", config, "10Gi", timestamp)
	if err := t.applyPVManifest(ctx, "grafana", grafanaPV); err != nil {
		return fmt.Errorf("failed to create Grafana PV: %w", err)
	}

	grafanaPVC := t.generateStaticPVCManifest("grafana", config, "10Gi", timestamp)
	if err := t.applyPVCManifest(ctx, "grafana", grafanaPVC); err != nil {
		return fmt.Errorf("failed to create Grafana PVC: %w", err)
	}
	fmt.Println("✅ Created Grafana PV and PVC")

	return nil
}

// cleanupExistingMonitoringStorage removes existing monitoring PVs and PVCs
func (t *ThanosStack) cleanupExistingMonitoringStorage(ctx context.Context, config *MonitoringConfig) error {
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
			fmt.Printf("✅ Prepared %d existing PVs\n", deletedPVs)
		}
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
			// Extract timestamp from PV
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
	grafanaURL := t.getGrafanaURL(ctx, config)
	alertManagerYaml, err := t.generateAlertManagerSecretConfig(config, grafanaURL)
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
func (t *ThanosStack) generateAlertManagerSecretConfig(config *MonitoringConfig, grafanaURL string) (string, error) {
	// Generate receivers configuration
	receivers := []map[string]interface{}{
		{
			"name": "telegram-critical",
		},
	}

	// Add null receiver for Watchdog
	receivers = append(receivers, map[string]interface{}{"name": "null"})

	// Add Email configurations
	if config.AlertManager.Email.Enabled {
		emailConfigs := []map[string]interface{}{}
		templates := t.generateAlertTemplates(grafanaURL)
		for _, email := range config.AlertManager.Email.CriticalReceivers {
			if email != "" {
				emailConfigs = append(emailConfigs, map[string]interface{}{
					"to": email,
					"headers": map[string]string{
						"subject": templates["email_subject"],
					},
					"html": templates["email_html"],
				})
			}
		}
		if len(emailConfigs) > 0 {
			receivers[0]["email_configs"] = emailConfigs
		}
	}

	// Add Telegram configurations
	if config.AlertManager.Telegram.Enabled {
		telegramConfigs := []map[string]interface{}{}
		templates := t.generateAlertTemplates(grafanaURL)
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
					"message":    templates["telegram_message"],
				})
			}
		}
		if len(telegramConfigs) > 0 {
			receivers[0]["telegram_configs"] = telegramConfigs
		}
	}

	// Only add global SMTP config if email is enabled
	alertManagerConfig := map[string]interface{}{
		"route": map[string]interface{}{
			"group_by":        []string{"alertname", "severity", "component", "chain_name", "namespace"},
			"group_wait":      "30s",
			"group_interval":  "5m",
			"repeat_interval": "4h",
			"receiver":        "telegram-critical",
			"routes": []map[string]interface{}{
				{
					"match":    map[string]string{"alertname": "Watchdog"},
					"receiver": "null",
				},
			},
		},
		"receivers": receivers,
	}

	// Add global SMTP config only if email is enabled
	if config.AlertManager.Email.Enabled {
		alertManagerConfig["global"] = map[string]interface{}{
			"smtp_smarthost":     config.AlertManager.Email.SmtpSmarthost,
			"smtp_from":          config.AlertManager.Email.SmtpFrom,
			"smtp_auth_username": config.AlertManager.Email.SmtpAuthUsername,
			"smtp_auth_password": config.AlertManager.Email.SmtpAuthPassword,
		}
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
	// Get all PrometheusRules in the monitoring namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", config.Namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		// If no PrometheusRules exist, that's fine
		return nil
	}

	if strings.TrimSpace(output) == "" {
		fmt.Println("✅ No existing Alerting Rules found")
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

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "prometheusrule", ruleName, "-n", config.Namespace); err != nil {
			fmt.Printf("⚠️  Failed to delete Alerting Rule %s: %v\n", ruleName, err)
			// Continue with other rules even if one fails
		}
	}

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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
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
        namespace: "%s"
      annotations:
        summary: "Pod is crash looping"
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} is restarting frequently"
`,
		config.HelmReleaseName,
		config.Namespace,
		config.HelmReleaseName,
		config.HelmReleaseName,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
		config.ChainName,
		config.Namespace,
	)

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

// getGrafanaURL returns the full Grafana dashboard URL using the ALB Ingress
func (t *ThanosStack) getGrafanaURL(ctx context.Context, config *MonitoringConfig) string {
	// Try to get the ALB Ingress hostname using the actual Helm release name
	ingressName := fmt.Sprintf("%s-grafana", config.HelmReleaseName)
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace, "-o", "jsonpath={.items[?(@.metadata.name==\""+ingressName+"\")].status.loadBalancer.ingress[0].hostname}")
	if err != nil || output == "" {
		// Fallback to using ExternalURL template variable for dynamic resolution
		return "{{ .ExternalURL }}/d/thanos-stack/thanos-stack-overview?orgId=1&refresh=30s"
	}
	return "http://" + output + "/d/thanos-stack/thanos-stack-overview?orgId=1&refresh=30s"
}
