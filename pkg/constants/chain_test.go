package constants

import "testing"

// TestPresetModulesCrossTrade verifies crossTrade module alignment across presets.
// TDD RED: DeFi_has_crossTrade_true and Gaming_has_no_crossTrade fail with current code.
// TDD GREEN: All subtests pass after Plan 02-02 alignment changes.
func TestPresetModulesCrossTrade(t *testing.T) {
	t.Run("DeFi_has_crossTrade_true", func(t *testing.T) {
		val, ok := PresetModules[PresetDeFi]["crossTrade"]
		if !ok {
			t.Errorf("PresetModules[%q] should have crossTrade key", PresetDeFi)
		}
		if !val {
			t.Errorf("PresetModules[%q][crossTrade] = %v, want true", PresetDeFi, val)
		}
	})

	t.Run("Gaming_has_no_crossTrade", func(t *testing.T) {
		_, ok := PresetModules[PresetGaming]["crossTrade"]
		if ok {
			t.Errorf("PresetModules[%q] should not have crossTrade key", PresetGaming)
		}
	})

	t.Run("Full_has_crossTrade_true", func(t *testing.T) {
		val, ok := PresetModules[PresetFull]["crossTrade"]
		if !ok {
			t.Errorf("PresetModules[%q] should have crossTrade key", PresetFull)
		}
		if !val {
			t.Errorf("PresetModules[%q][crossTrade] = %v, want true", PresetFull, val)
		}
	})

	t.Run("General_has_no_crossTrade", func(t *testing.T) {
		_, ok := PresetModules[PresetGeneral]["crossTrade"]
		if ok {
			t.Errorf("PresetModules[%q] should not have crossTrade key", PresetGeneral)
		}
	})
}
