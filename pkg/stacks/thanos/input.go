package thanos

import (
	"context"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) inputDeployContracts() (*DeployContractsInput, error) {
	fmt.Println("You are about to deploy the L1 contracts.")
	var (
		l1RPCUrl  string
		l1RRCKind string
		err       error
	)
	for {
		fmt.Print("Please enter your L1 RPC URL: ")
		l1RPCUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading L1 RPC URL: %s", err)
			return nil, err
		}

		client, err := ethclient.Dial(l1RPCUrl)
		if err != nil {
			fmt.Printf("Invalid L1 RPC URL: %s. Please try again", l1RPCUrl)
			continue
		}
		blockNo, err := client.BlockNumber(context.Background())
		if err != nil {
			fmt.Printf("Failed to retrieve block number: %s", err)
			continue
		}
		if blockNo == 0 {
			fmt.Printf("The L1 RPC URL is not returning any blocks. Please try again")
			continue
		}

		l1RRCKind = utils.DetectRPCKind(l1RPCUrl)
		break
	}

	fmt.Print("Please enter your admin seed phrase: ")
	seed, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error while reading the seed phrase: %s", err)
		return nil, err
	}

	faultProof := false
	fmt.Print("Would you like to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	faultProof, err = scanner.ScanBool()
	if err != nil {
		fmt.Printf("Error while reading the fault-proof system setting: %s", err)
		return nil, err
	}

	return &DeployContractsInput{
		l1RPCurl:   l1RPCUrl,
		l1Provider: l1RRCKind,
		seed:       seed,
		falutProof: faultProof,
	}, nil
}

func (t *ThanosStack) inputAWSLogin() (*types.AWSLogin, error) {
	var (
		awsAccessKeyID, awsSecretKey string
		err                          error
	)
	for {
		fmt.Print("Please enter your AWS access key (learn more): ")
		awsAccessKeyID, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS access key")
			return nil, err
		}
		if awsAccessKeyID == "" {
			fmt.Println("Error: AWS access key ID cannot be empty")
			continue
		}
		if !utils.IsValidAWSAccessKey(awsAccessKeyID) {
			fmt.Println("Error: The AWS access key ID format is invalid")
			continue
		}
		break
	}

	for {
		fmt.Print("Please enter your AWS secret key (learn more): ")
		awsSecretKey, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS secret key")
			return nil, err
		}
		if awsSecretKey == "" {
			fmt.Println("Error: AWS secret key cannot be empty")
			continue
		}
		if !utils.IsValidAWSSecretKey(awsSecretKey) {
			fmt.Println("Error: The AWS secret key format is invalid")
			continue
		}
		break
	}

	fmt.Print("Please enter your AWS region (default: ap-northeast-2): ")
	awsRegion, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error while reading AWS region")
		return nil, err
	}
	if awsRegion == "" {
		awsRegion = "ap-northeast-2"
	}

	return &types.AWSLogin{
		SecretKey:     awsSecretKey,
		Region:        awsRegion,
		AccessKey:     awsAccessKeyID,
		DefaultFormat: "json",
	}, nil
}

func (t *ThanosStack) inputDeployInfra() (*DeployInfraInput, error) {
	fmt.Print("Please enter your chain name: ")
	chainName, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error while reading chain name: %s", err)
		return nil, err
	}

	var l1BeaconUrl string
	for {
		fmt.Print("Please enter your L1 beacon URL: ")
		l1BeaconUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading L1 beacon URL: %s\n", err)
			continue
		}

		if !utils.IsValidBeaconURL(l1BeaconUrl) {
			fmt.Println("Error: The URL provided does not return a valid beacon genesis response. Please enter a valid URL.")
			continue
		}

		break
	}

	return &DeployInfraInput{
		ChainName:   chainName,
		L1BeaconURL: l1BeaconUrl,
	}, nil
}

func (t *ThanosStack) cloneSourcecode(repositoryName, url string) error {
	existingSourcecode, err := utils.CheckExistingSourceCode(repositoryName)
	if err != nil {
		fmt.Println("Error while checking existing source code")
		return err
	}

	if !existingSourcecode {
		fmt.Printf("Cloning the %s repository...", repositoryName)
		err := utils.CloneRepo(url, repositoryName)
		if err != nil {
			fmt.Println("Error while cloning the repository")
			return err
		}
	}
	fmt.Printf("\r✅ Successfully cloned the %s repository!       \n", repositoryName)

	return nil
}

func (t *ThanosStack) inputHelmReleaseName(namespace string) (string, error) {
	var helmReleaseNameInput string
	var err error
	for {
		fmt.Print("Please enter a name for your Helm chart release: ")
		helmReleaseNameInput, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading Helm chart release name:", err)
			return "", err
		}

		if helmReleaseNameInput == "" {
			fmt.Println("Error: The release name cannot be empty. Please try again.")
			continue
		}

		releaseNameExist, err := utils.HelmReleaseExists(namespace, helmReleaseNameInput)
		if err != nil {
			fmt.Println("Error while checking if Helm chart release exists:", helmReleaseNameInput)
			return "", err
		}
		if releaseNameExist {
			fmt.Println("Error: This Helm release name already exists. Please choose a different name.")
			continue
		}

		break
	}
	return helmReleaseNameInput, nil
}
