package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"os"
	"strconv"
	"strings"

	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/tokamak-thanos/op-chain-ops/genesis"
	"github.com/tokamak-network/trh-sdk/flags"
	"github.com/urfave/cli/v3"
)

type deployContractsCLIConfig struct {
	stack      string
	saveConfig bool
	network    string
}

type config struct {
	l1RPCurl   string
	seed       string
	falutProof bool

	*deployContractsCLIConfig
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

func newDeployContractsCLIConfig(cmd *cli.Command) *deployContractsCLIConfig {
	return &deployContractsCLIConfig{
		stack:      cmd.String(flags.StackFlag.Name),
		saveConfig: cmd.Bool(flags.SaveConfigFlag.Name),
		network:    cmd.String(flags.NetworkFlag.Name),
	}
}

func inputConfig(cliConfig *deployContractsCLIConfig) (*config, error) {
	fmt.Println("You are deploying the L1 contracts.")

	scanner := bufio.NewScanner(os.Stdin)
	var l1RPCUrl string
	fmt.Print("1. Please input your L1 RPC URL: ")
	scanner.Scan()
	l1RPCUrl = scanner.Text()

	fmt.Print("2. Please input your admin seed phrase: ")
	scanner.Scan()
	seed := scanner.Text()

	faultProof := false
	fmt.Print("3. Do you want to enable the fault-proof system on your chain? [Yes or No] (default: No): ")
	scanner.Scan()
	response := scanner.Text()
	if strings.Compare(response, "Yes") == 0 {
		faultProof = true
	} else if strings.Compare(response, "No") != 0 {
		fmt.Println(`The response must be "Yes" or "No"`)
		return nil, inputCommandError
	}

	return &config{
		l1RPCurl:                 l1RPCUrl,
		seed:                     seed,
		falutProof:               faultProof,
		deployContractsCLIConfig: cliConfig,
	}, nil
}

type wallet struct {
	hdWallet *hdwallet.Wallet
}

func (w *wallet) getAddress(index int) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.hdWallet.Derive(path, false)
	return account.Address.Hex()
}

func (w *wallet) getPrivateKey(index int) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.hdWallet.Derive(path, false)
	privateKey, _ := w.hdWallet.PrivateKeyHex(account)

	return privateKey
}

type account struct {
	address string
	balance string
}

type operator int

const (
	admin operator = iota
	sequencer
	batcher
	proposer
	challenger
)

type indexAccount struct {
	index   int
	address string
}

type operatorMap map[operator]indexAccount

func getAccountMap(l1RPC string, seed string) map[int]account {
	client, err := ethclient.Dial(l1RPC)
	if err != nil {
		return nil
	}

	w, err := hdwallet.NewFromMnemonic(seed)
	if err != nil {
		return nil
	}

	wallet := &wallet{
		hdWallet: w,
	}

	accounts := make(map[int]account)
	for i := 0; i < 16; i++ {
		hexAddress := wallet.getAddress(i)
		address := common.HexToAddress(hexAddress)
		balance, _ := client.BalanceAt(context.Background(), address, nil)
		accounts[i] = account{
			address: string(hexAddress),
			balance: fmt.Sprintf("%.2f", utils.WeiToEther(balance)),
		}
	}

	return accounts
}

func displayAccounts(selectedAccounts map[int]bool, accounts map[int]account) {
	count := 0
	for i := 0; i < len(accounts) && count < 10; i++ {
		if !selectedAccounts[i] {
			account := accounts[i]
			fmt.Printf("\t%d. %s(%s ETH)\n", i, account.address, account.balance)
			count++
		}
	}
}

