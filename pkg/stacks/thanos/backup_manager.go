package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// BackupStatus prints EFS and RDS backup status
func (t *ThanosStack) BackupStatus(ctx context.Context) error {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	fmt.Printf("Region: %s, Namespace: %s, Account ID: %s\n", region, namespace, accountID)
	fmt.Println("")

	if efsID, err := utils.DetectEFSId(ctx, namespace); err == nil && efsID != "" {
		arn := utils.BuildEFSArn(region, accountID, efsID)
		fmt.Printf("📁 EFS Backup Status\n")
		fmt.Printf("   FileSystemId: %s\n", efsID)
		fmt.Printf("   ARN: %s\n", arn)

		// Check if EFS is protected
		cnt, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources", "--region", region, "--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", arn), "--output", "text")
		if err != nil {
			fmt.Printf("   Protected: ❌ Error checking protection status: %v\n", err)
		} else {
			protected := strings.TrimSpace(cnt)
			if protected == "1" {
				fmt.Printf("   Protected: ✅ true\n")
			} else {
				fmt.Printf("   Protected: ❌ false\n")
			}
		}

		// Check latest recovery point
		rp, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", "max_by(RecoveryPoints,&CreationDate).CreationDate", "--output", "text")
		if err != nil {
			fmt.Printf("   Latest recovery point: ❌ Error checking recovery points: %v\n", err)
		} else {
			rpTrimmed := strings.TrimSpace(rp)
			if rpTrimmed == "None" || rpTrimmed == "" {
				fmt.Printf("   Latest recovery point: ⚠️  None (no backups found)\n")
			} else {
				fmt.Printf("   Latest recovery point: ✅ %s\n", rpTrimmed)

				// Calculate expected expiry date if creation date is available
				if rpTrimmed != "None" && rpTrimmed != "" {
					// Parse the creation date
					creationTime, err := time.Parse("2006-01-02T15:04:05.000000-07:00", rpTrimmed)
					if err != nil {
						// Try alternative format
						creationTime, err = time.Parse("2006-01-02T15:04:05.000000+09:00", rpTrimmed)
					}
					if err == nil {
						expiryTime := creationTime.AddDate(0, 0, 35) // 35 days from creation
						fmt.Printf("   Expected expiry date: 📅 %s (35 days from creation)\n", expiryTime.Format("2006-01-02T15:04:05-07:00"))
					}
				}
			}
		}

		// Check backup vaults (simplified query)
		vaults, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", "RecoveryPoints[].BackupVaultName", "--output", "text")
		if err != nil {
			fmt.Printf("   Vaults: ❌ Error checking vaults: %v\n", err)
		} else {
			vaultsTrimmed := strings.TrimSpace(vaults)
			if vaultsTrimmed == "None" || vaultsTrimmed == "" {
				fmt.Printf("   Vaults: ⚠️  None (no backups found)\n")
			} else {
				fmt.Printf("   Vaults: ✅ %s\n", vaultsTrimmed)
			}
		}
	} else {
		fmt.Printf("📁 EFS Backup Status\n")
		fmt.Printf("   ❌ Not detected in cluster PVs: %v\n", err)
	}

	fmt.Println("")

	rdsID := utils.RDSIdentifierFromNamespace(namespace)
	fmt.Printf("🗄️  RDS Backup Status\n")
	fmt.Printf("   Instance ID: %s\n", rdsID)

	// Check RDS instance status
	q := "DBInstances[0].{Id:DBInstanceIdentifier,Retention:BackupRetentionPeriod,Window:PreferredBackupWindow,Status:DBInstanceStatus}"
	info, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances", "--region", region, "--db-instance-identifier", rdsID, "--query", q, "--output", "table")
	if err != nil {
		fmt.Printf("   Instance Status: ❌ Error checking instance status: %v\n", err)
		fmt.Printf("   Instance may not exist or you may not have permission to access it\n")
	} else {
		infoTrimmed := strings.TrimSpace(info)
		if infoTrimmed == "" {
			fmt.Printf("   Instance Status: ⚠️  No instance information found\n")
		} else {
			fmt.Printf("   Instance Status:\n%s\n", infoTrimmed)
		}
	}

	// Check RDS snapshots
	lastSnap, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-snapshots", "--region", region, "--db-instance-identifier", rdsID, "--snapshot-type", "automated", "--query", "max_by(DBSnapshots,&SnapshotCreateTime).{Id:DBSnapshotIdentifier,Time:SnapshotCreateTime}", "--output", "table")
	if err != nil {
		fmt.Printf("   Latest Snapshot: ❌ Error checking snapshots: %v\n", err)
	} else {
		lastSnapTrimmed := strings.TrimSpace(lastSnap)
		if lastSnapTrimmed == "" {
			fmt.Printf("   Latest Snapshot: ⚠️  No automated snapshots found\n")
		} else {
			fmt.Printf("   Latest Snapshot:\n%s\n", lastSnapTrimmed)
		}
	}

	return nil
}

// BackupSnapshot triggers on-demand EFS backup and RDS snapshot
func (t *ThanosStack) BackupSnapshot(ctx context.Context, note string) error {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return err
	}

	if efsID, err := utils.DetectEFSId(ctx, namespace); err == nil && efsID != "" {
		arn := utils.BuildEFSArn(region, accountID, efsID)

		// Get the correct backup vault name and IAM role
		backupVaultName := fmt.Sprintf("%s-backup-vault", namespace)
		iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-service-role", accountID, namespace)

		// Start backup job with proper parameters
		backupJobOutput, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
			"--region", region,
			"--backup-vault-name", backupVaultName,
			"--resource-arn", arn,
			"--iam-role-arn", iamRoleArn)

		if err != nil {
			fmt.Printf("📁 EFS: ❌ Failed to start backup job: %v\n", err)
			fmt.Printf("   Backup vault: %s\n", backupVaultName)
			fmt.Printf("   IAM role: %s\n", iamRoleArn)
		} else {
			jobId := strings.TrimSpace(backupJobOutput)
			fmt.Printf("📁 EFS: ✅ On-demand backup started successfully\n")
			fmt.Printf("   Job ID: %s\n", jobId)
			fmt.Printf("   Backup vault: %s\n", backupVaultName)
		}
	} else {
		fmt.Printf("📁 EFS: ❌ Not detected in cluster PVs: %v\n", err)
	}

	rdsID := utils.RDSIdentifierFromNamespace(namespace)

	// Check if RDS instance exists before attempting to create snapshot
	_, err = utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances", "--region", region, "--db-instance-identifier", rdsID, "--query", "DBInstances[0].DBInstanceIdentifier", "--output", "text")
	if err != nil {
		fmt.Printf("🗄️  RDS: ⚠️  Instance not found, skipping RDS snapshot\n")
		fmt.Printf("   Instance ID: %s\n", rdsID)
		fmt.Printf("   Reason: RDS instance does not exist in this deployment\n")
	} else {
		// RDS instance exists, proceed with snapshot creation
		snapID := fmt.Sprintf("%s-manual-%d", rdsID, time.Now().Unix())
		if strings.TrimSpace(note) != "" {
			snapID = fmt.Sprintf("%s-%s", snapID, strings.ReplaceAll(note, " ", "-"))
		}

		// Create RDS snapshot
		if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "create-db-snapshot", "--region", region, "--db-instance-identifier", rdsID, "--db-snapshot-identifier", snapID); err != nil {
			fmt.Printf("🗄️  RDS: ❌ Failed to create snapshot: %v\n", err)
			fmt.Printf("   Instance ID: %s\n", rdsID)
			fmt.Printf("   Snapshot ID: %s\n", snapID)
		} else {
			fmt.Printf("🗄️  RDS: ✅ Manual snapshot started successfully\n")
			fmt.Printf("   Instance ID: %s\n", rdsID)
			fmt.Printf("   Snapshot ID: %s\n", snapID)
		}
	}
	return nil
}

