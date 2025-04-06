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
	// STEP 2: Destroy tokamak-thanos-stack/terraform/thanos-stack resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/thanos-stack")
	if err != nil {
		fmt.Println("Error running thanos-stack terraform destroy:", err)
		return err
	}

	// STEP 3: Destroy tokamak-thanos-stack/terraform/backend resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/backend")
	if err != nil {
		fmt.Println("Error running backend terraform destroy", err)
		return err
	}

	// STEP 4: delete the `tokamak-thanos-stack/thanos-stack` state to prevent conflicts when reinstalling the stack
	// This is a workaround for the issue where the bucket name is deleted but the state file remains containing the old bucket name.
	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		`cd tokamak-thanos-stack/terraform/thanos-stack && rm -rf terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl .terraform/`,
	}...)
	if err != nil {
		fmt.Printf("Error deleting .envrc file %v\n", err)
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
