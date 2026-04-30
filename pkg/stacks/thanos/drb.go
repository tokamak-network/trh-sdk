package thanos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// InstallDRB installs the DRB (Distributed Random Beacon) VRF node via Helm.
// The DRB contract is already injected into genesis at predeploy address 0x4200...0060.
// This function deploys the off-chain DRB operator node that submits random values.
//
// Requires:
//   - t.deployConfig.K8s set (chain deployed)
//   - t.deployConfig.L2RpcUrl set (AWS EKS path)
//   - t.deployConfig.AdminPrivateKey set
func (t *ThanosStack) InstallDRB(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}
	if t.deployConfig.L2RpcUrl == "" {
		return fmt.Errorf("L2 RPC URL is not set — DRB install requires the chain to be deployed on AWS EKS")
	}

	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-vrf", t.deploymentPath)
	namespace := t.deployConfig.K8s.Namespace

	// Check if DRB chart exists (tokamak-thanos-stack must be cloned first)
	if _, err := os.Stat(chartPath); err != nil {
		return fmt.Errorf("drb-vrf chart not found at %s: clone tokamak-thanos-stack first (run deploy-chain)", chartPath)
	}

	imageTag := t.drbImageTag()

	t.logger.Infof("🔧 Installing DRB VRF node (image tag: %s)...", imageTag)

	args := []string{
		"upgrade",
		"--install",
		"drb-vrf",
		chartPath,
		"--namespace", namespace,
		"--create-namespace",
		"--set", fmt.Sprintf("drb_vrf.env.l2_rpc_url=%s", t.deployConfig.L2RpcUrl),
		"--set", fmt.Sprintf("drb_vrf.env.contract_address=%s", drbPredeployAddress),
		"--set", fmt.Sprintf("drb_vrf.env.operator_private_key=%s", t.deployConfig.AdminPrivateKey),
		"--set", fmt.Sprintf("drb_vrf.image.tag=%s", imageTag),
	}

	output, err := utils.ExecuteCommand(ctx, "helm", args...)
	if err != nil {
		t.logger.Errorw("❌ Failed to install DRB VRF node", "err", err, "helm_output", output)
		return fmt.Errorf("helm install drb-vrf failed: %w", err)
	}

	t.logger.Info("✅ DRB VRF node installed successfully")
	return nil
}

// UninstallDRB removes the DRB VRF node.
// On AWS EKS: uninstalls the Helm release.
// On local Docker: stops and removes DRB Docker Compose services.
func (t *ThanosStack) UninstallDRB(ctx context.Context) error {
	// Local Docker path
	if t.deployConfig.K8s == nil {
		composePath := filepath.Join(t.deploymentPath, "docker-compose.local.yml")
		t.logger.Info("Stopping and removing DRB containers...")
		if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
			"-f", composePath, "rm", "-f", "-s", "drb-leader", "drb-postgres"); err != nil {
			return fmt.Errorf("failed to remove DRB containers: %w", err)
		}
		t.logger.Info("✅ DRB containers removed successfully")
		return nil
	}

	// AWS EKS path: Helm uninstall
	namespace := t.deployConfig.K8s.Namespace

	releases, err := utils.FilterHelmReleases(ctx, namespace, "drb-vrf")
	if err != nil {
		t.logger.Warnw("Could not list DRB helm releases, attempting direct uninstall", "err", err)
		releases = []string{"drb-vrf"}
	}

	for _, release := range releases {
		t.logger.Infow("Uninstalling DRB Helm release", "release", release, "namespace", namespace)
		_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				t.logger.Info("DRB release not found, already uninstalled")
				continue
			}
			return fmt.Errorf("helm uninstall drb-vrf failed: %w", err)
		}
	}

	t.logger.Info("✅ DRB VRF node uninstalled successfully")
	return nil
}

// drbImageTag returns the DRB node Docker image tag for the current network.
func (t *ThanosStack) drbImageTag() string {
	if t.deployConfig == nil || t.deployConfig.Network == "" {
		return "sha-8c37f63"
	}
	return "sha-8c37f63" // default; future: read from constants.DockerImageTag[network]
}
