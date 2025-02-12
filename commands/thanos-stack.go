package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"os"
	"strconv"
)

type ThanosStack struct {
	Network string
}

func NewThanosStack(network string) *ThanosStack {
	return &ThanosStack{
		Network: network,
	}
}

func (t *ThanosStack) Deploy() error {
	switch t.Network {
	case constants.LocalDevnet:
		return t.deployLocalDevnet()
	default:
		return fmt.Errorf("network %s is not supported", t.Network)
	}
}

func (t *ThanosStack) DeployContracts() error {
	if t.Network == constants.LocalDevnet {
		return fmt.Errorf("network %s doesn't need to deploy the contracts, please running `tokamak-cli-sdk deploy` instead", constants.LocalDevnet)
	}
	var err error
	//// STEP 1. Input the parameters
	cfg, err := inputConfig()
	if err != nil {
		return err
	}

	dManager := newDeployConfigManager(cfg, t.Network)

	// Select operators Accounts
	operators, err := selectAccounts(dManager.l1Provider, cfg.seed)
	if err != nil {
		return err
	}

	if len(operators) == 0 {
		return fmt.Errorf("no operator found")
	}

	for k, v := range operators {
		fmt.Printf("%d account: %d, address: %s\n", k, types.Operator(v.Index), v.Address)
	}

	dManager.makeDeplyConfigJson(operators)

	// STEP 2. Clone the repository
	err = t.cloneSourcecode()
	if err != nil {
		return err
	}

	// STEP 3. Build the contracts
	//doneCh := make(chan bool)
	//go utils.ShowLoadingAnimation(doneCh, "Building the contracts...")
	//_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh build")
	//if err != nil {
	//	doneCh <- true
	//	fmt.Print("\r❌ Build the contracts failed!       \n")
	//	return err
	//}
	//doneCh <- true
	//fmt.Print("\r✅ Build the contracts completed!       \n")

	// STEP 4. Deploy the contracts
	//go utils.ShowLoadingAnimation(doneCh, "Deploying the contracts...")
	gsAdminPrivateKey := operators[0].PrivateKey

	// STEP 4.1. Generate the .env file
	_, err = utils.ExecuteCommand("bash", "-c", fmt.Sprintf("cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && echo 'export GS_ADMIN_PRIVATE_KEY=%s' > .env && echo 'export L1_RPC_URL=%s' >> .env", gsAdminPrivateKey, cfg.l1RPCurl))
	if err != nil {
		//doneCh <- true
		fmt.Print("\r❌ Make .env file failed!       \n")
		return err
	}

	// STEP 4.2. Copy the config file into the scripts folder
	_, err = utils.ExecuteCommand("bash", "-c", "cp ./deploy-config.json tokamak-thanos/packages/tokamak/contracts-bedrock/scripts")
	if err != nil {
		//doneCh <- true
		fmt.Print("\r❌ Copy the config file successfully!       \n")
		return err
	}

	// STEP 4.3. Deploy contracts
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos/packages/tokamak/contracts-bedrock/scripts && bash ./start-deploy.sh deploy -e .env -c deploy-config.json")
	if err != nil {
		//doneCh <- true
		fmt.Print("\r❌ Build the contracts failed!       \n")
		return err
	}
	fmt.Print("\r✅ Deploy the contracts completed!       \n")

	//doneCh <- true
	return nil
}

func (t *ThanosStack) deployLocalDevnet() error {
	err := t.cloneSourcecode()
	if err != nil {
		return err
	}

	doneCh := make(chan bool)
	go utils.ShowLoadingAnimation(doneCh, "Installing the devnet packages...")
	_, err = utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && bash ./install-devnet-packages.sh")
	if err != nil {
		doneCh <- true
		fmt.Print("\r❌ Installation failed!       \n")
		return err
	}
	fmt.Print("\r✅ Installation completed!       \n")
	doneCh <- true

	go utils.ShowLoadingAnimation(doneCh, "Making the devnet up...")
	output, err := utils.ExecuteCommand("bash", "-c", "cd tokamak-thanos && make devnet-up")
	if err != nil {
		doneCh <- true
		fmt.Printf("\r❌ Make devnet failed!       \n Detail: %s", output)

		return err
	}

	fmt.Print("\r✅ Devnet up!       \n")

	return nil
}

