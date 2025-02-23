package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// https://github.com/tokamak-network/tokamak-thanos/blob/main/op-service/sources/receipts_rpc.go#L173
const (
	RPCKindAlchemy    string = "alchemy"
	RPCKindQuickNode  string = "quicknode"
	RPCKindInfura     string = "infura"
	RPCKindParity     string = "parity"
	RPCKindNethermind string = "nethermind"
	RPCKindDebugGeth  string = "debug_geth"
	RPCKindErigon     string = "erigon"
	RPCKindBasic      string = "basic"    // try only the standard most basic receipt fetching
	RPCKindAny        string = "any"      // try any method available
	RPCKindStandard   string = "standard" // try standard methods, including newer optimized standard RPC methods
)

func DetectRPCKind(rpcURL string) string {
	requestBody := []byte(`{"jsonrpc":"2.0","method":"web3_clientVersion","params":[],"id":1}`)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error querying RPC:", err)
		return RPCKindStandard
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return RPCKindStandard
	}
	var rpcResponse struct {
		Result string `json:"result"`
	}
	err = json.Unmarshal(body, &rpcResponse)
	if err != nil {
		return RPCKindStandard
	}

	clientVersion := rpcResponse.Result

	switch {
	case strings.Contains(rpcURL, "alchemy"):
		return RPCKindAlchemy
	case strings.Contains(rpcURL, "quicknode"):
		return RPCKindQuickNode
	case strings.Contains(rpcURL, "infura"):
		return RPCKindInfura
	}

	switch {
	case strings.Contains(clientVersion, "Geth"):
		return RPCKindDebugGeth
	case strings.Contains(clientVersion, "OpenEthereum"), strings.Contains(clientVersion, "Parity"):
		return RPCKindParity
	case strings.Contains(clientVersion, "Nethermind"):
		return RPCKindNethermind
	case strings.Contains(clientVersion, "Erigon"):
		return RPCKindErigon
	default:
		return RPCKindStandard
	}
}
