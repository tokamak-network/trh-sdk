package runner

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/backup"
)

// BackupStartBackupJob starts an on-demand backup job and returns the job ID.
func (r *NativeAWSRunner) BackupStartBackupJob(ctx context.Context, region, vaultName, resourceArn, iamRoleArn string) (string, error) {
	out, err := r.backupClient(region).StartBackupJob(ctx, &backup.StartBackupJobInput{
		BackupVaultName: aws.String(vaultName),
		ResourceArn:     aws.String(resourceArn),
		IamRoleArn:      aws.String(iamRoleArn),
	})
	if err != nil {
		return "", fmt.Errorf("aws BackupStartBackupJob: %w", err)
	}
	return aws.ToString(out.BackupJobId), nil
}

// BackupDescribeBackupJob returns the status of a backup job.
func (r *NativeAWSRunner) BackupDescribeBackupJob(ctx context.Context, region, jobID string) (string, error) {
	out, err := r.backupClient(region).DescribeBackupJob(ctx, &backup.DescribeBackupJobInput{
		BackupJobId: aws.String(jobID),
	})
	if err != nil {
		return "", fmt.Errorf("aws BackupDescribeBackupJob: %w", err)
	}
	return string(out.State), nil
}

// BackupStartRestoreJob starts a restore job and returns the job ID.
func (r *NativeAWSRunner) BackupStartRestoreJob(ctx context.Context, region, recoveryPointArn, iamRoleArn string, metadata map[string]string) (string, error) {
	out, err := r.backupClient(region).StartRestoreJob(ctx, &backup.StartRestoreJobInput{
		RecoveryPointArn: aws.String(recoveryPointArn),
		IamRoleArn:       aws.String(iamRoleArn),
		Metadata:         metadata,
	})
	if err != nil {
		return "", fmt.Errorf("aws BackupStartRestoreJob: %w", err)
	}
	return aws.ToString(out.RestoreJobId), nil
}

// BackupDescribeRestoreJob returns the status and created resource ARN of a restore job.
func (r *NativeAWSRunner) BackupDescribeRestoreJob(ctx context.Context, region, jobID string) (BackupRestoreJobStatus, error) {
	out, err := r.backupClient(region).DescribeRestoreJob(ctx, &backup.DescribeRestoreJobInput{
		RestoreJobId: aws.String(jobID),
	})
	if err != nil {
		return BackupRestoreJobStatus{}, fmt.Errorf("aws BackupDescribeRestoreJob: %w", err)
	}
	return BackupRestoreJobStatus{
		Status:             string(out.Status),
		CreatedResourceArn: aws.ToString(out.CreatedResourceArn),
		StatusMessage:      aws.ToString(out.StatusMessage),
	}, nil
}

// BackupListRecoveryPointsByResource lists recovery points for a resource ARN.
func (r *NativeAWSRunner) BackupListRecoveryPointsByResource(ctx context.Context, region, resourceArn string) ([]BackupRecoveryPoint, error) {
	out, err := r.backupClient(region).ListRecoveryPointsByResource(ctx, &backup.ListRecoveryPointsByResourceInput{
		ResourceArn: aws.String(resourceArn),
	})
	if err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByResource: %w", err)
	}
	points := make([]BackupRecoveryPoint, 0, len(out.RecoveryPoints))
	for _, rp := range out.RecoveryPoints {
		points = append(points, BackupRecoveryPoint{
			RecoveryPointArn: aws.ToString(rp.RecoveryPointArn),
			BackupVaultName:  aws.ToString(rp.BackupVaultName),
			CreationDate:     aws.ToTime(rp.CreationDate),
			ExpiryDate:       nil,
			Status:           string(rp.Status),
		})
	}
	return points, nil
}

// BackupListRecoveryPointsByVault lists recovery points in a backup vault.
func (r *NativeAWSRunner) BackupListRecoveryPointsByVault(ctx context.Context, region, vaultName string) ([]string, error) {
	out, err := r.backupClient(region).ListRecoveryPointsByBackupVault(ctx, &backup.ListRecoveryPointsByBackupVaultInput{
		BackupVaultName: aws.String(vaultName),
	})
	if err != nil {
		return nil, fmt.Errorf("aws BackupListRecoveryPointsByVault: %w", err)
	}
	arns := make([]string, 0, len(out.RecoveryPoints))
	for _, rp := range out.RecoveryPoints {
		arns = append(arns, aws.ToString(rp.RecoveryPointArn))
	}
	return arns, nil
}

