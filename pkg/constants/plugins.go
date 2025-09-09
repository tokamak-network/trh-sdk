package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"
const PluginDRB = "drb"

var SupportedPlugins = map[string]bool{
	PluginBridge:        true,
	PluginBlockExplorer: true,
	PluginMonitoring:    true,
	PluginCrossTrade:    true,
	PluginDRB:           true,
}

var SupportedPluginsList = []string{
	PluginBridge,
	PluginBlockExplorer,
	PluginMonitoring,
	PluginCrossTrade,
	PluginDRB,
}
