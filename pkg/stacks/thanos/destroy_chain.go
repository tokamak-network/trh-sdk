package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// --------------------------------------------- Destroy command -------------------------------------//

func (t *ThanosStack) Destroy(ctx context.Context) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.destroyDevnet(ctx)
	case constants.Testnet, constants.Mainnet:
		return t.destroyInfraOnAWS(ctx)
	}
	return nil
}

func (t *ThanosStack) destroyDevnet(ctx context.Context) error {
	output, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("cd %s/tokamak-thanos && make nuke", t.deploymentPath))
	if err != nil {
		t.logger.Error("❌ Devnet cleanup failed!", "details", output)
		return err
	}

	t.logger.Info("✅ Devnet network destroyed successfully!")

	return nil
}

func (t *ThanosStack) destroyInfraOnAWS(ctx context.Context) error {
	var (
		err error
	)

	var namespace string
	if t.deployConfig.K8s != nil {
		namespace = t.deployConfig.K8s.Namespace
	}

	if t.awsProfile == nil {
		t.logger.Error("AWS profile is not set")
		return fmt.Errorf("AWS profile is not set")
	}

	// Perform backup cleanup early while Kubernetes context still exists
	if err := t.CleanupUnusedBackupResources(ctx); err != nil {
		t.logger.Warnf("Failed to cleanup unused backup resources: %v", err)
	}

	helmReleases, err := utils.GetHelmReleases(ctx, namespace)
	if err != nil {
		t.logger.Warnf("Failed to retrieve Helm releases: %v. Continuing without uninstalling Helm releases.", err)
		helmReleases = []string{} // Continue with empty list
	}

	if len(helmReleases) > 0 {
		failedReleases := []string{}
		for _, release := range helmReleases {
			if strings.Contains(release, namespace) || strings.Contains(release, "op-bridge") || strings.Contains(release, "block-explorer") || strings.Contains(release, constants.MonitoringNamespace) {
				t.logger.Infof("Uninstalling Helm release: %s in namespace: %s...", release, namespace)
				_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
				if err != nil {
					t.logger.Warnf("Failed to uninstall Helm release %s in namespace %s: %v. Continuing with other releases.", release, namespace, err)
					failedReleases = append(failedReleases, release)
				}
			}
		}

		if len(failedReleases) == 0 {
			t.logger.Info("Helm release removed successfully")
		} else {
			t.logger.Warnf("Some Helm releases failed to uninstall: %v", failedReleases)
		}
	}

	// Delete monitoring resources
	err = t.UninstallMonitoring(ctx)
	if err != nil {
		t.logger.Warnf("Failed to uninstall monitoring resources: %v. Continuing with destroy process.", err)
		// Continue even if monitoring uninstall fails, as monitoring may not exist
	}

	// Uninstall uptime-service if needed
	err = t.UninstallUptimeService(ctx)
	if err != nil {
		t.logger.Warnf("Failed to uninstall uptime-service: %v. Continuing with destroy process.", err)
	}

	// Uninstall thanos-logs stack
	err = t.UninstallMonitoringThanosLogsStack(ctx, "")
	if err != nil {
		t.logger.Warnf("Failed to uninstall thanos-logs stack: %v. Continuing with destroy process.", err)
	}

	// Delete namespace before destroying the infrastructure
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteK8sNamespace(ctxTimeout, namespace)
	if err != nil {
		t.logger.Error("❌ Failed to delete namespace", "namespace", namespace, "err", err)
		return err
	}
	t.logger.Info("✅ Namespace destroyed successfully!")

	err = t.clearTerraformState(ctx)
	if err != nil {
		t.logger.Error("❌ Failed to clear terraform state", "err", err)
		return err
	}

	t.deployConfig.K8s = nil
	t.deployConfig.ChainName = ""
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Warnf("Failed to write the updated config: %v. Resources are already destroyed.", err)
		// Continue even if config write fails, as resources are already destroyed
	}

	t.logger.Info("✅The chain has been destroyed successfully!")
	return nil
}
