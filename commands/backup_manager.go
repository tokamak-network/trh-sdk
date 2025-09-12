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
		return executeBackupManagerAction(ctx, thanosStack, flags)
	}
}

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

	// Post-restore attach options
	AttachEfs  string
	AttachPVCs string
	AttachSTSs string

	// Configure options
	ConfigDaily string
	ConfigKeep  string
	ConfigReset bool
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

// executeBackupManagerAction routes to the appropriate backup manager action
func executeBackupManagerAction(ctx context.Context, thanosStack *thanos.ThanosStack, flags *BackupManagerFlags) error {
	switch {
	case flags.ShowStatus:
		return thanosStack.BackupStatus(ctx)
	case flags.StartBackup:
		return thanosStack.BackupSnapshot(ctx)
	case flags.ListPoints:
		return thanosStack.BackupList(ctx, flags.Limit)
	case flags.DoAttach:
		return thanosStack.BackupAttach(ctx, &flags.AttachEfs, &flags.AttachPVCs, &flags.AttachSTSs)
	case flags.DoRestore:
		return thanosStack.BackupRestore(ctx)
	case flags.DoConfigure:
		return thanosStack.BackupConfigure(ctx, &flags.ConfigDaily, &flags.ConfigKeep, &flags.ConfigReset)
	default:
		return errors.New("no action specified. Try --status, --snapshot, --list, --restore, or --config")
	}
}
