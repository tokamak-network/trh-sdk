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
		fmt.Printf("\r❌ Devnet cleanup failed!       \n Details: %s", output)
		return err
	}

	fmt.Print("\r✅ Devnet network destroyed successfully!       \n")

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
		fmt.Println("AWS profile is not set")
		return fmt.Errorf("AWS profile is not set")
	}

	helmReleases, err := utils.GetHelmReleases(ctx, namespace)
	if err != nil {
		fmt.Println("Error retrieving Helm releases:", err)
	}

	if len(helmReleases) > 0 {
		for _, release := range helmReleases {
			if strings.Contains(release, namespace) || strings.Contains(release, "op-bridge") || strings.Contains(release, "block-explorer") || strings.Contains(release, "monitoring") {
				fmt.Printf("Uninstalling Helm release: %s in namespace: %s...\n", release, namespace)
				_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
				if err != nil {
					fmt.Println("Error removing Helm release:", err)
					return err
				}
			}
		}

		fmt.Println("Helm release removed successfully")
	}

	// Delete monitoring resources
	err = t.UninstallMonitoring(ctx)
	if err != nil {
		fmt.Println("Error uninstalling monitoring:", err)
	}

	// Delete namespace before destroying the infrastructure
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	err = t.tryToDeleteK8sNamespace(ctxTimeout, namespace)
	if err != nil {
		fmt.Println("Error deleting namespace:", err)
	} else {
		fmt.Println("✅ Namespace destroyed successfully!")
	}

	err = t.clearTerraformState(ctx)
	if err != nil {
		fmt.Printf("Failed to clear the existing terraform state, err: %s", err.Error())
		return err
	}

	t.deployConfig.K8s = nil
	t.deployConfig.ChainName = ""
	err = t.deployConfig.WriteToJSONFile(t.deploymentPath)
	if err != nil {
		fmt.Printf("Failed to write the updated config, err: %s", err.Error())
		return err
	}

	fmt.Println("✅The chain has been destroyed successfully!")
	return nil
}
