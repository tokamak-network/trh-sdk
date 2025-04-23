package commands

import (
	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
)

type Dependencies struct {
	Docker bool
	K8s    bool
	Helm   bool
}

func (c *Dependencies) Check(args []string) {
	c.Docker = dependencies.CheckDockerInstallation()

	c.K8s = dependencies.CheckK8sInstallation()

	c.Helm = dependencies.CheckHelmInstallation()
}

func (c *Dependencies) Install(args []string) error {
	c.Check(args)

	return nil
}