// BackupList lists recent EFS recovery points and RDS snapshots
func (t *ThanosStack) BackupList(ctx context.Context, limit string) error {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return err
	}
	limit = strings.TrimSpace(limit)

	fmt.Printf("Region: %s, Namespace: %s, Account ID: %s\n", region, namespace, accountID)
	fmt.Println("")

	if efsID, err := utils.DetectEFSId(ctx, namespace); err == nil && efsID != "" {
		arn := utils.BuildEFSArn(region, accountID, efsID)
		fmt.Printf("📁 EFS Recovery Points (FileSystemId: %s)\n", efsID)

		// First get the data in JSON format to process expiry dates
		jsonQuery := "reverse(sort_by(RecoveryPoints,&CreationDate))[:10]"
		if limit != "" {
			jsonQuery = fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s]", limit)
		}
		jsonOut, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", jsonQuery, "--output", "json")
		if err != nil {
			fmt.Printf("   ❌ Error retrieving recovery points: %v\n", err)
		} else {
			// Process JSON to calculate expiry dates
			jsonOutTrimmed := strings.TrimSpace(jsonOut)
			if jsonOutTrimmed == "" || jsonOutTrimmed == "[]" {
				fmt.Printf("   ⚠️  No recovery points found\n")
			} else {
				// For now, use the table format but with calculated expiry dates
				query := "reverse(sort_by(RecoveryPoints,&CreationDate))[:10].{Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}"
				if limit != "" {
					query = fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s].{Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}", limit)
				}
				out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", query, "--output", "table")
				if err != nil {
					fmt.Printf("   ❌ Error retrieving recovery points: %v\n", err)
				} else {
					outTrimmed := strings.TrimSpace(out)
					if outTrimmed == "" {
						fmt.Printf("   ⚠️  No recovery points found\n")
					} else {
						// Add indentation to the table output and replace the header
						lines := strings.Split(outTrimmed, "\n")
						for i, line := range lines {
							if i == 0 {
								// Replace the AWS CLI generated header with a custom one
								fmt.Printf("   -------------------------------------------------\n")
								fmt.Printf("   |                EFS Recovery Points            |\n")
								fmt.Printf("   +---------+-------------------------------------+\n")
							} else if i == 1 {
								// Skip the original header line
								continue
							} else {
								// Replace "None" expiry with calculated date
								if strings.Contains(line, "|  Expiry |  None") {
									// Find the creation date from the previous line
									if i > 2 && i < len(lines) {
										prevLine := lines[i-1]
										if strings.Contains(prevLine, "|  Created|") {
											parts := strings.Split(prevLine, "|")
											if len(parts) >= 3 {
												createdStr := strings.TrimSpace(parts[2])
												if createdStr != "" && createdStr != "None" {
													// Parse the creation date
													creationTime, err := time.Parse("2006-01-02T15:04:05.000000-07:00", createdStr)
													if err != nil {
														creationTime, err = time.Parse("2006-01-02T15:04:05.000000+09:00", createdStr)
													}
													if err == nil {
														expiryTime := creationTime.AddDate(0, 0, 35)
														expiryStr := expiryTime.Format("2006-01-02T15:04:05-07:00")
														// Replace the "None" with calculated expiry date
														modifiedLine := strings.Replace(line, "|  Expiry |  None", fmt.Sprintf("|  Expiry |  %s (calc)", expiryStr), 1)
														fmt.Printf("   %s\n", modifiedLine)
														continue
													}
												}
											}
										}
									}
								}
								fmt.Printf("   %s\n", line) // Regular data rows
							}
						}
					}
				}
			}
		}
	} else {
		fmt.Printf("📁 EFS Recovery Points\n")
		fmt.Printf("   ❌ Not detected in cluster PVs: %v\n", err)
	}

	fmt.Println("")

	rdsID := utils.RDSIdentifierFromNamespace(namespace)
	fmt.Printf("🗄️  RDS Snapshots (Instance ID: %s)\n", rdsID)

	q := "reverse(sort_by(DBSnapshots,&SnapshotCreateTime))[:10].{Id:DBSnapshotIdentifier,Time:SnapshotCreateTime,Type:SnapshotType}"
	if limit != "" {
		q = fmt.Sprintf("reverse(sort_by(DBSnapshots,&SnapshotCreateTime))[:%s].{Id:DBSnapshotIdentifier,Time:SnapshotCreateTime,Type:SnapshotType}", limit)
	}
	out, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-snapshots", "--region", region, "--db-instance-identifier", rdsID, "--query", q, "--output", "table")
	if err != nil {
		fmt.Printf("   ❌ Error retrieving snapshots: %v\n", err)
	} else {
		outTrimmed := strings.TrimSpace(out)
		if outTrimmed == "" {
			fmt.Printf("   ⚠️  No snapshots found\n")
		} else {
			// Add indentation to the table output and replace the header
			lines := strings.Split(outTrimmed, "\n")
			for i, line := range lines {
				if i == 0 {
					// Replace the AWS CLI generated header with a custom one
					fmt.Printf("   -------------------------------------------------\n")
					fmt.Printf("   |                RDS Snapshots                    |\n")
					fmt.Printf("   +---------+-------------------------------------+\n")
				} else if i == 1 {
					// Skip the original header line
					continue
				} else {
					fmt.Printf("   %s\n", line) // Data rows
				}
			}
		}
	}

	return nil
}

// BackupRestore provides a fully interactive restore experience for EFS and RDS
func (t *ThanosStack) BackupRestore(ctx context.Context, point *string, newFS *bool, snap *string, newRDSId *string) error {
	// Validate prerequisites
	if err := t.validateRestorePrerequisites(ctx); err != nil {
		return fmt.Errorf("prerequisites validation failed: %w", err)
	}

	// Validate AWS credentials and permissions
	if err := t.validateAWSCredentials(ctx); err != nil {
		return fmt.Errorf("AWS credentials validation failed: %w", err)
	}

	// Always use interactive mode
	return t.interactiveRestore(ctx)
}

// validateRestorePrerequisites checks if required tools and dependencies are available
func (t *ThanosStack) validateRestorePrerequisites(ctx context.Context) error {
	fmt.Println("[VALIDATION] Checking prerequisites...")

	// Check AWS CLI
	if _, err := utils.ExecuteCommand(ctx, "aws", "--version"); err != nil {
		return fmt.Errorf("AWS CLI is not installed or not accessible: %w", err)
	}

	// Check kubectl
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "version", "--client"); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}

	// Check if we can access the cluster
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "cluster-info"); err != nil {
		return fmt.Errorf("cannot access Kubernetes cluster: %w", err)
	}

	fmt.Println("[VALIDATION] ✅ Prerequisites check passed")
	return nil
}

// validateAWSCredentials validates AWS credentials and permissions
func (t *ThanosStack) validateAWSCredentials(ctx context.Context) error {
	fmt.Println("[VALIDATION] Checking AWS credentials and permissions...")

	// Check AWS credentials
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// Check AWS Backup permissions
	if _, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults", "--region", t.deployConfig.AWS.Region, "--max-items", "1"); err != nil {
		return fmt.Errorf("insufficient AWS Backup permissions: %w", err)
	}

	// Check RDS permissions
	if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances", "--region", t.deployConfig.AWS.Region, "--max-items", "1"); err != nil {
		return fmt.Errorf("insufficient RDS permissions: %w", err)
	}

	fmt.Printf("[VALIDATION] ✅ AWS credentials valid (Account: %s)\n", accountID)
	return nil
}

