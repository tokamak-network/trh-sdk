package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// CleanupUnusedBackupResources removes unused EFS filesystems and old recovery points during deploy
func CleanupUnusedBackupResources(
	ctx context.Context,
	ar runner.AWSRunner,
	l *zap.SugaredLogger,
	region, namespace string,
	retentionDays int,
) error {
	l.Info("🧹 Cleaning up unused backup resources...")
	if retentionDays <= 0 {
		retentionDays = 14
	}

	// Get account ID
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// 1. Cleanup old recovery points
	if err := CleanupOldRecoveryPoints(ctx, ar, l, region, namespace, accountID, retentionDays); err != nil {
		l.Warnf("Failed to cleanup old recovery points: %v", err)
	}

	// 2. Cleanup unused EFS filesystems
	if err := CleanupUnusedEFS(ctx, ar, l, region, namespace, retentionDays); err != nil {
		l.Warnf("Failed to cleanup unused EFS: %v", err)
	}

	// 3. Cleanup backup vaults with namespace prefix
	if err := CleanupBackupVaults(ctx, ar, l, region, namespace); err != nil {
		l.Warnf("Failed to cleanup backup vaults: %v", err)
	}

	l.Info("✅ Backup resources cleanup completed")
	return nil
}

// CleanupOldRecoveryPoints removes recovery points older than retentionDays
func CleanupOldRecoveryPoints(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, namespace, accountID string, retentionDays int) error {
	l.Info("🗑️ Cleaning up old recovery points...")

	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		l.Warn("Could not detect current EFS ID, skipping recovery point cleanup")
		return nil
	}

	arn := utils.BuildEFSArn(region, accountID, currentEfsID)

	if retentionDays <= 0 {
		retentionDays = 14
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)

	type rpEntry struct {
		Arn     string
		Created string
	}

	var recoveryPoints []rpEntry
	if ar != nil {
		rps, err := ar.BackupListRecoveryPointsByResource(ctx, region, arn)
		if err != nil {
			return fmt.Errorf("failed to list old recovery points: %w", err)
		}
		for _, rp := range rps {
			if rp.CreationDate.Before(cutoff) || rp.CreationDate.Equal(cutoff) {
				recoveryPoints = append(recoveryPoints, rpEntry{
					Arn:     rp.RecoveryPointArn,
					Created: rp.CreationDate.Format(time.RFC3339),
				})
			}
		}
	} else {
		cutoffDate := cutoff.Format("2006-01-02T15:04:05.000000-07:00")
		rpList, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
			"--region", region,
			"--resource-arn", arn,
			"--query", fmt.Sprintf("RecoveryPoints[?CreationDate<='%s'].{Arn:RecoveryPointArn,Created:CreationDate}", cutoffDate),
			"--output", "json")
		if err != nil {
			return fmt.Errorf("failed to list old recovery points: %w", err)
		}
		if strings.TrimSpace(rpList) == "[]" || strings.TrimSpace(rpList) == "" {
			l.Info("No old recovery points found to cleanup")
			return nil
		}
		var parsed []struct {
			Arn     string `json:"Arn"`
			Created string `json:"Created"`
		}
		if err := json.Unmarshal([]byte(rpList), &parsed); err != nil {
			return fmt.Errorf("failed to parse recovery points list: %w", err)
		}
		for _, p := range parsed {
			recoveryPoints = append(recoveryPoints, rpEntry{Arn: p.Arn, Created: p.Created})
		}
	}

	if len(recoveryPoints) == 0 {
		l.Info("No old recovery points found to cleanup")
		return nil
	}

	vaultName := fmt.Sprintf("%s-backup-vault", namespace)
	deletedCount := 0
	for _, rp := range recoveryPoints {
		l.Infof("Deleting old recovery point: %s (created: %s)", rp.Arn, rp.Created)

		if ar != nil {
			err = ar.BackupDeleteRecoveryPoint(ctx, region, vaultName, rp.Arn)
		} else {
			_, err = utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
				"--region", region,
				"--backup-vault-name", vaultName,
				"--recovery-point-arn", rp.Arn)
		}

		if err != nil {
			l.Warnf("Failed to delete recovery point %s: %v", rp.Arn, err)
		} else {
			deletedCount++
			l.Infof("✅ Deleted recovery point: %s", rp.Arn)
		}
	}

	if deletedCount > 0 {
		l.Infof("✅ Deleted %d old recovery points", deletedCount)
	}

	return nil
}

