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

// InteractiveRestore provides an interactive flow:
// - Detect current EFS and list recent recovery points
// - Prompt user to select a recovery point
// - Start restore, monitor progress, and finalize via provided callbacks
// - Optionally attach the restored EFS to workloads
func InteractiveRestore(
	ctx context.Context,
	l *zap.SugaredLogger,
	region, namespace string,
	restoreEFS func(context.Context, string) (string, error),
	monitorRestore func(context.Context, string) (string, error),
	handleCompletion func(context.Context, string) (string, error),
	executeAttach func(context.Context, string, *string, *string, *string) error,
) error {
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}
	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		return fmt.Errorf("failed to detect current EFS in namespace %s: %w", namespace, err)
	}
	resourceArn := utils.BuildEFSArn(region, accountID, efsID)

	query := "reverse(sort_by(RecoveryPoints,&CreationDate))[:20].{Arn:RecoveryPointArn,Created:CreationDate,Status:Status}"
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--region", region,
		"--resource-arn", resourceArn,
		"--query", query,
		"--output", "json",
	)
	if err != nil {
		return fmt.Errorf("failed to list recovery points: %w", err)
	}
	out = strings.TrimSpace(out)
	if out == "" || out == "[]" {
		return fmt.Errorf("no recovery points found for %s", resourceArn)
	}
	var items []struct {
		Arn     string `json:"Arn"`
		Created string `json:"Created"`
		Status  string `json:"Status"`
	}
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return fmt.Errorf("failed to parse recovery points: %w", err)
	}

	l.Info("Select a recovery point to restore:")
	l.Info("")

	// Display recovery points in a more user-friendly format
	for i, it := range items {
		l.Infof("  [%d] Created: %s", i, it.Created)
		l.Infof("      Status: %s", it.Status)
		l.Infof("      ARN: %s", it.Arn)
		l.Info("")
	}

	var sel int
	fmt.Print("Enter index: ")
	if _, err := fmt.Scanf("%d", &sel); err != nil || sel < 0 || sel >= len(items) {
		return fmt.Errorf("invalid selection")
	}
	selectedArn := items[sel].Arn

	jobID, err := restoreEFS(ctx, selectedArn)
	if err != nil {
		return err
	}
	newEfsID, err := monitorRestore(ctx, jobID)
	if err != nil {
		return err
	}

	l.Info("")
	l.Infof("✅ Restore completed. New EFS: %s", newEfsID)

	// Tag restored EFS with Name=<namespace> for easier identification
	if err := TagEFSWithName(ctx, region, newEfsID, namespace); err != nil {
		l.Warnf("Failed to tag EFS %s with Name=%s: %v", newEfsID, namespace, err)
	} else {
		l.Infof("✅ Tagged EFS %s with Name=%s", newEfsID, namespace)
	}

	// Ask user if they want to attach the restored EFS immediately
	l.Info("")
	var response string
	fmt.Print("Would you like to attach the restored EFS to workloads now? (y/n) ")
	if _, err := fmt.Scanf("%s", &response); err != nil {
		l.Warnf("Failed to read input: %v", err)
		response = "n"
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response == "y" || response == "yes" {
		l.Info("")
		l.Info("🔗 Starting attach process...")

		// Use default PVCs and StatefulSets for attach
		defaultPVCs := "op-geth,op-node"
		defaultSTSs := "op-geth,op-node"

		if err := executeAttach(ctx, newEfsID, &defaultPVCs, &defaultSTSs, nil); err != nil {
			l.Errorf("❌ Attach failed: %v", err)
			l.Info("You can manually attach later with:")
			l.Infof("  ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
			return err
		}

		l.Info("")
		l.Info("✅ Attach completed successfully!")
	} else {
		l.Info("")
		l.Info("You can attach workloads to the restored EFS later with:")
		l.Infof("  ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
	}

	return nil
}

// RestoreEFS starts restore job and returns the job ID
func RestoreEFS(ctx context.Context, region string, recoveryPointArn string, getIAMRole func(context.Context) (string, error)) (string, error) {
	if !strings.Contains(recoveryPointArn, "arn:aws:backup:") {
		return "", fmt.Errorf("invalid recovery point ARN format: %s", recoveryPointArn)
	}
	vaultsOutput, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults",
		"--region", region,
		"--query", "BackupVaultList[].BackupVaultName",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("failed to list backup vaults: %w", err)
	}
	vaultNames := strings.Fields(strings.TrimSpace(vaultsOutput))
	var foundVault string
	for _, vaultName := range vaultNames {
		if _, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", region,
			"--backup-vault-name", vaultName,
			"--recovery-point-arn", recoveryPointArn,
			"--query", "RecoveryPoint.BackupVaultName",
			"--output", "text"); err == nil {
			foundVault = vaultName
			break
		}
	}
	if foundVault == "" {
		if _, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", region,
			"--recovery-point-arn", recoveryPointArn,
			"--query", "RecoveryPoint.BackupVaultName",
			"--output", "text"); err != nil {
			return "", fmt.Errorf("recovery point not found or not accessible: %w", err)
		}
	}
	iamRoleArn, err := getIAMRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get IAM role for restore: %w", err)
	}
	// For EFS restore, we need to provide file-system-id, newfilesystem flag, creationtoken, and kmskeyid
	// Generate a unique filesystem ID and creation token using current timestamp
	fsID := fmt.Sprintf("fs-restored-%d", time.Now().Unix())
	creationToken := fmt.Sprintf("restore-token-%d", time.Now().Unix())
	// Use the same KMS key as the original EFS for consistency
	kmsKeyId := "arn:aws:kms:ap-northeast-2:778187209079:key/2dcda2df-c6f8-4d75-a2d5-e4d964f57f21"
	meta := fmt.Sprintf(`{"file-system-id":"%s","newfilesystem":"true","creationtoken":"%s","kmskeyid":"%s","performancemode":"generalPurpose","encrypted":"true"}`, fsID, creationToken, kmsKeyId)
	jobId, err := utils.ExecuteCommand(ctx,
		"aws", "backup", "start-restore-job",
		"--region", region,
		"--recovery-point-arn", recoveryPointArn,
		"--iam-role-arn", iamRoleArn,
		"--metadata", meta,
		"--query", "RestoreJobId",
		"--output", "text",
	)
	if err != nil {
		return "", fmt.Errorf("failed to start restore job: %w", err)
	}
	return strings.TrimSpace(jobId), nil
}

