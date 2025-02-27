package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/urfave/cli/v3"
)

func ActionVersion() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println(constants.VERSION)
		return nil
	}
}
