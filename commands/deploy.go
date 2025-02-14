package commands

import (
	"context"
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/urfave/cli/v3"
)

type Deploy struct {
	Network string
	Stack   string
}

func Execute(network, stack string) error {
	if !constants.SupportedStacks[stack] {
		return fmt.Errorf("unsupported stack: %s", stack)
	}

	if !constants.SupportedNetworks[network] {
		return fmt.Errorf("unsupported network: %s", network)
	}

	switch stack {
	case constants.ThanosStack:
		thanosStack := NewThanosStack(network)
		err := thanosStack.Deploy()
		if err != nil {
			fmt.Println("Error deploying Thanos Stack")
			return err
		}
	}
	return nil
}

func ActionDeploy() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		var err error
		network := cmd.String("network")
		stack := cmd.String("stack")
		if network == "" {
			fmt.Print("Input network(local-devnet, testnet, mainnet): ")
			network, err = scanner.ScanString()
			if err != nil {
				fmt.Println("Error parsing the network: ", err)
				return err
			}

			if !constants.SupportedNetworks[network] {
				return fmt.Errorf("unsupported network: %s", network)
			}
		}
		if stack == "" {
			fmt.Print("Input stack(thanos): ")
			stack, err = scanner.ScanString()
			if err != nil {
				fmt.Println("Error parsing the stack: ", err)
				return err
			}

			if !constants.SupportedStacks[stack] {
				return fmt.Errorf("unsupported stack: %s", stack)
			}
		}

		return Execute(network, stack)
	}
}