// MonitorEFSRestoreJob polls restore job until completion
func MonitorEFSRestoreJob(ctx context.Context, l *zap.SugaredLogger, region string, jobId string) (string, error) {
	const maxAttempts = 120
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("restore monitoring cancelled: %w", ctx.Err())
		default:
		}
		status, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
			"--region", region,
			"--restore-job-id", jobId,
			"--query", "Status",
			"--output", "text")
		if err != nil {
			return "", fmt.Errorf("failed to get restore job status: %w", err)
		}
		status = strings.TrimSpace(status)
		l.Infof("Job status: %s (attempt %d/%d)", status, i+1, maxAttempts)
		switch status {
		case "COMPLETED":
			createdArn, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
				"--region", region,
				"--restore-job-id", jobId,
				"--query", "CreatedResourceArn",
				"--output", "text")
			if err != nil {
				return "", fmt.Errorf("failed to get created resource ARN: %w", err)
			}
			createdArn = strings.TrimSpace(createdArn)
			l.Infof("Restore completed. CreatedResourceArn: %s", createdArn)
			if !strings.Contains(createdArn, ":file-system/") {
				return "", nil
			}
			parts := strings.Split(createdArn, "/")
			if len(parts) == 0 {
				return "", nil
			}
			newFsId := parts[len(parts)-1]
			if err := SetEFSThroughputElastic(ctx, region, newFsId); err != nil {
				l.Warnf("Failed to set EFS ThroughputMode to elastic: %v", err)
			} else {
				l.Info("✅ ThroughputMode set to elastic")
			}
			return newFsId, nil
		case "ABORTED", "FAILED":
			msg, _ := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
				"--region", region,
				"--restore-job-id", jobId,
				"--query", "StatusMessage",
				"--output", "text")
			return "", fmt.Errorf("restore job failed: %s", strings.TrimSpace(msg))
		}
		time.Sleep(30 * time.Second)
	}
	return "", fmt.Errorf("restore job monitoring timed out")
}

