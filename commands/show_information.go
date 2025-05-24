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

func ActionShowInformation() cli.ActionFunc {
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

		return ShowInformation(ctx, network, stack, config, selectedDeployment.DeploymentPath)
	}
}

func ShowInformation(ctx context.Context, network, stack string, config *types.Config, deploymentPath string) error {
	fileName := fmt.Sprintf("%s/logs/show_info_%s_%s_%d.log", deploymentPath, stack, network, time.Now().Unix())
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
		return thanosStack.ShowInformation(ctx)
	}

	return nil
}
