package thanos

import (
	"context"
	"path/filepath"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// UninstallDRB removes the DRB VRF node.
func (t *ThanosStack) UninstallDRB(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		composePath := filepath.Join(t.deploymentPath, "docker-compose.local.yml")
		t.logger.Info("Stopping and removing DRB containers...")
		if err := utils.ExecuteCommandStream(ctx, t.logger, "docker", "compose",
			"-f", composePath, "rm", "-f", "-s", "drb-leader", "drb-postgres"); err != nil {
			return err
		}
		t.logger.Info("✅ DRB containers removed successfully")
		return nil
	}

	namespace := t.deployConfig.K8s.Namespace
	t.logger.Infof("Uninstalling DRB VRF node from namespace %s...", namespace)
	output, err := utils.ExecuteCommand(ctx, "helm", "uninstall", "drb-vrf", "--namespace", namespace)
	if err != nil {
		t.logger.Warnf("helm uninstall drb-vrf: %s", output)
		return err
	}
	t.logger.Info("✅ DRB VRF node uninstalled successfully")
	return nil
}
