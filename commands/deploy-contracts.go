package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/urfave/cli/v3"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)
		rollupConfig := cmd.String(flags.RollupConfigFlag.Name)
		amount := float64(cmd.Float(flags.AmountFlag.Name))
		useTon := cmd.Bool(flags.UseTonFlag.Name)
		memo := cmd.String(flags.MemoFlag.Name)

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(network, stack)

			return thanosStack.DeployContracts(rollupConfig, amount, useTon, memo)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
