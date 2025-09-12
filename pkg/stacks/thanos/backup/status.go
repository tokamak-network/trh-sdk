package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// GatherBackupStatusInfo collects backup status information using region/namespace
func GatherBackupStatusInfo(ctx context.Context, region, namespace string) (*types.BackupStatusInfo, error) {
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
	statusInfo.IsProtected = checkEFSProtectionStatus(ctx, region, statusInfo.ARN)
	statusInfo.LatestRecoveryPoint = getLatestRecoveryPoint(ctx, region, statusInfo.ARN)

	if statusInfo.LatestRecoveryPoint != "" && statusInfo.LatestRecoveryPoint != "None" {
		// Only calculate expiry date if retention is not unlimited
		if types.DefaultBackupRetentionDays > 0 {
			statusInfo.ExpectedExpiryDate = calculateExpectedExpiryDate(statusInfo.LatestRecoveryPoint)
		}
	}

	statusInfo.BackupVaults = getBackupVaults(ctx, region, statusInfo.ARN)

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
		l.Infof("   Latest recovery point: ✅ %s", statusInfo.LatestRecoveryPoint)
		if types.DefaultBackupRetentionDays == 0 {
			l.Infof("   Expected expiry date: 📅 None (unlimited retention)")
		} else if statusInfo.ExpectedExpiryDate != "" {
			l.Infof("   Expected expiry date: 📅 %s (%d days from creation)", statusInfo.ExpectedExpiryDate, types.DefaultBackupRetentionDays)
		}
	}

	// Handle backup vaults display
	if len(statusInfo.BackupVaults) == 0 {
		l.Warn("   Vaults: ⚠️  None (no backups found)")
	} else {
		l.Infof("   Vaults: ✅ %s", strings.Join(statusInfo.BackupVaults, ", "))
	}

	l.Info("")
}

func checkEFSProtectionStatus(ctx context.Context, region, arn string) bool {
	cnt, err := utils.ExecuteCommand(ctx, "aws", "backup", "list-protected-resources",
		"--region", region,
		"--query", fmt.Sprintf("length(Results[?ResourceArn=='%s'])", arn),
		"--output", "text")
	if err != nil {
		return false
	}
	return strings.TrimSpace(cnt) == "1"
}

func getLatestRecoveryPoint(ctx context.Context, region, arn string) string {
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

func calculateExpectedExpiryDate(creationDate string) string {
	// If retention is unlimited (0 days), return empty string
	if types.DefaultBackupRetentionDays == 0 {
		return ""
	}

	creationTime, err := time.Parse(types.TimeFormatISO8601, creationDate)
	if err != nil {
		creationTime, err = time.Parse(types.TimeFormatISO8601KST, creationDate)
	}
	if err != nil {
		return ""
	}
	expiryTime := creationTime.AddDate(0, 0, types.DefaultBackupRetentionDays)
	return expiryTime.Format(types.TimeFormatISO8601)
}

func getBackupVaults(ctx context.Context, region, arn string) []string {
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