func (t *ThanosStack) cloneSourcecode() error {
	doneCh := make(chan bool)
	defer close(doneCh)
	existingSourcecode, err := utils.CheckExistingSourceCode("tokamak-thanos")
	if err != nil {
		fmt.Println("Error checking existing source code")
		return err
	}

	if !existingSourcecode {
		go utils.ShowLoadingAnimation(doneCh, "Cloning the Thanos Stack repository...")
		err := utils.CloneRepo("https://github.com/tokamak-network/tokamak-thanos.git", "tokamak-thanos")
		doneCh <- true
		if err != nil {
			fmt.Println("Error cloning the repo")
			return err
		}
	}
	fmt.Print("\r✅ Clone the Thanos Stack repository successfully!       \n")

	return nil
}

type deployContractFlags struct {
	stack      string
	saveConfig bool
	network    string
}

type config struct {
	l1RPCurl   string
	seed       string
	falutProof bool
}

type deployContractsError struct {
	code    int
	message string
}

func (e *deployContractsError) Error() string {
	return fmt.Sprintf("%d: %s", e.code, e.message)
}

var (
	inputCommandError = &deployContractsError{
		code:    0,
		message: "Not vaild input",
	}
)

func inputConfig() (*config, error) {
	fmt.Println("You are deploying the L1 contracts.")

	fmt.Print("1. Please input your L1 RPC URL: ")
	l1RPCUrl, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning L1 RPC URL: %s", err)
		return nil, err
	}

	fmt.Print("2. Please input your admin seed phrase: ")
	seed, err := scanner.ScanString()
	if err != nil {
		fmt.Printf("Error scanning the seed phrase: %s", err)
		return nil, err
	}

	faultProof := false
	fmt.Print("3. Do you want to enable the fault-proof system on your chain? [Y or N] (default: N): ")
	faultProof, err = scanner.ScanBool()
	if err != nil {
		fmt.Printf("Error scanning the fault-proof system setting: %s", err)
		return nil, err
	}

	return &config{
		l1RPCurl:   l1RPCUrl,
		seed:       seed,
		falutProof: faultProof,
	}, nil
}

func displayAccounts(selectedAccounts map[int]bool, accounts map[int]types.Account) {
	count := 0
	for i := 0; i < len(accounts) && count < 10; i++ {
		if !selectedAccounts[i] {
			account := accounts[i]
			fmt.Printf("\t%d. %s(%s ETH)\n", i, account.Address, account.Balance)
			count++
		}
	}
}

func selectAccounts(client *ethclient.Client, seed string) (types.OperatorMap, error) {
	fmt.Println("Getting accounts...")
	accounts, err := types.GetAccountMap(context.Background(), client, seed)
	if err != nil {
		return nil, err
	}

	selectedAccounts := make(map[int]bool)

	prompts := []string{
		"Select admin acount from the following ones[minimum 0.6 ETH]",
		"Select sequencer acount from the following ones",
		"Select batcher acount from the following ones[recommend 0.3 ETH]",
		"Select proposer acount from the following ones[recommend 0.3 ETH]",
		"Select challenger acount from the following ones[recommend 0.3 ETH]",
	}

	operators := make(types.OperatorMap)

	for i := 0; i < 5; i++ {
		fmt.Println(prompts[i])
		displayAccounts(selectedAccounts, accounts)
		fmt.Print("Enter the number: ")
		input, err := scanner.ScanString()
		if err != nil {
			fmt.Printf("Failed to scan input: %s", err)
			return nil, err
		}

		selectedIndex, err := strconv.Atoi(input)
		if err != nil || selectedIndex < 0 || selectedIndex >= len(accounts) || selectedAccounts[selectedIndex] {
			fmt.Println("Invalid selection. Please try again.")
			i--
			continue
		}

		selectedAccounts[selectedIndex] = true
		operators[types.Operator(i)] = types.IndexAccount{
			Index:      selectedIndex,
			Address:    accounts[selectedIndex].Address,
			PrivateKey: accounts[selectedIndex].PrivateKey,
		}
	}

	return operators, nil
}

