package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
	"github.com/tokamak-network/trh-sdk/pkg/stacks/thanos"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionShowLogs() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		var network, stack string
		config, err := utils.ReadConfigFromJSONFile()
		if err != nil {
			fmt.Println("Error reading settings.json")
			return err
		}

		component := cmd.String("component")

		if config == nil {
			network = constants.LocalDevnet
			stack = constants.ThanosStack
		} else {
			network = config.Network
			stack = config.Stack
		}
		return ShowLogs(ctx, network, stack, component, config)
	}
}

func ShowLogs(ctx context.Context, network, stack string, component string, config *types.Config) error {

	switch stack {
	case constants.ThanosStack:
		thanosStack := thanos.NewThanosStack(network, stack)
		logging.InitLogger(fmt.Sprintf("logs/show_logs_%s_%s_%s_%d.log", stack, network, component, time.Now().Unix()))
		return thanosStack.ShowLogs(ctx, config, component)
	}

	return nil
}
