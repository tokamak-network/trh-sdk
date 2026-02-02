package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/backup"
	"github.com/aws/aws-sdk-go-v2/service/efs"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// InteractiveRestoreWithSelection displays recovery points and lets user select one
func InteractiveRestoreWithSelection(
	ctx context.Context,
	l *zap.SugaredLogger,
	region, namespace string,
	executeRestore func(context.Context, string) (*types.BackupRestoreInfo, error),
) error {
	// Get AWS account ID and EFS ID
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		return fmt.Errorf("failed to detect current EFS in namespace %s: %w", namespace, err)
	}

	resourceArn := utils.BuildEFSArn(region, accountID, efsID)

	// List recovery points
	rps, err := ListRecoveryPoints(ctx, region, resourceArn, "20")
	if err != nil {
		return fmt.Errorf("failed to list recovery points: %w", err)
	}

	// Filter only COMPLETED recovery points
	var availablePoints []types.RecoveryPoint
	for _, rp := range rps {
		if strings.ToUpper(rp.Status) == "COMPLETED" {
			availablePoints = append(availablePoints, rp)
		}
	}

	if len(availablePoints) == 0 {
		return fmt.Errorf("no available recovery points found for EFS %s", efsID)
	}

	// Display recovery points to user
	l.Info("")
	l.Infof("📦 Available Recovery Points (%d)", len(availablePoints))
	l.Info("")

	for i, rp := range availablePoints {
		createdRelative := formatRelativeTime(rp.Created)

		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		l.Infof("#%-2d", i+1) // Display 1-based index like --list
		l.Infof("    🔑 ARN      : %s", rp.RecoveryPointARN)
		l.Infof("    🗄️  Vault    : %s", rp.Vault)
		l.Infof("    📅 Created  : %s %s", rp.Created, createdRelative)

		if rp.Expiry != "" {
			expiryRelative := formatRelativeTime(rp.Expiry)
			l.Infof("    ⏰ Expires  : %s %s", rp.Expiry, expiryRelative)
		} else {
			l.Infof("    ⏰ Expires  : Never")
		}

		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if i < len(availablePoints)-1 {
			l.Info("")
		}
	}
	l.Info("")

	// Get user selection (1-based index)
	var sel int
	fmt.Printf("Enter index (1-%d): ", len(availablePoints))
	if _, err := fmt.Scanf("%d", &sel); err != nil || sel < 1 || sel > len(availablePoints) {
		return fmt.Errorf("invalid selection: must be between 1 and %d", len(availablePoints))
	}

	selectedArn := availablePoints[sel-1].RecoveryPointARN // Convert to 0-based array index

	// Execute restore with selected ARN
	_, err = executeRestore(ctx, selectedArn)
	return err
}

