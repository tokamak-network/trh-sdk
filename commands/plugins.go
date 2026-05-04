package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	crosstrade "github.com/tokamak-network/trh-sdk/pkg/stacks/thanos/cross-trade"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

// allPluginsCanWorkWithoutChain returns true if every plugin in the list can be installed without a deployed chain.
func allPluginsCanWorkWithoutChain(plugins []string) bool {
	for _, p := range plugins {
		if !constants.CanPluginWorkWithoutChain(p) {
			return false
		}
	}
	return len(plugins) > 0
}

func ActionInstallationPlugins() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var network, stack string

		var config *types.Config

		var awsConfig *types.AWSConfig

		deploymentPath, err := os.Getwd()
		if err != nil {
			return err
		}

		// Validate plugins FIRST before doing any setup
		plugins := cmd.Args().Slice()
		if len(plugins) == 0 {
			fmt.Print("Please specify at least one plugin to install(e.g: bridge)")
			return nil
		}

		// Validate all plugin names before proceeding
		for _, pluginName := range plugins {
			if !constants.SupportedPlugins[pluginName] {
				return fmt.Errorf("plugin '%s' is not supported. Supported plugins: %v", pluginName, constants.SupportedPluginsList)
			}
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
			// Handle empty strings - treat as if not set
			if config.Network == "" {
				network = constants.LocalDevnet
			} else {
				network = config.Network
			}
			if config.Stack == "" {
				stack = constants.ThanosStack
			} else {
				stack = config.Stack
			}
			awsConfig = config.AWS
		}

		if !constants.SupportedStacks[stack] {
			return fmt.Errorf("unsupported stack: %s", stack)
		}
		if !constants.SupportedNetworks[network] {
			return fmt.Errorf("unsupported network: %s", network)
		}

		// Plugins that work without chain can use Testnet for logging when in LocalDevnet
		allPluginsWorkWithoutChain := allPluginsCanWorkWithoutChain(plugins)

		if network == constants.LocalDevnet {
			if allPluginsWorkWithoutChain {
				// Allow: chain-independent plugins (e.g. DRB) can install without a deployed chain
				// Keep network as LocalDevnet - no need to convert to Testnet
			} else {
				fmt.Println("You are in local devnet mode. Please specify the network and stack.")
				return nil
			}
		}

		// Only prompt for AWS login if needed (after all validations)
		// Check if awsConfig is nil OR if credentials are empty
		// Regular-node also needs AWS for EC2 provisioning
		if awsConfig == nil || awsConfig.AccessKey == "" || awsConfig.SecretKey == "" {
			awsConfig, err = thanos.InputAWSLogin()
			if err != nil {
				fmt.Printf("Failed to login AWS: %s \n", err)
				return err
			}
			// Save AWS credentials to settings.json
			if config == nil {
				config = &types.Config{}
			}
			config.AWS = awsConfig
			if err := config.WriteToJSONFile(deploymentPath); err != nil {
				fmt.Printf("Warning: Failed to save AWS credentials to settings.json: %s\n", err)
			} else {
				fmt.Println("✅ AWS credentials saved to settings.json")
			}
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

			// Plugins that work without chain can proceed in LocalDevnet
			allPluginsWorkWithoutChain := allPluginsCanWorkWithoutChain(plugins)

			if network == constants.LocalDevnet && !allPluginsWorkWithoutChain {
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

						// Some plugins can work without existing chain deployment
						if (config == nil || config.K8s == nil) && !constants.CanPluginWorkWithoutChain(pluginName) {
							return fmt.Errorf("the chain has not been deployed yet, please deploy the chain first")
						}

						var displayNamespace string
						if pluginName == constants.PluginMonitoring {
							displayNamespace = constants.MonitoringNamespace
						} else if pluginName == constants.PluginDRB {
							displayNamespace = constants.DRBNamespace
						} else {
							if !constants.CanPluginWorkWithoutChain(pluginName) && (config == nil || config.K8s == nil) {
								return fmt.Errorf("the chain has not been deployed yet, please deploy the chain first")
							}
							displayNamespace = config.K8s.Namespace
						}

						fmt.Printf("Installing plugin: %s in namespace: %s...\n", pluginName, displayNamespace)

						switch pluginName {
						case constants.PluginUptimeService:
							config, err := thanosStack.GetUptimeServiceConfig(ctx)
							if err != nil {
								return fmt.Errorf("failed to get uptime-service configuration: %w", err)
							}
							_, err = thanosStack.InstallUptimeService(ctx, config)
							if err != nil {
								return thanosStack.UninstallUptimeService(ctx)
							}
							return nil
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
							// Check if monitoring namespace already exists
							exists, err := utils.CheckNamespaceExists(ctx, constants.MonitoringNamespace)
							if err != nil {
								fmt.Printf("Error checking monitoring namespace: %v\n", err)
								return err
							}

							if exists {
								fmt.Println("✅ Monitoring plugin is already installed")
								return nil
							}

							// Get monitoring configuration
							installMonitoringInput, err := thanos.InputInstallMonitoring()
							if err != nil || installMonitoringInput == nil {
								fmt.Println("Error installing monitoring:", err)
								return err
							}

							// Validate monitoring input
							if err := installMonitoringInput.Validate(); err != nil {
								return fmt.Errorf("invalid monitoring configuration: %w", err)
							}

							config, err := thanosStack.GetMonitoringConfig(ctx, installMonitoringInput.AdminPassword, installMonitoringInput.AlertManager, installMonitoringInput.LoggingEnabled)
							if err != nil {
								return fmt.Errorf("failed to get monitoring configuration: %w", err)
							}
							monitoringInfo, err := thanosStack.InstallMonitoring(ctx, config)
							if err != nil {
								fmt.Println("Error installing monitoring:", err)
								return thanosStack.UninstallMonitoring(ctx)
							}

							// Display monitoring information using the returned MonitoringInfo
							thanosStack.DisplayMonitoringInfo(monitoringInfo)

							return nil
						case constants.PluginCrossTrade:
							// Get the cross-trade type from command flags
							crossTradeType := strings.TrimSpace(strings.ToLower(cmd.String("type")))
							if crossTradeType == "" {
								crossTradeType = string(constants.CrossTradeDeployModeL2ToL2)
							}

							mode := constants.CrossTradeDeployMode(crossTradeType)

							registerNewChain := cmd.Bool("register-chain")
							registerNewTokens := cmd.Bool("register-tokens")

							if registerNewChain && registerNewTokens {
								return fmt.Errorf("flags --register-chain and --register-tokens cannot be used at the same time")
							}

							// Validate the cross-trade type
							if !constants.IsSupportedCrossTradeDeployMode(constants.CrossTradeDeployMode(crossTradeType)) {
								return fmt.Errorf("unsupported cross-trade type: %s. Supported types: %s, %s",
									crossTradeType, constants.CrossTradeDeployModeL2ToL2, constants.CrossTradeDeployModeL2ToL1)
							}

							if registerNewChain {
								inputs, err := crosstrade.GetNewChainRegistrationInputs(ctx, l, deploymentPath, constants.CrossTradeDeployMode(crossTradeType), config)
								if err != nil {
									return err
								}

								// Register new chain on the existing cross-trade plugin
								_, err = thanosStack.DeployCrossTradeContracts(ctx, inputs, false)
								if err != nil {
									return err
								}

								return nil
							}

							if registerNewTokens {
								// Only register new tokens on the existing cross-trade plugin
								registerTokenInputs, err := crosstrade.GetRegisterTokensFromPrompt(
									ctx,
									l,
									deploymentPath,
									constants.CrossTradeDeployMode(crossTradeType),
									config.CrossTrade[mode],
								)
								if err != nil {
									return err
								}
								_, err = thanosStack.RegisterNewTokensOnExistingCrossTrade(
									ctx,
									constants.CrossTradeDeployMode(crossTradeType),
									registerTokenInputs,
								)
								if err != nil {
									return err
								}
								return nil
							}

							// Otherwise, deploy the cross trade plugin from scratch
							inputs, err := crosstrade.GetCrossTradeContractsInputs(ctx, l, deploymentPath, constants.CrossTradeDeployMode(crossTradeType), config)
							if err != nil {
								return err
							}

							_, err = thanosStack.DeployCrossTradeContracts(ctx, inputs, true)
							if err != nil {
								return err
							}

							return nil

						case constants.PluginDRB:
							if err := thanosStack.InstallDRB(ctx); err != nil {
								return thanosStack.UninstallDRB(ctx)
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
							displayNamespace = constants.MonitoringNamespace
						} else if pluginName == constants.PluginDRB {
							displayNamespace = constants.DRBNamespace
						} else {
							displayNamespace = config.K8s.Namespace
						}

						fmt.Printf("Uninstalling plugin: %s in namespace: %s...\n", pluginName, displayNamespace)

						switch pluginName {
						case constants.PluginUptimeService:
							return thanosStack.UninstallUptimeService(ctx)
						case constants.PluginBridge:
							return thanosStack.UninstallBridge(ctx)
						case constants.PluginBlockExplorer:
							return thanosStack.UninstallBlockExplorer(ctx)
						case constants.PluginMonitoring:
							return thanosStack.UninstallMonitoring(ctx)
						case constants.PluginCrossTrade:
							crossTradeType := strings.TrimSpace(strings.ToLower(cmd.String("type")))
							if crossTradeType == "" {
								crossTradeType = string(constants.CrossTradeDeployModeL2ToL2)
							}

							// Validate the cross-trade type
							if !constants.IsSupportedCrossTradeDeployMode(constants.CrossTradeDeployMode(crossTradeType)) {
								return fmt.Errorf("unsupported cross-trade type: %s. Supported types: %s, %s",
									crossTradeType, constants.CrossTradeDeployModeL2ToL2, constants.CrossTradeDeployModeL2ToL1)
							}

							return thanosStack.UninstallCrossTrade(ctx, constants.CrossTradeDeployMode(crossTradeType))
						case constants.PluginDRB:
							return thanosStack.UninstallDRB(ctx)
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
