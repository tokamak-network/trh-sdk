package dependencies

import (
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func CheckK8sInstallation() bool {
	// Check if kubectl is installed
	_, err := utils.ExecuteCommand("kubectl", "version", "--client")
	if err != nil {
		fmt.Println("❌ kubectl is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ kubectl is installed")

	// Check if Kubernetes cluster is accessible
	_, err = utils.ExecuteCommand("kubectl", "cluster-info")
	if err != nil {
		fmt.Println("❌ Unable to access Kubernetes cluster")
		return true
	}
	fmt.Println("✅ Kubernetes cluster is accessible")

	return true
}

func CheckHelmInstallation() bool {
	_, err := utils.ExecuteCommand("helm", "version")
	if err != nil {
		fmt.Println("❌ Helm is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Helm is installed")
	return true
}

func CheckDockerInstallation() bool {
	// Check if Docker is installed
	_, err := utils.ExecuteCommand("docker", "--version")
	if err != nil {
		fmt.Println("❌ Docker is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Docker is installed")

	// Check if Docker daemon is running
	_, err = utils.ExecuteCommand("docker", "info")
	if err != nil {
		fmt.Println("❌ Docker is installed but the daemon is not running")
		return true
	}

	fmt.Println("✅ Docker daemon is running")

	return true
}

func CheckTerraformInstallation() bool {
	_, err := utils.ExecuteCommand("terraform", "--version")
	if err != nil {
		fmt.Println("❌ Terraform is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Terraform is installed")
	return true
}

func CheckAwsCLIInstallation() bool {
	_, err := utils.ExecuteCommand("aws", "--version")
	if err != nil {
		fmt.Println("❌ AWS CLI is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ AWS CLI is installed")
	return true
}

func CheckDirenvInstallation() bool {
	_, err := utils.ExecuteCommand("direnv", "--version")
	if err != nil {
		fmt.Println("❌ direnv is not installed or not found in PATH")
		return false
	}

	fmt.Println("✅ direnv is installed")
	return true
}
