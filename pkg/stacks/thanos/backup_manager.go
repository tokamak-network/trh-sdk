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

	t.logger.Infof("Region: %s, Namespace: %s, Account ID: %s", region, namespace, accountID)

	if efsID, err := utils.DetectEFSId(ctx, namespace); err == nil && efsID != "" {
		arn := utils.BuildEFSArn(region, accountID, efsID)
		t.logger.Info("📁 EFS Backup Status")
		t.logger.Infof("   ARN: %s", arn)

		// Check if EFS is protected
		cnt, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources", "--region", region, "--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", arn), "--output", "text")
		if err != nil {
			t.logger.Errorf("   Protected: ❌ Error checking protection status: %v", err)
		} else {
			protected := strings.TrimSpace(cnt)
			if protected == "1" {
				t.logger.Info("   Protected: ✅ true")
			} else {
				t.logger.Warn("   Protected: ❌ false")
			}
		}

		// Check latest recovery point
		rp, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", "max_by(RecoveryPoints,&CreationDate).CreationDate", "--output", "text")
		if err != nil {
			t.logger.Errorf("   Latest recovery point: ❌ Error checking recovery points: %v", err)
		} else {
			rpTrimmed := strings.TrimSpace(rp)
			if rpTrimmed == "None" || rpTrimmed == "" {
				t.logger.Warn("   Latest recovery point: ⚠️  None (no backups found)")
			} else {
				t.logger.Infof("   Latest recovery point: ✅ %s", rpTrimmed)

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
						t.logger.Infof("   Expected expiry date: 📅 %s (35 days from creation)", expiryTime.Format("2006-01-02T15:04:05-07:00"))
					}
				}
			}
		}

		// Check backup vaults (simplified query)
		vaultsJSON, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", "RecoveryPoints[].BackupVaultName", "--output", "json")
		if err != nil {
			t.logger.Errorf("   Vaults: ❌ Error checking vaults: %v", err)
		} else {
			vaultsJSON = strings.TrimSpace(vaultsJSON)
			if vaultsJSON == "" || vaultsJSON == "null" || vaultsJSON == "[]" {
				t.logger.Warn("   Vaults: ⚠️  None (no backups found)")
			} else {
				var names []string
				if err := json.Unmarshal([]byte(vaultsJSON), &names); err == nil && len(names) > 0 {
					seen := map[string]struct{}{}
					unique := ""
					for _, n := range names {
						if _, ok := seen[n]; ok {
							continue
						}
						seen[n] = struct{}{}
						unique = n
						break
					}
					if unique != "" {
						t.logger.Infof("   Vaults: ✅ %s", unique)
					} else {
						t.logger.Warn("   Vaults: ⚠️  None (no backups found)")
					}
				} else {
					t.logger.Warn("   Vaults: ⚠️  None (no backups found)")
				}
			}
		}
	} else {
		t.logger.Infof("📁 EFS Backup Status")
		t.logger.Errorf("   ❌ Not detected in cluster PVs: %v", err)
	}

	t.logger.Info("")

	return nil
}

// BackupSnapshot triggers on-demand EFS backup and RDS snapshot
func (t *ThanosStack) BackupSnapshot(ctx context.Context) error {
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
			t.logger.Errorf("📁 EFS: ❌ Failed to start backup job: %v", err)
			t.logger.Infof("   Backup vault: %s", backupVaultName)
			t.logger.Infof("   IAM role: %s", iamRoleArn)
		} else {
			jobId := strings.TrimSpace(backupJobOutput)
			t.logger.Infof("📁 EFS: ✅ On-demand backup started successfully")
			t.logger.Infof("   Job ID: %s", jobId)
			t.logger.Infof("   Backup vault: %s", backupVaultName)
		}
	} else {
		t.logger.Infof("📁 EFS: ❌ Not detected in cluster PVs: %v", err)
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

	t.logger.Infof("Region: %s, Namespace: %s, Account ID: %s", region, namespace, accountID)
	t.logger.Info("")

	if efsID, err := utils.DetectEFSId(ctx, namespace); err == nil && efsID != "" {
		arn := utils.BuildEFSArn(region, accountID, efsID)
		t.logger.Infof("📁 EFS Recovery Points (FileSystemId: %s)", efsID)

		// First get the data in JSON format to process expiry dates
		jsonQuery := "reverse(sort_by(RecoveryPoints,&CreationDate))[:10]"
		if limit != "" {
			jsonQuery = fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s]", limit)
		}
		jsonOut, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", jsonQuery, "--output", "json")
		if err != nil {
			t.logger.Infof("   ❌ Error retrieving recovery points: %v", err)
		} else {
			// Process JSON to calculate expiry dates
			jsonOutTrimmed := strings.TrimSpace(jsonOut)
			if jsonOutTrimmed == "" || jsonOutTrimmed == "[]" {
				t.logger.Infof("   ⚠️  No recovery points found")
			} else {
				// For now, use the table format but with calculated expiry dates
				query := "reverse(sort_by(RecoveryPoints,&CreationDate))[:10].{Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}"
				if limit != "" {
					query = fmt.Sprintf("reverse(sort_by(RecoveryPoints,&CreationDate))[:%s].{Vault:BackupVaultName,Created:CreationDate,Expiry:ExpiryDate,Status:Status}", limit)
				}
				out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", query, "--output", "table")
				if err != nil {
					t.logger.Infof("   ❌ Error retrieving recovery points: %v", err)
				} else {
					outTrimmed := strings.TrimSpace(out)
					if outTrimmed == "" {
						t.logger.Infof("   ⚠️  No recovery points found")
					} else {
						// Add indentation to the table output and replace the header
						lines := strings.Split(outTrimmed, "")
						for i, line := range lines {
							if i == 0 {
								// Replace the AWS CLI generated header with a custom one
								t.logger.Infof("   -------------------------------------------------")
								t.logger.Infof("   |                EFS Recovery Points            |")
								t.logger.Infof("   +---------+-------------------------------------+")
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
														t.logger.Infof("   %s", modifiedLine)
														continue
													}
												}
											}
										}
									}
								}
								t.logger.Infof("   %s", line) // Regular data rows
							}
						}
					}
				}
			}
		}
	} else {
		t.logger.Infof("📁 EFS Recovery Points")
		t.logger.Errorf("   ❌ Not detected in cluster PVs: %v", err)
	}

	t.logger.Info("")

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
	t.logger.Info("Checking prerequisites...")

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

	t.logger.Info("✅ Prerequisites check passed")
	return nil
}

// validateAWSCredentials validates AWS credentials and permissions
func (t *ThanosStack) validateAWSCredentials(ctx context.Context) error {
	t.logger.Info("Checking AWS credentials and permissions...")

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

	t.logger.Infof("✅ AWS credentials valid (Account: %s)", accountID)
	return nil
}

// restoreEFS handles EFS restore from recovery point
func (t *ThanosStack) restoreEFS(ctx context.Context, recoveryPointArn string) (string, error) {
	t.logger.Infof("Starting restore from recovery point: %s", recoveryPointArn)

	// Validate recovery point ARN format
	if !strings.Contains(recoveryPointArn, "arn:aws:backup:") {
		return "", fmt.Errorf("invalid recovery point ARN format: %s", recoveryPointArn)
	}

	// Check if recovery point exists and is accessible
	// First, get the recovery point details to extract vault name

	// For this ARN format, we need to find the vault name by listing all vaults
	// and checking which one contains this recovery point

	// List all backup vaults
	vaultsOutput, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults",
		"--region", t.deployConfig.AWS.Region,
		"--query", "BackupVaultList[].BackupVaultName",
		"--output", "text")

	if err != nil {
		return "", fmt.Errorf("failed to list backup vaults: %w", err)
	}

	vaultNames := strings.Fields(strings.TrimSpace(vaultsOutput))

	// Try to find the vault that contains this recovery point
	var recoveryPointDetails string
	var foundVault string

	for _, vaultName := range vaultNames {
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
			break
		}
	}

	if foundVault == "" {
		// If we couldn't find it in any vault, try without vault name
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
		_, fullErr := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
			"--region", t.deployConfig.AWS.Region,
			"--recovery-point-arn", recoveryPointArn)
		if fullErr != nil {
			return "", fmt.Errorf("recovery point not found or not accessible: %w (full error: %v)", err, fullErr)
		}

		return "", fmt.Errorf("recovery point not found or not accessible: %w", err)
	}

	vaultName := strings.TrimSpace(recoveryPointDetails)
	if vaultName == "" {
		return "", fmt.Errorf("could not extract vault name from recovery point")
	}

	// Get appropriate IAM role for restore
	iamRoleArn, err := t.getRestoreIAMRole(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get IAM role for restore: %w", err)
	}

	// Start restore job
	// For EFS restore, we need to get the required metadata from the recovery point
	// and specify that we want a new filesystem
	recoveryPointMetadata, err := utils.ExecuteCommand(ctx, "aws", "backup", "get-recovery-point-restore-metadata",
		"--backup-vault-name", vaultName,
		"--recovery-point-arn", recoveryPointArn,
		"--region", t.deployConfig.AWS.Region,
		"--query", "RestoreMetadata",
		"--output", "json")
	if err != nil {
		return "", fmt.Errorf("failed to get recovery point metadata: %w", err)
	}

	// Parse the metadata to get the file-system-id
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(recoveryPointMetadata), &metadata); err != nil {
		return "", fmt.Errorf("failed to parse recovery point metadata: %w", err)
	}

	// Add the newfilesystem flag to create a new EFS
	// Add required EFS restore metadata
	metadata["newfilesystem"] = "true"
	metadata["performancemode"] = "generalPurpose"
	metadata["creationtoken"] = fmt.Sprintf("%s-restore-%d", t.deployConfig.K8s.Namespace, time.Now().Unix())
	metadata["encrypted"] = "true"

	// Get KMS key ID from original EFS
	efsInfo, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-id", metadata["file-system-id"].(string),
		"--query", "FileSystems[0].KmsKeyId",
		"--output", "text")
	if err == nil && strings.TrimSpace(efsInfo) != "" {
		metadata["kmskeyid"] = strings.TrimSpace(efsInfo)
	}

	// Convert back to JSON
	metaBytes, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("failed to marshal restore metadata: %w", err)
	}
	meta := string(metaBytes)

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

	t.logger.Infof("Restore job started, JobId: %s", jobId)

	// Monitor restore job progress
	return t.monitorEFSRestoreJob(ctx, jobId)
}

