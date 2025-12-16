package thanos

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Constants for Thanos Stack components
var (
	// CoreComponents defines the core Thanos Stack components
	CoreComponents = []string{"op-node", "op-geth", "op-batcher", "op-proposer"}
)

// getLogger returns a logger instance, creating a default one if nil
func (t *ThanosStack) getLogger() *zap.SugaredLogger {
	if t.logger == nil {
		return zap.NewExample().Sugar()
	}
	return t.logger
}

// InstallMonitoring installs monitoring plugin using Helm
func (t *ThanosStack) InstallMonitoring(ctx context.Context, config *types.MonitoringConfig) (*types.MonitoringInfo, error) {
	logger := t.getLogger()

	if t.deployConfig == nil || t.deployConfig.K8s == nil {
		logger.Warn("Deploy configuration is not initialized, skip monitoring installation")
		return nil, nil
	}

	logger.Info("🚀 Starting monitoring installation...")

	// Ensure monitoring namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		logger.Errorw("Failed to ensure monitoring namespace exists", "err", err)
		return nil, fmt.Errorf("failed to ensure monitoring namespace exists: %w", err)
	}

	// Check and cleanup old Grafana database before installation
	logger.Info("Checking for old Grafana database files from previous installations")
	if err := t.cleanupOldGrafanaDatabase(ctx); err != nil {
		logger.Warnw("Failed to cleanup old Grafana database", "err", err)
		// Continue installation even if cleanup fails
	}

	// Deploy infrastructure if persistence is enabled
	if config.EnablePersistence {
		logger.Info("Deploying monitoring infrastructure (persistence enabled)")
		if err := t.deployMonitoringInfrastructure(ctx, config); err != nil {
			logger.Errorw("Failed to deploy monitoring infrastructure", "err", err)
			return nil, fmt.Errorf("failed to deploy monitoring infrastructure: %w", err)
		}
	}

	// Generate values file
	logger.Info("Generating values file for monitoring plugin")
	if err := t.generateValuesFile(config); err != nil {
		logger.Errorw("Failed to generate values file", "err", err)
		return nil, fmt.Errorf("failed to generate values file: %w", err)
	}

	// Update chart dependencies
	logger.Info("Updating Helm chart dependencies")
	out, err := utils.ExecuteCommand(ctx, "helm", "dependency", "update", config.ChartsPath)
	if err != nil {
		logger.Errorw("Failed to update chart dependencies", "err", err, "output", out)
	}

	// Install monitoring plugin
	logger.Infow("Installing monitoring plugin via Helm", "release", config.HelmReleaseName)
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
	out, err = utils.ExecuteCommand(ctx, "helm", installCmd...)
	if err != nil {
		logger.Errorw("Failed to install monitoring plugin", "err", err, "output", out)
	}

	// Clean up existing resources before creating new ones
	logger.Info("Cleaning up existing dashboard ConfigMaps")
	if err := t.cleanupExistingDashboardConfigMaps(ctx, config); err != nil {
		logger.Errorw("Failed to cleanup existing dashboard ConfigMaps", "err", err)
		// Continue with installation even if cleanup fails
	}

	// Create additional resources
	logger.Info("Creating AlertManager secret")
	if err := t.createAlertManagerSecret(ctx, config); err != nil {
		logger.Errorw("Failed to create AlertManager secret", "err", err)
		return nil, fmt.Errorf("failed to create AlertManager secret: %w", err)
	}
	if err := t.createPrometheusRule(ctx, config); err != nil {
		logger.Errorw("Failed to create PrometheusRule", "err", err)
		return nil, fmt.Errorf("failed to create PrometheusRule: %w", err)
	}
	logger.Info("Creating dashboard configmaps")
	if err := t.createDashboardConfigMaps(ctx, config); err != nil {
		logger.Errorw("Failed to create dashboard configmaps", "err", err)
		return nil, fmt.Errorf("failed to create dashboard configmaps: %w", err)
	}

	// Install AWS CLI sidecar for log collection if logging is enabled
	if config.LoggingEnabled {
		logger.Info("Installing AWS CLI sidecar for log collection")
		ns := strings.TrimSpace(t.deployConfig.K8s.Namespace)
		if ns == "" {
			logger.Warn("K8s namespace is not set in deploy config. Continuing without log collection")
		} else if exists, err := utils.CheckNamespaceExists(ctx, ns); err != nil {
			logger.Errorw("Failed to check namespace existence", "err", err)
		} else if !exists {
			logger.Errorw("Namespace does not exist", "namespace", ns)
		} else {
			// Get logging configuration from deploy config
			var loggingConfig *types.LoggingConfig
			if t.deployConfig != nil && t.deployConfig.LoggingConfig != nil {
				loggingConfig = t.deployConfig.LoggingConfig
			} else {
				// Use default values if no logging config exists
				loggingConfig = &types.LoggingConfig{
					Enabled:             true,
					CloudWatchRetention: 30,
					CollectionInterval:  30,
				}
			}

			if err := t.installLogCollectionSidecarDeployment(ctx, ns, loggingConfig); err != nil {
				logger.Errorw("Failed to install AWS CLI sidecar", "err", err)
				// Continue with installation even if log collection fails
				logger.Warn("Continuing without log collection")
			}
		}
	}

	monitoringInfo := t.createMonitoringInfo(ctx, config)
	if monitoringInfo == nil {
		logger.Error("ALB Ingress is not ready after installation")
		return nil, fmt.Errorf("ALB Ingress is not ready after installation")
	}

	// Check CloudWatch Log Groups status if logging is enabled
	if config.LoggingEnabled {
		logger.Info("Checking CloudWatch Log Groups status")
		ns := strings.TrimSpace(t.deployConfig.K8s.Namespace)
		if ns == "" {
			logger.Errorw("K8s namespace is not set in deploy config")
		} else if err := t.checkCloudWatchLogGroupsStatus(ctx, ns); err != nil {
			logger.Errorw("Failed to check CloudWatch Log Groups status", "err", err)
		}
	}

	logger.Info("Monitoring installation completed successfully")
	return monitoringInfo, nil
}

// GetMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) GetMonitoringConfig(ctx context.Context, adminPassword string, alertManagerConfig types.AlertManagerConfig, loggingEnabled bool) (*types.MonitoringConfig, error) {
	// Remove trailing % character from admin password if present
	adminPassword = strings.TrimSuffix(adminPassword, "%")

	if t.deployConfig == nil {
		return nil, fmt.Errorf("deploy configuration is not initialized")
	}

	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")
	helmReleaseName := fmt.Sprintf("monitoring-%d", time.Now().Unix())

	chartsPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/monitoring", t.deploymentPath)
	if _, err := os.Stat(chartsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chart directory not found: %s", chartsPath)
	}

	if t.deployConfig.K8s == nil {
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	serviceNames, err := t.getServiceNames(ctx, t.deployConfig.K8s.Namespace)
	if err != nil {
		return nil, fmt.Errorf("error getting service names: %w", err)
	}

	efsFileSystemId, err := t.getEFSFileSystemId(ctx, chainName)
	if err != nil {
		return nil, fmt.Errorf("error getting EFS filesystem ID: %w", err)
	}

	config := &types.MonitoringConfig{
		Namespace:         constants.MonitoringNamespace,
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
		LoggingEnabled:    loggingEnabled,
	}

	return config, nil
}

