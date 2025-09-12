package backup

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// GatherBackupConfigInfo builds BackupConfigInfo and optionally prints usage via provided logger
func GatherBackupConfigInfo(
	region, namespace string,
	daily *string, keep *string, reset *bool,
	infof func(string, ...any),
) (*types.BackupConfigInfo, error) {
	if (daily == nil || strings.TrimSpace(*daily) == "") &&
		(keep == nil || strings.TrimSpace(*keep) == "") &&
		(reset == nil || !*reset) {
		if infof != nil {
			infof("📋 Backup Configuration Usage")
			infof("")
			infof("COMMAND:")
			infof("  trh-sdk backup-manager --config [OPTIONS]")
			infof("")
			infof("OPTIONS:")
			infof("  --daily HH:MM     Daily backup time in UTC (e.g., 03:00)")
			infof("  --keep DAYS       Retention days (0 = unlimited)")
			infof("  --reset           Reset to defaults (03:00 UTC, unlimited)")
			infof("")
			infof("EXAMPLES:")
			infof("  trh-sdk backup-manager --config --daily 02:30")
			infof("  trh-sdk backup-manager --config --keep 60")
			infof("  trh-sdk backup-manager --config --daily 01:00 --keep 30")
			infof("  trh-sdk backup-manager --config --reset")
		}
		return nil, nil
	}

	return &types.BackupConfigInfo{
		Region:    region,
		Namespace: namespace,
		Daily:     getStringValue(daily),
		Keep:      getStringValue(keep),
		Reset:     getBoolValue(reset),
	}, nil
}

// BuildTerraformArgs builds terraform -var arguments based on configuration and logs decisions
func BuildTerraformArgs(info *types.BackupConfigInfo, infof func(string, ...any)) []string {
	varArgs := []string{"-auto-approve"}
	if info == nil {
		return varArgs
	}
	if info.Reset {
		if infof != nil {
			infof("Resetting backup configuration to default values...")
		}
		varArgs = append(varArgs, `-var=backup_schedule_cron="cron(0 3 * * ? *)"`, "-var=backup_delete_after_days=0")
		return varArgs
	}
	if strings.TrimSpace(info.Daily) != "" {
		cron := convertTimeToCron(info.Daily)
		varArgs = append(varArgs, fmt.Sprintf(`-var=backup_schedule_cron="%s"`, cron))
		if infof != nil {
			infof("Setting backup schedule to: %s UTC", info.Daily)
		}
	}
	if strings.TrimSpace(info.Keep) != "" {
		if strings.TrimSpace(info.Keep) == "0" {
			varArgs = append(varArgs, "-var=backup_delete_after_days=0")
			if infof != nil {
				infof("Setting backup retention to: unlimited (recommended for blockchain)")
			}
		} else {
			varArgs = append(varArgs, fmt.Sprintf("-var=backup_delete_after_days=%s", strings.TrimSpace(info.Keep)))
			if infof != nil {
				infof("Setting backup retention to: %s days", strings.TrimSpace(info.Keep))
			}
		}
	}
	return varArgs
}

// ExecuteTerraformCommands executes terraform init/apply in the given root and logs via callbacks
func ExecuteTerraformCommands(
	ctx context.Context,
	tfRoot string,
	varArgs []string,
	infof func(string, ...any),
	warnf func(string, ...any),
) error {
	if infof != nil {
		infof("Applying EFS backup configuration...")
	}

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(tfRoot); err != nil {
		return fmt.Errorf("failed to change to terraform directory %s: %w", tfRoot, err)
	}
	defer os.Chdir(originalDir)

	if _, err := utils.ExecuteCommand(ctx, "direnv", "allow"); err != nil {
		if warnf != nil {
			warnf("Failed to run direnv allow: %v", err)
		}
	}

	if err := os.Chdir("thanos-stack"); err != nil {
		return fmt.Errorf("failed to change to thanos-stack directory: %w", err)
	}

	if infof != nil {
		infof("Initializing terraform...")
	}

	output, err := utils.ExecuteCommand(ctx, "bash", "-c", "source ../.envrc && terraform init")
	if err != nil {
		return fmt.Errorf("terraform init failed: %w\nOutput: %s", err, output)
	}

	if infof != nil {
		infof("Applying terraform changes...")
	}

	applyCmd := fmt.Sprintf("source ../.envrc && terraform apply %s", strings.Join(varArgs, " "))
	output, err = utils.ExecuteCommand(ctx, "bash", "-c", applyCmd)
	if err != nil {
		return fmt.Errorf("terraform apply failed: %w\nOutput: %s", err, output)
	}

	if infof != nil {
		infof("✅ Backup configuration applied successfully")
	}

	return nil
}

// ConfigureExecute applies configuration via provided exec helper
func ConfigureExecute(
	ctx context.Context,
	exec func(context.Context, *types.BackupConfigInfo) error,
	info *types.BackupConfigInfo,
) error {
	if info == nil {
		return nil
	}
	return exec(ctx, info)
}

// ExecuteBackupConfiguration verifies terraform paths and runs terraform using injected helpers
func ExecuteBackupConfiguration(
	ctx context.Context,
	deploymentPath string,
	info *types.BackupConfigInfo,
	buildArgs func(*types.BackupConfigInfo) []string,
	execTerraform func(context.Context, string, []string) error,
) error {
	if info == nil {
		return nil
	}
	tfRoot := fmt.Sprintf("%s/tokamak-thanos-stack/terraform", deploymentPath)
	thanosStackPath := fmt.Sprintf("%s/thanos-stack", tfRoot)
	if _, err := os.Stat(thanosStackPath); os.IsNotExist(err) {
		return fmt.Errorf("terraform thanos-stack directory not found: %s", thanosStackPath)
	}
	envrcPath := fmt.Sprintf("%s/.envrc", tfRoot)
	if _, err := os.Stat(envrcPath); os.IsNotExist(err) {
		return fmt.Errorf("terraform .envrc file not found: %s", envrcPath)
	}
	varArgs := buildArgs(info)
	return execTerraform(ctx, tfRoot, varArgs)
}

// helpers
func convertTimeToCron(timeStr string) string {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return ""
	}
	return fmt.Sprintf("cron(%s %s * * ? *)", parts[1], parts[0])
}

func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return strings.TrimSpace(*ptr)
}
func getBoolValue(ptr *bool) bool {
	if ptr == nil {
		return false
	}
	return *ptr
}
