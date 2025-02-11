package commands

import (
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

type Deploy struct {
	Network string
	Stack   string
}

func (d *Deploy) Execute() error {
	if !constants.SupportedStacks[d.Stack] {
		return fmt.Errorf("unsupported stack: %s", d.Stack)
	}

	if !constants.SupportedNetworks[d.Network] {
		return fmt.Errorf("unsupported network: %s", d.Network)
	}

	switch d.Stack {
	case constants.ThanosStack:
		thanosStack := NewThanosStack(d.Network)
		err := thanosStack.Deploy()
		if err != nil {
			fmt.Println("Error deploying Thanos Stack")
			return err
		}
	}
	return nil
}