// UninstallMonitoring removes monitoring plugin
func (t *ThanosStack) UninstallMonitoring(ctx context.Context) error {
	logger := t.getLogger()
	monitoringNamespace := constants.MonitoringNamespace

	// Check if monitoring namespace exists first
	exists, err := utils.CheckNamespaceExists(ctx, monitoringNamespace)
	if err != nil {
		logger.Errorw("Failed to check monitoring namespace existence", "err", err)
		return err
	}

	if !exists {
		// Monitoring namespace doesn't exist, skip uninstallation silently
		return nil
	}

	logger.Info("Starting monitoring uninstallation...")

	if t.deployConfig == nil || t.deployConfig.K8s == nil {
		logger.Warn("Deploy configuration is not initialized, skip monitoring uninstallation")
		return nil
	}

	// Early return if deploy config is not available
	if t.deployConfig == nil {
		logger.Warnw("DeployConfig is nil, skipping namespace-specific cleanup")
		return nil
	}
	if t.deployConfig.K8s == nil {
		logger.Warnw("K8s config is nil, skipping namespace-specific cleanup")
		return nil
	}

	// Use namespace from deploy config; skip sidecar cleanup if invalid
	ns := strings.TrimSpace(t.deployConfig.K8s.Namespace)
	if ns == "" {
		logger.Warnw("K8s namespace is not set in deploy config")
		return nil
	}

	exists, err = utils.CheckNamespaceExists(ctx, ns)
	if err != nil {
		logger.Errorw("Failed to check namespace existence", "err", err)
		return nil
	}
	if !exists {
		return nil
	}

	// Clean up Sidecar deployments
	logger.Info("Cleaning up Sidecar deployments")
	if err := t.cleanupSidecarDeployments(ctx, ns); err != nil {
		logger.Warnw("Failed to cleanup Sidecar deployments", "err", err)
	}

	// Clean up RBAC resources
	logger.Info("Cleaning up RBAC resources")
	if err := t.cleanupRBACResources(ctx); err != nil {
		logger.Warnw("Failed to cleanup RBAC resources", "err", err)
	}

	// Clean up Grafana database BEFORE helm uninstall
	// This must run before helm deletes the Grafana pod, because we need the pod
	// to execute commands inside it to delete the database files from EFS
	logger.Info("Cleaning up Grafana database files")
	if err := t.cleanupGrafanaDatabaseFiles(ctx); err != nil {
		logger.Warnw("Failed to cleanup Grafana database files", "err", err)
		// Don't fail the uninstall if cleanup fails
	}

	releases, err := utils.FilterHelmReleases(ctx, monitoringNamespace, constants.MonitoringNamespace)
	if err != nil {
		logger.Errorw("Failed to filter Helm releases", "err", err)
		return err
	}

	for _, release := range releases {
		logger.Infow("Uninstalling Helm release", "release", release, "namespace", monitoringNamespace)
		out, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", monitoringNamespace)
		logger.Infow("Helm uninstall output", "output", out, "release", release, "namespace", monitoringNamespace)
		if err != nil {
			logger.Errorw("Failed to uninstall Helm release", "err", err, "release", release, "namespace", monitoringNamespace)
			return err
		}
	}

	// Delete monitoring namespace with timeout and forced cleanup
	logger.Infow("Deleting monitoring namespace", "namespace", monitoringNamespace)
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteMonitoringNamespace(ctxTimeout, monitoringNamespace)
	if err != nil {
		logger.Errorw("Failed to delete monitoring namespace", "err", err, "namespace", monitoringNamespace)
		return err
	}
	logger.Info("🧹 Monitoring plugin uninstalled successfully")
	return nil
}

// cleanupSidecarDeployments cleans up Sidecar deployments
func (t *ThanosStack) cleanupSidecarDeployments(ctx context.Context, namespace string) error {
	logger := t.getLogger()

	// List of Sidecar deployments to delete
	sidecarDeployments := []string{
		"thanos-logs-sidecar",
	}

	// Clean up deployments
	for _, deployment := range sidecarDeployments {
		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "deployment", deployment, "-n", namespace, "--ignore-not-found=true")
		if err != nil {
			logger.Warnw("Failed to delete Sidecar deployment", "deployment", deployment, "err", err)
		}
	}

	// Delete ConfigMap (if exists)
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "configmap", "thanos-logs-sidecar-config", "-n", namespace, "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete logs sidecar ConfigMap", "err", err)
	}

	logger.Info("Sidecar deployments cleanup completed")
	return nil
}

// cleanupRBACResources cleans up RBAC resources
func (t *ThanosStack) cleanupRBACResources(ctx context.Context) error {
	logger := t.getLogger()

	// Delete RBAC resources silently
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "clusterrolebinding", "thanos-logs-sidecar-binding", "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete ClusterRoleBinding", "err", err)
	}

	_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "clusterrole", "thanos-logs-sidecar-role", "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete ClusterRole", "err", err)
	}

	// Delete ServiceAccount from the deploy-config namespace only
	ns := ""
	if t.deployConfig != nil && t.deployConfig.K8s != nil {
		ns = strings.TrimSpace(t.deployConfig.K8s.Namespace)
	}
	if ns == "" {
		logger.Errorw("K8s namespace is not set in deploy config")
		return nil
	}
	_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "serviceaccount", "thanos-logs-sidecar", "-n", ns, "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete ServiceAccount", "namespace", ns, "err", err)
	}

	return nil
}

// tryToDeleteMonitoringNamespace attempts to delete monitoring namespace with forced cleanup if stuck in Terminating state
func (t *ThanosStack) tryToDeleteMonitoringNamespace(ctx context.Context, namespace string) error {
	logger := t.getLogger()

	if namespace == "" {
		logger.Warn("Monitoring namespace is empty, skipping namespace deletion")
		return nil
	}

	// First attempt normal deletion
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Initial namespace deletion failed", "err", err)
	}

	// Wait a bit for normal deletion to complete
	time.Sleep(10 * time.Second)

	// Check if namespace still exists
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "-o", "json")
	if err != nil {
		// Namespace doesn't exist, deletion successful
		logger.Infow("Monitoring namespace deleted successfully", "namespace", namespace)
		return nil
	}

	// Parse namespace status
	var namespaceStatus struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			Finalizers []string `json:"finalizers"`
		} `json:"spec"`
		Status struct {
			Phase string `json:"phase"`
		} `json:"status"`
	}

	if err := json.Unmarshal([]byte(output), &namespaceStatus); err != nil {
		logger.Errorw("Error unmarshalling monitoring namespace status", "err", err)
		return err
	}

	if namespaceStatus.Status.Phase == "Terminating" {
		logger.Warnw("Monitoring namespace is stuck in Terminating state, forcing cleanup", "namespace", namespace)

		// Remove finalizers to force completion
		namespaceStatus.Spec.Finalizers = make([]string, 0)

		// Write to temporary file
		tmpFile, err := os.Create("/tmp/monitoring-namespace.json")
		if err != nil {
			logger.Errorw("Error creating temporary file for monitoring namespace", "err", err)
			return err
		}
		defer func() {
			tmpFile.Close()
			os.Remove("/tmp/monitoring-namespace.json")
		}()

		encoder := json.NewEncoder(tmpFile)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(namespaceStatus)
		if err != nil {
			logger.Errorw("Error encoding monitoring namespace", "err", err)
			return err
		}

		// Close file before kubectl command
		tmpFile.Close()

		// Apply the changes to force finalize
		_, err = utils.ExecuteCommand(ctx, "kubectl", "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", "/tmp/monitoring-namespace.json")
		if err != nil {
			logger.Errorw("Error applying changes to monitoring namespace", "err", err)
			return err
		}

		logger.Infow("Successfully forced monitoring namespace cleanup", "namespace", namespace)
	}

	return nil
}

