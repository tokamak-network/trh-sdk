package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"
const PluginDRB = "drb-vrf"
const PluginUptimeService = "uptime-service"


var SupportedPlugins = map[string]bool{
	PluginBridge:        true,
	PluginBlockExplorer: true,
	PluginMonitoring:    true,
	PluginCrossTrade:    true,
	PluginDRB:           true,
	PluginUptimeService: true,
}

var SupportedPluginsList = []string{
	PluginBridge,
	PluginBlockExplorer,
	PluginMonitoring,
	PluginCrossTrade,
	PluginDRB,
	PluginUptimeService,
}

// PluginsThatWorkWithoutChain lists plugins that can be installed without a deployed chain.
var PluginsThatWorkWithoutChain = map[string]bool{}

// CanPluginWorkWithoutChain returns true if the plugin can be installed without an existing chain.
func CanPluginWorkWithoutChain(pluginName string) bool {
	return PluginsThatWorkWithoutChain[pluginName]
}
