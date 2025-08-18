package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) clearTerraformState(ctx context.Context) error {
	// STEP 1: Destroy tokamak-thanos-stack/terraform/block-explorer resources
	err := t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/block-explorer", t.deploymentPath))
	t.logger.Info("Destroying block-explorer terraform resources")
	if err != nil {
		t.logger.Error("Error running block-explorer terraform destroy", "err", err)
		return err
	}

	// STEP 2: Destroy tokamak-thanos-stack/terraform/thanos-stack resources
	// Check the bucket name in the state file exists
	// If it doesn't exist, we need to delete the state file to prevent conflicts when reinstalling the stack
	thanosStackTerraformPath := fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack/.terraform/terraform.tfstate", t.deploymentPath)
	if utils.CheckFileExists(thanosStackTerraformPath) {
		state, err := utils.ReadThanosStackTerraformState(thanosStackTerraformPath)
		if err != nil {
			t.logger.Error("Error reading terraform state file", "err", err)
			return err
		}

		t.logger.Info("Checking bucket existence", "bucket", state.Backend.Config.Bucket)

		bucketExist := utils.BucketExists(ctx, t.awsProfile.S3Client, state.Backend.Config.Bucket)
		if bucketExist {
			t.logger.Info("Destroying thanos-stack terraform resources")
			err = t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/thanos-stack", t.deploymentPath))
			if err != nil {
				t.logger.Error("Error running thanos-stack terraform destroy:", "err", err)
				return err
			}
		}
	}

	// STEP 3: Destroy tokamak-thanos-stack/terraform/backend resources
	t.logger.Info("Destroying backend terraform resources")
	err = t.destroyTerraform(ctx, fmt.Sprintf("%s/tokamak-thanos-stack/terraform/backend", t.deploymentPath))
	if err != nil {
		t.logger.Error("Error running backend terraform destroy", "err", err)
		return err
	}

	// STEP 4: delete the `tokamak-thanos-stack/thanos-stack` state to prevent conflicts when reinstalling the stack
	// This is a workaround for the issue where the bucket name is deleted but the state file remains containing the old bucket name.
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s/tokamak-thanos-stack/terraform/thanos-stack && rm -rf terraform.tfstate terraform.tfstate.backup .terraform.lock.hcl .terraform/`, t.deploymentPath),
	}...)
	if err != nil {
		t.logger.Error("Error deleting thanos-stack terraform state", "err", err)
		return err
	}

	return nil
}

func (t *ThanosStack) destroyTerraform(ctx context.Context, path string) error {
	if !utils.CheckDirExists(path) {
		return nil
	}

	if !utils.CheckFileExists(fmt.Sprintf("%s/../.envrc", path)) {
		return nil
	}

	// Check state before destroying
	output, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("cd %s && source ../.envrc &&  terraform state list", path))
	if err != nil || strings.TrimSpace(output) == "" {
		return nil
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", []string{
		"-c",
		fmt.Sprintf(`cd %s && source ../.envrc && terraform destroy -auto-approve -parallelism=100`, path),
	}...)
	if err != nil {
		t.logger.Error("Error running terraform destroy for", "path", path, "err", err)
	}

	return nil
}
