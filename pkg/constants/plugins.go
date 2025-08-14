package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"

var SupportedPlugins = map[string]bool{
	PluginBridge:        true,
	PluginBlockExplorer: true,
	PluginMonitoring:    true,
	PluginCrossTrade:    true,
}
