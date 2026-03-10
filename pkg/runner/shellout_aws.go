package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShellOutAWSRunner implements AWSRunner by shelling out to the aws CLI binary.
type ShellOutAWSRunner struct{}

// GetCallerIdentityAccount returns the AWS account ID of the current caller.
func (r *ShellOutAWSRunner) GetCallerIdentityAccount(ctx context.Context) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	if err != nil {
		return "", fmt.Errorf("aws GetCallerIdentityAccount: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// IAMRoleExists checks whether the named IAM role exists.
func (r *ShellOutAWSRunner) IAMRoleExists(ctx context.Context, roleName string) (bool, error) {
	_, err := utils.ExecuteCommand(ctx, "aws", "iam", "get-role", "--role-name", roleName)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") || strings.Contains(err.Error(), "not found") {
			return false, nil
		}
		return false, fmt.Errorf("aws IAMRoleExists: %w", err)
	}
	return true, nil
}

// EFSDescribeFileSystems returns file systems, optionally filtered by ID.
func (r *ShellOutAWSRunner) EFSDescribeFileSystems(ctx context.Context, region, fsID string) ([]EFSFileSystem, error) {
	args := []string{"efs", "describe-file-systems", "--region", region}
	if fsID != "" {
		args = append(args, "--file-system-id", fsID)
	}
	out, err := utils.ExecuteCommand(ctx, "aws", args...)
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeFileSystems: %w", err)
	}
	var result struct {
		FileSystems []struct {
			FileSystemID   string `json:"FileSystemId"`
			Name           string `json:"Name"`
			LifeCycleState string `json:"LifeCycleState"`
			CreationTime   string `json:"CreationTime"`
			ThroughputMode string `json:"ThroughputMode"`
			Tags           []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags"`
		} `json:"FileSystems"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws EFSDescribeFileSystems: parse response: %w", err)
	}
	fsList := make([]EFSFileSystem, 0, len(result.FileSystems))
	for _, fs := range result.FileSystems {
		name := fs.Name
		if name == "" {
			for _, tag := range fs.Tags {
				if tag.Key == "Name" {
					name = tag.Value
				}
			}
		}
		ct, _ := time.Parse(time.RFC3339, fs.CreationTime)
		fsList = append(fsList, EFSFileSystem{
			FileSystemID:   fs.FileSystemID,
			Name:           name,
			LifeCycleState: fs.LifeCycleState,
			CreationTime:   ct,
			ThroughputMode: fs.ThroughputMode,
		})
	}
	return fsList, nil
}

// EFSCreateMountTarget creates a mount target for the given file system.
func (r *ShellOutAWSRunner) EFSCreateMountTarget(ctx context.Context, region, fsID, subnetID string, securityGroups []string) error {
	args := []string{"efs", "create-mount-target",
		"--file-system-id", fsID,
		"--subnet-id", subnetID,
		"--region", region,
	}
	if len(securityGroups) > 0 {
		args = append(args, "--security-groups")
		args = append(args, securityGroups...)
	}
	_, err := utils.ExecuteCommand(ctx, "aws", args...)
	if err != nil {
		return fmt.Errorf("aws EFSCreateMountTarget: %w", err)
	}
	return nil
}

// EFSDescribeMountTargets returns mount targets for a file system.
func (r *ShellOutAWSRunner) EFSDescribeMountTargets(ctx context.Context, region, fsID string) ([]EFSMountTarget, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets",
		"--file-system-id", fsID,
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargets: %w", err)
	}
	var result struct {
		MountTargets []struct {
			MountTargetID        string `json:"MountTargetId"`
			SubnetID             string `json:"SubnetId"`
			AvailabilityZoneName string `json:"AvailabilityZoneName"`
		} `json:"MountTargets"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargets: parse response: %w", err)
	}
	mts := make([]EFSMountTarget, 0, len(result.MountTargets))
	for _, mt := range result.MountTargets {
		mts = append(mts, EFSMountTarget{
			MountTargetID:        mt.MountTargetID,
			SubnetID:             mt.SubnetID,
			AvailabilityZoneName: mt.AvailabilityZoneName,
		})
	}
	return mts, nil
}

