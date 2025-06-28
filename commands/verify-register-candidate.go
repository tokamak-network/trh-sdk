package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionVerifyRegisterCandidates() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var network, stack string

		var config *types.Config

		// Retrieve the current working directory
		deploymentPath, err := os.Getwd()
		if err != nil {
			return err
		}

		config, err = utils.ReadConfigFromJSONFile(deploymentPath)
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

		if !constants.SupportedStacks[stack] {
			return fmt.Errorf("unsupported stack: %s", stack)
		}

		if !constants.SupportedNetworks[network] {
			return fmt.Errorf("unsupported network: %s", network)
		}

		switch config.Stack {
		case constants.ThanosStack:
			if config.Network == constants.Mainnet {
				return fmt.Errorf("register candidates verification is not supported on Mainnet")
			}

			// Initialize the logger
			now := time.Now().Unix()
			fileName := fmt.Sprintf("%s/logs/verify_register_candidate_%s_%s_%d.log", deploymentPath, config.Network, config.Stack, now)
			l, err := logging.InitLogger(fileName)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			thanosStack, err := thanos.NewThanosStack(ctx, l, config.Network, true, deploymentPath, config.AWS)
			if err != nil {
				return fmt.Errorf("failed to create thanos stack: %v", err)
			}

			registerCandidate, err := thanosStack.InputRegisterCandidate()
			if err != nil {
				return fmt.Errorf("❌ failed to get register candidate input: %w", err)
			}
			err = thanosStack.VerifyRegisterCandidates(ctx, registerCandidate)
			return err
		default:
			return fmt.Errorf("unsupported stack: %s", config.Stack)
		}
	}
}
