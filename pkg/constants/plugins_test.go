package constants_test

import (
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// TestPluginDRB_Registered verifies that PluginDRB is registered in
// SupportedPlugins and SupportedPluginsList (added in PRD 2).
func TestPluginDRB_Registered(t *testing.T) {
	if !constants.SupportedPlugins[constants.PluginDRB] {
		t.Errorf("SupportedPlugins[%q] is false, want true", constants.PluginDRB)
	}

	found := false
	for _, p := range constants.SupportedPluginsList {
		if p == constants.PluginDRB {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("PluginDRB (%q) not found in SupportedPluginsList", constants.PluginDRB)
	}
}

// TestPluginDRB_Value verifies the plugin name matches the Helm release name.
func TestPluginDRB_Value(t *testing.T) {
	if constants.PluginDRB != "drb-vrf" {
		t.Errorf("PluginDRB = %q, want %q", constants.PluginDRB, "drb-vrf")
	}
}

// TestSupportedPlugins_AllListedInMap verifies that SupportedPluginsList and
// SupportedPlugins map are consistent (no orphan entries).
func TestSupportedPlugins_AllListedInMap(t *testing.T) {
	for _, p := range constants.SupportedPluginsList {
		if !constants.SupportedPlugins[p] {
			t.Errorf("plugin %q is in SupportedPluginsList but not in SupportedPlugins map", p)
		}
	}
	if len(constants.SupportedPluginsList) != len(constants.SupportedPlugins) {
		t.Errorf("SupportedPluginsList length %d != SupportedPlugins map size %d — lists are out of sync",
			len(constants.SupportedPluginsList), len(constants.SupportedPlugins))
	}
}