// monitorEFSRestoreJob monitors EFS restore job progress
func (t *ThanosStack) monitorEFSRestoreJob(ctx context.Context, jobId string) (string, error) {
	t.logger.Info("Monitoring restore job progress...")

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
		t.logger.Infof("Job status: %s (attempt %d/%d)", status, i+1, maxAttempts)

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
			t.logger.Infof("Unknown status: %s, continuing to monitor...", status)
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
	t.logger.Infof("Restore completed. CreatedResourceArn: %s", createdArn)

	var newFsId string
	if strings.Contains(createdArn, ":file-system/") {
		parts := strings.Split(createdArn, "/")
		if len(parts) > 0 {
			newFsId = parts[len(parts)-1]

			// Ensure throughput mode is elastic on the restored EFS
			if err := t.setEFSThroughputElastic(ctx, newFsId); err != nil {
				t.logger.Warnf("Failed to set EFS ThroughputMode to elastic: %v", err)
			} else {
				t.logger.Info("✅ ThroughputMode set to elastic")
			}
		}
	}

	return newFsId, nil
}

// setEFSThroughputElastic updates an EFS to elastic throughput mode
func (t *ThanosStack) setEFSThroughputElastic(ctx context.Context, fsId string) error {
	if strings.TrimSpace(fsId) == "" {
		return fmt.Errorf("empty file system id")
	}
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "update-file-system",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-id", fsId,
		"--throughput-mode", "elastic")
	return err
}

// getRestoreIAMRole gets appropriate IAM role for restore operations
func (t *ThanosStack) getRestoreIAMRole(ctx context.Context) (string, error) {
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	namespace := t.deployConfig.K8s.Namespace

	// First, try the namespace-based backup service role (same as used in backup creation)
	namespaceRoleName := fmt.Sprintf("%s-backup-service-role", namespace)
	namespaceRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, namespaceRoleName)

	// Check if the namespace-based role exists and is accessible
	if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", namespaceRoleName); err == nil {
		return namespaceRoleArn, nil
	}

	// If namespace-based role not found, try AWS managed roles
	awsManagedRoles := []string{
		"AWSBackupDefaultServiceRole",
		"AWSServiceRoleForBackup",
	}

	for _, roleName := range awsManagedRoles {
		roleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, roleName)

		if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
			return roleArn, nil
		}
	}

	// If no suitable role found, return the namespace-based role and let AWS handle the error
	t.logger.Warnf("No suitable IAM role found, using namespace-based role: %s", namespaceRoleArn)
	return namespaceRoleArn, nil
}

// BackupAttach switches workloads to use the new EFS and verifies readiness
func (t *ThanosStack) BackupAttach(ctx context.Context, efsId *string, pvcs *string, stss *string) error {
	namespace := t.deployConfig.K8s.Namespace

	// Check if no parameters provided - show usage
	if (efsId == nil || strings.TrimSpace(*efsId) == "") &&
		(pvcs == nil || strings.TrimSpace(*pvcs) == "") &&
		(stss == nil || strings.TrimSpace(*stss) == "") {
		t.showAttachUsage()
		return fmt.Errorf("at least one parameter (--efs-id, --pvc, or --sts) must be provided")
	}

	t.logger.Info("Verifying restored data and switching workloads...")

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

	// Update PV volumeHandle if EFS ID provided
	if newEfs != "" {
		// Note: StorageClass validation is not needed for BackupAttach
		// since we only modify existing PV volumeHandle, not create new resources

		// 1) Try to replicate mount targets of the currently used EFS to the new EFS FIRST
		if srcEfs, err := utils.DetectEFSId(ctx, namespace); err == nil && strings.TrimSpace(srcEfs) != "" {
			if err := t.replicateEFSMountTargets(ctx, t.deployConfig.AWS.Region, strings.TrimSpace(srcEfs), newEfs); err != nil {
				t.logger.Warnf("Failed to replicate EFS mount targets from %s to %s: %v", srcEfs, newEfs, err)
			} else {
				t.logger.Infof("✅ Replicated mount targets from %s to %s (region: %s)", srcEfs, newEfs, t.deployConfig.AWS.Region)
				// Validate destination EFS mount targets and security groups before moving on
				if vErr := t.validateEFSMountTargets(ctx, t.deployConfig.AWS.Region, newEfs); vErr != nil {
					t.logger.Warnf("Mount target validation for %s reported issues: %v", newEfs, vErr)
				} else {
					t.logger.Infof("✅ Mount targets validated for %s", newEfs)
				}
			}
		} else {
			t.logger.Warnf("Could not detect source EFS in namespace %s. Skipping mount-target replication.", namespace)
		}

		// 2) Preflight DNS/NFS connectivity test from inside the namespace
		// if err := t.preflightEFSConnectivity(ctx, namespace, newEfs, t.deployConfig.AWS.Region); err != nil {
		// 	return fmt.Errorf("preflight connectivity failed: %w", err)
		// }

		// 3) Now verify data via temporary pod (after mount targets are ready)
		if err := t.verifyEFSData(ctx, namespace); err != nil {
			t.logger.Warnf("EFS data verification failed: %v", err)
		}

		// Ensure verify pod is removed before proceeding to destructive changes
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")

		// 2) Backup PV/PVC definitions before destructive changes
		{
			choice := "y"
			fmt.Print("Run PV/PVC backup before changes? (y/n): ")
			fmt.Scanf("%s", &choice)
			choice = strings.ToLower(strings.TrimSpace(choice))
			if choice == "y" || choice == "yes" {
				if err := t.backupPvPvc(ctx, namespace); err != nil {
					t.logger.Warnf("Failed to run PV/PVC backup script: %v", err)
				}
			} else {
				t.logger.Info("Skipped PV/PVC backup by user choice")
			}
		}

		if err := t.updatePVVolumeHandles(ctx, namespace, newEfs, pvcs); err != nil {
			return fmt.Errorf("failed to update PV volume handles: %w", err)
		}
	} else {
		t.logger.Info("Skipped PV update (no --efs-id provided).")

		// Still verify current EFS data even when not switching to new EFS
		if err := t.verifyEFSData(ctx, namespace); err != nil {
			t.logger.Warnf("Current EFS data verification failed: %v", err)
		}

		// Ensure verify pod is removed
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	}

	// Restart workloads if specified
	if stss != nil && strings.TrimSpace(*stss) != "" {
		if err := t.restartStatefulSets(ctx, namespace, strings.TrimSpace(*stss)); err != nil {
			return fmt.Errorf("failed to restart StatefulSets: %w", err)
		}
	}

	// Post-attach verification: Verify EFS data integrity through actual service restart
	if newEfs != "" {
		t.logger.Info("Starting post-attach verification...")
		if err := t.verifyPostAttach(ctx, namespace); err != nil {
			t.logger.Warnf("Post-attach verification failed: %v", err)
			t.logger.Warn("This may indicate data corruption or incomplete restore")
		} else {
			t.logger.Info("✅ Post-attach verification completed successfully")
		}
	}
	// Update AWS Backup protected resources if EFS changed
	if newEfs != "" {
		t.logger.Info("")
		if err := t.updateBackupProtectedResources(ctx, newEfs); err != nil {
			t.logger.Warnf("Failed to update AWS Backup protected resources: %v", err)
			t.logger.Info("You may need to manually update backup configuration for the new EFS")
		} else {
			t.logger.Info("✅ AWS Backup protected resources updated successfully")
		}
	}

	// Summary
	t.printAttachSummary(ctx, newEfs, pvcs, stss)
	return nil
}

// verifyPostAttach performs post-attach verification by monitoring op-geth and op-node pods
func (t *ThanosStack) verifyPostAttach(ctx context.Context, namespace string) error {
	t.logger.Info("Checking pod status after EFS attach...")

	components := []string{"op-geth", "op-node"}
	var errors []string

	for _, component := range components {
		if err := t.monitorComponentPod(ctx, namespace, component); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", component, err))
		}
	}

	if len(errors) > 0 {
		t.logger.Infof("❌ Failed: %s", strings.Join(errors, ", "))
		return fmt.Errorf("verification failed")
	}

	return nil
}

// monitorComponentPod monitors a component pod until it's running successfully
func (t *ThanosStack) monitorComponentPod(ctx context.Context, namespace, component string) error {
	// Find pod by component name pattern
	podName, err := t.findComponentPod(ctx, namespace, component)
	if err != nil {
		return fmt.Errorf("failed to find %s pod: %w", component, err)
	}

	t.logger.Infof("[MONITOR-%s] Found pod: %s", strings.ToUpper(component), podName)

	// Wait for pod to be ready (up to 5 minutes)
	t.logger.Infof("Checking %s pod status...", component)
	maxWaitTime := 60 // 60 * 5s = 5 minutes

	for i := 0; i < maxWaitTime; i++ {
		// Get pod status
		phase, _ := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.phase}")
		ready, _ := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.containerStatuses[0].ready}")

		phase = strings.TrimSpace(phase)
		ready = strings.TrimSpace(ready)

		// Check for success condition
		if phase == "Running" && ready == "true" {
			t.logger.Infof("✅ %s is ready", component)
			return nil
		}

		// Check for error conditions
		if phase == "Failed" || phase == "CrashLoopBackOff" {
			return t.handlePodError(ctx, namespace, component, podName, phase)
		}

		// Print status every 30 seconds
		if i%6 == 0 {
			t.logger.Infof("%s status: %s", component, phase)
		}

		time.Sleep(5 * time.Second)
	}

	// Timeout - pod not ready
	t.logger.Infof("❌ %s timeout after 5 minutes", component)
	return fmt.Errorf("%s pod timeout", component)
}

// findComponentPod finds the pod name for a given component
func (t *ThanosStack) findComponentPod(ctx context.Context, namespace, component string) (string, error) {
	// Try to find pod by name pattern
	podList, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pods", "-n", namespace,
		"-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return "", fmt.Errorf("failed to list pods: %w", err)
	}

	pods := strings.Fields(strings.TrimSpace(podList))
	for _, pod := range pods {
		if strings.Contains(pod, component) {
			return pod, nil
		}
	}

	return "", fmt.Errorf("no pod found matching pattern '%s'", component)
}

