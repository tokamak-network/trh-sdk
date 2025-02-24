package main

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"log"
	"os"

	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/urfave/cli/v3"
)

func main() {

	cmd := &cli.Command{
		Name:  "tokamak-sdk-cli",
		Usage: "make an explosive entrance",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return nil
		},
		Commands: []*cli.Command{
			{
				Name:   "deploy-contracts",
				Usage:  "Deploy contracts",
				Flags:  flags.DeployContractsFlag,
				Action: commands.ActionDeployContracts(),
			},
			{
				Name:   "deploy",
				Usage:  "Deploy infrastructure",
				Action: commands.ActionDeploy(),
			},
			{
				Name:  "dependencies",
				Usage: "Dependencies",

				Commands: []*cli.Command{
					{
						Name:  "setup",
						Usage: "Install the dependencies",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							fmt.Println("Install the dependencies...")
							fmt.Print("Would you like to install dependencies? (y/N): ")
							choose, err := scanner.ScanBool()
							if err != nil {
								return err
							}

							if choose {
								fmt.Println("Installing dependencies...")
								// Add installation logic here
								dependenciesCmd := commands.Dependencies{}
								return dependenciesCmd.Install(cmd.Args().Slice())

							} else {
								fmt.Println("Installation skipped.")
							}
							return nil
						},
					},
					{
						Name:  "check",
						Usage: "remove an existing template",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							dependenciesCmd := commands.Dependencies{}
							dependenciesCmd.Check(cmd.Args().Slice())
							return nil
						},
					},
				},
			},
			{
				Name:   "destroy",
				Usage:  "Destroy infrastructure",
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
				Action: commands.ActionInstallPlugins(),
			},
			{
				Name:      "install",
				Usage:     "Install plugins",
				ArgsUsage: "[plugins...]",
				Action:    commands.ActionInstallPlugins(),
			},
			{
				Name:   "register-candidates",
				Usage:  "Register candidates",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:     "rpc-url",
						Usage:    "L1 RPC URL",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "rollup-config",
						Usage:    "Address of the rollup config contract",
						Required: true,
					},
					&cli.StringFlag{
						Name:     "private-key",
						Usage:    "Private key for transaction signing",
						Required: true,
					},
					&cli.Float64Flag{
						Name:     "amount",
						Usage:    "Amount of TON to stake (minimum 1000.1)",
						Required: true,
					},
					&cli.StringFlag{
						Name:  "memo",
						Usage: "Memo for the registration",
						Value: "",
					},
					&cli.BoolFlag{
						Name:  "use-ton",
						Usage: "Use TON instead of WTON for staking",
						Value: false,
					},
				},
				Action: commands.ActionRegisterCandidates(),
			}
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
