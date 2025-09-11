package backup

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// SnapshotExecute triggers an on-demand EFS backup
func SnapshotExecute(ctx context.Context, l *zap.SugaredLogger, region, namespace string) error {
	accountID, err := utils.DetectAWSAccountID(ctx)
	if err != nil {
		return err
	}

	efsID, err := utils.DetectEFSId(ctx, namespace)
	if err != nil || strings.TrimSpace(efsID) == "" {
		l.Infof("📁 EFS: ❌ Not detected in cluster PVs: %v", err)
		return nil
	}

	arn := utils.BuildEFSArn(region, accountID, efsID)
	backupVaultName := fmt.Sprintf("%s-backup-vault", namespace)
	iamRoleArn := fmt.Sprintf("arn:aws:iam::%s:role/%s-backup-service-role", accountID, namespace)

	out, err := utils.ExecuteCommand(ctx, "aws", "backup", "start-backup-job",
		"--region", region,
		"--backup-vault-name", backupVaultName,
		"--resource-arn", arn,
		"--iam-role-arn", iamRoleArn,
	)
	if err != nil {
		l.Errorf("📁 EFS: ❌ Failed to start backup job: %v", err)
		l.Infof("   Backup vault: %s", backupVaultName)
		l.Infof("   IAM role: %s", iamRoleArn)
		return nil
	}

	jobID := strings.TrimSpace(out)
	l.Infof("📁 EFS: ✅ On-demand backup started successfully")
	l.Infof("   Job ID: %s", jobID)
	l.Infof("   Backup vault: %s", backupVaultName)
	return nil
}
