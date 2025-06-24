package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
	"gopkg.in/yaml.v3"
)

// InstallMonitoring installs monitoring stack using Helm dependencies
func (t *ThanosStack) installMonitoring(ctx context.Context) error {
	fmt.Println("🚀 Starting monitoring installation...")

	// Get monitoring configuration
	config, err := t.getMonitoringConfig(ctx)
	if err != nil {
		return fmt.Errorf("failed to get monitoring configuration: %w", err)
	}

	// Deploy Terraform infrastructure if persistent storage is enabled
	if config.EnablePersistence {
		fmt.Println("📦 Deploying persistent storage infrastructure...")
		if err := t.deployMonitoringInfrastructure(ctx, config); err != nil {
			return fmt.Errorf("failed to deploy monitoring infrastructure: %w", err)
		}
	}

	// Generate values file
	if err := t.generateValuesFile(config); err != nil {
		return fmt.Errorf("failed to generate values file: %w", err)
	}

	// Update chart dependencies
	fmt.Println("📦 Updating chart dependencies...")
	if _, err := utils.ExecuteCommand("helm", "dependency", "update", config.ChartsPath); err != nil {
		return fmt.Errorf("failed to update chart dependencies: %w", err)
	}

	// Install monitoring stack with error monitoring
	fmt.Printf("⚙️  Installing monitoring stack '%s' in namespace '%s'...\n", config.HelmReleaseName, config.Namespace)
	installCmd := []string{
		"upgrade", "--install",
		config.HelmReleaseName,
		config.ChartsPath,
		"--values", config.ValuesFilePath,
		"--namespace", config.Namespace,
		"--create-namespace",
		"--timeout", "10m",
		"--wait",
	}

	// Start error monitoring in background
	errorChan := make(chan error, 1)
	go t.monitorInstallationErrors(config, errorChan)

	if _, err := utils.ExecuteCommand("helm", installCmd...); err != nil {
		// Installation failed, gather error information
		fmt.Println("\n❌ Installation failed! Gathering error information...")
		t.gatherInstallationErrors(config)
		return fmt.Errorf("failed to install monitoring stack: %w", err)
	}

	// Stop error monitoring
	close(errorChan)

	// Create dashboard ConfigMaps after successful Helm installation
	if err := t.createDashboardConfigMaps(config); err != nil {
		fmt.Printf("⚠️  Failed to create dashboard ConfigMaps: %v\n", err)
		fmt.Println("   Dashboards can be imported manually later")
	}

	// Display access information
	t.displayMonitoringInfo(config)

	return nil
}

// displayMonitoringInfo shows access information for the monitoring stack
func (t *ThanosStack) displayMonitoringInfo(config *MonitoringConfig) {
	fmt.Println("\n🎉 Monitoring Stack Installation Complete!")
	fmt.Println("==========================================")

	fmt.Printf("📊 **Grafana Dashboard Access:**\n")
	fmt.Printf("   • Username: admin\n")
	fmt.Printf("   • Password: %s\n", config.AdminPassword)
	fmt.Printf("   • Namespace: %s\n", config.Namespace)
	fmt.Printf("   • Release: %s\n\n", config.HelmReleaseName)

	// Wait for ALB ingress endpoint to be ready
	fmt.Println("🔗 **ALB Ingress Endpoint:**")
	grafanaURL := t.waitForIngressEndpoint(config)

	if grafanaURL != "" {
		fmt.Printf("   🌐 **Grafana Web URL: %s**\n", grafanaURL)
		fmt.Printf("   🎯 You can now access Grafana directly via the web!\n\n")
	} else {
		fmt.Printf("   ⚠️  ALB Ingress endpoint not ready within timeout\n")
		fmt.Printf("   🔧 Check status: kubectl get ingress -n %s -w\n\n", config.Namespace)
	}

	fmt.Printf("🔗 **Local Access Commands (Alternative):**\n")
	fmt.Printf("   # Port forward to access Grafana locally:\n")
	fmt.Printf("   kubectl port-forward -n %s svc/%s-grafana 3000:80\n", config.Namespace, config.HelmReleaseName)
	fmt.Printf("   # Then visit: http://localhost:3000\n\n")
}

// MonitoringConfig holds all configuration needed for monitoring installation
type MonitoringConfig struct {
	Namespace             string
	HelmReleaseName       string
	AdminPassword         string
	L1RpcUrl              string
	ServiceNames          map[string]string
	EnablePersistence     bool
	UseStaticProvisioning bool // Use static provisioning with pre-created PV/PVC
	ChartsPath            string
	ValuesFilePath        string
	ChainName             string
}

// getMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) getMonitoringConfig(ctx context.Context) (*MonitoringConfig, error) {
	config := &MonitoringConfig{
		Namespace: "monitoring",
		L1RpcUrl:  t.deployConfig.L1RPCURL,
		ChainName: t.deployConfig.ChainName,
	}

	// Use timestamped release name for monitoring
	timestamp := time.Now().Unix()
	config.HelmReleaseName = fmt.Sprintf("monitoring-%d", timestamp)

	// Get admin password from user
	fmt.Print("🔐 Enter Grafana admin password: ")
	adminPassword, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("error reading admin password: %w", err)
	}
	if adminPassword == "" {
		return nil, fmt.Errorf("admin password cannot be empty")
	}
	config.AdminPassword = adminPassword

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("error determining current directory: %w", err)
	}

	// Set charts path
	config.ChartsPath = fmt.Sprintf("%s/tokamak-thanos-stack/charts/monitoring", cwd)
	if _, err := os.Stat(config.ChartsPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("chart directory not found: %s", config.ChartsPath)
	}

	// Get service names dynamically from trh-sdk configuration
	// Note: Services are in the original Thanos Stack namespace
	serviceNames, err := t.getServiceNames(t.deployConfig.K8s.Namespace, config.ChainName)
	if err != nil {
		return nil, fmt.Errorf("error getting service names: %w", err)
	}
	config.ServiceNames = serviceNames

	// Ask user about persistent storage preference first
	fmt.Print("🤔 Do you want to use persistent storage for monitoring data? (y/n): ")
	usePersistence, err := scanner.ScanString()
	if err != nil {
		return nil, fmt.Errorf("error reading storage preference: %w", err)
	}

	if strings.ToLower(usePersistence) == "y" || strings.ToLower(usePersistence) == "yes" {
		config.EnablePersistence = true
		config.UseStaticProvisioning = true
	} else {
		config.EnablePersistence = false
		config.UseStaticProvisioning = false
	}

	return config, nil
}

// detectFargateEnvironment checks if the cluster is running on AWS Fargate
func (t *ThanosStack) detectFargateEnvironment() bool {
	// Check for Fargate nodes by looking for fargate in node names or labels
	output, err := utils.ExecuteCommand("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		// If we can't get nodes, assume it might be Fargate for safety
		fmt.Printf("⚠️  Could not check node information: %s\n", err)
		return false
	}

	// Check if any node name contains "fargate"
	if strings.Contains(strings.ToLower(output), "fargate") {
		return true
	}

	// Check node labels for Fargate indicators
	labelOutput, err := utils.ExecuteCommand("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].metadata.labels}")
	if err == nil && strings.Contains(strings.ToLower(labelOutput), "fargate") {
		return true
	}

	// Check for EKS Fargate profile indicators
	profileOutput, err := utils.ExecuteCommand("kubectl", "get", "nodes", "-o", "jsonpath={.items[*].metadata.labels.eks\\.amazonaws\\.com/compute-type}")
	if err == nil && strings.Contains(strings.ToLower(profileOutput), "fargate") {
		return true
	}

	return false
}

// generateValuesFile creates the values.yaml file for monitoring configuration
func (t *ThanosStack) generateValuesFile(config *MonitoringConfig) error {
	fmt.Println("📝 Generating monitoring values file...")

	// Create values configuration matching the chart's values.yaml structure exactly
	valuesConfig := map[string]interface{}{
		"createNamespace": false, // Handled by Helm --create-namespace flag
		"global": map[string]interface{}{
			"l1RpcUrl": config.L1RpcUrl,
			"storage": map[string]interface{}{
				"enabled":               config.EnablePersistence,
				"useStaticProvisioning": config.UseStaticProvisioning,
				"awsRegion":             t.deployConfig.AWS.Region,
			},
			"securityContext": map[string]interface{}{
				"runAsNonRoot":             true,
				"runAsUser":                65534,
				"runAsGroup":               65534,
				"readOnlyRootFilesystem":   false,
				"allowPrivilegeEscalation": false,
			},
			"podSecurityContext": map[string]interface{}{
				"runAsNonRoot": true,
				"runAsUser":    65534,
				"runAsGroup":   65534,
				"fsGroup":      65534,
			},
		},
		"thanosStack": map[string]interface{}{
			"chainName":   config.ChainName,
			"namespace":   t.deployConfig.K8s.Namespace, // Use the Thanos Stack namespace
			"releaseName": config.ChainName,
		},
		"kube-prometheus-stack": map[string]interface{}{
			"prometheus": map[string]interface{}{
				"prometheusSpec": t.generatePrometheusSpec(config),
			},
			"grafana": t.generateGrafanaConfig(config),
			"alertmanager": map[string]interface{}{
				"enabled": false,
			},
			"nodeExporter": map[string]interface{}{
				"enabled": false,
			},
			"kubeStateMetrics": map[string]interface{}{
				"enabled": true,
			},
		},
		"prometheus-blackbox-exporter": map[string]interface{}{
			"enabled": true,
			"config": map[string]interface{}{
				"modules": map[string]interface{}{
					"http_post_eth_node_synced_2xx": map[string]interface{}{
						"prober":  "http",
						"timeout": "10s",
						"http": map[string]interface{}{
							"method": "POST",
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
							},
							"body":                            `{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`,
							"valid_status_codes":              []int{200},
							"preferred_ip_protocol":           "ip4",
							"fail_if_body_not_matches_regexp": []string{`"result"\s*:\s*false`},
						},
					},
					"http_post_eth_block_number_2xx": map[string]interface{}{
						"prober":  "http",
						"timeout": "10s",
						"http": map[string]interface{}{
							"method": "POST",
							"headers": map[string]interface{}{
								"Content-Type": "application/json",
							},
							"body":                            `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":83}`,
							"valid_status_codes":              []int{200},
							"preferred_ip_protocol":           "ip4",
							"fail_if_body_not_matches_regexp": []string{`"result"\s*:\s*"0x[0-9a-fA-F]+"`},
						},
					},
					"tcp_connect": map[string]interface{}{
						"prober":  "tcp",
						"timeout": "5s",
					},
				},
			},
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"cpu":    "100m",
					"memory": "128Mi",
				},
				"limits": map[string]interface{}{
					"cpu":    "500m",
					"memory": "256Mi",
				},
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"port": 9115,
			},
			"serviceMonitor": map[string]interface{}{
				"enabled": true,
				"defaults": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": "blackbox-exporter",
					},
					"interval":      "30s",
					"scrapeTimeout": "30s",
				},
			},
		},
		"scrapeTargets":           t.generateScrapeTargets(config),
		"additionalScrapeConfigs": []interface{}{}, // Use Helm template-based configuration instead
	}

	// Convert to YAML
	yamlData, err := yaml.Marshal(valuesConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal values to YAML: %w", err)
	}

	// Set values file path to terraform/thanos-stack directory
	terraformDir := fmt.Sprintf("%s/../../tokamak-thanos-stack/terraform/thanos-stack",
		filepath.Dir(config.ChartsPath))

	// Create directory if it doesn't exist
	if err := os.MkdirAll(terraformDir, 0755); err != nil {
		return fmt.Errorf("failed to create terraform directory: %w", err)
	}

	config.ValuesFilePath = filepath.Join(terraformDir, "monitoring-values.yaml")

	// Write to file
	if err := os.WriteFile(config.ValuesFilePath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write values file: %w", err)
	}

	fmt.Printf("✅ Generated values file: %s\n", config.ValuesFilePath)
	return nil
}

