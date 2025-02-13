package types

import (
	"fmt"
	hdwallet "github.com/ethereum-optimism/go-ethereum-hdwallet"
)

type Wallet struct {
	HdWallet *hdwallet.Wallet
}

func (w *Wallet) GetAddress(index int) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.HdWallet.Derive(path, false)
	return account.Address.Hex()
}

func (w *Wallet) getPrivateKey(index int) string {
	path := hdwallet.MustParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	account, _ := w.HdWallet.Derive(path, false)
	privateKey, _ := w.HdWallet.PrivateKeyHex(account)

	return privateKey
}
