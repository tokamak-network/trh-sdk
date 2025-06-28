package commands

import (
	"context"

	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
)

type Dependencies struct {
	Docker bool
	K8s    bool
	Helm   bool
}

func (c *Dependencies) Check(ctx context.Context, args []string) {
	c.Docker = dependencies.CheckDockerInstallation(ctx)

	c.K8s = dependencies.CheckK8sInstallation(ctx)

	c.Helm = dependencies.CheckHelmInstallation(ctx)
}

func (c *Dependencies) Install(ctx context.Context, args []string) error {
	c.Check(ctx, args)

	return nil
}
