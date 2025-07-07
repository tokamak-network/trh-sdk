package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	ResourceName      string
	// AlertManager configuration
	AlertManager AlertManagerConfig
}

// AlertManagerConfig holds alertmanager-specific configuration
type AlertManagerConfig struct {
	Telegram TelegramConfig
	Email    EmailConfig
}

// TelegramConfig holds Telegram notification configuration
type TelegramConfig struct {
	Enabled           bool
	ApiToken          string
	CriticalReceivers []TelegramReceiver
}

// TelegramReceiver represents a Telegram chat recipient
type TelegramReceiver struct {
	ChatId string
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled           bool
	SmtpSmarthost     string
	SmtpFrom          string
	SmtpAuthUsername  string
	SmtpAuthPassword  string
	DefaultReceivers  []string
	CriticalReceivers []string
}

const (
	monitoringNamespace = "monitoring"
)

// InstallMonitoring installs monitoring stack using Helm dependencies
func (t *ThanosStack) InstallMonitoring(ctx context.Context, config *MonitoringConfig) (string, error) {
	fmt.Println("🚀 Starting monitoring installation...")

	// Display AlertManager configuration summary (using predefined values from charts/monitoring/values.yaml)
	fmt.Println(t.GetAlertManagerConfigSummary(config))

	// Deploy Terraform infrastructure if persistent storage is enabled
	if config.EnablePersistence {
		fmt.Println("📦 Deploying persistent storage infrastructure...")
		if err := t.deployMonitoringInfrastructure(ctx, config); err != nil {
			return "", fmt.Errorf("failed to deploy monitoring infrastructure: %w", err)
		}
	}

	// Generate values file
	if err := t.generateValuesFile(ctx, config); err != nil {
		return "", fmt.Errorf("failed to generate values file: %w", err)
	}

	// Update chart dependencies
	fmt.Println("📦 Updating chart dependencies...")
	if _, err := utils.ExecuteCommand(ctx, "helm", "dependency", "update", config.ChartsPath); err != nil {
		return "", fmt.Errorf("failed to update chart dependencies: %w", err)
	}

	// Install monitoring stack with error monitoring
	fmt.Printf("⚙️  Installing monitoring stack '%s' in namespace '%s'...\n", config.HelmReleaseName, config.Namespace)
	installCmd := []string{
		"upgrade", "--install",
		config.HelmReleaseName,
		config.ChartsPath,
		"--values", config.ValuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
		"--timeout", "15m", // Extended timeout for complex monitoring stack
		"--wait",
		"--wait-for-jobs", // Wait for init jobs to complete
	}

	// Start error monitoring in background
	errorChan := make(chan error, 1)
	go t.monitorInstallationErrors(ctx, config, errorChan)

	if _, err := utils.ExecuteCommand(ctx, "helm", installCmd...); err != nil {
		// Installation failed, gather error information
		fmt.Println("\n❌ Installation failed! Gathering error information...")
		t.gatherInstallationErrors(ctx, config)
		return "", fmt.Errorf("failed to install monitoring stack: %w", err)
	}

	// Stop error monitoring
	close(errorChan)

	// Create AlertManager Secret after successful Helm installation
	if err := t.createAlertManagerSecret(ctx, config); err != nil {
		fmt.Printf("⚠️  Failed to create AlertManager configuration secret: %v\n", err)
		fmt.Println("   You can create it manually later using kubectl")
	}

	// Create dashboard ConfigMaps after successful Helm installation
	if err := t.createDashboardConfigMaps(ctx, config); err != nil {
		fmt.Printf("⚠️  Failed to create dashboard ConfigMaps: %v\n", err)
		fmt.Println("   Dashboards can be imported manually later")
	}

	// Display access information
	grafanaURL := t.displayMonitoringInfo(ctx, config)

	return grafanaURL, nil
}

// GetMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) GetMonitoringConfig(ctx context.Context, adminPassword string) (*MonitoringConfig, error) {
	// Use timestamped release name for monitoring
	resourceName := convertChainNameToResourceName(t.deployConfig.ChainName)
	timestamp := time.Now().Unix()
	helmReleaseName := fmt.Sprintf("%s-%d", monitoringNamespace, timestamp)

	// Get current working directory
	cwd := t.deploymentPath

	// Set charts path
	chartsPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/monitoring", cwd)
	if _, err := os.Stat(chartsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chart directory not found: %s", chartsPath)
	}

	// Get service names dynamically from trh-sdk configuration
	serviceNames, err := t.getServiceNames(ctx, t.deployConfig.K8s.Namespace, resourceName)
	if err != nil {
		return nil, fmt.Errorf("error getting service names: %w", err)
	}

	// Get EFS filesystem ID from existing op-geth PV
	efsFileSystemId, err := t.getEFSFileSystemId(ctx, resourceName)
	if err != nil {
		return nil, fmt.Errorf("error getting EFS filesystem ID: %w", err)
	}

	config := &MonitoringConfig{
		Namespace:         monitoringNamespace,
		HelmReleaseName:   helmReleaseName,
		AdminPassword:     adminPassword,
		L1RpcUrl:          t.deployConfig.L1RPCURL,
		ServiceNames:      serviceNames,
		EnablePersistence: true,
		EFSFileSystemId:   efsFileSystemId,
		ChartsPath:        chartsPath,
		ValuesFilePath:    "", // Will be set in generateValuesFile
		ResourceName:      resourceName,
		AlertManager:      t.getDefaultAlertManagerConfig(),
	}

	return config, nil
}

