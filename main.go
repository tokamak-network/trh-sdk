package main

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/commands"
	"github.com/urfave/cli/v3"
	"log"
	"os"
	"strings"
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("added task: ", cmd.Args().First())
					return nil
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
							var response string
							n, err := fmt.Scanln(&response)
							// Blank input, default No
							if n == 0 {
								fmt.Println("Installation skipped.")
								return nil
							}

							if strings.ToLower(response) != "n" && strings.ToLower(response) != "y" {
								return fmt.Errorf("Invalid input")
							}
							if err != nil {
								return err
							}

							if strings.ToLower(response) == "y" {
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
