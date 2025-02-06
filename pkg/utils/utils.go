package utils

import (
	"math/big"
	"os/exec"
	"strings"

	"github.com/ethereum/go-ethereum/params"
)

func ExecuteCommand(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func weiToEther(wei *big.Int) *big.Float {
	ether := new(big.Float).SetInt(wei)
	wetiToEtherFactor := new(big.Float).SetInt(big.NewInt(params.Ether))
	ether.Quo(ether, wetiToEtherFactor)
	return ether
}