// generatePrometheusSpec creates Prometheus specification with proper storage configuration
func (t *ThanosStack) generatePrometheusSpec(config *MonitoringConfig) map[string]interface{} {
	prometheusSpec := map[string]interface{}{
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "1500m",
				"memory": "3Gi",
			},
		},
		"retention":               "1y",
		"retentionSize":           "10GB",
		"scrapeInterval":          "1m",
		"evaluationInterval":      "1m",
		"additionalScrapeConfigs": []interface{}{}, // Configured via Helm template
		"securityContext": map[string]interface{}{
			"runAsNonRoot":             true,
			"runAsUser":                65534,
			"runAsGroup":               65534,
			"readOnlyRootFilesystem":   false,
			"allowPrivilegeEscalation": false,
		},
		"podSecurityContext": map[string]interface{}{
			"runAsNonRoot": true,
			"runAsUser":    65534,
			"runAsGroup":   65534,
			"fsGroup":      65534,
		},
	}

	// Use EmptyDir for temporary storage (Static Provisioning handles persistent storage)
	prometheusSpec["storageSpec"] = map[string]interface{}{}

	return prometheusSpec
}

// generateGrafanaConfig creates Grafana configuration with proper service naming and storage
func (t *ThanosStack) generateGrafanaConfig(config *MonitoringConfig) map[string]interface{} {
	grafanaConfig := map[string]interface{}{
		"enabled":       true,
		"adminUser":     "admin",
		"adminPassword": config.AdminPassword,
		"resources": map[string]interface{}{
			"requests": map[string]interface{}{
				"cpu":    "1500m",
				"memory": "4Gi",
			},
		},
		// Use dynamic service naming (removes fullnameOverride for timestamp compatibility)
		// "fullnameOverride": removed to allow timestamped release names,
		"ingress": map[string]interface{}{
			"enabled":          true,
			"ingressClassName": "alb",
			"annotations": map[string]interface{}{
				"alb.ingress.kubernetes.io/scheme":       "internet-facing",
				"alb.ingress.kubernetes.io/target-type":  "ip",
				"alb.ingress.kubernetes.io/group.name":   "thanos-monitoring",
				"alb.ingress.kubernetes.io/listen-ports": `[{"HTTP":80}]`,
			},
			"hosts": []string{}, // Empty hosts array for ALB auto-discovery
			"path":  "/",        // Single path for Grafana ingress
		},
		"defaultDashboardsEnabled":  false,
		"defaultDashboardsTimezone": "utc",
		"sidecar": map[string]interface{}{
			"dashboards": map[string]interface{}{
				"enabled":         true,
				"label":           "grafana_dashboard",
				"labelValue":      "1",
				"searchNamespace": "ALL",
			},
			"datasources": map[string]interface{}{
				"enabled":                  true,
				"defaultDatasourceEnabled": true,
			},
		},
		"securityContext": map[string]interface{}{
			"runAsNonRoot":             true,
			"runAsUser":                65534,
			"runAsGroup":               65534,
			"readOnlyRootFilesystem":   false,
			"allowPrivilegeEscalation": false,
		},
		"podSecurityContext": map[string]interface{}{
			"runAsNonRoot": true,
			"runAsUser":    65534,
			"runAsGroup":   65534,
			"fsGroup":      65534,
		},
	}

	// Use EmptyDir for temporary storage (Static Provisioning handles persistent storage)
	grafanaConfig["persistence"] = map[string]interface{}{
		"enabled": false,
	}

	return grafanaConfig
}

