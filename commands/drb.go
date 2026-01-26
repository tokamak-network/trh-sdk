package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/urfave/cli/v3"
)

func ActionDRBLeaderInfo() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		deploymentPath, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}

		infoFilePath := fmt.Sprintf("%s/drb-leader-info.json", deploymentPath)

		// Check if file exists
		if _, err := os.Stat(infoFilePath); os.IsNotExist(err) {
			return fmt.Errorf("DRB leader info file not found at %s\nPlease run 'trh-sdk install drb' first to deploy the leader node", infoFilePath)
		}

		// Read and parse JSON file
		infoJSON, err := os.ReadFile(infoFilePath)
		if err != nil {
			return fmt.Errorf("failed to read leader info file: %w", err)
		}

		var leaderInfo types.DRBLeaderInfo
		if err := json.Unmarshal(infoJSON, &leaderInfo); err != nil {
			return fmt.Errorf("failed to parse leader info file: %w", err)
		}

		// Display leader information
		fmt.Println("\n--------------------------------")
		fmt.Println("DRB Leader Node Information")
		fmt.Println("--------------------------------")
		fmt.Printf("Leader URL:              %s\n", leaderInfo.LeaderURL)
		fmt.Printf("Leader IP:               %s\n", leaderInfo.LeaderIP)
		fmt.Printf("Leader Port:             %d\n", leaderInfo.LeaderPort)
		fmt.Printf("Leader Peer ID:          %s\n", leaderInfo.LeaderPeerID)
		fmt.Printf("Leader EOA:              %s\n", leaderInfo.LeaderEOA)
		fmt.Printf("CommitReveal2L2 Address: %s\n", leaderInfo.CommitReveal2L2Address)
		if leaderInfo.ConsumerExampleV2Address != "" {
			fmt.Printf("ConsumerExampleV2 Address: %s\n", leaderInfo.ConsumerExampleV2Address)
		}
		fmt.Printf("Chain ID:                %d\n", leaderInfo.ChainID)
		fmt.Printf("RPC URL:                 %s\n", leaderInfo.RPCURL)
		fmt.Printf("Cluster Name:      %s\n", leaderInfo.ClusterName)
		fmt.Printf("Namespace:         %s\n", leaderInfo.Namespace)
		fmt.Printf("Deployed At:       %s\n", leaderInfo.DeploymentTimestamp)
		fmt.Println("--------------------------------")
		fmt.Printf("\nFile location: %s\n", infoFilePath)
		fmt.Println("--------------------------------\n")

		return nil
	}
}
