package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// InstallMonitoring installs monitoring stack using Helm
func (t *ThanosStack) InstallMonitoring(ctx context.Context, config *types.MonitoringConfig) (*types.MonitoringInfo, error) {
	logger := t.l
	// fallback to zap.NewExample().Sugar() if logger is nil
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	logger.Info("🚀 Starting monitoring installation...")

	// Ensure monitoring namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		logger.Errorw("Failed to ensure monitoring namespace exists", "err", err)
		return nil, fmt.Errorf("failed to ensure monitoring namespace exists: %w", err)
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
	logger.Info("Generating values file for monitoring stack")
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

	// Clear Helm cache to prevent conflicts
	logger.Info("Clearing Helm cache")
	_, err = utils.ExecuteCommand(ctx, "helm", "repo", "update")
	if err != nil {
		logger.Errorw("Failed to update Helm repos", "err", err)
	}

	// Install monitoring stack
	logger.Infow("Installing monitoring stack via Helm", "release", config.HelmReleaseName)
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
		logger.Errorw("Failed to install monitoring stack", "err", err, "output", out)
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

	// Install AWS Fluent Bit for log collection if logging is enabled
	if config.LoggingEnabled {
		logger.Info("Installing AWS Fluent Bit for log collection")
		if err := t.installFluentBit(ctx, config); err != nil {
			logger.Errorw("Failed to install AWS Fluent Bit", "err", err)
			// Continue with installation even if Fluent Bit fails
			logger.Warn("Continuing without log collection")
		}
	}

	monitoringInfo := t.createMonitoringInfo(ctx, config)
	if monitoringInfo == nil {
		logger.Error("ALB Ingress is not ready after installation")
		return nil, fmt.Errorf("ALB Ingress is not ready after installation")
	}

	logger.Info("Monitoring installation completed successfully")
	return monitoringInfo, nil
}

// GetMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) GetMonitoringConfig(ctx context.Context, adminPassword string, alertManagerConfig types.AlertManagerConfig, loggingEnabled bool) (*types.MonitoringConfig, error) {
	// Remove trailing % character from admin password if present
	adminPassword = strings.TrimSuffix(adminPassword, "%")

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

	config := &types.MonitoringConfig{
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
		LoggingEnabled:    loggingEnabled,
	}

	return config, nil
}

// UninstallMonitoring removes monitoring plugin
func (t *ThanosStack) UninstallMonitoring(ctx context.Context) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}
	logger.Info("Starting monitoring uninstallation...")
	monitoringNamespace := "monitoring"

	// Get actual namespace for Thanos Stack components
	actualNamespace, err := t.getActualNamespace(ctx)
	if err != nil {
		logger.Warnw("Failed to get actual namespace, will skip sidecar cleanup", "err", err)
	} else {
		// Clean up Sidecar deployments
		logger.Info("Cleaning up Sidecar deployments")
		if err := t.cleanupSidecarDeployments(ctx, actualNamespace); err != nil {
			logger.Warnw("Failed to cleanup Sidecar deployments", "err", err)
		}

		// Clean up RBAC resources
		logger.Info("Cleaning up RBAC resources")
		if err := t.cleanupRBACResources(ctx); err != nil {
			logger.Warnw("Failed to cleanup RBAC resources", "err", err)
		}
	}

	releases, err := utils.FilterHelmReleases(ctx, monitoringNamespace, "monitoring")
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

	logger.Infow("Deleting monitoring namespace", "namespace", monitoringNamespace)
	_, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", monitoringNamespace, "--ignore-not-found=true")
	if err != nil {
		logger.Errorw("Failed to delete namespace", "err", err, "namespace", monitoringNamespace)
		return err
	}
	logger.Info("🧹 Monitoring plugin uninstalled successfully")
	return nil
}

// cleanupSidecarDeployments cleans up Sidecar deployments
func (t *ThanosStack) cleanupSidecarDeployments(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	// List of Sidecar deployments to delete
	sidecarDeployments := []string{
		"op-node-sidecar",
		"op-geth-sidecar",
		"op-batcher-sidecar",
		"op-proposer-sidecar",
	}

	for _, deployment := range sidecarDeployments {
		logger.Infow("Deleting Sidecar deployment", "deployment", deployment, "namespace", namespace)
		out, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "deployment", deployment, "-n", namespace, "--ignore-not-found=true")
		if err != nil {
			logger.Warnw("Failed to delete Sidecar deployment", "deployment", deployment, "err", err, "output", out)
		} else {
			logger.Infow("Successfully deleted Sidecar deployment", "deployment", deployment)
		}
	}

	// Delete ConfigMap
	logger.Info("Deleting Fluent Bit Sidecar ConfigMap")
	out, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "configmap", "fluent-bit-sidecar-config", "-n", namespace, "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete Fluent Bit Sidecar ConfigMap", "err", err, "output", out)
	} else {
		logger.Info("Successfully deleted Fluent Bit Sidecar ConfigMap")
	}

	return nil
}