// EFSDescribeMountTargetSecurityGroups returns security group IDs for a mount target.
func (r *ShellOutAWSRunner) EFSDescribeMountTargetSecurityGroups(ctx context.Context, region, mountTargetID string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups",
		"--mount-target-id", mountTargetID,
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargetSecurityGroups: %w", err)
	}
	var result struct {
		SecurityGroups []string `json:"SecurityGroups"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws EFSDescribeMountTargetSecurityGroups: parse response: %w", err)
	}
	return result.SecurityGroups, nil
}

// EFSDeleteMountTarget deletes a mount target.
func (r *ShellOutAWSRunner) EFSDeleteMountTarget(ctx context.Context, region, mountTargetID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-mount-target",
		"--mount-target-id", mountTargetID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EFSDeleteMountTarget: %w", err)
	}
	return nil
}

// EFSDeleteFileSystem deletes an EFS file system.
func (r *ShellOutAWSRunner) EFSDeleteFileSystem(ctx context.Context, region, fsID string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "delete-file-system",
		"--file-system-id", fsID,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EFSDeleteFileSystem: %w", err)
	}
	return nil
}

// EFSUpdateFileSystem updates an EFS file system (e.g. throughput mode).
func (r *ShellOutAWSRunner) EFSUpdateFileSystem(ctx context.Context, region, fsID, throughputMode string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "update-file-system",
		"--file-system-id", fsID,
		"--throughput-mode", throughputMode,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EFSUpdateFileSystem: %w", err)
	}
	return nil
}

// EFSTagResource applies tags to an EFS resource.
func (r *ShellOutAWSRunner) EFSTagResource(ctx context.Context, region, resourceID string, tags map[string]string) error {
	tagArgs := make([]string, 0, len(tags))
	for k, v := range tags {
		tagArgs = append(tagArgs, fmt.Sprintf("Key=%s,Value=%s", k, v))
	}
	_, err := utils.ExecuteCommand(ctx, "aws", "efs", "tag-resource",
		"--resource-id", resourceID,
		"--tags", strings.Join(tagArgs, " "),
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws EFSTagResource: %w", err)
	}
	return nil
}

// BackupStartBackupJob starts an on-demand backup job and returns the job ID.
func (r *ShellOutAWSRunner) BackupStartBackupJob(ctx context.Context, region, vaultName, resourceArn, iamRoleArn string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--backup-vault-name", vaultName,
		"--resource-arn", resourceArn,
		"--iam-role-arn", iamRoleArn,
		"--region", region)
	if err != nil {
		return "", fmt.Errorf("aws BackupStartBackupJob: %w", err)
	}
	var result struct {
		BackupJobID string `json:"BackupJobId"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return "", fmt.Errorf("aws BackupStartBackupJob: parse response: %w", err)
	}
	return result.BackupJobID, nil
}

