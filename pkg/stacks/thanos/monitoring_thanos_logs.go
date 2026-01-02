package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// Installing monitoring thanos logs stack
func (t *ThanosStack) InstallMonitoringThanosLogsStack(ctx context.Context, config *types.ThanosLogsConfig) (string, error) {
	logger := t.getLogger()

	if t.deployConfig.K8s == nil {
		logger.Error("K8s configuration is not set. Please run the deploy command first")
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	// Ensure Grafana is already installed
	logger.Info("Checking if Grafana is installed...")
	grafanaInstalled, err := t.isGrafanaInstalled(ctx)
	if err != nil {
		logger.Errorw("Failed to check if Grafana is installed", "err", err)
		return "", fmt.Errorf("failed to check if Grafana is installed: %w", err)
	}

	if !grafanaInstalled {
		logger.Info("Grafana is not installed. Installing monitoring plugin...")
		if err := t.installMonitoringWithDefaults(ctx, config.GrafanaPassword); err != nil {
			logger.Errorw("Failed to install monitoring plugin", "err", err)
			return "", fmt.Errorf("failed to install monitoring plugin: %w", err)
		}

		// Wait a bit for Grafana to be ready
		logger.Info("Waiting for Grafana to be ready...")
		time.Sleep(10 * time.Second)
	}

	logger.Info("✅ Grafana is installed and ready. Proceeding with Thanos logs stack installation...")

	// Check if Thanos logs stack is already installed
	logger.Info("Checking if Thanos logs stack is already installed...")
	existingReleases, err := utils.FilterHelmReleases(ctx, config.Namespace, "thanos-logs")
	if err != nil {
		logger.Errorw("Failed to check if Loki is installed", "err", err)
		return "", err
	}

	if len(existingReleases) > 0 {
		logger.Info("Thanos logs is already installed. Skipping installation.")
		return "Thanos logs is already installed", nil
	}

	// Ensure namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		logger.Errorw("Failed to ensure namespace exists", "err", err)
		return "", err
	}

	// Create S3 buckets for Thanos logs storage
	logger.Info("Creating S3 buckets for Thanos logs storage...")
	if err := t.createThanosLogsS3Buckets(ctx, config); err != nil {
		logger.Errorw("Failed to create Thanos logs S3 buckets", "err", err)
		return "", fmt.Errorf("failed to create Thanos logs S3 buckets: %w", err)
	}

	// Get Loki chart path and values file
	chartsPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/loki", t.deploymentPath)
	valuesFilePath := fmt.Sprintf("%s/values.yaml", chartsPath)

	// Check if required chart is present in tokamak-thanos-stack directory
	if _, err := os.Stat(valuesFilePath); err != nil {
		logger.Errorw("Loki values.yaml not found", "path", valuesFilePath, "err", err)
		return "", fmt.Errorf("Loki values.yaml not found at %s: %w", valuesFilePath, err)
	}

	// Update Thanos logs Helm values file
	logger.Info("Updating Thanos logs Helm values file...")
	if err := t.updateThanosLogsValuesFile(config, valuesFilePath); err != nil {
		logger.Errorw("Failed to update Thanos logs values file", "err", err)
		return "", fmt.Errorf("failed to update Thanos logs values file: %w", err)
	}

	// Install Loki via Helm
	logger.Infow("Installing Loki via Helm", "release", config.HelmReleaseName, "namespace", config.Namespace)
	installCmd := []string{
		"upgrade", "--install",
		config.HelmReleaseName,
		chartsPath,
		"--values", valuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
		"--timeout", "15m",
		"--wait",
		"--wait-for-jobs",
	}
	out, err := utils.ExecuteCommand(ctx, "helm", installCmd...)
	if err != nil {
		logger.Errorw("Failed to install Loki", "err", err, "output", out)
		return "", fmt.Errorf("failed to install Loki: %w", err)
	}

	logger.Info("✅ Loki installed successfully")

	// Wait for Loki to be ready
	logger.Info("Waiting for Loki to be ready...")
	time.Sleep(10 * time.Second)

	// Configure Loki datasource in Grafana
	logger.Info("Configuring Loki datasource in Grafana...")
	if err := t.configureLokiDatasourceInGrafana(ctx, config); err != nil {
		logger.Warnw("Failed to configure Loki datasource in Grafana", "err", err)
		// Don't fail the installation if datasource configuration fails
		logger.Info("Thanos logs installation completed, but datasource configuration failed. You may need to configure it manually in Grafana.")
	} else {
		logger.Info("✅ Loki datasource configured in Grafana")
	}

	// Install Alloy helm chart after Loki is installed
	logger.Info("Installing Alloy helm chart...")
	if err := t.installAlloy(ctx, config); err != nil {
		logger.Warnw("Failed to install Alloy helm chart", "err", err)
		// Don't fail the installation if Alloy installation fails
		logger.Info("Thanos logs installation completed, but Alloy installation failed. You may need to install it manually.")
	} else {
		logger.Info("✅ Alloy installed successfully")
	}

	logger.Info("🎉 Thanos logs stack installation completed successfully!")
	return "Thanos logs stack installed successfully", nil
}