// cleanupRBACResources cleans up RBAC resources
func (t *ThanosStack) cleanupRBACResources(ctx context.Context) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	// Delete ClusterRoleBinding
	logger.Info("Deleting Fluent Bit Sidecar ClusterRoleBinding")
	out, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "clusterrolebinding", "fluent-bit-sidecar-binding", "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete Fluent Bit Sidecar ClusterRoleBinding", "err", err, "output", out)
	} else {
		logger.Info("Successfully deleted Fluent Bit Sidecar ClusterRoleBinding")
	}

	// Delete ClusterRole
	logger.Info("Deleting Fluent Bit Sidecar ClusterRole")
	out, err = utils.ExecuteCommand(ctx, "kubectl", "delete", "clusterrole", "fluent-bit-sidecar-role", "--ignore-not-found=true")
	if err != nil {
		logger.Warnw("Failed to delete Fluent Bit Sidecar ClusterRole", "err", err, "output", out)
	} else {
		logger.Info("Successfully deleted Fluent Bit Sidecar ClusterRole")
	}

	// Delete ServiceAccount from all namespaces (it might be in multiple namespaces)
	namespaces := []string{"theo0730-78s3a", "monitoring"}
	for _, namespace := range namespaces {
		logger.Infow("Deleting Fluent Bit Sidecar ServiceAccount", "namespace", namespace)
		out, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "serviceaccount", "fluent-bit-sidecar", "-n", namespace, "--ignore-not-found=true")
		if err != nil {
			logger.Warnw("Failed to delete Fluent Bit Sidecar ServiceAccount", "namespace", namespace, "err", err, "output", out)
		} else {
			logger.Infow("Successfully deleted Fluent Bit Sidecar ServiceAccount", "namespace", namespace)
		}
	}

	return nil
}

// DisplayMonitoringInfo displays monitoring information
func (t *ThanosStack) DisplayMonitoringInfo(monitoringInfo *types.MonitoringInfo) {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}
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
func (t *ThanosStack) generateValuesFile(config *types.MonitoringConfig) error {
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
		// Get actual namespace from deployed pods
		actualNamespace, err := t.getActualNamespace(context.Background())
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to get actual namespace, using chain name: %v\n", err)
			actualNamespace = config.ChainName
		}

		// Define log groups for different components using actual namespace
		logGroups := []map[string]interface{}{
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-node", actualNamespace),
				"retention":   30,
				"description": "OP Node logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-batcher", actualNamespace),
				"retention":   30,
				"description": "OP Batcher logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-proposer", actualNamespace),
				"retention":   30,
				"description": "OP Proposer logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/op-geth", actualNamespace),
				"retention":   30,
				"description": "OP Geth logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/blockscout", actualNamespace),
				"retention":   30,
				"description": "BlockScout logs",
			},
			{
				"name":        fmt.Sprintf("/aws/eks/%s/application", actualNamespace),
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

	// Debug: Print the generated values for troubleshooting
	fmt.Printf("🔍 Generated monitoring values:\n%s\n", string(yamlContent))

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

// getActualNamespace gets the actual namespace where Thanos Stack components are deployed
func (t *ThanosStack) getActualNamespace(ctx context.Context) (string, error) {
	// Try to find namespace by looking for Thanos Stack pods
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-A", "--no-headers", "-o", "custom-columns=NAMESPACE:.metadata.namespace")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get namespaces: %w", err)
	}

	namespaces := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Look for namespace containing Thanos Stack components
	for _, namespace := range namespaces {
		namespace = strings.TrimSpace(namespace)
		if namespace == "" {
			continue
		}

		// Check if this namespace contains Thanos Stack pods
		cmd = exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
		podOutput, err := cmd.Output()
		if err != nil {
			continue
		}

		podNames := strings.Split(strings.TrimSpace(string(podOutput)), "\n")
		for _, podName := range podNames {
			podName = strings.TrimSpace(podName)
			if strings.Contains(podName, "op-node") ||
				strings.Contains(podName, "op-geth") ||
				strings.Contains(podName, "op-batcher") ||
				strings.Contains(podName, "op-proposer") {
				return namespace, nil
			}
		}
	}

	return "", fmt.Errorf("no Thanos Stack namespace found")
}

// installFluentBit installs AWS Fluent Bit for log collection
func (t *ThanosStack) installFluentBit(ctx context.Context, config *types.MonitoringConfig) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	// Get actual namespace where Thanos Stack components are deployed
	actualNamespace, err := t.getActualNamespace(ctx)
	if err != nil {
		return fmt.Errorf("failed to get actual namespace: %w", err)
	}

	logger.Infow("Installing AWS Fluent Bit", "namespace", actualNamespace)

	// Install AWS Fluent Bit as Sidecar containers
	logger.Info("Installing AWS Fluent Bit as Sidecar containers")
	if err := t.installFluentBitSidecar(ctx, actualNamespace); err != nil {
		return fmt.Errorf("failed to install AWS Fluent Bit via Sidecar: %w", err)
	}

	logger.Info("AWS Fluent Bit installed successfully")
	return nil
}