// DirectRestore executes restore from a specific recovery point ARN without user interaction
func DirectRestore(
	ctx context.Context,
	l *zap.SugaredLogger,
	region, namespace string,
	recoveryPointArn string,
	restoreEFS func(context.Context, string) (string, error),
	monitorRestore func(context.Context, string, func(string, float64)) (string, error),
	executeAttach func(context.Context, string, *string, *string, *string) error,
	attach *bool,
	pvcs *string,
	stss *string,
	progressReporter func(string, float64),
) (*types.BackupRestoreInfo, error) {
	// Validate ARN
	if !strings.Contains(recoveryPointArn, "arn:aws:backup:") {
		return nil, fmt.Errorf("invalid recovery point ARN: %s", recoveryPointArn)
	}

	l.Info("")
	l.Infof("🔄 Starting restore from recovery point...")
	l.Infof("    ARN: %s", recoveryPointArn)
	l.Info("")

	if progressReporter != nil {
		progressReporter("Starting restore job...", 10.0)
	}

	// Step 1: Start restore
	jobID, err := restoreEFS(ctx, recoveryPointArn)
	if err != nil {
		return nil, fmt.Errorf("failed to start restore: %w", err)
	}

	l.Infof("📝 Restore job started: %s", jobID)
	l.Info("")

	// Step 2: Monitor restore
	newEfsID, err := monitorRestore(ctx, jobID, progressReporter)
	if err != nil {
		return nil, fmt.Errorf("restore failed: %w", err)
	}

	l.Info("")
	l.Infof("✅ Restore completed. New EFS: %s", newEfsID)

	if progressReporter != nil {
		progressReporter("Restore completed, tagging resource...", 90.0)
	}

	// Step 3: Tag EFS
	if err := TagEFSWithName(ctx, region, newEfsID, namespace); err != nil {
		l.Warnf("Failed to tag EFS %s with Name=%s: %v", newEfsID, namespace, err)
	} else {
		l.Infof("✅ Tagged EFS %s with Name=%s", newEfsID, namespace)
	}

	defaultPVCs := "op-geth,op-node"
	defaultSTSs := "op-geth,op-node"
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		defaultPVCs = strings.TrimSpace(*pvcs)
	}
	if stss != nil && strings.TrimSpace(*stss) != "" {
		defaultSTSs = strings.TrimSpace(*stss)
	}

	// Build restore info for API response
	restoreInfo := &types.BackupRestoreInfo{
		Region:           region,
		Namespace:        namespace,
		EFSID:            "", // Will be set by caller if needed
		ARN:              "",
		RecoveryPointARN: recoveryPointArn,
		NewEFSID:         newEfsID,
		JobID:            jobID,
		Status:           "COMPLETED",
		SuggestedEFSID:   newEfsID,
		SuggestedPVCs:    defaultPVCs,
		SuggestedSTSs:    defaultSTSs,
	}

	// Step 4: Ask user if they want to attach (only for CLI)
	l.Info("")
	shouldAttach := false
	if attach != nil {
		shouldAttach = *attach
	} else if progressReporter == nil {
		response := "n"
		fmt.Print("Would you like to attach the restored EFS to workloads now? (y/n) ")
		if _, err := fmt.Scanf("%s", &response); err != nil {
			l.Warnf("Failed to read input: %v", err)
			response = "n"
		}
		response = strings.ToLower(strings.TrimSpace(response))
		shouldAttach = response == "y" || response == "yes"
	} else {
		l.Info("Non-interactive mode: skipping attach prompt.")
	}
	if shouldAttach {
		l.Info("")
		l.Info("🔗 Starting attach process...")

		if progressReporter != nil {
			progressReporter("Attaching to workloads...", 95.0)
		}

		if err := executeAttach(ctx, newEfsID, &defaultPVCs, &defaultSTSs, nil); err != nil {
			l.Errorf("❌ Attach failed: %v", err)
			l.Info("You can manually attach later with:")
			l.Infof("  ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
			restoreInfo.Status = "COMPLETED_WITH_ATTACH_ERROR"
			return restoreInfo, err
		}

		l.Info("")
		l.Info("✅ Attach completed successfully!")
		restoreInfo.Status = "COMPLETED_WITH_ATTACH"
	} else {
		l.Info("")
		l.Info("You can attach workloads to the restored EFS later with:")
		l.Infof("  ./trh-sdk backup-manager --attach --efs-id %s --pvc op-geth,op-node --sts op-geth,op-node", newEfsID)
	}

	if progressReporter != nil {
		progressReporter("Operation completed", 100.0)
	}

	return restoreInfo, nil
}

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

	query := "reverse(sort_by(RecoveryPoints,&CreationDate))[:20].{RecoveryPointArn:RecoveryPointArn,Created:CreationDate,Expiry:ExpiryDate,Status:Status,Vault:BackupVaultName}"
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
	var allItems []struct {
		RecoveryPointArn string `json:"RecoveryPointArn"`
		Created          string `json:"Created"`
		Expiry           string `json:"Expiry"`
		Status           string `json:"Status"`
		Vault            string `json:"Vault"`
	}
	if err := json.Unmarshal([]byte(out), &allItems); err != nil {
		return fmt.Errorf("failed to parse recovery points: %w", err)
	}

	// Filter only COMPLETED recovery points
	var items []struct {
		RecoveryPointArn string `json:"RecoveryPointArn"`
		Created          string `json:"Created"`
		Expiry           string `json:"Expiry"`
		Status           string `json:"Status"`
		Vault            string `json:"Vault"`
	}
	for _, item := range allItems {
		if strings.ToUpper(item.Status) == "COMPLETED" {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return fmt.Errorf("no available recovery points found for %s", resourceArn)
	}

	l.Info("")
	l.Infof("📦 Available Recovery Points (%d)", len(items))
	l.Info("")

	// Display recovery points in card style format
	for i, it := range items {
		createdRelative := formatRelativeTimeRestore(it.Created)

		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		l.Infof("[%d]", i)
		l.Infof("    🔑 ARN      : %s", it.RecoveryPointArn)
		l.Infof("    🗄️  Vault    : %s", it.Vault)
		l.Infof("    📅 Created  : %s %s", it.Created, createdRelative)

		// Handle expiry date - show "Never" if no expiry is set
		if strings.TrimSpace(it.Expiry) == "" {
			l.Infof("    ⏰ Expires  : Never")
		} else {
			expiryRelative := formatRelativeTimeRestore(it.Expiry)
			l.Infof("    ⏰ Expires  : %s %s", it.Expiry, expiryRelative)
		}

		l.Infof("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		if i < len(items)-1 {
			l.Info("")
		}
	}
	l.Info("")

	var sel int
	fmt.Print("Enter index: ")
	if _, err := fmt.Scanf("%d", &sel); err != nil || sel < 0 || sel >= len(items) {
		return fmt.Errorf("invalid selection")
	}
	selectedArn := items[sel].RecoveryPointArn

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

	// Use AWS managed EFS encryption key (always available in all AWS accounts)
	kmsKeyId := "alias/aws/elasticfilesystem"

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

// MonitorEFSRestoreJob monitors the status of a restore job
func MonitorEFSRestoreJob(ctx context.Context, l *zap.SugaredLogger, region string, jobId string, progressReporter func(string, float64)) (string, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}
	client := backup.NewFromConfig(cfg)
	efsClient := efs.NewFromConfig(cfg)

	l.Infof("⏳ Monitoring restore job %s...", jobId)

	const maxAttempts = 120 // 10 minutes total (120 * 5s)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for i := 0; i < maxAttempts; i++ {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			resp, err := client.DescribeRestoreJob(ctx, &backup.DescribeRestoreJobInput{
				RestoreJobId: aws.String(jobId),
			})
			if err != nil {
				l.Warnf("Failed to describe restore job (attempt %d/%d): %v", i+1, maxAttempts, err)
				continue
			}

			status := string(resp.Status)
			l.Infof("   Status: %s", status)

			if progressReporter != nil {
				// Interpolate progress 10% -> 90%
				percent := 10.0 + (float64(i)/float64(maxAttempts))*80.0
				if percent > 90 {
					percent = 90
				}
				progressReporter(fmt.Sprintf("Restoring: %s", status), percent)
			}

			if status == "COMPLETED" {
				if resp.CreatedResourceArn == nil {
					return "", fmt.Errorf("restore job completed but no resource ARN returned")
				}

				// Extract file-system-id from ARN
				// ARN format: arn:aws:elasticfilesystem:region:account-id:file-system/fs-id
				parts := strings.Split(*resp.CreatedResourceArn, "/")
				if len(parts) < 2 {
					return "", fmt.Errorf("invalid resource ARN format: %s", *resp.CreatedResourceArn)
				}
				newFsID := parts[len(parts)-1]

				l.Infof("✅ Restore job completed successfully. New EFS ID: %s", newFsID)

				// Verify EFS state
				l.Infof("Verifying new EFS state...")
				efsResp, err := efsClient.DescribeFileSystems(ctx, &efs.DescribeFileSystemsInput{
					FileSystemId: aws.String(newFsID),
				})
				if err != nil {
					l.Warnf("Could not verify EFS state immediately: %v", err)
				} else if len(efsResp.FileSystems) > 0 {
					l.Infof("   EFS Lifecycle State: %s", efsResp.FileSystems[0].LifeCycleState)
				}

				// Set EFS throughput mode to elastic for better performance
				if err := SetEFSThroughputElastic(ctx, region, newFsID); err != nil {
					l.Warnf("Failed to set EFS ThroughputMode to elastic: %v", err)
				} else {
					l.Info("✅ ThroughputMode set to elastic")
				}

				return newFsID, nil
			}

			if status == "FAILED" || status == "ABORTED" || status == "EXPIRED" {
				msg := "checker failed"
				if resp.StatusMessage != nil {
					msg = *resp.StatusMessage
				}
				return "", fmt.Errorf("restore job failed with status %s: %s", status, msg)
			}
		}
	}

	return "", fmt.Errorf("restore job monitoring timed out after 10 minutes")
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

