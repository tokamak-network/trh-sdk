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
	fmt.Println("You are deploying the L1 contracts.")
	var (
		l1RPCUrl string
		err      error
	)
	for {
		fmt.Print("Please input your L1 RPC URL: ")
		l1RPCUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error scanning L1 RPC URL: %s", err)
			return nil, err
		}

		client, err := ethclient.Dial(l1RPCUrl)
		if err != nil {
			fmt.Printf("Invalid L1 RPC URL: %s, please try again", l1RPCUrl)
			continue
		}
		blockNo, err := client.BlockNumber(context.Background())
		if err != nil {
			fmt.Printf("Error getting block number: %s", err)
			continue
		}
		if blockNo == 0 {
			fmt.Printf("L1 RPC URL does not have a block number, please try again")
			continue
		}
		break
	}

	fmt.Print("Please input your L1 provider: ")
	l1Provider, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning L1 provider: %s", err)
		return nil, err
	}

	fmt.Print("Please input your admin seed phrase: ")
	seed, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning the seed phrase: %s", err)
		return nil, err
	}

	faultProof := false
	fmt.Print("Do you want to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	faultProof, err = scanner.ScanBool()
	if err != nil {
		fmt.Printf("Error scanning the fault-proof system setting: %s", err)
		return nil, err
	}

	return &DeployContractsInput{
		l1RPCurl:   l1RPCUrl,
		l1Provider: l1Provider,
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
		fmt.Print("Please enter the AWS access key(read more): ")
		awsAccessKeyID, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning AWS access key")
			return nil, err
		}
		if awsAccessKeyID == "" {
			fmt.Println("Error: AWS access key ID is empty")
			continue
		}
		if !utils.IsValidAWSAccessKey(awsAccessKeyID) {
			fmt.Println("Error: AWS access key ID is invalid")
			continue
		}
		break
	}

	for {
		fmt.Print("Please enter the AWS secret key(read more): ")
		awsSecretKey, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning AWS secret key")
			return nil, err
		}
		if awsSecretKey == "" {
			fmt.Println("Error: AWS secret key is empty")
			continue
		}
		if !utils.IsValidAWSSecretKey(awsSecretKey) {
			fmt.Println("Error: AWS secret key is invalid")
			continue
		}
		break
	}

	fmt.Print("Please enter the AWS region(default ap-northeast-2): ")
	awsRegion, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS region")
		return nil, err
	}
	if awsRegion == "" {
		awsRegion = "ap-northeast-2"
	}

	fmt.Print("Please enter the format file(default \"json\"): ")
	defaultFormatFile, err := scanner.ScanString()
	if err != nil {
		fmt.Println("Error scanning AWS format file")
		return nil, err
	}
	if defaultFormatFile == "" {
		defaultFormatFile = "json"
	}

	return &types.AWSLogin{
		SecretKey:     awsSecretKey,
		Region:        awsRegion,
		AccessKey:     awsAccessKeyID,
		DefaultFormat: defaultFormatFile,
	}, nil
}

func (t *ThanosStack) inputDeployInfra() (*DeployInfraInput, error) {
	fmt.Print("Please input your chain name: ")
	chainName, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning chain name: %s", err)
		return nil, err
	}

	var l1BeaconUrl string
	for {
		fmt.Print("Please input your L1 beacon URL: ")
		l1BeaconUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error scanning L1 beacon URL: %s\n", err)
			continue
		}

		if !utils.IsValidBeaconURL(l1BeaconUrl) {
			fmt.Println("Error: The URL does not return a valid beacon genesis response. Please enter a correct URL.")
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
	doneCh := make(chan bool)
	defer close(doneCh)
	existingSourcecode, err := utils.CheckExistingSourceCode(repositoryName)
	if err != nil {
		fmt.Println("Error checking existing source code")
		return err
	}

	if !existingSourcecode {
		go utils.ShowLoadingAnimation(doneCh, fmt.Sprintf("Cloning the %s repository...", repositoryName))
		err := utils.CloneRepo(url, repositoryName)
		doneCh <- true
		if err != nil {
			fmt.Println("Error cloning the repo")
			return err
		}
	}
	fmt.Printf("\r✅ Clone the %s repository successfully!       \n", repositoryName)

	return nil
}
