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
		vaultsJSON, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource", "--region", region, "--resource-arn", arn, "--query", "RecoveryPoints[].BackupVaultName", "--output", "json")
		if err != nil {
			fmt.Printf("   Vaults: ❌ Error checking vaults: %v\n", err)
		} else {
			vaultsJSON = strings.TrimSpace(vaultsJSON)
			if vaultsJSON == "" || vaultsJSON == "null" || vaultsJSON == "[]" {
				fmt.Printf("   Vaults: ⚠️  None (no backups found)\n")
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
						fmt.Printf("   Vaults: ✅ %s\n", unique)
					} else {
						fmt.Printf("   Vaults: ⚠️  None (no backups found)\n")
					}
				} else {
					fmt.Printf("   Vaults: ⚠️  None (no backups found)\n")
				}
			}
		}
	} else {
		fmt.Printf("📁 EFS Backup Status\n")
		fmt.Printf("   ❌ Not detected in cluster PVs: %v\n", err)
	}

	fmt.Println("")

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

			// Ensure throughput mode is elastic on the restored EFS
			if err := t.setEFSThroughputElastic(ctx, newFsId); err != nil {
				fmt.Printf("[WARNING] Failed to set EFS ThroughputMode to elastic: %v\n", err)
			} else {
				fmt.Println("[EFS] ✅ ThroughputMode set to elastic")
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

	fmt.Printf("[DEBUG] Trying namespace-based backup service role: %s\n", namespaceRoleArn)

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
		fmt.Printf("[DEBUG] Trying AWS managed role: %s\n", roleArn)

		if _, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName); err == nil {
			return roleArn, nil
		}
	}

	// If no suitable role found, return the namespace-based role and let AWS handle the error
	fmt.Printf("[WARNING] No suitable IAM role found, using namespace-based role: %s\n", namespaceRoleArn)
	return namespaceRoleArn, nil
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

	// Ensure verify pod is removed before proceeding to health check
	_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")

	// Update PV volumeHandle if EFS ID provided
	if newEfs != "" {
		// Try to replicate mount targets of the currently used EFS to the new EFS
		if srcEfs, err := utils.DetectEFSId(ctx, namespace); err == nil && strings.TrimSpace(srcEfs) != "" {
			if err := t.replicateEFSMountTargets(ctx, t.deployConfig.AWS.Region, strings.TrimSpace(srcEfs), newEfs); err != nil {
				fmt.Printf("[WARNING] Failed to replicate EFS mount targets from %s to %s: %v\n", srcEfs, newEfs, err)
			} else {
				fmt.Printf("[ATTACH] ✅ Replicated mount targets from %s to %s (region: %s)\n", srcEfs, newEfs, t.deployConfig.AWS.Region)
			}
		} else {
			fmt.Printf("[WARNING] Could not detect source EFS in namespace %s. Skipping mount-target replication.\n", namespace)
		}

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
	// if err := t.performHealthCheck(ctx, namespace); err != nil {
	// 	return fmt.Errorf("health check failed: %w", err)
	// }

	fmt.Println("[POST-RESTORE] Completed basic post-restore steps.")

	// Summary
	t.printAttachSummary(newEfs, pvcs, stss)
	return nil
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
		fmt.Println("[ATTACH] No mount targets found on source EFS; skipping replication")
		return nil
	}

	// For each source mount target, fetch SGs and create on destination
	for _, mt := range mtResp.MountTargets {
		if strings.TrimSpace(mt.SubnetId) == "" || strings.TrimSpace(mt.MountTargetId) == "" {
			continue
		}
		sgOut, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups", "--region", region, "--mount-target-id", mt.MountTargetId, "--query", "SecurityGroups", "--output", "text")
		if err != nil {
			fmt.Printf("[WARNING] Failed to get SGs for %s: %v\n", mt.MountTargetId, err)
			continue
		}
		sgOut = strings.TrimSpace(sgOut)
		if sgOut == "" {
			fmt.Printf("[WARNING] No SGs for %s; skipping subnet %s\n", mt.MountTargetId, mt.SubnetId)
			continue
		}
		// create mount target; ignore errors if already exists
		args := []string{"efs", "create-mount-target", "--region", region, "--file-system-id", dstFs, "--subnet-id", mt.SubnetId, "--security-groups"}
		args = append(args, strings.Fields(sgOut)...)
		if _, err := utils.ExecuteCommand(ctx, "aws", args...); err != nil {
			fmt.Printf("[ATTACH] Note: create-mount-target may have failed/exists for subnet %s: %v\n", mt.SubnetId, err)
		} else {
			fmt.Printf("[ATTACH] Created mount target on subnet %s (AZ %s) for %s\n", mt.SubnetId, mt.AvailabilityZoneName, dstFs)
		}
	}
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

	fmt.Printf("[VERIFY] Using PVC: %s\n", opGethPVC)

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
	fmt.Println("[VERIFY] Waiting for verification pod to complete...")
	for i := 0; i < 60; i++ { // Increased timeout to 2 minutes
		status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", "verify-efs", "-o", "jsonpath={.status.phase}")
		if err != nil {
			fmt.Printf("[VERIFY] Pod status check failed: %v\n", err)
			time.Sleep(2 * time.Second)
			continue
		}

		status = strings.TrimSpace(status)
		if status == "Succeeded" {
			fmt.Println("[VERIFY] ✅ EFS data verification completed successfully")
			return nil
		}
		if status == "Failed" {
			// Get pod logs for debugging
			logs, _ := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", "verify-efs")
			fmt.Printf("[VERIFY] Pod failed. Logs: %s\n", logs)
			return fmt.Errorf("verification pod failed")
		}
		if status == "Pending" {
			fmt.Printf("[VERIFY] Pod is pending... (attempt %d/60)\n", i+1)
		}
		if status == "Running" {
			fmt.Printf("[VERIFY] Pod is running... (attempt %d/60)\n", i+1)
		}
		time.Sleep(2 * time.Second)
	}

	// Clean up the pod
	utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	return fmt.Errorf("verification pod timed out after 2 minutes")
}

