package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// CleanupUnusedBackupResources removes unused EFS filesystems and old recovery points during deploy
func CleanupUnusedBackupResources(
	ctx context.Context,
	l *zap.SugaredLogger,
	region, namespace string,
) error {
	l.Info("🧹 Cleaning up unused backup resources...")

	// Get account ID
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// 1. Cleanup old recovery points
	if err := CleanupOldRecoveryPoints(ctx, l, region, namespace, accountID); err != nil {
		l.Warnf("Failed to cleanup old recovery points: %v", err)
	}

	// 2. Cleanup unused EFS filesystems
	if err := CleanupUnusedEFS(ctx, l, region, namespace); err != nil {
		l.Warnf("Failed to cleanup unused EFS: %v", err)
	}

	// 3. Cleanup backup vaults with namespace prefix
	if err := CleanupBackupVaults(ctx, l, region, namespace); err != nil {
		l.Warnf("Failed to cleanup backup vaults: %v", err)
	}

	l.Info("✅ Backup resources cleanup completed")
	return nil
}

// CleanupOldRecoveryPoints removes recovery points older than 7 days
func CleanupOldRecoveryPoints(ctx context.Context, l *zap.SugaredLogger, region, namespace, accountID string) error {
	l.Info("🗑️ Cleaning up old recovery points...")

	// Get current EFS ID to find its recovery points
	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		l.Warn("Could not detect current EFS ID, skipping recovery point cleanup")
		return nil
	}

	arn := utils.BuildEFSArn(region, accountID, currentEfsID)

	// List recovery points older than 7 days
	cutoffDate := time.Now().AddDate(0, 0, -7).Format("2006-01-02T15:04:05.000000-07:00")

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

	// Parse and delete old recovery points
	var recoveryPoints []struct {
		Arn     string `json:"Arn"`
		Created string `json:"Created"`
	}

	if err := json.Unmarshal([]byte(rpList), &recoveryPoints); err != nil {
		return fmt.Errorf("failed to parse recovery points list: %w", err)
	}

	deletedCount := 0
	for _, rp := range recoveryPoints {
		l.Infof("Deleting old recovery point: %s (created: %s)", rp.Arn, rp.Created)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
			"--region", region,
			"--backup-vault-name", fmt.Sprintf("%s-backup-vault", namespace),
			"--recovery-point-arn", rp.Arn)

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
func CleanupUnusedEFS(ctx context.Context, l *zap.SugaredLogger, region, namespace string) error {
	l.Info("🗑️ Cleaning up unused EFS filesystems...")

	// Get current EFS ID in use
	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		l.Warn("Could not detect current EFS ID, skipping EFS cleanup")
		return nil
	}

	// List all EFS filesystems

	efsList, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--query", "FileSystems[?FileSystemId != `"+currentEfsID+"`].{Id:FileSystemId,Name:Name,State:LifeCycleState}",
		"--output", "json")

	if err != nil {
		return fmt.Errorf("failed to list EFS filesystems: %w", err)
	}

	if strings.TrimSpace(efsList) == "[]" || strings.TrimSpace(efsList) == "" {
		l.Info("No unused EFS filesystems found to cleanup")
		return nil
	}

	// Parse JSON and filter by namespace pattern
	var efsData []struct {
		Id    string `json:"Id"`
		Name  string `json:"Name"`
		State string `json:"State"`
	}
	if err := json.Unmarshal([]byte(efsList), &efsData); err != nil {
		return fmt.Errorf("failed to parse EFS list JSON: %w", err)
	}

	// Filter EFS by namespace pattern
	var unusedEFS []string
	for _, efs := range efsData {
		// Skip if Name is null or doesn't match namespace pattern
		if efs.Name == "" || !strings.HasPrefix(efs.Name, namespace) {
			continue
		}
		if efs.State == "available" {
			unusedEFS = append(unusedEFS, efs.Id)
		}
	}

	if len(unusedEFS) == 0 {
		l.Info("No unused EFS filesystems found to cleanup")
		return nil
	}

	// Delete unused EFS filesystems
	deletedCount := 0
	for _, efsID := range unusedEFS {

		l.Infof("Deleting unused EFS: %s", efsID)

		// Delete mount targets first
		if err := DeleteEFSMountTargets(ctx, l, region, efsID); err != nil {
			l.Warnf("Failed to delete mount targets for %s: %v", efsID, err)
			continue
		}

		// Wait for mount targets to be deleted
		time.Sleep(10 * time.Second)

		// Delete the EFS filesystem
		_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
			"--region", region,
			"--file-system-id", efsID)

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
func DeleteEFSMountTargets(ctx context.Context, l *zap.SugaredLogger, region, efsId string) error {
	// List mount targets
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

		_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
			"--region", region,
			"--mount-target-id", mtId)

		if err != nil {
			l.Warnf("Failed to delete mount target %s: %v", mtId, err)
		}
	}

	return nil
}

// CleanupBackupVaults removes backup vaults with namespace prefix
func CleanupBackupVaults(ctx context.Context, l *zap.SugaredLogger, region, namespace string) error {
	l.Info("🗑️ Cleaning up backup vaults...")

	// List all backup vaults with namespace prefix
	// Pattern matches:
	// - {namespace}-backup-vault (created by BackupSnapshot and Terraform)
	// - {namespace}-* (any other namespace-prefixed vaults)
	vaultPattern := namespace + "-"

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

	// Parse vault list
	var vaults []struct {
		BackupVaultName string `json:"BackupVaultName"`
	}

	if err := json.Unmarshal([]byte(vaultsList), &vaults); err != nil {
		return fmt.Errorf("failed to parse backup vaults list: %w", err)
	}

	deletedCount := 0
	for _, vault := range vaults {
		l.Infof("Processing backup vault: %s", vault.BackupVaultName)

		// First, delete all recovery points in the vault
		if err := DeleteAllRecoveryPointsInVault(ctx, l, region, vault.BackupVaultName); err != nil {
			l.Warnf("Failed to delete recovery points in vault %s: %v", vault.BackupVaultName, err)
			continue
		}

		// Wait a bit for recovery points to be fully deleted
		time.Sleep(5 * time.Second)

		// Delete the backup vault
		l.Infof("Deleting backup vault: %s", vault.BackupVaultName)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-backup-vault",
			"--region", region,
			"--backup-vault-name", vault.BackupVaultName)

		if err != nil {
			l.Warnf("Failed to delete backup vault %s: %v", vault.BackupVaultName, err)
		} else {
			deletedCount++
			l.Infof("✅ Deleted backup vault: %s", vault.BackupVaultName)
		}
	}

	if deletedCount > 0 {
		l.Infof("✅ Deleted %d backup vaults", deletedCount)
	}

	return nil
}

// DeleteAllRecoveryPointsInVault removes all recovery points from a backup vault
func DeleteAllRecoveryPointsInVault(ctx context.Context, l *zap.SugaredLogger, region, vaultName string) error {
	l.Infof("Deleting all recovery points in vault: %s", vaultName)

	// List all recovery points in the vault
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

	// Parse recovery points list
	var recoveryPointArns []string
	if err := json.Unmarshal([]byte(rpList), &recoveryPointArns); err != nil {
		return fmt.Errorf("failed to parse recovery points list for vault %s: %w", vaultName, err)
	}

	deletedCount := 0
	for _, rpArn := range recoveryPointArns {
		l.Infof("Deleting recovery point: %s", rpArn)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
			"--region", region,
			"--backup-vault-name", vaultName,
			"--recovery-point-arn", rpArn)

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
