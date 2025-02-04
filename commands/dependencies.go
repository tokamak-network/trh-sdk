package commands

import (
	"fmt"
)

func checkK8sInstallation() error {
	fmt.Println("Checking Kubernetes installation...")

	// Check if kubectl is installed
	_, err := executeCommand("kubectl", "version", "--client")
	if err != nil {
		fmt.Println("❌ kubectl is not installed or not in PATH.")
		return err

	}
	fmt.Println("✅ kubectl installed")

	// Check if Kubernetes cluster is accessible
	_, err = executeCommand("kubectl", "cluster-info")
	if err != nil {
		fmt.Println("❌ Kubernetes cluster is not accessible")
		return err

	}
	fmt.Println("✅ Kubernetes cluster is running")

	return nil
}

func checkHelmInstallation() error {
	_, err := executeCommand("helm", "version")
	if err != nil {
		fmt.Println("❌ Helm is not installed or not in PATH.")
		return err
	}
	fmt.Println("✅ Helm installed")
	return nil
}

func checkDockerInstallation() error {
	fmt.Println("Checking Docker installation...")

	// Check if Docker is installed
	_, err := executeCommand("docker", "--version")
	if err != nil {
		fmt.Println("❌ Docker is not installed or not in PATH.")
		return err
	}
	fmt.Println("✅ Docker installed")

	// Check if Docker daemon is running
	_, err = executeCommand("docker", "info")
	if err != nil {
		fmt.Println("Docker is installed but not running.")
		return err
	}
	fmt.Println("✅ Docker is running")
	return nil
}

type Dependencies struct {
}

func (c *Dependencies) Check(args []string) error {
	err := checkDockerInstallation()
	if err != nil {
		return err
	}

	err = checkK8sInstallation()
	if err != nil {
		return err
	}

	err = checkHelmInstallation()
	if err != nil {
		return err
	}

	return err
}
