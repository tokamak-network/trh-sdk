package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/urfave/cli/v3"
)

func Run() {
	cmd := &cli.Command{
		Name:  "trh-sdk",
		Usage: "Ignite your own L2 development",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "backup-manager",
				Usage: "Manage L2 backups and restores (EFS)",
				Description: `Examples:
    trh-sdk backup-manager --status
    trh-sdk backup-manager --snapshot
    trh-sdk backup-manager --list --limit 5  # Show only 5 most recent backups (default: 20)
    trh-sdk backup-manager --restore
    trh-sdk backup-manager --config --daily 03:00 --keep 35
    trh-sdk backup-manager --attach --efs-id fs-1234567890abcdef0 --pvc op-geth,op-node --sts op-geth,op-node
    `,
				Flags:  flags.BackupManagerFlags,
				Action: commands.ActionBackupManager(),
			},
			{
				Name:   "deploy-contracts",
				Usage:  "Deploy contracts on L1",
				Flags:  flags.DeployContractsFlag,
				Action: commands.ActionDeployContracts(),
				Description: `Deploy contracts on L1

Examples:
  # Deploy contracts on L1 with registering candidate
  trh-sdk deploy-contracts --network testnet --stack thanos 

  # Deploy contracts on L1 without registering candidate
  trh-sdk deploy-contracts --network testnet --stack thanos --no-candidate
  `,
			},
			{
				Name:   "deploy",
				Usage:  "Deploy infrastructure and bring up the L2 network",
				Action: commands.ActionDeploy(),
				Description: `Deploy infrastructure and bring up the L2 network. If you want to deploy the devnet network, you can skip the deployment of contracts step.

Examples:
  # Deploy infrastructure and bring up the L2 testnet network
  trh-sdk deploy --network testnet --stack thanos && trh-sdk deploy

  # Deploy infrastructure and bring up the L2 devnet network
  trh-sdk deploy
  `,
			},
			{
				Name:   "destroy",
				Usage:  "Destroy deployed infrastructure and bring down the L2 network",
				Action: commands.ActionDestroyInfra(),
				Description: `Destroy deployed infrastructure and bring down the L2 network.

Examples:
  # Destroy infrastructure and bring down the L2 network(testnet, mainnet and devnet)
  trh-sdk destroy
`,
			},
			{
				Name:  "install",
				Usage: fmt.Sprintf("Install plugins(allowed: %s)", strings.Join(constants.SupportedPluginsList, ", ")),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "type",
						Usage: "Plugin type (e.g. drb: leader|regular, cross-trade)",
						Value: "",
					},
				},
				Action: commands.ActionInstallationPlugins(),
				Description: `Install plugins

Examples:
  # Install block-explorer and bridge plugins
  trh-sdk install block-explorer bridge
  
  # Install DRB leader node
  trh-sdk install drb --type leader
  
  # Install DRB regular node
  trh-sdk install drb --type regular
  `,
			},
			{
				Name:  "uninstall",
				Usage: fmt.Sprintf("Uninstall plugins(allowed: %s)", strings.Join(constants.SupportedPluginsList, ", ")),
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "type",
						Usage: "Optional type of the plugin (e.g. drb: leader|regular)",
						Value: "",
					},
				},
				Action: commands.ActionInstallationPlugins(),
				Description: `Uninstall plugins

Examples:
  # Uninstall block-explorer and bridge plugins
  trh-sdk uninstall block-explorer bridge

  # Uninstall DRB leader node
  trh-sdk uninstall drb --type leader
  
  # Uninstall DRB regular node
  trh-sdk uninstall drb --type regular
  `,
			},
			{
				Name:   "version",
				Usage:  "Show SDK version",
				Action: commands.ActionVersion(),
				Description: `Show SDK version

Examples:
  # Show SDK version
  trh-sdk version
  `,
			},
			{
				Name:   "info",
				Usage:  "Show information about the running chain",
				Action: commands.ActionShowInformation(),
				Description: `Get information about the running chain

Examples:
  # Get information about the running chain
  trh-sdk info
  `,
			},
			{
				Name:  "logs",
				Usage: "Show logs of the running chain",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "component",
						Aliases:  []string{"c"},
						Required: true,
						Usage:    fmt.Sprintf("Component name (allowed: %s)", strings.Join(allowedComponentList(), ", ")),
						Value:    "",
					},
					&cli.BoolFlag{
						Name:     "troubleshoot",
						Aliases:  []string{"t"},
						Required: false,
						Usage:    "Show logs of the running chain with troubleshoot mode",
					},
				},
				Action: commands.ActionShowLogs(),
				Description: `Show logs of the running chain

Examples:
  # Show logs of the running chain
  trh-sdk logs --component op-node --troubleshoot
  `,
			},
			{
				Name:   "update",
				Usage:  "Update the config of the running chain",
				Action: commands.ActionUpdateNetwork(),
				Description: `Update the config of the running chain

Examples:
  # Update the config of the running chain
  trh-sdk update
  `,
			},
			{
				Name:   "upgrade",
				Usage:  "Upgrade the trh-sdk latest version",
				Action: commands.ActionUpgrade(),
				Description: `Upgrade the trh-sdk latest version

Examples:
  # Upgrade the trh-sdk latest version
  trh-sdk upgrade
  `,
			},
			{
				Name:   "verify-register-candidate",
				Usage:  "Verify and Register Candidate",
				Action: commands.ActionVerifyRegisterCandidates(),
				Description: `Verify and Register Candidate

Examples:
  # Verify and Register Candidate
  trh-sdk verify-register-candidate
  `,
			},
			{
				Name:  "alert-config",
				Usage: "Customize alert notifications and rules",
				Description: `Examples:
  # Check alert status and rules
  trh-sdk alert-config --status

  # Interactive rule configuration
  trh-sdk alert-config --rule set

  # Disable email channel
  trh-sdk alert-config --channel email --disable

  # Configure email channel
  trh-sdk alert-config --channel email --configure

  # Disable telegram channel
  trh-sdk alert-config --channel telegram --disable

  # Configure telegram channel
  trh-sdk alert-config --channel telegram --configure

  # Reset all alert rules to default values
  trh-sdk alert-config --rule reset`,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:     "status",
						Aliases:  []string{"s"},
						Required: false,
						Usage:    "Show current alert status",
					},
					&cli.StringFlag{
						Name:     "channel",
						Aliases:  []string{"c"},
						Required: false,
						Usage:    "Channel type (email, telegram)",
						Value:    "",
					},
					&cli.BoolFlag{
						Name:     "disable",
						Required: false,
						Usage:    "Disable the specified channel",
					},
					&cli.BoolFlag{
						Name:     "configure",
						Required: false,
						Usage:    "Configure the specified channel",
					},

					&cli.StringFlag{
						Name:     "rule",
						Required: false,
						Usage:    "Rule action (reset, set)",
						Value:    "",
					},
				},
				Action: commands.ActionAlertConfig(),
			},
			{
				Name:  "log-collection",
				Usage: "Manage CloudWatch logging settings and download logs",
				Description: `Manage CloudWatch logging settings and download logs for AWS CLI sidecar log collection.

Options:
  --enable                    Enable CloudWatch log collection
  --disable                   Disable CloudWatch log collection
  --retention <days>          Set CloudWatch log retention period in days (e.g., 7, 30, 90)
  --interval <seconds>        Set log collection interval in seconds (e.g., 30, 60, 120)
  --show                      Show current logging configuration

Subcommand Download Options:
  --download                  Download logs from running components
  --component <name>          Component to download logs from (op-node, op-geth, op-batcher, op-proposer, all)
  --hours <number>            Number of hours to look back for logs
  --minutes <number>          Number of minutes to look back for logs
  --keyword <text>            Keyword to filter logs (case-insensitive)

Examples:
  # Log configuration
  trh-sdk log-collection --enable
  trh-sdk log-collection --retention 30
  trh-sdk log-collection --interval 60
  trh-sdk log-collection --show

  # Log download from running components
  trh-sdk log-collection --download --component op-node --hours 7
  trh-sdk log-collection --download --component all --hours 24 --keyword error
  trh-sdk log-collection --download --component op-geth --minutes 30 --keyword warning

  # Apply all settings at once
  trh-sdk log-collection --enable --retention 90 --interval 60`,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "enable",
						Usage: "Enable CloudWatch log collection",
					},
					&cli.BoolFlag{
						Name:  "disable",
						Usage: "Disable CloudWatch log collection",
					},
					&cli.StringFlag{
						Name:  "retention",
						Usage: "CloudWatch log retention period in days (e.g., 7, 30, 90)",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "interval",
						Usage: "Log collection interval in seconds (e.g., 30, 60, 120)",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "show",
						Usage: "Show current logging configuration",
					},
					&cli.BoolFlag{
						Name:  "download",
						Usage: "Download logs from running components",
					},

					&cli.StringFlag{
						Name:  "component",
						Usage: "Component to download logs from (op-node, op-geth, op-batcher, op-proposer, all)",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "hours",
						Usage: "Number of hours to look back for logs",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "minutes",
						Usage: "Number of minutes to look back for logs",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "keyword",
						Usage: "Keyword to filter logs (case-insensitive)",
						Value: "",
					},
				},
				Action: commands.ActionLogCollection(),
			},
			{
				Name:   "register-metadata",
				Usage:  "Register L2 Metadata",
				Action: commands.ActionRegisterMetadata(),
			},
			{
				Name:  "drb",
				Usage: "DRB (Distributed Random Beacon) commands",
				Commands: []*cli.Command{
					{
						Name:   "leader-info",
						Usage:  "Display DRB leader node connection information",
						Action: commands.ActionDisplayDRBLeaderInfo(),
						Description: `Display DRB leader node connection information from drb-leader-info.json

Examples:
  # Display leader node information
  trh-sdk drb leader-info
  `,
					},
				},
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func allowedComponentList() []string {
	components := make([]string, 0)
	for c := range thanos.SupportedLogsComponents {
		components = append(components, c)
	}
	return components
}