// generateScrapeTargets creates scrape target configuration for dynamic L2 Stack services
// This provides the default configuration which can be overridden in values.yaml
func (t *ThanosStack) generateScrapeTargets(config *MonitoringConfig) map[string]interface{} {
	// Use default values from chart, allowing for runtime customization
	return map[string]interface{}{
		"op-node": map[string]interface{}{
			"enabled":  true,
			"port":     7300,
			"path":     "/metrics",
			"interval": "30s",
		},
		"op-batcher": map[string]interface{}{
			"enabled":  true,
			"port":     7300,
			"path":     "/metrics",
			"interval": "30s",
		},
		"op-proposer": map[string]interface{}{
			"enabled":  true,
			"port":     7300,
			"path":     "/metrics",
			"interval": "30s",
		},
		"op-geth": map[string]interface{}{
			"enabled":  true,
			"port":     6060,
			"path":     "/debug/metrics/prometheus",
			"interval": "30s",
		},
		"blockscout": map[string]interface{}{
			"enabled":  true,
			"port":     3000,
			"path":     "/metrics",
			"interval": "1m",
		},
		"block-explorer-frontend": map[string]interface{}{
			"enabled":  true,
			"port":     80,
			"path":     "/api/healthz",
			"interval": "1m",
		},
	}
}

// getServiceNames returns a map of component names to their Kubernetes service names
func (t *ThanosStack) getServiceNames(namespace, chainName string) (map[string]string, error) {
	fmt.Printf("🔍 Discovering services in namespace: %s\n", namespace)

	// First, get all services in the namespace
	output, err := utils.ExecuteCommand("kubectl", "get", "services", "-n", namespace, "-o", "custom-columns=NAME:.metadata.name,TYPE:.spec.type", "--no-headers")
	if err != nil {
		return nil, fmt.Errorf("failed to get services in namespace %s: %w", namespace, err)
	}

	if strings.TrimSpace(output) == "" {
		return nil, fmt.Errorf("no services found in namespace %s", namespace)
	}

	// Parse the service list
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var allServices []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 1 {
			serviceName := fields[0]
			allServices = append(allServices, serviceName)
		}
	}

	// Component mapping patterns for OP Stack services (dashboard-compatible naming)
	componentPatterns := map[string][]string{
		"op-node": {
			"op-node", "node", "opnode",
			fmt.Sprintf("%s-op-node", chainName),
			fmt.Sprintf("%s-node", chainName),
			fmt.Sprintf("%s-thanos-stack-op-node", chainName),
		},
		"op-batcher": {
			"op-batcher", "batcher", "opbatcher",
			fmt.Sprintf("%s-op-batcher", chainName),
			fmt.Sprintf("%s-batcher", chainName),
			fmt.Sprintf("%s-thanos-stack-op-batcher", chainName),
		},
		"op-proposer": {
			"op-proposer", "proposer", "opproposer",
			fmt.Sprintf("%s-op-proposer", chainName),
			fmt.Sprintf("%s-proposer", chainName),
			fmt.Sprintf("%s-thanos-stack-op-proposer", chainName),
		},
		"op-geth": {
			"op-geth", "geth", "opgeth", "l2geth",
			fmt.Sprintf("%s-op-geth", chainName),
			fmt.Sprintf("%s-geth", chainName),
			fmt.Sprintf("%s-thanos-stack-op-geth", chainName),
		},
		"blockscout": {
			"blockscout", "explorer", "block-explorer",
			fmt.Sprintf("%s-blockscout", chainName),
			fmt.Sprintf("%s-explorer", chainName),
			fmt.Sprintf("%s-thanos-stack-blockscout", chainName),
		},
		"block-explorer-frontend": {
			"block-explorer-frontend", "frontend", "explorer-frontend",
			fmt.Sprintf("%s-block-explorer-frontend", chainName),
			fmt.Sprintf("%s-frontend", chainName),
			fmt.Sprintf("%s-thanos-stack-block-explorer-frontend", chainName),
		},
	}

	serviceNames := make(map[string]string)

	// Try to match services to components
	for component, patterns := range componentPatterns {
		var foundService string

		// First try exact matches
		for _, pattern := range patterns {
			for _, service := range allServices {
				if service == pattern {
					foundService = service
					break
				}
			}
			if foundService != "" {
				break
			}
		}

		// If no exact match, try substring matching
		if foundService == "" {
			for _, pattern := range patterns {
				for _, service := range allServices {
					if strings.Contains(strings.ToLower(service), strings.ToLower(pattern)) ||
						strings.Contains(strings.ToLower(pattern), strings.ToLower(service)) {
						foundService = service
						break
					}
				}
				if foundService != "" {
					break
				}
			}
		}

		if foundService != "" {
			serviceNames[component] = foundService
		} else {
			// Try with timestamped release name pattern for monitoring compatibility
			timestampedName := fmt.Sprintf("%s-thanos-stack-%s", chainName, component)
			serviceNames[component] = timestampedName
		}
	}

	if len(serviceNames) == 0 {
		return nil, fmt.Errorf("no matching OP Stack services found in namespace %s", namespace)
	}

	return serviceNames, nil
}