// updatePVVolumeHandles updates PV volume handles to point to new EFS
func (t *ThanosStack) updatePVVolumeHandles(ctx context.Context, namespace, newEfs string, pvcs *string) error {
	fmt.Printf("[ATTACH] Updating PV volume handles to EFS: %s\n", newEfs)

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
				fmt.Printf("[WARNING] PVC alias '%s' did not match any PVC in namespace %s\n", alias, namespace)
				continue
			}

			// Validate resolved PVC exists
			if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", resolvedPVC); err != nil {
				fmt.Printf("[WARNING] PVC %s not found after resolution, skipping\n", resolvedPVC)
				continue
			}

			fmt.Printf("[ATTACH] Resolved PVC alias '%s' -> '%s'\n", alias, resolvedPVC)
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
				fmt.Printf("[ATTACH] Found PVC: %s\n", pvc)
			}
		}
	}

	if len(targetPVCs) == 0 {
		return fmt.Errorf("no target PVCs found to update")
	}

	// Since PV volumeHandle is immutable, we need to delete and recreate PVs
	fmt.Printf("[ATTACH] PV volumeHandle is immutable, deleting and recreating PVs with new EFS...\n")

	successCount := 0
	for _, pvcName := range targetPVCs {
		pvcName = strings.TrimSpace(pvcName)
		if pvcName == "" {
			continue
		}

		// Get current PV name from PVC
		pvName, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.spec.volumeName}")
		if err != nil {
			fmt.Printf("[ERROR] Failed to get PV name for PVC %s: %v\n", pvcName, err)
			continue
		}
		pvName = strings.TrimSpace(pvName)
		if pvName == "" {
			fmt.Printf("[WARNING] PVC %s has no volumeName, skipping\n", pvcName)
			continue
		}

		fmt.Printf("[ATTACH] Processing PVC: %s (PV: %s)\n", pvcName, pvName)

		// Delete the PVC first (this will also delete the PV if it's dynamically provisioned)
		fmt.Printf("[ATTACH] Deleting PVC %s...\n", pvcName)

		// First, delete pods that use this PVC
		podsUsingPVC, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pods", "-o", "jsonpath={range .items[?(@.spec.volumes[*].persistentVolumeClaim.claimName=='"+pvcName+"')]}{.metadata.name}{\"\\n\"}{end}")
		if err == nil && strings.TrimSpace(podsUsingPVC) != "" {
			for _, podName := range strings.Fields(podsUsingPVC) {
				fmt.Printf("[ATTACH] Deleting pod %s that uses PVC %s...\n", podName, pvcName)
				utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", podName, "--ignore-not-found=true")
			}
			// Wait for pods to be deleted
			time.Sleep(5 * time.Second)
		}

		// Delete PVC
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pvc", pvcName); err != nil {
			fmt.Printf("[ERROR] Failed to delete PVC %s: %v\n", pvcName, err)
			continue
		}

		// Delete the old PV
		fmt.Printf("[ATTACH] Deleting old PV %s...\n", pvName)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", pvName, "--ignore-not-found=true"); err != nil {
			fmt.Printf("[WARNING] Failed to delete PV %s: %v\n", pvName, err)
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
			fmt.Printf("[ERROR] Failed to create temporary PV YAML file: %v\n", err)
			continue
		}
		defer os.Remove(tempPVFile)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVFile); err != nil {
			fmt.Printf("[ERROR] Failed to create new PV %s: %v\n", pvName, err)
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
			fmt.Printf("[ERROR] Failed to create temporary PVC YAML file: %v\n", err)
			continue
		}
		defer os.Remove(tempPVCFile)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVCFile); err != nil {
			fmt.Printf("[ERROR] Failed to create new PVC %s: %v\n", pvcName, err)
			continue
		}

		// Wait for PVC to be bound
		fmt.Printf("[ATTACH] Waiting for PVC %s to be bound...\n", pvcName)
		for i := 0; i < 30; i++ { // Wait up to 30 seconds
			status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.status.phase}")
			if err == nil && strings.TrimSpace(status) == "Bound" {
				break
			}
			time.Sleep(1 * time.Second)
		}

		fmt.Printf("[ATTACH] ✅ PVC %s and PV %s recreated successfully with new EFS\n", pvcName, pvName)
		successCount++
	}

	if successCount == 0 {
		return fmt.Errorf("failed to recreate any PVCs/PVs")
	}

	fmt.Printf("[ATTACH] ✅ Successfully recreated %d/%d PVCs with new EFS\n", successCount, len(targetPVCs))
	return nil
}

