package thanos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	backup "github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/backup"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// BackupStatus prints EFS backup status
func (t *ThanosStack) BackupStatus(ctx context.Context) (*types.BackupStatusInfo, error) {
	statusInfo, err := backup.GatherBackupStatusInfo(ctx, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace)
	if err != nil {
		return nil, err
	}
	backup.DisplayBackupStatus(t.logger, statusInfo)
	return statusInfo, nil
}

// BackupSnapshot triggers on-demand EFS backup and returns snapshot information
func (t *ThanosStack) BackupSnapshot(ctx context.Context, progressReporter func(string, float64)) (*types.BackupSnapshotInfo, error) {
	snapshotInfo, err := backup.SnapshotExecute(ctx, t.logger, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace, progressReporter)
	if err != nil {
		return nil, err
	}
	return snapshotInfo, nil
}

// BackupList lists recent EFS recovery points and returns comprehensive information
func (t *ThanosStack) BackupList(ctx context.Context, limit string) (*types.BackupListInfo, error) {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return nil, err
	}
	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		t.logger.Infof("📁 EFS Recovery Points")
		t.logger.Errorf("   ❌ Not detected in cluster PVs: %v", err)
		return nil, err
	}
	arn := utils.BuildEFSArn(region, accountID, efsID)

	if strings.TrimSpace(limit) == "" {
		limit = "20"
	}

	rps, err := backup.ListRecoveryPoints(ctx, region, arn, strings.TrimSpace(limit))
	if err != nil {
		t.logger.Infof("   ❌ Error retrieving recovery points: %v", err)
		return nil, err
	}
	backup.DisplayRecoveryPoints(t.logger, rps)

	// Return comprehensive backup list information
	return &types.BackupListInfo{
		Region:         region,
		Namespace:      namespace,
		EFSID:          efsID,
		ResourceARN:    arn,
		Limit:          limit,
		RecoveryPoints: rps,
	}, nil
}

// BackupRestore executes EFS restore from a recovery point ARN and returns restore information
func (t *ThanosStack) BackupRestore(ctx context.Context, recoveryPointArn string, attach *bool, pvcs *string, stss *string, progressReporter func(string, float64)) (*types.BackupRestoreInfo, error) {
	// Validate ARN
	if !strings.Contains(recoveryPointArn, "arn:aws:backup:") {
		return nil, fmt.Errorf("invalid recovery point ARN format: %s", recoveryPointArn)
	}

	// Get current EFS ID for tracking
	currentEfsID, err := utils.DetectEFSId(ctx, t.deployConfig.K8s.Namespace)
	if err != nil {
		currentEfsID = "" // Not critical, continue
	}

	restoreInfo, err := backup.DirectRestore(
		ctx,
		t.logger,
		t.deployConfig.AWS.Region,
		t.deployConfig.K8s.Namespace,
		recoveryPointArn,
		func(c context.Context, arn string) (string, error) {
			return backup.RestoreEFS(c, t.deployConfig.AWS.Region, arn, func(c2 context.Context) (string, error) {
				acct, err := utils.DetectAWSAccountID(c2)
				if err != nil {
					return "", err
				}
				return backup.GetRestoreIAMRole(c2, t.logger, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace, acct)
			})
		},
		func(c context.Context, job string, reporter func(string, float64)) (string, error) {
			return backup.MonitorEFSRestoreJob(c, t.logger, t.deployConfig.AWS.Region, job, reporter)
		},
		func(c context.Context, efsId string, pvcs, stss, other *string) error {
			defaultBackup := true
			_, err := t.BackupAttach(c, &efsId, pvcs, stss, &defaultBackup, progressReporter)
			return err
		},
		attach,
		pvcs,
		stss,
		progressReporter,
	)

	if err != nil {
		return nil, err
	}

	// Build and return BackupRestoreInfo
	accountID, _ := utils.DetectAWSAccountID(ctx)
	efsArn := utils.BuildEFSArn(t.deployConfig.AWS.Region, accountID, currentEfsID)

	return &types.BackupRestoreInfo{
		Region:           t.deployConfig.AWS.Region,
		Namespace:        t.deployConfig.K8s.Namespace,
		EFSID:            currentEfsID,
		ARN:              efsArn,
		RecoveryPointARN: recoveryPointArn,
		NewEFSID:         restoreInfo.NewEFSID,
		JobID:            restoreInfo.JobID,
		Status:           restoreInfo.Status,
		SuggestedEFSID:   restoreInfo.SuggestedEFSID,
		SuggestedPVCs:    restoreInfo.SuggestedPVCs,
		SuggestedSTSs:    restoreInfo.SuggestedSTSs,
	}, nil
}

