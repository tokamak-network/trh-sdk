package types

// SafeWalletInfo contains Safe wallet configuration details
type SafeWalletInfo struct {
	Address   string   `json:"address"`
	Owners    []string `json:"owners"`
	Threshold uint64   `json:"threshold"`
}

// CandidateRegistrationInfo contains DAO candidate registration details
type CandidateRegistrationInfo struct {
	StakingAmount       float64 `json:"staking_amount"`
	RollupConfigAddress string  `json:"rollup_config_address"`
	CandidateName       string  `json:"candidate_name"`
	CandidateMemo       string  `json:"candidate_memo"`
}

// RegistrationAdditionalInfo combines all registration-related information
type RegistrationAdditionalInfo struct {
	SafeWallet            *SafeWalletInfo            `json:"safe_wallet,omitempty"`
	CandidateRegistration *CandidateRegistrationInfo `json:"candidate_registration,omitempty"`
}
