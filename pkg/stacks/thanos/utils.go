package thanos

import (
	"fmt"
	"math/big"

	"github.com/tokamak-network/trh-sdk/pkg/utils"

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
