package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/urfave/cli/v3"
)

type contextKey string

const (
	noCandidateKey contextKey = "no-candidate"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)
		noCandidate := cmd.Bool(flags.NoCandidateFlag.Name)

		// Use the custom key type
		ctxWithValue := context.WithValue(ctx, noCandidateKey, noCandidate)

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(network, stack)
			return thanosStack.DeployContracts(ctxWithValue)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