func (t *ThanosStack) uninstallMonitoring(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	_, _, err := t.loginAWS(ctx)
	if err != nil {
		fmt.Println("Error to login in AWS:", err)
		return err
	}

	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	// Use the correct monitoring namespace instead of Thanos Stack namespace
	monitoringNamespace := "monitoring"

	// Find monitoring releases in the monitoring namespace
	releases, err := utils.FilterHelmReleases(monitoringNamespace, "monitoring")
	if err != nil {
		fmt.Println("Error to filter helm releases:", err)
		return err
	}

	for _, release := range releases {
		fmt.Printf("🗑️  Uninstalling monitoring release: %s\n", release)
		_, err = utils.ExecuteCommand("helm", []string{
			"uninstall",
			release,
			"--namespace",
			monitoringNamespace,
		}...)
		if err != nil {
			fmt.Println("Error uninstalling monitoring helm chart:", err)
			return err
		}
	}

	fmt.Println("✅ Uninstall monitoring component successfully")

	return nil
}

// getEFSFileSystemId retrieves EFS file system ID from AWS

// waitForIngressEndpoint waits for the ALB ingress endpoint to be ready
func (t *ThanosStack) waitForIngressEndpoint(config *MonitoringConfig) string {
	fmt.Println("⏳ Waiting for ALB Ingress endpoint to be provisioned...")
	fmt.Println("   (This may take 2-3 minutes for AWS ALB to be created)")

	maxRetries := 30
	retryInterval := 10 * time.Second

	for i := 0; i < maxRetries; i++ {
		// Check if ingress exists first
		ingressExists, err := utils.ExecuteCommand("kubectl", "get", "ingress", "-n", config.Namespace, "-o", "name")
		if err != nil || strings.TrimSpace(ingressExists) == "" {
			fmt.Printf("   ⏳ [%d/%d] Ingress not found yet, waiting...\n", i+1, maxRetries)
			time.Sleep(retryInterval)
			continue
		}

		// Get ingress hostname
		hostname, err := utils.ExecuteCommand("kubectl", "get", "ingress", "-n", config.Namespace,
			"-o", "jsonpath={.items[0].status.loadBalancer.ingress[0].hostname}")

		if err == nil && strings.TrimSpace(hostname) != "" {
			grafanaURL := fmt.Sprintf("http://%s", strings.TrimSpace(hostname))
			fmt.Printf("   ✅ ALB Ingress endpoint ready: %s\n", grafanaURL)

			return grafanaURL
		} else {
			fmt.Printf("   ⏳ [%d/%d] ALB provisioning in progress...\n", i+1, maxRetries)
		}

		time.Sleep(retryInterval)
	}

	fmt.Printf("   ⚠️  Timeout waiting for ALB Ingress endpoint (%d minutes)\n", (maxRetries*int(retryInterval.Seconds()))/60)
	fmt.Printf("   💡 ALB provisioning can take up to 5-10 minutes\n")
	fmt.Printf("   🔧 Check status manually: kubectl get ingress -n %s -w\n", config.Namespace)

	return ""
}

// monitorInstallationErrors monitors installation progress and reports errors in real-time
func (t *ThanosStack) monitorInstallationErrors(config *MonitoringConfig, errorChan chan error) {
	fmt.Println("🔍 Starting installation issue monitoring...")

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-errorChan:
			// Installation completed or stopped
			return
		case <-ticker.C:
			// Check for failed pods
			t.checkFailedPods(config)
			// Check for pending pods with issues
			t.checkPendingPods(config)
			// Check recent events for errors
			t.checkRecentEvents(config)
		}
	}
}

// gatherInstallationErrors gathers comprehensive error information when installation fails
func (t *ThanosStack) gatherInstallationErrors(config *MonitoringConfig) {
	fmt.Println("\n🔍 Gathering detailed error information...")
	fmt.Println("=" + strings.Repeat("=", 50))

	// 1. Check Helm release status
	fmt.Println("\n📊 Helm Release Status:")
	if output, err := utils.ExecuteCommand("helm", "status", config.HelmReleaseName, "-n", config.Namespace); err != nil {
		fmt.Printf("❌ Failed to get Helm status: %v\n", err)
	} else {
		fmt.Println(output)
	}

	// 2. Check failed pods with detailed information
	fmt.Println("\n🚨 Failed Pods Analysis:")
	t.analyzeFailedPods(config)

	// 3. Check recent events
	fmt.Println("\n📅 Recent Events (Last 10):")
	if output, err := utils.ExecuteCommand("kubectl", "get", "events", "-n", config.Namespace,
		"--sort-by=.lastTimestamp", "--field-selector=type=Warning"); err != nil {
		fmt.Printf("❌ Failed to get events: %v\n", err)
	} else {
		fmt.Println(output)
	}

	// 4. Check resource quotas and limits
	fmt.Println("\n💾 Resource Status:")
	t.checkResourceStatus(config)

	// 5. Check storage issues
	fmt.Println("\n🗄️  Storage Issues:")
	t.checkStorageIssues(config)

	// 6. Provide troubleshooting commands
	fmt.Println("\n🛠️  Troubleshooting Commands:")
	t.provideTroubleshootingCommands(config)
}

// checkFailedPods checks for failed pods and reports them
func (t *ThanosStack) checkFailedPods(config *MonitoringConfig) {
	output, err := utils.ExecuteCommand("kubectl", "get", "pods", "-n", config.Namespace,
		"--field-selector=status.phase=Failed", "-o", "custom-columns=NAME:.metadata.name,REASON:.status.reason")

	if err == nil && strings.TrimSpace(output) != "" && !strings.Contains(output, "No resources found") {
		fmt.Printf("🚨 Failed pods detected:\n%s\n", output)
	}
}

