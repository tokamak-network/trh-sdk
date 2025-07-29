package types

type ChainInformation struct {
	L2RpcUrl      string `json:"l2_rpc_url"`
	BridgeUrl     string `json:"bridge_url,omitempty"`
	BlockExplorer string `json:"block_explorer,omitempty"`
}
