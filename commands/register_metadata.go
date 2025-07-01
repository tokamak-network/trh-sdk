package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionRegisterMetadata() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		config, err := utils.ReadConfigFromJSONFile()
		if err != nil || config == nil {
			return fmt.Errorf("Check if contracts deployed on L1, use `deploy-contracts` command for that: %v", err)
		}

		switch config.Stack {
		case constants.ThanosStack:
			thanosStack := thanos.NewThanosStack(config.Network, config.Stack, config)
			if config.Network == "Mainnet" {
				return fmt.Errorf("register metadata is not supported on Mainnet")
			}
			err = thanosStack.RegisterMetadata(ctx)
			return err
		default:
			return fmt.Errorf("unsupported stack: %s", config.Stack)
		}
	}
}
