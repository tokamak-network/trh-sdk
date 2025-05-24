package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/urfave/cli/v3"
)

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		stack := cmd.String(flags.StackFlag.Name)
		network := cmd.String(flags.NetworkFlag.Name)

		now := time.Now().Unix()
		deploymentPath := fmt.Sprintf("deployments/%s-%s-%d", stack, network, now)

		// Initialize the logger
		fileName := fmt.Sprintf("%s/logs/deploy_contracts_%s_%s_%d.log", deploymentPath, stack, network, now)
		l := logging.InitLogger(fileName)

		switch stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(l, network, stack, nil, nil, true, deploymentPath)
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
