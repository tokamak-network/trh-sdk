package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/urfave/cli/v3"
)

func ActionInstallPlugins() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		var network, stack string
		config, err := types.ReadConfigFromJSONFile()
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

		return InstallPlugins(network, stack, plugins, config)
	}
}

func InstallPlugins(network, stack string, pluginNames []string, config *types.Config) error {

	switch stack {
	case constants.ThanosStack:
		thanosStack := thanos.NewThanosStack(network, stack)
		return thanosStack.InstallPlugins(pluginNames, config)
	}

	return nil
}
