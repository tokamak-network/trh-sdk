package utils

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func GetDepployments() ([]*types.Deployment, error) {
	var deployments []*types.Deployment
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error getting current working directory: %w", err)
	}

	files, err := os.ReadDir(cwd + "/deployments")
	if err != nil {
		return nil, fmt.Errorf("error reading directory: %w", err)
	}

	for _, file := range files {
		deploymentPath := file.Name()
		parts := strings.Split(deploymentPath, "-")
		if len(parts) < 3 {
			continue
		}
		stack := parts[0]
		network := parts[1]
		deployments = append(deployments, &types.Deployment{
			DeploymentPath: "deployments/" + deploymentPath,
			Network:        network,
			Stack:          stack,
		})
	}

	return deployments, nil
}

func SelectDeployment() (*types.Deployment, error) {
	deployments, err := GetDepployments()
	if err != nil {
		return nil, fmt.Errorf("error getting deployments: %w", err)
	}
	for i, deployment := range deployments {
		fmt.Printf("[%d] - Deployment Path: %s, network: %s, stack: %s \n", i, deployment.DeploymentPath, deployment.Network, deployment.Stack)
	}

	if len(deployments) == 0 {
		fmt.Println("No deployments found.")
		return nil, nil
	}

	fmt.Print("Do you want to use the recent deployments? (Y/n): ")
	choose, err := scanner.ScanBool(true)
	if err != nil {
		log.Fatalf("Failed to read input, err: %s", err.Error())
	}

	var selectedDeployment *types.Deployment
	if choose {
		for {
			var input int
			fmt.Print("Please select a working deployment: ")
			input, err := scanner.ScanInt()
			if err != nil {
				fmt.Println("Invalid input. Please enter a number.")
				continue
			}
			if input < 0 || input >= len(deployments) {
				fmt.Println("Invalid selection. Please try again.")
				continue
			}
			selectedDeployment = deployments[input]
			fmt.Printf("You selected deployment: %s\n", selectedDeployment.DeploymentPath)
			break
		}
	} else {
		fmt.Println("You are working on a new deployment.")
	}

	return selectedDeployment, nil
}
