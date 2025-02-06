package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	hdwallet "github.com/miguelmota/go-ethereum-hdwallet"
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

func (w *wallet) getAddress(index uint8) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.hdWallet.Derive(path, false)
	return account.Address.Hex()
}

func (w *wallet) getPrivateKey(index uint8) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.hdWallet.Derive(path, false)
	privateKey, _ := w.hdWallet.PrivateKeyHex(account)

	return privateKey
}

type accountsIndex struct {
	admin      uint8
	sequencer  uint8
	batcher    uint8
	proposer   uint8
	challenger uint8
}

func selectOperatorAccounts(l1RPC string, seed string) (*accountsIndex, error) {
	client, err := ethclient.Dial(l1RPC)
	if err != nil {
		return nil, err
	}

	w, err := hdwallet.NewFromMnemonic(seed)
	if err != nil {
		return nil, err
	}

	wallet := &wallet{
		hdWallet: w,
	}

	var admin, sequencer, batcher, proposer, challenger uint8

	count := uint8(0)
	index := uint8(0)
	fmt.Println("Select admin acount from the following ones[minimum 0.6 ETH] (default: 0)")
	for {
		if count > 10 {
			break
		}
		address := wallet.getAddress(index)
		account := common.HexToAddress(address)
		balance, _ := client.BalanceAt(context.Background(), account, nil)
		fmt.Printf("\t%d. %s(%.2f ETH)\n", index, address, weiToEther(balance))
		count++
		index++
	}
	fmt.Print("Enter the number: ")
	fmt.Scanf("%d", &admin)
	count = 0
	index = 0

	fmt.Println("Select sequencer acount from the following ones (default: 0)")
	for {
		if count > 10 {
			break
		}
		address := wallet.getAddress(count)
		account := common.HexToAddress(address)
		balance, _ := client.BalanceAt(context.Background(), account, nil)
		fmt.Printf("\t%d. %s(%.2f ETH)\n", count, address, weiToEther(balance))
		count++
	}
	fmt.Print("Enter the number: ")
	fmt.Scanf("%d", &admin)
	count = 0

	return &accountsIndex{
		admin:      admin,
		sequencer:  sequencer,
		batcher:    batcher,
		proposer:   proposer,
		challenger: challenger,
	}, nil
}

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cliCfg := newDeployContractsCLIConfig(cmd)
		cfg, err := inputConfig(cliCfg)
		if err != nil {
			return err
		}
		fmt.Println(cfg.stack)

		accounts, _ := selectOperatorAccounts(cfg.l1RPCurl, cfg.seed)
		fmt.Println(accounts)

		return nil
	}
}
