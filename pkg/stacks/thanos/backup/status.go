package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// GatherBackupStatusInfo collects backup status information using region/namespace
func GatherBackupStatusInfo(ctx context.Context, ar runner.AWSRunner, region, namespace string) (*types.BackupStatusInfo, error) {
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to detect AWS account ID: %w", err)
	}

	statusInfo := &types.BackupStatusInfo{
		Region:    region,
		Namespace: namespace,
		AccountID: accountID,
	}

	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || efsID == "" {
		return statusInfo, fmt.Errorf("failed to detect EFS ID: %w", err)
	}

	statusInfo.EFSID = efsID
	statusInfo.ARN = utils.BuildEFSArn(region, accountID, efsID)
	statusInfo.IsProtected = checkEFSProtectionStatus(ctx, ar, region, statusInfo.ARN)
	statusInfo.LatestRecoveryPoint = getLatestRecoveryPoint(ctx, ar, region, statusInfo.ARN)

	statusInfo.BackupVaults = getBackupVaults(ctx, ar, region, statusInfo.ARN)

	schedule, nextBackup, expiryDate, err := getBackupPlanInfo(ctx, ar, region, namespace, statusInfo.LatestRecoveryPoint)
	if err != nil {
		return statusInfo, fmt.Errorf("failed to get backup plan info: %w", err)
	}

	statusInfo.BackupSchedule = schedule
	statusInfo.NextBackupTime = nextBackup
	statusInfo.ExpectedExpiryDate = expiryDate

	return statusInfo, nil
}

// DisplayBackupStatus prints the backup status with provided logger
func DisplayBackupStatus(l *zap.SugaredLogger, statusInfo *types.BackupStatusInfo) {
	l.Infof("Region: %s, Namespace: %s, Account ID: %s", statusInfo.Region, statusInfo.Namespace, statusInfo.AccountID)
	l.Info("📁 EFS Backup Status")

	if statusInfo.EFSID == "" {
		l.Error("   ❌ EFS not detected in cluster PVs")
		return
	}

	l.Infof("   ARN: %s", statusInfo.ARN)
	if statusInfo.IsProtected {
		l.Info("   Protected: ✅ true")
	} else {
		l.Warn("   Protected: ❌ false")
	}

	// Handle latest recovery point display
	if statusInfo.LatestRecoveryPoint == "" || statusInfo.LatestRecoveryPoint == "None" {
		l.Warn("   Latest recovery point: ⚠️  None (no backups found)")
	} else {
		l.Infof("   Latest recovery point: %s", statusInfo.LatestRecoveryPoint)
	}

	// Handle expected expiry date display
	if statusInfo.ExpectedExpiryDate != "" {
		l.Infof("   Expected expiry date: 📅 %s", statusInfo.ExpectedExpiryDate)
	}

	// Handle backup vaults display
	if len(statusInfo.BackupVaults) == 0 {
		l.Warn("   Vaults: ⚠️  None (no backups found)")
	} else {
		l.Infof("   Vaults: %s", strings.Join(statusInfo.BackupVaults, ", "))
	}

	// Handle backup schedule display
	if statusInfo.BackupSchedule != "" {
		l.Infof("   Schedule: 📅 %s", statusInfo.BackupSchedule)
		if statusInfo.NextBackupTime != "" {
			l.Infof("   Next backup: ⏰ %s", statusInfo.NextBackupTime)
		}
	}

	l.Info("")
}

func checkEFSProtectionStatus(ctx context.Context, ar runner.AWSRunner, region, arn string) bool {
	if ar != nil {
		protected, err := ar.BackupIsResourceProtected(ctx, region, arn)
		if err != nil {
			return false
		}
		return protected
	}
	cnt, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", arn),
		"--output", "text")
	if err != nil {
		return false
	}
	return strings.TrimSpace(cnt) == "1"
}

func getLatestRecoveryPoint(ctx context.Context, ar runner.AWSRunner, region, arn string) string {
	if ar != nil {
		rps, err := ar.BackupListRecoveryPointsByResource(ctx, region, arn)
		if err != nil || len(rps) == 0 {
			return ""
		}
		latest := rps[0].CreationDate
		for _, rp := range rps[1:] {
			if rp.CreationDate.After(latest) {
				latest = rp.CreationDate
			}
		}
		return latest.Format(time.RFC3339)
	}
	rp, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--region", region,
		"--resource-arn", arn,
		"--query", "max_by(RecoveryPoints,&CreationDate).CreationDate",
		"--output", "text")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(rp)
}

