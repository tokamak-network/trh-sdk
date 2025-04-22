package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) clearTerraformState(ctx context.Context, logFileName string) error {
	// STEP 1: Destroy tokamak-thanos-stack/terraform/block-explorer resources
	err := t.destroyTerraform("tokamak-thanos-stack/terraform/block-explorer", logFileName)
	fmt.Println("Destroying block-explorer terraform resources")
	if err != nil {
		fmt.Println("Error running block-explorer terraform destroy", err)
		return err
	}

	// STEP 2: Destroy tokamak-thanos-stack/terraform/thanos-stack resources
	// Check the bucket name in the state file exists
	// If it doesn't exist, we need to delete the state file to prevent conflicts when reinstalling the stack
	thanosStackTerraformPath := "tokamak-thanos-stack/terraform/thanos-stack/.terraform/terraform.tfstate"
	if utils.CheckFileExists(thanosStackTerraformPath) {
		state, err := utils.ReadThanosStackTerraformState(thanosStackTerraformPath)
		if err != nil {
			fmt.Println("Error reading terraform state file", err)
			return err
		}

		fmt.Println("Checking bucket existence", state.Backend.Config.Bucket)

		bucketExist := utils.BucketExists(ctx, t.s3Client, state.Backend.Config.Bucket)
		if bucketExist {
			fmt.Println("Destroying thanos-stack terraform resources")
			err = t.destroyTerraform("tokamak-thanos-stack/terraform/thanos-stack", logFileName)
			if err != nil {
				fmt.Println("Error running thanos-stack terraform destroy:", err)
				return err
			}
		}
	}

	// STEP 3: Destroy tokamak-thanos-stack/terraform/backend resources
	fmt.Println("Destroying backend terraform resources")
	err = t.destroyTerraform("tokamak-thanos-stack/terraform/backend", logFileName)
	if err != nil {
		fmt.Println("Error running backend terraform destroy", err)
		return err
	}

	// STEP 4: delete the `tokamak-thanos-stack/thanos-stack` state to prevent conflicts when reinstalling the stack
	// This is a workaround for the issue where the bucket name is deleted but the state file remains containing the old bucket name.
	err = utils.ExecuteCommandStream("bash", logFileName, []string{
		"-c",
		`cd tokamak-thanos-stack/terraform/thanos-stack && rm -rf terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl .terraform/`,
	}...)
	if err != nil {
		fmt.Println("Error deleting thanos-stack terraform state", err)
		return err
	}

	return nil
}

func (t *ThanosStack) destroyTerraform(path string, logFileName string) error {
	if !utils.CheckDirExists(path) {
		return nil
	}

	if !utils.CheckFileExists(fmt.Sprintf("%s/../.envrc", path)) {
		return nil
	}

	// Check state before destroying
	output, err := utils.ExecuteCommand("bash", logFileName, "-c", fmt.Sprintf("cd %s && source ../.envrc &&  terraform state list", path))
	if err != nil || strings.TrimSpace(output) == "" {
		return nil
	}

	err = utils.ExecuteCommandStream("bash", logFileName, []string{
		"-c",
		fmt.Sprintf(`cd %s && source ../.envrc && terraform destroy -auto-approve -parallelism=100`, path),
	}...)
	if err != nil {
		fmt.Printf("Error running terraform destroy for %s: %v\n", path, err)
	}

	return nil
}
