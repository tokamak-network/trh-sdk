package commands

import (
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func checkK8sInstallation() bool {
	// Check if kubectl is installed
	_, err := utils.ExecuteCommand("kubectl", "version", "--client")
	if err != nil {
		fmt.Println("❌ kubectl is not installed or not in PATH.")
		return false

	}
	fmt.Println("✅ Kubectl installed")

	// Check if Kubernetes cluster is accessible
	_, err = utils.ExecuteCommand("kubectl", "cluster-info")
	if err != nil {
		fmt.Println("❌ Kubernetes cluster is not accessible")
		return true

	}
	fmt.Println("✅ Kubernetes cluster is running")

	return true
}

func checkHelmInstallation() bool {
	_, err := utils.ExecuteCommand("helm", "version")
	if err != nil {
		fmt.Println("❌ Helm is not installed or not in PATH.")
		return false
	}
	fmt.Println("✅ Helm installed")
	return true
}

func checkDockerInstallation() bool {
	// Check if Docker is installed
	_, err := utils.ExecuteCommand("docker", "--version")
	if err != nil {
		fmt.Println("❌ Docker is not installed or not in PATH.")
		return false
	}
	fmt.Println("✅ Docker installed")

	// Check if Docker daemon is running
	_, err = utils.ExecuteCommand("docker", "info")
	if err != nil {
		fmt.Println("Docker is installed but not running.")
		return true
	}

	fmt.Println("✅ Docker is running")

	return true
}

type Dependencies struct {
	Docker bool
	K8s    bool
	Helm   bool
}

func (c *Dependencies) Check(args []string) {
	c.Docker = checkDockerInstallation()

	c.K8s = checkK8sInstallation()

	c.Helm = checkHelmInstallation()
}

func (c *Dependencies) Install(args []string) error {
	c.Check(args)

	return nil
}
