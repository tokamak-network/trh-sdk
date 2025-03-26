package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type DeployContractsInput struct {
	l1Provider string
	l1RPCurl   string
	seed       string
	fraudProof bool
}

type DeployInfraInput struct {
	ChainName   string
	L1BeaconURL string
}

type InstallBlockExplorerInput struct {
	DatabaseUsername       string
	DatabasePassword       string
	CoinmarketcapKey       string
	CoinmarketcapTokenID   string
	WalletConnectProjectID string
}

func (t *ThanosStack) inputDeployContracts(ctx context.Context) (*DeployContractsInput, error) {
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
		blockNo, err := client.BlockNumber(ctx)
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

	fraudProof := false
	//fmt.Print("Would you like to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	//fraudProof, err = scanner.ScanBool()
	//if err != nil {
	//	fmt.Printf("Error while reading the fault-proof system setting: %s", err)
	//	return nil, err
	//}

	return &DeployContractsInput{
		l1RPCurl:   l1RPCUrl,
		l1Provider: l1RRCKind,
		seed:       seed,
		fraudProof: fraudProof,
	}, nil
}

func (t *ThanosStack) inputAWSLogin() (*types.AWSConfig, error) {
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

	return &types.AWSConfig{
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

func (t *ThanosStack) inputInstallBlockExplorer() (*InstallBlockExplorerInput, error) {
	var (
		databaseUserName,
		databasePassword,
		coinmarketcapKey,
		coinmarketcapTokenID,
		walletConnectID string
		err error
	)

	for {
		fmt.Print("Please input your database username: ")
		databaseUserName, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning database name: ", err)
			return nil, err
		}
		databaseUserName = strings.ToLower(databaseUserName)

		if databaseUserName == "" {
			fmt.Println("Database username cannot be empty")
			continue
		}

		if err := utils.ValidatePostgresUsername(databaseUserName); err != nil {
			fmt.Printf("Database username is invalid, err: %s", err.Error())
			continue
		}

		if !utils.IsValidRDSUsername(databaseUserName) {
			fmt.Println("Database user name is invalid, please try again")
			continue
		}
		break
	}

	for {
		fmt.Print("Please input your database password: ")
		databasePassword, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning database name:", err)
			return nil, err
		}

		if databasePassword == "" {
			fmt.Println("Database password cannot be empty")
			continue
		}

		if !utils.IsValidRDSPassword(databasePassword) {
			fmt.Println("Database password is invalid, please try again")
			continue
		}
		break
	}

	for {
		fmt.Print("Please input your CoinMarketCap key(read more): ")
		coinmarketcapKey, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning CoinMarketCap key:", err)
			return nil, err
		}

		if coinmarketcapKey == "" {
			fmt.Println("Coinmarketcap key cannot be empty")
			continue
		}
		break
	}

	for {
		fmt.Print("Please input your CoinMarketCap token id(read more): ")
		coinmarketcapTokenID, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning CoinMarketCap token id:", err)
			return nil, err
		}

		if coinmarketcapTokenID == "" {
			fmt.Println("Coinmarketcap ID cannot be empty")
			continue
		}
		break
	}

	for {
		fmt.Print("Please input your wallet connect id(read more): ")
		walletConnectID, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning wallet connect id:", err)
			return nil, err
		}

		if walletConnectID == "" {
			fmt.Println("WalletConnectID cannot be empty")
			continue
		}
		break
	}

	return &InstallBlockExplorerInput{
		DatabaseUsername:       databaseUserName,
		DatabasePassword:       databasePassword,
		CoinmarketcapKey:       coinmarketcapKey,
		CoinmarketcapTokenID:   coinmarketcapTokenID,
		WalletConnectProjectID: walletConnectID,
	}, nil
}

func (t *ThanosStack) cloneSourcecode(repositoryName, url string) error {
	existingSourcecode, err := utils.CheckExistingSourceCode(repositoryName)
	if err != nil {
		fmt.Println("Error while checking existing source code")
		return err
	}

	if !existingSourcecode {
		err := utils.CloneRepo(url, repositoryName)
		if err != nil {
			fmt.Println("Error while cloning the repository")
			return err
		}
	}
	fmt.Printf("\r✅ Successfully cloned the %s repository!       \n", repositoryName)

	return nil
}
