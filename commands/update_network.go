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

func ActionUpdateNetwork() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var network, stack string

		var config *types.Config

		var awsConfig *types.AWSConfig

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
			awsConfig = config.AWS
		}

		if network == constants.LocalDevnet {
			fmt.Println("You are in local devnet mode. Please specify the network and stack.")
			return nil
		}

		if awsConfig == nil {
			awsConfig, err = thanos.InputAWSLogin()
			if err != nil {
				fmt.Printf("Failed to login AWS: %s \n", err)
				return err
			}
		}

		// Initialize the logger
		fileName := fmt.Sprintf("%s/logs/update_network_%s_%s_%d.log", deploymentPath, stack, network, time.Now().Unix())
		l := logging.InitLogger(fileName)

		switch stack {
		case constants.ThanosStack:
			thanosStack, err := thanos.NewThanosStack(l, network, true, deploymentPath, awsConfig)
			if err != nil {
				fmt.Println("Failed to initialize thanos stack", "err", err)
				return err
			}
			inputs, err := thanos.GetUpdateNetworkInputs(ctx)
			if err != nil {
				fmt.Println("Error getting update network inputs")
				return err
			}

			if inputs == nil {
				return nil
			}

			return thanosStack.UpdateNetwork(ctx, inputs)
		}

		return nil
	}
}
