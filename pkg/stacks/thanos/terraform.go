package thanos

import (
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) clearTerraformState() error {
	// STEP 1: Destroy tokamak-thanos-stack/terraform/block-explorer resources
	err := t.destroyTerraform("tokamak-thanos-stack/terraform/block-exlorer")
	if err != nil {
		fmt.Println("Error running block-explorer terraform destroy", err)
		return err
	}
	// STEP 1: Destroy tokamak-thanos-stack/terraform/thanos-stack resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/thanos-stack")
	if err != nil {
		fmt.Println("Error running thanos-stack terraform destroy:", err)
		return err
	}

	// STEP 2: Destroy tokamak-thanos-stack/terraform/backend resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/backend")
	if err != nil {
		fmt.Println("Error running backend terraform destroy", err)
		return err
	}

	return nil
}

func (t *ThanosStack) destroyTerraform(path string) error {
	if !utils.CheckDirExists(path) {
		return nil
	}

	if !utils.CheckFileExists(fmt.Sprintf("%s/../.envrc", path)) {
		return nil
	}

	err := utils.ExecuteCommandStream("bash", []string{
		"-c",
		fmt.Sprintf(`cd %s && source ../.envrc && terraform destroy -auto-approve -parallelism=100`, path),
	}...)
	if err != nil {
		fmt.Printf("Error running terraform destroy for %s: %v\n", path, err)
	}

	return nil
}
