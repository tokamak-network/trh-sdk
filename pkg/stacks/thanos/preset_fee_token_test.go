package thanos

import (
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func makeTestInput(preset, feeToken string) *DeployContractsInput {
	return &DeployContractsInput{
		L1RPCurl: "http://localhost:8545",
		ChainConfiguration: &types.ChainConfiguration{
			L2BlockTime:              2,
			L1BlockTime:              12,
			BatchSubmissionFrequency: 1440, // 120 * 12
			ChallengePeriod:          12,
			OutputRootFrequency:      240, // 120 * 2
		},
		Preset:   preset,
		FeeToken: feeToken,
	}
}

// TestInitDeployConfigTemplate_FeeTokenMapping verifies that each fee token
// correctly populates NativeToken* fields in the deploy config template.
func TestInitDeployConfigTemplate_FeeTokenMapping(t *testing.T) {
	l1ChainID := constants.EthereumSepoliaChainID
	l2ChainID := uint64(1234)

	tests := []struct {
		feeToken       string
		wantName       string
		wantSymbol     string
		wantAddrPrefix string // just check it's non-zero or zero
	}{
		{constants.FeeTokenTON, "Tokamak Network Token", "TON", "0xa30fe40"},
		{constants.FeeTokenETH, "Ether", "ETH", "0x000000000000000000000000000000000000000"},
		{constants.FeeTokenUSDT, "Tether USD", "USDT", "0xaa8e23"},
		{constants.FeeTokenUSDC, "USD Coin", "USDC", "0x1c7d4b"},
	}

	for _, tc := range tests {
		t.Run(tc.feeToken, func(t *testing.T) {
			input := makeTestInput(constants.PresetDeFi, tc.feeToken)
			tpl := initDeployConfigTemplate(input, l1ChainID, l2ChainID, "")

			if tpl.NativeTokenName != tc.wantName {
				t.Errorf("NativeTokenName: got %q, want %q", tpl.NativeTokenName, tc.wantName)
			}
			if tpl.NativeTokenSymbol != tc.wantSymbol {
				t.Errorf("NativeTokenSymbol: got %q, want %q", tpl.NativeTokenSymbol, tc.wantSymbol)
			}
			if !strings.HasPrefix(strings.ToLower(tpl.NativeTokenAddress), strings.ToLower(tc.wantAddrPrefix)) {
				t.Errorf("NativeTokenAddress: got %q, want prefix %q", tpl.NativeTokenAddress, tc.wantAddrPrefix)
			}
		})
	}
}

// TestInitDeployConfigTemplate_PresetGeneral verifies that DeFi-specific fields
// are cleared for the General preset.
func TestInitDeployConfigTemplate_PresetGeneral(t *testing.T) {
	input := makeTestInput(constants.PresetGeneral, constants.FeeTokenTON)
	tpl := initDeployConfigTemplate(input, constants.EthereumSepoliaChainID, 5678, "")

	if tpl.L1UsdcAddr != "0x0000000000000000000000000000000000000000" {
		t.Errorf("General preset should clear L1UsdcAddr, got %s", tpl.L1UsdcAddr)
	}
	if tpl.UniswapV3FactoryFee500 != 0 {
		t.Errorf("General preset should clear UniswapV3FactoryFee500, got %d", tpl.UniswapV3FactoryFee500)
	}
}

// TestInitDeployConfigTemplate_PresetGaming verifies that VRF/AA fields are
// populated for the Gaming preset.
func TestInitDeployConfigTemplate_PresetGaming(t *testing.T) {
	input := makeTestInput(constants.PresetGaming, constants.FeeTokenTON)
	input.VRFAdmin = "0xABCDEF"
	input.AAPaymasterSigner = "0x123456"

	tpl := initDeployConfigTemplate(input, constants.EthereumSepoliaChainID, 9999, "")

	if tpl.VRFAdmin != "0xABCDEF" {
		t.Errorf("VRFAdmin: got %q, want %q", tpl.VRFAdmin, "0xABCDEF")
	}
	if tpl.AAPaymasterSigner != "0x123456" {
		t.Errorf("AAPaymasterSigner: got %q, want %q", tpl.AAPaymasterSigner, "0x123456")
	}
}

// TestInitDeployConfigTemplate_PresetFull verifies Full preset sets VRF/AA fields.
func TestInitDeployConfigTemplate_PresetFull(t *testing.T) {
	input := makeTestInput(constants.PresetFull, constants.FeeTokenUSDC)
	input.VRFAdmin = "0xVRF"
	input.AAPaymasterSigner = "0xAA"

	tpl := initDeployConfigTemplate(input, constants.EthereumSepoliaChainID, 11111, "")

	if tpl.VRFAdmin != "0xVRF" {
		t.Errorf("VRFAdmin: got %q, want %q", tpl.VRFAdmin, "0xVRF")
	}
	if tpl.Preset != constants.PresetFull {
		t.Errorf("Preset: got %q, want %q", tpl.Preset, constants.PresetFull)
	}
}

// TestInitDeployConfigTemplate_NativeCurrencyLabelBytes verifies that
// NativeCurrencyLabelBytes matches the fee token symbol.
func TestInitDeployConfigTemplate_NativeCurrencyLabelBytes(t *testing.T) {
	tests := []struct {
		feeToken string
		want     []byte // first N bytes of the label
	}{
		{constants.FeeTokenTON, []byte("TON")},
		{constants.FeeTokenETH, []byte("ETH")},
		{constants.FeeTokenUSDT, []byte("USDT")},
		{constants.FeeTokenUSDC, []byte("USDC")},
	}

	for _, tc := range tests {
		t.Run(tc.feeToken, func(t *testing.T) {
			input := makeTestInput(constants.PresetDeFi, tc.feeToken)
			tpl := initDeployConfigTemplate(input, constants.EthereumSepoliaChainID, 1, "")

			if len(tpl.NativeCurrencyLabelBytes) != 32 {
				t.Fatalf("NativeCurrencyLabelBytes length: got %d, want 32", len(tpl.NativeCurrencyLabelBytes))
			}
			for i, b := range tc.want {
				if tpl.NativeCurrencyLabelBytes[i] != uint64(b) {
					t.Errorf("byte[%d]: got %d, want %d (%c)", i, tpl.NativeCurrencyLabelBytes[i], b, b)
				}
			}
			// Remaining bytes should be zero
			for i := len(tc.want); i < 32; i++ {
				if tpl.NativeCurrencyLabelBytes[i] != 0 {
					t.Errorf("byte[%d] should be 0, got %d", i, tpl.NativeCurrencyLabelBytes[i])
				}
			}
		})
	}
}

// TestGetFeeTokenConfig verifies the fee token config lookup for Sepolia.
func TestGetFeeTokenConfig(t *testing.T) {
	chainID := constants.EthereumSepoliaChainID

	ton := constants.GetFeeTokenConfig(constants.FeeTokenTON, chainID)
	if ton.Symbol != "TON" {
		t.Errorf("TON symbol: got %q", ton.Symbol)
	}
	if ton.L1Address == "" {
		t.Error("TON L1 address should not be empty for Sepolia")
	}

	eth := constants.GetFeeTokenConfig(constants.FeeTokenETH, chainID)
	if eth.L1Address != "0x0000000000000000000000000000000000000000" {
		t.Errorf("ETH L1 address: got %q", eth.L1Address)
	}

	usdt := constants.GetFeeTokenConfig(constants.FeeTokenUSDT, chainID)
	if usdt.L1Address == "" {
		t.Error("USDT L1 address should not be empty for Sepolia")
	}

	usdc := constants.GetFeeTokenConfig(constants.FeeTokenUSDC, chainID)
	if usdc.L1Address == "" {
		t.Error("USDC L1 address should not be empty for Sepolia")
	}
}