func (t *ThanosStack) UninstallMonitoring(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	// Find monitoring releases in the monitoring namespace
	releases, err := utils.FilterHelmReleases(ctx, monitoringNamespace, "monitoring")
	if err != nil {
		fmt.Println("Error to filter helm releases:", err)
		return err
	}

	if len(releases) == 0 {
		fmt.Println("No monitoring releases found")
		return nil
	}

	// Store release names for cleanup
	var releasesToCleanup []string

	for _, release := range releases {
		fmt.Printf("🗑️  Uninstalling monitoring release: %s\n", release)
		releasesToCleanup = append(releasesToCleanup, release)

		_, err = utils.ExecuteCommand(ctx, "helm", []string{
			"uninstall",
			release,
			"--namespace",
			monitoringNamespace,
		}...)
		if err != nil {
			fmt.Println("Error uninstalling monitoring helm chart:", err)
			return err
		}
	}

	// Clean up orphaned services in kube-system after Helm uninstall
	if len(releasesToCleanup) > 0 {
		if err := t.cleanupOrphanedKubeSystemServices(ctx, releasesToCleanup); err != nil {
			fmt.Printf("⚠️  Warning: Failed to cleanup orphaned services: %v\n", err)
			// Continue anyway - this is cleanup, not critical
		}
	}

	resourceName := convertChainNameToResourceName(t.deployConfig.ChainName)

	// Get timestamp from existing op-geth PV to match naming pattern
	timestamp, err := t.getTimestampFromExistingPV(ctx, resourceName)
	if err != nil {
		return fmt.Errorf("failed to get timestamp from existing PV: %w", err)
	}
	if err := t.cleanupExistingMonitoringResources(ctx, monitoringNamespace, resourceName, timestamp); err != nil {
		fmt.Printf("⚠️  Warning: Failed to cleanup existing resources: %v\n", err)
		// Continue anyway - we'll try to create new ones
	}

	// delete the namespace
	if err := t.tryToDeleteK8sNamespace(ctx, monitoringNamespace); err != nil {
		fmt.Printf("⚠️  Warning: Failed to delete namespace %s: %v\n", monitoringNamespace, err)
		// Continue anyway - this is cleanup, not critical
	}

	fmt.Println("✅ Uninstall monitoring component successfully")

	return nil
}

// displayMonitoringInfo shows access information for the monitoring stack
func (t *ThanosStack) displayMonitoringInfo(ctx context.Context, config *MonitoringConfig) string {
	fmt.Println("\n🎉 Monitoring Stack Installation Complete!")
	fmt.Println("==========================================")

	fmt.Printf("📊 **Grafana Dashboard Access:**\n")
	fmt.Printf("   • Username: admin\n")
	fmt.Printf("   • Password: %s\n", config.AdminPassword)
	fmt.Printf("   • Namespace: %s\n", config.Namespace)
	fmt.Printf("   • Release: %s\n\n", config.HelmReleaseName)

	// Wait for ALB ingress endpoint to be ready
	fmt.Println("🔗 **ALB Ingress Endpoint:**")
	grafanaURL := t.waitForIngressEndpoint(ctx, config)

	if grafanaURL != "" {
		fmt.Printf("   🌐 Grafana Web URL: %s \n", grafanaURL)
		fmt.Printf("   🎯 You can now access Grafana directly via the web!\n\n")
	} else {
		fmt.Printf("   ⚠️  ALB Ingress endpoint not ready within timeout\n")
		fmt.Printf("   🔧 Check status: kubectl get ingress -n %s -w\n\n", config.Namespace)
	}

	fmt.Printf("🔗 **Local Access Commands (Alternative):**\n")
	fmt.Printf("   # Port forward to access Grafana locally:\n")
	fmt.Printf("   kubectl port-forward -n %s svc/%s-grafana 3000:80\n", config.Namespace, config.HelmReleaseName)
	fmt.Printf("   # Then visit: http://localhost:3000\n\n")

	return grafanaURL
}

// generateValuesFile creates the values.yaml file for monitoring configuration
func (t *ThanosStack) generateValuesFile(ctx context.Context, config *MonitoringConfig) error {
	fmt.Println("📝 Generating monitoring values file...")

	// Create values configuration with only dynamically set values
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
			"chainName":   t.deployConfig.ChainName,
			"namespace":   t.deployConfig.K8s.Namespace,
			"releaseName": config.ResourceName,
		},
		"kube-prometheus-stack": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": t.generatePrometheusStorageSpec(config),
			},
			"grafana": t.generateGrafanaStorageConfig(ctx, config),
		},
	}

	// Generate YAML content
	yamlContent, err := yaml.Marshal(valuesConfig)
	if err != nil {
		return fmt.Errorf("error marshaling values to YAML: %w", err)
	}

	// Create terraform/thanos-stack directory if it doesn't exist
	configFileDir := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath)
	if err := os.MkdirAll(configFileDir, os.ModePerm); err != nil {
		return fmt.Errorf("error creating directory: %w", err)
	}

	// Write values file to terraform/thanos-stack directory
	valuesFilePath := filepath.Join(configFileDir, "monitoring-values.yaml")
	if err := os.WriteFile(valuesFilePath, yamlContent, 0644); err != nil {
		return fmt.Errorf("error writing values file: %w", err)
	}

	config.ValuesFilePath = valuesFilePath
	fmt.Printf("✅ Generated values file: %s\n", valuesFilePath)
	return nil
}

// generatePrometheusStorageSpec creates Prometheus storage specification
func (t *ThanosStack) generatePrometheusStorageSpec(config *MonitoringConfig) map[string]interface{} {
	spec := map[string]interface{}{}

	// Add storage configuration if persistence is enabled
	if config.EnablePersistence {
		// Static Provisioning: Disable volumeClaimTemplate since we create PVC manually
		// Prometheus will use the manually created PVC
		fmt.Println("📦 Using manually created PVC for Prometheus Static Provisioning")

		// Fix EFS permission issue: Run Prometheus as grafana user (472)
		// This ensures compatibility with EFS directories owned by grafana
		spec["securityContext"] = map[string]interface{}{
			"runAsUser":    472,
			"runAsGroup":   472,
			"runAsNonRoot": true,
			"fsGroup":      472,
			"seccompProfile": map[string]interface{}{
				"type": "RuntimeDefault",
			},
		}
	}

	return spec
}

// generateGrafanaStorageConfig creates Grafana storage configuration
func (t *ThanosStack) generateGrafanaStorageConfig(ctx context.Context, config *MonitoringConfig) map[string]interface{} {
	grafanaConfig := map[string]interface{}{
		"adminPassword": config.AdminPassword,
	}

	// Add storage configuration if persistence is enabled
	if config.EnablePersistence {
		// For Static Provisioning, use existingClaim instead of volumeName
		// This prevents Grafana from creating a new PVC
		timestamp, err := t.getTimestampFromExistingPV(ctx, config.ResourceName)
		if err != nil {
			fmt.Printf("⚠️  Warning: Could not get timestamp for Grafana PV naming: %v\n", err)
			timestamp = "static" // Fallback
		}

		// Use the PVC name that we created in createStaticPVs
		existingClaimName := fmt.Sprintf("%s-%s-thanos-stack-grafana", config.ResourceName, timestamp)

		persistenceConfig := map[string]interface{}{
			"enabled":       true,
			"existingClaim": existingClaimName, // Use existing PVC instead of creating new one
			// Remove storageClassName to prevent dynamic provisioning
			// Remove volumeName as we're using existingClaim
			"accessModes": []string{"ReadWriteMany"},
			"size":        "10Gi",
		}

		grafanaConfig["persistence"] = persistenceConfig
		fmt.Printf("📦 Grafana will use existing PVC: %s\n", existingClaimName)
	} else {
		grafanaConfig["persistence"] = map[string]interface{}{
			"enabled": false,
		}
	}

	return grafanaConfig
}