// DisplayMonitoringInfo displays monitoring information
func (t *ThanosStack) DisplayMonitoringInfo(monitoringInfo *types.MonitoringInfo) {
	logger := t.getLogger()
	logger.Infow("Monitoring Info", "info", monitoringInfo)
	fmt.Printf("\n🎉 Monitoring Installation Complete!\n")
	fmt.Printf("🌐 Grafana URL: %s\n", monitoringInfo.GrafanaURL)
	fmt.Printf("👤 Username: %s\n", monitoringInfo.Username)
	fmt.Printf("🔑 Password: %s\n", monitoringInfo.Password)
	fmt.Printf("📁 Namespace: %s\n", monitoringInfo.Namespace)
	fmt.Printf("🔗 Release Name: %s\n", monitoringInfo.ReleaseName)
	fmt.Printf("⛓️  Chain Name: %s\n", monitoringInfo.ChainName)
}

// createMonitoringInfo creates MonitoringInfo by checking ALB Ingress status
func (t *ThanosStack) createMonitoringInfo(ctx context.Context, config *types.MonitoringConfig) *types.MonitoringInfo {
	// Check ALB Ingress status and get URL
	albURL := t.checkALBIngressStatus(ctx, config)
	monitoringInfo := &types.MonitoringInfo{
		Username:     "admin",
		Password:     config.AdminPassword,
		Namespace:    config.Namespace,
		ReleaseName:  config.HelmReleaseName,
		ChainName:    config.ChainName,
		AlertManager: config.AlertManager,
	}

	if albURL != "" {
		monitoringInfo.GrafanaURL = albURL
		return monitoringInfo
	} else {
		// ALB Ingress is not ready, return error
		return nil
	}
}

// checkALBIngressStatus checks ALB Ingress status and returns the URL
func (t *ThanosStack) checkALBIngressStatus(ctx context.Context, config *types.MonitoringConfig) string {
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
	return ""
}

// generateValuesFile creates the values.yaml file
func (t *ThanosStack) generateValuesFile(config *types.MonitoringConfig) error {
	if t.deployConfig == nil || t.deployConfig.AWS == nil {
		return fmt.Errorf("deploy configuration is not properly initialized")
	}

	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

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
			"grafana": t.generateGrafanaStorageConfig(config),
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

	// Add CloudWatch Logs configuration when logging is enabled
	if config.LoggingEnabled {
		// Use namespace from deploy config
		ns := strings.TrimSpace(t.deployConfig.K8s.Namespace)
		if ns == "" {
			return fmt.Errorf("k8s namespace is not set in deploy config")
		}

		// Define log groups for different components using validated namespace
		logGroups := []map[string]interface{}{
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-node", ns),
				"retention":   30,
				"description": "OP Node logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-batcher", ns),
				"retention":   30,
				"description": "OP Batcher logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-proposer", ns),
				"retention":   30,
				"description": "OP Proposer logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-geth", ns),
				"retention":   30,
				"description": "OP Geth logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/blockscout", ns),
				"retention":   30,
				"description": "BlockScout logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/application", ns),
				"retention":   30,
				"description": "General application logs",
			},
		}

		valuesConfig["cloudwatch-logs"] = map[string]interface{}{
			"enabled":   true,
			"region":    t.deployConfig.AWS.Region,
			"logGroups": logGroups,
		}

		// Add CloudWatch datasource to Grafana with AWS credentials
		grafanaConfig := valuesConfig["kube-prometheus-stack"].(map[string]interface{})["grafana"].(map[string]interface{})
		grafanaConfig["datasources"] = map[string]interface{}{
			"datasources.yaml": map[string]interface{}{
				"apiVersion": 1,
				"datasources": []map[string]interface{}{
					{
						"name":      "CloudWatch",
						"type":      "cloudwatch",
						"access":    "proxy",
						"isDefault": false,
						"jsonData": map[string]interface{}{
							"authType":      "keys",
							"defaultRegion": t.deployConfig.AWS.Region,
						},
						"secureJsonData": map[string]interface{}{
							"accessKey": t.deployConfig.AWS.AccessKey,
							"secretKey": t.deployConfig.AWS.SecretKey,
						},
					},
				},
			},
		}
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

