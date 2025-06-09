package dependencies

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func CheckK8sInstallation(ctx context.Context) bool {
	// Check if kubectl is installed
	_, err := utils.ExecuteCommand(ctx, "kubectl", "version", "--client")
	if err != nil {
		fmt.Println("❌ kubectl is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ kubectl is installed")

	return true
}

func CheckHelmInstallation(ctx context.Context) bool {
	_, err := utils.ExecuteCommand(ctx, "helm", "version")
	if err != nil {
		fmt.Println("❌ Helm is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Helm is installed")
	return true
}

func CheckDockerInstallation(ctx context.Context) bool {
	// Check if Docker is installed
	_, err := utils.ExecuteCommand(ctx, "docker", "--version")
	if err != nil {
		fmt.Println("❌ Docker is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Docker is installed")

	// Check if Docker daemon is running
	_, err = utils.ExecuteCommand(ctx, "docker", "info")
	if err != nil {
		fmt.Println("❌ Docker is installed but the daemon is not running")
		return true
	}

	fmt.Println("✅ Docker daemon is running")

	return true
}

func GetArchitecture(ctx context.Context) (string, error) {
	// Get machine architecture
	arch, err := utils.ExecuteCommand(ctx, "uname", "-m")
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

func CheckTerraformInstallation(ctx context.Context) bool {
	terraformVersion, err := utils.ExecuteCommand(ctx, "terraform", "--version")
	if err != nil {
		fmt.Println("❌ Terraform is not installed or not found in PATH")
		return false
	}

	fmt.Printf("✅ Terraform is installed: %s \n", terraformVersion)
	return true
}

func CheckAwsCLIInstallation(ctx context.Context) bool {
	_, err := utils.ExecuteCommand(ctx, "aws", "--version")
	if err != nil {
		fmt.Println("❌ AWS CLI is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ AWS CLI is installed")
	return true
}

func CheckDirenvInstallation(ctx context.Context) bool {
	_, err := utils.ExecuteCommand(ctx, "direnv", "--version")
	if err != nil {
		fmt.Println("❌ direnv is not installed or not found in PATH")
		return false
	}

	fmt.Println("✅ direnv is installed")
	return true
}

func CheckPnpmInstallation(ctx context.Context) bool {
	_, err := utils.ExecuteCommand(ctx, "pnpm", "--version")
	if err != nil {
		fmt.Println("❌ pnpm is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ pnpm is installed")
	return true
}

func CheckFoundryInstallation(ctx context.Context) bool {
	_, err := utils.ExecuteCommand(ctx, "forge", "--version")
	if err != nil {
		fmt.Println("❌ forge is not installed or not found in PATH")
		return false
	}

	_, err = utils.ExecuteCommand(ctx, "anvil", "--version")
	if err != nil {
		fmt.Println("❌ anvil is not installed or not found in PATH")
		return false
	}

	_, err = utils.ExecuteCommand(ctx, "cast", "--version")
	if err != nil {
		fmt.Println("❌ cast is not installed or not found in PATH")
		return false
	}
	fmt.Println("✅ Foundry is installed")
	return true
}
