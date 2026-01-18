package constants

const PluginBridge = "bridge"
const PluginBlockExplorer = "block-explorer"
const PluginMonitoring = "monitoring"
const PluginCrossTrade = "cross-trade"
const PluginUptimeService = "uptime-service"
const PluginMonitoringThanosLogs = "monitoring-thanos-logs"

var SupportedPlugins = map[string]bool{
	PluginBridge:               true,
	PluginBlockExplorer:        true,
	PluginMonitoring:           true,
	PluginCrossTrade:           true,
	PluginUptimeService:        true,
	PluginMonitoringThanosLogs: true,
}

var SupportedPluginsList = []string{
	PluginBridge,
	PluginBlockExplorer,
	PluginMonitoring,
	PluginCrossTrade,
	PluginUptimeService,
	PluginMonitoringThanosLogs,
}
