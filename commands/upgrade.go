package commands

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/urfave/cli/v3"
)

func ActionUpgrade() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		fmt.Println("🔄 Upgrading trh-sdk...")
		_, err := utils.ExecuteCommand(ctx, "bash", "-c", "go install github.com/tokamak-network/trh-sdk@latest")
		if err != nil {
			fmt.Printf("Failed to upgrade trh-sdk, err: %v \n", err)
		}

		fmt.Println("✅ trh-sdk upgraded successfully.")
		return nil
	}
}