// getDefaultAlertManagerConfig returns AlertManager configuration from charts/monitoring/values.yaml
func (t *ThanosStack) getDefaultAlertManagerConfig() AlertManagerConfig {
	return AlertManagerConfig{
		Telegram: TelegramConfig{
			Enabled:  true,
			ApiToken: "7904495507:AAE54gXGoj5X7oLsQHk_xzMFdO1kkn4xME8",
			CriticalReceivers: []TelegramReceiver{
				{ChatId: "1266746900"},
			},
		},
		Email: EmailConfig{
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

// generateAlertManagerConfig creates AlertManager configuration for values file
// AlertManager configuration is now handled via dynamically generated Secret

// generateTelegramReceivers converts TelegramReceiver slice to map slice for YAML
func (t *ThanosStack) generateTelegramReceivers(receivers []TelegramReceiver) []map[string]interface{} {
	var result []map[string]interface{}
	for _, receiver := range receivers {
		if receiver.ChatId != "" { // Only include non-empty chat IDs
			result = append(result, map[string]interface{}{
				"chat_id": receiver.ChatId,
			})
		}
	}
	return result
}

// getServiceNames returns a map of component names to their Kubernetes service names
func (t *ThanosStack) getServiceNames(ctx context.Context, namespace, resourceName string) (map[string]string, error) {
	fmt.Printf("🔍 Discovering services in namespace: %s\n", namespace)

	// First, get all services in the namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "services", "-n", namespace, "-o", "custom-columns=NAME:.metadata.name,TYPE:.spec.type", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("failed to get services in namespace %s: %w", namespace, err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("no services found in namespace %s", namespace)
	}

	// Parse the service list
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var allServices []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			serviceName := fields[0]
			allServices = append(allServices, serviceName)
		}
	}

	// Component mapping patterns for OP Stack services (dashboard-compatible naming)
	componentPatterns := map[string][]string{
		"op-node": {
			"op-node", "node", "opnode",
			fmt.Sprintf("%s-op-node", resourceName),
			fmt.Sprintf("%s-node", resourceName),
			fmt.Sprintf("%s-thanos-stack-op-node", resourceName),
		},
		"op-batcher": {
			"op-batcher", "batcher", "opbatcher",
			fmt.Sprintf("%s-op-batcher", resourceName),
			fmt.Sprintf("%s-batcher", resourceName),
			fmt.Sprintf("%s-thanos-stack-op-batcher", resourceName),
		},
		"op-proposer": {
			"op-proposer", "proposer", "opproposer",
			fmt.Sprintf("%s-op-proposer", resourceName),
			fmt.Sprintf("%s-proposer", resourceName),
			fmt.Sprintf("%s-thanos-stack-op-proposer", resourceName),
		},
		"op-geth": {
			"op-geth", "geth", "opgeth", "l2geth",
			fmt.Sprintf("%s-op-geth", resourceName),
			fmt.Sprintf("%s-geth", resourceName),
			fmt.Sprintf("%s-thanos-stack-op-geth", resourceName),
		},
		"blockscout": {
			"blockscout", "explorer", "block-explorer",
			fmt.Sprintf("%s-blockscout", resourceName),
			fmt.Sprintf("%s-explorer", resourceName),
			fmt.Sprintf("%s-thanos-stack-blockscout", resourceName),
		},
		"block-explorer-frontend": {
			"block-explorer-frontend", "frontend", "explorer-frontend",
			fmt.Sprintf("%s-block-explorer-frontend", resourceName),
			fmt.Sprintf("%s-frontend", resourceName),
			fmt.Sprintf("%s-thanos-stack-block-explorer-frontend", resourceName),
		},
	}

	serviceNames := make(map[string]string)

	// Try to match services to components
	for component, patterns := range componentPatterns {
		var foundService string

		// First try exact matches
		for _, pattern := range patterns {
			for _, service := range allServices {
				if service == pattern {
					foundService = service
					break
				}
			}
			if foundService != "" {
				break
			}
		}

		// If no exact match, try substring matching
		if foundService == "" {
			for _, pattern := range patterns {
				for _, service := range allServices {
					if strings.Contains(strings.ToLower(service), strings.ToLower(pattern)) ||
						strings.Contains(strings.ToLower(pattern), strings.ToLower(service)) {
						foundService = service
						break
					}
				}
				if foundService != "" {
					break
				}
			}
		}

		if foundService != "" {
			serviceNames[component] = foundService
		} else {
			// Try with timestamped release name pattern for monitoring compatibility
			timestampedName := fmt.Sprintf("%s-thanos-stack-%s", resourceName, component)
			serviceNames[component] = timestampedName
		}
	}

	if len(serviceNames) == 0 {
		return nil, fmt.Errorf("no matching OP Stack services found in namespace %s", namespace)
	}

	return serviceNames, nil
}

// cleanupOrphanedKubeSystemServices removes orphaned services in kube-system left by monitoring releases
func (t *ThanosStack) cleanupOrphanedKubeSystemServices(ctx context.Context, releases []string) error {
	// Get all services in kube-system
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "svc", "-n", "kube-system", "-o", "name")
	if err != nil {
		return fmt.Errorf("failed to get services in kube-system: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		fmt.Println("✅ No services found in kube-system")
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var servicesToDelete []string

	// Find services that match any of the release names
	for _, line := range lines {
		serviceName := strings.TrimPrefix(line, "service/")

		// Check if this service belongs to any of our monitoring releases
		for _, release := range releases {
			if strings.Contains(serviceName, release) {
				servicesToDelete = append(servicesToDelete, serviceName)
				break
			}
		}
	}

	// Delete orphaned services
	if len(servicesToDelete) > 0 {
		for _, serviceName := range servicesToDelete {
			_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "svc", serviceName, "-n", "kube-system", "--ignore-not-found=true")
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete service %s: %v\n", serviceName, err)
			}
		}
	}

	if err := t.cleanupGenericMonitoringServices(ctx); err != nil {
		fmt.Printf("⚠️  Warning: Failed to cleanup generic monitoring services: %v\n", err)
	}

	return nil
}

