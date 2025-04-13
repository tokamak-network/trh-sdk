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

func CheckPnpmInstallation() bool {
	_, err := utils.ExecuteCommand("pnpm", "--version")
	if err != nil {
		fmt.Println("❌ pnpm is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ pnpm is installed")
	return true
}

func CheckFoundryInstallation() bool {
	_, err := utils.ExecuteCommand("forge", "--version")
	if err != nil {
		fmt.Println("❌ forge is not installed or not found in PATH")
		return false
	}

	_, err = utils.ExecuteCommand("anvil", "--version")
	if err != nil {
		fmt.Println("❌ anvil is not installed or not found in PATH")
		return false
	}

	_, err = utils.ExecuteCommand("cast", "--version")
	if err != nil {
		fmt.Println("❌ cast is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Foundry is installed")
	return true
}