func selectAccounts(l1RPC string, seed string) operatorMap {
	fmt.Println("Getting accounts...")
	accounts := getAccountMap(l1RPC, seed)

	selectedAccounts := make(map[int]bool)
	scanner := bufio.NewScanner(os.Stdin)

	prompts := []string{
		"Select admin acount from the following ones[minimum 0.6 ETH]",
		"Select sequencer acount from the following ones",
		"Select batcher acount from the following ones",
		"Select proposer acount from the following ones",
		"Select challenger acount from the following ones",
	}

	operators := make(operatorMap)

	for i := 0; i < 5; i++ {
		fmt.Println(prompts[i])
		displayAccounts(selectedAccounts, accounts)
		fmt.Print("Enter the number: ")
		scanner.Scan()
		input := scanner.Text()

		selectedIndex, err := strconv.Atoi(input)
		if err != nil || selectedIndex < 0 || selectedIndex >= len(accounts) || selectedAccounts[selectedIndex] {
			fmt.Println("Invalid selection. Please try again.")
			i--
			continue
		}

		selectedAccounts[selectedIndex] = true
		operators[operator(i)] = indexAccount{
			index:   selectedIndex,
			address: accounts[selectedIndex].address,
		}
	}

	return operators
}

type deployConfigTemplate struct {
	NativeTokenName                          string   `json:"nativeTokenName"`
	NativeTokenSymbol                        string   `json:"nativeTokenSymbol"`
	NativeTokenAddress                       string   `json:"nativeTokenAddress"`
	FinalSystemOwner                         string   `json:"finalSystemOwner"`
	SuperchainConfigGuardian                 string   `json:"superchainConfigGuardian"`
	L1StartingBlockTag                       string   `json:"l1StartingBlockTag,omitempty"`
	L1ChainID                                uint64   `json:"l1ChainId"`
	L2ChainID                                uint64   `json:"l2ChainId"`
	L2BlockTime                              uint64   `json:"l2BlockTime"`
	L1BlockTime                              uint64   `json:"l1BlockTime"`
	MaxSequencerDrift                        uint64   `json:"maxSequencerDrift"`
	SequencerWindowSize                      uint64   `json:"sequencerWindowSize"`
	ChannelTimeout                           uint64   `json:"channelTimeout"`
	P2pSequencerAddress                      string   `json:"p2pSequencerAddress"`
	BatchInboxAddress                        string   `json:"batchInboxAddress"`
	BatchSenderAddress                       string   `json:"batchSenderAddress"`
	L2OutputOracleSubmissionInterval         uint64   `json:"l2OutputOracleSubmissionInterval"`
	L2OutputOracleStartingTimestamp          uint64   `json:"l2OutputOracleStartingTimestamp"`
	L2OutputOracleStartingBlockNumber        uint64   `json:"l2OutputOracleStartingBlockNumber"`
	L2OutputOracleProposer                   string   `json:"l2OutputOracleProposer"`
	L2OutputOracleChallenger                 string   `json:"l2OutputOracleChallenger"`
	FinalizationPeriodSeconds                uint64   `json:"finalizationPeriodSeconds"`
	ProxyAdminOwner                          string   `json:"proxyAdminOwner"`
	BaseFeeVaultRecipient                    string   `json:"baseFeeVaultRecipient"`
	L1FeeVaultRecipient                      string   `json:"l1FeeVaultRecipient"`
	SequencerFeeVaultRecipient               string   `json:"sequencerFeeVaultRecipient"`
	BaseFeeVaultMinimumWithdrawalAmount      string   `json:"baseFeeVaultMinimumWithdrawalAmount"`
	L1FeeVaultMinimumWithdrawalAmount        string   `json:"l1FeeVaultMinimumWithdrawalAmount"`
	SequencerFeeVaultMinimumWithdrawalAmount string   `json:"sequencerFeeVaultMinimumWithdrawalAmount"`
	BaseFeeVaultWithdrawalNetwork            uint64   `json:"baseFeeVaultWithdrawalNetwork"`
	L1FeeVaultWithdrawalNetwork              uint64   `json:"l1FeeVaultWithdrawalNetwork"`
	SequencerFeeVaultWithdrawalNetwork       uint64   `json:"sequencerFeeVaultWithdrawalNetwork"`
	EnableGovernance                         bool     `json:"enableGovernance"`
	GovernanceTokenName                      string   `json:"governanceTokenName,omitempty"`
	GovernanceTokenSymbol                    string   `json:"governanceTokenSymbol,omitempty"`
	GovernanceTokenOwner                     string   `json:"governanceTokenOwner,omitempty"`
	L2GenesisBlockGasLimit                   string   `json:"l2GenesisBlockGasLimit"`
	L2GenesisBlockBaseFeePerGas              string   `json:"l2GenesisBlockBaseFeePerGas"`
	GasPriceOracleOverhead                   uint64   `json:"gasPriceOracleOverhead"`
	GasPriceOracleScalar                     uint64   `json:"gasPriceOracleScalar"`
	Eip1559Denominator                       uint64   `json:"eip1559Denominator"`
	Eip1559Elasticity                        uint64   `json:"eip1559Elasticity"`
	Eip1559DenominatorCanyon                 uint64   `json:"eip1559DenominatorCanyon"`
	L2GenesisRegolithTimeOffset              string   `json:"l2GenesisRegolithTimeOffset"`
	L2GenesisCanyonTimeOffset                string   `json:"l2GenesisCanyonTimeOffset"`
	L2GenesisDeltaTimeOffset                 string   `json:"l2GenesisDeltaTimeOffset"`
	L2GenesisEcotoneTimeOffset               string   `json:"l2GenesisEcotoneTimeOffset"`
	SystemConfigStartBlock                   uint64   `json:"systemConfigStartBlock"`
	RequiredProtocolVersion                  string   `json:"requiredProtocolVersion"`
	RecommendedProtocolVersion               string   `json:"recommendedProtocolVersion"`
	FaultGameAbsolutePrestate                string   `json:"faultGameAbsolutePrestate"`
	FaultGameMaxDepth                        uint64   `json:"faultGameMaxDepth"`
	FaultGameClockExtension                  uint64   `json:"faultGameClockExtension"`
	FaultGameMaxClockDuration                uint64   `json:"faultGameMaxClockDuration"`
	FaultGameGenesisBlock                    uint64   `json:"faultGameGenesisBlock"`
	FaultGameGenesisOutputRoot               string   `json:"faultGameGenesisOutputRoot"`
	FaultGameSplitDepth                      uint64   `json:"faultGameSplitDepth"`
	FaultGameWithdrawalDelay                 uint64   `json:"faultGameWithdrawalDelay"`
	PreimageOracleMinProposalSize            uint64   `json:"preimageOracleMinProposalSize"`
	PreimageOracleChallengePeriod            uint64   `json:"preimageOracleChallengePeriod"`
	ProofMaturityDelaySeconds                uint64   `json:"proofMaturityDelaySeconds"`
	DisputeGameFinalityDelaySeconds          uint64   `json:"disputeGameFinalityDelaySeconds"`
	RespectedGameType                        uint64   `json:"respectedGameType"`
	UseFaultProofs                           bool     `json:"useFaultProofs"`
	L1UsdcAddr                               string   `json:"l1UsdcAddr"`
	UsdcTokenName                            string   `json:"usdcTokenName"`
	NewPauser                                string   `json:"newPauser"`
	NewBlacklister                           string   `json:"newBlacklister"`
	MasterMinterOwner                        string   `json:"masterMinterOwner"`
	FiatTokenOwner                           string   `json:"fiatTokenOwner"`
	FactoryV2addr                            string   `json:"factoryV2addr"`
	NativeCurrencyLabelBytes                 []uint64 `json:"nativeCurrencyLabelBytes"`
	UniswapV3FactoryOwner                    string   `json:"uniswapV3FactoryOwner"`
	UniswapV3FactoryFee500                   uint64   `json:"uniswapV3FactoryFee500"`
	UniswapV3FactoryTickSpacing10            uint64   `json:"uniswapV3FactoryTickSpacing10"`
	UniswapV3FactoryFee3000                  uint64   `json:"uniswapV3FactoryFee3000"`
	UniswapV3FactoryTickSpacing60            uint64   `json:"uniswapV3FactoryTickSpacing60"`
	UniswapV3FactoryFee10000                 uint64   `json:"uniswapV3FactoryFee10000"`
	UniswapV3FactoryTickSpacing200           uint64   `json:"uniswapV3FactoryTickSpacing200"`
	UniswapV3FactoryFee100                   uint64   `json:"uniswapV3FactoryFee100"`
	UniswapV3FactoryTickSpacing1             uint64   `json:"uniswapV3FactoryTickSpacing1"`
	PairInitCodeHash                         string   `json:"pairInitCodeHash"`
	PoolInitCodeHash                         string   `json:"poolInitCodeHash"`
	UniversalRouterRewardsDistributor        string   `json:"universalRouterRewardsDistributor"`
}

