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
		var network, stack string
		var err error
		var selectedDeployment *types.Deployment
		selectedDeployment, err = utils.SelectDeployment()
		if err != nil {
			fmt.Println("Error selecting deployment:", err)
			return err
		}

		if selectedDeployment == nil {
			fmt.Println("No deployment selected.")
			return nil
		}

		config, err := utils.ReadConfigFromJSONFile(selectedDeployment.DeploymentPath)
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

		component := cmd.String("component")
		isTroubleshoot := cmd.Bool("troubleshoot")

		return ShowLogs(ctx, network, stack, component, isTroubleshoot, config, selectedDeployment.DeploymentPath)
	}
}

func ShowLogs(ctx context.Context, network, stack string, component string, isTroubleshoot bool, config *types.Config, deploymentPath string) error {
	// Initialize the logger
	fileName := fmt.Sprintf("%s/logs/show_logs_%s_%s_%d.log", deploymentPath, stack, network, time.Now().Unix())
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

		thanosStack := thanos.NewThanosStack(l, network, stack, config, awsProfile, true, deploymentPath)
		return thanosStack.ShowLogs(ctx, config, component, isTroubleshoot)
	}

	return nil
}