// checkPendingPods checks for pending pods with issues
func (t *ThanosStack) checkPendingPods(config *MonitoringConfig) {
	output, err := utils.ExecuteCommand("kubectl", "get", "pods", "-n", config.Namespace,
		"--field-selector=status.phase=Pending", "-o", "custom-columns=NAME:.metadata.name,STATUS:.status.phase,REASON:.status.conditions[0].reason")

	if err == nil && strings.TrimSpace(output) != "" && !strings.Contains(output, "No resources found") {
		fmt.Printf("⏳ Pending pods with issues:\n%s\n", output)
	}
}

// checkRecentEvents checks for recent error events
func (t *ThanosStack) checkRecentEvents(config *MonitoringConfig) {
	output, err := utils.ExecuteCommand("kubectl", "get", "events", "-n", config.Namespace,
		"--field-selector=type=Warning", "--sort-by=.lastTimestamp", "-o", "custom-columns=TIME:.lastTimestamp,REASON:.reason,MESSAGE:.message")

	if err == nil && strings.TrimSpace(output) != "" && !strings.Contains(output, "No resources found") {
		lines := strings.Split(output, "\n")
		if len(lines) > 1 { // More than just header
			// Safely calculate the start index to avoid negative slice bounds
			startIndex := len(lines) - 3
			if startIndex < 1 { // Ensure we don't go below the header line
				startIndex = 1
			}
			fmt.Printf("⚠️  Recent warning events:\n%s\n", strings.Join(lines[startIndex:], "\n"))
		}
	}
}

// analyzeFailedPods provides detailed analysis of failed pods
func (t *ThanosStack) analyzeFailedPods(config *MonitoringConfig) {
	// Get all pods in the namespace
	output, err := utils.ExecuteCommand("kubectl", "get", "pods", "-n", config.Namespace, "-o", "wide")
	if err != nil {
		fmt.Printf("❌ Failed to get pods: %v\n", err)
		return
	}

	fmt.Println("All pods status:")
	fmt.Println(output)

	// Get detailed info for non-running pods
	lines := strings.Split(output, "\n")
	if len(lines) <= 1 {
		fmt.Println("No pod data available")
		return
	}

	for _, line := range lines[1:] { // Skip header
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			podName := fields[0]
			status := fields[2]

			if !strings.Contains(status, "Running") && !strings.Contains(status, "Completed") {
				fmt.Printf("\n🔍 Analyzing pod: %s (Status: %s)\n", podName, status)

				// Describe pod
				if descOutput, err := utils.ExecuteCommand("kubectl", "describe", "pod", podName, "-n", config.Namespace); err == nil {
					// Extract events section
					descLines := strings.Split(descOutput, "\n")
					inEvents := false
					for _, descLine := range descLines {
						if strings.Contains(descLine, "Events:") {
							inEvents = true
						}
						if inEvents {
							fmt.Println(descLine)
						}
					}
				}

				// Get logs if available
				if logOutput, err := utils.ExecuteCommand("kubectl", "logs", podName, "-n", config.Namespace, "--tail=10"); err == nil && logOutput != "" {
					fmt.Printf("Recent logs:\n%s\n", logOutput)
				}
			}
		}
	}
}

// checkResourceStatus checks resource quotas and node capacity
func (t *ThanosStack) checkResourceStatus(config *MonitoringConfig) {
	// Check resource quotas
	if output, err := utils.ExecuteCommand("kubectl", "get", "resourcequota", "-n", config.Namespace); err == nil {
		fmt.Printf("Resource quotas:\n%s\n", output)
	}

	// Check node resources
	if output, err := utils.ExecuteCommand("kubectl", "top", "nodes"); err == nil {
		fmt.Printf("Node resource usage:\n%s\n", output)
	} else {
		fmt.Println("⚠️  Metrics server not available for resource usage")
	}
}

