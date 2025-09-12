package flags

import "github.com/urfave/cli/v3"

const envPrefix = "TRH_SDK"

func PrefixEnvVars(prefix, name string) []string {
	return []string{prefix + "_" + name}
}

var (
	StackFlag = &cli.StringFlag{
		Name:     "stack",
		Usage:    "Select stack(thanos)",
		Value:    "thanos",
		Sources:  cli.EnvVars(PrefixEnvVars(envPrefix, "STACK")...),
		Required: true,
	}

	NetworkFlag = &cli.StringFlag{
		Name:    "network",
		Usage:   "Select Network Environment [testnet, mainnet]",
		Value:   "testnet",
		Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "NETWORK")...),
	}

	NoCandidateFlag = &cli.BoolFlag{
		Name:    "no-candidate",
		Usage:   "Skip candidate registration after contract deployment",
		Value:   false,
		Sources: cli.EnvVars(PrefixEnvVars(envPrefix, "NO_CANDIDATE")...),
	}
)

var DeployContractsFlag = []cli.Flag{
	StackFlag,
	NetworkFlag,
	NoCandidateFlag,
}

var (
	// BackupManager primary action flags
	BackupStatusFlag = &cli.BoolFlag{
		Name:  "status",
		Usage: "Show backup protection status and latest points",
	}

	BackupSnapshotFlag = &cli.BoolFlag{
		Name:  "snapshot",
		Usage: "Create an on-demand backup for EFS",
	}

	BackupListFlag = &cli.BoolFlag{
		Name:  "list",
		Usage: "List recovery points",
	}

	BackupRestoreFlag = &cli.BoolFlag{
		Name:  "restore",
		Usage: "Interactive restore chain data",
	}

	BackupConfigFlag = &cli.BoolFlag{
		Name:  "config",
		Usage: "Configure backup settings",
	}

	BackupAttachFlag = &cli.BoolFlag{
		Name:  "attach",
		Usage: "Attach workloads to a new EFS and verify (can be used after restore or standalone)",
	}

	// BackupManager common option flags
	BackupLimitFlag = &cli.StringFlag{
		Name:  "limit",
		Usage: "Limit number of entries when listing (default: 20)",
	}

	// BackupManager attach option flags
	BackupEfsIdFlag = &cli.StringFlag{
		Name:  "efs-id",
		Usage: "New EFS FileSystemId (fs-xxxx) to switch to",
	}

	BackupPvcFlag = &cli.StringFlag{
		Name:  "pvc",
		Usage: "Comma-separated PVC names to switch (e.g., op-geth,op-node)",
	}

	BackupStsFlag = &cli.StringFlag{
		Name:  "sts",
		Usage: "Comma-separated StatefulSet names to restart and verify",
	}

	// BackupManager config option flags
	BackupDailyFlag = &cli.StringFlag{
		Name:  "daily",
		Usage: "Daily time (UTC) e.g. 03:00 (converted to cron)",
	}

	BackupKeepFlag = &cli.StringFlag{
		Name:  "keep",
		Usage: "EFS keep days (retention)",
	}

	BackupResetFlag = &cli.BoolFlag{
		Name:  "reset",
		Usage: "Reset to defaults (EFS daily 03:00, unlimited keep)",
	}
)

var BackupManagerFlags = []cli.Flag{
	// primary actions
	BackupStatusFlag,
	BackupSnapshotFlag,
	BackupListFlag,
	BackupRestoreFlag,
	BackupConfigFlag,
	BackupAttachFlag,

	// list options
	BackupLimitFlag,

	// attach options (EFS)
	BackupEfsIdFlag,
	BackupPvcFlag,
	BackupStsFlag,

	// config options
	BackupDailyFlag,
	BackupKeepFlag,
	BackupResetFlag,
}