// isGrafanaInstalled checks if Grafana is installed by checking for Grafana pod in monitoring namespace
func (t *ThanosStack) isGrafanaInstalled(ctx context.Context) (bool, error) {
	logger := t.getLogger()

	// First check if monitoring namespace exists
	exists, err := utils.CheckNamespaceExists(ctx, constants.MonitoringNamespace)
	if err != nil {
		return false, fmt.Errorf("failed to check monitoring namespace: %w", err)
	}
	if !exists {
		logger.Info("Monitoring namespace does not exist, Grafana is not installed")
		return false, nil
	}

	// Check if Grafana pod exists
	grafanaPods, err := utils.GetPodNamesByLabel(ctx, constants.MonitoringNamespace, "app.kubernetes.io/name=grafana")
	if err != nil {
		logger.Infow("Failed to check Grafana pod, assuming not installed", "err", err)
		return false, nil
	}

	if len(grafanaPods) == 0 {
		logger.Info("No Grafana pod found, Grafana is not installed")
		return false, nil
	}

	logger.Info("Grafana is already installed")
	return true, nil
}

// installMonitoringWithDefaults installs monitoring with default configuration
func (t *ThanosStack) installMonitoringWithDefaults(ctx context.Context, grafanaPassword string) error {
	logger := t.getLogger()
	logger.Info("Grafana is not installed. Installing monitoring plugin with default configuration...")

	// Create default AlertManager config (disabled)
	defaultAlertManagerConfig := types.AlertManagerConfig{
		Telegram: types.TelegramConfig{
			Enabled: false,
		},
		Email: types.EmailConfig{
			Enabled: false,
		},
	}

	// Get monitoring configuration
	monitoringConfig, err := t.GetMonitoringConfig(ctx, grafanaPassword, defaultAlertManagerConfig, false)
	if err != nil {
		return fmt.Errorf("failed to get monitoring configuration: %w", err)
	}

	// Install monitoring
	_, err = t.InstallMonitoring(ctx, monitoringConfig)
	if err != nil {
		return fmt.Errorf("failed to install monitoring: %w", err)
	}

	logger.Info("✅ Monitoring plugin installed successfully")
	return nil
}

