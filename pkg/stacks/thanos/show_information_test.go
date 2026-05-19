package thanos

import (
	"testing"
)

// TestBuildIngressURLs_BlockExplorerWithoutRunningPod verifies that ShowInformation
// returns a block-explorer URL when the ingress has an ELB address but the pod is not
// yet Running. InstallBlockExplorer returns as soon as K8s ingress gets an ELB hostname,
// which can happen before the pod transitions to Running. Requiring both pod AND ingress
// caused the auto-mark to be skipped, leaving the integration stuck in AwaitingConfig.
func TestBuildIngressURLs_BlockExplorerWithoutRunningPod(t *testing.T) {
	podStatuses := map[string]bool{
		"chain":             true,
		"bridge":            false,
		"block-explorer-be": false, // pod NOT yet Running
	}
	ingresses := map[string][]string{
		"mychain-12345":              {"l2.example.com"},
		"block-explorer-be-1748400000": {"blockscout.example.com"},
	}
	namespace := "mychain-12345"

	_, _, blockExplorerUrl := buildIngressURLs(podStatuses, ingresses, namespace)

	if blockExplorerUrl != "http://blockscout.example.com" {
		t.Errorf("expected block explorer URL when ingress present but pod not running; got %q", blockExplorerUrl)
	}
}

// TestBuildIngressURLs_AllServicesRunning verifies the happy path:
// when all pods are Running and ingresses have addresses, all URLs are populated.
func TestBuildIngressURLs_AllServicesRunning(t *testing.T) {
	podStatuses := map[string]bool{
		"chain":             true,
		"bridge":            true,
		"block-explorer-be": true,
	}
	ingresses := map[string][]string{
		"mychain-12345":              {"l2.example.com"},
		"bridge-12345":               {"bridge.example.com"},
		"block-explorer-be-1748400000": {"blockscout.example.com"},
	}
	namespace := "mychain-12345"

	l2Url, bridgeUrl, blockExplorerUrl := buildIngressURLs(podStatuses, ingresses, namespace)

	if l2Url != "http://l2.example.com" {
		t.Errorf("l2Url: got %q, want %q", l2Url, "http://l2.example.com")
	}
	if bridgeUrl != "http://bridge.example.com" {
		t.Errorf("bridgeUrl: got %q, want %q", bridgeUrl, "http://bridge.example.com")
	}
	if blockExplorerUrl != "http://blockscout.example.com" {
		t.Errorf("blockExplorerUrl: got %q, want %q", blockExplorerUrl, "http://blockscout.example.com")
	}
}

// TestBuildIngressURLs_EmptyIngress verifies that empty-address ingresses are skipped.
func TestBuildIngressURLs_EmptyIngress(t *testing.T) {
	podStatuses := map[string]bool{
		"chain":             true,
		"bridge":            false,
		"block-explorer-be": false,
	}
	ingresses := map[string][]string{
		"block-explorer-be-1748400000": {}, // no address yet
	}
	namespace := "mychain-12345"

	_, _, blockExplorerUrl := buildIngressURLs(podStatuses, ingresses, namespace)

	if blockExplorerUrl != "" {
		t.Errorf("expected empty URL when ingress has no address; got %q", blockExplorerUrl)
	}
}
