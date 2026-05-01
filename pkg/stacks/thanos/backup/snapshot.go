package backup

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// SnapshotExecute triggers an on-demand EFS backup and returns snapshot information
// SnapshotExecute triggers an on-demand EFS backup and returns snapshot information
func SnapshotExecute(ctx context.Context, l *zap.SugaredLogger, region, namespace string, progressReporter func(string, float64)) (*types.BackupSnapshotInfo, error) {
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return nil, err
	}

	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		l.Infof("📁 EFS: ❌ Not detected in cluster PVs: %v", err)
		return nil, fmt.Errorf("failed to detect EFS: %w", err)
	}

	arn := utils.BuildEFSArn(region, accountID, efsID)
	backupVaultName := fmt.Sprintf("%s-backup-vault", namespace)
	iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-service-role", accountID, namespace)

	if progressReporter != nil {
		progressReporter("Starting backup job...", 10.0)
	}

	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--region", region,
		"--backup-vault-name", backupVaultName,
		"--resource-arn", arn,
		"--iam-role-arn", iamRoleArn,
		"--query", "BackupJobId",
		"--output", "text",
	)
	if err != nil {
		l.Errorf("📁 EFS: ❌ Failed to start backup job: %v", err)
		l.Infof("   Backup vault: %s", backupVaultName)
		l.Infof("   IAM role: %s", iamRoleArn)
		return nil, fmt.Errorf("failed to start backup job: %w", err)
	}

	jobID := strings.TrimSpace(out)
	l.Infof("📁 EFS: ✅ On-demand backup started successfully")
	l.Infof("   Job ID: %s", jobID)

	// Monitor the job until completion if a progress reporter is provided
	// or we can just monitor it always to ensure we return only when done or failed?
	// The original implementation returned immediately. The new requirement implies we should track it.
	// We will track it.

	const maxAttempts = 120
	// We'll estimate progress based on attempts for now since AWS Backup doesn't give percentage for EFS easily
	// Start at 10%, max at 90% before completion

	l.Infof("⏳ Monitoring backup job %s...", jobID)

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			status, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-backup-job",
				"--region", region,
				"--backup-job-id", jobID,
				"--query", "State",
				"--output", "text",
			)
			if err != nil {
				l.Warnf("Failed to check backup status: %v", err)
				continue
			}
			status = strings.TrimSpace(status)

			if progressReporter != nil {
				// Fake progress interpolation
				percent := 10.0 + (float64(i)/float64(maxAttempts))*80.0
				if percent > 90 {
					percent = 90
				}
				progressReporter(fmt.Sprintf("Backup in progress: %s", status), percent)
			}

			if status == "COMPLETED" {
				if progressReporter != nil {
					progressReporter("Backup completed successfully", 100.0)
				}
				l.Infof("✅ Backup job completed successfully")
				return &types.BackupSnapshotInfo{
					Region:    region,
					Namespace: namespace,
					EFSID:     efsID,
					ARN:       arn,
					JobID:     jobID,
					Status:    "COMPLETED",
				}, nil
			} else if status == "FAILED" || status == "ABORTED" || status == "EXPIRED" {
				return nil, fmt.Errorf("backup job failed with status: %s", status)
			}
		}
	}

	return nil, fmt.Errorf("backup job monitoring timed out")
}