func getBackupVaults(ctx context.Context, ar runner.AWSRunner, region, arn string) []string {
	if ar != nil {
		rps, err := ar.BackupListRecoveryPointsByResource(ctx, region, arn)
		if err != nil {
			return nil
		}
		seen := map[string]struct{}{}
		var unique []string
		for _, rp := range rps {
			if _, ok := seen[rp.BackupVaultName]; !ok {
				seen[rp.BackupVaultName] = struct{}{}
				unique = append(unique, rp.BackupVaultName)
			}
		}
		return unique
	}
	vaultsJSON, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-recovery-points-by-resource",
		"--region", region,
		"--resource-arn", arn,
		"--query", "RecoveryPoints[].BackupVaultName",
		"--output", "json")
	if err != nil {
		return nil
	}
	vaultsJSON = strings.TrimSpace(vaultsJSON)
	if vaultsJSON == "" || vaultsJSON == "null" || vaultsJSON == "[]" {
		return nil
	}
	var names []string
	if err := json.Unmarshal([]byte(vaultsJSON), &names); err != nil {
		return nil
	}
	seen := map[string]struct{}{}
	var unique []string
	for _, n := range names {
		if _, ok := seen[n]; !ok {
			seen[n] = struct{}{}
			unique = append(unique, n)
		}
	}
	return unique
}

// getBackupPlanInfo retrieves comprehensive backup plan information from AWS Backup
// Returns: (schedule, nextBackupTime, expiryDate, error)
func getBackupPlanInfo(ctx context.Context, ar runner.AWSRunner, region, namespace, latestRecoveryPoint string) (string, string, string, error) {
	planName := fmt.Sprintf("%s-backup-plan", namespace)

	var scheduleExpr string
	var deleteAfterDays int

	if ar != nil {
		plans, err := ar.BackupListBackupPlans(ctx, region)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to list backup plans: %w", err)
		}
		var planId string
		for _, p := range plans {
			if p.BackupPlanName == planName {
				planId = p.BackupPlanID
				break
			}
		}
		if planId == "" {
			return "", "", "", fmt.Errorf("no backup plan found with name '%s'", planName)
		}
		detail, err := ar.BackupGetBackupPlan(ctx, region, planId)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to get backup plan details (plan ID: %s): %w", planId, err)
		}
		if len(detail.Rules) == 0 {
			return "", "", "", fmt.Errorf("backup plan '%s' has no rules configured", planName)
		}
		scheduleExpr = detail.Rules[0].ScheduleExpression
		deleteAfterDays = detail.Rules[0].DeleteAfterDays
	} else {
		plansJSON, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-backup-plans",
			"--region", region,
			"--query", fmt.Sprintf("BackupPlansList[?BackupPlanName=='%s']", planName),
			"--output", "json")
		if err != nil {
			return "", "", "", fmt.Errorf("failed to list backup plans: %w", err)
		}
		plansJSON = strings.TrimSpace(plansJSON)
		if plansJSON == "" || plansJSON == "null" || plansJSON == "[]" {
			return "", "", "", fmt.Errorf("no backup plan found with name '%s'", planName)
		}
		var plans []struct {
			BackupPlanId   string `json:"BackupPlanId"`
			BackupPlanName string `json:"BackupPlanName"`
		}
		if err := json.Unmarshal([]byte(plansJSON), &plans); err != nil {
			return "", "", "", fmt.Errorf("failed to parse backup plans JSON: %w", err)
		}
		if len(plans) == 0 {
			return "", "", "", fmt.Errorf("backup plan list is empty for '%s'", planName)
		}
		planId := plans[0].BackupPlanId
		planDetails, err := utils.ExecuteCommand(ctx, "aws", "backup", "get-backup-plan",
			"--region", region,
			"--backup-plan-id", planId,
			"--output", "json")
		if err != nil {
			return "", "", "", fmt.Errorf("failed to get backup plan details (plan ID: %s): %w", planId, err)
		}
		var planInfo struct {
			BackupPlan struct {
				Rules []struct {
					ScheduleExpression string `json:"ScheduleExpression"`
					Lifecycle          *struct {
						DeleteAfterDays int `json:"DeleteAfterDays"`
					} `json:"Lifecycle"`
				} `json:"Rules"`
			} `json:"BackupPlan"`
		}
		if err := json.Unmarshal([]byte(planDetails), &planInfo); err != nil {
			return "", "", "", fmt.Errorf("failed to parse backup plan details JSON: %w", err)
		}
		if len(planInfo.BackupPlan.Rules) == 0 {
			return "", "", "", fmt.Errorf("backup plan '%s' has no rules configured", planName)
		}
		scheduleExpr = planInfo.BackupPlan.Rules[0].ScheduleExpression
		if planInfo.BackupPlan.Rules[0].Lifecycle != nil {
			deleteAfterDays = planInfo.BackupPlan.Rules[0].Lifecycle.DeleteAfterDays
		}
	}

	humanSchedule := parseCronToHuman(scheduleExpr)
	nextBackup := calculateNextBackupTime(scheduleExpr)

	var lifecycle *struct {
		DeleteAfterDays int `json:"DeleteAfterDays"`
	}
	if deleteAfterDays > 0 {
		lifecycle = &struct {
			DeleteAfterDays int `json:"DeleteAfterDays"`
		}{DeleteAfterDays: deleteAfterDays}
	}
	expiryDate := calculateExpiryDateFromPlan(lifecycle, latestRecoveryPoint)

	return humanSchedule, nextBackup, expiryDate, nil
}

