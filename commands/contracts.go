package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)

		config, err := utils.ReadConfigFromJSONFile()
		if err != nil {
			fmt.Println("Error reading settings.json")
			return err
		}

		// Initialize the logger
		fileName := fmt.Sprintf("logs/deploy_contracts_%s_%s_%d.log", stack, network, time.Now().Unix())
		logging.InitLogger(fileName)

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(network, stack, config, nil, true)
			// STEP 1. Input the parameters
			fmt.Println("You are about to deploy the L1 contracts.")
			deployContractsConfig, err := thanosStack.InputDeployContracts(ctx)
			if err != nil {
				return err
			}
			return thanosStack.DeployContracts(ctx, deployContractsConfig)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
