package commands

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ActionBackupManager provides backup/restore operations for EFS and RDS
func ActionBackupManager() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// CWD
		deploymentPath, err := os.Getwd()
		if err != nil {
			return err
		}

		// Flags
		showStatus := cmd.Bool("status")
		startBackup := cmd.Bool("snapshot")
		listPoints := cmd.Bool("list")
		doRestore := cmd.Bool("restore")
		doConfigure := cmd.Bool("config")

		// target flags removed; manager operates on both EFS and RDS

		limitStr := cmd.String("limit")

		// post-restore attach flags
		doAttach := cmd.Bool("attach")
		attachEfs := cmd.String("efs-id")
		attachPVCs := cmd.String("pvc")
		attachSTSs := cmd.String("sts")

		// Configure flags (module/Terraform path)
		// EFS
		cfgDaily := cmd.String("daily")
		cfgKeep := cmd.String("keep")
		cfgReset := cmd.Bool("reset")
		// RDS
		cfgRdsKeep := cmd.String("keep-rds")
		cfgRdsWindow := cmd.String("backup-window")

		// Load settings.json (needed before logger for stack/network naming)
		config, err := utils.ReadConfigFromJSONFile(deploymentPath)
		if err != nil || config == nil || config.AWS == nil || config.K8s == nil {
			return fmt.Errorf("failed to read settings.json (ensure the L2 has been deployed): %w", err)
		}
		network := config.Network
		stack := config.Stack

		// Initialize the logger (stack, network in filename)
		now := time.Now().Unix()
		logFile := fmt.Sprintf("%s/logs/backup_manager_%s_%s_%d.log", deploymentPath, network, stack, now)
		l, err := logging.InitLogger(logFile)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		// Create ThanosStack instance
		thanosStack, err := thanos.NewThanosStack(ctx, l, network, false, deploymentPath, config.AWS)
		if err != nil {
			return fmt.Errorf("failed to create ThanosStack instance: %w", err)
		}

		// Dispatch
		switch {
		case showStatus:
			return thanosStack.BackupStatus(ctx)
		case startBackup:
			return thanosStack.BackupSnapshot(ctx)
		case listPoints:
			return thanosStack.BackupList(ctx, limitStr)
		case doAttach:
			// standalone post-restore attach mode
			e := attachEfs
			pvcsArg := attachPVCs
			stssArg := attachSTSs
			return thanosStack.BackupAttach(ctx, &e, &pvcsArg, &stssArg)
		case doRestore:
			return thanosStack.BackupRestore(ctx, nil, nil, nil, nil)
		case doConfigure:
			// take addresses for optional pointer arguments
			daily := cfgDaily
			keep := cfgKeep
			rst := cfgReset
			rdsKeep := cfgRdsKeep
			win := cfgRdsWindow
			return thanosStack.BackupConfigure(ctx, nil, nil, &daily, &keep, nil, &rst, &rdsKeep, &win)
		default:
			return errors.New("no action specified. Try --status, --snapshot, --list, --restore, or --config")
		}
	}
}
