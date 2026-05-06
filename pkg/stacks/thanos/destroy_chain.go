package thanos

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/backup"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// --------------------------------------------- Destroy command -------------------------------------//

func (t *ThanosStack) Destroy(ctx context.Context) error {
	switch t.network {
	case constants.LocalDevnet:
		return t.destroyDevnet(ctx)
	case constants.Testnet, constants.Mainnet:
		if !t.isAWSDeployment() {
			return t.destroyLocalNetwork(ctx)
		}
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

// uninstallFeatures removes all optional Helm-managed features from the K8s cluster.
// It uses a best-effort strategy: each uninstaller is called regardless of prior failures.
// All errors are collected and returned joined; the caller decides whether to treat them as fatal.
func (t *ThanosStack) uninstallFeatures(ctx context.Context) error {
	type uninstaller struct {
		name string
		fn   func(context.Context) error
	}

	uninstallers := []uninstaller{
		{"DRB", t.UninstallDRB},
		{"CrossTrade", t.UninstallCrossTradeAWS},
		{"Bridge", t.UninstallBridge},
		{"BlockExplorer", t.UninstallBlockExplorer},
		{"Monitoring", t.UninstallMonitoring},
		{"UptimeService", t.UninstallUptimeService},
	}

	var errs []error
	for _, u := range uninstallers {
		if err := u.fn(ctx); err != nil {
			t.logger.Warnf("Failed to uninstall %s: %v", u.name, err)
			errs = append(errs, fmt.Errorf("uninstall %s: %w", u.name, err))
		}
	}

	return errors.Join(errs...)
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

	// Uninstall all optional features (best-effort: continues on error)
	if err := t.uninstallFeatures(ctx); err != nil {
		t.logger.Warnf("Some features failed to uninstall (will be cleaned up by Terraform): %v", err)
		// intentionally non-fatal: Terraform teardown removes underlying infrastructure
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

	// Delete EFS mount targets first — DetectEFSId reads PVCs in the
	// namespace, so this must run before namespace deletion. Removing the
	// mount targets also frees the EFS PV finalizers that often stall
	// namespace termination.
	if t.deployConfig.AWS != nil {
		if efsID, detectErr := utils.DetectEFSId(ctx, namespace); detectErr == nil && efsID != "" {
			t.logger.Infof("Deleting EFS mount targets for %s before namespace deletion...", efsID)
			if mtErr := backup.DeleteEFSMountTargets(ctx, t.logger, t.deployConfig.AWS.Region, efsID); mtErr != nil {
				t.logger.Warnf("Failed to delete EFS mount targets: %v. Continuing with destroy.", mtErr)
			} else {
				t.logger.Info("EFS mount targets deleted, waiting for cleanup...")
				time.Sleep(30 * time.Second)
			}
		}
	}

	// Best-effort namespace deletion. If this fails, terraform destroy still
	// removes the EKS cluster and the namespace disappears with it.
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	if err := t.tryToDeleteK8sNamespace(ctxTimeout, namespace); err != nil {
		t.logger.Warnw("Namespace deletion did not complete; Terraform destroy will continue and clean up the cluster", "namespace", namespace, "err", err)
	} else {
		t.logger.Info("✅ Namespace destroyed successfully!")
	}

	err = t.clearTerraformState(ctx)
	if err != nil {
		t.logger.Error("❌ Failed to clear terraform state", "err", err)
		return err
	}

	// runs postdestroy verification to catch any orphaned resources
	if namespace != "" {
		if err := t.VerifyAndCleanupResources(ctx, namespace); err != nil {
			t.logger.Warnf("Post-destroy verification encountered issues: %v", err)
		}
	}

	t.deployConfig.K8s = nil
	t.deployConfig.ChainName = ""
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		t.logger.Warnf("Failed to write the updated config: %v. Resources are already destroyed.", err)
		// Continue even if config write fails, as resources are already destroyed
	}

	t.logger.Info("The chain has been destroyed successfully!")
	return nil
}
