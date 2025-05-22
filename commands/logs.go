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

func ActionShowLogs() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		var network, stack string
		config, err := utils.ReadConfigFromJSONFile()
		if err != nil {
			fmt.Println("Error reading settings.json")
			return err
		}

		component := cmd.String("component")
		isTroubleshoot := cmd.Bool("troubleshoot")

		if config == nil {
			network = constants.LocalDevnet
			stack = constants.ThanosStack
		} else {
			network = config.Network
			stack = config.Stack
		}
		return ShowLogs(ctx, network, stack, component, isTroubleshoot, config)
	}
}

func ShowLogs(ctx context.Context, network, stack string, component string, isTroubleshoot bool, config *types.Config) error {
	// Initialize the logger
	fileName := fmt.Sprintf("logs/show_logs_%s_%s_%d.log", stack, network, time.Now().Unix())
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
		return thanosStack.ShowLogs(ctx, config, component, isTroubleshoot)
	}

	return nil
}
