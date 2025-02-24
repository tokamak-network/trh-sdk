package thanos

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var mapAccountIndexes = map[int]string{
	0: "Admin",
	1: "Sequencer",
	2: "Batcher",
	3: "Proposer",
	4: "Challenger",
}

func displayAccounts(accounts map[int]types.Account) {
	sortedAccounts := make([]types.Account, len(accounts), len(accounts))
	for i, account := range accounts {
		sortedAccounts[i] = account
	}

	for i, account := range sortedAccounts {
		fmt.Printf("\t%d. %s(%s ETH)\n", i, account.Address, account.Balance)
	}
}

func selectAccounts(client *ethclient.Client, enableFraudProof bool, seed string) (types.OperatorMap, error) {
	fmt.Println("Retrieving accounts...")
	accounts, err := types.GetAccountMap(context.Background(), client, seed)
	if err != nil {
		return nil, err
	}

	selectedAccountsIndex := [5]int{-1, -1, -1, -1, -1}

	prompts := []string{
		"Select an admin account from the following list (minimum 0.6 ETH required)",
		"Select a sequencer account from the following list",
		"Select a batcher account from the following list (recommended 0.3 ETH)",
		"Select a proposer account from the following list (recommended 0.3 ETH)",
	}
	if enableFraudProof {
		prompts = append(prompts, "Select a challenger account from the following list (recommended 0.3 ETH)")
	}
	operators := make(types.OperatorMap)

	displayAccounts(accounts)
	for i := 0; i < len(prompts); i++ {
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

			for j, selectedAccountIndex := range selectedAccountsIndex {
				if selectingIndex == selectedAccountIndex {
					fmt.Printf("You selected this account as the %s. Do you want to want to continue(y/N): ", mapAccountIndexes[j])
					nextInput, err := scanner.ScanBool()
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
			operators[types.Operator(i)] = &types.IndexAccount{
				Index:      selectingIndex,
				Address:    accounts[selectingIndex].Address,
				PrivateKey: accounts[selectingIndex].PrivateKey,
			}
			break
		}
	}

	sortedOperators := make([]*types.IndexAccount, len(operators), len(operators))
	for i, operator := range operators {
		sortedOperators[i] = operator
	}
	for i, operator := range sortedOperators {
		fmt.Printf("%s account address: %s\n", mapAccountIndexes[i], operator.Address)
	}

	return operators, nil
}

func makeDeployContractConfigJsonFile(l1Provider *ethclient.Client, operators types.OperatorMap, deployContractTemplate *types.DeployConfigTemplate) error {
	for role, account := range operators {
		switch role {
		case types.Admin:
			deployContractTemplate.FinalSystemOwner = account.Address
			deployContractTemplate.SuperchainConfigGuardian = account.Address
			deployContractTemplate.ProxyAdminOwner = account.Address
			deployContractTemplate.BaseFeeVaultRecipient = account.Address
			deployContractTemplate.L1FeeVaultRecipient = account.Address
			deployContractTemplate.SequencerFeeVaultRecipient = account.Address
			deployContractTemplate.NewPauser = account.Address
			deployContractTemplate.NewBlacklister = account.Address
			deployContractTemplate.MasterMinterOwner = account.Address
			deployContractTemplate.FiatTokenOwner = account.Address
			deployContractTemplate.UniswapV3FactoryOwner = account.Address
			deployContractTemplate.UniversalRouterRewardsDistributor = account.Address
		case types.Sequencer:
			deployContractTemplate.P2pSequencerAddress = account.Address
		case types.Batcher:
			deployContractTemplate.BatchSenderAddress = account.Address
		case types.Proposer:
			deployContractTemplate.L2OutputOracleProposer = account.Address
		case types.Challenger:
			deployContractTemplate.L2OutputOracleChallenger = account.Address
		}
	}

	// Fetch the latest block
	latest, err := l1Provider.BlockByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println("Error retrieving latest block")
		return err
	}

	deployContractTemplate.L1StartingBlockTag = latest.Hash().Hex()
	deployContractTemplate.L2OutputOracleStartingTimestamp = latest.Time()

	file, err := os.Create("deploy-config.json")
	defer file.Close()
	if err != nil {
		fmt.Printf("Failed to create configuration file: %s", err)
		return err
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(deployContractTemplate)
	if err != nil {
		return err
	}

	return nil
}

func initDeployConfigTemplate(enableFraudProof bool, network string) *types.DeployConfigTemplate {
	var l1ChainId uint64
	var l2ChainId uint64
	var batchInboxAddress string
	var finalizationPeriodSeconds uint64
	var nativeTokenAddress string
	var l2OutputOracleSubmissionInterval uint64
	var l1UsdcAddr string
	basebatchInboxAddress := "0xff00000000000000000000000000000000000000"

	if network == constants.Testnet {
		l1ChainId = 11155111
		finalizationPeriodSeconds = 12
		nativeTokenAddress = "0xa30fe40285b8f5c0457dbc3b7c8a280373c40044"
		l2OutputOracleSubmissionInterval = 120
		l1UsdcAddr = "0x94a9D9AC8a22534E3FaCa9F4e7F2E2cf85d5E4C8"
	} else if network == constants.Mainnet {
		l1ChainId = 1
		finalizationPeriodSeconds = 604800
		nativeTokenAddress = "0x2be5e8c109e2197D077D13A82dAead6a9b3433C5"
		l2OutputOracleSubmissionInterval = 10800
		l1UsdcAddr = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	}

	l2ChainId = 111551119876
	batchInboxAddress = fmt.Sprintf("%s%d", basebatchInboxAddress[:len(basebatchInboxAddress)-len(fmt.Sprintf("%d", l2ChainId))], l2ChainId)

	defaultTemplate := &types.DeployConfigTemplate{
		NativeTokenName:                          "Tokamak Network Token",
		NativeTokenSymbol:                        "TON",
		NativeTokenAddress:                       nativeTokenAddress,
		L1ChainID:                                l1ChainId,
		L2ChainID:                                l2ChainId,
		L2BlockTime:                              2,
		L1BlockTime:                              12,
		MaxSequencerDrift:                        600,
		SequencerWindowSize:                      3600,
		ChannelTimeout:                           300,
		BatchInboxAddress:                        batchInboxAddress,
		L2OutputOracleSubmissionInterval:         l2OutputOracleSubmissionInterval,
		L2OutputOracleStartingBlockNumber:        0,
		FinalizationPeriodSeconds:                finalizationPeriodSeconds,
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
		L1UsdcAddr:                               l1UsdcAddr,
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
	}

	return defaultTemplate

}

func loginAWS(awsLoginInputs *types.AWSLogin) (*aws.AccountProfile, error) {
	fmt.Println("Authenticating AWS account...")
	awsProfileAccount, err := aws.LoginAWS(awsLoginInputs.AccessKey, awsLoginInputs.SecretKey, awsLoginInputs.Region, awsLoginInputs.DefaultFormat)
	if err != nil {
		return nil, fmt.Errorf("Failed to authenticate AWS credentials: %s", err)
	}

	return awsProfileAccount, nil
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
	writer.WriteString(fmt.Sprintf("export TF_VAR_thanos_stack_name=\"%s\"\n", config.ThanosStackName))
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
	writer.WriteString(fmt.Sprintf("export TF_VAR_vpc_name=\"${TF_VAR_thanos_stack_name}/VPC\"\n"))

	writer.WriteString(fmt.Sprintf("export TF_VAR_eks_cluster_admins='[\"%s\"]'\n", config.EksClusterAdmins))

	writer.WriteString(fmt.Sprintf("export TF_VAR_genesis_file_path=\"%s\"\n", "config-files/genesis.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_rollup_file_path=\"%s\"\n", "config-files/rollup.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_prestate_file_path=\"%s\"\n", "config-files/prestate.json"))
	writer.WriteString(fmt.Sprintf("export TF_VAR_prestate_hash=\"%s\"\n", "0x03ab262ce124af0d5d328e09bf886a2b272fe960138115ad8b94fdc3034e3155"))

	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_deployments_path=\"%s\"\n", config.DeploymentsPath))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_rpc_url=\"%s\"\n", config.L1RpcUrl))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_rpc_provider=\"%s\"\n", config.L1RpcProvider))
	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_l1_beacon_url=\"%s\"\n", config.L1BeaconUrl))

	err = writer.Flush()
	if err != nil {
		return err
	}
	fmt.Println("Environment configuration file (.envrc) has been successfully generated!")
	return nil
}
