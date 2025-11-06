package thanos

import (
	"context"
	"time"
	"strings"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

//Insatlling uptime-kuma
func (t *ThanosStack) InstallUptimeKuma(ctx context.Context, config *types.UptimeKumaConfig) (string, error) {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return "", fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	//Checking if uptime-kuma is already running
	uptimeKumapods,err := utils.GetPodNamesByLabel(ctx,config.Namespace,"uptime-kuma");
	if err != nil {
		t.logger.Error("Error to get uptime kuma pods", "err", err)
		return "", err
	}

	var kumaUrl string
	helmReleaseName := fmt.Sprintf("%s-%d", "uptime-kuma", time.Now().Unix())

	if len(uptimeKumapods) > 0 {
		t.logger.Info("Uptime Kuma is already running")
		// Get existing Helm release name from running installation
		existingReleases, err := utils.FilterHelmReleases(ctx, config.Namespace, "uptime-kuma")
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
				kumaUrl = "http://" + k8sServices[0]
				
				break
			}
			time.Sleep(15 * time.Second)
		}
		t.logger.Infof("😃uptime-kuma is already running. You can access it at: %s", kumaUrl)
		return kumaUrl, nil
	}

	// Set helmReleaseName for new installation
	config.HelmReleaseName = helmReleaseName

	err = t.cloneSourcecode(ctx, "tokamak-thanos-stack", "https://github.com/tokamak-network/tokamak-thanos-stack.git")
	if err != nil {
		t.logger.Error("Error cloning repository", "err", err)
		return "", err
	}

	t.logger.Info("🚀 Installing uptime kuma...")

	// Ensure uptime-kuma namespace exists
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		t.logger.Errorw("Failed to ensure uptime-kuma namespace exists", "err", err)
		return "",fmt.Errorf("failed to ensure uptime-kuma namespace exists: %w", err)
	}

	// Deploy infrastructure if persistence is enabled
	if config.IsPersistenceEnable {
		t.logger.Info("Deploying uptime-kuma infrastructure (persistence enabled)")
		if err := t.deployUptimeKumaInfrastructure(ctx, config); err != nil {
			t.logger.Errorw("Failed to deploy uptime-kuma infrastructure", "err", err)
			return "",fmt.Errorf("failed to deploy uptime-kuma infrastructure: %w", err)
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
	valuesFilePath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/uptime-kuma/values.yaml", t.deploymentPath)
	pvcName := fmt.Sprintf("%s-pvc", config.HelmReleaseName)

	// Update helm chart service fields
	err = utils.UpdateYAMLField(valuesFilePath, "service.type", "LoadBalancer")
	if err != nil {
		return "", fmt.Errorf("failed to update service.type: %w", err)
	}

	err = utils.UpdateYAMLField(valuesFilePath, "service.port", 80)
	if err != nil {
		return "", fmt.Errorf("failed to update service.port: %w", err)
	}

	err = utils.UpdateYAMLField(valuesFilePath, "service.targetPort", 3001)
	if err != nil {
		return "", fmt.Errorf("failed to update service.targetPort: %w", err)
	}

	// Set AWS LoadBalancer annotations
	annotations := map[string]interface{}{
		"service.beta.kubernetes.io/aws-load-balancer-type": "nlb-ip",
		"service.beta.kubernetes.io/aws-load-balancer-scheme": "internet-facing",
	}
	err = utils.UpdateYAMLField(valuesFilePath, "service.annotations", annotations)
	if err != nil {
		return "", fmt.Errorf("failed to update service.annotations: %w", err)
	}

	// Set pod labels so GetPodNamesByLabel can find pods with "uptime-kuma" label
	podLabels := map[string]interface{}{
		"uptime-kuma": "",
	}
	err = utils.UpdateYAMLField(valuesFilePath, "podLabels", podLabels)
	if err != nil {
		return "", fmt.Errorf("failed to update podLabels: %w", err)
	}

	// Update helm chart volume fields
	err = utils.UpdateYAMLField(valuesFilePath, "volume.enabled", true)
	if err != nil {
		return "", fmt.Errorf("failed to update volume.enabled: %w", err)
	}

	err = utils.UpdateYAMLField(valuesFilePath, "volume.existingClaim", pvcName)
	if err != nil {
		return "", fmt.Errorf("failed to update volume.existingClaim: %w", err)
	}

	err = utils.UpdateYAMLField(valuesFilePath, "volume.storageClassName", "efs-sc")
	if err != nil {
		return "", fmt.Errorf("failed to update volume.storageClassName: %w", err)
	}

	err = utils.UpdateYAMLField(valuesFilePath, "volume.size", "4Gi")
	if err != nil {
		return "", fmt.Errorf("failed to update volume.size: %w", err)
	}

	args := []string{
		"install",
		helmReleaseName,
		fmt.Sprintf("%s/tokamak-thanos-stack/charts/uptime-kuma", t.deploymentPath),
		"--values", valuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
	}

	//Installing helm chart for uptime-kuma
	output, err := utils.ExecuteCommand(ctx, "helm", args...)

	if err != nil {
		t.logger.Error(
			"❌ Error installing uptime kuma", 
			"err", err,                     
			"helm_output", output,         
		)
		return "", err 
	}
	t.logger.Info("✅ Install uptime kuma successfully")

	//Fetching the service loadbalancer URL
	for {
		k8sServices, err := utils.GetAddressByService(ctx, config.Namespace, helmReleaseName)
		if err != nil {
			t.logger.Error("Error retrieving service addresses", "err", err, "details", k8sServices)
			return "", err
		}

		if len(k8sServices) > 0 {
			kumaUrl = "http://" + k8sServices[0]
			break
		}

		time.Sleep(15 * time.Second)
	}
	t.logger.Infof("✅ uptime-kuma is up and running. You can access it at: %s", kumaUrl)

	return kumaUrl, nil

}

//Uninstalling uptime-kuma
func (t *ThanosStack) UninstallUptimeKuma(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		t.logger.Error("K8s configuration is not set. Please run the deploy command first")
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	kumaNamespace := constants.UptimeKumaNamespace
	fmt.Printf("namespace while uninstalling : %s ", kumaNamespace)

	// Check if namespace exists first
	exists, err := utils.CheckNamespaceExists(ctx, kumaNamespace)
	if err != nil {
		t.logger.Errorw("Failed to check uptime-kuma namespace existence", "err", err)
		return err
	}

	if !exists {
		t.logger.Info("Uptime-kuma namespace does not exist, skipping uninstallation")
		return nil
	}

	// Uninstall Helm releases
	releases, err := utils.FilterHelmReleases(ctx, kumaNamespace, "uptime-kuma")
	if err != nil {
		t.logger.Error("Error to filter helm releases", "err", err)
		return err
	}

	for _, release := range releases {
		t.logger.Infow("Uninstalling Helm release", "release", release, "namespace", kumaNamespace)
		_, err = utils.ExecuteCommand(ctx, "helm", []string{
			"uninstall",
			release,
			"--namespace",
			kumaNamespace,
		}...)
		if err != nil {
			t.logger.Error("❌ Error uninstalling uptime-kuma helm chart", "err", err)
			return err
		}
	}

	// Clean up PVCs that might be left behind
	config := &types.UptimeKumaConfig{
		Namespace: kumaNamespace,
	}
	if err := t.cleanupExistingUptimeKumaStorage(ctx, config); err != nil {
		t.logger.Warnw("❌ Failed to cleanup uptime-kuma storage", "err", err)
	}

	// Delete namespace 
    // t.logger.Info(fmt.Sprintf("❌ Deleting uptime-kuma namespace: %s", kumaNamespace))
    // err = t.tryToDeleteK8sNamespace(ctx, kumaNamespace)
    // if err != nil {
    //     t.logger.Errorw("Failed to delete uptime-kuma namespace", "err", err, "namespace", kumaNamespace)
    //     return err

	t.logger.Info("✅ Uninstall of uptime-kuma successfully!")
	return nil
}


// GetUptimeKumaConfig gathers all required configuration for uptime-kuma
func (t *ThanosStack) GetUptimeKumaConfig(ctx context.Context) (*types.UptimeKumaConfig, error) {

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

	config := &types.UptimeKumaConfig{
		Namespace:         		constants.UptimeKumaNamespace,
		IsPersistenceEnable: 	true,
		EFSFileSystemId:   		efsFileSystemId,
		ChainName:         		chainName,
	}

	return config, nil
}

// deployUptimeKumaInfrastructure creates PVs for Static Provisioning
func (t *ThanosStack) deployUptimeKumaInfrastructure(ctx context.Context, config *types.UptimeKumaConfig) error {
	if err := t.ensureNamespaceExists(ctx, config.Namespace); err != nil {
		return fmt.Errorf("❌ failed to ensure namespace exists: %w", err)
	}

	// Clean up existing UptimeKuma PVs and PVCs
	if err := t.cleanupExistingUptimeKumaStorage(ctx, config); err != nil {
		return fmt.Errorf("❌ failed to cleanup existing uptime-kuma storage: %w", err)
	}

	timestamp, err := utils.GetTimestampFromExistingPV(ctx, config.ChainName)
	if err != nil {
		return fmt.Errorf("❌ failed to get timestamp from existing PV: %w", err)
	}

	// Create PV and PVC using generic functions from utils/storage.go
	kumaPV := utils.GenerateStaticPVManifest("uptime-kuma", config, "4Gi", timestamp)
	if err := utils.ApplyPVManifest(ctx, t.deploymentPath, "uptime-kuma", kumaPV, "UptimeKuma"); err != nil {
		return fmt.Errorf("❌ failed to create kuma PV: %w", err)
	}

	kumaPVC := utils.GenerateStaticPVCManifest("uptime-kuma", config, "4Gi", timestamp)
	if err := utils.ApplyPVCManifest(ctx, t.deploymentPath, "uptime-kuma", kumaPVC, "UptimeKuma"); err != nil {
		return fmt.Errorf("failed to create kuma PVC: %w", err)
	}
	fmt.Println("✅ Created uptime-kuma PV and PVC")

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


// cleanupExistingKumaStorage removes existing Kuma PVs and PVCs
func (t *ThanosStack) cleanupExistingUptimeKumaStorage(ctx context.Context, config *types.UptimeKumaConfig) error {
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

	return nil
}