// BackupDescribeBackupJob returns the status of a backup job.
func (r *ShellOutAWSRunner) BackupDescribeBackupJob(ctx context.Context, region, jobID string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-backup-job",
		"--backup-job-id", jobID,
		"--region", region,
		"--query", "State",
		"--output", "text")
	if err != nil {
		return "", fmt.Errorf("aws BackupDescribeBackupJob: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// BackupStartRestoreJob starts a restore job and returns the job ID.
func (r *ShellOutAWSRunner) BackupStartRestoreJob(ctx context.Context, region, recoveryPointArn, iamRoleArn string, metadata map[string]string) (string, error) {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("aws BackupStartRestoreJob: marshal metadata: %w", err)
	}
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-restore-job",
		"--recovery-point-arn", recoveryPointArn,
		"--iam-role-arn", iamRoleArn,
		"--metadata", string(metadataJSON),
		"--region", region)
	if err != nil {
		return "", fmt.Errorf("aws BackupStartRestoreJob: %w", err)
	}
	var result struct {
		RestoreJobID string `json:"RestoreJobId"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return "", fmt.Errorf("aws BackupStartRestoreJob: parse response: %w", err)
	}
	return result.RestoreJobID, nil
}

// BackupDescribeRestoreJob returns the status and created resource ARN of a restore job.
func (r *ShellOutAWSRunner) BackupDescribeRestoreJob(ctx context.Context, region, jobID string) (BackupRestoreJobStatus, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-restore-job",
		"--restore-job-id", jobID,
		"--region", region)
	if err != nil {
		return BackupRestoreJobStatus{}, fmt.Errorf("aws BackupDescribeRestoreJob: %w", err)
	}
	var result struct {
		Status             string `json:"Status"`
		CreatedResourceArn string `json:"CreatedResourceArn"`
		StatusMessage      string `json:"StatusMessage"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return BackupRestoreJobStatus{}, fmt.Errorf("aws BackupDescribeRestoreJob: parse response: %w", err)
	}
	return BackupRestoreJobStatus{
		Status:             result.Status,
		CreatedResourceArn: result.CreatedResourceArn,
		StatusMessage:      result.StatusMessage,
	}, nil
}

// BackupListRecoveryPointsByResource lists recovery points for a resource ARN.
func (r *ShellOutAWSRunner) BackupListRecoveryPointsByResource(ctx context.Context, region, resourceArn string) ([]BackupRecoveryPoint, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--resource-arn", resourceArn,
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByResource: %w", err)
	}
	var result struct {
		RecoveryPoints []struct {
			RecoveryPointArn string  `json:"RecoveryPointArn"`
			BackupVaultName  string  `json:"BackupVaultName"`
			CreationDate     string  `json:"CreationDate"`
			ExpiryDate       *string `json:"ExpiryDate"`
			Status           string  `json:"Status"`
		} `json:"RecoveryPoints"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByResource: parse response: %w", err)
	}
	points := make([]BackupRecoveryPoint, 0, len(result.RecoveryPoints))
	for _, rp := range result.RecoveryPoints {
		cd, _ := time.Parse(time.RFC3339, rp.CreationDate)
		var ed *time.Time
		if rp.ExpiryDate != nil {
			t, _ := time.Parse(time.RFC3339, *rp.ExpiryDate)
			ed = &t
		}
		points = append(points, BackupRecoveryPoint{
			RecoveryPointArn: rp.RecoveryPointArn,
			BackupVaultName:  rp.BackupVaultName,
			CreationDate:     cd,
			ExpiryDate:       ed,
			Status:           rp.Status,
		})
	}
	return points, nil
}

// BackupListRecoveryPointsByVault lists recovery points in a backup vault.
func (r *ShellOutAWSRunner) BackupListRecoveryPointsByVault(ctx context.Context, region, vaultName string) ([]string, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-backup-vault",
		"--backup-vault-name", vaultName,
		"--region", region,
		"--query", "RecoveryPoints[].RecoveryPointArn",
		"--output", "json")
	if err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByVault: %w", err)
	}
	var arns []string
	if err := json.Unmarshal([]byte(out), &arns); err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByVault: parse response: %w", err)
	}
	return arns, nil
}

// BackupDeleteRecoveryPoint deletes a recovery point from a vault.
func (r *ShellOutAWSRunner) BackupDeleteRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-recovery-point",
		"--backup-vault-name", vaultName,
		"--recovery-point-arn", recoveryPointArn,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws BackupDeleteRecoveryPoint: %w", err)
	}
	return nil
}

// BackupListBackupVaults lists backup vaults.
func (r *ShellOutAWSRunner) BackupListBackupVaults(ctx context.Context, region string) ([]BackupVault, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-vaults",
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws BackupListBackupVaults: %w", err)
	}
	var result struct {
		BackupVaultList []struct {
			BackupVaultName string `json:"BackupVaultName"`
		} `json:"BackupVaultList"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws BackupListBackupVaults: parse response: %w", err)
	}
	vaults := make([]BackupVault, 0, len(result.BackupVaultList))
	for _, v := range result.BackupVaultList {
		vaults = append(vaults, BackupVault{BackupVaultName: v.BackupVaultName})
	}
	return vaults, nil
}