// BackupRestoreInteractive provides interactive recovery point selection and restoration
func (t *ThanosStack) BackupRestoreInteractive(ctx context.Context) error {
	return backup.InteractiveRestoreWithSelection(
		ctx,
		t.logger,
		t.deployConfig.AWS.Region,
		t.deployConfig.K8s.Namespace,
		func(c context.Context, arn string) (*types.BackupRestoreInfo, error) {
			return t.BackupRestore(c, arn, nil, nil, nil, nil)
		},
	)
}

// BackupAttach switches workloads to use the new EFS and verifies readiness, returns attach information
func (t *ThanosStack) BackupAttach(ctx context.Context, efsId *string, pvcs *string, stss *string, backupPvPvc *bool, progressReporter func(string, float64)) (*types.BackupAttachInfo, error) {
	// gather info via backup subpackage
	info, err := backup.GatherBackupAttachInfo(
		ctx,
		t.deployConfig.K8s.Namespace,
		t.deployConfig.AWS.Region,
		efsId,
		pvcs,
		stss,
		func() { backup.ShowAttachUsage(t.logger) },
	)
	if err != nil {
		return nil, err
	}
	// execute via backup subpackage with injected helpers
	err = backup.ExecuteBackupAttach(
		ctx,
		t.logger,
		info,
		backup.ValidateAttachPrerequisites,
		func(c context.Context, ns string) error {
			return backup.VerifyEFSData(c, ns, func(ctx context.Context, namespace string) error {
				return backup.VerifyEFSDataImpl(ctx, t.logger, namespace)
			})
		},
		func(c context.Context, ns string, stsCSV string) error {
			return backup.RestartStatefulSets(c, t.logger, ns, stsCSV)
		},
		func(c context.Context, ai *types.BackupAttachInfo, pr func(string, float64)) error {
			return backup.ExecuteEFSOperationsFull(c, t.logger, ai, func(ctx context.Context, namespace string) error {
				return backup.VerifyEFSDataImpl(ctx, t.logger, namespace)
			}, backupPvPvc, pr)
		},
		progressReporter,
	)
	if err != nil {
		return nil, err
	}

	// Update status after successful execution
	info.Status = "attached"
	return info, nil
}

// BackupPvPvcExport generates PV/PVC backup artifacts and returns the output directory.
func (t *ThanosStack) BackupPvPvcExport(ctx context.Context) (string, error) {
	backupDir := filepath.Join(t.deploymentPath, "k8s-efs-backup")
	return backup.BackupPvPvcToDir(ctx, t.logger, t.deployConfig.K8s.Namespace, backupDir)
}

// BackupConfigure applies EFS backup configuration via Terraform and returns configuration info
func (t *ThanosStack) BackupConfigure(ctx context.Context, daily *string, keep *string, reset *bool) (*types.BackupConfigInfo, error) {
	info, err := backup.GatherBackupConfigInfo(
		t.deployConfig.AWS.Region,
		t.deployConfig.K8s.Namespace,
		daily, keep, reset,
		func(format string, args ...any) { t.logger.Infof(format, args...) },
	)
	if err != nil {
		return nil, err
	}
	buildArgs := func(ci *types.BackupConfigInfo) []string {
		return backup.BuildTerraformArgs(ci, func(format string, args ...any) { t.logger.Infof(format, args...) })
	}
	execTf := func(c context.Context, root string, args []string) error {
		return backup.ExecuteTerraformCommands(
			c,
			root,
			args,
			func(format string, a ...any) { t.logger.Infof(format, a...) },
			func(format string, a ...any) { t.logger.Warnf(format, a...) },
		)
	}
	err = backup.ExecuteBackupConfiguration(ctx, t.deploymentPath, info, buildArgs, execTf)
	if err != nil {
		return nil, err
	}

	// Update status after successful execution
	info.Status = "applied"
	return info, nil
}

// CleanupUnusedBackupResources removes unused EFS filesystems and old recovery points during deploy
func (t *ThanosStack) CleanupUnusedBackupResources(ctx context.Context) error {
	// Check if deployConfig is available
	if t.deployConfig == nil {
		return fmt.Errorf("deployConfig is not available - cannot cleanup backup resources")
	}
	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not available - cannot cleanup backup resources")
	}
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("kubernetes configuration is not available - cannot cleanup backup resources")
	}

	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	retentionDays := 14
	if v := os.Getenv("TRH_EFS_CLEANUP_RETENTION_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			retentionDays = parsed
		}
	}

	return backup.CleanupUnusedBackupResources(ctx, t.logger, region, namespace, retentionDays)
}

// initializeBackupSystem initializes or reconciles the AWS Backup configuration for the current stack
func (t *ThanosStack) initializeBackupSystem(ctx context.Context, chainName string) error {
	// Check if deployConfig is available
	if t.deployConfig == nil {
		return fmt.Errorf("deployConfig is not available - cannot initialize backup system")
	}
	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not available - cannot initialize backup system")
	}
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("kubernetes configuration is not available - cannot initialize backup system")
	}

	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	return backup.InitializeBackupSystem(ctx, t.logger, region, namespace, chainName)
}