// handlePodError handles pod error conditions with essential diagnostics
func (t *ThanosStack) handlePodError(ctx context.Context, namespace, component, podName, errorType string) error {
	t.logger.Errorf("%s pod failed: %s", component, errorType)

	// Get recent logs for diagnosis
	logs, err := utils.ExecuteCommand(ctx, "kubectl", "logs", podName, "-n", namespace, "--tail=20")
	if err == nil && strings.TrimSpace(logs) != "" {
		t.logger.Errorf("Recent logs:\n%s", logs)
	}

	// Get critical events
	events, err := utils.ExecuteCommand(ctx, "kubectl", "get", "events", "-n", namespace,
		"--field-selector", fmt.Sprintf("involvedObject.name=%s", podName),
		"--sort-by", ".lastTimestamp", "-o", "custom-columns=REASON:.reason,MESSAGE:.message", "--no-headers")
	if err == nil && strings.TrimSpace(events) != "" {
		t.logger.Errorf("Events: %s", strings.TrimSpace(events))
	}

	return fmt.Errorf("%s pod error: %s", component, errorType)
}

// verifyComponentHealth performs basic health checks
func (t *ThanosStack) verifyComponentHealth(ctx context.Context, namespace, component, podName string) error {
	// Simple readiness check - if pod is running and ready, consider it healthy
	ready, _ := utils.ExecuteCommand(ctx, "kubectl", "get", "pod", podName, "-n", namespace, "-o", "jsonpath={.status.containerStatuses[0].ready}")
	if strings.TrimSpace(ready) == "true" {
		return nil
	}
	return fmt.Errorf("%s health check failed", component)
}

// verifyGethHealth performs op-geth specific health checks
func (t *ThanosStack) verifyGethHealth(ctx context.Context, namespace, podName string) error {
	t.logger.Info("Attempting Geth RPC health check...")

	// Try to execute a simple RPC call inside the pod
	rpcResult, err := utils.ExecuteCommand(ctx, "kubectl", "exec", podName, "-n", namespace, "--",
		"timeout", "10", "geth", "attach", "/db/geth.ipc", "--exec", "eth.blockNumber")

	if err != nil {
		return fmt.Errorf("geth RPC call failed: %w", err)
	}

	blockNum := strings.TrimSpace(rpcResult)
	if blockNum != "" && blockNum != "null" && blockNum != "0" {
		t.logger.Infof("✅ Successfully retrieved block number: %s", blockNum)

		// Additional check: Get chain ID
		chainIdResult, err := utils.ExecuteCommand(ctx, "kubectl", "exec", podName, "-n", namespace, "--",
			"timeout", "10", "geth", "attach", "/db/geth.ipc", "--exec", "eth.chainId")

		if err == nil {
			chainId := strings.TrimSpace(chainIdResult)
			if chainId != "" && chainId != "null" {
				t.logger.Infof("✅ Chain ID: %s", chainId)
			}
		}

		return nil
	}

	return fmt.Errorf("geth RPC returned empty or invalid block number: %s", blockNum)
}

// verifyOpNodeHealth performs op-node specific health checks
func (t *ThanosStack) verifyOpNodeHealth(ctx context.Context, namespace, podName string) error {
	t.logger.Info("Attempting op-node health check...")

	// Check if op-node is responding to basic queries
	// Try to get sync status or version info
	versionResult, err := utils.ExecuteCommand(ctx, "kubectl", "exec", podName, "-n", namespace, "--",
		"timeout", "10", "curl", "-s", "http://localhost:8545", "-X", "POST",
		"-H", "Content-Type: application/json",
		"-d", `{"jsonrpc":"2.0","method":"optimism_version","params":[],"id":1}`)

	if err != nil {
		// Fallback: Check if the process is running
		processCheck, procErr := utils.ExecuteCommand(ctx, "kubectl", "exec", podName, "-n", namespace, "--",
			"pgrep", "-f", "op-node")

		if procErr != nil {
			return fmt.Errorf("op-node process check failed: %w", procErr)
		}

		if strings.TrimSpace(processCheck) != "" {
			t.logger.Infof("✅ op-node process is running (PID: %s)", strings.TrimSpace(processCheck))
			return nil
		}

		return fmt.Errorf("op-node RPC call failed and process not found: %w", err)
	}

	// Parse JSON response to check if it's valid
	if strings.Contains(versionResult, "jsonrpc") || strings.Contains(versionResult, "result") {
		t.logger.Infof("✅ op-node RPC is responding")
		return nil
	}

	return fmt.Errorf("op-node RPC returned unexpected response: %s", versionResult)
}

// replicateEFSMountTargets replicates mount targets (subnet + security groups) from src EFS to dst EFS
func (t *ThanosStack) replicateEFSMountTargets(ctx context.Context, region, srcFs, dstFs string) error {
	if strings.TrimSpace(srcFs) == "" || strings.TrimSpace(dstFs) == "" || srcFs == dstFs {
		return nil
	}

	// Get source mount targets
	mtJSON, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets", "--region", region, "--file-system-id", srcFs, "--output", "json")
	if err != nil {
		return fmt.Errorf("failed to describe source mount targets: %w", err)
	}
	type mtItem struct {
		MountTargetId        string `json:"MountTargetId"`
		SubnetId             string `json:"SubnetId"`
		AvailabilityZoneName string `json:"AvailabilityZoneName"`
	}
	var mtResp struct {
		MountTargets []mtItem `json:"MountTargets"`
	}
	if err := json.Unmarshal([]byte(mtJSON), &mtResp); err != nil {
		return fmt.Errorf("failed to parse mount targets: %w", err)
	}
	if len(mtResp.MountTargets) == 0 {
		t.logger.Info("No mount targets found on source EFS; skipping replication")
		return nil
	}

	// For each source mount target, fetch SGs and create on destination
	for _, mt := range mtResp.MountTargets {
		if strings.TrimSpace(mt.SubnetId) == "" || strings.TrimSpace(mt.MountTargetId) == "" {
			continue
		}
		sgOut, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups", "--region", region, "--mount-target-id", mt.MountTargetId, "--query", "SecurityGroups", "--output", "text")
		if err != nil {
			t.logger.Warnf("Failed to get SGs for %s: %v", mt.MountTargetId, err)
			continue
		}
		sgOut = strings.TrimSpace(sgOut)
		if sgOut == "" {
			t.logger.Warnf("No SGs for %s; skipping subnet %s", mt.MountTargetId, mt.SubnetId)
			continue
		}
		// create mount target; ignore errors if already exists
		args := []string{"efs", "create-mount-target", "--region", region, "--file-system-id", dstFs, "--subnet-id", mt.SubnetId, "--security-groups"}
		args = append(args, strings.Fields(sgOut)...)
		if _, err := utils.ExecuteCommand(ctx, "aws", args...); err != nil {
			t.logger.Infof("Note: create-mount-target may have failed/exists for subnet %s: %v", mt.SubnetId, err)
		} else {
			t.logger.Infof("Created mount target on subnet %s (AZ %s) for %s", mt.SubnetId, mt.AvailabilityZoneName, dstFs)
		}
	}
	return nil
}

// validateEFSMountTargets verifies that an EFS has mount targets with security groups that allow NFS (TCP 2049)
// and the referenced subnets are available. It logs detailed findings and returns an error if any critical
// validation fails. Non-critical findings are emitted as warnings.
func (t *ThanosStack) validateEFSMountTargets(ctx context.Context, region, fsId string) error {
	if strings.TrimSpace(fsId) == "" {
		return fmt.Errorf("empty file system id")
	}

	// Describe mount targets for the destination EFS
	mtJSON, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets", "--region", region, "--file-system-id", fsId, "--output", "json")
	if err != nil {
		return fmt.Errorf("failed to describe mount targets for %s: %w", fsId, err)
	}

	type mtItem struct {
		MountTargetId string `json:"MountTargetId"`
		SubnetId      string `json:"SubnetId"`
	}
	var mtResp struct {
		MountTargets []mtItem `json:"MountTargets"`
	}
	if jErr := json.Unmarshal([]byte(mtJSON), &mtResp); jErr != nil {
		return fmt.Errorf("failed to parse mount targets for %s: %w", fsId, jErr)
	}
	if len(mtResp.MountTargets) == 0 {
		return fmt.Errorf("no mount targets found on EFS %s", fsId)
	}

	criticalIssues := []string{}

	for _, mt := range mtResp.MountTargets {
		subnetId := strings.TrimSpace(mt.SubnetId)
		if subnetId == "" || strings.TrimSpace(mt.MountTargetId) == "" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("invalid mount target entry (id=%s subnet=%s)", mt.MountTargetId, mt.SubnetId))
			continue
		}

		// 1) Check subnet availability
		state, sErr := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-subnets", "--region", region, "--subnet-ids", subnetId, "--query", "Subnets[0].State", "--output", "text")
		if sErr != nil {
			t.logger.Infof("⚠️  Failed to check subnet state for %s: %v", subnetId, sErr)
		} else if strings.TrimSpace(state) != "available" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("subnet %s state is %s (expected available)", subnetId, strings.TrimSpace(state)))
		} else {
			t.logger.Infof("✅ Subnet %s is available", subnetId)
		}

		// 2) Check SGs allow TCP 2049
		sgText, gErr := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups", "--region", region, "--mount-target-id", mt.MountTargetId, "--query", "SecurityGroups", "--output", "text")
		if gErr != nil {
			criticalIssues = append(criticalIssues, fmt.Sprintf("failed to get security groups for mount target %s: %v", mt.MountTargetId, gErr))
			continue
		}
		sgText = strings.TrimSpace(sgText)
		if sgText == "" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("no security groups attached to mount target %s", mt.MountTargetId))
			continue
		}
		sgs := strings.Fields(sgText)

		sgAllowsNFS := false
		for _, sg := range sgs {
			// Query SG ingress rules for TCP 2049
			permJSON, pErr := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-security-groups", "--region", region, "--group-ids", sg, "--query", "SecurityGroups[0].IpPermissions", "--output", "json")
			if pErr != nil {
				t.logger.Infof("⚠️  Failed to read SG %s permissions: %v", sg, pErr)
				continue
			}

			var perms []struct {
				IpProtocol string `json:"IpProtocol"`
				FromPort   *int   `json:"FromPort"`
				ToPort     *int   `json:"ToPort"`
			}
			if uErr := json.Unmarshal([]byte(permJSON), &perms); uErr != nil {
				t.logger.Infof("⚠️  Failed to parse SG %s permissions: %v", sg, uErr)
				continue
			}
			for _, ip := range perms {
				if strings.ToLower(strings.TrimSpace(ip.IpProtocol)) != "tcp" {
					continue
				}
				if ip.FromPort != nil && ip.ToPort != nil && *ip.FromPort <= 2049 && 2049 <= *ip.ToPort {
					sgAllowsNFS = true
					break
				}
			}
			if sgAllowsNFS {
				t.logger.Infof("✅ SG %s allows TCP 2049", sg)
				break
			} else {
				t.logger.Infof("ℹ️  SG %s has no explicit TCP 2049 rule", sg)
			}
		}

		if !sgAllowsNFS {
			criticalIssues = append(criticalIssues, fmt.Sprintf("no attached SG on mount target %s allows TCP 2049", mt.MountTargetId))
		}
	}

	if len(criticalIssues) > 0 {
		return fmt.Errorf("mount target validation failed: %s", strings.Join(criticalIssues, "; "))
	}

	return nil
}

