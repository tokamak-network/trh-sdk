package thanos

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

var (
	chainNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9 ]*$`)
)

type DeployContractsInput struct {
	l1Provider         string
	l1RPCurl           string
	l1ChainID          uint64
	seed               string
	fraudProof         bool
	ChainConfiguration *types.ChainConfiguration
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
	l1RPCUrl, l1RRCKind, l1ChainID, err := t.inputL1RPC(ctx)
	if err != nil {
		fmt.Printf("Error while reading L1 RPC URL: %s", err)
		return nil, err
	}

	fmt.Print("Please enter your admin seed phrase: ")
	seed, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error while reading the seed phrase: %s", err)
		return nil, err
	}

	if seed == "" {
		fmt.Println("Error: Seed phrase cannot be empty")
		return nil, fmt.Errorf("seed phrase cannot be empty")
	}

	fraudProof := false
	//fmt.Print("Would you like to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	//fraudProof, err = scanner.ScanBool()
	//if err != nil {
	//	fmt.Printf("Error while reading the fault-proof system setting: %s", err)
	//	return nil, err
	//}

	fmt.Print("Would you like to perform advanced configurations? (Refer to the SDK Guide for more details) (Y/n): ")
	wantAdvancedConfigs, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Printf("Error while reading advanced configurations option: %s", err)
		return nil, err
	}

	var (
		maxChannelDuration               uint64 = constants.L1ChainConfigurations[l1ChainID].MaxChannelDuration
		l2OutputOracleSubmissionInterval uint64 = constants.L1ChainConfigurations[l1ChainID].L2OutputOracleSubmissionInterval
		finalizationPeriodSeconds        uint64 = constants.L1ChainConfigurations[l1ChainID].FinalizationPeriodSeconds
		l1BlockTime                      uint64 = constants.L1ChainConfigurations[l1ChainID].BlockTimeInSeconds

		l2BlockTime              uint64 = constants.DefaultL2BlockTimeInSeconds
		batchSubmissionFrequency uint64 = maxChannelDuration * l1BlockTime
		outputFrequency          uint64 = l2OutputOracleSubmissionInterval * l2BlockTime
		challengePeriod          uint64 = finalizationPeriodSeconds
	)

	if wantAdvancedConfigs {
		for {
			fmt.Printf("L2 Block Time (default: %d seconds): ", constants.DefaultL2BlockTimeInSeconds)
			value, err := scanner.ScanInt()
			if err != nil {
				fmt.Printf("Error while reading L2 block time: %s", err)
				continue
			}

			if value < 0 {
				fmt.Println("Error: L2 block time must be greater than 0")
				continue
			} else if value > 0 {
				l2BlockTime = uint64(value)
			} else {
				l2BlockTime = constants.DefaultL2BlockTimeInSeconds
			}

			break
		}

		for {
			fmt.Printf("Batch Submission Frequency (Default: %d L1 blocks ≈ %d seconds, must be a multiple of %d): ", maxChannelDuration, l1BlockTime*maxChannelDuration, l1BlockTime)
			value, err := scanner.ScanInt()
			if err != nil {
				fmt.Printf("Error while reading batch submission frequency: %s", err)
				continue
			}

			if value < 0 {
				fmt.Println("Error: Batch submission frequency must be greater than 0")
				continue
			} else if (uint64(value) % l1BlockTime) != 0 {
				fmt.Printf("Error: Batch submission frequency must be a multiple of %d \n", l1BlockTime)
				continue
			} else if value > 0 {
				batchSubmissionFrequency = uint64(value)
			} else {
				batchSubmissionFrequency = maxChannelDuration * l1BlockTime
			}

			break
		}

		for {
			fmt.Printf("Output Root Frequency (Default: %d L2 blocks ≈ %d seconds, must be a multiple of %d): ",
				l2OutputOracleSubmissionInterval, l2OutputOracleSubmissionInterval*l2BlockTime, l2BlockTime)
			value, err := scanner.ScanInt()
			if err != nil {
				fmt.Printf("Error while reading output frequency: %s", err)
				continue
			}

			if value < 0 {
				fmt.Println("Error: Output frequency must be greater than 0")
				continue
			} else if (uint64(value) % l2BlockTime) != 0 {
				fmt.Printf("Error: Output frequency must be a multiple of %d \n", l2BlockTime)
				continue
			} else if value > 0 {
				outputFrequency = uint64(value)
			} else {
				outputFrequency = l2OutputOracleSubmissionInterval * l2BlockTime
			}

			break
		}
		if l1ChainID != 1 {
			for {
				fmt.Printf("Challenge Period (Default: %d seconds): ", finalizationPeriodSeconds)
				value, err := scanner.ScanInt()
				if err != nil {
					fmt.Printf("Error while reading challenge period: %s", err)
					continue
				}

				if value < 0 {
					fmt.Println("Error: Challenge period must be greater than 0")
					continue
				} else if value > 0 {
					challengePeriod = uint64(value)
				} else {
					challengePeriod = finalizationPeriodSeconds
				}
				break
			}
		}

	}

	return &DeployContractsInput{
		l1RPCurl:   l1RPCUrl,
		l1Provider: l1RRCKind,
		l1ChainID:  l1ChainID,
		seed:       seed,
		fraudProof: fraudProof,
		ChainConfiguration: &types.ChainConfiguration{
			L2BlockTime:              l2BlockTime,
			L1BlockTime:              l1BlockTime,
			BatchSubmissionFrequency: batchSubmissionFrequency,
			ChallengePeriod:          challengePeriod,
			OutputRootFrequency:      outputFrequency,
		},
	}, nil
}

func (t *ThanosStack) inputL1RPC(ctx context.Context) (l1RPCUrl string, l1RRCKind string, l1ChainID uint64, err error) {
	for {
		fmt.Print("Please enter your L1 RPC URL: ")
		l1RPCUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading L1 RPC URL: %s", err)
			return "", "", 0, err
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

		// Fetch L1 ChainId
		chainID, err := client.ChainID(ctx)
		if err != nil || chainID == nil {
			fmt.Printf("Failed to retrieve chain ID: %s", err)
			continue
		}

		l1ChainID = chainID.Uint64()
		break
	}

	return l1RPCUrl, l1RRCKind, l1ChainID, nil
}

func (t *ThanosStack) inputAWSLogin() (*types.AWSConfig, error) {
	var (
		awsAccessKeyID, awsSecretKey, awsRegion string
		err                                     error
	)
	for {
		fmt.Print("Please enter your AWS access key: ")
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
		fmt.Print("Please enter your AWS secret key: ")
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

	for {
		fmt.Print("Please enter your AWS region (default: ap-northeast-2): ")
		awsRegion, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS region")
			return nil, err
		}
		if awsRegion == "" {
			awsRegion = "ap-northeast-2"
		}
		fmt.Println("Verifying region availability...")
		if !aws.IsAvailableRegion(awsAccessKeyID, awsSecretKey, awsRegion) {
			fmt.Println("Error: The AWS region is not available. Please try again.")
			continue
		}
		break
	}

	return &types.AWSConfig{
		SecretKey:     awsSecretKey,
		Region:        awsRegion,
		AccessKey:     awsAccessKeyID,
		DefaultFormat: "json",
	}, nil
}

func (t *ThanosStack) inputDeployInfra() (*DeployInfraInput, error) {
	var (
		chainName   string
		l1BeaconURL string
		err         error
	)
	for {
		fmt.Print("Please enter your chain name: ")
		chainName, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading chain name: %s", err)
			return nil, err
		}

		chainName = strings.Join(strings.Fields(chainName), " ")

		if chainName == "" {
			fmt.Println("Error: Chain name cannot be empty")
			continue
		}

		if !chainNameRegex.MatchString(chainName) {
			fmt.Println("Input must contain only letters (a-z, A-Z), numbers (0-9), spaces. Special characters are not allowed")
			continue
		}

		break
	}

	l1BeaconURL, err = t.inputL1BeaconURL()
	if err != nil {
		fmt.Printf("Error while reading L1 beacon URL: %s", err)
		return nil, err
	}

	return &DeployInfraInput{
		ChainName:   chainName,
		L1BeaconURL: l1BeaconURL,
	}, nil
}

func (t *ThanosStack) inputL1BeaconURL() (string, error) {
	var (
		l1BeaconURL string
		err         error
	)
	for {
		fmt.Print("Please enter your L1 beacon URL: ")
		l1BeaconURL, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading L1 beacon URL: %s\n", err)
			continue
		}

		if !utils.IsValidBeaconURL(l1BeaconURL) {
			fmt.Println("Error: The URL provided does not return a valid beacon genesis response. Please enter a valid URL.")
			continue
		}

		break
	}

	return l1BeaconURL, nil
}

func (t *ThanosStack) inputInstallBlockExplorer() (*InstallBlockExplorerInput, error) {
	var (
		databaseUserName,
		databasePassword,
		coinmarketcapKey,
		//coinmarketcapTokenID,
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
		fmt.Print("Please input your CoinMarketCap key: ")
		coinmarketcapKey, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error scanning CoinMarketCap key:", err)
			return nil, err
		}

		if coinmarketcapKey == "" {
			fmt.Println("CoinMarketCap key cannot be empty")
			continue
		}
		break
	}

	//for {
	//	fmt.Print("Please input your CoinMarketCap Token ID: ")
	//	coinmarketcapTokenID, err = scanner.ScanString()
	//	if err != nil {
	//		fmt.Println("Error scanning CoinMarketCap token id:", err)
	//		return nil, err
	//	}
	//
	//	if coinmarketcapTokenID == "" {
	//		fmt.Println("Coinmarketcap ID cannot be empty")
	//		continue
	//	}
	//	break
	//}

	for {
		fmt.Print("Please input your wallet connect id: ")
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
		CoinmarketcapTokenID:   constants.TonCoinMarketCapTokenID,
		WalletConnectProjectID: walletConnectID,
	}, nil
}

func (t *ThanosStack) inputRegisterCandidate() (*RegisterCandidateInput, error) {
	var (
		amount   float64
		memo     string
		useWTON  bool
		nameInfo string
		err      error
	)
	for {
		fmt.Print("Please enter the amount of TON to stake (minimum 1000.1): ")
		amount, err = scanner.ScanFloat()
		if err != nil {
			fmt.Printf("Error while reading amount: %s\n", err)
			continue
		}
		if amount < 1000.1 {
			fmt.Println("Error: Amount must be at least 1000.1 TON")
			continue
		}
		break
	}

	fmt.Print("Please enter a memo for the registration (optional): ")
	memo, err = scanner.ScanString()
	if err != nil {
		fmt.Printf("Error while reading memo: %s", err)
		return nil, err
	}

	fmt.Print("Please enter a name for the registration (optional): ")
	nameInfo, err = scanner.ScanString()
	if err != nil {
		fmt.Printf("Error while reading name: %s", err)
		return nil, err
	}
	fmt.Print("Would you like to use WTON instead of TON for staking? [Y or N] (default: N): ")
	useWTON, err = scanner.ScanBool(false)
	if err != nil {
		fmt.Printf("Error while reading use-wton setting: %s", err)
		return nil, err
	}
	//TODO: Check and update this with further updates
	if useWTON {
		fmt.Printf("Currently only TON is accepted %s", err)
		return nil, err
	}

	return &RegisterCandidateInput{
		amount:   amount,
		useTon:   !useWTON,
		memo:     memo,
		nameInfo: nameInfo,
	}, nil
}
