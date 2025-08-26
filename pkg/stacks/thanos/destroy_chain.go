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

	helmReleases, err := utils.GetHelmReleases(ctx, namespace)
	if err != nil {
		t.logger.Error("Error retrieving Helm releases", "err", err)
	}

	if len(helmReleases) > 0 {
		for _, release := range helmReleases {
			if strings.Contains(release, namespace) || strings.Contains(release, "op-bridge") || strings.Contains(release, "block-explorer") || strings.Contains(release, constants.MonitoringNamespace) {
				t.logger.Info("Uninstalling Helm release: %s in namespace: %s...", release, namespace)
				_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
				if err != nil {
					t.logger.Error("Error removing Helm release", "err", err)
					return err
				}
			}
		}

		t.logger.Info("Helm release removed successfully")
	}

	// Delete monitoring resources
	err = t.UninstallMonitoring(ctx)
	if err != nil {
		t.logger.Error("Error uninstalling monitoring", "err", err)
	}

	// Delete namespace before destroying the infrastructure
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteK8sNamespace(ctxTimeout, namespace)
	if err != nil {
		t.logger.Error("Error deleting namespace", "err", err)
	} else {
		t.logger.Info("✅ Namespace destroyed successfully!")
	}

	err = t.clearTerraformState(ctx)
	if err != nil {
		t.logger.Error("Failed to clear the existing terraform state", "err", err)
		return err
	}

	t.deployConfig.K8s = nil
	t.deployConfig.ChainName = ""
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Error("Failed to write the updated config", "err", err)
		return err
	}

	t.logger.Info("✅The chain has been destroyed successfully!")
	return nil
}