// installFluentBitSidecar installs AWS Fluent Bit as Sidecar containers
func (t *ThanosStack) installFluentBitSidecar(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	logger.Info("Installing AWS Fluent Bit as Sidecar containers")

	// Create Fluent Bit ConfigMap for sidecar
	fluentBitConfigMap := fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-sidecar-config
  namespace: %s
data:
  fluent-bit.conf: |
    [SERVICE]
        Parsers_File    parsers.conf
        HTTP_Server     On
        HTTP_Listen     0.0.0.0
        HTTP_Port       2020

    [INPUT]
        Name                tail
        Tag                 kube.*
        Path                /var/log/containers/*.log
        Parser              docker
        DB                  /var/log/flb_kube.db
        Skip_Long_Lines     On
        Refresh_Interval    10
        Mem_Buf_Limit      5MB

    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL           https://kubernetes.default.svc:443
        Kube_CA_Path       /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_Path    /var/run/secrets/kubernetes.io/serviceaccount/token
        Merge_Log          On
        Merge_Log_Key      log_processed
        K8S-Logging.Parser On
        K8S-Logging.Exclude On
        Use_Kubelet        Off
        Kubelet_Port       10250

    [OUTPUT]
        Name                cloudwatch
        Match               *
        region              %s
        log_group_name      /aws/eks/%s/op-node
        log_stream_prefix   fluentbit-sidecar-
        auto_create_group   true
        use_put_log_events true
        log_retention_days  30

  parsers.conf: |
    [PARSER]
        Name        docker
        Format      json
        Time_Key    time
        Time_Format %%Y-%%m-%%dT%%H:%%M:%%S.%%L
        Time_Keep   On
`, namespace, t.deployConfig.AWS.Region, namespace)

	// Apply Fluent Bit ConfigMap
	logger.Info("Creating Fluent Bit Sidecar ConfigMap")
	if err := t.applyManifest(ctx, fluentBitConfigMap); err != nil {
		return fmt.Errorf("failed to create Fluent Bit Sidecar ConfigMap: %w", err)
	}

	// Create ServiceAccount for sidecar
	serviceAccount := fmt.Sprintf(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluent-bit-sidecar
  namespace: %s
`, namespace)

	// Create ClusterRole for log access
	clusterRole := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: fluent-bit-sidecar-role
rules:
- apiGroups: [""]
  resources: ["pods", "pods/log"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["namespaces"]
  verbs: ["get", "list"]
`)

	// Create ClusterRoleBinding
	clusterRoleBinding := fmt.Sprintf(`
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: fluent-bit-sidecar-binding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: fluent-bit-sidecar-role
subjects:
- kind: ServiceAccount
  name: fluent-bit-sidecar
  namespace: %s
`, namespace)

	// Apply ClusterRole
	logger.Info("Creating Fluent Bit Sidecar ClusterRole")
	if err := t.applyManifest(ctx, clusterRole); err != nil {
		return fmt.Errorf("failed to create Fluent Bit Sidecar ClusterRole: %w", err)
	}

	// Apply ClusterRoleBinding
	logger.Info("Creating Fluent Bit Sidecar ClusterRoleBinding")
	if err := t.applyManifest(ctx, clusterRoleBinding); err != nil {
		return fmt.Errorf("failed to create Fluent Bit Sidecar ClusterRoleBinding: %w", err)
	}

	// Apply ServiceAccount
	logger.Info("Creating Fluent Bit Sidecar ServiceAccount")
	if err := t.applyManifest(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create Fluent Bit Sidecar ServiceAccount: %w", err)
	}

	// Create sidecar deployment for each Thanos Stack component
	logger.Info("Creating sidecar deployments for Thanos Stack components")
	if err := t.createSidecarDeployments(ctx, namespace); err != nil {
		return fmt.Errorf("failed to create sidecar deployments: %w", err)
	}

	logger.Info("AWS Fluent Bit Sidecar installed successfully")
	return nil
}

// patchPodsWithSidecar patches existing pods to add Fluent Bit sidecar
func (t *ThanosStack) patchPodsWithSidecar(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	// Get existing Thanos Stack pods
	cmd := exec.CommandContext(ctx, "kubectl", "get", "pods", "-n", namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	podNames := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, podName := range podNames {
		podName = strings.TrimSpace(podName)
		if podName == "" {
			continue
		}

		// Check if pod is a Thanos Stack component
		if strings.Contains(podName, "op-node") ||
			strings.Contains(podName, "op-geth") ||
			strings.Contains(podName, "op-batcher") ||
			strings.Contains(podName, "op-proposer") {

			logger.Infow("Patching pod with Fluent Bit sidecar", "pod", podName)

			// Create sidecar patch
			sidecarPatch := fmt.Sprintf(`
{
  "spec": {
    "template": {
      "spec": {
        "containers": [
          {
            "name": "fluent-bit-sidecar",
            "image": "public.ecr.aws/aws-observability/aws-for-fluent-bit:stable",
            "imagePullPolicy": "Always",
            "env": [
              {
                "name": "AWS_REGION",
                "value": "%s"
              },
              {
                "name": "AWS_ACCESS_KEY_ID",
                "value": "%s"
              },
              {
                "name": "AWS_SECRET_ACCESS_KEY",
                "value": "%s"
              }
            ],
            "volumeMounts": [
              {
                "name": "varlog",
                "mountPath": "/var/log"
              },
              {
                "name": "varlibdockercontainers",
                "mountPath": "/var/lib/docker/containers",
                "readOnly": true
              },
              {
                "name": "fluentbitconfig",
                "mountPath": "/fluent-bit/etc/"
              }
            ]
          }
        ],
        "volumes": [
          {
            "name": "varlog",
            "hostPath": {
              "path": "/var/log"
            }
          },
          {
            "name": "varlibdockercontainers",
            "hostPath": {
              "path": "/var/lib/docker/containers"
            }
          },
          {
            "name": "fluentbitconfig",
            "configMap": {
              "name": "fluent-bit-sidecar-config"
            }
          }
        ]
      }
    }
  }
}
`, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

			// Write patch to temporary file
			tmpFile, err := os.CreateTemp("", "sidecar-patch-*.json")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(sidecarPatch); err != nil {
				return fmt.Errorf("failed to write patch to temp file: %w", err)
			}
			tmpFile.Close()

			// Apply patch
			out, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "deployment", strings.TrimSuffix(podName, "-0"), "-n", namespace, "--type", "strategic", "--patch", fmt.Sprintf("$(cat %s)", tmpFile.Name()))
			if err != nil {
				logger.Warnw("Failed to patch pod", "pod", podName, "err", err, "output", out)
				continue
			}

			logger.Infow("Successfully patched pod", "pod", podName)
		}
	}

	return nil
}

// installFluentBitCronJob installs AWS Fluent Bit as CronJob
func (t *ThanosStack) installFluentBitCronJob(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	logger.Info("Installing AWS Fluent Bit as CronJob")

	// Create CronJob for log collection
	cronJob := fmt.Sprintf(`
apiVersion: batch/v1
kind: CronJob
metadata:
  name: log-collector
  namespace: %s
spec:
  schedule: "*/5 * * * *"  # 5분마다 실행
  jobTemplate:
    spec:
      template:
        spec:
          serviceAccountName: fluent-bit-cronjob
          containers:
          - name: log-collector
            image: amazon/aws-cli:latest
            command:
            - /bin/bash
            - -c
            - |
              # op-node 로그 수집
              kubectl logs -n %s theo0730-78s3a-1753840526-thanos-stack-op-node-0 --tail=100 --since=5m | \
              while read line; do
                aws logs put-log-events \
                  --log-group-name "/aws/eks/%s/op-node" \
                  --log-stream-name "cronjob-collection" \
                  --log-events timestamp=$(date +%%s)000,message="$line" \
                  --region %s
              done
              
              # op-geth 로그 수집
              kubectl logs -n %s theo0730-78s3a-1753840526-thanos-stack-op-geth-0 --tail=100 --since=5m | \
              while read line; do
                aws logs put-log-events \
                  --log-group-name "/aws/eks/%s/op-geth" \
                  --log-stream-name "cronjob-collection" \
                  --region %s \
                  --log-events timestamp=$(date +%%s)000,message="$line"
              done
              
              # op-batcher 로그 수집
              kubectl logs -n %s theo0730-78s3a-1753840526-thanos-stack-op-batcher-* --tail=100 --since=5m | \
              while read line; do
                aws logs put-log-events \
                  --log-group-name "/aws/eks/%s/op-batcher" \
                  --log-stream-name "cronjob-collection" \
                  --region %s \
                  --log-events timestamp=$(date +%%s)000,message="$line"
              done
              
              # op-proposer 로그 수집
              kubectl logs -n %s theo0730-78s3a-1753840526-thanos-stack-op-proposer-* --tail=100 --since=5m | \
              while read line; do
                aws logs put-log-events \
                  --log-group-name "/aws/eks/%s/op-proposer" \
                  --log-stream-name "cronjob-collection" \
                  --region %s \
                  --log-events timestamp=$(date +%%s)000,message="$line"
              done
            env:
            - name: AWS_REGION
              value: "%s"
            - name: AWS_ACCESS_KEY_ID
              value: "%s"
            - name: AWS_SECRET_ACCESS_KEY
              value: "%s"
          restartPolicy: OnFailure
`, namespace, namespace, namespace, t.deployConfig.AWS.Region, namespace, namespace, t.deployConfig.AWS.Region, namespace, namespace, t.deployConfig.AWS.Region, namespace, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply CronJob
	logger.Info("Creating Log Collector CronJob")
	if err := t.applyManifest(ctx, cronJob); err != nil {
		return fmt.Errorf("failed to create Log Collector CronJob: %w", err)
	}

	// Create ServiceAccount for CronJob
	serviceAccount := fmt.Sprintf(`
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluent-bit-cronjob
  namespace: %s
`, namespace)

	// Apply ServiceAccount
	logger.Info("Creating Fluent Bit CronJob ServiceAccount")
	if err := t.applyManifest(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create Fluent Bit CronJob ServiceAccount: %w", err)
	}

	logger.Info("AWS Fluent Bit CronJob installed successfully")
	return nil
}

// installManualLogCollection installs manual log collection script
func (t *ThanosStack) installManualLogCollection(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	logger.Info("Installing manual log collection script")

	// Create log collection script
	script := fmt.Sprintf(`#!/bin/bash
# Manual Log Collection Script for Thanos Stack
# Usage: ./collect-logs.sh

NAMESPACE="%s"
REGION="%s"
AWS_ACCESS_KEY_ID="%s"
AWS_SECRET_ACCESS_KEY="%s"

echo "Starting log collection for Thanos Stack components..."

# op-node 로그 수집
echo "Collecting op-node logs..."
kubectl logs -n $NAMESPACE theo0730-78s3a-1753840526-thanos-stack-op-node-0 --tail=100 --since=5m | \
while read line; do
    aws logs put-log-events \
        --log-group-name "/aws/eks/%s/op-node" \
        --log-stream-name "manual-collection" \
        --log-events timestamp=$(date +%%s)000,message="$line" \
        --region $REGION
done

# op-geth 로그 수집
echo "Collecting op-geth logs..."
kubectl logs -n $NAMESPACE theo0730-78s3a-1753840526-thanos-stack-op-geth-0 --tail=100 --since=5m | \
while read line; do
    aws logs put-log-events \
        --log-group-name "/aws/eks/%s/op-geth" \
        --log-stream-name "manual-collection" \
        --region $REGION \
        --log-events timestamp=$(date +%%s)000,message="$line"
done

# op-batcher 로그 수집
echo "Collecting op-batcher logs..."
kubectl logs -n $NAMESPACE theo0730-78s3a-1753840526-thanos-stack-op-batcher-* --tail=100 --since=5m | \
while read line; do
    aws logs put-log-events \
        --log-group-name "/aws/eks/%s/op-batcher" \
        --log-stream-name "manual-collection" \
        --region $REGION \
        --log-events timestamp=$(date +%%s)000,message="$line"
done

# op-proposer 로그 수집
echo "Collecting op-proposer logs..."
kubectl logs -n $NAMESPACE theo0730-78s3a-1753840526-thanos-stack-op-proposer-* --tail=100 --since=5m | \
while read line; do
    aws logs put-log-events \
        --log-group-name "/aws/eks/%s/op-proposer" \
        --log-stream-name "manual-collection" \
        --region $REGION \
        --log-events timestamp=$(date +%%s)000,message="$line"
done

echo "Log collection completed!"
`, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey, namespace, namespace, namespace, namespace)

	// Write script to file
	scriptPath := filepath.Join(t.deploymentPath, "collect-logs.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		return fmt.Errorf("failed to write log collection script: %w", err)
	}

	logger.Infow("Manual log collection script created", "path", scriptPath)
	logger.Info("To collect logs manually, run: ./collect-logs.sh")

	return nil
}

// createSidecarDeployments creates sidecar deployments for Thanos Stack components
func (t *ThanosStack) createSidecarDeployments(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	// Create sidecar deployment for op-node
	opNodeSidecar := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: op-node-sidecar
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: op-node-sidecar
  template:
    metadata:
      labels:
        app: op-node-sidecar
    spec:
      serviceAccountName: fluent-bit-sidecar
      containers:
      - name: log-collector
        image: amazon/aws-cli:latest
        command:
        - /bin/bash
        - -c
        - |
          # Install kubectl
          curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
          chmod +x kubectl
          mv kubectl /usr/local/bin/
          
          while true; do
            # op-node 로그 수집
            NODE_POD=$(kubectl get pods -n %s --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-node" | head -1)
            if [ ! -z "$NODE_POD" ]; then
              # 로그 스트림 생성 확인
              aws logs create-log-stream --log-group-name "/aws/eks/theo0730-78s3a/op-node" --log-stream-name "sidecar-collection" --region %s 2>/dev/null || true
              
              kubectl logs -n %s $NODE_POD --tail=50 --since=1m | \
              while read line; do
                # 로그 메시지 JSON 이스케이프 처리 (개선된 버전)
                if [ ! -z "$line" ]; then
                  # 간단한 메시지로 전송 (특수문자 제거)
                  CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                  aws logs put-log-events \
                    --log-group-name "/aws/eks/theo0730-78s3a/op-node" \
                    --log-stream-name "sidecar-collection" \
                    --log-events timestamp=$(date +%%s)000,message="$CLEAN_MESSAGE" \
                    --region %s || true
                fi
              done
            fi
            
            sleep 30
          done
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
        volumeMounts:
        - name: fluentbitconfig
          mountPath: /fluent-bit/etc/
      volumes:
      - name: fluentbitconfig
        configMap:
          name: fluent-bit-sidecar-config
`, namespace, namespace, t.deployConfig.AWS.Region, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply op-node sidecar deployment
	logger.Info("Creating op-node sidecar deployment")
	if err := t.applyManifest(ctx, opNodeSidecar); err != nil {
		return fmt.Errorf("failed to create op-node sidecar deployment: %w", err)
	}

	// Create sidecar deployment for op-geth
	opGethSidecar := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: op-geth-sidecar
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: op-geth-sidecar
  template:
    metadata:
      labels:
        app: op-geth-sidecar
    spec:
      serviceAccountName: fluent-bit-sidecar
      containers:
      - name: log-collector
        image: amazon/aws-cli:latest
        command:
        - /bin/bash
        - -c
        - |
          # Install kubectl
          curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
          chmod +x kubectl
          mv kubectl /usr/local/bin/
          
          while true; do
            # op-geth 로그 수집
            GETH_POD=$(kubectl get pods -n %s --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-geth" | head -1)
            if [ ! -z "$GETH_POD" ]; then
              # 로그 스트림 생성 확인
              aws logs create-log-stream --log-group-name "/aws/eks/theo0730-78s3a/op-geth" --log-stream-name "sidecar-collection" --region %s 2>/dev/null || true
              
              kubectl logs -n %s $GETH_POD --tail=50 --since=1m | \
              while read line; do
                # 로그 메시지 JSON 이스케이프 처리 (개선된 버전)
                if [ ! -z "$line" ]; then
                  # 간단한 메시지로 전송 (특수문자 제거)
                  CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                  aws logs put-log-events \
                    --log-group-name "/aws/eks/theo0730-78s3a/op-geth" \
                    --log-stream-name "sidecar-collection" \
                    --log-events timestamp=$(date +%%s)000,message="$CLEAN_MESSAGE" \
                    --region %s || true
                fi
              done
            fi
            
            sleep 30
          done
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
        volumeMounts:
        - name: fluentbitconfig
          mountPath: /fluent-bit/etc/
      volumes:
      - name: fluentbitconfig
        configMap:
          name: fluent-bit-sidecar-config
`, namespace, namespace, t.deployConfig.AWS.Region, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply op-geth sidecar deployment
	logger.Info("Creating op-geth sidecar deployment")
	if err := t.applyManifest(ctx, opGethSidecar); err != nil {
		return fmt.Errorf("failed to create op-geth sidecar deployment: %w", err)
	}

	// Create sidecar deployment for op-batcher
	opBatcherSidecar := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: op-batcher-sidecar
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: op-batcher-sidecar
  template:
    metadata:
      labels:
        app: op-batcher-sidecar
    spec:
      serviceAccountName: fluent-bit-sidecar
      containers:
      - name: log-collector
        image: amazon/aws-cli:latest
        command:
        - /bin/bash
        - -c
        - |
          # Install kubectl
          curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
          chmod +x kubectl
          mv kubectl /usr/local/bin/
          
          while true; do
            # op-batcher 로그 수집
            BATCHER_PODS=$(kubectl get pods -n %s --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-batcher")
            if [ ! -z "$BATCHER_PODS" ]; then
              # 로그 스트림 생성 확인
              aws logs create-log-stream --log-group-name "/aws/eks/theo0730-78s3a/op-batcher" --log-stream-name "sidecar-collection" --region %s 2>/dev/null || true
              
              echo "$BATCHER_PODS" | while read BATCHER_POD; do
                if [ ! -z "$BATCHER_POD" ]; then
                  kubectl logs -n %s $BATCHER_POD --tail=50 --since=1m | \
                  while read line; do
                    # 로그 메시지 JSON 이스케이프 처리 (개선된 버전)
                    if [ ! -z "$line" ]; then
                      # 간단한 메시지로 전송 (특수문자 제거)
                      CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                      aws logs put-log-events \
                        --log-group-name "/aws/eks/theo0730-78s3a/op-batcher" \
                        --log-stream-name "sidecar-collection" \
                        --log-events timestamp=$(date +%%s)000,message="$CLEAN_MESSAGE" \
                        --region %s || true
                    fi
                  done
                fi
              done
            fi
            
            sleep 30
          done
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
        volumeMounts:
        - name: fluentbitconfig
          mountPath: /fluent-bit/etc/
      volumes:
      - name: fluentbitconfig
        configMap:
          name: fluent-bit-sidecar-config
`, namespace, namespace, t.deployConfig.AWS.Region, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply op-batcher sidecar deployment
	logger.Info("Creating op-batcher sidecar deployment")
	if err := t.applyManifest(ctx, opBatcherSidecar); err != nil {
		return fmt.Errorf("failed to create op-batcher sidecar deployment: %w", err)
	}

	// Create sidecar deployment for op-proposer
	opProposerSidecar := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: op-proposer-sidecar
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app: op-proposer-sidecar
  template:
    metadata:
      labels:
        app: op-proposer-sidecar
    spec:
      serviceAccountName: fluent-bit-sidecar
      containers:
      - name: log-collector
        image: amazon/aws-cli:latest
        command:
        - /bin/bash
        - -c
        - |
          # Install kubectl
          curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
          chmod +x kubectl
          mv kubectl /usr/local/bin/
          
          while true; do
            # op-proposer 로그 수집
            PROPOSER_PODS=$(kubectl get pods -n %s --no-headers -o custom-columns="NAME:.metadata.name" | grep "thanos-stack-op-proposer")
            if [ ! -z "$PROPOSER_PODS" ]; then
              # 로그 스트림 생성 확인
              aws logs create-log-stream --log-group-name "/aws/eks/theo0730-78s3a/op-proposer" --log-stream-name "sidecar-collection" --region %s 2>/dev/null || true
              
              echo "$PROPOSER_PODS" | while read PROPOSER_POD; do
                if [ ! -z "$PROPOSER_POD" ]; then
                  kubectl logs -n %s $PROPOSER_POD --tail=50 --since=1m | \
                  while read line; do
                    # 로그 메시지 JSON 이스케이프 처리 (개선된 버전)
                    if [ ! -z "$line" ]; then
                      # 간단한 메시지로 전송 (특수문자 제거)
                      CLEAN_MESSAGE=$(echo "$line" | tr -d '"' | tr -d "'" | tr -d '\\' | tr -d '\n' | tr -d '\r' | tr -d '\t')
                      aws logs put-log-events \
                        --log-group-name "/aws/eks/theo0730-78s3a/op-proposer" \
                        --log-stream-name "sidecar-collection" \
                        --log-events timestamp=$(date +%%s)000,message="$CLEAN_MESSAGE" \
                        --region %s || true
                    fi
                  done
                fi
              done
            fi
            
            sleep 30
          done
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
        volumeMounts:
        - name: fluentbitconfig
          mountPath: /fluent-bit/etc/
      volumes:
      - name: fluentbitconfig
        configMap:
          name: fluent-bit-sidecar-config
`, namespace, namespace, t.deployConfig.AWS.Region, namespace, t.deployConfig.AWS.Region, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply op-proposer sidecar deployment
	logger.Info("Creating op-proposer sidecar deployment")
	if err := t.applyManifest(ctx, opProposerSidecar); err != nil {
		return fmt.Errorf("failed to create op-proposer sidecar deployment: %w", err)
	}

	logger.Info("All sidecar deployments created successfully")
	return nil
}

// installFluentBitDaemonSet installs AWS Fluent Bit as DaemonSet
func (t *ThanosStack) installFluentBitDaemonSet(ctx context.Context, namespace string) error {
	logger := t.l
	if logger == nil {
		logger = zap.NewExample().Sugar()
	}

	logger.Info("Installing AWS Fluent Bit as DaemonSet")

	// Create Fluent Bit ConfigMap
	fluentBitConfigMap := fmt.Sprintf(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: fluent-bit-config
  namespace: kube-system
data:
  fluent-bit.conf: |
    [SERVICE]
        Parsers_File    parsers.conf
        HTTP_Server     On
        HTTP_Listen     0.0.0.0
        HTTP_Port       2020

    [INPUT]
        Name                tail
        Tag                 kube.*
        Path                /var/log/containers/*.log
        Parser              docker
        DB                  /var/log/flb_kube.db
        Skip_Long_Lines     On
        Refresh_Interval    10
        Mem_Buf_Limit      5MB

    [FILTER]
        Name                kubernetes
        Match               kube.*
        Kube_URL           https://kubernetes.default.svc:443
        Kube_CA_Path       /var/run/secrets/kubernetes.io/serviceaccount/ca.crt
        Kube_Token_Path    /var/run/secrets/kubernetes.io/serviceaccount/token
        Merge_Log          On
        Merge_Log_Key      log_processed
        K8S-Logging.Parser On
        K8S-Logging.Exclude On
        Use_Kubelet        Off
        Kubelet_Port       10250

    [FILTER]
        Name                grep
        Match               kube.*
        Regex               kubernetes.*%s.*
        Exclude             kubernetes.*kube-system.*

    [OUTPUT]
        Name                cloudwatch
        Match               *
        region              %s
        log_group_name      /aws/eks/%s/op-node
        log_stream_prefix   fluentbit-
        auto_create_group   true
        use_put_log_events true
        log_retention_days  30

  parsers.conf: |
    [PARSER]
        Name        docker
        Format      json
        Time_Key    time
        Time_Format %%Y-%%m-%%dT%%H:%%M:%%S.%%L
        Time_Keep   On
`, namespace, t.deployConfig.AWS.Region, namespace)

	// Apply Fluent Bit ConfigMap
	logger.Info("Creating Fluent Bit ConfigMap")
	if err := t.applyManifest(ctx, fluentBitConfigMap); err != nil {
		return fmt.Errorf("failed to create Fluent Bit ConfigMap: %w", err)
	}

	// Create Fluent Bit DaemonSet
	fluentBitDaemonSet := fmt.Sprintf(`
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: fluent-bit
  namespace: kube-system
  labels:
    app: fluent-bit
spec:
  selector:
    matchLabels:
      app: fluent-bit
  template:
    metadata:
      labels:
        app: fluent-bit
    spec:
      serviceAccountName: fluent-bit
      containers:
      - name: fluent-bit
        image: public.ecr.aws/aws-observability/aws-for-fluent-bit:stable
        imagePullPolicy: Always
        env:
        - name: AWS_REGION
          value: "%s"
        - name: AWS_ACCESS_KEY_ID
          value: "%s"
        - name: AWS_SECRET_ACCESS_KEY
          value: "%s"
        volumeMounts:
        - name: varlog
          mountPath: /var/log
        - name: varlibdockercontainers
          mountPath: /var/lib/docker/containers
          readOnly: true
        - name: fluentbitconfig
          mountPath: /fluent-bit/etc/
        - name: flb-ca-cert
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
          readOnly: true
      terminationGracePeriodSeconds: 10
      volumes:
      - name: varlog
        hostPath:
          path: /var/log
      - name: varlibdockercontainers
        hostPath:
          path: /var/lib/docker/containers
      - name: fluentbitconfig
        configMap:
          name: fluent-bit-config
      - name: flb-ca-cert
        projected:
          defaultMode: 420
          sources:
          - serviceAccountToken:
              expirationSeconds: 3607
              path: token
          - configMap:
              items:
              - key: ca.crt
                path: ca.crt
              name: kube-root-ca.crt
`, t.deployConfig.AWS.Region, t.deployConfig.AWS.AccessKey, t.deployConfig.AWS.SecretKey)

	// Apply Fluent Bit DaemonSet
	logger.Info("Creating Fluent Bit DaemonSet")
	if err := t.applyManifest(ctx, fluentBitDaemonSet); err != nil {
		return fmt.Errorf("failed to create Fluent Bit DaemonSet: %w", err)
	}

	// Create ServiceAccount
	serviceAccount := `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: fluent-bit
  namespace: kube-system
`

	// Apply ServiceAccount
	logger.Info("Creating Fluent Bit ServiceAccount")
	if err := t.applyManifest(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to create Fluent Bit ServiceAccount: %w", err)
	}

	logger.Info("AWS Fluent Bit DaemonSet installed successfully")
	return nil
}

