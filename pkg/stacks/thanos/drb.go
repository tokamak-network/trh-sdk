package thanos

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type DRBInputs struct {
	PrivateKey string `json:"private_key"`
	RrpcUrl    string `json:"rpc"`
}

type DRBOutput struct {
}

func (t *ThanosStack) InstallDRB(ctx context.Context, drbInputs *DRBInputs) (*DRBOutput, error) {

	// Clone cross trade repository
	err := t.cloneSourcecode(ctx, "Commit-Reveal2", "https://github.com/tokamak-network/Commit-Reveal2.git")
	if err != nil {
		return nil, fmt.Errorf("failed to clone Commit-Reveal2 repository: %s", err)
	}

	client, err := ethclient.Dial(drbInputs.RrpcUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to dial RPC URL: %w", err)
	}
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd Commit-Reveal2 && git checkout service")
	if err != nil {
		return nil, fmt.Errorf("failed to checkout service branch: %s", err)
	}

	t.logger.Info("Start to build Commit-Reveal2 contracts")

	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", "cd Commit-Reveal2 && forge clean && forge build")
	if err != nil {
		return nil, fmt.Errorf("failed to build the Commit-Reveal2 contracts: %s", err)
	}

	// Get address from private key
	address, err := utils.GetAddressFromPrivateKey(drbInputs.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get the address from the private key: %s", err)
	}

	// Make .env file
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", fmt.Sprintf("cd Commit-Reveal2 && echo 'PRIVATE_KEY=%s' > .env && echo 'DEPLOYER=%s' >> .env", drbInputs.PrivateKey, address.Hex()))
	if err != nil {
		return nil, fmt.Errorf("failed to make .env file: %s", err)
	}

	t.logger.Info("Start to deploy Commit-Reveal2 contracts")
	command := fmt.Sprintf("cd Commit-Reveal2 && forge script script/DeployCommitReveal2.s.sol:DeployCommitReveal2 --private-key %s --broadcast --rpc-url %s", drbInputs.PrivateKey, drbInputs.RrpcUrl)
	t.logger.Infof("Deploying Commit-Reveal2 contracts %s", command)
	err = utils.ExecuteCommandStream(ctx, t.logger, "bash", "-c", command)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy the Commit-Reveal2 contracts: %s", err)
	}

	// Get the contract address from the output
	addresses, err := t.getContractAddressFromOutput(ctx, "Commit-Reveal2", "DeployCommitReveal2.s.sol", chainID.Uint64())
	if err != nil {
		return nil, fmt.Errorf("failed to get the contract address: %s", err)
	}

	var contractAddress string
	for contractName, address := range addresses {
		if contractName == "CommitReveal2" {
			contractAddress = address
		}
	}

	if contractAddress == "" {
		return nil, fmt.Errorf("CommitReveal2 contract address not found")
	}

	t.logger.Infof("CommitReveal2 contract address %s", contractAddress)

	return &DRBOutput{}, nil
}

func (t *ThanosStack) deployNodes(ctx context.Context, drbInputs *DRBInputs, contractAddress string) error {
	// Clone nodes repository
	err := t.cloneSourcecode(ctx, "DRB-node", "https://github.com/tokamak-network/DRB-node.git")
	if err != nil {
		return fmt.Errorf("failed to clone DRB nodes repository: %s", err)
	}

	return nil

}

func (t *ThanosStack) UninstallDRB(ctx context.Context) error {
	return nil
}
