package thanos

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

var (
	chainNameRegex = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9 ]*$`)
)

type DeployContractsInput struct {
	L1RPCurl           string
	ChainConfiguration *types.ChainConfiguration
	Operators          *types.Operators
	RegisterCandidate  *RegisterCandidateInput
}

func (c *DeployContractsInput) Validate(ctx context.Context, registerCandidate bool) error {
	if c.L1RPCurl == "" {
		return errors.New("l1RPCurl is required")
	}

	l1Client, err := ethclient.Dial(c.L1RPCurl)
	if err != nil {
		return fmt.Errorf("failed to connect l1 client: %w", err)
	}

	l1ChainId, err := l1Client.ChainID(ctx)
	if err != nil {
		return fmt.Errorf("failed to get l1 chain id: %w", err)
	}

	if c.ChainConfiguration == nil {
		return errors.New("ChainConfiguration is required")
	}

	if c.Operators == nil {
		return errors.New("operators is required")
	}

	if err := c.ChainConfiguration.Validate(l1ChainId.Uint64()); err != nil {
		return fmt.Errorf("chain configuration is invalid: %w", err)
	}

	if registerCandidate {
		if c.RegisterCandidate == nil {
			return errors.New("register candidate is required")
		}

		if err := c.RegisterCandidate.Validate(ctx); err != nil {
			return fmt.Errorf("register candidate is invalid: %w", err)
		}
	}

	return nil
}

type DeployInfraInput struct {
	ChainName           string
	L1BeaconURL         string
	IgnoreInstallBridge bool
}

func (c *DeployInfraInput) Validate(ctx context.Context) error {
	if c.L1BeaconURL == "" {
		return errors.New("L1BeaconURL is required")
	}

	if !utils.IsValidBeaconURL(c.L1BeaconURL) {
		return errors.New("invalid L1BeaconURL")
	}

	if !chainNameRegex.MatchString(c.ChainName) {
		return errors.New("invalid chain name, chain name must contain only letters (a-z, A-Z), numbers (0-9), spaces. Special characters are not allowed")
	}

	return nil
}

type InstallBlockExplorerInput struct {
	DatabaseUsername       string
	DatabasePassword       string
	CoinmarketcapKey       string
	CoinmarketcapTokenID   string
	WalletConnectProjectID string
}

type InstallMonitoringInput struct {
	AdminPassword string
}

func (c *InstallBlockExplorerInput) Validate(ctx context.Context) error {
	if c.DatabaseUsername == "" {
		return errors.New("database username is required")
	}
	if c.DatabasePassword == "" {
		return errors.New("database password is required")
	}

	if c.CoinmarketcapKey == "" {
		return errors.New("coinmarketcap key is required")
	}

	if c.WalletConnectProjectID == "" {
		return errors.New("wallet connect project id is required")
	}

	if err := utils.ValidatePostgresUsername(c.DatabaseUsername); err != nil {
		return errors.New("database username is invalid")
	}

	if !utils.IsValidRDSUsername(c.DatabaseUsername) {
		return errors.New("database username is invalid")
	}

	if !utils.IsValidRDSPassword(c.DatabasePassword) {
		return errors.New("database password is invalid")
	}

	// fill out missing fields by default values
	if c.CoinmarketcapTokenID == "" {
		c.CoinmarketcapTokenID = constants.TonCoinMarketCapTokenID
	}

	return nil
}

type UpdateNetworkInput struct {
	L1RPC       string
	L1BeaconURL string
}

func (c *UpdateNetworkInput) Validate(ctx context.Context) error {
	if c.L1RPC == "" {
		return errors.New("l1RPC is required")
	}

	if c.L1BeaconURL == "" {
		return errors.New("l1BeaconURL is required")
	}

	if !utils.IsValidL1RPC(c.L1RPC) {
		return fmt.Errorf("l1RPC is invalid")
	}

	if !utils.IsValidBeaconURL(c.L1BeaconURL) {
		return fmt.Errorf("l1BeaconURL is invalid")
	}

	return nil
}

func InputDeployContracts(ctx context.Context) (*DeployContractsInput, error) {
	l1RPCUrl, _, l1ChainID, err := inputL1RPC(ctx)
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

	// Select operators Accounts
	l1Client, err := ethclient.Dial(l1RPCUrl)
	if err != nil {
		fmt.Printf("Error while connecting to L1 RPC: %s", err)
		return nil, err
	}
	operators, err := SelectAccounts(ctx, l1Client, fraudProof, seed)
	if err != nil {
		return nil, err
	}

	if operators == nil {
		return nil, fmt.Errorf("no operators were found")
	}

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
		// If we deploy the mainnet network, the challenge period must be 7 days.
		if l1ChainID != constants.EthereumMainnetChainID {
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
		L1RPCurl: l1RPCUrl,
		ChainConfiguration: &types.ChainConfiguration{
			L2BlockTime:              l2BlockTime,
			L1BlockTime:              l1BlockTime,
			BatchSubmissionFrequency: batchSubmissionFrequency,
			ChallengePeriod:          challengePeriod,
			OutputRootFrequency:      outputFrequency,
		},
		Operators: operators,
	}, nil
}

func inputL1RPC(ctx context.Context) (l1RPCUrl string, l1RRCKind string, l1ChainID uint64, err error) {
	for {
		fmt.Print("Please enter your L1 RPC URL: ")
		l1RPCUrl, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading L1 RPC URL: %s", err)
			return "", "", 0, err
		}

		isValidL1Rpc := utils.IsValidL1RPC(l1RPCUrl)
		if !isValidL1Rpc {
			fmt.Println("Invalid L1 RPC, please try again")
			continue
		}

		client, err := ethclient.Dial(l1RPCUrl)
		if err != nil {
			fmt.Println("Failed to connect l1 RPC", "err", err)
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

func InputDeployInfra() (*DeployInfraInput, error) {
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

	l1BeaconURL, err = inputL1BeaconURL()
	if err != nil {
		fmt.Printf("Error while reading L1 beacon URL: %s", err)
		return nil, err
	}

	return &DeployInfraInput{
		ChainName:   chainName,
		L1BeaconURL: l1BeaconURL,
	}, nil
}

func inputL1BeaconURL() (string, error) {
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

func InputInstallBlockExplorer() (*InstallBlockExplorerInput, error) {
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

func InputInstallMonitoring() (*InstallMonitoringInput, error) {
	var (
		adminPassword string
		err           error
	)

	for {
		// Get admin password from user
		fmt.Print("🔐 Enter Grafana admin password: ")
		adminPassword, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading admin password: %s", err)
			continue
		}
		if adminPassword == "" {
			fmt.Println("Admin password cannot be empty")
			continue
		}
		break
	}

	return &InstallMonitoringInput{
		AdminPassword: adminPassword,
	}, nil
}

func GetUpdateNetworkInputs(ctx context.Context) (*UpdateNetworkInput, error) {
	// Step 2. Get the input from users
	// Step 2.1. Get L1 RPC

	var updateNetworkInput UpdateNetworkInput

	fmt.Print("Do you want to update the L1 RPC? (Y/n): ")
	wantUpdateL1RPC, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning the L1 RPC option", err)
		return nil, err
	}
	if wantUpdateL1RPC {
		l1RPC, _, _, err := inputL1RPC(ctx)
		if err != nil {
			fmt.Println("Error scanning the L1 RPC URL", err)
			return nil, err
		}

		updateNetworkInput.L1RPC = l1RPC
	}

	// Step 2.2. Get the Beacon RPC
	fmt.Print("Do you want to update the L1 Beacon RPC? (Y/n): ")
	wantUpdateL1BeaconRPC, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning the L1 Beacon RPC option", err)
		return nil, err
	}
	if wantUpdateL1BeaconRPC {
		l1BeaconRPC, err := inputL1BeaconURL()
		if err != nil {
			fmt.Println("Error scanning the L1 Beacon RPC URL", err)
		}
		updateNetworkInput.L1BeaconURL = l1BeaconRPC
	}

	fmt.Print("Do you want to update the network? (Y/n): ")
	wantUpdate, err := scanner.ScanBool(true)
	if err != nil {
		fmt.Println("Error scanning input:", err)
		return nil, err
	}

	if !wantUpdate {
		fmt.Println("Skip to update the network")
		return nil, nil
	}

	return &updateNetworkInput, nil
}

func InputAWSLogin() (*types.AWSConfig, error) {
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

func InputRegisterCandidate() (*RegisterCandidateInput, error) {
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

	for {
		fmt.Print("Please enter a memo for the registration: ")
		memo, err = scanner.ScanString()
		if err != nil {
			fmt.Printf("Error while reading memo: %s", err)
			return nil, err
		}

		if memo == "" {
			fmt.Println("Memo cannot be empty")
			continue
		}
		break
	}

	fmt.Print("Please enter a name for the registration (default: \"\"): ")
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
		Amount:   amount,
		UseTon:   !useWTON,
		Memo:     memo,
		NameInfo: nameInfo,
	}, nil
}

func SelectAccounts(ctx context.Context, client *ethclient.Client, enableFraudProof bool, seed string) (*types.Operators, error) {
	fmt.Println("Retrieving accounts...")
	accounts, err := utils.GetAccountMap(ctx, client, seed)
	if err != nil {
		return nil, err
	}

	selectedAccountsIndex := [5]int{-1, -1, -1, -1, -1}

	// get suggestion gas
	suggestionGas, err := client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	// We are using the gas price of 2x of the suggestion gas
	suggestionGas = new(big.Int).Mul(suggestionGas, big.NewInt(2))

	minimumBalanceForAdmin := new(big.Int).Mul(estimatedDeployContracts, suggestionGas)

	prompts := []string{
		fmt.Sprintf("Select an admin account from the following list (minimum %.4f ETH required)", utils.WeiToEther(minimumBalanceForAdmin)),
		"Select a sequencer account from the following list(No minimum requirement)",
		"Select a batcher account from the following list (recommended 0.3 ETH)",
		"Select a proposer account from the following list (recommended 0.3 ETH)",
	}
	if enableFraudProof {
		prompts = append(prompts, "Select a challenger account from the following list (recommended 0.3 ETH)")
	}

	operatorsMap := make(types.OperatorMap)

	displayAccounts(accounts)
	for i := range prompts {
		operator := types.Operator(i)
	startLoop:
		for {
			fmt.Println(prompts[i])
			fmt.Print("Enter the account number: ")
			input, err := scanner.ScanString()
			if err != nil {
				fmt.Printf("Failed to read input: %s", err)
				return nil, err
			}

			selectingIndex, err := strconv.Atoi(input)
			if err != nil || selectingIndex < 0 || selectingIndex >= len(accounts) {
				fmt.Println("Invalid selection. Please try again.")
				goto startLoop
			}

			selectedAccount := accounts[selectingIndex]
			selectedAccountBalance, ok := new(big.Int).SetString(selectedAccount.Balance, 10)
			if !ok {
				selectedAccountBalance = zeroBalance
			}

			switch operator {
			case types.Admin:
				if selectedAccountBalance.Cmp(minimumBalanceForAdmin) < 0 {
					fmt.Printf("The selecting account balance(%.4f ETH) is smaller than the expecting gas(%.4f ETH) to deploy the contracts \n", utils.WeiToEther(selectedAccountBalance), utils.WeiToEther(minimumBalanceForAdmin))
					goto startLoop
				}
			case types.Batcher, types.Challenger, types.Proposer:
				if selectedAccountBalance.Cmp(zeroBalance) <= 0 {
					fmt.Printf("The balance of %s must be greater than zero\n", mapAccountIndexes[i])
					goto startLoop
				}
			default:
			}

			for j, selectedAccountIndex := range selectedAccountsIndex {
				if selectingIndex == selectedAccountIndex {
					fmt.Printf("You selected this account as the %s. Do you want to want to continue(y/N): ", mapAccountIndexes[j])
					nextInput, err := scanner.ScanBool(false)
					if err != nil {
						return nil, err
					}
					if !nextInput {
						goto startLoop
					} else {
						break
					}
				}
			}

			selectedAccountsIndex[i] = selectingIndex
			operatorsMap[operator] = &types.IndexAccount{
				Address:    selectedAccount.Address,
				PrivateKey: selectedAccount.PrivateKey,
			}
			break
		}
	}

	sortedOperators := make([]*types.IndexAccount, len(operatorsMap))
	for i, operator := range operatorsMap {
		sortedOperators[i] = operator
	}

	var operators types.Operators
	for i, operator := range sortedOperators {
		fmt.Printf("%s account address: %s\n", mapAccountIndexes[i], operator.Address)

		switch types.Operator(i) {
		case types.Admin:
			operators.AdminPrivateKey = operator.PrivateKey
		case types.Sequencer:
			operators.SequencerPrivateKey = operator.PrivateKey
		case types.Batcher:
			operators.BatcherPrivateKey = operator.PrivateKey
		case types.Proposer:
			operators.ProposerPrivateKey = operator.PrivateKey
		case types.Challenger:
			operators.ChallengerPrivateKey = operator.PrivateKey
		default:
			return nil, errors.New("unknown operator type")
		}
	}

	return &operators, nil
}

func makeDeployContractConfigJsonFile(
	ctx context.Context,
	l1Provider *ethclient.Client,
	operators *types.Operators,
	deployContractTemplate *types.DeployConfigTemplate,
	filePath string,
) error {
	if operators == nil {
		return fmt.Errorf("operators cannot be nil")
	}

	if l1Provider == nil {
		return fmt.Errorf("l1Provider cannot be nil")
	}

	if deployContractTemplate == nil {
		return fmt.Errorf("deployContractTemplate cannot be nil")
	}

	if account := operators.AdminPrivateKey; account != "" {
		address, err := utils.GetAddressFromPrivateKey(account)
		if err != nil {
			fmt.Printf("Error getting address from private key: %s", err)
			return err
		}

		deployContractTemplate.FinalSystemOwner = address.Hex()
		deployContractTemplate.SuperchainConfigGuardian = address.Hex()
		deployContractTemplate.ProxyAdminOwner = address.Hex()
		deployContractTemplate.BaseFeeVaultRecipient = address.Hex()
		deployContractTemplate.L1FeeVaultRecipient = address.Hex()
		deployContractTemplate.SequencerFeeVaultRecipient = address.Hex()
		deployContractTemplate.NewPauser = address.Hex()
		deployContractTemplate.NewBlacklister = address.Hex()
		deployContractTemplate.MasterMinterOwner = address.Hex()
		deployContractTemplate.FiatTokenOwner = address.Hex()
		deployContractTemplate.UniswapV3FactoryOwner = address.Hex()
		deployContractTemplate.UniversalRouterRewardsDistributor = address.Hex()
	}
	if account := operators.SequencerPrivateKey; account != "" {
		address, err := utils.GetAddressFromPrivateKey(account)
		if err != nil {
			fmt.Printf("Error getting address from private key: %s", err)
			return err
		}
		deployContractTemplate.P2pSequencerAddress = address.Hex()
	}
	if account := operators.BatcherPrivateKey; account != "" {
		address, err := utils.GetAddressFromPrivateKey(account)
		if err != nil {
			fmt.Printf("Error getting address from private key: %s", err)
			return err
		}
		deployContractTemplate.BatchSenderAddress = address.Hex()
	}
	if account := operators.ProposerPrivateKey; account != "" {
		address, err := utils.GetAddressFromPrivateKey(account)
		if err != nil {
			fmt.Printf("Error getting address from private key: %s", err)
			return err
		}
		deployContractTemplate.L2OutputOracleProposer = address.Hex()
	}
	if account := operators.ChallengerPrivateKey; account != "" {
		address, err := utils.GetAddressFromPrivateKey(account)
		if err != nil {
			fmt.Printf("Error getting address from private key: %s", err)
			return err
		}
		deployContractTemplate.L2OutputOracleChallenger = address.Hex()
	}

	// Fetch the latest block
	latest, err := l1Provider.BlockByNumber(ctx, nil)
	if err != nil {
		fmt.Println("Error retrieving latest block")
		return err
	}

	deployContractTemplate.L1StartingBlockTag = latest.Hash().Hex()
	deployContractTemplate.L2OutputOracleStartingTimestamp = latest.Time()

	file, err := os.Create(filePath)
	if err != nil {
		fmt.Printf("Failed to create configuration file: %s", err)
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(deployContractTemplate)
	if err != nil {
		return err
	}

	return nil
}

func initDeployConfigTemplate(deployConfigInputs *DeployContractsInput, l1ChainID, l2ChainId uint64) *types.DeployConfigTemplate {
	var (
		chainConfiguration = deployConfigInputs.ChainConfiguration
	)

	if chainConfiguration == nil {
		panic("ChainConfiguration is empty")
	}

	var (
		l2BlockTime                      = chainConfiguration.L2BlockTime
		l1ChainId                        = l1ChainID
		l2OutputOracleSubmissionInterval = chainConfiguration.GetL2OutputOracleSubmissionInterval()
		finalizationPeriods              = chainConfiguration.GetFinalizationPeriodSeconds()
		enableFraudProof                 = false
	)

	defaultTemplate := &types.DeployConfigTemplate{
		NativeTokenName:                          "Tokamak Network Token",
		NativeTokenSymbol:                        "TON",
		NativeTokenAddress:                       constants.L1ChainConfigurations[l1ChainId].L2NativeTokenAddress,
		L1ChainID:                                l1ChainId,
		L2ChainID:                                l2ChainId,
		L2BlockTime:                              l2BlockTime,
		L1BlockTime:                              12,
		MaxSequencerDrift:                        600,
		SequencerWindowSize:                      3600,
		ChannelTimeout:                           300,
		BatchInboxAddress:                        utils.GenerateBatchInboxAddress(l2ChainId),
		L2OutputOracleSubmissionInterval:         l2OutputOracleSubmissionInterval,
		L2OutputOracleStartingBlockNumber:        0,
		FinalizationPeriodSeconds:                finalizationPeriods,
		BaseFeeVaultMinimumWithdrawalAmount:      "0x8ac7230489e80000",
		L1FeeVaultMinimumWithdrawalAmount:        "0x8ac7230489e80000",
		SequencerFeeVaultMinimumWithdrawalAmount: "0x8ac7230489e80000",
		BaseFeeVaultWithdrawalNetwork:            0,
		L1FeeVaultWithdrawalNetwork:              0,
		SequencerFeeVaultWithdrawalNetwork:       0,
		EnableGovernance:                         false,
		L2GenesisBlockGasLimit:                   "0x1c9c380",
		L2GenesisBlockBaseFeePerGas:              "0x3b9aca00",
		GasPriceOracleOverhead:                   188,
		GasPriceOracleScalar:                     684000,
		Eip1559Denominator:                       50,
		Eip1559Elasticity:                        6,
		Eip1559DenominatorCanyon:                 250,
		L2GenesisRegolithTimeOffset:              "0x0",
		L2GenesisCanyonTimeOffset:                "0x0",
		L2GenesisDeltaTimeOffset:                 "0x0",
		L2GenesisEcotoneTimeOffset:               "0x0",
		SystemConfigStartBlock:                   0,
		RequiredProtocolVersion:                  "0x0000000000000000000000000000000000000003000000010000000000000000",
		RecommendedProtocolVersion:               "0x0000000000000000000000000000000000000003000000010000000000000000",
		FaultGameAbsolutePrestate:                "0x03ab262ce124af0d5d328e09bf886a2b272fe960138115ad8b94fdc3034e3155",
		FaultGameMaxDepth:                        73,
		FaultGameClockExtension:                  10800,
		FaultGameMaxClockDuration:                302400,
		FaultGameGenesisBlock:                    0,
		FaultGameGenesisOutputRoot:               "0xDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF",
		FaultGameSplitDepth:                      30,
		FaultGameWithdrawalDelay:                 604800,
		PreimageOracleMinProposalSize:            126000,
		PreimageOracleChallengePeriod:            86400,
		ProofMaturityDelaySeconds:                604800,
		DisputeGameFinalityDelaySeconds:          302400,
		RespectedGameType:                        0,
		UseFaultProofs:                           enableFraudProof,
		L1UsdcAddr:                               constants.L1ChainConfigurations[l1ChainId].USDCAddress,
		UsdcTokenName:                            "Bridged USDC (Tokamak Network)",
		FactoryV2addr:                            "0x0000000000000000000000000000000000000000",
		NativeCurrencyLabelBytes:                 []uint64{84, 87, 79, 78, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
		UniswapV3FactoryOwner:                    "0x7b91111ec983c13b3C2F36C8A84a5099225786FA",
		UniswapV3FactoryFee500:                   500,
		UniswapV3FactoryTickSpacing10:            10,
		UniswapV3FactoryFee3000:                  3000,
		UniswapV3FactoryTickSpacing60:            60,
		UniswapV3FactoryFee10000:                 10000,
		UniswapV3FactoryTickSpacing200:           200,
		UniswapV3FactoryFee100:                   100,
		UniswapV3FactoryTickSpacing1:             1,
		PairInitCodeHash:                         "0x96e8ac4277198ff8b6f785478aa9a39f403cb768dd02cbee326c3e7da348845f",
		PoolInitCodeHash:                         "0xe34f199b19b2b4f47f68442619d555527d244f78a3297ea89325f843f87b8b54",
		GovernanceTokenName:                      "Optimism",
		GovernanceTokenOwner:                     "0x0000000000000000000000000000000000000333",
		GovernanceTokenSymbol:                    "OP",
		L2OutputOracleChallenger:                 "0x0000000000000000000000000000000000000001",
		ReuseDeployment:                          true,
	}

	return defaultTemplate

}

func makeTerraformEnvFile(dirPath string, config types.TerraformEnvConfig) error {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return nil
	}
	filePath := filepath.Join(dirPath, ".envrc")
	output, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error creating environment file:", err)
		return err
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	writer.WriteString(fmt.Sprintf("export TF_VAR_thanos_stack_name=\"%s\"\n", config.Namespace))
	writer.WriteString(fmt.Sprintf("export TF_VAR_aws_region=\"%s\"\n", config.AwsRegion))

	writer.WriteString(fmt.Sprintf("export TF_VAR_backend_bucket_name=\"%s\"\n", ""))

	writer.WriteString(fmt.Sprintf("export TF_CLI_ARGS_init=\"%s\"\n", "-backend-config='bucket=$TF_VAR_backend_bucket_name'"))
	writer.WriteString(fmt.Sprintf("export TF_CLI_ARGS_init=\"%s\"\n", "$TF_CLI_ARGS_init -backend-config='region=${TF_VAR_aws_region}'"))

	writer.WriteString(fmt.Sprintf("export TF_VAR_sequencer_key=\"%s\"\n", config.SequencerKey))
	writer.WriteString(fmt.Sprintf("export TF_VAR_batcher_key=\"%s\"\n", config.BatcherKey))
	writer.WriteString(fmt.Sprintf("export TF_VAR_proposer_key=\"%s\"\n", config.ProposerKey))
	writer.WriteString(fmt.Sprintf("export TF_VAR_challenger_key=\"%s\"\n", config.ChallengerKey))

	writer.WriteString(fmt.Sprintf("export TF_VAR_azs='[\"%s\"]'\n", strings.Join(config.Azs, "\", \"")))
	writer.WriteString(fmt.Sprintf("export TF_VAR_vpc_cidr=\"%s\"\n", "192.168.0.0/16"))
	writer.WriteString("export TF_VAR_vpc_name=\"${TF_VAR_thanos_stack_name}/VPC\"\n")

	writer.WriteString(fmt.Sprintf("export TF_VAR_eks_cluster_admins='[\"%s\"]'\n", config.EksClusterAdmins))

	writer.WriteString(fmt.Sprintf("export TF_VAR_genesis_file_path=\"%s\"\n", "config-files/genesis.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_rollup_file_path=\"%s\"\n", "config-files/rollup.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_prestate_file_path=\"%s\"\n", "config-files/prestate.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_prestate_hash=\"%s\"\n", "0x03ab262ce124af0d5d328e09bf886a2b272fe960138115ad8b94fdc3034e3155"))

	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_deployments_path=\"%s\"\n", config.DeploymentFilePath))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_rpc_url=\"%s\"\n", config.L1RpcUrl))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_rpc_provider=\"%s\"\n", config.L1RpcProvider))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_beacon_url=\"%s\"\n", config.L1BeaconUrl))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_op_geth_image_tag=\"%s\"\n", config.OpGethImageTag))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_thanos_stack_image_tag=\"%s\"\n", config.ThanosStackImageTag))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_max_channel_duration=\"%d\"\n", config.MaxChannelDuration))

	err = writer.Flush()
	if err != nil {
		return err
	}
	fmt.Println("Environment configuration file (.envrc) has been successfully generated!")
	return nil
}

func updateTerraformEnvFile(dirPath string, config types.UpdateTerraformEnvConfig) error {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return err
	}

	filePath := filepath.Join(dirPath, ".envrc")

	// Read existing file content
	existingContent := make(map[string]string)
	if _, err := os.Stat(filePath); err == nil {
		file, err := os.Open(filePath)
		if err != nil {
			fmt.Println("Error reading environment file:", err)
			return err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "export ") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(parts[0][7:]) // Remove "export " prefix
					value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
					existingContent[key] = value
				}
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Error scanning environment file:", err)
			return err
		}
	}

	// Prepare new values
	newValues := map[string]string{
		"TF_VAR_stack_l1_rpc_url":             config.L1RpcUrl,
		"TF_VAR_stack_l1_rpc_provider":        config.L1RpcProvider,
		"TF_VAR_stack_l1_beacon_url":          config.L1BeaconUrl,
		"TF_VAR_stack_op_geth_image_tag":      config.OpGethImageTag,
		"TF_VAR_stack_thanos_stack_image_tag": config.ThanosStackImageTag,
	}

	// Update or add new values
	for key, value := range newValues {
		existingContent[key] = value
	}

	// Write updated content back to the file
	output, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error opening environment file for writing:", err)
		return err
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	for key, value := range existingContent {
		// Two special cases, they are flatted arrays(convert from arrays to strings). We keep this format, don't put the quotes
		if key == "TF_VAR_azs" || key == "TF_VAR_eks_cluster_admins" {
			writer.WriteString(fmt.Sprintf("export %s=%s\n", key, value))

			continue
		}

		_, err := writer.WriteString(fmt.Sprintf("export %s=\"%s\"\n", key, value))
		if err != nil {
			return err
		}
	}

	if err = writer.Flush(); err != nil {
		return err
	}

	fmt.Println("Environment configuration file (.envrc) has been successfully updated!")
	return nil
}

func makeBlockExplorerEnvs(dirPath string, filename string, config types.BlockExplorerEnvs) error {
	if err := os.MkdirAll(dirPath, os.ModePerm); err != nil {
		fmt.Println("Error creating directory:", err)
		return err
	}

	filePath := filepath.Join(dirPath, filename)

	output, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Println("Error opening environment file:", err)
		return err
	}
	defer output.Close()

	writer := bufio.NewWriter(output)

	envVars := []string{
		fmt.Sprintf("export TF_VAR_db_username=\"%s\"\n", config.BlockExplorerDatabaseUserName),
		fmt.Sprintf("export TF_VAR_db_password=\"%s\"\n", config.BlockExplorerDatabasePassword),
		fmt.Sprintf("export TF_VAR_db_name=\"%s\"\n", config.BlockExplorerDatabaseName),
		fmt.Sprintf("export TF_VAR_vpc_id=\"%s\"\n", config.VpcId),
	}

	for _, envVar := range envVars {
		_, err = writer.WriteString(envVar)
		if err != nil {
			return err
		}
	}

	if err = writer.Flush(); err != nil {
		return err
	}

	fmt.Println("Environment configuration file (.envrc) has been successfully updated!")
	return nil
}

func (t *ThanosStack) cloneSourcecode(ctx context.Context, repositoryName, url string) error {
	existingSourcecode, err := utils.CheckExistingSourceCode(t.deploymentPath, repositoryName)
	if err != nil {
		fmt.Println("Error while checking existing source code")
		return err
	}

	if !existingSourcecode {
		err := utils.CloneRepo(ctx, t.l, t.deploymentPath, url, repositoryName)
		if err != nil {
			fmt.Println("Error while cloning the repository")
			return err
		}
		return nil
	}

	// Case 2: Repo exists → try pulling
	t.l.Info("Repository exists. Trying to pull latest changes...", "repo", repositoryName)
	err = utils.PullLatestCode(ctx, t.l, t.deploymentPath, repositoryName)
	if err == nil {
		t.l.Info("Successfully pulled latest changes", "repo", repositoryName)
		fmt.Printf("\r✅ Clone the %s repository successfully \n", repositoryName)
		return nil
	}

	// Case 3: Pull failed → likely broken repo → remove and re-clone
	t.l.Warn("Pull failed. Re-cloning repository...", "repo", repositoryName, "err", err)
	if removeErr := os.RemoveAll(fmt.Sprintf("%s/%s", t.deploymentPath, repositoryName)); removeErr != nil {
		t.l.Error("Failed to remove broken repository folder", "path", t.deploymentPath, "repo", repositoryName, "err", removeErr)
		return removeErr
	}

	t.l.Info("Re-cloning repository after cleanup...", "repo", repositoryName)
	err = utils.CloneRepo(ctx, t.l, t.deploymentPath, url, repositoryName)
	if err != nil {
		t.l.Error("Failed to re-clone repository", "repo", repositoryName, "err", err)
		return err
	}

	fmt.Printf("\r✅ Clone the %s repository successfully \n", repositoryName)

	return nil
}
