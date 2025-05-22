package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

func ActionInstallationPlugins() cli.ActionFunc {
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

		plugins := cmd.Args().Slice()
		if len(plugins) == 0 {
			fmt.Print("Please specify at least one plugin to install(e.g: bridge)")
			return nil
		}

		// Initialize the logger
		fileName := fmt.Sprintf("logs/%s_plugins_%s_%s_%d.log", cmd.Name, stack, network, time.Now().Unix())
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
