package commands

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionRegisterMetadata() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		deploymentPath, err := os.Getwd()
		if err != nil {
			return err
		}

		config, err := utils.ReadConfigFromJSONFile(deploymentPath)
		if err != nil || config == nil {
			return fmt.Errorf("Check if contracts deployed on L1, use `deploy-contracts` command for that: %v", err)
		}

		switch config.Stack {
		case constants.ThanosStack:
			now := time.Now().Unix()
			fileName := fmt.Sprintf("%s/logs/register_metadata_%s_%s_%d.log", deploymentPath, config.Network, config.Stack, now)
			l, err := logging.InitLogger(fileName)
			if err != nil {
				return fmt.Errorf("failed to initialize logger: %w", err)
			}

			thanosStack, err := thanos.NewThanosStack(ctx, l, config.Network, true, deploymentPath, config.AWS)
			if err != nil {
				return fmt.Errorf("failed to create thanos stack: %v", err)
			}
			err = thanosStack.RegisterMetadata(ctx)
			return err
		default:
			return fmt.Errorf("unsupported stack: %s", config.Stack)
		}
	}
}
