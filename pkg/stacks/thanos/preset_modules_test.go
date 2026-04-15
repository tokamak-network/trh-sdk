package thanos

import (
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// TestPresetModulesMatrix verifies that PresetModules contains the correct
// module flags for each preset (from PRD 2 / ADR ① enable/disable matrix).
func TestPresetModulesMatrix(t *testing.T) {
	type moduleExpect struct {
		module  string
		enabled bool
	}

	tests := []struct {
		preset  string
		expects []moduleExpect
	}{
		{
			preset: constants.PresetGeneral,
			expects: []moduleExpect{
				{"bridge", true},
				{"blockExplorer", true},
				{"monitoring", false},
				{"uptimeService", false},
				{"crossTrade", false},
				{"drb", false},
				{"aaPaymaster", false},
			},
		},
		{
			preset: constants.PresetDeFi,
			expects: []moduleExpect{
				{"bridge", true},
				{"blockExplorer", true},
				{"monitoring", true},
				{"uptimeService", true},
				{"crossTrade", true},
				{"drb", false},
				{"aaPaymaster", false},
			},
		},
		{
			preset: constants.PresetGaming,
			expects: []moduleExpect{
				{"bridge", true},
				{"blockExplorer", true},
				{"monitoring", true},
				{"uptimeService", true},
				{"crossTrade", false},
				{"drb", true},
				{"aaPaymaster", true},
			},
		},
		{
			preset: constants.PresetFull,
			expects: []moduleExpect{
				{"bridge", true},
				{"blockExplorer", true},
				{"monitoring", true},
				{"uptimeService", true},
				{"crossTrade", true},
				{"drb", true},
				{"aaPaymaster", true},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.preset, func(t *testing.T) {
			modules, ok := constants.PresetModules[tc.preset]
			if !ok {
				t.Fatalf("PresetModules[%q] not found", tc.preset)
			}
			for _, e := range tc.expects {
				got := modules[e.module]
				if got != e.enabled {
					t.Errorf("preset=%s module=%s: got enabled=%v, want %v",
						tc.preset, e.module, got, e.enabled)
				}
			}
		})
	}
}

// TestPresetModules_NoStakingV2OrBackup verifies that out-of-scope modules
// (stakingV2, backup) are not present in any preset definition.
func TestPresetModules_NoStakingV2OrBackup(t *testing.T) {
	forbidden := []string{"stakingV2", "backup"}
	for preset, modules := range constants.PresetModules {
		for _, key := range forbidden {
			if modules[key] {
				t.Errorf("preset=%s contains forbidden module=%s (out of scope per PRD)", preset, key)
			}
		}
	}
}
