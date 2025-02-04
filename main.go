package main

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/commands"
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
				Action: func(ctx context.Context, cmd *cli.Command) error {
					fmt.Println("added task: ", cmd.Args().First())
					return nil
				},
			},
			{
				Name:    "dependencies",
				Aliases: []string{"a"},
				Usage:   "Dependencies",
				Commands: []*cli.Command{
					{
						Name:  "setup",
						Usage: "",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							fmt.Println("new task template: ", cmd.Args().First())
							return nil
						},
					},
					{
						Name:  "check",
						Usage: "remove an existing template",
						Action: func(ctx context.Context, cmd *cli.Command) error {
							dependenciesCmd := commands.Dependencies{}
							return dependenciesCmd.Check(cmd.Args().Slice())
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
