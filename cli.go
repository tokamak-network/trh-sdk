package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
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
				Name:   "deploy-contracts",
				Usage:  "Deploy contracts on L1",
				Flags:  flags.DeployContractsFlag,
				Action: commands.ActionDeployContracts(),
			},
			{
				Name:   "deploy",
				Usage:  "Deploy infrastructure and bring up the L2 network",
				Action: commands.ActionDeploy(),
			},
			{
				Name:  "dependencies",
				Usage: "Check and install dependencies",

				Commands: []*cli.Command{
					{
						Name:  "setup",
						Usage: "Install the dependencies",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							fmt.Println("Install the dependencies...")
							fmt.Print("Would you like to install dependencies? (y/N): ")
							choose, err := scanner.ScanBool(false)
							if err != nil {
								return err
							}

							if choose {
								fmt.Println("Installing dependencies...")
								// Add installation logic here
								dependenciesCmd := commands.Dependencies{}
								return dependenciesCmd.Install(ctx, cmd.Args().Slice())

							} else {
								fmt.Println("Installation skipped.")
							}
							return nil
						},
					},
					{
						Name:  "check",
						Usage: "Check the dependencies",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							dependenciesCmd := commands.Dependencies{}
							dependenciesCmd.Check(ctx, cmd.Args().Slice())
							return nil
						},
					},
				},
			},
			{
				Name:   "destroy",
				Usage:  "Destroy deployed infrastructure and bring down the L2 network",
				Action: commands.ActionDestroyInfra(),
			},
			{
				Name:  "install",
				Usage: "Install plugins",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "plugins",
						Usage: "",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "stack",
						Usage: "Tech stack",
						Value: "",
					},
				},
				Action: commands.ActionInstallationPlugins(),
			},
			{
				Name:  "uninstall",
				Usage: "Uninstall plugins",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "plugins",
						Usage: "",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "stack",
						Usage: "Tech stack",
						Value: "",
					},
				},
				Action: commands.ActionInstallationPlugins(),
			},
			{
				Name:   "version",
				Usage:  "Show SDK version",
				Action: commands.ActionVersion(),
			},
			{
				Name:   "info",
				Usage:  "Show information about the running chain",
				Action: commands.ActionShowInformation(),
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
			},
			{
				Name:   "update",
				Usage:  "Update the config of the running chain",
				Action: commands.ActionUpdateNetwork(),
			},
			{
				Name:   "upgrade",
				Usage:  "Upgrade the trh-sdk latest version",
				Action: commands.ActionUpgrade(),
			},
			{
				Name:   "verify-register-candidate",
				Usage:  "Verify and Register Candidate",
				Action: commands.ActionVerifyRegisterCandidates(),
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
