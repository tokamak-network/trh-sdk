package thanos

import (
	"fmt"
	"math/big"

	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

var estimatedDeployContracts = new(big.Int).SetInt64(80_000_000)
var zeroBalance = new(big.Int).SetInt64(0)

// deployGasPriceMultiplier is applied to the L1 SuggestGasPrice value before
// passing it to tokamak-deployer as a fixed gas price via --gas-price. Using
// 2× the suggested price gives enough headroom that the deployer's
// bump-on-timeout retry loop (which replaces stuck TXs with a 1.25× bumped
// version, up to 3 attempts) effectively never needs to fire.
//
// Rationale for 2×: a 10-minute deploy window rarely sees suggested price
// more than double on Sepolia. The balance precheck at 3× keeps the
// affordability envelope wider than the cost actually incurred.
var deployGasPriceMultiplier = big.NewInt(2)

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