// restartStatefulSets restarts specified StatefulSets
func (t *ThanosStack) restartStatefulSets(ctx context.Context, namespace, stss string) error {
	fmt.Printf("[RESTART] Restarting StatefulSets: %s\n", stss)

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
			fmt.Printf("[WARNING] StatefulSet alias '%s' not found in namespace %s, skipping\n", sts, namespace)
			continue
		}

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "rollout", "restart", fmt.Sprintf("statefulset/%s", resolved)); err != nil {
			fmt.Printf("[ERROR] Failed to restart StatefulSet %s: %v\n", resolved, err)
		} else {
			fmt.Printf("[RESTART] ✅ StatefulSet %s restarted successfully\n", resolved)
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
			fmt.Printf("[HEALTH] ✅ %s restarted (pod: %s)\n", comp, pod)
		} else {
			fmt.Printf("[HEALTH] ℹ️ %s shows no container restarts (pod: %s)\n", comp, pod)
		}
	}

	if !restarted["op-geth"] || !restarted["op-node"] {
		fmt.Println("[HEALTH] Warning: One or more components did not report a container restart.")
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
		fmt.Printf("[HEALTH] ❌ Failed to fetch L2 block: %v\n", l2Err)
	} else {
		fmt.Printf("[HEALTH] L2 latest block: %s (via %s)\n", l2Block, l2URL)
	}

	l1Block, l1Err := queryBlock(l1URL)
	if l1Err != nil {
		fmt.Printf("[HEALTH] ❌ Failed to fetch L1 block: %v\n", l1Err)
	} else {
		fmt.Printf("[HEALTH] L1 latest block (on-chain): %s (via %s)\n", l1Block, l1URL)
	}

	// Additionally, fetch L1 block height as seen by op-node (synced view)
	opNodeSynced := ""
	if svcs, err := utils.GetServiceNames(ctx, namespace, "op-node"); err == nil && len(svcs) > 0 {
		metricsURL := fmt.Sprintf("http://%s.%s.svc.cluster.local:7300/metrics", svcs[0], namespace)
		val, err := utils.ExecuteCommand(ctx, "bash", "-c", fmt.Sprintf("curl -s %s | grep -m1 '^op_node_l1_head' | awk '{print $NF}'", metricsURL))
		if err == nil && strings.TrimSpace(val) != "" {
			opNodeSynced = strings.TrimSpace(val)
			fmt.Printf("[HEALTH] L1 latest block (op-node synced): %s (via %s)\n", opNodeSynced, metricsURL)
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
						fmt.Printf("[HEALTH] L1 latest block (op-node synced): %s (via %s rollup_getInfo)\n", opNodeSynced, opNodeRPC)
						break
					}
				}
				if opNodeSynced == "" {
					fmt.Printf("[HEALTH] ℹ️ Could not parse op-node synced L1 block from rollup_getInfo\n")
				}
			} else {
				fmt.Printf("[HEALTH] ℹ️ Could not reach op-node metrics or RPC for synced L1 block\n")
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

	// Get backup vault name from namespace
	vaultName := fmt.Sprintf("%s-backup-vault", namespace)

	fmt.Printf("\n[SELECTION] Available EFS Recovery Points from Backup Vault: %s\n", vaultName)
	fmt.Println("   -------------------------------------------------")

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
		fmt.Printf("   %d. %s\n", i+1, displayDate)
		fmt.Printf("      📁 Vault: %s\n", vaultName)
		fmt.Printf("      🗂️  EFS ID: %s\n", efsID)
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

// interactiveRestore provides a fully interactive restore experience for EFS and RDS
func (t *ThanosStack) interactiveRestore(ctx context.Context) error {
	fmt.Println("\n🔄 Starting Interactive Restore")
	fmt.Println("================================")

	// EFS-only flow
	return t.interactiveEFSRestore(ctx)
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
				fmt.Printf("   ❌ Attach failed: %v\n", err)
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

// verifyEFSRestore verifies that the EFS restore was successful
func (t *ThanosStack) verifyEFSRestore(ctx context.Context, efsID string) error {
	fmt.Println("\n🔍 Verifying EFS Restore...")

	// Check EFS status
	status, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-id", efsID,
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
		"--file-system-id", efsID,
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
