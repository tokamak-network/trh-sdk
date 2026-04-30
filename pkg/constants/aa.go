package constants

// AAPresetsWithPaymaster lists presets that include AA predeploy contracts (all presets).
var AAPresetsWithPaymaster = map[string]bool{
	PresetGeneral: true,
	PresetDeFi:    true,
	PresetGaming:  true,
	PresetFull:    true,
}

// IsAAPreset returns true if the given preset includes AA infrastructure.
func IsAAPreset(preset string) bool {
	return AAPresetsWithPaymaster[preset]
}

// NeedsAASetup returns true when AA paymaster configuration is required:
// All presets include AA contracts; setup is needed when fee token is not TON (the L2 native token).
func NeedsAASetup(preset, feeToken string) bool {
	return IsAAPreset(preset) && feeToken != FeeTokenTON
}
