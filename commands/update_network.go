package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionUpdateNetwork() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		var network, stack string
		config, err := utils.ReadConfigFromJSONFile()
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
		return UpdateNetwork(ctx, network, stack, config)
	}
}

func UpdateNetwork(ctx context.Context, network, stack string, config *types.Config) error {
	if network == constants.LocalDevnet {
		fmt.Println("You are using the local devnet. No need to update the network.")
		return nil
	}

	// Initialize the logger
	fileName := fmt.Sprintf("logs/update_network_%s_%s_%d.log", stack, network, time.Now().Unix())
	l := logging.InitLogger(fileName)

	switch stack {
	case constants.ThanosStack:
		var awsProfile *types.AWSProfile
		var err error
		if network == constants.Testnet || network == constants.Mainnet {
			awsProfile, err = aws.LoginAWS(ctx, config)
			if err != nil {
				fmt.Println("Error logging into AWS")
				return err
			}
		}

		thanosStack := thanos.NewThanosStack(l, network, stack, config, awsProfile, true)
		err = thanosStack.GetUpdateNetworkParams(ctx)
		if err != nil {
			fmt.Println("Error getting update network parameters")
			return err
		}
		return thanosStack.UpdateNetwork(ctx)
	}

	return nil
}