// cleanupGenericMonitoringServices removes services with generic monitoring patterns
func (t *ThanosStack) cleanupGenericMonitoringServices(ctx context.Context) error {
	// Common patterns for monitoring services that might be left behind
	patterns := []string{
		"kubelet",
		"coredns",
		"kube-controller-manager",
		"kube-etcd",
		"kube-proxy",
		"kube-scheduler",
	}

	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "svc", "-n", "kube-system", "-o", "name")
	if err != nil {
		return fmt.Errorf("failed to get services in kube-system: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var servicesToDelete []string

	for _, line := range lines {
		serviceName := strings.TrimPrefix(line, "service/")

		// Check if this service contains "monitoring" and any of the patterns
		if strings.Contains(serviceName, "monitoring") {
			for _, pattern := range patterns {
				if strings.Contains(serviceName, pattern) {
					servicesToDelete = append(servicesToDelete, serviceName)
					break
				}
			}
		}
	}

	// Delete matching services
	if len(servicesToDelete) > 0 {
		for _, serviceName := range servicesToDelete {
			_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "svc", serviceName, "-n", "kube-system", "--ignore-not-found=true")
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete service %s: %v\n", serviceName, err)
			}
		}
	}

	return nil
}

// getEFSFileSystemId extracts EFS filesystem ID from existing PV
func (t *ThanosStack) getEFSFileSystemId(ctx context.Context, resourceName string) (string, error) {
	fmt.Println("🔍 Getting EFS filesystem ID from existing PV...")

	// Get all PVs and filter for op-geth
	pvListOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "name")
	if err != nil {
		return "", fmt.Errorf("failed to list PVs: %w", err)
	}

	var opGethPVName string
	lines := strings.Split(strings.TrimSpace(pvListOutput), "\n")
	for _, line := range lines {
		pvName := strings.TrimPrefix(line, "persistentvolume/")
		if strings.Contains(pvName, "thanos-stack-op-geth") && strings.Contains(pvName, resourceName) {
			opGethPVName = pvName
			break
		}
	}

	if opGethPVName == "" {
		return "", fmt.Errorf("no existing PV found for resource: %s", resourceName)
	}

	// Get volumeHandle from the specific PV
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", opGethPVName, "-o", "jsonpath={.spec.csi.volumeHandle}")
	if err != nil {
		return "", fmt.Errorf("failed to get volumeHandle from PV %s: %w", opGethPVName, err)
	}

	volumeHandle := strings.TrimSpace(output)
	if volumeHandle == "" {
		return "", fmt.Errorf("volumeHandle is empty for PV: %s", opGethPVName)
	}

	// Extract EFS filesystem ID (format: fs-xxxxxxxxx)
	if !strings.HasPrefix(volumeHandle, "fs-") {
		return "", fmt.Errorf("invalid EFS filesystem ID format: %s", volumeHandle)
	}

	fmt.Printf("✅ Found EFS filesystem ID: %s\n", volumeHandle)
	return volumeHandle, nil
}

// waitForIngressEndpoint waits for the ALB ingress endpoint to be ready
func (t *ThanosStack) waitForIngressEndpoint(ctx context.Context, config *MonitoringConfig) string {
	fmt.Println("⏳ Waiting for ALB Ingress endpoint to be provisioned...")
	fmt.Println("   (This may take 2-3 minutes for AWS ALB to be created)")

	maxRetries := 30
	retryInterval := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Check if ingress exists first
		ingressExists, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace, "-o", "name")
		if err != nil || strings.TrimSpace(ingressExists) == "" {
			fmt.Printf("   ⏳ [%d/%d] Ingress not found yet, waiting...\n", i+1, maxRetries)
			time.Sleep(retryInterval)
			continue
		}

		// Get ingress hostname
		hostname, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace,
			"-o", "jsonpath={.items[0].status.loadBalancer.ingress[0].hostname}")

		if err == nil && strings.TrimSpace(hostname) != "" {
			grafanaURL := fmt.Sprintf("http://%s", strings.TrimSpace(hostname))
			fmt.Printf("   ✅ ALB Ingress endpoint ready: %s\n", grafanaURL)

			return grafanaURL
		} else {
			fmt.Printf("   ⏳ [%d/%d] ALB provisioning in progress...\n", i+1, maxRetries)
		}

		time.Sleep(retryInterval)
	}

	fmt.Printf("   ⚠️  Timeout waiting for ALB Ingress endpoint (%d minutes)\n", (maxRetries*int(retryInterval.Seconds()))/60)
	fmt.Printf("   💡 ALB provisioning can take up to 5-10 minutes\n")
	fmt.Printf("   🔧 Check status manually: kubectl get ingress -n %s -w\n", config.Namespace)

	return ""
}

// monitorInstallationErrors monitors installation progress and reports errors in real-time
func (t *ThanosStack) monitorInstallationErrors(ctx context.Context, config *MonitoringConfig, errorChan chan error) {
	fmt.Println("🔍 Starting installation issue monitoring...")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-errorChan:
			// Installation completed or stopped
			return
		case <-ticker.C:
			// Check for pending pods with issues
			t.checkPendingPods(ctx, config)
		}
	}
}

// gatherInstallationErrors gathers comprehensive error information when installation fails
func (t *ThanosStack) gatherInstallationErrors(ctx context.Context, config *MonitoringConfig) {
	fmt.Println("\n🔍 Gathering detailed error information...")
	fmt.Println("=" + strings.Repeat("=", 50))

	// 1. Check Helm release status
	fmt.Println("\n📊 Helm Release Status:")
	if output, err := utils.ExecuteCommand(ctx, "helm", "status", config.HelmReleaseName, "-n", config.Namespace); err != nil {
		fmt.Printf("❌ Failed to get Helm status: %v\n", err)
	} else {
		fmt.Println(output)
	}

	// 2. Check failed pods with detailed information
	fmt.Println("\n🚨 Failed Pods Analysis:")
	t.analyzeFailedPods(ctx, config)

	// 3. Check recent events
	fmt.Println("\n📅 Recent Events (Last 10):")
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "events", "-n", config.Namespace,
		"--sort-by=.lastTimestamp", "--field-selector=type=Warning"); err != nil {
		fmt.Printf("❌ Failed to get events: %v\n", err)
	} else {
		fmt.Println(output)
	}

	// 4. Check resource quotas and limits
	fmt.Println("\n💾 Resource Status:")
	t.checkResourceStatus(ctx, config)

	// 5. Check storage issues
	fmt.Println("\n🗄️  Storage Issues:")
	t.checkStorageIssues(ctx, config)

	// 6. Provide troubleshooting commands
	fmt.Println("\n🛠️  Troubleshooting Commands:")
	t.provideTroubleshootingCommands(ctx, config)
}

