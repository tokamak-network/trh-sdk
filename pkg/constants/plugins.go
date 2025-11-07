package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"
const PluginUptimeService = "uptime-service"

var SupportedPlugins = map[string]bool{
	PluginBridge:        true,
	PluginBlockExplorer: true,
	PluginMonitoring:    true,
	PluginCrossTrade:    true,
	PluginUptimeService: true,
}

var SupportedPluginsList = []string{
	PluginBridge,
	PluginBlockExplorer,
	PluginMonitoring,
	PluginCrossTrade,
	PluginUptimeService,
}