// restoreEFS handles EFS restore from recovery point
func (t *ThanosStack) restoreEFS(ctx context.Context, recoveryPointArn string) (string, error) {
	fmt.Printf("[EFS] Starting restore from recovery point: %s\n", recoveryPointArn)

	// Validate recovery point ARN format
	if !strings.Contains(recoveryPointArn, "arn:aws:backup:") {
		return "", fmt.Errorf("invalid recovery point ARN format: %s", recoveryPointArn)
	}

	// Check if recovery point exists and is accessible
	// First, get the recovery point details to extract vault name
	fmt.Printf("[DEBUG] Checking recovery point: %s\n", recoveryPointArn)
	fmt.Printf("[DEBUG] Region: %s\n", t.deployConfig.AWS.Region)

	// For this ARN format, we need to find the vault name by listing all vaults
	// and checking which one contains this recovery point
	fmt.Printf("[DEBUG] ARN format doesn't contain vault name, searching for vault...\n")

	// List all backup vaults
	vaultsOutput, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults",
		"--region", t.deployConfig.AWS.Region,
		"--query", "BackupVaultList[].BackupVaultName",
		"--output", "text")

	if err != nil {
		return "", fmt.Errorf("failed to list backup vaults: %w", err)
	}

	vaultNames := strings.Fields(strings.TrimSpace(vaultsOutput))
	fmt.Printf("[DEBUG] Found vaults: %v\n", vaultNames)

	// Try to find the vault that contains this recovery point
	var recoveryPointDetails string
	var foundVault string

	for _, vaultName := range vaultNames {
		fmt.Printf("[DEBUG] Checking vault: %s\n", vaultName)

		// Try to describe the recovery point in this vault
		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", t.deployConfig.AWS.Region,
			"--backup-vault-name", vaultName,
			"--recovery-point-arn", recoveryPointArn,
			"--query", "RecoveryPoint.BackupVaultName",
			"--output", "text")

		if err == nil {
			recoveryPointDetails = vaultName // Use the vault name directly instead of querying it
			foundVault = vaultName
			fmt.Printf("[DEBUG] Found recovery point in vault: %s\n", vaultName)
			break
		}
	}

	if foundVault == "" {
		// If we couldn't find it in any vault, try without vault name
		fmt.Printf("[DEBUG] Recovery point not found in any vault, trying without backup-vault-name...\n")
		recoveryPointDetails, err = utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", t.deployConfig.AWS.Region,
			"--recovery-point-arn", recoveryPointArn,
			"--query", "RecoveryPoint.BackupVaultName",
			"--output", "text")
	} else {
		err = nil // We found it, so no error
	}

	if err != nil {
		// Try without query first to get more detailed error
		fmt.Printf("[DEBUG] Failed with query, trying without query...\n")
		fullDetails, fullErr := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", t.deployConfig.AWS.Region,
			"--recovery-point-arn", recoveryPointArn)
		if fullErr != nil {
			return "", fmt.Errorf("recovery point not found or not accessible: %w (full error: %v)", err, fullErr)
		}
		fmt.Printf("[DEBUG] Full recovery point details: %s\n", fullDetails)
		return "", fmt.Errorf("recovery point not found or not accessible: %w", err)
	}

	vaultName := strings.TrimSpace(recoveryPointDetails)
	fmt.Printf("[DEBUG] Extracted vault name: %s\n", vaultName)
	if vaultName == "" {
		return "", fmt.Errorf("could not extract vault name from recovery point")
	}

	// Get appropriate IAM role for restore
	iamRoleArn, err := t.getRestoreIAMRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get IAM role for restore: %w", err)
	}

	// Start restore job
	// For EFS restore, we need to specify that we want a new filesystem
	meta := `{"newfilesystem": "true"}`
	fmt.Printf("[DEBUG] Starting restore job with:\n")
	fmt.Printf("[DEBUG]   Region: %s\n", t.deployConfig.AWS.Region)
	fmt.Printf("[DEBUG]   Recovery Point ARN: %s\n", recoveryPointArn)
	fmt.Printf("[DEBUG]   IAM Role ARN: %s\n", iamRoleArn)
	fmt.Printf("[DEBUG]   Metadata: %s\n", meta)

	jobId, err := utils.ExecuteCommand(ctx,
		"aws", "backup", "start-restore-job",
		"--region", t.deployConfig.AWS.Region,
		"--recovery-point-arn", recoveryPointArn,
		"--iam-role-arn", iamRoleArn,
		"--metadata", meta,
		"--query", "RestoreJobId",
		"--output", "text",
	)
	if err != nil {
		// Try to get more detailed error information
		fmt.Printf("[DEBUG] Restore job failed, trying without query to get full error...\n")
		fullOutput, fullErr := utils.ExecuteCommand(ctx, "aws", "backup", "start-restore-job",
			"--region", t.deployConfig.AWS.Region,
			"--recovery-point-arn", recoveryPointArn,
			"--iam-role-arn", iamRoleArn,
			"--metadata", meta)
		if fullErr != nil {
			return "", fmt.Errorf("failed to start restore job: %w (full error: %v, output: %s)", err, fullErr, fullOutput)
		}
		return "", fmt.Errorf("failed to start restore job: %w", err)
	}

	jobId = strings.TrimSpace(jobId)
	if jobId == "" {
		return "", fmt.Errorf("restore job ID is empty")
	}

	fmt.Printf("[EFS] Restore job started, JobId: %s\n", jobId)

	// Monitor restore job progress
	return t.monitorEFSRestoreJob(ctx, jobId)
}

// restoreRDS handles RDS restore from snapshot
func (t *ThanosStack) restoreRDS(ctx context.Context, snapshotId, targetInstanceId string) error {
	fmt.Printf("[RDS] Starting restore from snapshot: %s to instance: %s\n", snapshotId, targetInstanceId)

	// Validate snapshot exists
	if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-snapshots",
		"--region", t.deployConfig.AWS.Region,
		"--db-snapshot-identifier", snapshotId); err != nil {
		return fmt.Errorf("snapshot not found or not accessible: %w", err)
	}

	// Check if target instance already exists
	if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", t.deployConfig.AWS.Region,
		"--db-instance-identifier", targetInstanceId); err == nil {
		return fmt.Errorf("target RDS instance already exists: %s", targetInstanceId)
	}

	// Start RDS restore
	if _, err := utils.ExecuteCommand(ctx, "aws", "rds", "restore-db-instance-from-db-snapshot",
		"--region", t.deployConfig.AWS.Region,
		"--db-snapshot-identifier", snapshotId,
		"--db-instance-identifier", targetInstanceId); err != nil {
		return fmt.Errorf("failed to start RDS restore: %w", err)
	}

	fmt.Printf("[RDS] Restore started for instance: %s\n", targetInstanceId)

	// Monitor RDS restore progress
	return t.monitorRDSRestore(ctx, targetInstanceId)
}

// monitorEFSRestoreJob monitors EFS restore job progress
func (t *ThanosStack) monitorEFSRestoreJob(ctx context.Context, jobId string) (string, error) {
	fmt.Println("[EFS] Monitoring restore job progress...")

	const maxAttempts = 120 // 60 minutes with 30-second intervals
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("restore monitoring cancelled: %w", ctx.Err())
		default:
		}

		status, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
			"--region", t.deployConfig.AWS.Region,
			"--restore-job-id", jobId,
			"--query", "Status",
			"--output", "text")
		if err != nil {
			return "", fmt.Errorf("failed to get restore job status: %w", err)
		}

		status = strings.TrimSpace(status)
		fmt.Printf("[EFS] Job status: %s (attempt %d/%d)\n", status, i+1, maxAttempts)

		switch status {
		case "COMPLETED":
			return t.handleEFSRestoreCompletion(ctx, jobId)
		case "ABORTED", "FAILED":
			// Get detailed error information for failed jobs
			errorDetails, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
				"--region", t.deployConfig.AWS.Region,
				"--restore-job-id", jobId,
				"--query", "StatusMessage",
				"--output", "text")
			if err == nil {
				errorDetails = strings.TrimSpace(errorDetails)
				return "", fmt.Errorf("restore job failed with status: %s, error: %s", status, errorDetails)
			}
			return "", fmt.Errorf("restore job failed with status: %s", status)
		case "RUNNING", "PENDING":
			// Continue monitoring
		default:
			fmt.Printf("[EFS] Unknown status: %s, continuing to monitor...\n", status)
		}

		time.Sleep(30 * time.Second)
	}

	return "", fmt.Errorf("restore job monitoring timed out after %d minutes", maxAttempts/2)
}

