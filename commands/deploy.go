package commands

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"

	"github.com/tokamak-network/trh-sdk/pkg/types"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type Deploy struct {
	Network string
	Stack   string
}

func Execute(ctx context.Context, network, stack string, config *types.Config) error {
	if !constants.SupportedStacks[stack] {
		return fmt.Errorf("unsupported stack: %s", stack)
	}

	if !constants.SupportedNetworks[network] {
		return fmt.Errorf("unsupported network: %s", network)
	}

	switch stack {
	case constants.ThanosStack:
		thanosStack := thanos.NewThanosStack(network, stack, config)
		err := thanosStack.Deploy(ctx)
		if err != nil {
			fmt.Println("Error deploying Thanos Stack")
			return err
		}
	}
	return nil
}

func ActionDeploy() cli.ActionFunc {
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
		return Execute(ctx, network, stack, config)
	}
}