// CleanupUnusedEFS removes EFS filesystems that are not currently in use
func CleanupUnusedEFS(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, namespace string, retentionDays int) error {
	l.Info("🗑️ Cleaning up unused EFS filesystems...")
	if retentionDays <= 0 {
		retentionDays = 14
	}

	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		l.Warn("Could not detect current EFS ID, skipping EFS cleanup")
		return nil
	}

	type efsEntry struct {
		Id      string
		Name    string
		State   string
		Created string
	}

	var efsData []efsEntry
	if ar != nil {
		fsList, err := ar.EFSDescribeFileSystems(ctx, region, "")
		if err != nil {
			return fmt.Errorf("failed to list EFS filesystems: %w", err)
		}
		for _, fs := range fsList {
			if fs.FileSystemID != currentEfsID {
				efsData = append(efsData, efsEntry{
					Id:      fs.FileSystemID,
					Name:    fs.Name,
					State:   fs.LifeCycleState,
					Created: fs.CreationTime.Format(time.RFC3339),
				})
			}
		}
	} else {
		efsList, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
			"--region", region,
			"--query", "FileSystems[?FileSystemId != `"+currentEfsID+"`].{Id:FileSystemId,Name:Name,State:LifeCycleState,Created:CreationTime}",
			"--output", "json")
		if err != nil {
			return fmt.Errorf("failed to list EFS filesystems: %w", err)
		}
		if strings.TrimSpace(efsList) == "[]" || strings.TrimSpace(efsList) == "" {
			l.Info("No unused EFS filesystems found to cleanup")
			return nil
		}
		var parsed []struct {
			Id      string `json:"Id"`
			Name    string `json:"Name"`
			State   string `json:"State"`
			Created string `json:"Created"`
		}
		if err := json.Unmarshal([]byte(efsList), &parsed); err != nil {
			return fmt.Errorf("failed to parse EFS list JSON: %w", err)
		}
		for _, p := range parsed {
			efsData = append(efsData, efsEntry{Id: p.Id, Name: p.Name, State: p.State, Created: p.Created})
		}
	}

	var unusedEFS []string
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	for _, fs := range efsData {
		if fs.Name == "" || !strings.HasPrefix(fs.Name, namespace) {
			continue
		}
		if fs.Created != "" {
			createdAt, parseErr := time.Parse(time.RFC3339, fs.Created)
			if parseErr != nil {
				createdAt, parseErr = time.Parse(time.RFC3339Nano, fs.Created)
			}
			if parseErr == nil && createdAt.After(cutoff) {
				continue
			}
		}
		if fs.State == "available" {
			unusedEFS = append(unusedEFS, fs.Id)
		}
	}

	if len(unusedEFS) == 0 {
		l.Info("No unused EFS filesystems found to cleanup")
		return nil
	}

	deletedCount := 0
	for _, efsID := range unusedEFS {
		l.Infof("Deleting unused EFS: %s", efsID)

		if err := DeleteEFSMountTargets(ctx, ar, l, region, efsID); err != nil {
			l.Warnf("Failed to delete mount targets for %s: %v", efsID, err)
			continue
		}

		time.Sleep(10 * time.Second)

		if ar != nil {
			err = ar.EFSDeleteFileSystem(ctx, region, efsID)
		} else {
			_, err = utils.ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
				"--region", region,
				"--file-system-id", efsID)
		}

		if err != nil {
			l.Warnf("Failed to delete EFS %s: %v", efsID, err)
		} else {
			deletedCount++
			l.Infof("✅ Deleted EFS filesystem: %s", efsID)
		}
	}

	if deletedCount > 0 {
		l.Infof("✅ Deleted %d unused EFS filesystems", deletedCount)
	}

	return nil
}

// DeleteEFSMountTargets removes all mount targets for an EFS filesystem
func DeleteEFSMountTargets(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, efsId string) error {
	if ar != nil {
		mts, err := ar.EFSDescribeMountTargets(ctx, region, efsId)
		if err != nil {
			return fmt.Errorf("failed to list mount targets: %w", err)
		}
		for _, mt := range mts {
			if mt.MountTargetID == "" {
				continue
			}
			l.Infof("Deleting mount target: %s", mt.MountTargetID)
			if err := ar.EFSDeleteMountTarget(ctx, region, mt.MountTargetID); err != nil {
				l.Warnf("Failed to delete mount target %s: %v", mt.MountTargetID, err)
			}
		}
		return nil
	}

	// Fallback: shellout
	mtList, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets",
		"--region", region,
		"--file-system-id", efsId,
		"--query", "MountTargets[].MountTargetId",
		"--output", "text")
	if err != nil {
		return fmt.Errorf("failed to list mount targets: %w", err)
	}

	mountTargets := strings.Fields(strings.TrimSpace(mtList))
	for _, mtId := range mountTargets {
		if mtId == "" || mtId == "None" {
			continue
		}
		l.Infof("Deleting mount target: %s", mtId)
		if _, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
			"--region", region,
			"--mount-target-id", mtId); err != nil {
			l.Warnf("Failed to delete mount target %s: %v", mtId, err)
		}
	}

	return nil
}