// checkPendingPods checks for pending pods with issues
func (t *ThanosStack) checkPendingPods(ctx context.Context, config *MonitoringConfig) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", config.Namespace,
		"--field-selector=status.phase=Pending", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase,REASON:.status.conditions[0].reason")

	if err == nil && strings.TrimSpace(output) != "" && !strings.Contains(output, "No resources found") {
		fmt.Printf("⏳ Pending pods with issues:\n%s\n", output)
	}
}

// analyzeFailedPods provides detailed analysis of failed pods
func (t *ThanosStack) analyzeFailedPods(ctx context.Context, config *MonitoringConfig) {
	// Get all pods in the namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", config.Namespace, "-o", "wide")
	if err != nil {
		fmt.Printf("❌ Failed to get pods: %v\n", err)
		return
	}

	fmt.Println("All pods status:")
	fmt.Println(output)

	// Get detailed info for non-running pods
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		fmt.Println("No pod data available")
		return
	}

	for _, line := range lines[1:] { // Skip header
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			podName := fields[0]
			status := fields[2]

			if !strings.Contains(status, "Running") && !strings.Contains(status, "Completed") {
				fmt.Printf("\n🔍 Analyzing pod: %s (Status: %s)\n", podName, status)

				// Describe pod
				if descOutput, err := utils.ExecuteCommand(ctx, "kubectl", "describe", "pod", podName, "-n", config.Namespace); err == nil {
					// Extract events section
					descLines := strings.Split(descOutput, "\n")
					inEvents := false
					for _, descLine := range descLines {
						if strings.Contains(descLine, "Events:") {
							inEvents = true
						}
						if inEvents {
							fmt.Println(descLine)
						}
					}
				}

				// Get logs if available
				if logOutput, err := utils.ExecuteCommand(ctx, "kubectl", "logs", podName, "-n", config.Namespace, "--tail=10"); err == nil && logOutput != "" {
					fmt.Printf("Recent logs:\n%s\n", logOutput)
				}
			}
		}
	}
}

// checkResourceStatus checks resource quotas and node capacity
func (t *ThanosStack) checkResourceStatus(ctx context.Context, config *MonitoringConfig) {
	// Check resource quotas
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "resourcequota", "-n", config.Namespace); err == nil {
		fmt.Printf("Resource quotas:\n%s\n", output)
	}

	// Check node resources
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "top", "nodes"); err == nil {
		fmt.Printf("Node resource usage:\n%s\n", output)
	} else {
		fmt.Println("⚠️  Metrics server not available for resource usage")
	}
}

// checkStorageIssues checks for storage-related problems
func (t *ThanosStack) checkStorageIssues(ctx context.Context, config *MonitoringConfig) {
	// Check PVCs with detailed status
	fmt.Println("🗄️  Checking Persistent Volume Claims...")
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "-o", "wide"); err == nil {
		fmt.Printf("Persistent Volume Claims:\n%s\n", output)

		// Check for unbound PVCs
		if strings.Contains(output, "Pending") {
			fmt.Println("⚠️  Found pending PVCs - checking details...")

			// Get PVC details
			if pvcList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "-o", "name"); err == nil {
				pvcs := strings.Split(strings.TrimSpace(pvcList), "\n")
				for _, pvc := range pvcs {
					if pvc == "" {
						continue
					}
					pvcName := strings.TrimPrefix(pvc, "persistentvolumeclaim/")

					// Describe the PVC to get error details
					if descOutput, err := utils.ExecuteCommand(ctx, "kubectl", "describe", "pvc", pvcName, "-n", config.Namespace); err == nil {
						if strings.Contains(descOutput, "FailedBinding") || strings.Contains(descOutput, "no persistent volumes available") {
							fmt.Printf("❌ PVC %s binding failed:\n%s\n", pvcName, descOutput)
						}
					}
				}
			}
		}
	} else {
		fmt.Printf("❌ Failed to get PVCs: %v\n", err)
	}

	// Check storage classes
	fmt.Println("\n💾 Checking Storage Classes...")
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "storageclass", "-o", "wide"); err == nil {
		fmt.Printf("Available Storage Classes:\n%s\n", output)

		// Check specifically for EFS StorageClass
		if !strings.Contains(output, "efs-sc") {
			fmt.Println("⚠️  EFS StorageClass 'efs-sc' not found!")
			fmt.Println("💡 This is likely a Fargate environment where EFS CSI driver is not installed")
			fmt.Println("💡 The monitoring stack should use EmptyDir volumes instead")
		}
	} else {
		fmt.Printf("❌ Failed to get storage classes: %v\n", err)
	}

	// Check for EFS CSI driver
	fmt.Println("\n🔧 Checking EFS CSI Driver...")
	if output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "daemonset", "-n", "kube-system", "-l", "app=efs-csi-node"); err == nil && strings.TrimSpace(output) != "" {
		fmt.Printf("EFS CSI Driver status:\n%s\n", output)
	} else {
		fmt.Println("⚠️  EFS CSI Driver not found - this confirms Fargate environment")
		fmt.Println("💡 EmptyDir volumes will be used for storage")
	}
}

// provideTroubleshootingCommands provides useful commands for manual troubleshooting
func (t *ThanosStack) provideTroubleshootingCommands(_ context.Context, config *MonitoringConfig) {
	fmt.Printf(`
Manual Troubleshooting Commands:
================================

# Check all resources in namespace:
kubectl get all -n %s

# Watch pod status in real-time:
kubectl get pods -n %s -w

# Check events continuously:
kubectl get events -n %s -w

# Describe failed pods:
kubectl describe pods -n %s

# Check logs of specific pod:
kubectl logs <pod-name> -n %s -f

# Check Helm release history:
helm history %s -n %s

# Rollback if needed:
helm rollback %s -n %s

# Uninstall and retry:
helm uninstall %s -n %s
helm install %s %s --values %s -n %s --create-namespace

# Check cluster resources:
kubectl top nodes
kubectl get nodes -o wide

# Check for Fargate environment:
kubectl get nodes -o jsonpath='{.items[*].metadata.labels.eks\.amazonaws\.com/compute-type}'
kubectl get nodes -o jsonpath='{.items[*].metadata.name}'

# Check storage classes and EFS driver:
kubectl get storageclass
kubectl get daemonset -n kube-system -l app=efs-csi-node

# Check PVC binding issues:
kubectl get pvc -n %s -o wide
kubectl describe pvc -n %s

`, config.Namespace, config.Namespace, config.Namespace, config.Namespace,
		config.Namespace, config.HelmReleaseName, config.Namespace,
		config.HelmReleaseName, config.Namespace, config.HelmReleaseName,
		config.Namespace, config.ChartsPath,
		config.ValuesFilePath, config.Namespace, config.Namespace, config.Namespace)
}