type deployConfigManager struct {
	defaultDeployConfig *deployConfigTemplate
	deployConfig        *genesis.DeployConfig
}

func (dcm *deployConfigManager) makeDeplyConfigJson(operators operatorMap) {
	for role, account := range operators {
		switch role {
		case admin:
			dcm.defaultDeployConfig.FinalSystemOwner = account.address
			dcm.defaultDeployConfig.SuperchainConfigGuardian = account.address
			dcm.defaultDeployConfig.ProxyAdminOwner = account.address
			dcm.defaultDeployConfig.BaseFeeVaultRecipient = account.address
			dcm.defaultDeployConfig.L1FeeVaultRecipient = account.address
			dcm.defaultDeployConfig.SequencerFeeVaultRecipient = account.address
			dcm.defaultDeployConfig.NewPauser = account.address
			dcm.defaultDeployConfig.NewBlacklister = account.address
			dcm.defaultDeployConfig.MasterMinterOwner = account.address
			dcm.defaultDeployConfig.FiatTokenOwner = account.address
			dcm.defaultDeployConfig.UniswapV3FactoryOwner = account.address
			dcm.defaultDeployConfig.UniversalRouterRewardsDistributor = account.address
		case sequencer:
			dcm.defaultDeployConfig.P2pSequencerAddress = account.address
		case batcher:
			dcm.defaultDeployConfig.BatchSenderAddress = account.address
		case proposer:
			dcm.defaultDeployConfig.L2OutputOracleProposer = account.address
		case challenger:
			dcm.defaultDeployConfig.L2OutputOracleChallenger = account.address
		}
	}

	file, _ := os.Create("deploy-config.json")
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.Encode(dcm.defaultDeployConfig)
}

