package dependencies

import (
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func CheckK8sInstallation(logFileName string) bool {
	// Check if kubectl is installed
	_, err := utils.ExecuteCommand("kubectl", logFileName, "version", "--client")
	if err != nil {
		utils.LogToFile(logFileName, "❌ kubectl is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ kubectl is installed", true)

	return true
}

func CheckHelmInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("helm", logFileName, "version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ Helm is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ Helm is installed", true)
	return true
}

func CheckDockerInstallation(logFileName string) bool {
	// Check if Docker is installed
	_, err := utils.ExecuteCommand("docker", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ Docker is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ Docker is installed", true)

	// Check if Docker daemon is running
	_, err = utils.ExecuteCommand("docker", logFileName, "info")
	if err != nil {
		utils.LogToFile(logFileName, "❌ Docker is installed but the daemon is not running", true)
		return true
	}

	utils.LogToFile(logFileName, "✅ Docker daemon is running", true)

	return true
}

func CheckTerraformInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("terraform", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ Terraform is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ Terraform is installed", true)
	return true
}

func CheckAwsCLIInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("aws", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ AWS CLI is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ AWS CLI is installed", true)
	return true
}

func CheckDirenvInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("direnv", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ direnv is not installed or not found in PATH", true)
		return false
	}

	utils.LogToFile(logFileName, "✅ direnv is installed", true)
	return true
}

func CheckPnpmInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("pnpm", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("❌ pnpm is not installed or not found in PATH: %v", err), true)
		return false
	}
	utils.LogToFile(logFileName, "✅ pnpm is installed", true)
	return true
}

func CheckFoundryInstallation(logFileName string) bool {
	_, err := utils.ExecuteCommand("forge", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ forge is not installed or not found in PATH", true)
		return false
	}

	_, err = utils.ExecuteCommand("anvil", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ anvil is not installed or not found in PATH", true)
		return false
	}

	_, err = utils.ExecuteCommand("cast", logFileName, "--version")
	if err != nil {
		utils.LogToFile(logFileName, "❌ cast is not installed or not found in PATH", true)
		return false
	}
	utils.LogToFile(logFileName, "✅ Foundry is installed", true)
	return true
}
