package thanos

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

var estimatedDeployContracts = new(big.Int).SetInt64(80_000_000)
var zeroBalance = new(big.Int).SetInt64(0)

var mapAccountIndexes = map[int]string{
	0: "Admin",
	1: "Sequencer",
	2: "Batcher",
	3: "Proposer",
	4: "Challenger",
}

func displayAccounts(accounts map[int]types.Account) {
	sortedAccounts := make([]types.Account, len(accounts))
	for i, account := range accounts {
		sortedAccounts[i] = account
	}

	for i, account := range sortedAccounts {
		balance, _ := new(big.Int).SetString(account.Balance, 10)
		fmt.Printf("\t%d. %s(%.4f ETH)\n", i, account.Address, utils.WeiToEther(balance))
	}
}

func selectAccounts(ctx context.Context, client *ethclient.Client, enableFraudProof bool, seed string) (types.OperatorMap, error) {
	fmt.Println("Retrieving accounts...")
	accounts, err := types.GetAccountMap(ctx, client, seed)
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

	operators := make(types.OperatorMap)

	displayAccounts(accounts)
	for i := 0; i < len(prompts); i++ {
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
			operators[operator] = &types.IndexAccount{
				Index:      selectingIndex,
				Address:    selectedAccount.Address,
				PrivateKey: selectedAccount.PrivateKey,
			}
			break
		}
	}

	sortedOperators := make([]*types.IndexAccount, len(operators))
	for i, operator := range operators {
		sortedOperators[i] = operator
	}
	for i, operator := range sortedOperators {
		fmt.Printf("%s account address: %s\n", mapAccountIndexes[i], operator.Address)
	}

	return operators, nil
}

func makeDeployContractConfigJsonFile(ctx context.Context, l1Provider *ethclient.Client, operators types.OperatorMap, deployContractTemplate *types.DeployConfigTemplate) error {
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
	latest, err := l1Provider.BlockByNumber(ctx, nil)
	if err != nil {
		fmt.Println("Error retrieving latest block")
		return err
	}

	deployContractTemplate.L1StartingBlockTag = latest.Hash().Hex()
	deployContractTemplate.L2OutputOracleStartingTimestamp = latest.Time()

	file, err := os.Create("deploy-config.json")
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

func initDeployConfigTemplate(deployConfigInputs *DeployContractsInput, l2ChainId uint64) *types.DeployConfigTemplate {
	var (
		chainConfiguration = deployConfigInputs.ChainConfiguration
	)

	if chainConfiguration == nil {
		panic("ChainConfiguration is empty")
	}

	var (
		l2BlockTime                      = chainConfiguration.L2BlockTime
		l1ChainId                        = deployConfigInputs.l1ChainID
		l2OutputOracleSubmissionInterval = chainConfiguration.GetL2OutputOracleSubmissionInterval()
		finalizationPeriods              = chainConfiguration.GetFinalizationPeriodSeconds()
		enableFraudProof                 = deployConfigInputs.fraudProof
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

	writer.WriteString(fmt.Sprintf("export TF_VAR_stack_deployments_path=\"%s\"\n", config.DeploymentsPath))
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
	} else {
		err := utils.PullLatestCode(repositoryName)
		if err != nil {
			fmt.Printf("Error while pulling the latest code for repository %s: %v\n", repositoryName, err)
			return err
		}
	}
	fmt.Printf("\r✅ Clone the %s repository successfully \n", repositoryName)

	return nil
}
