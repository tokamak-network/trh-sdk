package utils

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/constants"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"

	ethCommon "github.com/ethereum/go-ethereum/common"
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

func GetAddressFromPrivateKey(privateKeyHex string) (ethCommon.Address, error) {
	trimmedPrivateKey := strings.TrimPrefix(privateKeyHex, "0x")
	privateKey, err := crypto.HexToECDSA(trimmedPrivateKey)
	if err != nil {
		return ethCommon.Address{}, fmt.Errorf("failed to parse private key: %w", err)
	}
	publicKey := privateKey.PublicKey
	address := crypto.PubkeyToAddress(publicKey)
	return address, nil
}
