package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"

	"github.com/urfave/cli/v3"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file:", err)
	}

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
				Name:   "version",
				Usage:  "SDK version",
				Action: commands.ActionVersion(),
			},
			{
				Name:   "verify-register-candidate",
				Usage:  "Verify and Register Candidate",
				Flags:  flags.VerifyRegisterCandidateFlag,
				Action: commands.ActionVerifyRegisterCandidates(),
			},
			{
				Name:   "serve",
				Usage:  "Start the HTTP server",
				Flags:  flags.StartServerFlag,
				Action: commands.ActionStartServer(),
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