type deployConfigManager struct {
	defaultDeployConfig *types.DeployConfigTemplate
	//flags               *deployContractFlags
	l1Provider *ethclient.Client
}

func (dcm *deployConfigManager) makeDeplyConfigJson(operators types.OperatorMap) {
	for role, account := range operators {
		switch role {
		case types.Admin:
			dcm.defaultDeployConfig.FinalSystemOwner = account.Address
			dcm.defaultDeployConfig.SuperchainConfigGuardian = account.Address
			dcm.defaultDeployConfig.ProxyAdminOwner = account.Address
			dcm.defaultDeployConfig.BaseFeeVaultRecipient = account.Address
			dcm.defaultDeployConfig.L1FeeVaultRecipient = account.Address
			dcm.defaultDeployConfig.SequencerFeeVaultRecipient = account.Address
			dcm.defaultDeployConfig.NewPauser = account.Address
			dcm.defaultDeployConfig.NewBlacklister = account.Address
			dcm.defaultDeployConfig.MasterMinterOwner = account.Address
			dcm.defaultDeployConfig.FiatTokenOwner = account.Address
			dcm.defaultDeployConfig.UniswapV3FactoryOwner = account.Address
			dcm.defaultDeployConfig.UniversalRouterRewardsDistributor = account.Address
		case types.Sequencer:
			dcm.defaultDeployConfig.P2pSequencerAddress = account.Address
		case types.Batcher:
			dcm.defaultDeployConfig.BatchSenderAddress = account.Address
		case types.Proposer:
			dcm.defaultDeployConfig.L2OutputOracleProposer = account.Address
		case types.Challenger:
			dcm.defaultDeployConfig.L2OutputOracleChallenger = account.Address
		}
	}

	// fetch the latest block
	latest, err := dcm.l1Provider.BlockByNumber(context.Background(), nil)
	if err != nil {
		fmt.Println("Error fetching latest block")
		return
	}

	dcm.defaultDeployConfig.L1StartingBlockTag = latest.Hash().Hex()
	dcm.defaultDeployConfig.L2OutputOracleStartingTimestamp = latest.Time()

	file, err := os.Create("deploy-config.json")
	defer file.Close()
	if err != nil {
		fmt.Printf("Failed to create file: %s", err)
		return
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(dcm.defaultDeployConfig)
}

func newDeployConfigManager(cfg *config, network string) *deployConfigManager {
	var l1ChainId uint64
	var l2ChainId uint64
	var batchInboxAddress string
	var finalizationPeriodSeconds uint64
	var nativeTokenAddress string
	var l2OutputOracleSubmissionInterval uint64
	var l1UsdcAddr string
	useFaultProofs := cfg.falutProof
	basebatchInboxAddress := "0xff00000000000000000000000000000000000000"

	if network == "testnet" {
		l1ChainId = 11155111
		finalizationPeriodSeconds = 12
		nativeTokenAddress = "0xa30fe40285b8f5c0457dbc3b7c8a280373c40044"
		l2OutputOracleSubmissionInterval = 120
		l1UsdcAddr = "0x94a9D9AC8a22534E3FaCa9F4e7F2E2cf85d5E4C8"
	} else if network == "mainnet" {
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
		UseFaultProofs:                           useFaultProofs,
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
	}

	l1Client, err := ethclient.Dial(cfg.l1RPCurl)
	if err != nil {
		fmt.Printf("Failed to dial l1 client: %v\n", err)
		return nil
	}
	return &deployConfigManager{
		defaultDeployConfig: defaultTemplate,
		l1Provider:          l1Client,
	}
}
