package utils

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"log"
	"math/big"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/params"
)

func WeiToEther(wei *big.Int) *big.Float {
	ether := new(big.Float).SetInt(wei)
	weiToEtherFactor := new(big.Float).SetInt(big.NewInt(params.Ether))
	ether.Quo(ether, weiToEtherFactor)
	return ether
}

func GWeiToEther(gwei *big.Int) *big.Float {
	ether := new(big.Float).SetInt(gwei)
	gweiToEtherFactor := new(big.Float).SetInt(big.NewInt(params.GWei))
	ether.Quo(ether, gweiToEtherFactor)
	return ether
}

func GWeiToWei(gwei *big.Int) *big.Int {
	return new(big.Int).Mul(gwei, new(big.Int).SetUint64(params.GWei))
}
func GenerateBatchInboxAddress(l2ChainId uint64) string {
	return fmt.Sprintf("%s%d", constants.BaseBatchInboxAddress[:len(constants.BaseBatchInboxAddress)-len(fmt.Sprintf("%d", l2ChainId))], l2ChainId)
}

func GetAddressFromPrivateKey(privateKeyStr string) (string, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyStr)
	if err != nil {
		log.Printf("Error converting hex to ECDSA: %v", err)
		return "", err
	}
	if privateKey == nil {
		log.Println("Private key is nil")
		return "", errors.New("private key is nil")
	}

	publicKey := privateKey.Public().(*ecdsa.PublicKey)
	address := crypto.PubkeyToAddress(*publicKey)

	return address.Hex(), nil
}

func GetAccountMap(ctx context.Context, client *ethclient.Client, seedPhrase string) (map[int]types.Account, error) {
	// Validate the mnemonic seed phrase
	if !bip39.IsMnemonicValid(seedPhrase) {
		return nil, errors.New("invalid mnemonic seed phrase")
	}
	accounts := make(map[int]types.Account)

	seed := bip39.NewSeed(seedPhrase, "")

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		log.Printf("Error creating master key: %v", err)
		return nil, err
	}

	hardened := func(i uint32) uint32 {
		return i + 0x80000000
	}

	purposeKey, err := masterKey.NewChildKey(hardened(44)) // 44'
	if err != nil {
		log.Printf("Error deriving purpose key: %v", err)
		return nil, err
	}
	coinTypeKey, err := purposeKey.NewChildKey(hardened(60)) // 60'
	if err != nil {
		log.Printf("Error deriving coin type key: %v", err)
		return nil, err
	}
	accountKey, err := coinTypeKey.NewChildKey(hardened(0)) // 0'
	if err != nil {
		log.Printf("Error deriving account key: %v", err)
		return nil, err
	}
	changeKey, err := accountKey.NewChildKey(0) // External (0)
	if err != nil {
		log.Printf("Error deriving change key: %v", err)
		return nil, err
	}

	accountCount := 10

	for i := range accountCount {
		childKey, err := changeKey.NewChildKey(uint32(i))
		if err != nil {
			log.Printf("Error deriving key for index %d: %v", i, err)
			return nil, err
		}

		privateKey, err := crypto.ToECDSA(childKey.Key)
		if err != nil {
			log.Printf("Error converting key to ECDSA format: %v", err)
			return nil, err
		}

		address, err := GetAddressFromPrivateKey(fmt.Sprintf("%064x", privateKey.D))
		if err != nil {
			fmt.Println("Error getting address from private key:", err)
			return nil, err
		}

		balance, err := client.BalanceAt(ctx, common.HexToAddress(address), nil)
		if err != nil {
			log.Printf("Error retrieving account balance: %v", err)
			return nil, err
		}
		account := types.Account{
			Address:    address,
			Balance:    balance.String(),
			PrivateKey: fmt.Sprintf("%064x", privateKey.D),
		}

		accounts[i] = account

	}

	return accounts, nil
}