// formatRelativeTimeRestore formats a timestamp to show relative time (e.g., "2 days ago")
func formatRelativeTimeRestore(timestamp string) string {
	if strings.TrimSpace(timestamp) == "" {
		return ""
	}

	// Try parsing with different time formats
	var t time.Time
	var err error

	// ISO8601 formats to try
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000000-07:00",
		"2006-01-02T15:04:05.000000Z07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		t, err = time.Parse(format, timestamp)
		if err == nil {
			break
		}
	}

	if err != nil {
		return ""
	}

	now := time.Now()
	duration := now.Sub(t)

	// If time is in the future
	if duration < 0 {
		duration = -duration
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24

		if days > 0 {
			return fmt.Sprintf("(in %d days)", days)
		} else if hours > 0 {
			return fmt.Sprintf("(in %d hours)", hours)
		} else {
			minutes := int(duration.Minutes())
			return fmt.Sprintf("(in %d minutes)", minutes)
		}
	}

	// If time is in the past
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24

	if days > 0 {
		return fmt.Sprintf("(%d days ago)", days)
	} else if hours > 0 {
		return fmt.Sprintf("(%d hours ago)", hours)
	} else {
		minutes := int(duration.Minutes())
		if minutes <= 0 {
			return "(just now)"
		}
		return fmt.Sprintf("(%d minutes ago)", minutes)
	}
}
