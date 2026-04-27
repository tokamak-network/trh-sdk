package thanos

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestBuildBatcherDAConfig_UsesCalldata verifies that op-batcher is always configured
// with calldata DA rather than blobs. Sepolia blob fees spike to astronomical values
// (~4e25 wei), causing either ErrBlobBaseFeeTooHigh (with threshold) or
// "insufficient funds" (without threshold, since BlobFeeCap = 4 * blob_base_fee).
func TestBuildBatcherDAConfig_UsesCalldata(t *testing.T) {
	useBlobs, daType := buildBatcherDAConfig()

	require.False(t, useBlobs, "UseBlobs must be false when using calldata DA")
	require.Equal(t, "calldata", daType, "DA type must be calldata for Sepolia reliability")
}
