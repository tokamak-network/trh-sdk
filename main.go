package main

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
	"log"
	"os"
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
				Name:  "deploy-contracts",
				Usage: "Deploy contracts",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("added task: ", cmd.Args().First())
					return nil
				},
			},
			{
				Name:  "deploy",
				Usage: "Deploy infrastructure",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "network",
						Usage: "Network to deploy to",
						Value: "",
					},
					&cli.StringFlag{
						Name:  "stack",
						Usage: "Tech stack",
						Value: "",
					},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					var err error
					network := cmd.String("network")
					stack := cmd.String("stack")
					if network == "" {
						fmt.Print("Input network(local-devnet, testnet, mainnet): ")
						network, err = scanner.ScanString()
						if err != nil {
							fmt.Println("Error parsing the network: ", err)
							return err
						}

						if !constants.SupportedNetworks[network] {
							return fmt.Errorf("unsupported network: %s", network)
						}
					}
					if stack == "" {
						fmt.Print("Input stack(thanos): ")
						stack, err = scanner.ScanString()
						if err != nil {
							fmt.Println("Error parsing the stack: ", err)
							return err
						}

						if !constants.SupportedStacks[stack] {
							return fmt.Errorf("unsupported stack: %s", stack)
						}
					}

					deployCommand := commands.Deploy{
						Network: network,
						Stack:   stack,
					}

					return deployCommand.Execute()
				},
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
							choose, err := utils.ScanBool()
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
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