// backupPvPvc runs the helper script to back up PV/PVC definitions before changes
func (t *ThanosStack) backupPvPvc(ctx context.Context, namespace string) error {
	// Ensure local backup script exists; if not, download into current directory
	local := "./backup_pv_pvc.sh"
	if _, err := os.Stat(local); err != nil {
		_ = os.MkdirAll("./scripts", 0755)
		url := "https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/backup_pv_pvc.sh"
		t.logger.Infof("Downloading backup script to %s...", local)
		if _, dErr := utils.ExecuteCommand(ctx, "bash", "-lc", fmt.Sprintf("curl -fsSL %s -o %s && chmod +x %s", url, local, local)); dErr != nil {
			return fmt.Errorf("failed to download backup script: %w", dErr)
		}
	}
	cmd := fmt.Sprintf("NAMESPACE=%s BACKUP_DIR=./k8s-efs-backup bash %s", namespace, local)
	_, err := utils.ExecuteCommand(ctx, "bash", "-lc", cmd)
	return err
}

// validateAttachPrerequisites checks if required tools and cluster access are available
func (t *ThanosStack) validateAttachPrerequisites(ctx context.Context) error {
	t.logger.Info("Checking attach prerequisites...")

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

	t.logger.Info("✅ Attach prerequisites check passed")
	return nil
}

// verifyEFSData creates a temporary pod to verify EFS data accessibility and op-geth metadata
func (t *ThanosStack) verifyEFSData(ctx context.Context, namespace string) error {
	t.logger.Info("Checking EFS data...")

	// Clean up any existing verify pod
	_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Find the correct PVC name for op-geth
	pvcsList, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}

	var opGethPVC string
	for _, pvc := range strings.Fields(pvcsList) {
		if strings.Contains(pvc, "op-geth") {
			opGethPVC = pvc
			break
		}
	}

	if opGethPVC == "" {
		return fmt.Errorf("no op-geth PVC found in namespace %s", namespace)
	}

	t.logger.Infof("Using PVC: %s", opGethPVC)

	// Create verify pod with op-geth metadata checking capabilities
	podYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: verify-efs
  namespace: %s
