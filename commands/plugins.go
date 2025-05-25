package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

func ActionInstallationPlugins() cli.ActionFunc {
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

		if awsConfig == nil {
			awsConfig, err = thanos.InputAWSLogin()
			if err != nil {
				fmt.Printf("Failed to login AWS: %s \n", err)
				return err
			}
		}

		if !constants.SupportedStacks[stack] {
			return fmt.Errorf("unsupported stack: %s", stack)
		}
		if !constants.SupportedNetworks[network] {
			return fmt.Errorf("unsupported network: %s", network)
		}

		if network == constants.LocalDevnet {
			fmt.Println("You are in local devnet mode. Please specify the network and stack.")
			return nil
		}

		plugins := cmd.Args().Slice()
		if len(plugins) == 0 {
			fmt.Print("Please specify at least one plugin to install(e.g: bridge)")
			return nil
		}

		// Initialize the logger
		fileName := fmt.Sprintf("%s/logs/%s_plugins_%s_%s_%d.log", deploymentPath, cmd.Name, stack, network, time.Now().Unix())
		l := logging.InitLogger(fileName)

		switch stack {
		case constants.ThanosStack:
			thanosStack, err := thanos.NewThanosStack(l, network, false, deploymentPath, awsConfig)
			if err != nil {
				fmt.Println("Failed to initialize thanos stack", "err", err)
				return err
			}

			if cmd.Name == "install" {
				switch stack {
				case constants.ThanosStack:
					return thanosStack.InstallPlugins(ctx, plugins)
				default:
					return nil
				}
			} else if cmd.Name == "uninstall" {
				switch stack {
				case constants.ThanosStack:
					return thanosStack.UninstallPlugins(ctx, plugins)
				default:
					return nil
				}
			}
			return nil
		default:
			return fmt.Errorf("unsupported stack: %s", stack)
		}
	}
}
