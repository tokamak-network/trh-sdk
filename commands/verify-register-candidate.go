package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/urfave/cli/v3"
)

func ActionVerifyRegisterCandidates() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(network, stack)

			return thanosStack.VerifyRegisterCandidates(ctx, false)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
