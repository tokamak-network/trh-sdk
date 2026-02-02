package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"
const PluginDRB = "drb"
const PluginUptimeService = "uptime-service"

// DRB type values for --type flag (trh-sdk install drb --type leader|regular)
const DRBTypeLeader  = "leader"
const DRBTypeRegular = "regular"

var SupportedPlugins = map[string]bool{
	PluginBridge:         true,
	PluginBlockExplorer:  true,
	PluginMonitoring:     true,
	PluginCrossTrade:     true,
	PluginDRB:            true,
	PluginUptimeService:  true,
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
// Add plugins here when they can run independently (e.g. DRB leader/regular on standalone infra).
var PluginsThatWorkWithoutChain = map[string]bool{
	PluginDRB: true,
}

// CanPluginWorkWithoutChain returns true if the plugin can be installed without an existing chain.
func CanPluginWorkWithoutChain(pluginName string) bool {
	return PluginsThatWorkWithoutChain[pluginName]
}
