package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)
		registerCandidate := !cmd.Bool(flags.NoCandidateFlag.Name)

		config, err := utils.ReadConfigFromJSONFile()
		if err != nil {
			fmt.Println("Error reading settings.json")
			return err
		}

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(network, stack, config)
			thanosStack.SetRegisterCandidate(registerCandidate)
			return thanosStack.DeployContracts(ctx)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