// deployMonitoringInfrastructure creates PVs for Static Provisioning using existing efs-sc
func (t *ThanosStack) deployMonitoringInfrastructure(ctx context.Context, config *MonitoringConfig) error {
	// Create namespace if it doesn't exist
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	// Create PVs using kubectl and existing efs-sc StorageClass
	if err := t.createStaticPVs(ctx, config); err != nil {
		return fmt.Errorf("failed to create static PVs: %w", err)
	}

	return nil
}

// ensureNamespaceExists checks if namespace exists and creates it if needed
func (t *ThanosStack) ensureNamespaceExists(ctx context.Context, namespace string) error {
	fmt.Printf("🔍 Checking if namespace '%s' exists...\n", namespace)

	// Check if namespace exists
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	if strings.TrimSpace(output) == "" {
		// Namespace doesn't exist, create it
		fmt.Printf("📦 Creating namespace '%s'...\n", namespace)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
		fmt.Printf("✅ Namespace '%s' created successfully\n", namespace)
	} else {
		fmt.Printf("✅ Namespace '%s' already exists\n", namespace)
	}

	return nil
}

// createStaticPVs creates PersistentVolumes and PVCs for Static Provisioning with op-geth/op-node naming pattern
func (t *ThanosStack) createStaticPVs(ctx context.Context, config *MonitoringConfig) error {
	// Get timestamp from existing op-geth PV to match naming pattern
	timestamp, err := t.getTimestampFromExistingPV(ctx, config.ResourceName)
	if err != nil {
		return fmt.Errorf("failed to get timestamp from existing PV: %w", err)
	}

	// Clean up existing PVs and PVCs for monitoring components
	fmt.Println("🧹 Cleaning up existing monitoring PVs and PVCs...")
	if err := t.cleanupExistingMonitoringResources(ctx, config.Namespace, config.ResourceName, timestamp); err != nil {
		fmt.Printf("⚠️  Warning: Failed to cleanup existing resources: %v\n", err)
		// Continue anyway - we'll try to create new ones
	}

	// Wait a moment for cleanup to complete
	time.Sleep(5 * time.Second)

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

	// Verify PV/PVC binding
	if err := t.verifyPVCBinding(ctx, config, timestamp); err != nil {
		fmt.Printf("⚠️  Warning: PV/PVC binding verification failed: %v\n", err)
		// Continue anyway - binding might take some time
	}

	return nil
}

// cleanupExistingMonitoringResources removes existing monitoring PVs and PVCs
func (t *ThanosStack) cleanupExistingMonitoringResources(ctx context.Context, namespace string, resourceName string, timestamp string) error {
	components := []string{"prometheus", "grafana"}

	for _, component := range components {
		pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", resourceName, timestamp, component)

		// Delete PVC first (it might be bound to the PV)
		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", pvName, "-n", namespace, "--ignore-not-found=true")
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to delete PVC %s: %v\n", pvName, err)
		}

		// Wait a moment for PVC deletion to complete
		time.Sleep(2 * time.Second)

		// Delete PV (it might be in Released state)
		_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", pvName, "--ignore-not-found=true")
		if err != nil {
			fmt.Printf("⚠️  Warning: Failed to delete PV %s: %v\n", pvName, err)
		}

		// Also try to delete any PVs that might have old naming patterns
		t.cleanupOldPVPattern(ctx, component, resourceName)
	}

	return nil
}

// cleanupOldPVPattern removes PVs with old naming patterns that might conflict
func (t *ThanosStack) cleanupOldPVPattern(ctx context.Context, component, resourceName string) {
	// Get all PVs and find ones that match the component pattern
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase", "--no-headers")
	if err != nil {
		return
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pvName := fields[0]
		pvStatus := fields[1]

		// Check if this PV matches our component and is in Released state
		if strings.Contains(pvName, resourceName) &&
			strings.Contains(pvName, fmt.Sprintf("thanos-stack-%s", component)) &&
			(pvStatus == "Released" || pvStatus == "Available") {

			fmt.Printf("🗑️  Cleaning up old %s PV: %s (status: %s)\n", component, pvName, pvStatus)
			_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", pvName, "--ignore-not-found=true")
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to delete old PV %s: %v\n", pvName, err)
			}
		}
	}
}

// verifyPVCBinding checks if PVCs are properly bound to PVs
func (t *ThanosStack) verifyPVCBinding(ctx context.Context, config *MonitoringConfig, timestamp string) error {
	components := []string{"prometheus", "grafana"}

	for _, component := range components {
		pvcName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.ResourceName, timestamp, component)

		// Check PVC status with timeout
		maxRetries := 12 // 1 minute total (5 seconds * 12)
		for i := 0; i < maxRetries; i++ {
			output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", pvcName, "-n", config.Namespace, "-o", "jsonpath={.status.phase}")
			if err != nil {
				fmt.Printf("⚠️  Warning: Failed to check PVC %s status: %v\n", pvcName, err)
				break
			}

			status := strings.TrimSpace(output)

			if status == "Bound" {
				fmt.Printf("✅ %s PVC is bound successfully\n", component)
				break
			} else if status == "Pending" && i == maxRetries-1 {
				return fmt.Errorf("%s PVC is still pending after timeout", component)
			}

			if i < maxRetries-1 {
				time.Sleep(5 * time.Second)
			}
		}
	}

	return nil
}

// getTimestampFromExistingPV extracts timestamp from op-geth PV name
func (t *ThanosStack) getTimestampFromExistingPV(ctx context.Context, resourceName string) (string, error) {
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	// Look for op-geth PV pattern: resourceName-timestamp-thanos-stack-op-geth
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, resourceName) && strings.Contains(line, "thanos-stack-op-geth") {
			// E.g: Extract timestamp from: theo0624-1750743380-thanos-stack-op-geth
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil // Return timestamp part
			}
		}
	}

	return "", fmt.Errorf("could not find existing op-geth PV to extract timestamp")
}

// generateStaticPVManifest generates PV manifest for Static Provisioning with op-geth/op-node naming pattern
func (t *ThanosStack) generateStaticPVManifest(component string, config *MonitoringConfig, size string, timestamp string) string {
	// Use same naming pattern as op-geth/op-node: resourceName-timestamp-thanos-stack-component
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.ResourceName, timestamp, component)

	// Use the same volumeHandle format as op-geth/op-node (no subdirectory)
	// EFS will store all components in the same filesystem root
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

