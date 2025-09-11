package thanos

import (
	"context"
	"fmt"
	"strings"

	backup "github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/backup"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// BackupStatus prints EFS backup status
func (t *ThanosStack) BackupStatus(ctx context.Context) error {
	statusInfo, err := backup.GatherBackupStatusInfo(ctx, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace)
	if err != nil {
		return err
	}
	backup.DisplayBackupStatus(t.logger, statusInfo)
	return nil
}

// BackupSnapshot triggers on-demand EFS backup
func (t *ThanosStack) BackupSnapshot(ctx context.Context) error {
	return backup.SnapshotExecute(ctx, t.logger, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace)
}

// BackupList lists recent EFS recovery points
func (t *ThanosStack) BackupList(ctx context.Context, limit string) error {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return err
	}
	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		t.logger.Infof("📁 EFS Recovery Points")
		t.logger.Errorf("   ❌ Not detected in cluster PVs: %v", err)
		return nil
	}
	arn := utils.BuildEFSArn(region, accountID, efsID)

	rps, err := backup.ListRecoveryPoints(ctx, region, arn, strings.TrimSpace(limit))
	if err != nil {
		t.logger.Infof("   ❌ Error retrieving recovery points: %v", err)
		return nil
	}
	backup.DisplayRecoveryPoints(t.logger, rps)
	return nil
}

// BackupRestore provides a fully interactive restore experience for EFS
func (t *ThanosStack) BackupRestore(ctx context.Context) error {
	return backup.InteractiveRestore(
		ctx,
		t.logger,
		t.deployConfig.AWS.Region,
		t.deployConfig.K8s.Namespace,
		func(c context.Context, arn string) (string, error) {
			return backup.RestoreEFS(c, t.deployConfig.AWS.Region, arn, func(c2 context.Context) (string, error) {
				acct, err := utils.DetectAWSAccountID(c2)
				if err != nil {
					return "", err
				}
				return backup.GetRestoreIAMRole(c2, t.logger, t.deployConfig.AWS.Region, t.deployConfig.K8s.Namespace, acct)
			})
		},
		func(c context.Context, job string) (string, error) {
			return backup.MonitorEFSRestoreJob(c, t.logger, t.deployConfig.AWS.Region, job)
		},
		func(c context.Context, job string) (string, error) {
			return backup.HandleEFSRestoreCompletion(c, t.logger, t.deployConfig.AWS.Region, job, backup.SetEFSThroughputElastic)
		},
		func(c context.Context, efsId string, pvcs, stss, other *string) error {
			// Use the same attach logic as BackupAttach
			return t.BackupAttach(c, &efsId, pvcs, stss)
		},
	)
}

// BackupAttach switches workloads to use the new EFS and verifies readiness
func (t *ThanosStack) BackupAttach(ctx context.Context, efsId *string, pvcs *string, stss *string) error {
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
		return err
	}
	// execute via backup subpackage with injected helpers
	return backup.ExecuteBackupAttach(
		ctx,
		t.logger,
		info,
		backup.ValidateAttachPrerequisites,
		func(c context.Context, ns string) error {
			return backup.VerifyEFSData(c, ns, func(ctx context.Context, namespace string) error {
				return backup.VerifyEFSDataImpl(ctx, t.logger, namespace)
			})
		},
		backup.RestartStatefulSets,
		func(c context.Context, ai *types.BackupAttachInfo) error {
			return backup.ExecuteEFSOperationsFull(c, t.logger, ai, func(ctx context.Context, namespace string) error {
				return backup.VerifyEFSDataImpl(ctx, t.logger, namespace)
			})
		},
	)
}

// BackupConfigure applies EFS backup configuration via Terraform
func (t *ThanosStack) BackupConfigure(ctx context.Context, daily *string, keep *string, reset *bool) error {
	info, err := backup.GatherBackupConfigInfo(
		t.deployConfig.AWS.Region,
		t.deployConfig.K8s.Namespace,
		daily, keep, reset,
		func(format string, args ...any) { t.logger.Infof(format, args...) },
	)
	if err != nil {
		return err
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
	return backup.ExecuteBackupConfiguration(ctx, t.deploymentPath, info, buildArgs, execTf)
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

	return backup.CleanupUnusedBackupResources(ctx, t.logger, region, namespace)
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
