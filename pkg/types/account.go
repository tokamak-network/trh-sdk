package types

import (
	"context"
	"fmt"
	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type Account struct {
	Address string
	Balance string
}

func GetAccountMap(ctx context.Context, l1RPC string, seed string) map[int]Account {
	client, err := ethclient.Dial(l1RPC)
	if err != nil {
		return nil
	}

	w, err := hdwallet.NewFromMnemonic(seed)
	if err != nil {
		return nil
	}

	wallet := &Wallet{
		HdWallet: w,
	}

	accounts := make(map[int]Account)
	for i := 0; i < 16; i++ {
		hexAddress := wallet.GetAddress(i)
		address := common.HexToAddress(hexAddress)
		balance, _ := client.BalanceAt(ctx, address, nil)
		accounts[i] = Account{
			Address: hexAddress,
			Balance: fmt.Sprintf("%.2f", utils.WeiToEther(balance)),
		}
	}

	return accounts
}
