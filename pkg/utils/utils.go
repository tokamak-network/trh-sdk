package utils

import (
	"github.com/ethereum/go-ethereum/params"
	"math/big"
)

func WeiToEther(wei *big.Int) *big.Float {
	ether := new(big.Float).SetInt(wei)
	wetiToEtherFactor := new(big.Float).SetInt(big.NewInt(params.Ether))
	ether.Quo(ether, wetiToEtherFactor)
	return ether
}