// applyManifest applies a Kubernetes manifest
func (t *ThanosStack) applyManifest(ctx context.Context, manifest string) error {
	// Write manifest to temporary file
	tmpFile, err := os.CreateTemp("", "k8s-manifest-*.yaml")
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
		return fmt.Errorf("failed to apply manifest: %w, output: %s", err, out)
	}

	return nil
}

// generatePrometheusStorageSpec creates Prometheus storage specification
func (t *ThanosStack) generatePrometheusStorageSpec(config *types.MonitoringConfig) map[string]interface{} {
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
func (t *ThanosStack) cleanupExistingMonitoringStorage(ctx context.Context, config *types.MonitoringConfig) error {
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
func (t *ThanosStack) generateStaticPVManifest(component string, config *types.MonitoringConfig, size string, timestamp string) string {
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

	_, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
	if err != nil {
		return fmt.Errorf("failed to apply PV manifest: %w", err)
	}

	return nil
}

// generateStaticPVCManifest generates PVC manifest
func (t *ThanosStack) generateStaticPVCManifest(component string, config *types.MonitoringConfig, size string, timestamp string) string {
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
			fmt.Printf("⚠️  Warning: Failed to delete ConfigMap %s: %v\n", configMapName, err)
		} else {
			fmt.Printf("✅ Deleted existing ConfigMap: %s\n", configMapName)
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

		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "prometheusrule", ruleName, "-n", config.Namespace)
		if err != nil {
			fmt.Printf("⚠️  Failed to delete Alerting Rule %s: %v\n", ruleName, err)
			// Continue with other rules even if one fails
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
        description: "OP Batcher balance is {{ $value }} ETH, below 0.01 ETH threshold"
    
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
        description: "OP Proposer balance is {{ $value }} ETH, below 0.01 ETH threshold"
    
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
        description: "No new blocks have been produced for more than 1 minute"
    
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
        description: "Pod {{ $labels.pod }} CPU usage has been above 80%% for more than 2 minutes"
    
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
        description: "Pod {{ $labels.pod }} memory usage has been above 80%% for more than 2 minutes"
    
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
        description: "Pod {{ $labels.pod }} in namespace {{ $labels.namespace }} has been restarting frequently for more than 2 minutes"
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