// generateStaticPVCManifest generates PVC manifest for Static Provisioning with selector
func (t *ThanosStack) generateStaticPVCManifest(component string, config *MonitoringConfig, size string, timestamp string) string {
	// Use same naming pattern as op-geth/op-node: resourceName-timestamp-thanos-stack-component
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.ResourceName, timestamp, component)
	pvcName := pvName // PVC name matches PV name for op-geth/op-node compatibility

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
	fmt.Println("📊 Creating dashboard ConfigMaps...")

	dashboardsPath := filepath.Join(config.ChartsPath, "dashboards")

	// Check if dashboards directory exists
	if _, err := os.Stat(dashboardsPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Dashboards directory not found: %s\n", dashboardsPath)
		return nil
	}

	// Read dashboard files
	files, err := os.ReadDir(dashboardsPath)
	if err != nil {
		return fmt.Errorf("failed to read dashboards directory: %w", err)
	}

	// Process each dashboard file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		dashboardPath := filepath.Join(dashboardsPath, file.Name())
		dashboardContent, err := os.ReadFile(dashboardPath)
		if err != nil {
			fmt.Printf("⚠️  Failed to read dashboard %s: %v\n", file.Name(), err)
			continue
		}

		// Create ConfigMap name from filename
		configMapName := fmt.Sprintf("dashboard-%s", strings.TrimSuffix(file.Name(), ".json"))

		// Indent dashboard content for YAML
		indentedContent := strings.ReplaceAll(string(dashboardContent), "\n", "\n    ")

		// Create ConfigMap YAML
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

		// Write to temporary file and apply
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dashboard-%s.yaml", configMapName))
		if err := os.WriteFile(tempFile, []byte(configMapYAML), 0644); err != nil {
			fmt.Printf("⚠️  Failed to write temp file for ConfigMap %s: %v\n", configMapName, err)
			continue
		}

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
			fmt.Printf("⚠️  Failed to create ConfigMap %s: %v\n", configMapName, err)
		}

		// Clean up temp file
		os.Remove(tempFile)
	}
	return nil
}

// NOTE: generateAdditionalScrapeConfigs function has been removed
// Scrape configuration is now handled entirely by Helm templates in charts/monitoring
// This eliminates the 3-way duplication between Go code, values.yaml, and templates

// UpdateTelegramConfig updates Telegram configuration for AlertManager
func (t *ThanosStack) UpdateTelegramConfig(config *MonitoringConfig, enabled bool, apiToken string, chatIds []string) {
	config.AlertManager.Telegram.Enabled = enabled
	config.AlertManager.Telegram.ApiToken = apiToken

	// Convert chat IDs to TelegramReceiver slice
	config.AlertManager.Telegram.CriticalReceivers = make([]TelegramReceiver, 0, len(chatIds))
	for _, chatId := range chatIds {
		if strings.TrimSpace(chatId) != "" {
			config.AlertManager.Telegram.CriticalReceivers = append(
				config.AlertManager.Telegram.CriticalReceivers,
				TelegramReceiver{ChatId: strings.TrimSpace(chatId)},
			)
		}
	}

	if enabled {
		fmt.Printf("✅ Telegram notifications enabled for %d recipients\n", len(config.AlertManager.Telegram.CriticalReceivers))
	} else {
		fmt.Println("❌ Telegram notifications disabled")
	}
}

// UpdateEmailConfig updates Email configuration for AlertManager
func (t *ThanosStack) UpdateEmailConfig(config *MonitoringConfig, enabled bool, smtpConfig EmailConfig) {
	config.AlertManager.Email.Enabled = enabled
	if enabled {
		config.AlertManager.Email.SmtpSmarthost = smtpConfig.SmtpSmarthost
		config.AlertManager.Email.SmtpFrom = smtpConfig.SmtpFrom
		config.AlertManager.Email.SmtpAuthUsername = smtpConfig.SmtpAuthUsername
		config.AlertManager.Email.SmtpAuthPassword = smtpConfig.SmtpAuthPassword
		config.AlertManager.Email.DefaultReceivers = smtpConfig.DefaultReceivers
		config.AlertManager.Email.CriticalReceivers = smtpConfig.CriticalReceivers

		fmt.Printf("✅ Email notifications enabled for %d default and %d critical recipients\n",
			len(config.AlertManager.Email.DefaultReceivers),
			len(config.AlertManager.Email.CriticalReceivers))
	} else {
		fmt.Println("❌ Email notifications disabled")
	}
}

// Alert Manager setup is now handled automatically with predefined values from values.yaml

// GetAlertManagerConfigSummary returns a summary of current AlertManager configuration
func (t *ThanosStack) GetAlertManagerConfigSummary(config *MonitoringConfig) string {
	var summary strings.Builder

	summary.WriteString("🔔 AlertManager Configuration Summary:\n")

	// Telegram status
	if config.AlertManager.Telegram.Enabled {
		summary.WriteString(fmt.Sprintf("  📱 Telegram: ✅ Enabled (%d recipients)\n",
			len(config.AlertManager.Telegram.CriticalReceivers)))
		if config.AlertManager.Telegram.ApiToken != "" {
			summary.WriteString("     Bot Token: Configured\n")
		}
	} else {
		summary.WriteString("  📱 Telegram: ❌ Disabled\n")
	}

	// Email status
	if config.AlertManager.Email.Enabled {
		summary.WriteString(fmt.Sprintf("  📧 Email: ✅ Enabled (%d default, %d critical recipients)\n",
			len(config.AlertManager.Email.DefaultReceivers),
			len(config.AlertManager.Email.CriticalReceivers)))
		if config.AlertManager.Email.SmtpFrom != "" {
			summary.WriteString(fmt.Sprintf("     From: %s\n", config.AlertManager.Email.SmtpFrom))
		}
	} else {
		summary.WriteString("  📧 Email: ❌ Disabled\n")
	}

	// Critical alerts info
	summary.WriteString("\n🚨 Critical Alerts Will Be Sent For:\n")
	summary.WriteString("  • OP Stack component failures (Node, Batcher, Proposer, Geth)\n")
	summary.WriteString("  • L1 RPC connection issues\n")
	summary.WriteString("  • Low ETH balances (< 0.01 ETH)\n")
	summary.WriteString("  • Block production stalls\n")
	summary.WriteString("  • System resource issues\n")

	return summary.String()
}

// Quick setup functions are no longer needed as configuration is managed automatically

