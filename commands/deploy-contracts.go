package commands

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)

		switch stack {
		case constants.ThanosStack:
			thanosStack := NewThanosStack(network)

			return thanosStack.DeployContracts()
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
