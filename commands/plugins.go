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
		l, err := logging.InitLogger(fileName)
		if err != nil {
			return fmt.Errorf("failed to initialize logger: %w", err)
		}

		switch stack {
		case constants.ThanosStack:
			thanosStack, err := thanos.NewThanosStack(ctx, l, network, true, deploymentPath, awsConfig)
			if err != nil {
				fmt.Println("Failed to initialize thanos stack", "err", err)
				return err
			}

			if network == constants.LocalDevnet {
				return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
			}

			if cmd.Name == "install" {
				switch stack {
				case constants.ThanosStack:
					for _, pluginName := range plugins {
						if !constants.SupportedPlugins[pluginName] {
							fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
							continue
						}

						var displayNamespace string
						if pluginName == constants.PluginMonitoring {
							displayNamespace = "monitoring"
						} else {
							displayNamespace = config.K8s.Namespace
						}

						fmt.Printf("Installing plugin: %s in namespace: %s...\n", pluginName, displayNamespace)

						switch pluginName {
						case constants.PluginBlockExplorer:
							installBlockExplorerInput, err := thanos.InputInstallBlockExplorer()
							if err != nil || installBlockExplorerInput == nil {
								fmt.Println("Error installing block explorer:", err)
								return err
							}

							_, err = thanosStack.InstallBlockExplorer(ctx, installBlockExplorerInput)
							if err != nil {
								return thanosStack.UninstallBlockExplorer(ctx)
							}
							return nil
						case constants.PluginBridge:
							_, err := thanosStack.InstallBridge(ctx)
							if err != nil {
								return thanosStack.UninstallBridge(ctx)
							}
							return nil
						case constants.PluginMonitoring:
							// Get monitoring configuration
							config, err := thanosStack.GetMonitoringConfig(ctx)
							if err != nil {
								return fmt.Errorf("failed to get monitoring configuration: %w", err)
							}
							err = thanosStack.InstallMonitoring(ctx, config)
							if err != nil {
								fmt.Println("Error installing monitoring:", err)
								return thanosStack.UninstallMonitoring(ctx)
							}
							return nil
						default:
							return nil
						}
					}
				default:
					return nil
				}
			} else if cmd.Name == "uninstall" {
				switch stack {
				case constants.ThanosStack:
					for _, pluginName := range plugins {
						if !constants.SupportedPlugins[pluginName] {
							fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
							continue
						}
						var displayNamespace string
						if pluginName == constants.PluginMonitoring {
							displayNamespace = "monitoring"
						} else {
							displayNamespace = config.K8s.Namespace
						}

						fmt.Printf("Uninstalling plugin: %s in namespace: %s...\n", pluginName, displayNamespace)

						switch pluginName {
						case constants.PluginBridge:
							return thanosStack.UninstallBridge(ctx)
						case constants.PluginBlockExplorer:
							return thanosStack.UninstallBlockExplorer(ctx)
						case constants.PluginMonitoring:
							return thanosStack.UninstallMonitoring(ctx)
						}
					}
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