spec:
  containers:
  - name: verify
    image: ethereum/client-go:latest
    command: ["/bin/sh", "-c"]
    args:
    - |
      # Check EFS accessibility
      ls -la /db > /dev/null || { echo "ERROR: EFS not accessible"; exit 1; }
      
      echo "🔍 Op-Geth Data Analysis"
      
      # Find the actual operational chaindata path (subPath-based)
      OPERATIONAL_CHAINDATA=""
      for subpath_dir in /db/*-op-geth; do
        if [ -d "$subpath_dir/geth/chaindata" ]; then
          OPERATIONAL_CHAINDATA="$subpath_dir/geth/chaindata"
          break
        fi
      done
      
      # Fallback to direct chaindata if no subPath found
      if [ -z "$OPERATIONAL_CHAINDATA" ] && [ -d "/db/chaindata" ]; then
        OPERATIONAL_CHAINDATA="/db/chaindata"
      fi
      
      if [ -n "$OPERATIONAL_CHAINDATA" ]; then
        echo "   ✅ Operational chaindata: $OPERATIONAL_CHAINDATA"
        chaindata_size=$(du -sh "$OPERATIONAL_CHAINDATA" 2>/dev/null | awk '{print $1}' || echo 'N/A')
        file_count=$(find "$OPERATIONAL_CHAINDATA" -type f 2>/dev/null | wc -l || echo '0')
        echo "   📊 Size: $chaindata_size, Files: $file_count"
        PRIMARY_CHAINDATA="$OPERATIONAL_CHAINDATA"
      else
        echo "   ❌ No operational chaindata found"
      fi
      
      echo "🔍 Chaindata Integrity Check:"
      
      if [ -n "$PRIMARY_CHAINDATA" ] && [ -d "$PRIMARY_CHAINDATA" ]; then
        file_count=$(find "$PRIMARY_CHAINDATA" -type f 2>/dev/null | wc -l || echo "0")
        
        if [ "$file_count" -gt 0 ]; then
          echo "✅ Contains $file_count files"
          
          # Check critical LevelDB files
          if [ -f "$PRIMARY_CHAINDATA/CURRENT" ]; then
            echo "✅ CURRENT file exists"
          else
            echo "❌ CURRENT file missing - database corrupted"
            exit 1
          fi
          
          if ls "$PRIMARY_CHAINDATA"/MANIFEST-* >/dev/null 2>&1; then
            echo "✅ MANIFEST file exists"
          else
            echo "❌ MANIFEST file missing - database corrupted"
            exit 1
          fi
          
          # Check for LOG files (indicates recent activity)
          log_files=$(find "$PRIMARY_CHAINDATA" -name "*.log" -type f 2>/dev/null | wc -l || echo "0")
          if [ "$log_files" -gt 0 ]; then
            echo "✅ Found $log_files LOG files"
          else
            echo "⚠️  No LOG files found"
          fi
          
          # Check for SST files (actual data)
          sst_files=$(find "$PRIMARY_CHAINDATA" -name "*.sst" -o -name "*.ldb" -type f 2>/dev/null | wc -l || echo "0")
          if [ "$sst_files" -gt 0 ]; then
            echo "✅ Found $sst_files data files"
          else
            echo "❌ No data files found - empty database"
            exit 1
          fi
          
          # Check for LOCK file (should not exist if geth is not running)
          if [ -f "$PRIMARY_CHAINDATA/LOCK" ]; then
            echo "⚠️  LOCK file exists - database may be in use"
          else
            echo "✅ No LOCK file - database available"
          fi
          
        else
          echo "❌ Chaindata directory is empty"
          exit 1
        fi
      else
        echo "❌ No operational chaindata found"
        exit 1
      fi
      echo ""
    volumeMounts:
    - name: efs-volume
      mountPath: /db
  volumes:
  - name: efs-volume
    persistentVolumeClaim:
      claimName: %s
  restartPolicy: Never`, namespace, opGethPVC)

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
	t.logger.Info("Waiting for verification pod to complete...")
	for i := 0; i < 90; i++ { // Increased timeout to 3 minutes for geth operations
		status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", "verify-efs", "-o", "jsonpath={.status.phase}")
		if err != nil {
			t.logger.Infof("Pod status check failed: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		status = strings.TrimSpace(status)
		if status == "Succeeded" {
			t.logger.Info("✅ EFS data verification completed successfully")

			// Get and display pod logs with metadata information
			logs, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", "verify-efs")
			if err != nil {
				t.logger.Infof("Warning: Could not retrieve verification logs: %v", err)
			} else {
				t.logger.Info(logs)
			}
			return nil
		}
		if status == "Failed" {
			// Get pod logs for analysis
			logs, _ := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", "verify-efs")
			t.logger.Infof("❌ Pod failed. Logs:\n%s", logs)
			return fmt.Errorf("verification pod failed")
		}
		if status == "Pending" {
			t.logger.Infof("Pod is pending... (attempt %d/90)", i+1)
		}
		if status == "Running" {
			t.logger.Infof("Pod is running... (attempt %d/90)", i+1)
		}
		time.Sleep(2 * time.Second)
	}

	// Clean up the pod
	utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	return fmt.Errorf("verification pod timed out after 3 minutes")
}

// updatePVVolumeHandles updates PV volume handles to point to new EFS
func (t *ThanosStack) updatePVVolumeHandles(ctx context.Context, namespace, newEfs string, pvcs *string) error {
	t.logger.Infof("Updating PV volume handles to EFS: %s", newEfs)

	var targetPVCs []string

	// Get target PVCs
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		// List all PVC names once for fuzzy matching
		allPVCsOut, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if err != nil {
			return fmt.Errorf("failed to list PVCs for matching: %w", err)
		}
		allPVCs := strings.Fields(allPVCsOut)

		for _, input := range strings.Split(strings.TrimSpace(*pvcs), ",") {
			alias := strings.TrimSpace(input)
			if alias == "" {
				continue
			}

			// Try exact match first
			resolvedPVC := ""
			for _, name := range allPVCs {
				if name == alias {
					resolvedPVC = name
					break
				}
			}

			// Fallback: fuzzy match (contains alias)
			if resolvedPVC == "" {
				for _, name := range allPVCs {
					if strings.Contains(name, alias) {
						resolvedPVC = name
						break
					}
				}
			}

			if resolvedPVC == "" {
				t.logger.Warnf("PVC alias '%s' did not match any PVC in namespace %s", alias, namespace)
				continue
			}

			// Validate resolved PVC exists
			if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", resolvedPVC); err != nil {
				t.logger.Warnf("PVC %s not found after resolution, skipping", resolvedPVC)
				continue
			}

			t.logger.Infof("Resolved PVC alias '%s' -> '%s'", alias, resolvedPVC)
			targetPVCs = append(targetPVCs, resolvedPVC)
		}
	} else {
		// If no PVCs specified, get all PVCs in namespace that match EFS patterns
		pvcsList, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if err != nil {
			return fmt.Errorf("failed to list PVCs: %w", err)
		}

		// Look for PVCs that contain op-geth or op-node in their names
		for _, pvc := range strings.Fields(pvcsList) {
			if strings.Contains(pvc, "op-geth") || strings.Contains(pvc, "op-node") {
				targetPVCs = append(targetPVCs, pvc)
				t.logger.Infof("Found PVC: %s", pvc)
			}
		}
	}

	if len(targetPVCs) == 0 {
		return fmt.Errorf("no target PVCs found to update")
	}

	// Since PV volumeHandle is immutable, we need to delete and recreate PVs
	t.logger.Infof("PV volumeHandle is immutable, deleting and recreating PVs with new EFS...")

	successCount := 0
	for _, pvcName := range targetPVCs {
		pvcName = strings.TrimSpace(pvcName)
		if pvcName == "" {
			continue
		}

		// Get current PV name from PVC
		pvName, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.spec.volumeName}")
		if err != nil {
			t.logger.Errorf("Failed to get PV name for PVC %s: %v", pvcName, err)
			continue
		}
		pvName = strings.TrimSpace(pvName)
		if pvName == "" {
			t.logger.Warnf("PVC %s has no volumeName, skipping", pvcName)
			continue
		}

		t.logger.Infof("Processing PVC: %s (PV: %s)", pvcName, pvName)

		// Delete the PVC first (this will also delete the PV if it's dynamically provisioned)
		t.logger.Infof("Deleting PVC %s...", pvcName)

		// First, delete pods that use this PVC
		podsUsingPVC, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pods", "-o", "jsonpath={range .items[?(@.spec.volumes[*].persistentVolumeClaim.claimName=='"+pvcName+"')]}{.metadata.name}{\"\\n\"}{end}")
		if err == nil && strings.TrimSpace(podsUsingPVC) != "" {
			for _, podName := range strings.Fields(podsUsingPVC) {
				t.logger.Infof("Deleting pod %s that uses PVC %s...", podName, pvcName)
				utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", podName, "--ignore-not-found=true")
			}
			// Wait for pods to be deleted
			time.Sleep(5 * time.Second)
		}

		// Delete PVC
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pvc", pvcName); err != nil {
			t.logger.Errorf("Failed to delete PVC %s: %v", pvcName, err)
			continue
		}

		// Delete the old PV
		t.logger.Infof("Deleting old PV %s...", pvName)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", pvName, "--ignore-not-found=true"); err != nil {
			t.logger.Warnf("Failed to delete PV %s: %v", pvName, err)
		}

		// Wait a moment for deletion to complete
		time.Sleep(2 * time.Second)

		// Create new PV with new EFS ID (following infrastructure pattern)
		newPVYaml := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: %s
  labels:
    app: %s
spec:
  capacity:
    storage: 500Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: efs-sc
  csi:
    driver: efs.csi.aws.com
    volumeHandle: %s`, pvName, pvName, newEfs)

		tempPVFile := fmt.Sprintf("/tmp/new-pv-%s.yaml", pvName)
		if err := os.WriteFile(tempPVFile, []byte(newPVYaml), 0644); err != nil {
			t.logger.Errorf("Failed to create temporary PV YAML file: %v", err)
			continue
		}
		defer os.Remove(tempPVFile)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVFile); err != nil {
			t.logger.Errorf("Failed to create new PV %s: %v", pvName, err)
			continue
		}

		// Create new PVC following infrastructure pattern (with selector and storage class)
		newPVCYaml := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  storageClassName: efs-sc
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 500Gi
  selector:
    matchLabels:
      app: %s
  volumeMode: Filesystem
  volumeName: %s`, pvcName, namespace, pvName, pvName)

		tempPVCFile := fmt.Sprintf("/tmp/new-pvc-%s.yaml", pvcName)
		if err := os.WriteFile(tempPVCFile, []byte(newPVCYaml), 0644); err != nil {
			t.logger.Errorf("Failed to create temporary PVC YAML file: %v", err)
			continue
		}
		defer os.Remove(tempPVCFile)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVCFile); err != nil {
			t.logger.Errorf("Failed to create new PVC %s: %v", pvcName, err)
			continue
		}

		// Wait for PVC to be bound
		t.logger.Infof("Waiting for PVC %s to be bound...", pvcName)
		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.status.phase}")
			if err == nil && strings.TrimSpace(status) == "Bound" {
				break
			}
			time.Sleep(1 * time.Second)
		}

		t.logger.Infof("✅ PVC %s and PV %s recreated successfully with new EFS", pvcName, pvName)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to recreate any PVCs/PVs")
	}

	t.logger.Infof("✅ Successfully recreated %d/%d PVCs with new EFS", successCount, len(targetPVCs))
	return nil
}

// restartStatefulSets restarts specified StatefulSets
func (t *ThanosStack) restartStatefulSets(ctx context.Context, namespace, stss string) error {
	t.logger.Infof("Restarting StatefulSets: %s", stss)

	successCount := 0

	// Fetch all StatefulSet names in namespace for alias resolution
	allSTSOut, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "statefulset", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		return fmt.Errorf("failed to list StatefulSets: %w", err)
	}
	allSTS := strings.Fields(allSTSOut)

	for _, sts := range strings.Split(stss, ",") {
		sts = strings.TrimSpace(sts)
		if sts == "" {
			continue
		}

		// Resolve alias to actual StatefulSet name
		resolved := ""
		// exact match first
		for _, name := range allSTS {
			if name == sts {
				resolved = name
				break
			}
		}
		// fallback contains match
		if resolved == "" {
			for _, name := range allSTS {
				if strings.Contains(name, sts) {
					resolved = name
					break
				}
			}
		}

		if resolved == "" {
			t.logger.Warnf("StatefulSet alias '%s' not found in namespace %s, skipping", sts, namespace)
			continue
		}

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "rollout", "restart", fmt.Sprintf("statefulset/%s", resolved)); err != nil {
			t.logger.Errorf("Failed to restart StatefulSet %s: %v", resolved, err)
		} else {
			t.logger.Infof("✅ StatefulSet %s restarted successfully", resolved)
			successCount++
		}
	}

	if successCount == 0 {
		return fmt.Errorf("failed to restart any StatefulSets")
	}

	t.logger.Infof("✅ Successfully restarted %d StatefulSets", successCount)
	return nil
}

// performHealthCheck performs health check on the cluster
func (t *ThanosStack) performHealthCheck(ctx context.Context, namespace string) error {
	t.logger.Info("Performing health check...")

	// 1) Check op-geth and op-node restarted
	labels := map[string]string{
		"op-geth": "app.kubernetes.io/name=thanos-stack-op-geth",
		"op-node": "app.kubernetes.io/name=thanos-stack-op-node",
	}

	restarted := map[string]bool{"op-geth": false, "op-node": false}
	for comp, label := range labels {
		pods, err := utils.GetPodNamesByLabel(ctx, namespace, label)
		if err != nil || len(pods) == 0 {
			return fmt.Errorf("%s pods not found: %w", comp, err)
		}
		pod := pods[0]
		restCounts, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", pod, "-o", "jsonpath={.status.containerStatuses[*].restartCount}")
		if err == nil && strings.TrimSpace(restCounts) != "" {
			for _, v := range strings.Fields(strings.TrimSpace(restCounts)) {
				if v != "0" {
					restarted[comp] = true
					break
				}
			}
		}
		if restarted[comp] {
			t.logger.Infof("✅ %s restarted (pod: %s)", comp, pod)
		} else {
			t.logger.Infof("ℹ️ %s shows no container restarts (pod: %s)", comp, pod)
		}
	}

	if !restarted["op-geth"] || !restarted["op-node"] {
		t.logger.Info("Warning: One or more components did not report a container restart.")
	}

	// 2) Fetch and print L1 and L2 block numbers
	// L2 RPC: try ingress that contains namespace (same heuristic as ShowInformation)
	l2URL := ""
	if ing, err := utils.GetIngresses(ctx, namespace); err == nil {
		for name, addrs := range ing {
			if strings.Contains(name, namespace) && len(addrs) > 0 {
				l2URL = fmt.Sprintf("http://%s", addrs[0])
				break
			}
		}
	}

	// Fallback: try service address discovery for op-geth rpc service
	if l2URL == "" {
		if svcs, err := utils.GetServiceNames(ctx, namespace, "op-geth"); err == nil && len(svcs) > 0 {
			// Typical service port for geth RPC is 8545
			l2URL = fmt.Sprintf("http://%s.%s.svc.cluster.local:8545", svcs[0], namespace)
		}
	}

	// L1 RPC: read from op-node env var OP_NODE__L1_ETH_RPC
	l1URL := ""
	if pods, err := utils.GetPodNamesByLabel(ctx, namespace, labels["op-node"]); err == nil && len(pods) > 0 {
		pod := pods[0]
		val, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", pod, "-o", "jsonpath={.spec.containers[0].env[?(@.name=='OP_NODE__L1_ETH_RPC')].value}")
		if err == nil {
			l1URL = strings.TrimSpace(val)
		}
	}

	// Query JSON-RPC eth_blockNumber
	queryBlock := func(url string) (string, error) {
		if strings.TrimSpace(url) == "" {
			return "", fmt.Errorf("empty rpc url")
		}
		payload := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`
		out, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("curl -s -X POST -H 'Content-Type: application/json' --data '%s' %s | jq -r .result", payload, url))
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(out), nil
	}

	l2Block, l2Err := queryBlock(l2URL)
	if l2Err != nil {
		t.logger.Infof("❌ Failed to fetch L2 block: %v", l2Err)
	} else {
		t.logger.Infof("L2 latest block: %s (via %s)", l2Block, l2URL)
	}

	l1Block, l1Err := queryBlock(l1URL)
	if l1Err != nil {
		t.logger.Infof("❌ Failed to fetch L1 block: %v", l1Err)
	} else {
		t.logger.Infof("L1 latest block (on-chain): %s (via %s)", l1Block, l1URL)
	}

	// Additionally, fetch L1 block height as seen by op-node (synced view)
	opNodeSynced := ""
	if svcs, err := utils.GetServiceNames(ctx, namespace, "op-node"); err == nil && len(svcs) > 0 {
		metricsURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:7300/metrics", svcs[0], namespace)
		val, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("curl -s %s | grep -m1 '^op_node_l1_head' | awk '{print $NF}'", metricsURL))
		if err == nil && strings.TrimSpace(val) != "" {
			opNodeSynced = strings.TrimSpace(val)
			t.logger.Infof("L1 latest block (op-node synced): %s (via %s)", opNodeSynced, metricsURL)
		} else {
			// Fallback: try rollup_getInfo on op-node RPC (default 8547)
			opNodeRPC := fmt.Sprintf("http://%s.%s.svc.cluster.local:8547", svcs[0], namespace)
			payload := `{"jsonrpc":"2.0","method":"rollup_getInfo","params":[],"id":1}`
			res, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("curl -s -X POST -H 'Content-Type: application/json' --data '%s' %s", payload, opNodeRPC))
			if err == nil && strings.TrimSpace(res) != "" {
				// Try to extract a few common paths for the L1 head number
				tryPaths := []string{
					".result.l1.head.number",
					".result.l1.head.current.number",
				}
				for _, p := range tryPaths {
					v, jqErr := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf(`echo %s | jq -r '%s'`, strings.ReplaceAll(strings.TrimSpace(res), "'", "\\'"), p))
					if jqErr == nil && strings.TrimSpace(v) != "" && strings.TrimSpace(v) != "null" {
						opNodeSynced = strings.TrimSpace(v)
						t.logger.Infof("L1 latest block (op-node synced): %s (via %s rollup_getInfo)", opNodeSynced, opNodeRPC)
						break
					}
				}
				if opNodeSynced == "" {
					t.logger.Infof("ℹ️ Could not parse op-node synced L1 block from rollup_getInfo")
				}
			} else {
				t.logger.Infof("ℹ️ Could not reach op-node metrics or RPC for synced L1 block")
			}
		}
	}

	if l2Err != nil {
		return fmt.Errorf("health check failed: L2 block unavailable")
	}
	// L1 is optional; do not fail if unavailable
	return nil
}

// printAttachSummary prints a summary of the attach operation
func (t *ThanosStack) printAttachSummary(ctx context.Context, newEfs string, pvcs *string, stss *string) {
	t.logger.Info("\n✅ Attach operation completed:")
	if newEfs != "" {
		t.logger.Infof("  EFS FileSystemId: %s", newEfs)
	}
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		t.logger.Infof("  PVCs: %s", strings.TrimSpace(*pvcs))
	}
	if stss != nil && strings.TrimSpace(*stss) != "" {
		t.logger.Infof("  StatefulSets: %s", strings.TrimSpace(*stss))
	}
	t.logger.Info("  Health Check: ✅ Passed")

	// Post-attach synchronization notice
	if newEfs != "" {
		namespace := t.deployConfig.K8s.Namespace

		// Get actual pod and service names dynamically
		opGethPodName, _ := t.findComponentPod(ctx, namespace, "op-geth")
		opNodePodName, _ := t.findComponentPod(ctx, namespace, "op-node")

		// Construct service name based on namespace pattern
		servicePrefix := fmt.Sprintf("%s-thanos-stack", namespace)
		opGethServiceName := fmt.Sprintf("%s-op-geth", servicePrefix)

		t.logger.Info("")
		t.logger.Info("📋 Important Notice:")
		t.logger.Info("  Geth client synchronization may take some time after recovery.")
		t.logger.Info("  Block explorer and additional services will operate normally")
		t.logger.Info("  once synchronization is complete.")
		t.logger.Info("")
		t.logger.Info("🔍 Monitor Synchronization Status:")
		if opGethPodName != "" {
			t.logger.Infof("  kubectl logs -f %s -n %s", opGethPodName, namespace)
		} else {
			t.logger.Infof("  kubectl logs -f -l app.kubernetes.io/name=op-geth -n %s", namespace)
		}
		if opNodePodName != "" {
			t.logger.Infof("  kubectl logs -f %s -n %s", opNodePodName, namespace)
		} else {
			t.logger.Infof("  kubectl logs -f -l app.kubernetes.io/name=op-node -n %s", namespace)
		}
		t.logger.Info("")
		t.logger.Info("📊 Check Latest Block Number:")
		t.logger.Infof("  kubectl port-forward svc/%s 8545:8545 -n %s", opGethServiceName, namespace)
		t.logger.Info("  curl -X POST -H \"Content-Type: application/json\" \\")
		t.logger.Info("    --data '{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}' \\")
		t.logger.Info("    http://localhost:8545")
		t.logger.Info("")
		t.logger.Info("⏱️  Expected sync time: 15-30 minutes (varies by restored data state)")
		t.logger.Info("⚠️  'header not found' warnings during sync are normal behavior.")
	}
}

// updateBackupProtectedResources updates AWS Backup protected resources to use the new EFS
func (t *ThanosStack) updateBackupProtectedResources(ctx context.Context, newEfsId string) error {
	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	newEfsArn := utils.BuildEFSArn(region, accountID, newEfsId)

	t.logger.Infof("Updating AWS Backup protected resources for new EFS: %s", newEfsId)

	// Check if the new EFS is already protected
	protectedCheck, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", newEfsArn),
		"--output", "text")
	if err != nil {
		return fmt.Errorf("failed to check if new EFS is protected: %w", err)
	}

	isAlreadyProtected := strings.TrimSpace(protectedCheck) == "1"

	if isAlreadyProtected {
		t.logger.Infof("New EFS %s is already protected by AWS Backup", newEfsId)
		return nil
	}

	// Get backup vault and IAM role information
	backupVaultName := fmt.Sprintf("%s-backup-vault", namespace)
	iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-service-role", accountID, namespace)

	// Start an initial backup job for the new EFS to register it as a protected resource
	t.logger.Info("Starting initial backup job to register new EFS as protected resource...")

	backupJobOutput, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--region", region,
		"--backup-vault-name", backupVaultName,
		"--resource-arn", newEfsArn,
		"--iam-role-arn", iamRoleArn)

	if err != nil {
		return fmt.Errorf("failed to start backup job for new EFS: %w", err)
	}

	jobId := strings.TrimSpace(backupJobOutput)
	t.logger.Infof("Initial backup job started for new EFS: %s", jobId)

	// Wait a moment for the backup job to register the resource
	time.Sleep(5 * time.Second)

	// Verify the new EFS is now protected
	verifyCheck, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", newEfsArn),
		"--output", "text")
	if err != nil {
		return fmt.Errorf("failed to verify new EFS protection status: %w", err)
	}

	if strings.TrimSpace(verifyCheck) == "1" {
		t.logger.Infof("✅ New EFS %s is now protected by AWS Backup", newEfsId)

		// Optionally clean up old protected resources
		if err := t.cleanupOldProtectedResources(ctx, region, accountID, namespace, newEfsArn); err != nil {
			t.logger.Warnf("Failed to cleanup old protected resources: %v", err)
		}

		return nil
	} else {
		return fmt.Errorf("new EFS %s was not registered as protected resource", newEfsId)
	}
}

// cleanupOldProtectedResources removes old EFS from protected resources if they're no longer in use
func (t *ThanosStack) cleanupOldProtectedResources(ctx context.Context, region, accountID, namespace, currentEfsArn string) error {
	// List all protected EFS resources
	protectedResources, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", "Results[?ResourceType=='EFS'].ResourceArn",
		"--output", "text")
	if err != nil {
		return fmt.Errorf("failed to list protected resources: %w", err)
	}

	protectedList := strings.Fields(strings.TrimSpace(protectedResources))

	for _, arn := range protectedList {
		if arn != currentEfsArn && strings.Contains(arn, accountID) {
			// Extract EFS ID from ARN
			parts := strings.Split(arn, "/")
			if len(parts) > 0 {
				oldEfsId := parts[len(parts)-1]

				// Check if this old EFS still exists and is not in use by any PV
				if exists, err := t.checkEFSExists(ctx, region, oldEfsId); err == nil && !exists {
					t.logger.Infof("Old EFS %s no longer exists, will be automatically removed from protected resources", oldEfsId)
				} else {
					t.logger.Infof("Old EFS %s still exists but is no longer used by this deployment", oldEfsId)
				}
			}
		}
	}

	return nil
}

// checkEFSExists verifies if an EFS still exists
func (t *ThanosStack) checkEFSExists(ctx context.Context, region, efsId string) (bool, error) {
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--file-system-id", efsId,
		"--query", "FileSystems[0].FileSystemId",
		"--output", "text")
	if err != nil {
		// If EFS doesn't exist, AWS CLI returns an error
		return false, nil
	}
	return true, nil
}

// showAttachUsage displays usage information for backup attach command
func (t *ThanosStack) showAttachUsage() {
	t.logger.Info("📋 Backup Attach Usage")
	t.logger.Info("")
	t.logger.Info("DESCRIPTION:")
	t.logger.Info("  Attach a new or existing EFS to workloads and verify readiness.")
	t.logger.Info("  This command is typically used after EFS restore operations.")
	t.logger.Info("")
	t.logger.Info("USAGE:")
	t.logger.Info("  ./trh-sdk backup-manager --attach [OPTIONS]")
	t.logger.Info("")
	t.logger.Info("OPTIONS:")
	t.logger.Info("  --efs-id <EFS_ID>     EFS filesystem ID to attach (e.g., fs-0123456789abcdef0)")
	t.logger.Info("  --pvc <PVC_LIST>      Comma-separated list of PVCs to update (e.g., op-geth,op-node)")
	t.logger.Info("  --sts <STS_LIST>      Comma-separated list of StatefulSets to restart (e.g., op-geth,op-node)")
	t.logger.Info("")
	t.logger.Info("EXAMPLES:")
	t.logger.Info("  # Attach new EFS and restart workloads")
	t.logger.Info("  ./trh-sdk backup-manager --attach \\")
	t.logger.Info("    --efs-id fs-0123456789abcdef0 \\")
	t.logger.Info("    --pvc op-geth,op-node \\")
	t.logger.Info("    --sts op-geth,op-node")
}

// BackupConfigure applies EFS backup configuration via Terraform
func (t *ThanosStack) BackupConfigure(ctx context.Context, daily *string, keep *string, reset *bool) error {
	// Check if no options provided - show usage
	if (daily == nil || strings.TrimSpace(*daily) == "") &&
		(keep == nil || strings.TrimSpace(*keep) == "") &&
		(reset == nil || !*reset) {
		t.showBackupConfigUsage()
		return nil
	}

	tfRoot := fmt.Sprintf("%s/tokamak-thanos-stack/terraform", t.deploymentPath)

	// Verify terraform directory exists
	thanosStackPath := fmt.Sprintf("%s/thanos-stack", tfRoot)
	if _, err := os.Stat(thanosStackPath); os.IsNotExist(err) {
		return fmt.Errorf("terraform thanos-stack directory not found: %s", thanosStackPath)
	}

	// Verify .envrc file exists
	envrcPath := fmt.Sprintf("%s/.envrc", tfRoot)
	if _, err := os.Stat(envrcPath); os.IsNotExist(err) {
		return fmt.Errorf("terraform .envrc file not found: %s", envrcPath)
	}

	varArgs := []string{"-auto-approve"}
	if reset != nil && *reset {
		t.logger.Info("Resetting backup configuration to default values...")
		varArgs = append(varArgs, `-var=backup_schedule_cron="cron(0 3 * * ? *)"`, "-var=backup_delete_after_days=35")
	} else {
		if daily != nil && strings.TrimSpace(*daily) != "" {
			// convert HH:MM -> cron(M H * * ? *) using provided hour and minute
			hhmm := strings.TrimSpace(*daily)
			parts := strings.Split(hhmm, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid time format: %s (expected HH:MM)", hhmm)
			}
			cron := fmt.Sprintf("cron(%s %s * * ? *)", parts[1], parts[0])
			varArgs = append(varArgs, fmt.Sprintf(`-var=backup_schedule_cron="%s"`, cron))
			t.logger.Infof("Setting backup schedule to: %s UTC", hhmm)
		}
		if keep != nil && strings.TrimSpace(*keep) != "" {
			varArgs = append(varArgs, fmt.Sprintf("-var=backup_delete_after_days=%s", strings.TrimSpace(*keep)))
			t.logger.Infof("Setting backup retention to: %s days", strings.TrimSpace(*keep))
		}
	}

	t.logger.Info("Applying EFS backup configuration...")

	// Change to terraform directory and execute commands
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Change to terraform root directory
	if err := os.Chdir(tfRoot); err != nil {
		return fmt.Errorf("failed to change to terraform directory %s: %w", tfRoot, err)
	}
	defer os.Chdir(originalDir) // Restore original directory

	// Allow direnv
	if _, err := utils.ExecuteCommand(ctx, "direnv", "allow"); err != nil {
		t.logger.Warnf("Failed to run direnv allow: %v", err)
	}

	// Source .envrc and change to thanos-stack directory
	if err := os.Chdir("thanos-stack"); err != nil {
		return fmt.Errorf("failed to change to thanos-stack directory: %w", err)
	}

	// Initialize terraform
	t.logger.Info("Initializing terraform...")
	t.logger.Infof("Current directory: %s", func() string { dir, _ := os.Getwd(); return dir }())
	if output, err := utils.ExecuteCommand(ctx, "bash", "-c", "source ../.envrc && terraform init"); err != nil {
		t.logger.Errorf("Terraform init failed: %v", err)
		t.logger.Errorf("Output: %s", output)
		return fmt.Errorf("terraform init failed: %w", err)
	}

	// Plan terraform changes (remove -auto-approve from plan args)
	planArgs := make([]string, 0)
	for _, arg := range varArgs {
		if arg != "-auto-approve" {
			planArgs = append(planArgs, arg)
		}
	}
	planCmd := fmt.Sprintf(`source ../.envrc && terraform plan %s`, strings.Join(planArgs, " "))
	t.logger.Info("Planning terraform changes...")
	if output, err := utils.ExecuteCommand(ctx, "bash", "-c", planCmd); err != nil {
		t.logger.Errorf("Terraform plan failed: %v", err)
		t.logger.Errorf("Output: %s", output)
		return fmt.Errorf("terraform plan failed: %w", err)
	}

	// Apply terraform changes
	applyCmd := fmt.Sprintf(`source ../.envrc && terraform apply %s`, strings.Join(varArgs, " "))
	t.logger.Info("Applying terraform changes...")
	if output, err := utils.ExecuteCommand(ctx, "bash", "-c", applyCmd); err != nil {
		t.logger.Errorf("Terraform apply failed: %v", err)
		t.logger.Errorf("Output: %s", output)
		return fmt.Errorf("terraform apply failed: %w", err)
	}

	// Show applied configuration
	t.logger.Info("✅ EFS backup configuration updated successfully")
	t.logger.Info("")
	t.logger.Info("📋 Applied Configuration:")

	// Extract and display applied values
	var appliedSchedule, appliedRetention string

	if reset != nil && *reset {
		appliedSchedule = "03:00 UTC"
		appliedRetention = "35 days"
	} else {
		// Extract from varArgs
		for _, arg := range varArgs {
			if strings.HasPrefix(arg, `-var=backup_schedule_cron=`) {
				cronValue := strings.TrimPrefix(arg, `-var=backup_schedule_cron="`)
				cronValue = strings.TrimSuffix(cronValue, `"`)
				// Extract minute and hour from cron expression: cron(M H * * ? *)
				if strings.HasPrefix(cronValue, "cron(") {
					cronParts := strings.Split(cronValue, " ")
					if len(cronParts) >= 3 {
						minute := strings.TrimPrefix(cronParts[0], "cron(")
						hour := cronParts[1]

						// Format hour and minute with leading zeros
						if len(hour) == 1 {
							hour = "0" + hour
						}
						if len(minute) == 1 {
							minute = "0" + minute
						}
						appliedSchedule = hour + ":" + minute + " UTC"
					}
				}
			}
			if strings.HasPrefix(arg, `-var=backup_delete_after_days=`) {
				appliedRetention = strings.TrimPrefix(arg, `-var=backup_delete_after_days=`) + " days"
			}
		}
	}

	if appliedSchedule != "" {
		t.logger.Infof("  Daily backup time: %s", appliedSchedule)
	}
	if appliedRetention != "" {
		t.logger.Infof("  Retention period:  %s", appliedRetention)
	}

	return nil
}

// showBackupConfigUsage displays usage information for backup configuration
func (t *ThanosStack) showBackupConfigUsage() {
	t.logger.Info("📋 Backup Configuration Usage")
	t.logger.Info("")
	t.logger.Info("COMMAND:")
	t.logger.Info("  trh-sdk backup-manager --config [OPTIONS]")
	t.logger.Info("")
	t.logger.Info("OPTIONS:")
	t.logger.Info("  --daily <HH:MM>   Set daily backup time (24-hour format, UTC)")
	t.logger.Info("  --keep <DAYS>     Set backup retention period in days")
	t.logger.Info("  --reset           Reset to default configuration")
	t.logger.Info("")
	t.logger.Info("EXAMPLES:")
	t.logger.Info("  # Set backup time to 02:30 UTC")
	t.logger.Info("  trh-sdk backup-manager --config --daily \"02:30\"")
	t.logger.Info("")
	t.logger.Info("  # Set retention period to 60 days")
	t.logger.Info("  trh-sdk backup-manager --config --keep \"60\"")
	t.logger.Info("")
	t.logger.Info("  # Set both backup time and retention")
	t.logger.Info("  trh-sdk backup-manager --config --daily \"01:00\" --keep \"30\"")
	t.logger.Info("")
	t.logger.Info("  # Reset to default values (03:00 UTC, 35 days)")
	t.logger.Info("  trh-sdk backup-manager --config --reset")
	t.logger.Info("")
	t.logger.Info("CURRENT DEFAULT VALUES:")
	t.logger.Info("  Daily backup time: 03:00 UTC")
	t.logger.Info("  Retention period:  35 days")
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
	t.logger.Info("Waiting for EFS to be available...")
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
	t.logger.Info("Starting initial backup job...")
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

	t.logger.Infof("✅ Initial backup job started for EFS: %s", efsID)
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

	// Get backup vault name from namespace
	vaultName := fmt.Sprintf("%s-backup-vault", namespace)

	t.logger.Infof("\nAvailable EFS Recovery Points from Backup Vault: %s", vaultName)
	t.logger.Info("   -------------------------------------------------")

	// Get recent recovery points from backup vault (max 5, EFS only)
	query := "reverse(sort_by(RecoveryPoints[?ResourceType=='EFS'],&CreationDate))[:5].{RecoveryPointArn:RecoveryPointArn,ResourceArn:ResourceArn,BackupVaultName:BackupVaultName,CreationDate:CreationDate,Status:Status,Lifecycle:Lifecycle}"
	jsonOut, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-backup-vault",
		"--region", region,
		"--backup-vault-name", vaultName,
		"--query", query,
		"--output", "json")

	if err != nil {
		return "", fmt.Errorf("failed to retrieve recovery points: %w", err)
	}

	jsonOutTrimmed := strings.TrimSpace(jsonOut)
	if jsonOutTrimmed == "" || jsonOutTrimmed == "[]" {
		return "", fmt.Errorf("no recovery points found")
	}

	// Parse JSON to extract recovery points
	var recoveryPoints []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonOutTrimmed), &recoveryPoints); err != nil {
		return "", fmt.Errorf("failed to parse recovery points: %w", err)
	}

	if len(recoveryPoints) == 0 {
		return "", fmt.Errorf("no recovery points found")
	}

	// Display recovery points in a simple list format
	t.logger.Info("   Available Recovery Points:")
	t.logger.Info("   " + strings.Repeat("─", 80))

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
				}
			}
		}

		// Determine availability status
		availability := "✅ Available"
		if status != "COMPLETED" {
			availability = "❌ Not Available"
		}

		// Extract EFS ID from ResourceArn
		efsID := "Unknown"
		if resourceArn, ok := point["ResourceArn"].(string); ok {
			if strings.Contains(resourceArn, "file-system/") {
				parts := strings.Split(resourceArn, "file-system/")
				if len(parts) > 1 {
					efsID = parts[1]
				}
			}
		}

		// Display in a clean list format
		t.logger.Infof("   %d. %s", i+1, displayDate)
		t.logger.Infof("      📁 Vault: %s", vaultName)
		t.logger.Infof("      🗂️  EFS ID: %s", efsID)
		t.logger.Infof("      📊 Status: %s", availability)
		t.logger.Infof("      🔗 ARN: %s", recoveryPointArn)
		t.logger.Info()

		options = append(options, recoveryPointArn)
	}

	t.logger.Info("   " + strings.Repeat("─", 80))

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
	t.logger.Infof("\n   ✅ Selected recovery point: %s", selectedArn)

	return selectedArn, nil
}

// interactiveRestore provides a fully interactive restore experience for EFS and RDS
func (t *ThanosStack) interactiveRestore(ctx context.Context) error {
	t.logger.Info("\n🔄 Starting Interactive Restore")
	t.logger.Info("================================")

	// EFS-only flow
	return t.interactiveEFSRestore(ctx)
}

// interactiveEFSRestore handles EFS-only restore
func (t *ThanosStack) interactiveEFSRestore(ctx context.Context) error {
	t.logger.Info("\n📁 EFS Restore")
	t.logger.Info("==============")

	// Select recovery point
	t.logger.Info("\n📋 Select Recovery Point")
	selectedPoint, err := t.selectRecoveryPoint(ctx)
	if err != nil {
		return fmt.Errorf("failed to select recovery point: %w", err)
	}

	// Execute restore
	t.logger.Info("\n🚀 Starting EFS Restore...")
	newEfsID, err := t.restoreEFS(ctx, selectedPoint)
	if err != nil {
		return fmt.Errorf("EFS restore failed: %w", err)
	}

	t.logger.Info("✅ EFS restore completed successfully!")

	if newEfsID != "" {
		t.logger.Infof("   New EFS ID: %s", newEfsID)
		t.logger.Info("   ")
		t.logger.Info("   Next steps:")
		t.logger.Info("   1. The new EFS needs to be attached to your workloads")
		t.logger.Info("   2. Run the following command to attach the new EFS:")
		t.logger.Infof("      ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
		t.logger.Info("   ")
		fmt.Print("   Would you like to proceed with attach now? (y/n): ")

		var attachChoice string
		fmt.Scanf("%s", &attachChoice)
		attachChoice = strings.ToLower(strings.TrimSpace(attachChoice))

		if attachChoice == "y" || attachChoice == "yes" {
			t.logger.Info("\n🔗 Starting Attach Process...")

			efsID := &newEfsID
			pvcs := "op-geth,op-node"
			sts := "op-geth,op-node"

			if err := t.BackupAttach(ctx, efsID, &pvcs, &sts); err != nil {
				t.logger.Infof("   ❌ Attach failed: %v", err)
				t.logger.Infof("   You can try attaching manually later using: ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
			}
		} else {
			t.logger.Infof("   ℹ️  You can attach the EFS later using: ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
		}
	} else {
		t.logger.Info("   ⚠️  No new EFS ID detected from restore")
		t.logger.Info("   You may need to manually attach the restored EFS")
	}
	return nil
}

// CleanupUnusedBackupResources removes unused EFS filesystems and old recovery points during deploy
func (t *ThanosStack) CleanupUnusedBackupResources(ctx context.Context) error {
	t.logger.Info("🧹 Cleaning up unused backup resources...")

	region := t.deployConfig.AWS.Region
	namespace := t.deployConfig.K8s.Namespace

	// Get account ID
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	// 1. Cleanup old recovery points
	if err := t.cleanupOldRecoveryPoints(ctx, region, namespace, accountID); err != nil {
		t.logger.Warnf("Failed to cleanup old recovery points: %v", err)
	}

	// 2. Cleanup unused EFS filesystems
	if err := t.cleanupUnusedEFS(ctx, region, namespace); err != nil {
		t.logger.Warnf("Failed to cleanup unused EFS: %v", err)
	}

	// 3. Cleanup backup vaults with namespace prefix
	if err := t.cleanupBackupVaults(ctx, region, namespace); err != nil {
		t.logger.Warnf("Failed to cleanup backup vaults: %v", err)
	}

	t.logger.Info("✅ Backup resources cleanup completed")
	return nil
}

// cleanupOldRecoveryPoints removes recovery points older than 7 days
func (t *ThanosStack) cleanupOldRecoveryPoints(ctx context.Context, region, namespace, accountID string) error {
	t.logger.Info("🗑️ Cleaning up old recovery points...")

	// Get current EFS ID to find its recovery points
	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		t.logger.Warn("Could not detect current EFS ID, skipping recovery point cleanup")
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
		t.logger.Info("No old recovery points found to cleanup")
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
		t.logger.Infof("Deleting old recovery point: %s (created: %s)", rp.Arn, rp.Created)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
			"--region", region,
			"--backup-vault-name", fmt.Sprintf("%s-backup-vault", namespace),
			"--recovery-point-arn", rp.Arn)

		if err != nil {
			t.logger.Warnf("Failed to delete recovery point %s: %v", rp.Arn, err)
		} else {
			deletedCount++
			t.logger.Infof("✅ Deleted recovery point: %s", rp.Arn)
		}
	}

	if deletedCount > 0 {
		t.logger.Infof("✅ Deleted %d old recovery points", deletedCount)
	}

	return nil
}

// cleanupUnusedEFS removes EFS filesystems that are not currently in use
func (t *ThanosStack) cleanupUnusedEFS(ctx context.Context, region, namespace string) error {
	t.logger.Info("🗑️ Cleaning up unused EFS filesystems...")

	// Get current EFS ID in use
	currentEfsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil {
		t.logger.Warn("Could not detect current EFS ID, skipping EFS cleanup")
		return nil
	}

	// List all EFS filesystems with our naming pattern
	efsPattern := fmt.Sprintf("%s-*", namespace)

	efsList, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", region,
		"--query", "FileSystems[?starts_with(Name, `"+efsPattern+"`) && FileSystemId != `"+currentEfsID+"`].{Id:FileSystemId,Name:Name,State:LifeCycleState}",
		"--output", "json")

	if err != nil {
		return fmt.Errorf("failed to list EFS filesystems: %w", err)
	}

	if strings.TrimSpace(efsList) == "[]" || strings.TrimSpace(efsList) == "" {
		t.logger.Info("No unused EFS filesystems found to cleanup")
		return nil
	}

	// Parse EFS list
	var filesystems []struct {
		Id    string `json:"Id"`
		Name  string `json:"Name"`
		State string `json:"State"`
	}

	if err := json.Unmarshal([]byte(efsList), &filesystems); err != nil {
		return fmt.Errorf("failed to parse EFS list: %w", err)
	}

	deletedCount := 0
	for _, efs := range filesystems {
		if efs.State != "available" {
			t.logger.Warnf("Skipping EFS %s (state: %s)", efs.Id, efs.State)
			continue
		}

		t.logger.Infof("Deleting unused EFS: %s (%s)", efs.Id, efs.Name)

		// Delete mount targets first
		if err := t.deleteEFSMountTargets(ctx, region, efs.Id); err != nil {
			t.logger.Warnf("Failed to delete mount targets for %s: %v", efs.Id, err)
			continue
		}

		// Wait for mount targets to be deleted
		time.Sleep(10 * time.Second)

		// Delete the EFS filesystem
		_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
			"--region", region,
			"--file-system-id", efs.Id)

		if err != nil {
			t.logger.Warnf("Failed to delete EFS %s: %v", efs.Id, err)
		} else {
			deletedCount++
			t.logger.Infof("✅ Deleted EFS filesystem: %s", efs.Id)
		}
	}

	if deletedCount > 0 {
		t.logger.Infof("✅ Deleted %d unused EFS filesystems", deletedCount)
	}

	return nil
}

// deleteEFSMountTargets removes all mount targets for an EFS filesystem
func (t *ThanosStack) deleteEFSMountTargets(ctx context.Context, region, efsId string) error {
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

		t.logger.Infof("Deleting mount target: %s", mtId)

		_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
			"--region", region,
			"--mount-target-id", mtId)

		if err != nil {
			t.logger.Warnf("Failed to delete mount target %s: %v", mtId, err)
		}
	}

	return nil
}

// cleanupBackupVaults removes backup vaults with namespace prefix
func (t *ThanosStack) cleanupBackupVaults(ctx context.Context, region, namespace string) error {
	t.logger.Info("🗑️ Cleaning up backup vaults...")

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
		t.logger.Infof("No backup vaults found with prefix: %s-*", namespace)
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
		t.logger.Infof("Processing backup vault: %s", vault.BackupVaultName)

		// First, delete all recovery points in the vault
		if err := t.deleteAllRecoveryPointsInVault(ctx, region, vault.BackupVaultName); err != nil {
			t.logger.Warnf("Failed to delete recovery points in vault %s: %v", vault.BackupVaultName, err)
			continue
		}

		// Wait a bit for recovery points to be fully deleted
		time.Sleep(5 * time.Second)

		// Delete the backup vault
		t.logger.Infof("Deleting backup vault: %s", vault.BackupVaultName)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-backup-vault",
			"--region", region,
			"--backup-vault-name", vault.BackupVaultName)

		if err != nil {
			t.logger.Warnf("Failed to delete backup vault %s: %v", vault.BackupVaultName, err)
		} else {
			deletedCount++
			t.logger.Infof("✅ Deleted backup vault: %s", vault.BackupVaultName)
		}
	}

	if deletedCount > 0 {
		t.logger.Infof("✅ Deleted %d backup vaults", deletedCount)
	}

	return nil
}

// deleteAllRecoveryPointsInVault removes all recovery points from a backup vault
func (t *ThanosStack) deleteAllRecoveryPointsInVault(ctx context.Context, region, vaultName string) error {
	t.logger.Infof("Deleting all recovery points in vault: %s", vaultName)

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
		t.logger.Infof("No recovery points found in vault: %s", vaultName)
		return nil
	}

	// Parse recovery points list
	var recoveryPointArns []string
	if err := json.Unmarshal([]byte(rpList), &recoveryPointArns); err != nil {
		return fmt.Errorf("failed to parse recovery points list for vault %s: %w", vaultName, err)
	}

	deletedCount := 0
	for _, rpArn := range recoveryPointArns {
		t.logger.Infof("Deleting recovery point: %s", rpArn)

		_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
			"--region", region,
			"--backup-vault-name", vaultName,
			"--recovery-point-arn", rpArn)

		if err != nil {
			t.logger.Warnf("Failed to delete recovery point %s: %v", rpArn, err)
		} else {
			deletedCount++
		}
	}

	if deletedCount > 0 {
		t.logger.Infof("Deleted %d recovery points from vault: %s", deletedCount, vaultName)
	}

	return nil
}
