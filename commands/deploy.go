package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/urfave/cli/v3"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

type Deploy struct {
	Network string
	Stack   string
}

func ActionDeploy() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var network, stack string
		var awsConfig *types.AWSConfig

		var config *types.Config
		now := time.Now().Unix()

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

		fileName := fmt.Sprintf("%s/logs/deploy_%s_%s_%d.log", deploymentPath, stack, network, now)
		l := logging.InitLogger(fileName)

		switch stack {
		case constants.ThanosStack:
			var err error
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
			}

			if infraOpt == constants.AWS {
				awsConfig, err = thanos.InputAWSLogin()
				if err != nil {
					fmt.Printf("Failed to login AWS: %s \n", err)
					return err
				}
			}

			thanosStack, err := thanos.NewThanosStack(l, network, false, deploymentPath, awsConfig)
			if err != nil {
				fmt.Println("Failed to initialize thanos stack", "err", err)
				return err
			}

			var inputs *thanos.DeployInfraInput
			if network == constants.Testnet || network == constants.Mainnet {
				inputs, err = thanos.InputDeployInfra()
				if err != nil {
					fmt.Println("Error collecting infrastructure deployment parameters:", err)
					return err
				}
			}

			err = thanosStack.Deploy(ctx, infraOpt, inputs)
			if err != nil {
				fmt.Println("Error deploying Thanos Stack")
				return err
			}
		}
		return nil
	}
}
