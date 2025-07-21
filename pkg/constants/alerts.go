package constants

// Alert names for monitoring system
const (
	// OP Stack Component Alerts
	AlertOpNodeDown     = "OpNodeDown"
	AlertOpBatcherDown  = "OpBatcherDown"
	AlertOpProposerDown = "OpProposerDown"
	AlertOpGethDown     = "OpGethDown"
	AlertL1RpcDown      = "L1RpcDown"

	// Balance Alerts
	AlertOpBatcherBalanceCritical  = "OpBatcherBalanceCritical"
	AlertOpProposerBalanceCritical = "OpProposerBalanceCritical"

	// System Alerts
	AlertBlockProductionStalled   = "BlockProductionStalled"
	AlertContainerCpuUsageHigh    = "ContainerCpuUsageHigh"
	AlertContainerMemoryUsageHigh = "ContainerMemoryUsageHigh"
	AlertPodCrashLooping          = "PodCrashLooping"
)

// Alert descriptions for user-friendly display
var AlertDescriptions = map[string]string{
	AlertOpNodeDown:                "OP Node is down",
	AlertOpBatcherDown:             "OP Batcher is down",
	AlertOpProposerDown:            "OP Proposer is down",
	AlertOpGethDown:                "OP Geth is down",
	AlertL1RpcDown:                 "L1 RPC connection failed",
	AlertOpBatcherBalanceCritical:  "OP Batcher ETH balance critically low",
	AlertOpProposerBalanceCritical: "OP Proposer ETH balance critically low",
	AlertBlockProductionStalled:    "Block production has stalled",
	AlertContainerCpuUsageHigh:     "High CPU usage in Thanos Stack pod",
	AlertContainerMemoryUsageHigh:  "High memory usage in Thanos Stack pod",
	AlertPodCrashLooping:           "Pod is crash looping",
}

// Configurable alerts that can be customized by users
var ConfigurableAlerts = map[string]string{
	AlertOpBatcherBalanceCritical:  "OP Batcher balance threshold",
	AlertOpProposerBalanceCritical: "OP Proposer balance threshold",
	AlertBlockProductionStalled:    "Block production stall detection",
	AlertContainerCpuUsageHigh:     "Container CPU usage threshold",
	AlertContainerMemoryUsageHigh:  "Container memory usage threshold",
	AlertPodCrashLooping:           "Pod crash loop detection",
}
