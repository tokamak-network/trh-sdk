package thanos

import (
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) clearTerraformState() error {
	// STEP 1: Destroy tokamak-thanos-stack/terraform/thanos-stack resources
	err := t.destroyTerraform("tokamak-thanos-stack/terraform/thanos-stack")
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

	// STEP 3: Destroy tokamak-thanos-stack/terraform/block-explorer resources
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/block-exlorer")
	if err != nil {
		fmt.Println("Error running block-explorer terraform destroy", err)
		return err
	}

	fmt.Println("Destroy the terraform resources successfully")

	return nil
}

func (t *ThanosStack) destroyTerraform(path string) error {
	if !checkDirExists(path) {
		fmt.Printf("Skipping %s: directory does not exist.\n", path)
		return nil
	}

	err := utils.ExecuteCommandStream("bash", []string{
		"-c",
		fmt.Sprintf(`cd %s && source ../.envrc && terraform destroy -auto-approve -parallelism=50`, path),
	}...)
	if err != nil {
		fmt.Printf("Error running terraform destroy for %s: %v\n", path, err)
		return err
	}
	fmt.Printf("%s terraform destroyed successfully.\n", path)

	err = utils.ExecuteCommandStream("bash", []string{
		"-c",
		fmt.Sprintf(`cd %s && terraform.tfstate terraform.tfstate.backup .terraform`, path),
	}...)
	if err != nil {
		fmt.Printf("Error delete terraform folder for %s: %v\n", path, err)
		return err
	}

	return nil
}