// HandleEFSRestoreCompletion extracts new EFS id and sets throughput
func HandleEFSRestoreCompletion(ctx context.Context, l *zap.SugaredLogger, region string, jobId string, setThroughput func(context.Context, string, string) error) (string, error) {
	createdArn, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
		"--region", region,
		"--restore-job-id", jobId,
		"--query", "CreatedResourceArn",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("failed to get created resource ARN: %w", err)
	}
	createdArn = strings.TrimSpace(createdArn)
	l.Infof("Restore completed. CreatedResourceArn: %s", createdArn)
	if !strings.Contains(createdArn, ":file-system/") {
		return "", nil
	}
	parts := strings.Split(createdArn, "/")
	if len(parts) == 0 {
		return "", nil
	}
	newFsId := parts[len(parts)-1]
	if err := setThroughput(ctx, region, newFsId); err != nil {
		l.Warnf("Failed to set EFS ThroughputMode to elastic: %v", err)
	} else {
		l.Info("✅ ThroughputMode set to elastic")
	}
	return newFsId, nil
}

// GetRestoreIAMRole tries managed and namespace-based roles to restore
func GetRestoreIAMRole(ctx context.Context, l *zap.SugaredLogger, region, namespace, accountID string) (string, error) {
	namespaceRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-restore-role", accountID, namespace)
	awsManagedRoles := []string{
		"AWSBackupDefaultServiceRole",
		fmt.Sprintf("%s-backup-service-role", namespace),
	}
	for _, roleName := range awsManagedRoles {
		var roleArn string
		if roleName == "AWSBackupDefaultServiceRole" {
			// AWS managed role has /service-role/ path
			roleArn = fmt.Sprintf("arn:aws:iam::%s:role/service-role/%s", accountID, roleName)
		} else {
			roleArn = fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
		}
		if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
			return roleArn, nil
		}
	}
	l.Warnf("No suitable IAM role found, using namespace-based role: %s", namespaceRoleArn)
	return namespaceRoleArn, nil
}

// SetEFSThroughputElastic updates an EFS to elastic throughput mode
func SetEFSThroughputElastic(ctx context.Context, region, fsId string) error {
	if strings.TrimSpace(fsId) == "" {
		return fmt.Errorf("empty file system id")
	}

	// First, check current EFS status and throughput mode
	describeOutput, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--file-system-id", fsId,
		"--query", "FileSystems[0].{ThroughputMode:ThroughputMode,LifeCycleState:LifeCycleState}",
		"--output", "json")
	if err != nil {
		return fmt.Errorf("failed to describe EFS %s: %w", fsId, err)
	}

	var efsInfo struct {
		ThroughputMode string `json:"ThroughputMode"`
		LifeCycleState string `json:"LifeCycleState"`
	}
	if err := json.Unmarshal([]byte(describeOutput), &efsInfo); err != nil {
		return fmt.Errorf("failed to parse EFS info: %w", err)
	}

	// Check if EFS is in a valid state for updates
	if efsInfo.LifeCycleState != "available" {
		return fmt.Errorf("EFS %s is not in available state (current: %s)", fsId, efsInfo.LifeCycleState)
	}

	// Check if throughput mode is already elastic
	if efsInfo.ThroughputMode == "elastic" {
		return nil // Already in elastic mode, no need to update
	}

	// Attempt to update throughput mode
	_, err = utils.ExecuteCommand(ctx, "aws", "efs", "update-file-system",
		"--region", region,
		"--file-system-id", fsId,
		"--throughput-mode", "elastic")

	if err != nil {
		// Check if it's a rate limiting error (24-hour limit)
		if strings.Contains(err.Error(), "254") || strings.Contains(err.Error(), "rate") {
			return fmt.Errorf("EFS throughput mode change rate limited (24-hour restriction). EFS %s will remain in %s mode", fsId, efsInfo.ThroughputMode)
		}
		return fmt.Errorf("failed to update EFS throughput mode: %w", err)
	}

	return nil
}

// TagEFSWithName applies a Name tag to the given EFS file system
func TagEFSWithName(ctx context.Context, region, fsId, name string) error {
	fsId = strings.TrimSpace(fsId)
	name = strings.TrimSpace(name)
	if fsId == "" || name == "" {
		return nil
	}
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "tag-resource",
		"--region", region,
		"--resource-id", fsId,
		"--tags", fmt.Sprintf("Key=Name,Value=%s", name),
	)
	return err
}