// checkStorageIssues checks for storage-related problems
func (t *ThanosStack) checkStorageIssues(config *MonitoringConfig) {
	// Check PVCs with detailed status
	fmt.Println("🗄️  Checking Persistent Volume Claims...")
	if output, err := utils.ExecuteCommand("kubectl", "get", "pvc", "-n", config.Namespace, "-o", "wide"); err == nil {
		fmt.Printf("Persistent Volume Claims:\n%s\n", output)

		// Check for unbound PVCs
		if strings.Contains(output, "Pending") {
			fmt.Println("⚠️  Found pending PVCs - checking details...")

			// Get PVC details
			if pvcList, err := utils.ExecuteCommand("kubectl", "get", "pvc", "-n", config.Namespace, "-o", "name"); err == nil {
				pvcs := strings.Split(strings.TrimSpace(pvcList), "\n")
				for _, pvc := range pvcs {
					if pvc == "" {
						continue
					}
					pvcName := strings.TrimPrefix(pvc, "persistentvolumeclaim/")

					// Describe the PVC to get error details
					if descOutput, err := utils.ExecuteCommand("kubectl", "describe", "pvc", pvcName, "-n", config.Namespace); err == nil {
						if strings.Contains(descOutput, "FailedBinding") || strings.Contains(descOutput, "no persistent volumes available") {
							fmt.Printf("❌ PVC %s binding failed:\n%s\n", pvcName, descOutput)
						}
					}
				}
			}
		}
	} else {
		fmt.Printf("❌ Failed to get PVCs: %v\n", err)
	}

	// Check storage classes
	fmt.Println("\n💾 Checking Storage Classes...")
	if output, err := utils.ExecuteCommand("kubectl", "get", "storageclass", "-o", "wide"); err == nil {
		fmt.Printf("Available Storage Classes:\n%s\n", output)

		// Check specifically for EFS StorageClass
		if !strings.Contains(output, "efs-sc") {
			fmt.Println("⚠️  EFS StorageClass 'efs-sc' not found!")
			fmt.Println("💡 This is likely a Fargate environment where EFS CSI driver is not installed")
			fmt.Println("💡 The monitoring stack should use EmptyDir volumes instead")
		}
	} else {
		fmt.Printf("❌ Failed to get storage classes: %v\n", err)
	}

	// Check for EFS CSI driver
	fmt.Println("\n🔧 Checking EFS CSI Driver...")
	if output, err := utils.ExecuteCommand("kubectl", "get", "daemonset", "-n", "kube-system", "-l", "app=efs-csi-node"); err == nil && strings.TrimSpace(output) != "" {
		fmt.Printf("EFS CSI Driver status:\n%s\n", output)
	} else {
		fmt.Println("⚠️  EFS CSI Driver not found - this confirms Fargate environment")
		fmt.Println("💡 EmptyDir volumes will be used for storage")
	}
}

// provideTroubleshootingCommands provides useful commands for manual troubleshooting
func (t *ThanosStack) provideTroubleshootingCommands(config *MonitoringConfig) {
	fmt.Printf(`
Manual Troubleshooting Commands:
================================

# Check all resources in namespace:
kubectl get all -n %s

# Watch pod status in real-time:
kubectl get pods -n %s -w

# Check events continuously:
kubectl get events -n %s -w

# Describe failed pods:
kubectl describe pods -n %s

# Check logs of specific pod:
kubectl logs <pod-name> -n %s -f

# Check Helm release history:
helm history %s -n %s

# Rollback if needed:
helm rollback %s -n %s

# Uninstall and retry:
helm uninstall %s -n %s
helm install %s %s --values %s -n %s --create-namespace

# Check cluster resources:
kubectl top nodes
kubectl get nodes -o wide

# Check for Fargate environment:
kubectl get nodes -o jsonpath='{.items[*].metadata.labels.eks\.amazonaws\.com/compute-type}'
kubectl get nodes -o jsonpath='{.items[*].metadata.name}'

# Check storage classes and EFS driver:
kubectl get storageclass
kubectl get daemonset -n kube-system -l app=efs-csi-node

# Check PVC binding issues:
kubectl get pvc -n %s -o wide
kubectl describe pvc -n %s

`, config.Namespace, config.Namespace, config.Namespace, config.Namespace,
		config.Namespace, config.HelmReleaseName, config.Namespace,
		config.HelmReleaseName, config.Namespace, config.HelmReleaseName,
		config.Namespace, config.HelmReleaseName, config.ChartsPath,
		config.ValuesFilePath, config.Namespace, config.Namespace, config.Namespace)
}

// createDashboardConfigMaps creates ConfigMaps for Grafana dashboards
// deployMonitoringInfrastructure deploys Terraform infrastructure for monitoring persistence
func (t *ThanosStack) deployMonitoringInfrastructure(ctx context.Context, config *MonitoringConfig) error {
	fmt.Println("🏗️  Deploying monitoring infrastructure with Terraform...")

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("error determining current directory: %w", err)
	}

	// Set Terraform working directory
	terraformDir := filepath.Join(cwd, "tokamak-thanos-stack", "terraform", "monitoring")
	if _, err := os.Stat(terraformDir); os.IsNotExist(err) {
		return fmt.Errorf("terraform monitoring directory not found: %s", terraformDir)
	}

	// Get cluster name from kubectl context
	clusterName, err := t.getClusterName()
	if err != nil {
		return fmt.Errorf("failed to get cluster name: %w", err)
	}

	// Get EFS file system ID for the cluster
	efsFileSystemId, err := t.getEFSFileSystemId(clusterName)
	if err != nil {
		return fmt.Errorf("failed to get EFS file system ID: %w", err)
	}

	// Change to terraform directory
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)

	if err := os.Chdir(terraformDir); err != nil {
		return fmt.Errorf("failed to change to terraform directory: %w", err)
	}

	// Initialize Terraform
	fmt.Println("🔧 Initializing Terraform...")
	if _, err := utils.ExecuteCommand("terraform", "init"); err != nil {
		return fmt.Errorf("failed to initialize terraform: %w", err)
	}

	// Apply Terraform with variables
	fmt.Println("🚀 Applying Terraform configuration...")
	applyArgs := []string{
		"apply",
		"-auto-approve",
		fmt.Sprintf("-var=cluster_name=%s", clusterName),
		fmt.Sprintf("-var=monitoring_stack_name=%s", config.ChainName),
		"-var=enable_monitoring_persistence=true",
		fmt.Sprintf("-var=aws_region=%s", t.deployConfig.AWS.Region),
		fmt.Sprintf("-var=efs_file_system_id=%s", efsFileSystemId),
	}

	if _, err := utils.ExecuteCommand("terraform", applyArgs...); err != nil {
		return fmt.Errorf("failed to apply terraform: %w", err)
	}

	fmt.Println("✅ Monitoring infrastructure deployed successfully!")
	return nil
}

