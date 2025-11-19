package thanos

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	socketio "github.com/maldikhan/go.socket.io/socket.io/v5/client"
	"github.com/maldikhan/go.socket.io/socket.io/v5/client/emit"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// Installing uptime-service
func (t *ThanosStack) InstallUptimeService(ctx context.Context, config *types.UptimeServiceConfig) (string, error) {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	//Checking if uptime-service is already running
	uptimeServicePods, err := utils.GetPodNamesByLabel(ctx, config.Namespace, "uptime-service")
	if err != nil {
		t.logger.Error("Failed to get uptime-service pods", "err", err)
		return "", err
	}

	var uptimeURL string
	helmReleaseName := fmt.Sprintf("%s-%d", "uptime-service", time.Now().Unix())

	if len(uptimeServicePods) > 0 {
		t.logger.Info("Uptime Service is already running")
		// Get existing Helm release name from running installation
		existingReleases, err := utils.FilterHelmReleases(ctx, config.Namespace, "uptime-service")
		if err != nil || len(existingReleases) == 0 {
			t.logger.Warn("Could not find existing Helm release, trying to get service URL with label selector")
		}
		// Use the first existing release name
		existingReleaseName := existingReleases[0]
		t.logger.Info(fmt.Sprintf("Using existing Helm release: %s", existingReleaseName))

		for {
			k8sServices, err := utils.GetAddressByService(ctx, config.Namespace, existingReleaseName)
			if err != nil {
				t.logger.Error("Error retrieving service addresses", "err", err, "details", k8sServices)
				return "", err
			}

			if len(k8sServices) > 0 {
				uptimeURL = "http://" + k8sServices[0]

				break
			}
			time.Sleep(15 * time.Second)
		}
		t.logger.Infof("✅ uptime-service is already running. You can access it at: %s", uptimeURL)
		return uptimeURL, nil
	}

	// Set helmReleaseName for new installation
	config.HelmReleaseName = helmReleaseName

	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		t.logger.Error("Error cloning repository", "err", err)
		return "", err
	}

	t.logger.Info("🚀 Installing uptime service...")

	// Ensure uptime-service namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		t.logger.Errorw("Failed to ensure uptime-service namespace exists", "err", err)
		return "", fmt.Errorf("failed to ensure uptime-service namespace exists: %w", err)
	}

	// Deploy infrastructure if persistence is enabled
	if config.IsPersistenceEnable {
		t.logger.Info("Deploying uptime-service infrastructure (persistence enabled)")
		if err := t.deployUptimeServiceInfrastructure(ctx, config); err != nil {
			t.logger.Errorw("Failed to deploy uptime-service infrastructure", "err", err)
			return "", fmt.Errorf("failed to deploy uptime-service infrastructure: %w", err)
		}

		// Wait for PVC to be bound before installing Helm chart
		pvcName := fmt.Sprintf("%s-pvc", config.HelmReleaseName)
		t.logger.Info(fmt.Sprintf("⏳ Waiting for PVC to be bound before installing Helm chart: %s", pvcName))
		if err := t.waitForPVCBound(ctx, config.Namespace, pvcName); err != nil {
			t.logger.Errorw("Failed to wait for PVC to be bound", "err", err, "pvc", pvcName)
			return "", fmt.Errorf("failed to wait for PVC to be bound: %w", err)
		}
		t.logger.Info("✅ PVC is bound and ready")
	}

	// Patch the existing values.yaml file
	valuesFilePath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/uptime-service/values.yaml", t.deploymentPath)
	pvcName := fmt.Sprintf("%s-pvc", config.HelmReleaseName)

	// Check if required chart is present in tokamak-thanos-stack directory
	if _, err := os.Stat(valuesFilePath); err != nil {
		t.logger.Errorw("Uptime-service values.yaml not found", "path", valuesFilePath, "err", err)
		return "", fmt.Errorf("❌ uptime-service values.yaml not found at %s: %w", valuesFilePath, err)
	}

	// Setting up the helm values
	helmValues := types.UptimeServiceHelmValues{
		NameOverride:     "uptime-service",
		FullnameOverride: helmReleaseName,
	}

	helmValues.Service.Type = "LoadBalancer"
	helmValues.Service.Port = 80
	helmValues.Service.TargetPort = 3001
	helmValues.Service.Annotations = map[string]interface{}{
		"service.beta.kubernetes.io/aws-load-balancer-type":   "nlb-ip",
		"service.beta.kubernetes.io/aws-load-balancer-scheme": "internet-facing",
	}

	helmValues.PodLabels = map[string]interface{}{
		"uptime-service": "",
	}

	helmValues.Volume.Enabled = true
	helmValues.Volume.ExistingClaim = pvcName
	helmValues.Volume.StorageClassName = "efs-sc"
	helmValues.Volume.Size = "4Gi"

	if err := utils.UpdateYAMLField(valuesFilePath, "nameOverride", helmValues.NameOverride); err != nil {
		return "", fmt.Errorf("failed to update nameOverride: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "fullnameOverride", helmValues.FullnameOverride); err != nil {
		return "", fmt.Errorf("failed to update fullnameOverride: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "service.type", helmValues.Service.Type); err != nil {
		return "", fmt.Errorf("failed to update service.type: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "service.port", helmValues.Service.Port); err != nil {
		return "", fmt.Errorf("failed to update service.port: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "service.targetPort", helmValues.Service.TargetPort); err != nil {
		return "", fmt.Errorf("failed to update service.targetPort: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "service.annotations", helmValues.Service.Annotations); err != nil {
		return "", fmt.Errorf("failed to update service.annotations: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "podLabels", helmValues.PodLabels); err != nil {
		return "", fmt.Errorf("failed to update podLabels: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "volume.enabled", helmValues.Volume.Enabled); err != nil {
		return "", fmt.Errorf("failed to update volume.enabled: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "volume.existingClaim", helmValues.Volume.ExistingClaim); err != nil {
		return "", fmt.Errorf("failed to update volume.existingClaim: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "volume.storageClassName", helmValues.Volume.StorageClassName); err != nil {
		return "", fmt.Errorf("failed to update volume.storageClassName: %w", err)
	}

	if err := utils.UpdateYAMLField(valuesFilePath, "volume.size", helmValues.Volume.Size); err != nil {
		return "", fmt.Errorf("failed to update volume.size: %w", err)
	}

	args := []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/uptime-service", t.deploymentPath),
		"--values", valuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
	}

	//Installing helm chart for uptime-service
	output, err := utils.ExecuteCommand(ctx, "helm", args...)

	if err != nil {
		t.logger.Error(
			"❌ Error installing uptime service",
			"err", err,
			"helm_output", output,
		)
		return "", err
	}
	t.logger.Info("✅ Install uptime service successfully")

	//Fetching the service loadbalancer URL
	for {
		k8sServices, err := utils.GetAddressByService(ctx, config.Namespace, helmReleaseName)
		if err != nil {
			t.logger.Error("Error retrieving service addresses", "err", err, "details", k8sServices)
			return "", err
		}

		if len(k8sServices) > 0 {
			uptimeURL = "http://" + k8sServices[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	t.logger.Infof("✅ uptime-service is up and running. You can access it at: %s", uptimeURL)

	// Wait for LoadBalancer to be publicly reachable
	t.logger.Info("⏳ Waiting for LoadBalancer to be publicly reachable...")
	checkInterval := 5 * time.Second

	for {
		if t.isURLReachable(uptimeURL) {
			t.logger.Info("✅ LoadBalancer is publicly reachable, configuring monitors...")
			// Configure default monitors automatically
			if err := t.handleDefaultMonitorSetup(ctx, uptimeURL, config.Namespace); err != nil {
				t.logger.Warn("Failed to configure default monitors, but uptime-service is running", "err", err)
			}
			break
		}
		time.Sleep(checkInterval)
	}

	return uptimeURL, nil

}

// Uninstalling uptime-service
func (t *ThanosStack) UninstallUptimeService(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	uptimeNamespace := constants.UptimeServiceNamespace
	fmt.Printf("namespace while uninstalling : %s ", uptimeNamespace)

	// Check if namespace exists first
	exists, err := utils.CheckNamespaceExists(ctx, uptimeNamespace)
	if err != nil {
		t.logger.Errorw("Failed to check uptime-service namespace existence", "err", err)
		return err
	}

	if !exists {
		t.logger.Info("Uptime-service namespace does not exist, skipping uninstallation")
		return nil
	}

	// Uninstall Helm releases
	releases, err := utils.FilterHelmReleases(ctx, uptimeNamespace, "uptime-service")
	if err != nil {
		t.logger.Error("Error to filter helm releases", "err", err)
		return err
	}

	for _, release := range releases {
		t.logger.Infow("Uninstalling Helm release", "release", release, "namespace", uptimeNamespace)
		_, err = utils.ExecuteCommand(ctx, "helm", []string{
			"uninstall",
			release,
			"--namespace",
			uptimeNamespace,
		}...)
		if err != nil {
			t.logger.Error("❌ Error uninstalling uptime-service helm chart", "err", err)
			return err
		}
	}

	// Delete namespace
	t.logger.Info(fmt.Sprintf("❌ Deleting uptime-service namespace: %s", uptimeNamespace))
	err = t.tryToDeleteK8sNamespace(ctx, uptimeNamespace)
	if err != nil {
		t.logger.Errorw("Failed to delete uptime-service namespace", "err", err, "namespace", uptimeNamespace)
		return err
	}

	// Clean up volumes that might be left behind
	config := &types.UptimeServiceConfig{
		Namespace: uptimeNamespace,
	}
	if err := t.cleanupExistingUptimeServiceStorage(ctx, config); err != nil {
		t.logger.Warnw("❌ Failed to cleanup uptime-service storage", "err", err)
	}

	t.logger.Info("✅ Uninstall of uptime-service successfully!")
	return nil
}

// GetUptimeServiceConfig gathers all required configuration for uptime-service
func (t *ThanosStack) GetUptimeServiceConfig(ctx context.Context) (*types.UptimeServiceConfig, error) {

	if t.deployConfig == nil {
		return nil, fmt.Errorf("deploy configuration is not initialized")
	}

	chainName := strings.ToLower(t.deployConfig.ChainName)
	chainName = strings.ReplaceAll(chainName, " ", "-")

	if t.deployConfig.K8s == nil {
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	efsFileSystemId, err := utils.GetEFSFileSystemId(ctx, chainName, t.deployConfig.AWS.Region)
	if err != nil {
		return nil, fmt.Errorf("error getting EFS filesystem ID: %w", err)
	}

	config := &types.UptimeServiceConfig{
		Namespace:           constants.UptimeServiceNamespace,
		IsPersistenceEnable: true,
		EFSFileSystemId:     efsFileSystemId,
		ChainName:           chainName,
	}

	return config, nil
}

// deployUptimeServiceInfrastructure creates PVs for Static Provisioning
func (t *ThanosStack) deployUptimeServiceInfrastructure(ctx context.Context, config *types.UptimeServiceConfig) error {
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return fmt.Errorf("❌ failed to ensure namespace exists: %w", err)
	}

	// Clean up existing UptimeService PVs and PVCs
	if err := t.cleanupExistingUptimeServiceStorage(ctx, config); err != nil {
		return fmt.Errorf("❌ failed to cleanup existing uptime-service storage: %w", err)
	}

	timestamp, err := utils.GetTimestampFromExistingPV(ctx, config.ChainName)
	if err != nil {
		return fmt.Errorf("❌ failed to get timestamp from existing PV: %w", err)
	}

	// Create PV and PVC using generic functions from utils/storage.go
	uptimePV := utils.GenerateStaticPVManifest("uptime-service", config, "4Gi", timestamp)
	if err := utils.ApplyPVManifest(ctx, t.deploymentPath, "uptime-service", uptimePV, "UptimeService"); err != nil {
		return fmt.Errorf("❌ failed to create uptime-service PV: %w", err)
	}

	uptimePVC := utils.GenerateStaticPVCManifest("uptime-service", config, "4Gi", timestamp)
	if err := utils.ApplyPVCManifest(ctx, t.deploymentPath, "uptime-service", uptimePVC, "UptimeService"); err != nil {
		return fmt.Errorf("failed to create uptime-service PVC: %w", err)
	}
	fmt.Println("✅ Created uptime-service PV and PVC")

	return nil
}

// waitForPVCBound waits for a specific PVC to be bound
func (t *ThanosStack) waitForPVCBound(ctx context.Context, namespace string, pvcName string) error {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		status, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", pvcName, "-n", namespace, "-o", "jsonpath={.status.phase}", "--ignore-not-found=true")
		if err != nil {
			t.logger.Warnw("Error checking PVC status", "err", err, "pvc", pvcName, "attempt", i+1)
		} else if strings.TrimSpace(status) == "Bound" {
			t.logger.Info(fmt.Sprintf("PVC is bound: %s", pvcName))
			return nil
		} else {
			t.logger.Infow("Waiting for PVC to be bound", "pvc", pvcName, "status", strings.TrimSpace(status), "attempt", i+1)
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("PVC %s not bound after %d attempts ", pvcName, maxRetries)
}

// cleanupExistingUptimeServiceStorage removes existing uptime-service PVs and PVCs
func (t *ThanosStack) cleanupExistingUptimeServiceStorage(ctx context.Context, config *types.UptimeServiceConfig) error {
	// Get existing monitoring PVCs
	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", config.Namespace, "--no-headers", "-o", "custom-columns=NAME:.metadata.name")
	if err != nil {
		return fmt.Errorf("failed to get existing PVCs: %w", err)
	}

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

		// Only delete Released PVs (not Bound or Available)
		if status == "Released" && (strings.Contains(pvName, "uptime-service")) {
			// Remove claimRef to allow reuse
			_, err = utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", pvName, "-p", `{"spec":{"claimRef":null}}`, "--type=merge")
			if err == nil {
				deletedPVs++
			}
		}
	}

	return nil
}

// handleDefaultMonitorSetup handles the monitor configuration flow
func (t *ThanosStack) handleDefaultMonitorSetup(ctx context.Context, uptimeURL string, uptimeNamespace string) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set")
	}

	chainNamespace := t.deployConfig.K8s.Namespace
	username := "admin"
	password := "admin@123"

	// Discover services in chain namespace
	services, err := t.discoverServicesForMonitoring(ctx, chainNamespace)
	if err != nil {
		return fmt.Errorf("failed to discover services: %w", err)
	}

	if len(services) == 0 {
		t.logger.Warn("No services found to monitor in chain namespace")
		return nil
	}

	// Port mapping for components
	portMap := map[string]int{
		"op-node":        8545,
		"op-geth":        8545,
		"op-batcher":     7300,
		"op-proposer":    7300,
		"bridge":         3000,
		"block-explorer": 3000,
	}

	// Suppress Socket.IO library's verbose debug logs
	originalLogOutput := log.Writer()
	log.SetOutput(io.Discard)
	defer log.SetOutput(originalLogOutput)

	// Create Socket.IO client
	client, err := socketio.NewClient(
		socketio.WithRawURL(uptimeURL),
	)
	if err != nil {
		log.SetOutput(originalLogOutput) // Restore before returning error
		return fmt.Errorf("failed to create Socket.IO client: %w", err)
	}
	defer client.Close()

	// Helper function to add a monitor (similar to main.go)
	addMonitor := func(componentName, serviceName string, port int) {
		internalURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, chainNamespace, port)
		monitor := map[string]interface{}{
			"name":                 componentName,
			"type":                 "http",
			"url":                  internalURL,
			"method":               "GET",
			"interval":             60,
			"timeout":              48,
			"accepted_statuscodes": []string{"200-299"},
			"active":               true,
		}

		err := client.Emit("add", monitor,
			emit.WithAck(func(response interface{}) {
				if resp, ok := response.(map[string]interface{}); ok {
					if okVal, hasOk := resp["ok"].(bool); hasOk && okVal {
						t.logger.Infof("✅ Monitor added: %s", componentName)
					} else {
						if msg, hasMsg := resp["msg"].(string); hasMsg {
							t.logger.Warnf("❌ Failed to add monitor: %s", msg)
						} else {
							t.logger.Warnf("❌ Failed to add monitor: %v", resp)
						}
					}
				}
			}),
			emit.WithTimeout(10*time.Second, func() {
				t.logger.Warnf("⏰ Monitor %s: acknowledgement timeout", componentName)
			}),
		)

		if err != nil {
			t.logger.Warnf("Error emitting add monitor event: %v", err)
		}
	}

	// Set up event handlers BEFORE connecting
	client.On("connect", func() {
		t.logger.Info("✅ Connected to Uptime Service!")

		// Setup with default credentials (for initial setup/signup)
		t.logger.Info("⏳ Setting up Uptime Service with default credentials...")
		err := client.Emit("setup", username, password,
			emit.WithAck(func(response interface{}) {
				if resp, ok := response.(map[string]interface{}); ok {
					if okVal, hasOk := resp["ok"].(bool); hasOk && okVal {
						t.logger.Info("✅ Setup successful!")

						// Small delay to ensure setup is complete
						time.Sleep(500 * time.Millisecond)

						// Login after signup
						t.logger.Info("🔐 Logging in after setup...")
						loginData := map[string]interface{}{
							"username": username,
							"password": password,
							"token":    "",
						}
						err := client.Emit("login", loginData,
							emit.WithAck(func(response interface{}) {
								if resp, ok := response.(map[string]interface{}); ok {
									if okVal, hasOk := resp["ok"].(bool); hasOk && okVal {
										t.logger.Info("🔐 Login successful!")

										// Add monitors for each discovered service
										t.logger.Info("➕ Adding monitors...")
										monitorsAdded := 0
										monitorsFailed := 0

										for componentName, serviceName := range services {
											port, exists := portMap[componentName]
											if !exists {
												t.logger.Warn("Unknown component, skipping", "component", componentName)
												monitorsFailed++
												continue
											}

											addMonitor(componentName, serviceName, port)
											monitorsAdded++
										}
										t.logger.Infof("✅ Added %d monitors, %d failed", monitorsAdded, monitorsFailed)
									} else {
										msg := "unknown error"
										if msgVal, hasMsg := resp["msg"].(string); hasMsg {
											msg = msgVal
										}
										t.logger.Warnf("❌ Login failed: %s", msg)
									}
								}
							}),
							emit.WithTimeout(10*time.Second, func() {
								t.logger.Warn("⏰ Login acknowledgement timeout")
							}),
						)
						if err != nil {
							t.logger.Warnf("Error emitting login event: %v", err)
						}
					} else {
						msg := "unknown error"
						if msgVal, hasMsg := resp["msg"].(string); hasMsg {
							msg = msgVal
						}
						t.logger.Warnf("❌ Setting Default Credentials failed: %s", msg)
					}
				}
			}),
			emit.WithTimeout(10*time.Second, func() {
				t.logger.Warn("⏰ Setup acknowledgement timeout")
			}),
		)
		if err != nil {
			t.logger.Warnf("Error emitting setup event: %v", err)
		}
	})

	// Connect to the server
	connectCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = client.Connect(connectCtx)
	if err != nil {
		return fmt.Errorf("failed to connect to Uptime Kuma: %w", err)
	}

	// Small delay to ensure connection is established and connect event fires
	time.Sleep(1 * time.Second)

	// Wait for operations to complete (setup/login/monitor addition are async)
	time.Sleep(15 * time.Second)

	t.logger.Info("✅ Monitors configured successfully!")
	return nil
}

// discoverServicesForMonitoring discovers services in the chain namespace
func (t *ThanosStack) discoverServicesForMonitoring(ctx context.Context, chainNamespace string) (map[string]string, error) {
	services := make(map[string]string)

	components := []struct {
		pattern string
		name    string
	}{
		{"op-node", "op-node"},
		{"op-geth", "op-geth"},
		{"op-batcher", "op-batcher"},
		{"op-proposer", "op-proposer"},
		{"bridge", "bridge"},
		{"block-explorer", "block-explorer"},
	}

	for _, comp := range components {
		svcNames, err := utils.GetServiceNames(ctx, chainNamespace, comp.pattern)
		if err != nil {
			t.logger.Warn("Failed to get service names", "component", comp.name, "err", err)
			continue
		}

		if len(svcNames) > 0 {
			services[comp.name] = svcNames[0]
			t.logger.Infof("Discovered service: %s -> %s", comp.name, svcNames[0])
		}
	}

	return services, nil
}

// isURLReachable checks if a URL is publicly reachable via HTTP
func (t *ThanosStack) isURLReachable(url string) bool {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		t.logger.Infof("⏳ Waiting for LoadBalancer to be publicly reachable...")
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500
}