// GetMonitoringThanosLogsConfig gathers all required configuration for Monitoring Thanos Logs installation
func (t *ThanosStack) GetMonitoringThanosLogsConfig(ctx context.Context, grafanaPassword string) (*types.ThanosLogsConfig, error) {
	if t.deployConfig == nil {
		return nil, fmt.Errorf("deploy configuration is not initialized")
	}

	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")
	helmReleaseName := fmt.Sprintf("thanos-logs-%d", time.Now().Unix())

	if t.deployConfig.K8s == nil {
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	config := &types.ThanosLogsConfig{
		Namespace:       constants.MonitoringThanosLogsNamespace,
		ChainName:       chainName,
		HelmReleaseName: helmReleaseName,
		GrafanaPassword: grafanaPassword,
	}

	return config, nil
}

// updateThanosLogsValuesFile updates the Thanos logs Helm values file
func (t *ThanosStack) updateThanosLogsValuesFile(config *types.ThanosLogsConfig, valuesFilePath string) error {
	if t.deployConfig == nil || t.deployConfig.AWS == nil {
		return fmt.Errorf("deploy configuration is not properly initialized")
	}

	thanosLogsHelmValues := types.ThanosLogsHelmValues{
		NameOverride:     config.HelmReleaseName,
		FullnameOverride: config.HelmReleaseName,
	}

	// Configure storage bucket names
	thanosLogsHelmValues.Storage.BucketNames = map[string]interface{}{
		"chunks": fmt.Sprintf("%s-thanos-logs-chunks", config.ChainName),
		"ruler":  fmt.Sprintf("%s-thanos-logs-ruler", config.ChainName),
		"admin":  fmt.Sprintf("%s-thanos-logs-admin", config.ChainName),
	}

	// Configure S3 storage
	thanosLogsHelmValues.Storage.S3.Endpoint = fmt.Sprintf("https://s3.%s.amazonaws.com", t.deployConfig.AWS.Region)
	thanosLogsHelmValues.Storage.S3.Region = t.deployConfig.AWS.Region
	thanosLogsHelmValues.Storage.S3.SecretAccessKey = t.deployConfig.AWS.SecretKey
	thanosLogsHelmValues.Storage.S3.AccessKeyId = t.deployConfig.AWS.AccessKey

	// Update YAML fields using UpdateYAMLField utility
	if err := utils.UpdateYAMLField(valuesFilePath, "nameOverride", thanosLogsHelmValues.NameOverride); err != nil {
		return fmt.Errorf("failed to update nameOverride: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "fullnameOverride", thanosLogsHelmValues.FullnameOverride); err != nil {
		return fmt.Errorf("failed to update fullnameOverride: %w", err)
	}

	// Update bucket names - Helm chart expects loki.storage.bucketNames.*
	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.bucketNames", thanosLogsHelmValues.Storage.BucketNames); err != nil {
		return fmt.Errorf("failed to update loki.storage.bucketNames: %w", err)
	}

	// Update S3 storage config - Helm chart expects loki.storage.s3.*
	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.s3.endpoint", thanosLogsHelmValues.Storage.S3.Endpoint); err != nil {
		return fmt.Errorf("failed to update loki.storage.s3.endpoint: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.s3.region", thanosLogsHelmValues.Storage.S3.Region); err != nil {
		return fmt.Errorf("failed to update loki.storage.s3.region: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.s3.secretAccessKey", thanosLogsHelmValues.Storage.S3.SecretAccessKey); err != nil {
		return fmt.Errorf("failed to update loki.storage.s3.secretAccessKey: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.s3.accessKeyId", thanosLogsHelmValues.Storage.S3.AccessKeyId); err != nil {
		return fmt.Errorf("failed to update loki.storage.s3.accessKeyId: %w", err)
	}

	// Set storage type to s3 - required by Helm chart
	if err := utils.UpdateYAMLField(valuesFilePath, "loki.storage.type", "s3"); err != nil {
		return fmt.Errorf("failed to update loki.storage.type: %w", err)
	}

	return nil
}

// createThanosLogsS3Buckets creates the required S3 buckets for Thanos logs storage
func (t *ThanosStack) createThanosLogsS3Buckets(ctx context.Context, config *types.ThanosLogsConfig) error {
	logger := t.getLogger()

	if t.deployConfig == nil || t.deployConfig.AWS == nil {
		return fmt.Errorf("deploy configuration is not properly initialized")
	}

	region := t.deployConfig.AWS.Region
	buckets := []string{
		fmt.Sprintf("%s-thanos-logs-chunks", config.ChainName),
		fmt.Sprintf("%s-thanos-logs-ruler", config.ChainName),
		fmt.Sprintf("%s-thanos-logs-admin", config.ChainName),
	}

	for _, bucketName := range buckets {
		// Check if bucket already exists
		checkCmd := []string{
			"s3api", "head-bucket",
			"--bucket", bucketName,
			"--region", region,
		}
		_, err := utils.ExecuteCommand(ctx, "aws", checkCmd...)
		if err == nil {
			logger.Infow("S3 bucket already exists", "bucket", bucketName)
			continue
		}

		// Create bucket if it doesn't exist
		logger.Infow("Creating S3 bucket", "bucket", bucketName, "region", region)
		createCmd := []string{
			"s3api", "create-bucket",
			"--bucket", bucketName,
		}

		// For regions outside us-east-1, we need to specify location constraint
		if region != "us-east-1" {
			locationConstraint := fmt.Sprintf(`{"LocationConstraint":"%s"}`, region)
			createCmd = append(createCmd, "--create-bucket-configuration", locationConstraint)
		}
		// For us-east-1, no location constraint is needed

		output, err := utils.ExecuteCommand(ctx, "aws", createCmd...)
		if err != nil {
			// Check if bucket was created by another process
			if strings.Contains(output, "BucketAlreadyOwnedByYou") || strings.Contains(output, "BucketAlreadyExists") {
				logger.Infow("Bucket already exists (created by another process)", "bucket", bucketName)
				continue
			}
			logger.Errorw("Failed to create S3 bucket", "bucket", bucketName, "err", err, "output", output)
			return fmt.Errorf("failed to create S3 bucket %s: %w", bucketName, err)
		}

		logger.Infow("✅ Successfully created S3 bucket", "bucket", bucketName)
	}

	return nil
}

// configureLokiDatasourceInGrafana adds Loki as a datasource in Grafana
func (t *ThanosStack) configureLokiDatasourceInGrafana(ctx context.Context, config *types.ThanosLogsConfig) error {
	logger := t.getLogger()

	// Get Grafana pod name using GetPodNamesByLabel
	grafanaPods, err := utils.GetPodNamesByLabel(ctx, constants.MonitoringNamespace, "app.kubernetes.io/name=grafana")
	if err != nil || len(grafanaPods) == 0 {
		return fmt.Errorf("failed to find Grafana pod: %w", err)
	}

	grafanaPodName := grafanaPods[0]

	// Get Loki service URL
	lokiServiceName := fmt.Sprintf("%s-gateway", config.HelmReleaseName)
	lokiURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:80", lokiServiceName, config.Namespace)

	grafanaPassword := config.GrafanaPassword
	if grafanaPassword == "" {
		return fmt.Errorf("Grafana password is required in config")
	}

	logger.Infow("Adding Loki datasource to Grafana", "lokiURL", lokiURL, "grafanaPod", grafanaPodName)

	// Use localhost:3000 when executing curl inside the Grafana pod
	grafanaInternalURL := "http://localhost:3000"

	// First check if Loki datasource already exists by checking the specific datasource endpoint
	checkScript := fmt.Sprintf(`response=$(curl -s -w "\nHTTP_CODE:%%{http_code}\n" -u admin:%s "%s/api/datasources/name/Loki"); http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2); if [ "$http_code" = "200" ]; then echo "yes"; else echo "no"; fi`, grafanaPassword, grafanaInternalURL)

	checkOutput, err := utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, grafanaPodName, "-c", "grafana", "--", "sh", "-c", checkScript)
	logger.Infow("Check datasource exists", "output", checkOutput, "err", err)

	if err == nil && strings.Contains(strings.TrimSpace(checkOutput), "yes") {
		logger.Info("Loki datasource already exists in Grafana")
		return nil
	}

	type GrafanaDatasourceConfig struct {
		Name           string                 `json:"name"`
		Type           string                 `json:"type"`
		Access         string                 `json:"access"`
		URL            string                 `json:"url"`
		IsDefault      bool                   `json:"isDefault"`
		JsonData       map[string]interface{} `json:"jsonData"`
		SecureJsonData map[string]interface{} `json:"secureJsonData"`
	}

	jsonData := map[string]interface{}{
		"maxLines":        1000,
		"httpHeaderName1": "X-Scope-OrgID",
	}

	secureJsonData := map[string]interface{}{
		"httpHeaderValue1": "default",
	}

	datasourceConfigStruct := GrafanaDatasourceConfig{
		Name:           "Loki",
		Type:           "loki",
		Access:         "proxy",
		URL:            lokiURL,
		IsDefault:      false,
		JsonData:       jsonData,
		SecureJsonData: secureJsonData,
	}

	// Marshal to JSON
	datasourceConfigJSON, err := json.Marshal(datasourceConfigStruct)
	if err != nil {
		return fmt.Errorf("failed to marshal datasource config: %w", err)
	}

	// Escape single quotes for shell command (replace ' with '\'')
	datasourceConfigEscaped := strings.ReplaceAll(string(datasourceConfigJSON), "'", "'\\''")

	// Add datasource via Grafana API using localhost inside the pod
	addDatasourceScript := fmt.Sprintf(`curl -s -w "\nHTTP_CODE:%%{http_code}\n" -X POST "%s/api/datasources" -H "Content-Type: application/json" -u admin:%s -d '%s'`, grafanaInternalURL, grafanaPassword, datasourceConfigEscaped)

	// Execute the script in Grafana pod
	output, err := utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, grafanaPodName, "-c", "grafana", "--", "sh", "-c", addDatasourceScript)

	// Check HTTP status code in output
	if strings.Contains(output, "HTTP_CODE:200") || strings.Contains(output, "HTTP_CODE:201") {
		logger.Info("Successfully added Loki datasource to Grafana")
		return nil
	}

	// Check for duplicate datasource error (409 Conflict or specific message)
	if strings.Contains(output, "HTTP_CODE:409") || (strings.Contains(output, "message") && strings.Contains(output, "Data source with the same name already exists")) {
		logger.Info("Loki datasource already exists in Grafana")
		return nil
	}

	// Check for other errors
	if err != nil {
		logger.Warnw("Failed to add Loki datasource via API", "err", err, "output", output)
		return fmt.Errorf("failed to add Loki datasource: %w, output: %s", err, output)
	}

	if strings.Contains(output, "error") || strings.Contains(output, "Error") || strings.Contains(output, "HTTP_CODE:4") || strings.Contains(output, "HTTP_CODE:5") {
		logger.Warnw("Error response from Grafana API", "output", output)
		return fmt.Errorf("Grafana API returned an error: %s", output)
	}

	logger.Info("Successfully added Loki datasource to Grafana")
	return nil
}

// UninstallMonitoringThanosLogsStack removes Thanos logs stack
func (t *ThanosStack) UninstallMonitoringThanosLogsStack(ctx context.Context, grafanaPassword string) error {
	logger := t.getLogger()

	if t.deployConfig.K8s == nil {
		logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	thanosLogsNamespace := constants.MonitoringThanosLogsNamespace
	logger.Infow("Uninstalling Thanos logs stack", "namespace", thanosLogsNamespace)

	// Check if namespace exists first
	exists, err := utils.CheckNamespaceExists(ctx, thanosLogsNamespace)
	if err != nil {
		logger.Errorw("Failed to check Thanos logs namespace existence", "err", err)
		return err
	}

	if !exists {
		logger.Info("Thanos logs namespace does not exist, skipping uninstallation")
		return nil
	}

	// Uninstall Helm releases (both thanos-logs and alloy)
	thanosLogsReleases, err := utils.FilterHelmReleases(ctx, thanosLogsNamespace, "thanos-logs")
	if err != nil {
		logger.Errorw("Error filtering helm releases for thanos-logs", "err", err)
		return err
	}

	alloyReleases, err := utils.FilterHelmReleases(ctx, thanosLogsNamespace, "alloy")
	if err != nil {
		logger.Errorw("Error filtering helm releases for alloy", "err", err)
		return err
	}

	// Combine both release lists
	allReleases := append(thanosLogsReleases, alloyReleases...)

	if len(allReleases) == 0 {
		logger.Info("No Thanos logs or Alloy Helm releases found, skipping uninstallation")
		return nil
	}

	for _, release := range allReleases {
		logger.Infow("Uninstalling Helm release", "release", release, "namespace", thanosLogsNamespace)
		_, err = utils.ExecuteCommand(ctx, "helm", []string{
			"uninstall",
			release,
			"--namespace",
			thanosLogsNamespace,
		}...)
		if err != nil {
			logger.Errorw("❌ Error uninstalling helm chart", "err", err, "release", release)
			return err
		}
		logger.Infow("✅ Successfully uninstalled Helm release", "release", release)
	}

	// Optionally remove Loki datasource from Grafana (warn but don't fail)
	logger.Info("Attempting to remove Loki datasource from Grafana...")
	if err := t.removeLokiDatasourceFromGrafana(ctx, grafanaPassword); err != nil {
		logger.Warnw("Failed to remove Loki datasource from Grafana (non-fatal)", "err", err)
		// Don't fail the uninstall if datasource removal fails
	} else {
		logger.Info("✅ Loki datasource removed from Grafana")
	}

	// Prompt user to delete S3 buckets
	fmt.Print("Do you want to delete the S3 buckets used by Thanos logs? (y/n): ")
	deleteBuckets, err := scanner.ScanBool(false)
	if err != nil {
		logger.Warnw("Failed to read user input for S3 bucket deletion", "err", err)
		logger.Info("Skipping S3 bucket deletion")
	} else if deleteBuckets {
		if err := t.deleteThanosLogsS3Buckets(ctx); err != nil {
			logger.Warnw("Failed to delete Thanos logs S3 buckets", "err", err)
			// Don't fail the uninstall if bucket deletion fails
		} else {
			logger.Info("✅ Thanos logs S3 buckets deleted successfully")
		}
	}

	logger.Info("✅ Uninstall of Thanos logs stack completed successfully!")
	return nil
}

// removeLokiDatasourceFromGrafana removes the Loki datasource from Grafana
func (t *ThanosStack) removeLokiDatasourceFromGrafana(ctx context.Context, grafanaPassword string) error {
	logger := t.getLogger()

	// Skip if password is not provided
	if grafanaPassword == "" {
		logger.Info("Grafana password not provided, skipping datasource removal")
		return nil
	}

	// Get Grafana pod name
	grafanaPods, err := utils.GetPodNamesByLabel(ctx, constants.MonitoringNamespace, "app.kubernetes.io/name=grafana")
	if err != nil || len(grafanaPods) == 0 {
		logger.Info("Grafana pod not found, skipping datasource removal")
		return nil
	}

	grafanaPodName := grafanaPods[0]
	grafanaInternalURL := "http://localhost:3000"

	logger.Infow("Removing Loki datasource from Grafana", "grafanaPod", grafanaPodName)

	// First check if datasource exists and get its ID
	checkScript := fmt.Sprintf(`response=$(curl -s -w "\nHTTP_CODE:%%{http_code}\n" -u admin:%s "%s/api/datasources/name/Loki"); http_code=$(echo "$response" | grep "HTTP_CODE:" | cut -d: -f2); if [ "$http_code" = "200" ]; then echo "$response" | grep -v "HTTP_CODE" | grep -o '"id":[0-9]*' | grep -o '[0-9]*' | head -1; fi`, grafanaPassword, grafanaInternalURL)

	dsIDOutput, err := utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, grafanaPodName, "-c", "grafana", "--", "sh", "-c", checkScript)
	if err != nil {
		logger.Warnw("Failed to check if Loki datasource exists", "err", err)
		return nil // Don't fail if check fails
	}

	dsID := strings.TrimSpace(dsIDOutput)
	if dsID == "" {
		logger.Info("Loki datasource not found in Grafana (may have been already removed)")
		return nil
	}

	// Delete the datasource by ID
	deleteScript := fmt.Sprintf(`curl -s -w "\nHTTP_CODE:%%{http_code}\n" -X DELETE -u admin:%s "%s/api/datasources/%s"`, grafanaPassword, grafanaInternalURL, dsID)

	output, err := utils.ExecuteCommand(ctx, "kubectl", "exec", "-n", constants.MonitoringNamespace, grafanaPodName, "-c", "grafana", "--", "sh", "-c", deleteScript)
	logger.Infow("Grafana API response for datasource deletion", "output", output, "err", err)

	// Check if deletion was successful (200 OK or 404 Not Found)
	if strings.Contains(output, "HTTP_CODE:200") || strings.Contains(output, "HTTP_CODE:404") {
		logger.Info("Loki datasource removed from Grafana")
		return nil
	}

	// If deletion failed, log warning but don't fail the uninstall
	logger.Warnw("Failed to remove Loki datasource from Grafana", "output", output)
	return nil
}

// deleteThanosLogsS3Buckets deletes the S3 buckets used by Thanos logs
func (t *ThanosStack) deleteThanosLogsS3Buckets(ctx context.Context) error {
	logger := t.getLogger()

	if t.deployConfig == nil || t.deployConfig.AWS == nil {
		return fmt.Errorf("deploy configuration is not properly initialized")
	}

	if t.deployConfig.ChainName == "" {
		logger.Warn("Chain name not found in deploy config, skipping S3 bucket deletion")
		return nil
	}

	region := t.deployConfig.AWS.Region
	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")

	buckets := []string{
		fmt.Sprintf("%s-thanos-logs-chunks", chainName),
		fmt.Sprintf("%s-thanos-logs-ruler", chainName),
		fmt.Sprintf("%s-thanos-logs-admin", chainName),
	}

	deletedCount := 0
	for _, bucketName := range buckets {
		// Check if bucket exists before attempting to delete
		checkCmd := []string{
			"s3api", "head-bucket",
			"--bucket", bucketName,
			"--region", region,
		}
		_, err := utils.ExecuteCommand(ctx, "aws", checkCmd...)
		if err != nil {
			logger.Infow("S3 bucket does not exist, skipping", "bucket", bucketName)
			continue
		}

		// Delete bucket (first remove all objects, then delete bucket)
		logger.Infow("Deleting S3 bucket", "bucket", bucketName)

		// First, delete all objects in the bucket
		deleteObjectsCmd := []string{
			"s3", "rm",
			"s3://" + bucketName,
			"--recursive",
		}
		_, err = utils.ExecuteCommand(ctx, "aws", deleteObjectsCmd...)
		if err != nil {
			logger.Warnw("Failed to delete objects from bucket (may be empty)", "bucket", bucketName, "err", err)
			// Continue to try deleting the bucket anyway
		}

		// Delete the bucket itself
		deleteBucketCmd := []string{
			"s3api", "delete-bucket",
			"--bucket", bucketName,
			"--region", region,
		}
		output, err := utils.ExecuteCommand(ctx, "aws", deleteBucketCmd...)
		if err != nil {
			logger.Warnw("Failed to delete S3 bucket", "bucket", bucketName, "err", err, "output", output)
			continue
		}

		deletedCount++
		logger.Infow("✅ Successfully deleted S3 bucket", "bucket", bucketName)
	}

	if deletedCount > 0 {
		logger.Infow("✅ Deleted S3 buckets", "count", deletedCount)
	} else {
		logger.Info("No S3 buckets found to delete")
	}

	return nil
}

// installAlloy installs the grafana alloy helm chart
func (t *ThanosStack) installAlloy(ctx context.Context, config *types.ThanosLogsConfig) error {
	logger := t.getLogger()

	// Check if Alloy is already installed
	existingReleases, err := utils.FilterHelmReleases(ctx, config.Namespace, "alloy")
	if err != nil {
		logger.Errorw("Failed to check if Alloy is installed", "err", err)
		return fmt.Errorf("failed to check if Alloy is installed: %w", err)
	}

	if len(existingReleases) > 0 {
		logger.Info("Alloy is already installed. Skipping installation.")
		return nil
	}

	// Get Alloy chart path
	alloyChartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/alloy", t.deploymentPath)

	// Check if chart exists
	if _, err := os.Stat(alloyChartPath); err != nil {
		logger.Errorw("Alloy chart not found", "path", alloyChartPath, "err", err)
		return fmt.Errorf("Alloy chart not found at %s: %w", alloyChartPath, err)
	}

	// Generate helm release name
	helmReleaseName := fmt.Sprintf("alloy-%d", time.Now().Unix())

	// Construct Loki service name
	lokiServiceName := fmt.Sprintf("%s-gateway.%s.svc.cluster.local:80", config.HelmReleaseName, config.Namespace)

	// Install Alloy via Helm
	logger.Infow("Installing Alloy via Helm", "release", helmReleaseName, "namespace", config.Namespace)
	installCmd := []string{
		"upgrade", "--install",
		helmReleaseName,
		alloyChartPath,
		"--namespace", config.Namespace,
		"--create-namespace",
		"--set", fmt.Sprintf("loki.serviceName=%s", lokiServiceName),
		"--timeout", "15m",
		"--wait",
		"--wait-for-jobs",
	}

	output, err := utils.ExecuteCommand(ctx, "helm", installCmd...)
	if err != nil {
		logger.Errorw("Failed to install Alloy", "err", err, "output", output)
		return fmt.Errorf("failed to install Alloy: %w", err)
	}

	logger.Info("✅ Alloy installed successfully")
	return nil
}
