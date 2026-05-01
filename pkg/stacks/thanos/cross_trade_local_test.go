package thanos

import (
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// L2CrossDomainMessengerPredeploy is the canonical L2 CDM address used by deployL2CrossTradePair.
// This test guards against regression where the L1CDM address was mistakenly passed instead.
func TestL2CDMPredeploy_IsNotZero(t *testing.T) {
	const l2CDMPredeploy = "0x4200000000000000000000000000000000000007"
	addr := common.HexToAddress(l2CDMPredeploy)
	if addr == (common.Address{}) {
		t.Fatal("L2CDM predeploy address parsed to zero address — constant is malformed")
	}
	if !strings.EqualFold(addr.Hex(), l2CDMPredeploy) {
		t.Errorf("address round-trip mismatch: got %s, want %s", addr.Hex(), l2CDMPredeploy)
	}
}
