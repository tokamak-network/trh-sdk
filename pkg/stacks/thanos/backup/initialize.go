package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// detectEFS detects EFS ID for the given namespace
func detectEFS(ctx context.Context, namespace string) (string, error) {
	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		return "", fmt.Errorf("failed to detect EFS for namespace %s: %w", namespace, err)
	}

	return strings.TrimSpace(efsID), nil
}

// executeInitialBackup starts and monitors the initial backup job
func executeInitialBackup(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, accountID, efsID, namespace string) error {
	l.Info("Executing initial backup...")

	jobID, err := startBackupJob(ctx, ar, region, accountID, efsID, namespace)
	if err != nil {
		return fmt.Errorf("failed to start backup job: %w", err)
	}

	l.Infof("Initial backup job started: %s", jobID)

	go func() {
		if err := waitForBackupCompletion(ctx, ar, l, region, jobID); err != nil {
			l.Warnf("Initial backup monitoring failed: %v", err)
		} else {
			l.Info("✅ Initial backup completed successfully")
		}
	}()

	l.Info("✅ Initial backup job initiated")
	return nil
}

// startBackupJob starts a backup job for the specified EFS
// Note: Backup vault is already created by Terraform
func startBackupJob(ctx context.Context, ar runner.AWSRunner, region, accountID, efsID, namespace string) (string, error) {
	iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/service-role/AWSBackupDefaultServiceRole", accountID)
	efsArn := utils.BuildEFSArn(region, accountID, efsID)
	vaultName := fmt.Sprintf("%s-backup-vault", namespace)

	if ar != nil {
		return ar.BackupStartBackupJob(ctx, region, vaultName, efsArn, iamRoleArn)
	}

	jobID, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--region", region,
		"--backup-vault-name", vaultName,
		"--resource-arn", efsArn,
		"--iam-role-arn", iamRoleArn,
		"--query", "BackupJobId",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("failed to start backup job: %w", err)
	}

	return strings.TrimSpace(jobID), nil
}

// waitForBackupCompletion monitors backup job until completion
func waitForBackupCompletion(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, jobID string) error {
	l.Infof("Monitoring backup job: %s", jobID)

	for i := 0; i < 60; i++ {
		var status string
		if ar != nil {
			s, err := ar.BackupDescribeBackupJob(ctx, region, jobID)
			if err != nil {
				return fmt.Errorf("failed to get backup job status: %w", err)
			}
			status = s
		} else {
			s, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-backup-job",
				"--region", region,
				"--backup-job-id", jobID,
				"--query", "State",
				"--output", "text")
			if err != nil {
				return fmt.Errorf("failed to get backup job status: %w", err)
			}
			status = strings.TrimSpace(s)
		}
		l.Infof("Backup job status: %s", status)

		switch status {
		case "COMPLETED":
			return nil
		case "FAILED", "ABORTED":
			return fmt.Errorf("backup job %s failed with status: %s", jobID, status)
		case "RUNNING", "CREATED":
			// Continue waiting
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("backup job %s timed out after 10 minutes", jobID)
}

// InitializeBackupSystem initializes the AWS Backup system for the current stack
func InitializeBackupSystem(ctx context.Context, ar runner.AWSRunner, l *zap.SugaredLogger, region, namespace, chainName string) error {
	l.Infof("Initializing backup system (chain: %s, ns: %s, region: %s)", chainName, namespace, region)

	// 1. Detect EFS ID for the namespace
	efsID, err := detectEFS(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to detect EFS: %w", err)
	}

	// 2. Get AWS account ID for backup job
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// 3. Execute initial backup job (infrastructure is already set up by Terraform)
	if err := executeInitialBackup(ctx, ar, l, region, accountID, efsID, namespace); err != nil {
		return fmt.Errorf("failed to execute initial backup: %w", err)
	}
	return nil
}
