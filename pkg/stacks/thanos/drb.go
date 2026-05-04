package thanos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// InstallDRB installs the DRB (Distributed Random Beacon) VRF node via Helm.
// The DRB contract is already injected into genesis at predeploy address 0x4200...0060.
// Deploys leader + 3 regular operator nodes, each with its own postgres sidecar.
//
// Requires:
//   - t.deployConfig.K8s set (chain deployed on AWS EKS)
//   - t.deployConfig.L2RpcUrl set
//   - t.deployConfig.Mnemonic set (for deterministic key derivation)
func (t *ThanosStack) InstallDRB(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}
	if t.deployConfig.L2RpcUrl == "" {
		return fmt.Errorf("L2 RPC URL is not set — DRB install requires the chain to be deployed on AWS EKS")
	}
	if t.deployConfig.Mnemonic == "" {
		return fmt.Errorf("mnemonic is not set — DRB install requires a mnemonic for key derivation")
	}

	chartPath := fmt.Sprintf("%s/tokamak-thanos-stack/charts/drb-vrf", t.deploymentPath)
	namespace := t.deployConfig.K8s.Namespace

	if _, err := os.Stat(chartPath); err != nil {
		return fmt.Errorf("drb-vrf chart not found at %s: clone tokamak-thanos-stack first (run deploy-chain)", chartPath)
	}

	accounts, err := DeriveDRBAccounts(t.deployConfig.Mnemonic)
	if err != nil {
		return fmt.Errorf("failed to derive DRB accounts: %w", err)
	}

	imageTag := constants.DockerImageTag[t.network].DRBNodeImageTag

	t.logger.Infof("🔧 Installing DRB VRF node (image tag: %s)...", imageTag)

	args := []string{
		"upgrade", "--install", "drb-vrf", chartPath,
		"--namespace", namespace, "--create-namespace",
		"--set-string", fmt.Sprintf("image.tag=%s", imageTag),
		"--set-string", fmt.Sprintf("contractAddress=%s", drbPredeployAddress),
		"--set-string", fmt.Sprintf("chainID=%d", t.deployConfig.L2ChainID),
		"--set-string", fmt.Sprintf("ethRpcUrls=%s", t.deployConfig.L2RpcUrl),
		"--set-string", fmt.Sprintf("leader.privateKey=%s", accounts.LeaderPrivateKey),
		"--set-string", fmt.Sprintf("leader.eoa=%s", accounts.LeaderEOA.Hex()),
		"--set-string", fmt.Sprintf("leader.peerID=%s", accounts.LeaderPeerID),
		"--set-string", fmt.Sprintf("leader.peerIDBytes=%s", base64.StdEncoding.EncodeToString(accounts.LeaderPeerIDBytes)),
	}

	regularPorts := []int{9601, 9602, 9603}
	for i, r := range accounts.Regulars {
		prefix := fmt.Sprintf("regulars[%d]", i)
		args = append(args,
			"--set-string", fmt.Sprintf("%s.privateKey=%s", prefix, r.PrivateKey),
			"--set-string", fmt.Sprintf("%s.peerID=%s", prefix, r.PeerID),
			"--set-string", fmt.Sprintf("%s.peerIDBytes=%s", prefix, base64.StdEncoding.EncodeToString(r.PeerIDBytes)),
			"--set", fmt.Sprintf("%s.port=%d", prefix, regularPorts[i]),
		)
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