// handleEFSRestoreCompletion processes successful EFS restore completion
func (t *ThanosStack) handleEFSRestoreCompletion(ctx context.Context, jobId string) (string, error) {
	createdArn, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
		"--region", t.deployConfig.AWS.Region,
		"--restore-job-id", jobId,
		"--query", "CreatedResourceArn",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("failed to get created resource ARN: %w", err)
	}

	createdArn = strings.TrimSpace(createdArn)
	fmt.Printf("[EFS] Restore completed. CreatedResourceArn: %s\n", createdArn)

	var newFsId string
	if strings.Contains(createdArn, ":file-system/") {
		parts := strings.Split(createdArn, "/")
		if len(parts) > 0 {
			newFsId = parts[len(parts)-1]
			fmt.Printf("[EFS] New FileSystemId: %s\n", newFsId)
			fmt.Printf("[TIP] You can now attach with: trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node\n", newFsId)
		}
	}

	return newFsId, nil
}

// monitorRDSRestore monitors RDS restore progress
func (t *ThanosStack) monitorRDSRestore(ctx context.Context, instanceId string) error {
	fmt.Printf("[RDS] Monitoring restore progress for instance: %s\n", instanceId)

	const maxAttempts = 120 // 60 minutes with 30-second intervals
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("RDS restore monitoring cancelled: %w", ctx.Err())
		default:
		}

		status, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
			"--region", t.deployConfig.AWS.Region,
			"--db-instance-identifier", instanceId,
			"--query", "DBInstances[0].DBInstanceStatus",
			"--output", "text")
		if err != nil {
			return fmt.Errorf("failed to get RDS instance status: %w", err)
		}

		status = strings.TrimSpace(status)
		fmt.Printf("[RDS] Instance status: %s (attempt %d/%d)\n", status, i+1, maxAttempts)

		switch status {
		case "available":
			fmt.Printf("[RDS] ✅ Restore completed successfully for instance: %s\n", instanceId)
			return nil
		case "failed", "deleted":
			return fmt.Errorf("RDS instance restore failed with status: %s", status)
		case "creating", "restoring":
			// Continue monitoring
		default:
			fmt.Printf("[RDS] Unknown status: %s, continuing to monitor...\n", status)
		}

		time.Sleep(30 * time.Second)
	}

	return fmt.Errorf("RDS restore monitoring timed out after %d minutes", maxAttempts/2)
}

// getRestoreIAMRole gets appropriate IAM role for restore operations
func (t *ThanosStack) getRestoreIAMRole(ctx context.Context) (string, error) {
	// Try to find a suitable IAM role for restore operations
	// First, try to get the default backup service role
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// Try to find a suitable IAM role for restore operations
	// List all IAM roles and look for backup-related roles
	rolesOutput, err := utils.ExecuteCommand(ctx, "aws", "iam", "list-roles",
		"--query", "Roles[?contains(RoleName, 'backup') || contains(RoleName, 'Backup') || contains(RoleName, 'restore') || contains(RoleName, 'Restore')].RoleName",
		"--output", "text")

	if err == nil {
		roleNames := strings.Fields(strings.TrimSpace(rolesOutput))
		fmt.Printf("[DEBUG] Found backup-related roles: %v\n", roleNames)

		// Try roles in order of preference
		preferredRoles := []string{
			"theo08123-apaic-backup-service-role", // Custom backup role
			"AWSServiceRoleForBackup",             // AWS managed backup role
			"AWSBackupDefaultServiceRole",         // AWS backup default role
		}

		for _, preferredRole := range preferredRoles {
			for _, roleName := range roleNames {
				if roleName == preferredRole {
					roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
					fmt.Printf("[DEBUG] Trying preferred role: %s\n", roleArn)

					// Check if this role has the necessary permissions
					if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
						return roleArn, nil
					}
				}
			}
		}

		// If no preferred role found, try any backup-related role
		for _, roleName := range roleNames {
			roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)
			fmt.Printf("[DEBUG] Trying role: %s\n", roleArn)

			// Check if this role has the necessary permissions
			if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
				return roleArn, nil
			}
		}
	}

	// Try the default backup service role pattern
	defaultRole := fmt.Sprintf("arn:aws:iam::%s:role/service-role/AWSBackupServiceRolePolicyForRestores", accountID)

	// Check if the role exists
	if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", "AWSBackupServiceRolePolicyForRestores"); err == nil {
		return defaultRole, nil
	}

	// Try alternative role patterns
	alternativeRoles := []string{
		fmt.Sprintf("arn:aws:iam::%s:role/AWSBackupServiceRolePolicyForRestores", accountID),
		fmt.Sprintf("arn:aws:iam::%s:role/backup-restore-role", accountID),
		fmt.Sprintf("arn:aws:iam::%s:role/aws-backup-restore-role", accountID),
	}

	for _, role := range alternativeRoles {
		roleName := strings.Split(role, "/")[len(strings.Split(role, "/"))-1]
		if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
			return role, nil
		}
	}

	// If no suitable role found, try to use the AWS Backup default service role
	// This is the role that AWS Backup uses by default
	awsBackupDefaultRole := fmt.Sprintf("arn:aws:iam::%s:role/service-role/AWSBackupDefaultServiceRole", accountID)
	fmt.Printf("[DEBUG] Trying AWS Backup default service role: %s\n", awsBackupDefaultRole)

	// If no suitable role found, return the default and let AWS handle the error
	fmt.Println("[WARNING] No suitable IAM role found, using default role (may fail if not configured)")
	return awsBackupDefaultRole, nil
}

// BackupAttach switches workloads to use the new EFS and verifies readiness
func (t *ThanosStack) BackupAttach(ctx context.Context, efsId *string, pvcs *string, stss *string) error {
	namespace := t.deployConfig.K8s.Namespace

	fmt.Println("[POST-RESTORE] Verifying restored data and switching workloads...")

	// Validate prerequisites
	if err := t.validateAttachPrerequisites(ctx); err != nil {
		return fmt.Errorf("attach prerequisites validation failed: %w", err)
	}

	// Verify EFS ID format if provided
	newEfs := ""
	if efsId != nil {
		newEfs = strings.TrimSpace(*efsId)
		if newEfs != "" && !strings.HasPrefix(newEfs, "fs-") {
			return fmt.Errorf("invalid EFS ID format: %s (should start with 'fs-')", newEfs)
		}
	}

	// Verify data via temporary pod
	if err := t.verifyEFSData(ctx, namespace); err != nil {
		fmt.Printf("[WARNING] EFS data verification failed: %v\n", err)
	}

	// Update PV volumeHandle if EFS ID provided
	if newEfs != "" {
		if err := t.updatePVVolumeHandles(ctx, namespace, newEfs, pvcs); err != nil {
			return fmt.Errorf("failed to update PV volume handles: %w", err)
		}
	} else {
		fmt.Println("[ATTACH] Skipped PV update (no --efs-id provided).")
	}

	// Restart workloads if specified
	if stss != nil && strings.TrimSpace(*stss) != "" {
		if err := t.restartStatefulSets(ctx, namespace, strings.TrimSpace(*stss)); err != nil {
			return fmt.Errorf("failed to restart StatefulSets: %w", err)
		}
	}

	// Health check
	if err := t.performHealthCheck(ctx, namespace); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	fmt.Println("[POST-RESTORE] Completed basic post-restore steps.")

	// Summary
	t.printAttachSummary(newEfs, pvcs, stss)
	return nil
}