// CleanupBackupVaults removes backup vaults with namespace prefix
func CleanupBackupVaults(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, namespace string) error {
	l.Info("🗑️ Cleaning up backup vaults...")

	vaultPattern := namespace + "-"

	var vaultNames []string
	if ar != nil {
		vaults, err := ar.BackupListBackupVaults(ctx, region)
		if err != nil {
			return fmt.Errorf("failed to list backup vaults: %w", err)
		}
		for _, v := range vaults {
			if strings.HasPrefix(v.BackupVaultName, vaultPattern) {
				vaultNames = append(vaultNames, v.BackupVaultName)
			}
		}
	} else {
		vaultsList, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults",
			"--region", region,
			"--query", "BackupVaultList[?starts_with(BackupVaultName, `"+vaultPattern+"`)]",
			"--output", "json")
		if err != nil {
			return fmt.Errorf("failed to list backup vaults: %w", err)
		}
		if strings.TrimSpace(vaultsList) == "[]" || strings.TrimSpace(vaultsList) == "" {
			l.Infof("No backup vaults found with prefix: %s-*", namespace)
			return nil
		}
		var parsed []struct {
			BackupVaultName string `json:"BackupVaultName"`
		}
		if err := json.Unmarshal([]byte(vaultsList), &parsed); err != nil {
			return fmt.Errorf("failed to parse backup vaults list: %w", err)
		}
		for _, v := range parsed {
			vaultNames = append(vaultNames, v.BackupVaultName)
		}
	}

	if len(vaultNames) == 0 {
		l.Infof("No backup vaults found with prefix: %s-*", namespace)
		return nil
	}

	deletedCount := 0
	for _, name := range vaultNames {
		l.Infof("Processing backup vault: %s", name)

		if err := DeleteAllRecoveryPointsInVault(ctx, ar, l, region, name); err != nil {
			l.Warnf("Failed to delete recovery points in vault %s: %v", name, err)
			continue
		}

		time.Sleep(5 * time.Second)

		l.Infof("Deleting backup vault: %s", name)

		var err error
		if ar != nil {
			err = ar.BackupDeleteBackupVault(ctx, region, name)
		} else {
			_, err = utils.ExecuteCommand(ctx, "aws", "backup", "delete-backup-vault",
				"--region", region,
				"--backup-vault-name", name)
		}

		if err != nil {
			l.Warnf("Failed to delete backup vault %s: %v", name, err)
		} else {
			deletedCount++
			l.Infof("✅ Deleted backup vault: %s", name)
		}
	}

	if deletedCount > 0 {
		l.Infof("✅ Deleted %d backup vaults", deletedCount)
	}

	return nil
}

// DeleteAllRecoveryPointsInVault removes all recovery points from a backup vault
func DeleteAllRecoveryPointsInVault(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, vaultName string) error {
	l.Infof("Deleting all recovery points in vault: %s", vaultName)

	var recoveryPointArns []string
	if ar != nil {
		arns, err := ar.BackupListRecoveryPointsByVault(ctx, region, vaultName)
		if err != nil {
			return fmt.Errorf("failed to list recovery points in vault %s: %w", vaultName, err)
		}
		recoveryPointArns = arns
	} else {
		rpList, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-backup-vault",
			"--region", region,
			"--backup-vault-name", vaultName,
			"--query", "RecoveryPoints[].RecoveryPointArn",
			"--output", "json")
		if err != nil {
			return fmt.Errorf("failed to list recovery points in vault %s: %w", vaultName, err)
		}
		if strings.TrimSpace(rpList) == "[]" || strings.TrimSpace(rpList) == "" {
			l.Infof("No recovery points found in vault: %s", vaultName)
			return nil
		}
		if err := json.Unmarshal([]byte(rpList), &recoveryPointArns); err != nil {
			return fmt.Errorf("failed to parse recovery points list for vault %s: %w", vaultName, err)
		}
	}

	if len(recoveryPointArns) == 0 {
		l.Infof("No recovery points found in vault: %s", vaultName)
		return nil
	}

	deletedCount := 0
	for _, rpArn := range recoveryPointArns {
		l.Infof("Deleting recovery point: %s", rpArn)

		var err error
		if ar != nil {
			err = ar.BackupDeleteRecoveryPoint(ctx, region, vaultName, rpArn)
		} else {
			_, err = utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
				"--region", region,
				"--backup-vault-name", vaultName,
				"--recovery-point-arn", rpArn)
		}

		if err != nil {
			l.Warnf("Failed to delete recovery point %s: %v", rpArn, err)
		} else {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		l.Infof("Deleted %d recovery points from vault: %s", deletedCount, vaultName)
	}

	return nil
}