func newDeployConfigManager(cfg *config) *deployConfigManager {
	var l1ChainId uint64
	var l2ChainId uint64
	var batchInboxAddress string
	var finalizationPeriodSeconds uint64
	var nativeTokenAddress string
	var l2OutputOracleSubmissionInterval uint64
	var l1UsdcAddr string
	useFaultProofs := cfg.falutProof
	basebatchInboxAddress := "0xff0000000000000000000000000000000000000"

	if cfg.network == "testnet" {
		l1ChainId = 11155111
		finalizationPeriodSeconds = 12
		nativeTokenAddress = "0xa30fe40285b8f5c0457dbc3b7c8a280373c40044"
		l2OutputOracleSubmissionInterval = 120
		l1UsdcAddr = "0x94a9D9AC8a22534E3FaCa9F4e7F2E2cf85d5E4C8"
	} else if cfg.network == "mainnet" {
		l1ChainId = 1
		finalizationPeriodSeconds = 604800
		nativeTokenAddress = "0x2be5e8c109e2197D077D13A82dAead6a9b3433C5"
		l2OutputOracleSubmissionInterval = 10800
		l1UsdcAddr = "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"
	}

	l2ChainId = 111551119876
	batchInboxAddress = fmt.Sprintf("%s%d", basebatchInboxAddress[:len(basebatchInboxAddress)-len(fmt.Sprintf("%d", l2ChainId))], l2ChainId)

	defaultTemplate := deployConfigTemplate{
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
	}

	return &deployConfigManager{
		defaultDeployConfig: &defaultTemplate,
	}
}

// CLI Action method
func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		// Get CLI config
		cliCfg := newDeployContractsCLIConfig(cmd)
		cfg, err := inputConfig(cliCfg)
		if err != nil {
			return err
		}

		// Select operators Accounts
		operators := selectAccounts(cfg.l1RPCurl, cfg.seed)
		for k, v := range operators {
			fmt.Printf("%d index: %d, address: %s\n", k, v.index, v.address)
		}

		dManager := newDeployConfigManager(cfg)
		dManager.makeDeplyConfigJson(operators)

		return nil
	}
}