// createSidecarDeployments creates a unified sidecar deployment for all Thanos Stack components
func (t *ThanosStack) createSidecarDeployments(ctx context.Context, namespace string, loggingConfig *types.LoggingConfig) error {
	logger := t.getLogger()
	// Always collect logs from all core components
	componentMap := map[string]bool{
		"op-node":     true,
		"op-geth":     true,
		"op-batcher":  true,
		"op-proposer": true,
	}

	// Create unified sidecar deployment for all enabled components
	unifiedSidecar := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: thanos-logs-sidecar
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: thanos-logs-sidecar
  template:
    metadata:
      labels:
        app: thanos-logs-sidecar
    spec:
      serviceAccountName: thanos-logs-sidecar
      containers:
      - name: log-collector
        image: amazon/aws-cli:latest
        command:
        - /bin/bash
        - -c
        - |
          NAMESPACE="%s"
          curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
          chmod +x kubectl
          mv kubectl /usr/local/bin/
          
          # Create log streams once at startup
          aws logs create-log-stream --log-group-name "/aws/eks/$NAMESPACE/op-node" --log-stream-name "sidecar-collection" --region $AWS_REGION 2>/dev/null || true
          aws logs create-log-stream --log-group-name "/aws/eks/$NAMESPACE/op-geth" --log-stream-name "sidecar-collection" --region $AWS_REGION 2>/dev/null || true
          aws logs create-log-stream --log-group-name "/aws/eks/$NAMESPACE/op-batcher" --log-stream-name "sidecar-collection" --region $AWS_REGION 2>/dev/null || true
          aws logs create-log-stream --log-group-name "/aws/eks/$NAMESPACE/op-proposer" --log-stream-name "sidecar-collection" --region $AWS_REGION 2>/dev/null || true
          
          while true; do
            # Collect op-node logs if enabled
            if [ "%s" = "true" ]; then
              NODE_POD=$(kubectl get pods -n $NAMESPACE --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-node" | head -1)
              if [ ! -z "$NODE_POD" ]; then
                kubectl logs -n $NAMESPACE $NODE_POD --tail=50 --since=1m | while read line; do
                  if [ ! -z "$line" ]; then
                    CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                    aws logs put-log-events --log-group-name "/aws/eks/$NAMESPACE/op-node" --log-stream-name "sidecar-collection" --log-events '[{"timestamp":'$(date +%%s)'000,"message":"'"$CLEAN_MESSAGE"'"}]' --region $AWS_REGION || true
                  fi
                done
              fi
            fi
            
            # Collect op-geth logs if enabled
            if [ "%s" = "true" ]; then
              GETH_POD=$(kubectl get pods -n $NAMESPACE --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-geth" | head -1)
              if [ ! -z "$GETH_POD" ]; then
                kubectl logs -n $NAMESPACE $GETH_POD --tail=50 --since=1m | while read line; do
                  if [ ! -z "$line" ]; then
                    CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                    aws logs put-log-events --log-group-name "/aws/eks/$NAMESPACE/op-geth" --log-stream-name "sidecar-collection" --log-events '[{"timestamp":'$(date +%%s)'000,"message":"'"$CLEAN_MESSAGE"'"}]' --region $AWS_REGION || true
                  fi
                done
              fi
            fi
            
            # Collect op-batcher logs if enabled
            if [ "%s" = "true" ]; then
              BATCHER_PODS=$(kubectl get pods -n $NAMESPACE --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-batcher")
              if [ ! -z "$BATCHER_PODS" ]; then
                echo "$BATCHER_PODS" | while read BATCHER_POD; do
                  if [ ! -z "$BATCHER_POD" ]; then
                    kubectl logs -n $NAMESPACE $BATCHER_POD --tail=50 --since=1m | while read line; do
                      if [ ! -z "$line" ]; then
                        CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                        aws logs put-log-events --log-group-name "/aws/eks/$NAMESPACE/op-batcher" --log-stream-name "sidecar-collection" --log-events '[{"timestamp":'$(date +%%s)'000,"message":"'"$CLEAN_MESSAGE"'"}]' --region $AWS_REGION || true
                      fi
                    done
                  fi
                done
              fi
            fi
            
            # Collect op-proposer logs if enabled
            if [ "%s" = "true" ]; then
              PROPOSER_PODS=$(kubectl get pods -n $NAMESPACE --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-proposer")
              if [ ! -z "$PROPOSER_PODS" ]; then
                
                echo "$PROPOSER_PODS" | while read PROPOSER_POD; do
                  if [ ! -z "$PROPOSER_POD" ]; then
                    kubectl logs -n $NAMESPACE $PROPOSER_POD --tail=50 --since=1m | while read line; do
                      if [ ! -z "$line" ]; then
                        CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                        aws logs put-log-events --log-group-name "/aws/eks/$NAMESPACE/op-proposer" --log-stream-name "sidecar-collection" --log-events '[{"timestamp":'$(date +%%s)'000,"message":"'"$CLEAN_MESSAGE"'"}]' --region $AWS_REGION || true
                      fi
                    done
                  fi
                done
              fi
            fi
            
            sleep %d
          done
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
`, namespace, namespace,
		fmt.Sprintf("%t", componentMap["op-node"]),
		fmt.Sprintf("%t", componentMap["op-geth"]),
		fmt.Sprintf("%t", componentMap["op-batcher"]),
		fmt.Sprintf("%t", componentMap["op-proposer"]),
		loggingConfig.CollectionInterval, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply unified sidecar deployment
	logger.Info("Creating unified thanos-logs-sidecar deployment")
	if err := t.applyManifest(ctx, unifiedSidecar); err != nil {
		return fmt.Errorf("failed to create unified thanos-logs-sidecar deployment: %w", err)
	}

	logger.Info("Unified sidecar deployment created successfully")

	// Create CloudWatch log groups for all components
	logger.Info("Creating CloudWatch log groups for log collection")
	if err := t.createCloudWatchLogGroups(ctx, namespace); err != nil {
		logger.Warnw("Failed to create CloudWatch log groups", "err", err)
		// Don't return error as sidecar deployment is successful
	}

	return nil
}

// createCloudWatchLogGroups creates CloudWatch log groups for all components
func (t *ThanosStack) createCloudWatchLogGroups(ctx context.Context, namespace string) error {
	logger := t.getLogger()

	// Define log group names for all components
	logGroups := []string{
		fmt.Sprintf("/aws/eks/%s/op-node", namespace),
		fmt.Sprintf("/aws/eks/%s/op-geth", namespace),
		fmt.Sprintf("/aws/eks/%s/op-batcher", namespace),
		fmt.Sprintf("/aws/eks/%s/op-proposer", namespace),
	}

	// Set AWS region
	region := "ap-northeast-2"
	if t.deployConfig.AWS != nil && t.deployConfig.AWS.Region != "" {
		region = t.deployConfig.AWS.Region
	}

	// Create each log group
	for _, logGroupName := range logGroups {
		// Create log group
		createLogGroupCmd := []string{
			"logs", "create-log-group",
			"--log-group-name", logGroupName,
			"--region", region,
		}

		out, err := utils.ExecuteCommand(ctx, "aws", createLogGroupCmd...)
		if err != nil {
			// If log group already exists, that's fine
			if strings.Contains(out, "ResourceAlreadyExistsException") {
				logger.Infof("Log group already exists: %s", logGroupName)
				continue
			}
			logger.Warnf("Failed to create log group %s: %v, output: %s", logGroupName, err, out)
			continue
		}

		logger.Infof("Successfully created log group: %s", logGroupName)

		// Create log stream for the log group
		logStreamName := "sidecar-collection"
		createLogStreamCmd := []string{
			"logs", "create-log-stream",
			"--log-group-name", logGroupName,
			"--log-stream-name", logStreamName,
			"--region", region,
		}

		out, err = utils.ExecuteCommand(ctx, "aws", createLogStreamCmd...)
		if err != nil {
			// If log stream already exists, that's fine
			if strings.Contains(out, "ResourceAlreadyExistsException") {
				logger.Infof("Log stream already exists: %s/%s", logGroupName, logStreamName)
				continue
			}
			logger.Warnf("Failed to create log stream %s/%s: %v, output: %s", logGroupName, logStreamName, err, out)
			continue
		}

		logger.Infof("Successfully created log stream: %s/%s", logGroupName, logStreamName)
	}
	return nil
}

// applyManifest applies a Kubernetes manifest
func (t *ThanosStack) applyManifest(ctx context.Context, manifest string) error {
	return t.applyManifestWithTempFile(ctx, manifest, "k8s-manifest-*.yaml")
}

// applyManifestWithTempFile applies a Kubernetes manifest using a temporary file
func (t *ThanosStack) applyManifestWithTempFile(ctx context.Context, manifest, tempFilePattern string) error {
	// Write manifest to temporary file
	tmpFile, err := os.CreateTemp("", tempFilePattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(manifest); err != nil {
		return fmt.Errorf("failed to write manifest to temp file: %w", err)
	}
	tmpFile.Close()

	// Apply manifest
	out, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile.Name())
	if err != nil {
		return fmt.Errorf("failed to apply manifest: %w, output: %s, temp file: %s", err, out, tmpFile.Name())
	}

	return nil
}

// generatePrometheusStorageSpec creates Prometheus storage specification
func (t *ThanosStack) generatePrometheusStorageSpec(config *types.MonitoringConfig) map[string]interface{} {
	if !config.EnablePersistence {
		return map[string]interface{}{}
	}

	// Use fixed PV name without timestamp for automatic reuse
	pvName := fmt.Sprintf("%s-thanos-stack-prometheus", config.ChainName)

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
func (t *ThanosStack) generateGrafanaStorageConfig(config *types.MonitoringConfig) map[string]interface{} {
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
		"telegram_message": "🚨 Critical Alert - {{ .GroupLabels.chain_name }}\n\nAlert Name: {{ .GroupLabels.alertname }}\nSeverity: {{ .GroupLabels.severity }}\nComponent: {{ .GroupLabels.component }}\n\nSummary: {{ .CommonAnnotations.summary }}\nDescription: {{ .CommonAnnotations.description }}\n\n⏰ Alert Time: {{ range .Alerts }}{{ .StartsAt }}{{ end }}\n\nDashboard: [View Details](" + grafanaURL + ")",
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
func (t *ThanosStack) deployMonitoringInfrastructure(ctx context.Context, config *types.MonitoringConfig) error {
	logger := t.getLogger()

	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return fmt.Errorf("failed to ensure namespace exists: %w", err)
	}

	// Clean up existing monitoring PVs and PVCs
	if err := t.cleanupExistingMonitoringStorage(ctx, config); err != nil {
		return fmt.Errorf("failed to cleanup existing monitoring storage: %w", err)
	}

	// Wait for PV deletion to complete with polling
	logger.Info("Waiting for Grafana PV deletion to complete")
	err := wait.PollImmediate(500*time.Millisecond, 30*time.Second, func() (bool, error) {
		// Check if Grafana PVs still exist
		output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
		if err != nil {
			return false, nil // Continue polling on error
		}
		// Check if any Grafana PV still exists
		return !strings.Contains(output, "grafana"), nil
	})
	if err != nil {
		logger.Warnw("Timeout waiting for Grafana PV deletion, continuing anyway", "error", err)
	}

	// Create Prometheus PV and PVC with fixed naming (no timestamp)
	// This allows automatic reuse of existing PVs to preserve monitoring history
	prometheusPV := t.generateStaticPVManifest("prometheus", config, "20Gi")
	if err := t.applyPVManifest(ctx, "prometheus", prometheusPV); err != nil {
		return fmt.Errorf("failed to create Prometheus PV: %w", err)
	}

	prometheusPVC := t.generateStaticPVCManifest("prometheus", config, "20Gi")
	if err := t.applyPVCManifest(ctx, "prometheus", prometheusPVC); err != nil {
		return fmt.Errorf("failed to create Prometheus PVC: %w", err)
	}
	fmt.Println("✅ Created Prometheus PV and PVC")

	// Create Grafana PV and PVC with fixed naming (no timestamp)
	// Database files are cleaned up separately during uninstall to prevent password mismatch
	grafanaPV := t.generateStaticPVManifest("grafana", config, "10Gi")
	if err := t.applyPVManifest(ctx, "grafana", grafanaPV); err != nil {
		return fmt.Errorf("failed to create Grafana PV: %w", err)
	}

	grafanaPVC := t.generateStaticPVCManifest("grafana", config, "10Gi")
	if err := t.applyPVCManifest(ctx, "grafana", grafanaPVC); err != nil {
		return fmt.Errorf("failed to create Grafana PVC: %w", err)
	}
	fmt.Println("✅ Created Grafana PV and PVC")

	return nil
}

// cleanupExistingMonitoringStorage removes existing monitoring PVs and PVCs
// Grafana PVs are deleted completely to prevent password mismatch on reinstall
// Prometheus PVs are patched to allow reuse and preserve monitoring history
func (t *ThanosStack) cleanupExistingMonitoringStorage(ctx context.Context, config *types.MonitoringConfig) error {
	logger := t.getLogger()

	// Get all PVCs once to reduce API calls
	logger.Info("Fetching all PVCs in namespace")
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get existing PVCs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var grafanaPVCs, otherPVCs []string

	// Separate Grafana PVCs from others
	for _, line := range lines {
		pvcName := strings.TrimSpace(line)
		if pvcName == "" {
			continue
		}
		if strings.Contains(pvcName, "grafana") {
			grafanaPVCs = append(grafanaPVCs, pvcName)
		} else {
			otherPVCs = append(otherPVCs, pvcName)
		}
	}

	// First, delete Grafana PVCs to unbind their PVs
	logger.Info("Cleaning up Grafana PVCs before PV cleanup")
	for _, pvcName := range grafanaPVCs {
		logger.Infow("Deleting Grafana PVC to unbind PV", "pvcName", pvcName)
		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", pvcName, "-n", config.Namespace, "--ignore-not-found=true")
		if err != nil {
			logger.Warnw("Failed to delete Grafana PVC", "pvcName", pvcName, "error", err)
		}
	}

	// Wait for PVCs to be deleted and PVs to be released with polling
	logger.Info("Waiting for Grafana PVC deletion to complete")
	err = wait.PollImmediate(500*time.Millisecond, 30*time.Second, func() (bool, error) {
		// Check if Grafana PVCs still exist
		output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
		if err != nil {
			return false, nil // Continue polling on error
		}
		// Check if any Grafana PVC still exists
		for _, pvcName := range grafanaPVCs {
			if strings.Contains(output, pvcName) {
				return false, nil // Still exists, continue polling
			}
		}
		return true, nil // All deleted
	})
	if err != nil {
		logger.Warnw("Timeout waiting for Grafana PVC deletion, continuing anyway", "error", err)
	}

	// Clean up remaining non-Grafana PVCs
	deletedPVCs := 0
	for _, pvcName := range otherPVCs {
		// Check if PVC is bound to a pod
		boundOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", pvcName, "-n", config.Namespace, "-o", "jsonpath={.status.phase}")
		if err == nil && strings.TrimSpace(boundOutput) == "Bound" {
			// Check if any pod is using this PVC
			podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", config.Namespace, "-o", "jsonpath={.items[*].spec.volumes[*].persistentVolumeClaim.claimName}")
			if err == nil && strings.Contains(podOutput, pvcName) {
				continue
			}
		}

		// Delete PVC
		_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "pvc", pvcName, "-n", config.Namespace, "--ignore-not-found=true")
		if err == nil {
			deletedPVCs++
		}
	}

	// Get existing monitoring PVs
	output, err = utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "--no-headers", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase")
	if err != nil {
		return fmt.Errorf("failed to get existing PVs: %w", err)
	}

	lines = strings.Split(strings.TrimSpace(output), "\n")
	deletedPVs := 0

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		pvName := parts[0]
		status := parts[1]

		// Handle Grafana PVs: Delete completely to prevent password mismatch on reinstall
		if strings.Contains(pvName, "thanos-stack-grafana") {
			// Delete Grafana PV regardless of status (PVC already deleted above)
			logger.Infow("Deleting Grafana PV to ensure clean reinstall", "pvName", pvName, "status", status)
			_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", pvName, "--ignore-not-found=true")
			if err == nil {
				deletedPVs++
				logger.Infow("Successfully deleted Grafana PV", "pvName", pvName)
			} else {
				logger.Warnw("Failed to delete Grafana PV", "pvName", pvName, "error", err)
			}
		} else if strings.Contains(pvName, "thanos-stack-prometheus") {
			// Handle Prometheus PVs: Patch to allow reuse (preserve monitoring history)
			if status == "Released" {
				_, err = utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", pvName, "-p", `{"spec":{"claimRef":null}}`, "--type=merge")
				if err == nil {
					deletedPVs++
				}
			}
		}
	}

	return nil
}

// cleanupGrafanaDatabaseFiles deletes Grafana database files to prevent password reuse
// This function should be called before uninstalling the monitoring stack to ensure
// the Grafana pod is still running when we try to delete the database files
func (t *ThanosStack) cleanupGrafanaDatabaseFiles(ctx context.Context) error {
	logger := t.getLogger()

	// Delete Grafana database files from the running pod
	logger.Info("Cleaning up Grafana database files")
	if err := t.deleteGrafanaDatabase(ctx); err != nil {
		logger.Warnw("Failed to delete Grafana database", "error", err)
		return err
	}

	logger.Info("✅ Grafana database cleanup completed")
	return nil
}

// deleteGrafanaDatabase deletes the Grafana SQLite database from running Grafana pod
// This is called during uninstall to prevent password reuse on next installation
func (t *ThanosStack) deleteGrafanaDatabase(ctx context.Context) error {
	logger := t.getLogger()

	// Find running Grafana pod
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", "-n", constants.MonitoringNamespace, "-l", "app.kubernetes.io/name=grafana", "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil || strings.TrimSpace(podOutput) == "" {
		logger.Info("No running Grafana pod found, skipping database cleanup")
		return nil
	}

	podName := strings.TrimSpace(strings.Split(podOutput, "\n")[0])
	logger.Infow("Found running Grafana pod, deleting DB files", "podName", podName)

	// Delete DB files directly from running Grafana pod (without shell, just rm command)
	_, err = utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, podName, "-c", "grafana", "--", "rm", "-f", "/var/lib/grafana/grafana.db", "/var/lib/grafana/grafana.db-shm", "/var/lib/grafana/grafana.db-wal")
	if err != nil {
		return fmt.Errorf("failed to delete DB files from pod %s: %w", podName, err)
	}

	logger.Info("✅ Successfully deleted Grafana database files")
	return nil
}

// cleanupOldGrafanaDatabase checks for existing Grafana pod and cleans up old database
// This is called during install to ensure a fresh database is created with the new password
func (t *ThanosStack) cleanupOldGrafanaDatabase(ctx context.Context) error {
	logger := t.getLogger()

	// Check if there's an existing Grafana pod from a previous installation
	podOutput, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", "-n", constants.MonitoringNamespace, "-l", "app.kubernetes.io/name=grafana", "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil || strings.TrimSpace(podOutput) == "" {
		logger.Info("No existing Grafana pod found, skipping database cleanup")
		return nil
	}

	podName := strings.TrimSpace(strings.Split(podOutput, "\n")[0])
	logger.Infow("Found existing Grafana pod from previous installation", "podName", podName)

	// Delete old database files from the existing pod
	logger.Info("Deleting old Grafana database files...")
	_, err = utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, podName, "-c", "grafana", "--", "rm", "-f", "/var/lib/grafana/grafana.db", "/var/lib/grafana/grafana.db-shm", "/var/lib/grafana/grafana.db-wal")
	if err != nil {
		logger.Warnw("Failed to delete old database files", "error", err)
		logger.Info("Continuing with installation anyway")
		return nil
	}

	logger.Info("Old database files deleted successfully")
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

// generateStaticPVManifest generates PV manifest with fixed naming (no timestamp)
func (t *ThanosStack) generateStaticPVManifest(component string, config *types.MonitoringConfig, size string) string {
	pvName := fmt.Sprintf("%s-thanos-stack-%s", config.ChainName, component)

	// All components use the same EFS filesystem ID as volumeHandle
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

	_, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
	if err != nil {
		return fmt.Errorf("failed to apply PV manifest: %w", err)
	}

	return nil
}

// generateStaticPVCManifest generates PVC manifest with fixed naming (no timestamp)
func (t *ThanosStack) generateStaticPVCManifest(component string, config *types.MonitoringConfig, size string) string {
	pvcName := fmt.Sprintf("%s-%s", config.HelmReleaseName, component)
	pvName := fmt.Sprintf("%s-thanos-stack-%s", config.ChainName, component)

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

	_, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
	if err != nil {
		return fmt.Errorf("failed to apply PVC manifest: %w", err)
	}

	return nil
}

// cleanupExistingDashboardConfigMaps removes existing dashboard ConfigMaps
func (t *ThanosStack) cleanupExistingDashboardConfigMaps(ctx context.Context, config *types.MonitoringConfig) error {
	// List and delete existing dashboard ConfigMaps
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "configmap", "-n", config.Namespace, "-l", "grafana_dashboard=1", "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		// If no ConfigMaps exist, that's fine
		return nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		configMapName := strings.TrimSpace(line)
		if configMapName == "" {
			continue
		}

		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "configmap", configMapName, "-n", config.Namespace, "--ignore-not-found=true")
		if err != nil {
			return fmt.Errorf("failed to delete configmap: %w", err)
		}
	}

	return nil
}

// createDashboardConfigMaps creates ConfigMaps for Grafana dashboards
func (t *ThanosStack) createDashboardConfigMaps(ctx context.Context, config *types.MonitoringConfig) error {
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

		_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
		if err != nil {
			continue
		}
		os.Remove(tempFile)
	}
	return nil
}

// createAlertManagerSecret creates AlertManager configuration secret
func (t *ThanosStack) createAlertManagerSecret(ctx context.Context, config *types.MonitoringConfig) error {
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
func (t *ThanosStack) generateAlertManagerSecretConfig(config *types.MonitoringConfig, grafanaURL string) (string, error) {
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
		for _, email := range config.AlertManager.Email.AlertReceivers {
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
			"group_wait":      "10s",
			"group_interval":  "1m",
			"repeat_interval": "10m",
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
			"smtp_auth_username": config.AlertManager.Email.SmtpFrom,
			"smtp_auth_password": config.AlertManager.Email.SmtpAuthPassword,
			"smtp_require_tls":   true, // Required for Gmail and other secure SMTP servers
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

	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to apply secret manifest: %w", err)
	}

	return nil
}

// createPrometheusRule creates PrometheusRule for alerts
func (t *ThanosStack) createPrometheusRule(ctx context.Context, config *types.MonitoringConfig) error {
	// Clean up existing PrometheusRules except thanos-stack-alerts
	if err := t.cleanupExistingPrometheusRules(ctx, config); err != nil {
		return fmt.Errorf("failed to cleanup existing PrometheusRules: %w", err)
	}

	manifest := t.generatePrometheusRuleManifest(config)
	return t.applyPrometheusRuleManifest(ctx, manifest)
}

// cleanupExistingPrometheusRules removes all PrometheusRules except thanos-stack-alerts
func (t *ThanosStack) cleanupExistingPrometheusRules(ctx context.Context, config *types.MonitoringConfig) error {
	logger := t.getLogger()
	// Get all PrometheusRules in the monitoring namespace
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "prometheusrule", "-n", config.Namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		// If no PrometheusRules exist, that's fine
		return nil
	}

	if strings.TrimSpace(output) == "" {
		return nil
	}

	ruleNames := strings.Split(strings.TrimSpace(output), " ")

	for _, ruleName := range ruleNames {
		if ruleName == "" {
			continue
		}

		// Skip thanos-stack-alerts
		if strings.Contains(ruleName, "thanos-stack-alerts") {
			continue
		}

		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "prometheusrule", ruleName, "-n", config.Namespace)
		if err != nil {
			// Continue with other rules even if one fails
			// Log the error but don't fail the entire operation
			logger.Warnw("Failed to delete PrometheusRule", "rule", ruleName, "err", err)
		}
	}

	return nil
}

// generatePrometheusRuleManifest generates the complete PrometheusRule YAML manifest
func (t *ThanosStack) generatePrometheusRuleManifest(config *types.MonitoringConfig) string {
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
    interval: 15s
    rules:
    - alert: OpNodeDown
      expr: absent(up{job="op-node"}) or up{job="op-node"} == 0
      for: 30s
      labels:
        severity: critical
        component: op-node
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Node is down"
        description: "OP Node has been down for more than 30 seconds"
    
    - alert: OpBatcherDown
      expr: absent(up{job="op-batcher"})
      for: 30s
      labels:
        severity: critical
        component: op-batcher
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Batcher is down"
        description: "OP Batcher has been down for more than 30 seconds"
    
    - alert: OpProposerDown
      expr: absent(up{job="op-proposer"})
      for: 30s
      labels:
        severity: critical
        component: op-proposer
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Proposer is down"
        description: "OP Proposer has been down for more than 30 seconds"
    
    - alert: OpGethDown
      expr: absent(up{job="op-geth"}) or up{job="op-geth"} == 0
      for: 30s
      labels:
        severity: critical
        component: op-geth
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Geth is down"
        description: "OP Geth has been down for more than 30 seconds"
    
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
      expr: op_batcher_default_balance < 0.01
      for: 10s
      labels:
        severity: critical
        component: op-batcher
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Batcher ETH balance critically low"
        description: "OP Batcher balance is {{ $value }} ETH, below threshold"
    
    - alert: OpProposerBalanceCritical
      expr: op_proposer_default_balance < 0.01
      for: 10s
      labels:
        severity: critical
        component: op-proposer
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "OP Proposer ETH balance critically low"
        description: "OP Proposer balance is {{ $value }} ETH, below threshold"
    
    - alert: BlockProductionStalled
      expr: increase(chain_head_block[5m]) == 0
      for: 1m
      labels:
        severity: critical
        component: op-geth
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "Block production has stalled"
        description: "No new blocks have been produced"
    
    - alert: ContainerCpuUsageHigh
      expr: (sum(rate(container_cpu_usage_seconds_total[5m])) by (pod) / sum(container_spec_cpu_quota/container_spec_cpu_period) by (pod)) * 100 > 80
      for: 2m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "High CPU usage in Thanos Stack pod"
        description: "Pod {{ $labels.pod }} CPU usage has been above threshold for more than 2 minutes"
    
    - alert: ContainerMemoryUsageHigh
      expr: (sum(container_memory_working_set_bytes) by (pod) / sum(container_spec_memory_limit_bytes) by (pod)) * 100 > 80
      for: 2m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "High memory usage in Thanos Stack pod"
        description: "Pod {{ $labels.pod }} memory usage has been above threshold for more than 2 minutes"
    
    - alert: PodCrashLooping
      expr: rate(kube_pod_container_status_restarts_total[5m]) > 0
      for: 2m
      labels:
        severity: critical
        component: kubernetes
        chain_name: "%s"
        namespace: "%s"
      annotations:
        summary: "Pod is crash looping"
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently"
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

	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to apply PrometheusRule manifest: %w", err)
	}

	return nil
}

// getGrafanaURL returns the full Grafana dashboard URL using the ALB Ingress
func (t *ThanosStack) getGrafanaURL(ctx context.Context, config *types.MonitoringConfig) string {
	// Try to get the ALB Ingress hostname using the actual Helm release name
	ingressName := fmt.Sprintf("%s-grafana", config.HelmReleaseName)
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "ingress", "-n", config.Namespace, "-o", "jsonpath={.items[?(@.metadata.name==\""+ingressName+"\")].status.loadBalancer.ingress[0].hostname}")
	if err != nil || output == "" {
		// Fallback to using ExternalURL template variable for dynamic resolution
		return "{{ .ExternalURL }}/d/thanos-stack-app-v9/thanos-stack-application-monitoring-dashboard?orgId=1&refresh=30s"
	}
	return "http://" + output + "/d/thanos-stack-app-v9/thanos-stack-application-monitoring-dashboard?orgId=1&refresh=30s"
}

// checkCloudWatchLogGroupsStatus checks the status of CloudWatch Log Groups
func (t *ThanosStack) checkCloudWatchLogGroupsStatus(ctx context.Context, namespace string) error {
	logger := t.getLogger()

	// Check CloudWatch Log Groups status silently
	components := CoreComponents

	for _, component := range components {
		logGroupName := fmt.Sprintf("/aws/eks/%s/%s", namespace, component)

		checkCmd := []string{
			"logs", "describe-log-groups",
			"--log-group-name-prefix", logGroupName,
			"--region", t.deployConfig.AWS.Region,
			"--query", "logGroups[?logGroupName==`" + logGroupName + "`]",
			"--output", "text",
		}

		_, err := utils.ExecuteCommand(ctx, "aws", checkCmd...)
		if err != nil {
			// Lower log noise in periodic checks; keep as debug unless troubleshooting
			logger.Infow("CloudWatch log group check failed", "logGroup", logGroupName, "err", err)
		}
	}

	return nil
}

// updateRetentionPolicy updates CloudWatch Log Group retention policy without restarting sidecar
func (t *ThanosStack) updateRetentionPolicy(ctx context.Context, namespace string, retention int) error {
	logger := t.getLogger()

	components := CoreComponents

	for _, component := range components {
		logGroupName := fmt.Sprintf("/aws/eks/%s/%s", namespace, component)

		// Update retention policy
		cmd := exec.CommandContext(ctx, "aws", "logs", "put-retention-policy",
			"--log-group-name", logGroupName,
			"--retention-in-days", strconv.Itoa(retention),
			"--region", t.deployConfig.AWS.Region)

		if _, err := cmd.CombinedOutput(); err != nil {
			logger.Warnw("Failed to update retention policy", "logGroup", logGroupName, "error", err)
		}
	}

	return nil
}

// updateCollectionInterval updates collection interval by restarting the sidecar with new interval
func (t *ThanosStack) updateCollectionInterval(ctx context.Context, namespace string, interval int) error {
	logger := t.getLogger()

	// Get current logging config
	loggingConfig := t.deployConfig.LoggingConfig
	if loggingConfig == nil {
		loggingConfig = &types.LoggingConfig{
			Enabled:             true,
			CloudWatchRetention: 30,
			CollectionInterval:  interval,
		}
	} else {
		loggingConfig.CollectionInterval = interval
	}

	// Delete existing sidecar deployment
	logger.Info("Deleting existing sidecar deployment to apply new interval...")
	cmd := exec.CommandContext(ctx, "kubectl", "delete", "deployment", "thanos-logs-sidecar",
		"-n", namespace, "--ignore-not-found=true")

	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Warnw("Failed to delete existing sidecar deployment", "error", err)
	}

	// Wait for deployment to be fully deleted
	time.Sleep(5 * time.Second)

	// Recreate sidecar with new interval
	logger.Info("Recreating sidecar deployment with new interval...")
	if err := t.createSidecarDeployments(ctx, namespace, loggingConfig); err != nil {
		logger.Errorw("Failed to recreate sidecar deployment", "error", err)
		return fmt.Errorf("failed to recreate sidecar deployment: %w", err)
	}
	return nil
}

// verifyRetentionPolicy verifies the actual retention policy of CloudWatch Log Groups
func (t *ThanosStack) verifyRetentionPolicy(ctx context.Context, namespace string) error {
	logger := t.getLogger()
	components := CoreComponents

	for _, component := range components {
		logGroupName := fmt.Sprintf("/aws/eks/%s/%s", namespace, component)

		cmd := exec.CommandContext(ctx, "aws", "logs", "describe-log-groups",
			"--log-group-name-prefix", logGroupName,
			"--region", t.deployConfig.AWS.Region,
			"--query", "logGroups[0].retentionInDays",
			"--output", "text")

		if _, err := cmd.CombinedOutput(); err != nil {
			logger.Errorw("Verification failed", "component", component)
		}
	}

	return nil
}

// verifyCollectionInterval verifies the actual collection interval from sidecar
func (t *ThanosStack) verifyCollectionInterval(ctx context.Context, namespace string) error {
	logger := t.getLogger()
	// Fetch sidecar pod via label selector
	pods, err := utils.GetPodNamesByLabel(ctx, namespace, "app=thanos-logs-sidecar")
	if err != nil || len(pods) == 0 {
		return fmt.Errorf("unified sidecar pod not found")
	}
	podName := pods[0]

	// Get pod spec as JSON and extract container command/args
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", podName, "-n", namespace, "-o", "json")
	if err != nil {
		return fmt.Errorf("failed to get sidecar pod spec: %w", err)
	}

	// Parse minimal JSON to locate sleep argument
	type containerSpec struct {
		Command []string `json:"command"`
		Args    []string `json:"args"`
	}
	var podJSON struct {
		Spec struct {
			Containers []containerSpec `json:"containers"`
		} `json:"spec"`
	}
	if err := json.Unmarshal([]byte(out), &podJSON); err != nil {
		return fmt.Errorf("failed to parse pod json: %w", err)
	}

	// Search sleep value in command or args
	sleepRe := regexp.MustCompile(`\bsleep\s+(\d+)\b`)
	for _, c := range podJSON.Spec.Containers {
		joined := strings.Join(append(c.Command, c.Args...), " ")
		if m := sleepRe.FindStringSubmatch(joined); len(m) > 1 {
			logger.Infow("Collection Interval", "interval", m[1])
			return nil
		}
	}
	logger.Errorw("Could not extract sleep interval from sidecar command")
	return nil
}

// GetDeployConfig returns the deploy configuration
func (t *ThanosStack) GetDeployConfig() *types.Config {
	return t.deployConfig
}

// InstallLogCollectionSidecar installs AWS CLI sidecar for log collection
func (t *ThanosStack) InstallLogCollectionSidecar(ctx context.Context, namespace string, loggingConfig *types.LoggingConfig) error {
	return t.installLogCollectionSidecarDeployment(ctx, namespace, loggingConfig)
}

// UpdateRetentionPolicy updates the CloudWatch log retention policy
func (t *ThanosStack) UpdateRetentionPolicy(ctx context.Context, namespace string, retention int) error {
	return t.updateRetentionPolicy(ctx, namespace, retention)
}

// UpdateCollectionInterval updates the log collection interval
func (t *ThanosStack) UpdateCollectionInterval(ctx context.Context, namespace string, interval int) error {
	return t.updateCollectionInterval(ctx, namespace, interval)
}

// CleanupSidecarDeployments removes sidecar deployments
func (t *ThanosStack) CleanupSidecarDeployments(ctx context.Context, namespace string) error {
	return t.cleanupSidecarDeployments(ctx, namespace)
}

// CleanupRBACResources removes RBAC resources
func (t *ThanosStack) CleanupRBACResources(ctx context.Context) error {
	return t.cleanupRBACResources(ctx)
}

// GenerateValuesFile generates the values.yaml file
func (t *ThanosStack) GenerateValuesFile(config *types.MonitoringConfig) error {
	return t.generateValuesFile(config)
}

// VerifyRetentionPolicy verifies the retention policy
func (t *ThanosStack) VerifyRetentionPolicy(ctx context.Context, namespace string) error {
	return t.verifyRetentionPolicy(ctx, namespace)
}

// VerifyCollectionInterval verifies the collection interval
func (t *ThanosStack) VerifyCollectionInterval(ctx context.Context, namespace string) error {
	return t.verifyCollectionInterval(ctx, namespace)
}

// installLogCollectionSidecarDeployment installs AWS CLI sidecar deployment
func (t *ThanosStack) installLogCollectionSidecarDeployment(ctx context.Context, namespace string, loggingConfig *types.LoggingConfig) error {

	// Create ServiceAccount for sidecar
	serviceAccount := fmt.Sprintf(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: thanos-logs-sidecar
  namespace: %s
`, namespace)

	// Create ClusterRole for log access
	clusterRole := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: thanos-logs-sidecar-role
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
`

	// Create ClusterRoleBinding
	clusterRoleBinding := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: thanos-logs-sidecar-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: thanos-logs-sidecar-role
subjects:
- kind: ServiceAccount
  name: thanos-logs-sidecar
  namespace: %s
`, namespace)

	// Apply ClusterRole
	if err := t.applyManifest(ctx, clusterRole); err != nil {
		return fmt.Errorf("failed to create logs sidecar ClusterRole: %w", err)
	}

	// Apply ClusterRoleBinding
	if err := t.applyManifest(ctx, clusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create logs sidecar ClusterRoleBinding: %w", err)
	}

	// Apply ServiceAccount
	if err := t.applyManifest(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create logs sidecar ServiceAccount: %w", err)
	}

	// Create sidecar deployment for log collection
	if err := t.createSidecarDeployments(ctx, namespace, loggingConfig); err != nil {
		return fmt.Errorf("failed to create sidecar deployments: %w", err)
	}

	return nil
}
