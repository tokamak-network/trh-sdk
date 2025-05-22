package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type Deploy struct {
	Network string
	Stack   string
}

func Execute(ctx context.Context, network, stack string, config *types.Config) error {
	if !constants.SupportedStacks[stack] {
		return fmt.Errorf("unsupported stack: %s", stack)
	}

	if !constants.SupportedNetworks[network] {
		return fmt.Errorf("unsupported network: %s", network)
	}

	// Initialize the logger
	fileName := fmt.Sprintf("logs/deploy_%s_%s_%d.log", stack, network, time.Now().Unix())
	logging.InitLogger(fileName)

	switch stack {
	case constants.ThanosStack:
		var err error
		var awsProfile *types.AWSProfile
		var infraOpt string

		if network == constants.LocalDevnet {
			infraOpt = "localhost"
		} else {
			fmt.Print("Please select your infrastructure provider [AWS] (default: AWS): ")
			input, err := scanner.ScanString()
			if err != nil {
				fmt.Printf("Error reading infrastructure selection: %s", err)
				return err
			}
			infraOpt = strings.ToLower(input)
			if infraOpt == "" {
				infraOpt = constants.AWS
			}

			switch infraOpt {
			case constants.AWS:
				fmt.Println("You selected AWS as your infrastructure provider.")

				awsProfile, err = aws.LoginAWS(ctx, config)
				if err != nil {
					fmt.Println("Error logging into AWS")
					return err
				}

			default:
				fmt.Printf("Unsupported infrastructure provider: %s\n", infraOpt)
			}
		}

		thanosStack := thanos.NewThanosStack(network, stack, config, awsProfile, true)
		err = thanosStack.Deploy(ctx, infraOpt)
		if err != nil {
			fmt.Println("Error deploying Thanos Stack")
			return err
		}
	}
	return nil
}

func ActionDeploy() cli.ActionFunc {
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

		return Execute(ctx, network, stack, config)
	}
}