// parseCronToHuman converts AWS Backup cron expression to human-readable format
// AWS Backup cron format: cron(minute hour day-of-month month day-of-week year)
// Example: cron(0 3 * * ? *) = daily at 03:00 UTC
func parseCronToHuman(cronExpr string) string {

	if strings.HasPrefix(cronExpr, "cron(") && strings.HasSuffix(cronExpr, ")") {
		// Remove cron() wrapper
		cron := strings.TrimPrefix(strings.TrimSuffix(cronExpr, ")"), "cron(")
		parts := strings.Fields(cron)

		if len(parts) >= 2 {
			minute := parts[0]
			hour := parts[1]

			// Convert to human-readable format
			if minute == "0" {
				return fmt.Sprintf("Daily at %s:00 UTC", hour)
			} else {
				return fmt.Sprintf("Daily at %s:%s UTC", hour, minute)
			}
		}
	}

	return "Custom schedule"
}

// calculateNextBackupTime calculates the next backup time based on cron expression
// For simplicity, assumes daily backup and calculates next occurrence
// Note: This is a basic implementation - for production, consider using a proper cron parser
func calculateNextBackupTime(cronExpr string) string {

	now := time.Now().UTC()
	// Get today's date at 00:00:00 UTC
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Extract hour and minute from cron expression
	if strings.HasPrefix(cronExpr, "cron(") && strings.HasSuffix(cronExpr, ")") {
		cron := strings.TrimPrefix(strings.TrimSuffix(cronExpr, ")"), "cron(")
		parts := strings.Fields(cron)

		if len(parts) >= 2 {
			minute := parts[0]
			hour := parts[1]

			// Parse hour and minute
			if h, err := strconv.Atoi(hour); err == nil {
				if m, err := strconv.Atoi(minute); err == nil {
					// Calculate today's backup time
					nextBackup := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, time.UTC)

					// If the time has passed today, schedule for tomorrow
					if nextBackup.Before(now) || nextBackup.Equal(now) {
						nextBackup = nextBackup.AddDate(0, 0, 1)
					}

					return nextBackup.Format("2006-01-02 15:04 UTC")
				}
			}
		}
	}

	// Fallback to next day at 03:00 UTC
	tomorrow := today.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 3, 0, 0, 0, time.UTC).Format("2006-01-02 15:04 UTC")
}

// calculateExpiryDateFromPlan calculates expiry date based on backup plan lifecycle policy
// Takes the lifecycle policy and latest recovery point creation date to calculate expiry
func calculateExpiryDateFromPlan(lifecycle *struct {
	DeleteAfterDays int `json:"DeleteAfterDays"`
}, latestRecoveryPoint string) string {
	// If no lifecycle policy or DeleteAfterDays is 0, retention is unlimited
	if lifecycle == nil || lifecycle.DeleteAfterDays == 0 {
		return "None (unlimited retention)"
	}

	// If no latest recovery point, cannot calculate expiry
	if latestRecoveryPoint == "" || latestRecoveryPoint == "None" {
		return "Unknown (no recovery points)"
	}

	// Parse the latest recovery point creation date
	creationTime, err := time.Parse(types.TimeFormatISO8601, latestRecoveryPoint)
	if err != nil {
		// Try KST format as fallback
		creationTime, err = time.Parse(types.TimeFormatISO8601KST, latestRecoveryPoint)
	}
	if err != nil {
		return "Unknown (invalid date format)"
	}

	// Calculate expiry date by adding retention days
	expiryTime := creationTime.AddDate(0, 0, lifecycle.DeleteAfterDays)
	return fmt.Sprintf("%s (%d days from creation)", expiryTime.Format("2006-01-02 15:04 UTC"), lifecycle.DeleteAfterDays)
}
