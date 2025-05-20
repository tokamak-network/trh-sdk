package dependencies

import (
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/logging"
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

func GetArchitecture() (string, error) {
	// Get machine architecture
	arch, err := utils.ExecuteCommand("uname", "-m")
	if err != nil {
		fmt.Println("❌ Failed to get machine architecture")
		return "", err
	}

	// Check if the architecture is supported
	if strings.Contains(arch, "x86_64") || strings.Contains(arch, "amd64") {
		arch = "amd64"
	} else if strings.Contains(arch, "aarch64") || strings.Contains(arch, "arm64") {
		arch = "arm64"
	} else if strings.Contains(arch, "armv61") {
		arch = "armv61"
	} else if strings.Contains(arch, "i386") {
		arch = "386"
	} else {
		fmt.Println("❌ Unsupported architecture:", arch)
		return "", fmt.Errorf("unsupported architecture: %s", arch)
	}

	return arch, nil
}

func CheckTerraformInstallation() bool {
	terraformVersion, err := utils.ExecuteCommand("terraform", "--version")
	if err != nil {
		fmt.Println("❌ Terraform is not installed or not found in PATH")
		return false
	}

	fmt.Printf("✅ Terraform is installed: %s \n", terraformVersion)
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
