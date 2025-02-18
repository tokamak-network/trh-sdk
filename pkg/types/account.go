package types

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"log"
)

type Account struct {
	Address    string
	Balance    string
	PrivateKey string
}

func GetAccountMap(ctx context.Context, client *ethclient.Client, seedPhrase string) (map[int]Account, error) {
	// Validate the mnemonic
	if !bip39.IsMnemonicValid(seedPhrase) {
		return nil, errors.New("invalid seed phrase")
	}
	accounts := make(map[int]Account)

	seed := bip39.NewSeed(seedPhrase, "")

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		log.Printf("Failed to create master key: %v", err)
		return nil, err
	}

	hardened := func(i uint32) uint32 {
		return i + 0x80000000
	}

	purposeKey, err := masterKey.NewChildKey(hardened(44)) // 44'
	if err != nil {
		log.Printf("Failed to derive purpose key: %v", err)
		return nil, err
	}
	coinTypeKey, err := purposeKey.NewChildKey(hardened(60)) // 60'
	if err != nil {
		log.Printf("Failed to derive coin type key: %v", err)
		return nil, err
	}
	accountKey, err := coinTypeKey.NewChildKey(hardened(0)) // 0'
	if err != nil {
		log.Printf("Failed to derive account key: %v", err)
		return nil, err
	}
	changeKey, err := accountKey.NewChildKey(0) // External (0)
	if err != nil {
		log.Printf("Failed to derive change key: %v", err)
		return nil, err
	}

	accountCount := 10

	for i := 0; i < accountCount; i++ {
		childKey, err := changeKey.NewChildKey(uint32(i))
		if err != nil {
			log.Printf("Failed to derive key for index %d: %v", i, err)
			return nil, err
		}

		privateKey, err := crypto.ToECDSA(childKey.Key)
		if err != nil {
			log.Printf("Failed to convert key to ECDSA: %v", err)
			return nil, err
		}
		publicKey := privateKey.Public().(*ecdsa.PublicKey)
		address := crypto.PubkeyToAddress(*publicKey)

		balance, err := client.BalanceAt(ctx, address, nil)
		if err != nil {
			log.Printf("Failed to get balance: %v", err)
			return nil, err
		}
		account := Account{
			Address:    address.Hex(),
			Balance:    fmt.Sprintf("%.2f", utils.WeiToEther(balance)),
			PrivateKey: fmt.Sprintf("%064x", privateKey.D),
		}

		accounts[i] = account

	}

	return accounts, nil
}
