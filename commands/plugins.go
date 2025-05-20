package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

func ActionInstallationPlugins() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		var network, stack string
		config, err := utils.ReadConfigFromJSONFile()
		if err != nil {
			fmt.Println("Error reading settings.json")
			return err
		}

		if config == nil {
			network = constants.LocalDevnet
			stack = constants.ThanosStack
		} else {
			network = config.Network
			stack = config.Stack
		}

		plugins := cmd.Args().Slice()
		if len(plugins) == 0 {
			fmt.Print("Please specify at least one plugin to install(e.g: bridge)")
			return nil
		}

		if cmd.Name == "install" {
			switch stack {
			case constants.ThanosStack:
				thanosStack := thanos.NewThanosStack(network, stack, config)
				return thanosStack.InstallPlugins(ctx, plugins)
			default:
				return nil
			}
		} else if cmd.Name == "uninstall" {
			switch stack {
			case constants.ThanosStack:
				thanosStack := thanos.NewThanosStack(network, stack, config)
				return thanosStack.UninstallPlugins(ctx, plugins)
			default:
				return nil
			}
		}
		return nil
	}
}
