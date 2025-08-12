package types

type ChainInformation struct {
	L2RpcUrl      string `json:"l2_rpc_url"`
	L2ChainID     int    `json:"l2_chain_id"`
	BridgeUrl     string `json:"bridge_url,omitempty"`
	BlockExplorer string `json:"block_explorer,omitempty"`
	MonitoringUrl string `json:"monitoring_url,omitempty"`
}
