package utils

import (
	"math/big"

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
