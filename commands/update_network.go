package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionUpdateNetwork() cli.ActionFunc {
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
		return UpdateNetwork(ctx, network, stack, config)
	}
}

func UpdateNetwork(ctx context.Context, network, stack string, config *types.Config) error {
	if network == constants.LocalDevnet {
		fmt.Println("You are using the local devnet. No need to update the network.")
		return nil
	}

	switch stack {
	case constants.ThanosStack:
		thanosStack := thanos.NewThanosStack(network, stack, config)
		return thanosStack.UpdateNetwork(ctx)
	}

	return nil
}
