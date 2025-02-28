package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/urfave/cli/v3"
)

func ActionDestroyInfra() cli.ActionFunc {
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
		return Destroy(network, stack, config)
	}
}

func Destroy(network, stack string, config *types.Config) error {

	switch stack {
	case constants.ThanosStack:
		thanosStack := thanos.NewThanosStack(network, stack)
		return thanosStack.Destroy(config)
	}

	return nil
}