// BackupDeleteRecoveryPoint deletes a recovery point from a vault.
func (r *NativeAWSRunner) BackupDeleteRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	_, err := r.backupClient(region).DeleteRecoveryPoint(ctx, &backup.DeleteRecoveryPointInput{
		BackupVaultName:  aws.String(vaultName),
		RecoveryPointArn: aws.String(recoveryPointArn),
	})
	if err != nil {
		return fmt.Errorf("aws BackupDeleteRecoveryPoint: %w", err)
	}
	return nil
}

// BackupListBackupVaults lists backup vaults.
func (r *NativeAWSRunner) BackupListBackupVaults(ctx context.Context, region string) ([]BackupVault, error) {
	out, err := r.backupClient(region).ListBackupVaults(ctx, &backup.ListBackupVaultsInput{})
	if err != nil {
		return nil, fmt.Errorf("aws BackupListBackupVaults: %w", err)
	}
	vaults := make([]BackupVault, 0, len(out.BackupVaultList))
	for _, v := range out.BackupVaultList {
		vaults = append(vaults, BackupVault{
			BackupVaultName: aws.ToString(v.BackupVaultName),
		})
	}
	return vaults, nil
}

// BackupDeleteBackupVault deletes a backup vault.
func (r *NativeAWSRunner) BackupDeleteBackupVault(ctx context.Context, region, vaultName string) error {
	_, err := r.backupClient(region).DeleteBackupVault(ctx, &backup.DeleteBackupVaultInput{
		BackupVaultName: aws.String(vaultName),
	})
	if err != nil {
		return fmt.Errorf("aws BackupDeleteBackupVault: %w", err)
	}
	return nil
}

// BackupIsResourceProtected checks if a resource ARN is protected.
func (r *NativeAWSRunner) BackupIsResourceProtected(ctx context.Context, region, resourceArn string) (bool, error) {
	out, err := r.backupClient(region).ListProtectedResources(ctx, &backup.ListProtectedResourcesInput{})
	if err != nil {
		return false, fmt.Errorf("aws BackupIsResourceProtected: %w", err)
	}
	for _, res := range out.Results {
		if aws.ToString(res.ResourceArn) == resourceArn {
			return true, nil
		}
	}
	return false, nil
}

// BackupDescribeRecoveryPoint checks whether a recovery point exists in a vault.
func (r *NativeAWSRunner) BackupDescribeRecoveryPoint(ctx context.Context, region, vaultName, recoveryPointArn string) error {
	_, err := r.backupClient(region).DescribeRecoveryPoint(ctx, &backup.DescribeRecoveryPointInput{
		BackupVaultName:  aws.String(vaultName),
		RecoveryPointArn: aws.String(recoveryPointArn),
	})
	if err != nil {
		return fmt.Errorf("aws BackupDescribeRecoveryPoint: %w", err)
	}
	return nil
}

// BackupListBackupPlans lists backup plans.
func (r *NativeAWSRunner) BackupListBackupPlans(ctx context.Context, region string) ([]BackupPlan, error) {
	out, err := r.backupClient(region).ListBackupPlans(ctx, &backup.ListBackupPlansInput{})
	if err != nil {
		return nil, fmt.Errorf("aws BackupListBackupPlans: %w", err)
	}
	plans := make([]BackupPlan, 0, len(out.BackupPlansList))
	for _, p := range out.BackupPlansList {
		plans = append(plans, BackupPlan{
			BackupPlanID:   aws.ToString(p.BackupPlanId),
			BackupPlanName: aws.ToString(p.BackupPlanName),
		})
	}
	return plans, nil
}

// BackupGetBackupPlan returns details for a specific backup plan.
func (r *NativeAWSRunner) BackupGetBackupPlan(ctx context.Context, region, planID string) (*BackupPlanDetail, error) {
	out, err := r.backupClient(region).GetBackupPlan(ctx, &backup.GetBackupPlanInput{
		BackupPlanId: aws.String(planID),
	})
	if err != nil {
		return nil, fmt.Errorf("aws BackupGetBackupPlan: %w", err)
	}
	detail := &BackupPlanDetail{
		Rules: make([]BackupPlanRule, 0, len(out.BackupPlan.Rules)),
	}
	for _, rule := range out.BackupPlan.Rules {
		deleteAfter := 0
		if rule.Lifecycle != nil && rule.Lifecycle.DeleteAfterDays != nil {
			deleteAfter = int(aws.ToInt64(rule.Lifecycle.DeleteAfterDays))
		}
		detail.Rules = append(detail.Rules, BackupPlanRule{
			ScheduleExpression: aws.ToString(rule.ScheduleExpression),
			DeleteAfterDays:    deleteAfter,
		})
	}
	return detail, nil
}
