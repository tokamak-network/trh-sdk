package crosstrade

import (
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

const (
	DeployL1CrossTradeL2L1 = "DeployL1CrossTrade_L2L1.s.sol"
	DeployL2CrossTradeL2L1 = "DeployL2CrossTrade_L2L1.s.sol"
	DeployL1CrossTradeL2L2 = "DeployL1CrossTrade_L2L2.s.sol"
	DeployL2CrossTradeL2L2 = "DeployL2CrossTrade_L2L2.s.sol"
)

const (
	L2L2CrossTradeProxyL1ContractName = "L2toL2CrossTradeProxyL1"
	L2L2CrossTradeL1ContractName      = "L2toL2CrossTradeL1"
	L1L2CrossTradeProxyL1ContractName = "L1CrossTradeProxy"
	L1L2CrossTradeL1ContractName      = "L1CrossTrade"

	L2L2CrossTradeProxyL2ContractName = "L2toL2CrossTradeProxy"
	L2L2CrossTradeL2ContractName      = "L2toL2CrossTradeL2"
	L1L2CrossTradeProxyL2ContractName = "L2CrossTradeProxy"
	L1L2CrossTradeL2ContractName      = "L2CrossTrade"
)

const (
	L2L2ScriptPath = "scripts/foundry_scripts"
	L1L2ScriptPath = "scripts/foundry_scripts/L2L1"
)

func GetL1L2ContractFileName(mode constants.CrossTradeDeployMode) (string, string, error) {
	var l1ContractFileName, l2ContractFileName string
	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		l1ContractFileName = DeployL1CrossTradeL2L1
		l2ContractFileName = DeployL2CrossTradeL2L1
	case constants.CrossTradeDeployModeL2ToL2:
		l1ContractFileName = DeployL1CrossTradeL2L2
		l2ContractFileName = DeployL2CrossTradeL2L2
	default:
		return "", "", fmt.Errorf("invalid cross trade deploy mode: %s", mode)
	}
	return l1ContractFileName, l2ContractFileName, nil
}

func GetDeploymentScriptPath(mode constants.CrossTradeDeployMode) (string, error) {
	var deploymentScriptPath string
	switch mode {
	case constants.CrossTradeDeployModeL2ToL1:
		deploymentScriptPath = L1L2ScriptPath
	case constants.CrossTradeDeployModeL2ToL2:
		deploymentScriptPath = L2L2ScriptPath
	default:
		return "", fmt.Errorf("invalid cross trade deploy mode: %s", mode)
	}
	return deploymentScriptPath, nil
}
