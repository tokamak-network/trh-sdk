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

// BackupManagerFlags represents all command line flags for backup manager
type BackupManagerFlags struct {
	// Primary actions
	ShowStatus  bool
	StartBackup bool
	ListPoints  bool
	DoRestore   bool
	DoConfigure bool
	DoAttach    bool

	// Common options
	Limit string

	// Restore options
	RestoreArn string

	// Post-restore attach options
	AttachEfs  string
	AttachPVCs string
	AttachSTSs string

	// Configure options
	ConfigDaily string
	ConfigKeep  string
	ConfigReset bool
}

// ActionBackupManager provides backup/restore operations for EFS
func ActionBackupManager() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get deployment path
		deploymentPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		// Parse command flags
		flags := parseBackupManagerFlags(cmd)

		// Validate flags
		if err := validateBackupManagerFlags(flags); err != nil {
			return err
		}

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

		// Execute the appropriate action
		switch {
		case flags.ShowStatus:
			_, err := thanosStack.BackupStatus(ctx)
			return err

		case flags.StartBackup:
			_, err := thanosStack.BackupSnapshot(ctx, nil)
			return err

		case flags.ListPoints:
			_, err := thanosStack.BackupList(ctx, flags.Limit)
			return err

		case flags.DoAttach:
			backup := true
			reporter := func(msg string, pct float64) {
				fmt.Printf(">> [%.1f%%] %s\n", pct, msg)
			}
			_, err := thanosStack.BackupAttach(ctx, &flags.AttachEfs, &flags.AttachPVCs, &flags.AttachSTSs, &backup, reporter)
			return err

		case flags.DoRestore:
			return handleRestore(ctx, thanosStack, flags)

		case flags.DoConfigure:
			_, err := thanosStack.BackupConfigure(ctx, &flags.ConfigDaily, &flags.ConfigKeep, &flags.ConfigReset)
			return err

		default:
			return errors.New("no action specified. Try --status, --snapshot, --list, --restore, --config, or --attach")
		}
	}
}

// handleRestore manages the restore process with interactive or direct mode
func handleRestore(ctx context.Context, thanosStack *thanos.ThanosStack, flags *BackupManagerFlags) error {
	// If ARN is provided, use direct restore mode
	if flags.RestoreArn != "" {
		_, err := thanosStack.BackupRestore(ctx, flags.RestoreArn, nil, nil, nil, nil)
		return err
	}

	// Otherwise, use interactive mode (default)
	return thanosStack.BackupRestoreInteractive(ctx)
}

// parseBackupManagerFlags extracts and parses all command line flags
func parseBackupManagerFlags(cmd *cli.Command) *BackupManagerFlags {
	return &BackupManagerFlags{
		// Primary actions
		ShowStatus:  cmd.Bool("status"),
		StartBackup: cmd.Bool("snapshot"),
		ListPoints:  cmd.Bool("list"),
		DoRestore:   cmd.Bool("restore"),
		DoConfigure: cmd.Bool("config"),
		DoAttach:    cmd.Bool("attach"),

		// Common options
		Limit: cmd.String("limit"),

		// Restore options
		RestoreArn: cmd.String("recovery-point-arn"),

		// Post-restore attach options
		AttachEfs:  cmd.String("efs-id"),
		AttachPVCs: cmd.String("pvc"),
		AttachSTSs: cmd.String("sts"),

		// Configure options
		ConfigDaily: cmd.String("daily"),
		ConfigKeep:  cmd.String("keep"),
		ConfigReset: cmd.Bool("reset"),
	}
}

// validateBackupManagerFlags validates that exactly one action is specified
func validateBackupManagerFlags(flags *BackupManagerFlags) error {
	actions := []bool{
		flags.ShowStatus,
		flags.StartBackup,
		flags.ListPoints,
		flags.DoRestore,
		flags.DoConfigure,
		flags.DoAttach,
	}

	actionCount := 0
	for _, action := range actions {
		if action {
			actionCount++
		}
	}

	if actionCount == 0 {
		return errors.New("no action specified. Try --status, --snapshot, --list, --restore, --config, or --attach")
	}

	if actionCount > 1 {
		return errors.New("only one action can be specified at a time")
	}

	return nil
}
