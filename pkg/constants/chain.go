package constants

import "fmt"

var L1ChainConfigurations = map[uint64]struct {
	NativeToken                      string `json:"native_token"`
	FinalizationPeriodSeconds        uint64 `json:"finalization_period_seconds"`
	L2OutputOracleSubmissionInterval uint64 `json:"l2_output_oracle_submission_interval"`
	USDCAddress                      string `json:"usdc_address"`
}{
	1: {
		NativeToken:                      "0x2be5e8c109e2197D077D13A82dAead6a9b3433C5",
		FinalizationPeriodSeconds:        604800,
		L2OutputOracleSubmissionInterval: 10800,
		USDCAddress:                      "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
	},
	11155111: {
		NativeToken:                      "0xa30fe40285b8f5c0457dbc3b7c8a280373c40044",
		FinalizationPeriodSeconds:        12,
		L2OutputOracleSubmissionInterval: 120,
		USDCAddress:                      "0xd718826bbc28e61dc93aacae04711c8e755b4915",
	},
	17000: {
		NativeToken:                      "0xe11Ad6B761D175042340a784640d3A6e373E52A5",
		FinalizationPeriodSeconds:        12,
		L2OutputOracleSubmissionInterval: 120,
		USDCAddress:                      "0xd718826bbc28e61dc93aacae04711c8e755b4915",
	},
}

const L2ChainId = 111551119876

var basebatchInboxAddress = "0xff00000000000000000000000000000000000000"
var BatchInboxAddress = fmt.Sprintf("%s%d", basebatchInboxAddress[:len(basebatchInboxAddress)-len(fmt.Sprintf("%d", L2ChainId))], L2ChainId)