// createAlertManagerSecret creates a Kubernetes Secret for AlertManager configuration
func (t *ThanosStack) createAlertManagerSecret(ctx context.Context, config *MonitoringConfig) error {
	fmt.Println("📝 Creating AlertManager configuration secret...")

	// Generate AlertManager configuration YAML
	alertManagerYaml, err := t.generateAlertManagerSecretConfig(config)
	if err != nil {
		return fmt.Errorf("failed to generate AlertManager configuration: %w", err)
	}

	// Create the Secret manifest
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

	// Apply the Secret
	if err := t.applySecretManifest(ctx, secretManifest); err != nil {
		return fmt.Errorf("failed to apply AlertManager secret: %w", err)
	}

	fmt.Println("✅ AlertManager configuration secret created successfully")
	return nil
}

// generateAlertManagerSecretConfig generates AlertManager configuration YAML
func (t *ThanosStack) generateAlertManagerSecretConfig(config *MonitoringConfig) (string, error) {
	// Generate AlertManager configuration
	alertManagerConfig := map[string]interface{}{
		"global": map[string]interface{}{
			"smtp_smarthost":     config.AlertManager.Email.SmtpSmarthost,
			"smtp_from":          config.AlertManager.Email.SmtpFrom,
			"smtp_auth_username": config.AlertManager.Email.SmtpAuthUsername,
			"smtp_auth_password": config.AlertManager.Email.SmtpAuthPassword,
			"smtp_auth_identity": config.AlertManager.Email.SmtpAuthUsername,
		},
		"route": map[string]interface{}{
			"group_by":        []string{"alertname", "cluster", "service", "severity"},
			"group_wait":      "30s",
			"group_interval":  "5m",
			"repeat_interval": "4h",
			"receiver":        "telegram-critical",
		},
		"inhibit_rules": []map[string]interface{}{
			{
				"source_match": map[string]string{
					"severity": "critical",
				},
				"target_match": map[string]string{
					"severity": "warning",
				},
				"equal": []string{"alertname", "instance"},
			},
		},
		"receivers": t.generateAlertManagerReceivers(config),
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(alertManagerConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal AlertManager config to YAML: %w", err)
	}

	// Base64 encode for Secret
	encoded := base64.StdEncoding.EncodeToString(yamlData)
	return encoded, nil
}

// generateAlertManagerReceivers generates receiver configurations for AlertManager
func (t *ThanosStack) generateAlertManagerReceivers(config *MonitoringConfig) []map[string]interface{} {
	receivers := []map[string]interface{}{
		{
			"name": "telegram-critical",
		},
	}

	// Add Telegram configuration if enabled
	if config.AlertManager.Telegram.Enabled {
		telegramConfigs := []map[string]interface{}{}
		for _, receiver := range config.AlertManager.Telegram.CriticalReceivers {
			if receiver.ChatId != "" {
				telegramConfigs = append(telegramConfigs, map[string]interface{}{
					"api_url":    fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", config.AlertManager.Telegram.ApiToken),
					"chat_id":    receiver.ChatId,
					"parse_mode": "Markdown",
					"message": `🚨 *Critical Alert - {{.GroupLabels.chain_name | title}} Chain*

*Alert:* {{.GroupLabels.alertname}}
*Severity:* {{.GroupLabels.severity | upper}}
*Component:* {{.GroupLabels.service}}
*Namespace:* {{.GroupLabels.namespace}}

*Description:* {{range .Alerts}}{{.Annotations.description}}{{end}}

*Time:* {{.GroupLabels.timestamp}}
*Dashboard:* [View Details]({{.ExternalURL}})
`,
				})
			}
		}
		if len(telegramConfigs) > 0 {
			receivers[0]["telegram_configs"] = telegramConfigs
		}
	}

	// Add Email configuration if enabled
	if config.AlertManager.Email.Enabled {
		emailConfigs := []map[string]interface{}{}
		for _, email := range config.AlertManager.Email.CriticalReceivers {
			if email != "" {
				emailConfigs = append(emailConfigs, map[string]interface{}{
					"to":      email,
					"from":    config.AlertManager.Email.SmtpFrom,
					"subject": "🚨 Critical Alert - {{.GroupLabels.chain_name | title}} Chain - {{.GroupLabels.alertname}}",
					"html": `<h2>🚨 Critical Alert - {{.GroupLabels.chain_name | title}} Chain</h2>
<table border="1" style="border-collapse: collapse;">
<tr><td><strong>Alert</strong></td><td>{{.GroupLabels.alertname}}</td></tr>
<tr><td><strong>Severity</strong></td><td>{{.GroupLabels.severity | upper}}</td></tr>
<tr><td><strong>Component</strong></td><td>{{.GroupLabels.service}}</td></tr>
<tr><td><strong>Namespace</strong></td><td>{{.GroupLabels.namespace}}</td></tr>
<tr><td><strong>Description</strong></td><td>{{range .Alerts}}{{.Annotations.description}}{{end}}</td></tr>
<tr><td><strong>Time</strong></td><td>{{.GroupLabels.timestamp}}</td></tr>
</table>
<br><a href="{{.ExternalURL}}">View Dashboard</a>`,
				})
			}
		}
		if len(emailConfigs) > 0 {
			receivers[0]["email_configs"] = emailConfigs
		}
	}

	return receivers
}

// applySecretManifest applies a Kubernetes Secret manifest
func (t *ThanosStack) applySecretManifest(ctx context.Context, manifest string) error {
	// Create temporary file for the manifest
	tempFile, err := os.CreateTemp("", "alertmanager-secret-*.yaml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// Write manifest to file
	if _, err := tempFile.WriteString(manifest); err != nil {
		return fmt.Errorf("failed to write manifest to file: %w", err)
	}
	tempFile.Close()

	// Apply the manifest
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile.Name()); err != nil {
		return fmt.Errorf("failed to apply secret manifest: %w", err)
	}

	return nil
}

// deleteAlertManagerSecret deletes the AlertManager configuration secret
func (t *ThanosStack) deleteAlertManagerSecret(ctx context.Context, namespace string) error {
	fmt.Println("🗑️  Deleting AlertManager configuration secret...")

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "secret", "alertmanager-config", "-n", namespace, "--ignore-not-found"); err != nil {
		return fmt.Errorf("failed to delete AlertManager secret: %w", err)
	}

	fmt.Println("✅ AlertManager configuration secret deleted successfully")
	return nil
}

func convertChainNameToResourceName(chainName string) string {
	return strings.ReplaceAll(strings.ToLower(chainName), " ", "-") // Match K8s naming convention
}
