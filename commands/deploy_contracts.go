package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
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
			balance: fmt.Sprintf("%.2f", weiToEther(balance)),
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

func ActionDeployContracts() cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		cliCfg := newDeployContractsCLIConfig(cmd)
		cfg, err := inputConfig(cliCfg)
		if err != nil {
			return err
		}
		fmt.Println(cfg.stack)

		operators := selectAccounts(cfg.l1RPCurl, cfg.seed)
		for k, v := range operators {
			fmt.Printf("%d index: %d, address: %s\n", k, v.index, v.address)
		}

		return nil
	}
}