// validateAttachPrerequisites checks if required tools and cluster access are available
func (t *ThanosStack) validateAttachPrerequisites(ctx context.Context) error {
	fmt.Println("[VALIDATION] Checking attach prerequisites...")

	// Check kubectl
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "version", "--client"); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}

	// Check if we can access the cluster
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "cluster-info"); err != nil {
		return fmt.Errorf("cannot access Kubernetes cluster: %w", err)
	}

	// Check if namespace exists
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", t.deployConfig.K8s.Namespace); err != nil {
		return fmt.Errorf("namespace %s does not exist or is not accessible: %w", t.deployConfig.K8s.Namespace, err)
	}

	fmt.Println("[VALIDATION] ✅ Attach prerequisites check passed")
	return nil
}

// verifyEFSData creates a temporary pod to verify EFS data accessibility
func (t *ThanosStack) verifyEFSData(ctx context.Context, namespace string) error {
	fmt.Println("[VERIFY] Creating temporary pod to verify EFS data...")

	// Clean up any existing verify pod
	_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Create verify pod
	podYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: verify-efs
  namespace: %s
spec:
  containers:
  - name: verify
    image: alpine:latest
    command: ["/bin/sh", "-c"]
    args: ["ls -la /db || true; echo 'EFS verification completed'; sleep 5"]
    volumeMounts:
    - name: efs-volume
      mountPath: /db
  volumes:
  - name: efs-volume
    persistentVolumeClaim:
      claimName: %s-op-geth
  restartPolicy: Never`, namespace, namespace)

	// Create pod using kubectl apply
	tempFile := fmt.Sprintf("/tmp/verify-efs-%s.yaml", namespace)
	if err := os.WriteFile(tempFile, []byte(podYaml), 0644); err != nil {
		return fmt.Errorf("failed to create temporary YAML file: %w", err)
	}
	defer os.Remove(tempFile)

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to create verify pod: %w", err)
	}

	// Wait for pod to complete
	fmt.Println("[VERIFY] Waiting for verification pod to complete...")
	for i := 0; i < 30; i++ {
		status, _ := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", "verify-efs", "-o", "jsonpath={.status.phase}")
		if strings.TrimSpace(status) == "Succeeded" {
			fmt.Println("[VERIFY] ✅ EFS data verification completed successfully")
			return nil
		}
		if strings.TrimSpace(status) == "Failed" {
			return fmt.Errorf("verification pod failed")
		}
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("verification pod timed out")
}

// updatePVVolumeHandles updates PV volume handles to point to new EFS
func (t *ThanosStack) updatePVVolumeHandles(ctx context.Context, namespace, newEfs string, pvcs *string) error {
	fmt.Printf("[ATTACH] Updating PV volume handles to EFS: %s\n", newEfs)

	var targetPVs []string

	// Get target PVCs
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		for _, pvc := range strings.Split(strings.TrimSpace(*pvcs), ",") {
			pvc = strings.TrimSpace(pvc)
			if pvc == "" {
				continue
			}

			// Validate PVC exists
			if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvc); err != nil {
				fmt.Printf("[WARNING] PVC %s not found, skipping\n", pvc)
				continue
			}

			pvName, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvc, "-o", "jsonpath={.spec.volumeName}")
			if err != nil {
				fmt.Printf("[WARNING] Failed to get PV name for PVC %s: %v\n", pvc, err)
				continue
			}

			pvName = strings.TrimSpace(pvName)
			if pvName != "" {
				targetPVs = append(targetPVs, pvName)
			}
		}
	} else {
		// If no PVCs specified, get all PVCs in namespace
		pvcsList, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if err != nil {
			return fmt.Errorf("failed to list PVCs: %w", err)
		}

		for _, pvc := range strings.Fields(pvcsList) {
			pvName, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvc, "-o", "jsonpath={.spec.volumeName}")
			if err == nil {
				pvName = strings.TrimSpace(pvName)
				if pvName != "" {
					targetPVs = append(targetPVs, pvName)
				}
			}
		}
	}

	if len(targetPVs) == 0 {
		return fmt.Errorf("no target PVs found to update")
	}

	// Update each PV
	successCount := 0
	for _, pv := range targetPVs {
		pv = strings.TrimSpace(pv)
		if pv == "" {
			continue
		}

		patch := fmt.Sprintf(`{"spec":{"csi":{"volumeHandle":"%s"}}}`, newEfs)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", pv, "--type=merge", "-p", patch); err != nil {
			fmt.Printf("[ERROR] Failed to patch PV %s: %v\n", pv, err)
		} else {
			fmt.Printf("[ATTACH] ✅ PV %s volumeHandle updated -> %s\n", pv, newEfs)
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to update any PV volume handles")
	}

	fmt.Printf("[ATTACH] ✅ Successfully updated %d/%d PV volume handles\n", successCount, len(targetPVs))
	return nil
}

// restartStatefulSets restarts specified StatefulSets
func (t *ThanosStack) restartStatefulSets(ctx context.Context, namespace, stss string) error {
	fmt.Printf("[RESTART] Restarting StatefulSets: %s\n", stss)

	successCount := 0
	for _, sts := range strings.Split(stss, ",") {
		sts = strings.TrimSpace(sts)
		if sts == "" {
			continue
		}

		// Validate StatefulSet exists
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "statefulset", sts); err != nil {
			fmt.Printf("[WARNING] StatefulSet %s not found, skipping\n", sts)
			continue
		}

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "rollout", "restart", fmt.Sprintf("statefulset/%s", sts)); err != nil {
			fmt.Printf("[ERROR] Failed to restart StatefulSet %s: %v\n", sts, err)
		} else {
			fmt.Printf("[RESTART] ✅ StatefulSet %s restarted successfully\n", sts)
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to restart any StatefulSets")
	}

	fmt.Printf("[RESTART] ✅ Successfully restarted %d StatefulSets\n", successCount)
	return nil
}

// performHealthCheck performs health check on the cluster
func (t *ThanosStack) performHealthCheck(ctx context.Context, namespace string) error {
	fmt.Println("[HEALTH] Performing health check...")

	// Check multiple labels for better coverage
	labels := []string{
		"app.kubernetes.io/name=thanos-stack-op-geth",
		"app.kubernetes.io/name=thanos-stack-op-node",
		"app.kubernetes.io/name=thanos-stack-op-batcher",
	}

	const maxAttempts = 30 // 5 minutes with 10-second intervals
	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("health check cancelled: %w", ctx.Err())
		default:
		}

		for _, label := range labels {
			out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pods", "-l", label, "-o", "jsonpath={.items[*].status.containerStatuses[*].ready}")
			if err == nil && strings.Contains(out, "true") {
				fmt.Printf("[HEALTH] ✅ Ready containers detected by label: %s\n", label)
				return nil
			}
		}

		fmt.Printf("[HEALTH] Waiting for ready containers... (attempt %d/%d)\n", i+1, maxAttempts)
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("health check timed out after %d minutes", maxAttempts/6)
}

// printAttachSummary prints a summary of the attach operation
func (t *ThanosStack) printAttachSummary(newEfs string, pvcs *string, stss *string) {
	fmt.Println("\n[SUMMARY] Attach operation completed:")
	if newEfs != "" {
		fmt.Printf("  EFS FileSystemId: %s\n", newEfs)
	}
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		fmt.Printf("  PVCs: %s\n", strings.TrimSpace(*pvcs))
	}
	if stss != nil && strings.TrimSpace(*stss) != "" {
		fmt.Printf("  StatefulSets: %s\n", strings.TrimSpace(*stss))
	}
	fmt.Println("  Health Check: ✅ Passed")
}

// BackupConfigure applies module configuration via Terraform
func (t *ThanosStack) BackupConfigure(ctx context.Context, on *bool, off *bool, daily *string, keep *string, vault *string, reset *bool, keepRDS *string, window *string) error {
	tfRoot := fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath)

	{
		varArgs := []string{"-auto-approve"}
		if reset != nil && *reset {
			varArgs = append(varArgs, "-var=backup_schedule_cron=cron(0 3 * * ? *)", "-var=backup_delete_after_days=35")
		} else {
			if daily != nil && strings.TrimSpace(*daily) != "" {
				// convert HH:MM -> cron(0 H * * ? *) using provided hour
				hhmm := strings.TrimSpace(*daily)
				parts := strings.Split(hhmm, ":")
				cron := fmt.Sprintf("cron(0 %s * * ? *)", parts[0])
				varArgs = append(varArgs, fmt.Sprintf("-var=backup_schedule_cron=%s", cron))
			}
			if keep != nil && strings.TrimSpace(*keep) != "" {
				varArgs = append(varArgs, fmt.Sprintf("-var=backup_delete_after_days=%s", strings.TrimSpace(*keep)))
			}
		}
		_, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("cd %s && source .envrc && cd thanos-stack && terraform init && terraform plan %s && terraform apply %s", tfRoot, strings.Join(varArgs, " "), strings.Join(varArgs, " ")))
		if err != nil {
			fmt.Println("[EFS] terraform apply failed:", err)
		} else {
			fmt.Println("[EFS] terraform apply completed")
		}
	}

	// RDS via block-explorer
	{
		varArgs := []string{"-auto-approve"}
		if reset != nil && *reset {
			varArgs = append(varArgs, "-var=backup_retention_period=14", "-var=preferred_backup_window=03:00-04:00")
		} else {
			if keepRDS != nil && strings.TrimSpace(*keepRDS) != "" {
				varArgs = append(varArgs, fmt.Sprintf("-var=backup_retention_period=%s", strings.TrimSpace(*keepRDS)))
			}
			if window != nil && strings.TrimSpace(*window) != "" {
				varArgs = append(varArgs, fmt.Sprintf("-var=preferred_backup_window=%s", strings.TrimSpace(*window)))
			}
		}
		_, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("cd %s && source .envrc && cd block-explorer && terraform init && terraform plan %s && terraform apply %s", tfRoot, strings.Join(varArgs, " "), strings.Join(varArgs, " ")))
		if err != nil {
			fmt.Println("[RDS] terraform apply failed:", err)
		} else {
			fmt.Println("[RDS] terraform apply completed")
		}
	}
	return nil
}

// initializeBackupSystem initializes the backup system after deployment
func (t *ThanosStack) initializeBackupSystem(ctx context.Context, chainName string) error {
	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS config is nil")
	}

	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s config is nil")
	}

	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	// Wait for EFS to be available
	fmt.Println("Waiting for EFS to be available...")
	efsID, err := t.waitForEFS(ctx, namespace, 60) // Wait up to 60 seconds
	if err != nil {
		return fmt.Errorf("failed to detect EFS: %w", err)
	}

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// Get backup vault name
	backupVaultName := fmt.Sprintf("%s-backup-vault", chainName)

	// Start initial backup job
	fmt.Println("Starting initial backup job...")
	efsArn := utils.BuildEFSArn(region, accountID, efsID)
	iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-service-role", accountID, chainName)

	_, err = utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--region", region,
		"--backup-vault-name", backupVaultName,
		"--resource-arn", efsArn,
		"--iam-role-arn", iamRoleArn)

	if err != nil {
		return fmt.Errorf("failed to start initial backup job: %w", err)
	}

	fmt.Printf("✅ Initial backup job started for EFS: %s\n", efsID)
	return nil
}

// waitForEFS waits for EFS to be available in the cluster
func (t *ThanosStack) waitForEFS(ctx context.Context, namespace string, maxWaitSeconds int) (string, error) {
	for i := 0; i < maxWaitSeconds; i++ {
		efsID, err := utils.DetectEFSId(ctx, namespace)
		if err == nil && efsID != "" {
			return efsID, nil
		}
		time.Sleep(1 * time.Second)
	}
	return "", fmt.Errorf("EFS not available after %d seconds", maxWaitSeconds)
}

// selectRecoveryPoint shows a menu of recent recovery points and allows user selection
func (t *ThanosStack) selectRecoveryPoint(ctx context.Context) (string, error) {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	// Detect EFS ID
	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		return "", fmt.Errorf("failed to detect EFS ID: %w", err)
	}

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	arn := utils.BuildEFSArn(region, accountID, efsID)

	fmt.Printf("\n[SELECTION] Available EFS Recovery Points (FileSystemId: %s)\n", efsID)
	fmt.Println("   -------------------------------------------------")

	// Get recent recovery points (max 3)
	query := "reverse(sort_by(RecoveryPoints,&CreationDate))[:3].{RecoveryPointArn:RecoveryPointArn,BackupVaultName:BackupVaultName,CreationDate:CreationDate,Status:Status,Lifecycle:Lifecycle}"
	jsonOut, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--region", region,
		"--resource-arn", arn,
		"--query", query,
		"--output", "json")

	if err != nil {
		return "", fmt.Errorf("failed to retrieve recovery points: %w", err)
	}

	jsonOutTrimmed := strings.TrimSpace(jsonOut)
	if jsonOutTrimmed == "" || jsonOutTrimmed == "[]" {
		return "", fmt.Errorf("no recovery points found")
	}

	// Debug: Print raw JSON for troubleshooting
	fmt.Printf("[DEBUG] Raw JSON response: %s\n", jsonOutTrimmed)

	// Parse JSON to extract recovery points
	var recoveryPoints []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutTrimmed), &recoveryPoints); err != nil {
		return "", fmt.Errorf("failed to parse recovery points: %w", err)
	}

	if len(recoveryPoints) == 0 {
		return "", fmt.Errorf("no recovery points found")
	}

	// Display recovery points in a simple list format
	fmt.Println("   Available Recovery Points:")
	fmt.Println("   " + strings.Repeat("─", 80))

	var options []string
	for i, point := range recoveryPoints {
		recoveryPointArn, _ := point["RecoveryPointArn"].(string)
		vaultName, _ := point["BackupVaultName"].(string)
		createdDate, _ := point["CreationDate"].(string)
		status, _ := point["Status"].(string)

		// Format creation date for display
		displayDate := createdDate
		if createdDate != "" {
			// Try to parse and format the date with various formats
			dateFormats := []string{
				"2006-01-02T15:04:05.000000-07:00",
				"2006-01-02T15:04:05.000000+09:00",
				"2006-01-02T15:04:05.000000Z",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05",
			}

			parsed := false
			for _, format := range dateFormats {
				if t, err := time.Parse(format, createdDate); err == nil {
					displayDate = t.Format("2006-01-02 15:04:05")
					parsed = true
					break
				}
			}

			// If parsing failed, try RFC3339 format
			if !parsed {
				if t, err := time.Parse(time.RFC3339, createdDate); err == nil {
					displayDate = t.Format("2006-01-02 15:04:05")
				} else {
					// Debug: print the original date format
					fmt.Printf("[DEBUG] Failed to parse date: %s\n", createdDate)
				}
			}
		}

		// Determine availability status
		availability := "✅ Available"
		if status != "COMPLETED" {
			availability = "❌ Not Available"
		}

		// Display in a clean list format
		fmt.Printf("   %d. %s\n", i+1, displayDate)
		fmt.Printf("      📁 Vault: %s\n", vaultName)
		fmt.Printf("      📊 Status: %s\n", availability)
		fmt.Printf("      🔗 ARN: %s\n", recoveryPointArn)
		fmt.Println()

		options = append(options, recoveryPointArn)
	}

	fmt.Println("   " + strings.Repeat("─", 80))

	// Get user selection
	fmt.Print("\n   Enter your choice (1-", len(options), "): ")

	// Read user input
	var choice int
	fmt.Scanf("%d", &choice)

	// Validate choice
	if choice < 1 || choice > len(options) {
		return "", fmt.Errorf("invalid choice: %d", choice)
	}

	selectedArn := options[choice-1]
	fmt.Printf("\n   ✅ Selected recovery point: %s\n", selectedArn)

	return selectedArn, nil
}

// selectRDSSnapshot shows a menu of recent RDS snapshots and allows user selection
func (t *ThanosStack) selectRDSSnapshot(ctx context.Context) (string, error) {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	rdsID := utils.RDSIdentifierFromNamespace(namespace)

	fmt.Printf("\n[SELECTION] Available RDS Snapshots (Instance: %s)\n", rdsID)
	fmt.Println("   -------------------------------------------------")

	// Get recent snapshots (max 5)
	query := "reverse(sort_by(DBSnapshots,&SnapshotCreateTime))[:5].{DBSnapshotIdentifier:DBSnapshotIdentifier,SnapshotCreateTime:SnapshotCreateTime,Status:Status,Engine:Engine}"
	jsonOut, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-snapshots",
		"--region", region,
		"--db-instance-identifier", rdsID,
		"--query", query,
		"--output", "json")

	if err != nil {
		return "", fmt.Errorf("failed to retrieve RDS snapshots: %w", err)
	}

	jsonOutTrimmed := strings.TrimSpace(jsonOut)
	if jsonOutTrimmed == "" || jsonOutTrimmed == "[]" {
		return "", fmt.Errorf("no RDS snapshots found for instance: %s", rdsID)
	}

	// Parse JSON to extract snapshots
	var snapshots []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutTrimmed), &snapshots); err != nil {
		return "", fmt.Errorf("failed to parse RDS snapshots: %w", err)
	}

	if len(snapshots) == 0 {
		return "", fmt.Errorf("no RDS snapshots found")
	}

	// Display snapshots in a simple list format
	fmt.Println("   Available RDS Snapshots:")
	fmt.Println("   " + strings.Repeat("─", 80))

	var options []string
	for i, snapshot := range snapshots {
		snapshotID, _ := snapshot["DBSnapshotIdentifier"].(string)
		createTime, _ := snapshot["SnapshotCreateTime"].(string)
		status, _ := snapshot["Status"].(string)
		engine, _ := snapshot["Engine"].(string)

		// Format creation date for display
		displayDate := createTime
		if createTime != "" {
			// Try to parse and format the date with various formats
			dateFormats := []string{
				"2006-01-02T15:04:05.000000-07:00",
				"2006-01-02T15:04:05.000000+09:00",
				"2006-01-02T15:04:05.000000Z",
				"2006-01-02T15:04:05Z",
				"2006-01-02T15:04:05",
			}

			parsed := false
			for _, format := range dateFormats {
				if t, err := time.Parse(format, createTime); err == nil {
					displayDate = t.Format("2006-01-02 15:04:05")
					parsed = true
					break
				}
			}

			// If parsing failed, try RFC3339 format
			if !parsed {
				if t, err := time.Parse(time.RFC3339, createTime); err == nil {
					displayDate = t.Format("2006-01-02 15:04:05")
				} else {
					// Debug: print the original date format
					fmt.Printf("[DEBUG] Failed to parse date: %s\n", createTime)
				}
			}
		}

		// Display in a clean list format
		fmt.Printf("   %d. %s\n", i+1, displayDate)
		fmt.Printf("      🗄️  Engine: %s\n", engine)
		fmt.Printf("      📊 Status: %s\n", status)
		fmt.Printf("      🔗 Snapshot ID: %s\n", snapshotID)
		fmt.Println()

		options = append(options, snapshotID)
	}

	fmt.Println("   " + strings.Repeat("─", 80))

	// Get user selection
	fmt.Print("\n   Enter your choice (1-", len(options), "): ")

	// Read user input
	var choice int
	fmt.Scanf("%d", &choice)

	// Validate choice
	if choice < 1 || choice > len(options) {
		return "", fmt.Errorf("invalid choice: %d", choice)
	}

	selectedSnapshot := options[choice-1]
	fmt.Printf("\n   ✅ Selected snapshot: %s\n", selectedSnapshot)

	return selectedSnapshot, nil
}

// interactiveRestore provides a fully interactive restore experience for EFS and RDS
func (t *ThanosStack) interactiveRestore(ctx context.Context) error {
	fmt.Println("\n🔄 Starting Interactive Restore")
	fmt.Println("================================")

	// Step 1: Choose what to restore
	fmt.Println("\n📋 Step 1: Choose Restore Type")
	fmt.Println("   What would you like to restore?")
	fmt.Println("   1. EFS (Elastic File System)")
	fmt.Println("   2. RDS (Database)")
	fmt.Println("   3. Both EFS and RDS")
	fmt.Println("   0. Cancel")
	fmt.Print("   Enter your choice (0-3): ")

	var restoreType int
	fmt.Scanf("%d", &restoreType)

	switch restoreType {
	case 0:
		fmt.Println("   ↩️  Restore cancelled. Returning to main menu...")
		return nil // Return to main menu instead of error
	case 1:
		return t.interactiveEFSRestore(ctx)
	case 2:
		return t.interactiveRDSRestore(ctx)
	case 3:
		return t.interactiveFullRestore(ctx)
	default:
		return fmt.Errorf("invalid choice: %d", restoreType)
	}
}

// interactiveEFSRestore handles EFS-only restore
func (t *ThanosStack) interactiveEFSRestore(ctx context.Context) error {
	fmt.Println("\n📁 EFS Restore")
	fmt.Println("==============")

	// Step 1: Select recovery point
	fmt.Println("\n📋 Step 1: Select Recovery Point")
	selectedPoint, err := t.selectRecoveryPoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to select recovery point: %w", err)
	}

	// Step 2: Restore Options
	fmt.Println("\n📋 Step 2: Restore Options")
	fmt.Println("   Will restore as new filesystem (recommended)")
	fmt.Println("   This prevents overwriting the existing EFS.")

	// Step 3: Execute restore
	fmt.Println("\n🚀 Starting EFS Restore...")
	newEfsID, err := t.restoreEFS(ctx, selectedPoint)
	if err != nil {
		return fmt.Errorf("EFS restore failed: %w", err)
	}

	// Step 5: Verify restore and provide post-restore guidance
	fmt.Println("\n📋 Step 4: Post-Restore Actions")
	fmt.Println("   ✅ EFS restore completed successfully!")

	if newEfsID != "" {
		fmt.Printf("   New EFS ID: %s\n", newEfsID)

		// Verify the restore
		if err := t.verifyEFSRestore(ctx, newEfsID); err != nil {
			fmt.Printf("   ⚠️  Verification warning: %v\n", err)
		}

		fmt.Println("   ")
		fmt.Println("   Next steps:")
		fmt.Println("   1. The new EFS needs to be attached to your workloads")
		fmt.Println("   2. Run the following command to attach the new EFS:")
		fmt.Printf("      ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node\n", newEfsID)
		fmt.Println("   ")
		fmt.Println("   Would you like to proceed with attach now? (y/n): ")

		var attachChoice string
		fmt.Scanf("%s", &attachChoice)
		attachChoice = strings.ToLower(strings.TrimSpace(attachChoice))

		if attachChoice == "y" || attachChoice == "yes" {
			fmt.Println("\n🔗 Starting Attach Process...")

			efsID := &newEfsID
			pvcs := "op-geth,op-node"
			sts := "op-geth,op-node"

			if err := t.BackupAttach(ctx, efsID, &pvcs, &sts); err != nil {
				fmt.Printf("   ⚠️  Attach failed: %v\n", err)
				fmt.Printf("   You can try attaching manually later using: ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node\n", newEfsID)
			} else {
				fmt.Println("   ✅ Attach completed successfully!")
			}
		} else {
			fmt.Printf("   ℹ️  You can attach the EFS later using: ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node\n", newEfsID)
		}
	} else {
		fmt.Println("   ⚠️  No new EFS ID detected from restore")
		fmt.Println("   You may need to manually attach the restored EFS")
	}

	fmt.Println("\n🎉 EFS restore completed!")
	return fmt.Errorf("completed") // Special marker for successful completion
}

// interactiveRDSRestore handles RDS-only restore
func (t *ThanosStack) interactiveRDSRestore(ctx context.Context) error {
	fmt.Println("\n🗄️  RDS Restore")
	fmt.Println("==============")

	// Step 1: Select snapshot
	fmt.Println("\n📋 Step 1: Select RDS Snapshot")
	selectedSnapshot, err := t.selectRDSSnapshot(ctx)
	if err != nil {
		return fmt.Errorf("failed to select RDS snapshot: %w", err)
	}

	// Step 2: Enter target instance name
	fmt.Println("\n📋 Step 2: Target Instance")
	fmt.Println("   Enter the name for the new RDS instance:")
	fmt.Print("   Instance name: ")

	var targetInstance string
	fmt.Scanf("%s", &targetInstance)
	targetInstance = strings.TrimSpace(targetInstance)

	if targetInstance == "" {
		namespace := t.deployConfig.K8s.Namespace
		rdsID := utils.RDSIdentifierFromNamespace(namespace)
		targetInstance = fmt.Sprintf("%s-restore-%d", rdsID, time.Now().Unix())
		fmt.Printf("   Using default name: %s\n", targetInstance)
	}

	// Step 3: Execute restore
	fmt.Println("\n🚀 Starting RDS Restore...")
	if err := t.restoreRDS(ctx, selectedSnapshot, targetInstance); err != nil {
		return fmt.Errorf("RDS restore failed: %w", err)
	}

	fmt.Println("\n📋 Step 4: Post-Restore Actions")
	fmt.Println("   ✅ RDS restore completed successfully!")
	fmt.Printf("   New RDS Instance: %s\n", targetInstance)

	// Verify the restore
	if err := t.verifyRDSRestore(ctx, targetInstance); err != nil {
		fmt.Printf("   ⚠️  Verification warning: %v\n", err)
	}

	fmt.Println("   ")
	fmt.Println("   Next steps:")
	fmt.Println("   1. Update your application configuration to use the new RDS instance")
	fmt.Println("   2. Test the connection to ensure it's working properly")
	fmt.Println("   3. Update DNS or connection strings if necessary")

	fmt.Println("\n🎉 RDS restore completed!")
	return fmt.Errorf("completed") // Special marker for successful completion
}

// interactiveFullRestore handles both EFS and RDS restore
func (t *ThanosStack) interactiveFullRestore(ctx context.Context) error {
	fmt.Println("\n🔄 Full Restore (EFS + RDS)")
	fmt.Println("============================")

	// Restore EFS first
	fmt.Println("\n📁 Step 1: EFS Restore")
	efsErr := t.interactiveEFSRestore(ctx)
	if efsErr != nil {
		if efsErr.Error() == "completed" {
			// EFS restore completed successfully, continue with RDS
		} else {
			return fmt.Errorf("EFS restore failed: %w", efsErr)
		}
	} else {
		// User went back from EFS restore
		fmt.Println("   ℹ️  EFS restore was cancelled")
		fmt.Println("   Do you want to continue with RDS restore? (y/n): ")

		var continueChoice string
		fmt.Scanf("%s", &continueChoice)
		continueChoice = strings.ToLower(strings.TrimSpace(continueChoice))

		if continueChoice != "y" && continueChoice != "yes" {
			fmt.Println("   ↩️  Going back to main menu...")
			return nil
		}
	}

	// Then restore RDS
	fmt.Println("\n🗄️  Step 2: RDS Restore")
	rdsErr := t.interactiveRDSRestore(ctx)
	if rdsErr != nil {
		if rdsErr.Error() == "completed" {
			// RDS restore completed successfully
		} else {
			return fmt.Errorf("RDS restore failed: %w", rdsErr)
		}
	} else {
		// User went back from RDS restore
		fmt.Println("   ℹ️  RDS restore was cancelled")
	}

	fmt.Println("\n🎉 Full restore completed successfully!")
	return nil
}

// verifyEFSRestore verifies that the EFS restore was successful
func (t *ThanosStack) verifyEFSRestore(ctx context.Context, efsID string) error {
	fmt.Println("\n🔍 Verifying EFS Restore...")

	// Check EFS status
	status, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-ids", efsID,
		"--query", "FileSystems[0].LifeCycleState",
		"--output", "text")

	if err != nil {
		return fmt.Errorf("failed to check EFS status: %w", err)
	}

	status = strings.TrimSpace(status)
	fmt.Printf("   📊 EFS Status: %s\n", status)

	if status != "available" {
		return fmt.Errorf("EFS is not in available state: %s", status)
	}

	// Check EFS mount targets
	mountTargets, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-id", efsID,
		"--query", "MountTargets[].LifeCycleState",
		"--output", "text")

	if err != nil {
		fmt.Printf("   ⚠️  Warning: Could not check mount targets: %v\n", err)
	} else {
		mountTargets = strings.TrimSpace(mountTargets)
		if mountTargets != "" {
			fmt.Printf("   📍 Mount Targets Status: %s\n", mountTargets)
		} else {
			fmt.Println("   📍 No mount targets found (normal for new EFS)")
		}
	}

	// Get EFS details for user reference
	efsDetails, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-ids", efsID,
		"--query", "FileSystems[0].{Name:Name,SizeInBytes:SizeInBytes.Value,Encrypted:Encrypted}",
		"--output", "json")

	if err == nil {
		fmt.Printf("   📋 EFS Details: %s\n", efsDetails)
	}

	// Provide AWS Console link
	region := t.deployConfig.AWS.Region
	consoleLink := fmt.Sprintf("https://%s.console.aws.amazon.com/efs/home?region=%s#/file-systems/%s", region, region, efsID)
	fmt.Printf("   🔗 AWS Console: %s\n", consoleLink)

	fmt.Println("   ✅ EFS restore verification completed!")
	return nil
}

// verifyRDSRestore verifies that the RDS restore was successful
func (t *ThanosStack) verifyRDSRestore(ctx context.Context, instanceID string) error {
	fmt.Println("\n🔍 Verifying RDS Restore...")

	// Check RDS instance status
	status, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", t.deployConfig.AWS.Region,
		"--db-instance-identifier", instanceID,
		"--query", "DBInstances[0].DBInstanceStatus",
		"--output", "text")

	if err != nil {
		return fmt.Errorf("failed to check RDS status: %w", err)
	}

	status = strings.TrimSpace(status)
	fmt.Printf("   📊 RDS Status: %s\n", status)

	if status != "available" {
		return fmt.Errorf("RDS is not in available state: %s", status)
	}

	// Get connection information
	endpoint, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", t.deployConfig.AWS.Region,
		"--db-instance-identifier", instanceID,
		"--query", "DBInstances[0].Endpoint.Address",
		"--output", "text")

	if err == nil {
		endpoint = strings.TrimSpace(endpoint)
		fmt.Printf("   🌐 Endpoint: %s\n", endpoint)
	}

	port, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", t.deployConfig.AWS.Region,
		"--db-instance-identifier", instanceID,
		"--query", "DBInstances[0].Endpoint.Port",
		"--output", "text")

	if err == nil {
		port = strings.TrimSpace(port)
		fmt.Printf("   🔌 Port: %s\n", port)
	}

	// Get RDS details
	rdsDetails, err := utils.ExecuteCommand(ctx, "aws", "rds", "describe-db-instances",
		"--region", t.deployConfig.AWS.Region,
		"--db-instance-identifier", instanceID,
		"--query", "DBInstances[0].{Engine:Engine,EngineVersion:EngineVersion,DBInstanceClass:DBInstanceClass,AllocatedStorage:AllocatedStorage}",
		"--output", "json")

	if err == nil {
		fmt.Printf("   📋 RDS Details: %s\n", rdsDetails)
	}

	// Provide AWS Console link
	region := t.deployConfig.AWS.Region
	consoleLink := fmt.Sprintf("https://%s.console.aws.amazon.com/rds/home?region=%s#database:id=%s;is-cluster=false", region, region, instanceID)
	fmt.Printf("   🔗 AWS Console: %s\n", consoleLink)

	fmt.Println("   ✅ RDS restore verification completed!")
	return nil
}
