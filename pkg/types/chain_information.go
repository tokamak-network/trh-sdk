package types

type ChainInformation struct {
	L1ChainID      int    `json:"l1_chain_id"`
	L2RpcUrl       string `json:"l2_rpc_url"`
	L2ChainID      int    `json:"l2_chain_id"`
	BridgeUrl      string `json:"bridge_url,omitempty"`
	BlockExplorer  string `json:"block_explorer,omitempty"`
	RollupFilePath string `json:"rollup_file_path,omitempty"`
	DeploymentPath string `json:"deployment_path,omitempty"`
	MonitoringUrl  string `json:"monitoring_url,omitempty"`
}