// BackupDeleteBackupVault deletes a backup vault.
func (r *ShellOutAWSRunner) BackupDeleteBackupVault(ctx context.Context, region, vaultName string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "backup", "delete-backup-vault",
		"--backup-vault-name", vaultName,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws BackupDeleteBackupVault: %w", err)
	}
	return nil
}

// BackupIsResourceProtected checks if a resource ARN is protected.
func (r *ShellOutAWSRunner) BackupIsResourceProtected(ctx context.Context, region, resourceArn string) (bool, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", fmt.Sprintf("Results[?ResourceArn=='%s']", resourceArn),
		"--output", "json")
	if err != nil {
		return false, fmt.Errorf("aws BackupIsResourceProtected: %w", err)
	}
	var results []json.RawMessage
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		return false, fmt.Errorf("aws BackupIsResourceProtected: parse response: %w", err)
	}
	return len(results) > 0, nil
}

// BackupDescribeRecoveryPoint checks whether a recovery point exists in a vault.
func (r *ShellOutAWSRunner) BackupDescribeRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	_, err := utils.ExecuteCommand(ctx, "aws", "backup", "describe-recovery-point",
		"--backup-vault-name", vaultName,
		"--recovery-point-arn", recoveryPointArn,
		"--region", region)
	if err != nil {
		return fmt.Errorf("aws BackupDescribeRecoveryPoint: %w", err)
	}
	return nil
}

// BackupListBackupPlans lists backup plans.
func (r *ShellOutAWSRunner) BackupListBackupPlans(ctx context.Context, region string) ([]BackupPlan, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-plans",
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws BackupListBackupPlans: %w", err)
	}
	var result struct {
		BackupPlansList []struct {
			BackupPlanID   string `json:"BackupPlanId"`
			BackupPlanName string `json:"BackupPlanName"`
		} `json:"BackupPlansList"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws BackupListBackupPlans: parse response: %w", err)
	}
	plans := make([]BackupPlan, 0, len(result.BackupPlansList))
	for _, p := range result.BackupPlansList {
		plans = append(plans, BackupPlan{
			BackupPlanID:   p.BackupPlanID,
			BackupPlanName: p.BackupPlanName,
		})
	}
	return plans, nil
}

// BackupGetBackupPlan returns details for a specific backup plan.
func (r *ShellOutAWSRunner) BackupGetBackupPlan(ctx context.Context, region, planID string) (*BackupPlanDetail, error) {
	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "get-backup-plan",
		"--backup-plan-id", planID,
		"--region", region)
	if err != nil {
		return nil, fmt.Errorf("aws BackupGetBackupPlan: %w", err)
	}
	var result struct {
		BackupPlan struct {
			Rules []struct {
				ScheduleExpression string `json:"ScheduleExpression"`
				Lifecycle          *struct {
					DeleteAfterDays int `json:"DeleteAfterDays"`
				} `json:"Lifecycle"`
			} `json:"Rules"`
		} `json:"BackupPlan"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		return nil, fmt.Errorf("aws BackupGetBackupPlan: parse response: %w", err)
	}
	detail := &BackupPlanDetail{
		Rules: make([]BackupPlanRule, 0, len(result.BackupPlan.Rules)),
	}
	for _, rule := range result.BackupPlan.Rules {
		deleteAfter := 0
		if rule.Lifecycle != nil {
			deleteAfter = rule.Lifecycle.DeleteAfterDays
		}
		detail.Rules = append(detail.Rules, BackupPlanRule{
			ScheduleExpression: rule.ScheduleExpression,
			DeleteAfterDays:    deleteAfter,
		})
	}
	return detail, nil
}
