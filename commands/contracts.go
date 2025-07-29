package commands

import (
	"context"
	"fmt"
	"os"
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
		enableRegisterCandidate := !cmd.Bool(flags.NoCandidateFlag.Name)

		now := time.Now().Unix()

		deploymentPath, err := os.Getwd()
		if err != nil {
			return err
		}

		// Initialize the logger
		fileName := fmt.Sprintf("%s/logs/deploy_contracts_%s_%s_%d.log", deploymentPath, stack, network, now)
		l, err := logging.InitLogger(fileName)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		switch stack {
		case constants.ThanosStack:
			thanosStack, err := thanos.NewThanosStack(ctx, l, network, true, deploymentPath, nil)
			if err != nil {
				fmt.Println("Failed to initialize thanos stack", "err", err)
				return err
			}
			// STEP 1. Input the parameters
			fmt.Println("You are about to deploy the L1 contracts.")
			deployContractsConfig, err := thanos.InputDeployContracts(ctx)
			if err != nil {
				return err
			}
			thanosStack.SetRegisterCandidate(enableRegisterCandidate)

			if enableRegisterCandidate {
				registerCandidateInputs, err := thanos.InputRegisterCandidate()
				if err != nil {
					return fmt.Errorf("❌ failed to get register candidate input: %w", err)
				}
				deployContractsConfig.RegisterCandidate = registerCandidateInputs
			}

			return thanosStack.DeployContracts(ctx, deployContractsConfig)
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
