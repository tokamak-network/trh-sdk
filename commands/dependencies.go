package commands

import (
	"fmt"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/dependencies"
)

type Dependencies struct {
	Docker bool
	K8s    bool
	Helm   bool
}

func (c *Dependencies) Check(args []string, logFileName string) {
	c.Docker = dependencies.CheckDockerInstallation(logFileName)

	c.K8s = dependencies.CheckK8sInstallation(logFileName)

	c.Helm = dependencies.CheckHelmInstallation(logFileName)
}

func (c *Dependencies) Install(args []string) error {
	logFileName := fmt.Sprintf("logs/install_dependencies_%s.log", time.Now().Format("2006-01-02_15-04-05"))
	c.Check(args, logFileName)

	return nil
}
