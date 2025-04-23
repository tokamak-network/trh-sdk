package commands

import (
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
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
	logging.InitLogger(fmt.Sprintf("logs/install_dependencies_%s.log", time.Now().Format("2006-01-02_15-04-05")))
	c.Check(args)

	return nil
}