// getClusterName gets the current EKS cluster name from kubectl context
func (t *ThanosStack) getClusterName() (string, error) {
	output, err := utils.ExecuteCommand("kubectl", "config", "current-context")
	if err != nil {
		return "", fmt.Errorf("failed to get current context: %w", err)
	}

	// Extract cluster name from context (format: arn:aws:eks:region:account:cluster/cluster-name)
	contextStr := strings.TrimSpace(output)
	if strings.Contains(contextStr, "/") {
		parts := strings.Split(contextStr, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// If context format is different, try to extract from cluster info
	clusterInfo, err := utils.ExecuteCommand("kubectl", "cluster-info")
	if err != nil {
		return "", fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Parse cluster info to extract cluster name
	lines := strings.Split(clusterInfo, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Kubernetes control plane") {
			// Extract cluster name from URL
			if strings.Contains(line, "https://") {
				start := strings.Index(line, "https://")
				if start != -1 {
					url := line[start:]
					if strings.Contains(url, ".") {
						parts := strings.Split(url, ".")
						if len(parts) > 0 && strings.Contains(parts[0], "-") {
							clusterParts := strings.Split(parts[0], "-")
							if len(clusterParts) >= 2 {
								return strings.Join(clusterParts[1:], "-"), nil
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("could not extract cluster name from context or cluster info")
}

// getEFSFileSystemId gets the EFS file system ID for the cluster
func (t *ThanosStack) getEFSFileSystemId(clusterName string) (string, error) {
	// Try to find EFS with cluster name tag
	output, err := utils.ExecuteCommand("aws", "efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--output", "json")
	if err != nil {
		return "", fmt.Errorf("failed to describe EFS file systems: %w", err)
	}

	// Parse JSON output to find matching EFS
	var efsResponse struct {
		FileSystems []struct {
			FileSystemId string `json:"FileSystemId"`
			Tags         []struct {
				Key   string `json:"Key"`
				Value string `json:"Value"`
			} `json:"Tags"`
		} `json:"FileSystems"`
	}

	if err := json.Unmarshal([]byte(output), &efsResponse); err != nil {
		return "", fmt.Errorf("failed to parse EFS response: %w", err)
	}

	// Look for EFS with matching cluster name
	for _, fs := range efsResponse.FileSystems {
		for _, tag := range fs.Tags {
			if tag.Key == "Name" && tag.Value == clusterName {
				return fs.FileSystemId, nil
			}
		}
	}

	return "", fmt.Errorf("no EFS file system found with Name tag matching cluster: %s", clusterName)
}

func (t *ThanosStack) createDashboardConfigMaps(config *MonitoringConfig) error {
	fmt.Println("📊 Creating dashboard ConfigMaps...")

	dashboardsPath := filepath.Join(config.ChartsPath, "dashboards")

	// Check if dashboards directory exists
	if _, err := os.Stat(dashboardsPath); os.IsNotExist(err) {
		fmt.Printf("⚠️  Dashboards directory not found: %s\n", dashboardsPath)
		return nil
	}

	// Read dashboard files
	files, err := os.ReadDir(dashboardsPath)
	if err != nil {
		return fmt.Errorf("failed to read dashboards directory: %w", err)
	}

	// Process each dashboard file
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		dashboardPath := filepath.Join(dashboardsPath, file.Name())
		dashboardContent, err := os.ReadFile(dashboardPath)
		if err != nil {
			fmt.Printf("⚠️  Failed to read dashboard %s: %v\n", file.Name(), err)
			continue
		}

		// Create ConfigMap name from filename
		configMapName := fmt.Sprintf("dashboard-%s", strings.TrimSuffix(file.Name(), ".json"))

		// Indent dashboard content for YAML
		indentedContent := strings.ReplaceAll(string(dashboardContent), "\n", "\n    ")

		// Create ConfigMap YAML
		configMapYAML := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
  labels:
    grafana_dashboard: "1"
data:
  %s: |
    %s`, configMapName, config.Namespace, file.Name(), indentedContent)

		// Write to temporary file and apply
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("dashboard-%s.yaml", configMapName))
		if err := os.WriteFile(tempFile, []byte(configMapYAML), 0644); err != nil {
			fmt.Printf("⚠️  Failed to write temp file for ConfigMap %s: %v\n", configMapName, err)
			continue
		}

		fmt.Printf("📊 Creating ConfigMap: %s\n", configMapName)
		if _, err := utils.ExecuteCommand("kubectl", "apply", "-f", tempFile); err != nil {
			fmt.Printf("⚠️  Failed to create ConfigMap %s: %v\n", configMapName, err)
		} else {
			fmt.Printf("✅ Created dashboard ConfigMap: %s\n", configMapName)
		}

		// Clean up temp file
		os.Remove(tempFile)
	}

	fmt.Println("✅ Dashboard ConfigMaps created successfully")
	return nil
}

// NOTE: generateAdditionalScrapeConfigs function has been removed
// Scrape configuration is now handled entirely by Helm templates in charts/monitoring
// This eliminates the 3-way duplication between Go code, values.yaml, and templates
