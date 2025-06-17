package thanos

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
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
	fmt.Println("📊 Creating dashboard ConfigMaps...")
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

	fmt.Printf("   # Port forward to access Prometheus locally:\n")
	fmt.Printf("   kubectl port-forward -n %s svc/%s-kube-prometheus-prometheus 9090:9090\n", config.Namespace, config.HelmReleaseName)
	fmt.Printf("   # Then visit: http://localhost:9090\n\n")

	fmt.Printf("📈 **Monitoring Targets:**\n")
	for component, serviceName := range config.ServiceNames {
		if serviceName != "" {
			fmt.Printf("   • %s: %s\n", component, serviceName)
		}
	}
	fmt.Printf("   • L1 RPC Health: %s\n\n", config.L1RpcUrl)

	fmt.Printf("🛠️  **Management Commands:**\n")
	fmt.Printf("   # Check status:\n")
	fmt.Printf("   helm status %s -n %s\n", config.HelmReleaseName, config.Namespace)
	fmt.Printf("   kubectl get pods -n %s\n\n", config.Namespace)

	fmt.Printf("   # Uninstall:\n")
	fmt.Printf("   helm uninstall %s -n %s\n", config.HelmReleaseName, config.Namespace)
}

// MonitoringConfig holds all configuration needed for monitoring installation
type MonitoringConfig struct {
	Namespace         string
	HelmReleaseName   string
	AdminPassword     string
	L1RpcUrl          string
	ServiceNames      map[string]string
	EnablePersistence bool
	ForceEFS          bool // Force EFS usage even in Fargate
	EfsId             string
	ChartsPath        string
	ValuesFilePath    string
	ChainName         string
}

// getMonitoringConfig gathers all required configuration for monitoring
func (t *ThanosStack) getMonitoringConfig(ctx context.Context) (*MonitoringConfig, error) {
	config := &MonitoringConfig{
		Namespace: "monitoring",
		L1RpcUrl:  t.deployConfig.L1RPCURL,
		ChainName: t.deployConfig.ChainName,
	}

	// Generate unique release name with timestamp
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

	// Check for Fargate environment first
	isFargate := t.detectFargateEnvironment()

	// Ask user about EFS preference
	fmt.Println("\n📦 Storage Configuration:")
	if isFargate {
		fmt.Println("🚀 Detected AWS Fargate environment")
		fmt.Println("⚠️  Fargate has limited storage options")
	}

	// Check if EFS StorageClass exists
	_, scErr := utils.ExecuteCommand("kubectl", "get", "storageclass", "efs-sc", "-o", "name")

	// Check if EFS CSI driver is actually running
	efsCSIAvailable := false
	if scErr == nil {
		fmt.Println("✅ EFS StorageClass 'efs-sc' found")

		// Check for EFS CSI controller (works in both EC2 and Fargate)
		_, controllerErr := utils.ExecuteCommand("kubectl", "get", "deployment", "-n", "kube-system", "-l", "app=efs-csi-controller", "-o", "name")
		if controllerErr == nil {
			// Check if controller pods are running
			controllerStatus, _ := utils.ExecuteCommand("kubectl", "get", "pods", "-n", "kube-system", "-l", "app=efs-csi-controller", "--field-selector=status.phase=Running", "-o", "name")
			if strings.TrimSpace(controllerStatus) != "" {
				fmt.Println("✅ EFS CSI Controller is running")
				efsCSIAvailable = true
			} else {
				fmt.Println("⚠️  EFS CSI Controller is not ready")
				efsCSIAvailable = false
			}
		} else {
			fmt.Println("❌ EFS CSI Controller not found")
			efsCSIAvailable = false
		}
	} else {
		fmt.Printf("❌ EFS StorageClass 'efs-sc' not found: %s\n", scErr)
		efsCSIAvailable = false
	}

	if !efsCSIAvailable {
		fmt.Println("📝 EFS not available - using EmptyDir volumes only")
		if isFargate {
			fmt.Println("💡 For EFS support in Fargate, install EFS CSI driver:")
			fmt.Println("   kubectl apply -k \"github.com/kubernetes-sigs/aws-efs-csi-driver/deploy/kubernetes/overlays/stable/ecr/?ref=release-1.7\"")
		}
		config.EnablePersistence = false
		config.ForceEFS = false
	} else {
		// Ask user preference
		fmt.Print("🤔 Do you want to use EFS for persistent storage? (y/n): ")
		useEFS, err := scanner.ScanString()
		if err != nil {
			return nil, fmt.Errorf("error reading EFS preference: %w", err)
		}

		if strings.ToLower(useEFS) == "y" || strings.ToLower(useEFS) == "yes" {
			config.ForceEFS = true

			// Try to get or create EFS
			efsId, err := t.getEFSFileSystemId(ctx)
			if err != nil {
				fmt.Printf("⚠️  EFS not available: %s\n", err)
				fmt.Println("🔧 EFS will be created automatically during installation")
				config.EnablePersistence = true
				config.EfsId = ""
			} else {
				fmt.Printf("✅ EFS available: %s\n", efsId)
				config.EnablePersistence = true
				config.EfsId = efsId
			}

			if isFargate {
				fmt.Println("📝 Applying EFS + Fargate configuration:")
				fmt.Println("   • Using EFS volumes for persistent storage")
				fmt.Println("   • Applying Fargate-compatible SecurityContext")
				fmt.Println("   • Using non-root user (UID: 65534)")
			} else {
				fmt.Println("📝 Using EFS for persistent storage")
			}
		} else {
			fmt.Println("📝 Using EmptyDir volumes (no persistent storage)")
			config.EnablePersistence = false
			config.ForceEFS = false

			if isFargate {
				fmt.Println("📝 Applying Fargate-compatible configuration:")
				fmt.Println("   • Using EmptyDir volumes")
				fmt.Println("   • Applying Fargate-compatible SecurityContext")
				fmt.Println("   • Using non-root user (UID: 65534)")
			}
		}
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

// getFargateSecurityContext returns Fargate-compatible security context
func (t *ThanosStack) getFargateSecurityContext() *types.SecurityContext {
	runAsNonRoot := true
	runAsUser := int64(65534) // nobody user
	runAsGroup := int64(65534)
	readOnlyRootFilesystem := false
	allowPrivilegeEscalation := false

	return &types.SecurityContext{
		RunAsNonRoot:             &runAsNonRoot,
		RunAsUser:                &runAsUser,
		RunAsGroup:               &runAsGroup,
		ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
		AllowPrivilegeEscalation: &allowPrivilegeEscalation,
	}
}

// getFargatePodSecurityContext returns Fargate-compatible pod security context
func (t *ThanosStack) getFargatePodSecurityContext() *types.PodSecurityContext {
	runAsNonRoot := true
	runAsUser := int64(65534) // nobody user
	runAsGroup := int64(65534)
	fsGroup := int64(65534)

	return &types.PodSecurityContext{
		RunAsNonRoot: &runAsNonRoot,
		RunAsUser:    &runAsUser,
		RunAsGroup:   &runAsGroup,
		FSGroup:      &fsGroup,
	}
}

// generateValuesFile creates the values.yaml file for monitoring configuration
func (t *ThanosStack) generateValuesFile(config *MonitoringConfig) error {
	fmt.Println("📝 Generating monitoring values file...")

	// Create monitoring configuration using structured types
	valuesConfig := types.ThanosStackMonitoringConfig{
		Global: types.GlobalConfig{
			L1RpcUrl: config.L1RpcUrl,
			Storage: types.StorageConfig{
				Enabled:         config.EnablePersistence,
				StorageClass:    "efs-sc",
				EfsFileSystemId: config.EfsId,
				ForceEFS:        config.ForceEFS,
				Prometheus: types.PrometheusStorageConfig{
					Size: "50Gi",
				},
				Grafana: types.GrafanaStorageConfig{
					Size: "10Gi",
				},
			},
		},
		ThanosStack: types.ThanosStackConfig{
			ReleaseName: config.ChainName,
			Namespace:   config.Namespace,
			ChainName:   config.ChainName,
		},
		KubePrometheusStack: types.KubePrometheusStackConfig{
			Enabled: true,
			Prometheus: types.PrometheusConfig{
				PrometheusSpec: types.PrometheusSpecConfig{
					Resources: types.ResourcesConfig{
						Requests: types.ResourceRequests{
							CPU:    "1500m",
							Memory: "3Gi",
						},
						Limits: types.ResourceLimits{
							CPU:    "2000m",
							Memory: "4Gi",
						},
					},
					Retention:               "1y",
					RetentionSize:           "10GB",
					ScrapeInterval:          "30s",
					EvaluationInterval:      "30s",
					StorageSpec:             t.getStorageSpecTyped(config),
					AdditionalScrapeConfigs: t.generateScrapeConfigsTyped(config),
					SecurityContext:         t.getFargateSecurityContext(),
					PodSecurityContext:      t.getFargatePodSecurityContext(),
				},
			},
			Grafana: types.GrafanaConfig{
				Enabled:       true,
				AdminUser:     "admin",
				AdminPassword: config.AdminPassword,
				Resources: types.ResourcesConfig{
					Requests: types.ResourceRequests{
						CPU:    "500m",
						Memory: "1Gi",
					},
					Limits: types.ResourceLimits{
						CPU:    "1000m",
						Memory: "2Gi",
					},
				},
				Persistence: t.getGrafanaPersistenceTyped(config),
				Ingress: types.IngressConfig{
					Enabled:   true,
					ClassName: "alb",
					Annotations: map[string]string{
						"alb.ingress.kubernetes.io/scheme":      "internet-facing",
						"alb.ingress.kubernetes.io/target-type": "ip",
						"alb.ingress.kubernetes.io/group.name":  "thanos-monitoring",
					},
				},
				Sidecar: types.SidecarConfig{
					Dashboards: types.DashboardSidecarConfig{
						Enabled:         true,
						Label:           "grafana_dashboard",
						LabelValue:      "1",
						SearchNamespace: "ALL",
					},
				},
				SecurityContext:    t.getFargateSecurityContext(),
				PodSecurityContext: t.getFargatePodSecurityContext(),
			},
			Alertmanager: types.AlertmanagerConfig{
				Enabled: false,
			},
			NodeExporter: types.NodeExporterConfig{
				Enabled: false, // Always disabled for Fargate compatibility
			},
			KubeStateMetrics: types.KubeStateMetricsConfig{
				Enabled: true,
			},
		},
		PrometheusBlackboxExporter: types.BlackboxExporterConfig{
			Enabled: true,
			Config: types.BlackboxConfig{
				Modules: map[string]types.BlackboxModule{
					"http_post_eth_node_synced_2xx": {
						Prober: "http",
						HTTP: &types.BlackboxHTTPConfig{
							Method: "POST",
							Headers: map[string]string{
								"Content-Type": "application/json",
							},
							Body:                       `{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`,
							ValidStatusCodes:           []int{200},
							FailIfBodyNotMatchesRegexp: []string{"false"},
						},
					},
					"http_post_eth_block_number_2xx": {
						Prober: "http",
						HTTP: &types.BlackboxHTTPConfig{
							Method: "POST",
							Headers: map[string]string{
								"Content-Type": "application/json",
							},
							Body:                       `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":83}`,
							ValidStatusCodes:           []int{200},
							FailIfBodyNotMatchesRegexp: []string{"0x"},
						},
					},
					"tcp_connect": {
						Prober: "tcp",
					},
				},
			},
			Resources: types.ResourcesConfig{
				Requests: types.ResourceRequests{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Limits: types.ResourceLimits{
					CPU:    "500m",
					Memory: "256Mi",
				},
			},
			ServiceMonitor: types.BlackboxServiceMonitorConfig{
				Enabled: true,
				Defaults: types.BlackboxServiceMonitorDefaults{
					Labels: map[string]string{
						"app": "blackbox-exporter",
					},
					Interval:      "30s",
					ScrapeTimeout: "30s",
					Targets: []types.BlackboxTarget{
						{
							Name:   "blackbox-eth-node-synced",
							URL:    config.L1RpcUrl,
							Module: "http_post_eth_node_synced_2xx",
						},
						{
							Name:   "blackbox-eth-block-number",
							URL:    config.L1RpcUrl,
							Module: "http_post_eth_block_number_2xx",
						},
					},
				},
			},
		},
	}

	// Marshal to YAML
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

// getStorageSpecTyped returns storage configuration using typed structs
func (t *ThanosStack) getStorageSpecTyped(config *MonitoringConfig) *types.StorageSpecConfig {
	if !config.EnablePersistence {
		return nil
	}

	// Use EFS-based storage if ForceEFS is enabled or EFS is available
	if config.ForceEFS || config.EfsId != "" {
		return &types.StorageSpecConfig{
			VolumeClaimTemplate: types.VolumeClaimTemplateConfig{
				Spec: types.VolumeClaimSpec{
					StorageClassName: "efs-sc",
					AccessModes:      []string{"ReadWriteMany"},
					Resources: types.VolumeClaimResources{
						Requests: types.VolumeClaimRequests{
							Storage: "50Gi",
						},
					},
				},
			},
		}
	}

	// For Fargate or when EFS is not available, return nil to use EmptyDir
	return nil
}

// getGrafanaPersistenceTyped returns Grafana persistence configuration using typed structs
func (t *ThanosStack) getGrafanaPersistenceTyped(config *MonitoringConfig) types.PersistenceConfig {
	if !config.EnablePersistence {
		return types.PersistenceConfig{
			Enabled: false,
		}
	}

	// Use EFS if ForceEFS is enabled or EFS is available
	if config.ForceEFS || config.EfsId != "" {
		return types.PersistenceConfig{
			Enabled:      true,
			Size:         "10Gi",
			StorageClass: "efs-sc",
		}
	}

	return types.PersistenceConfig{
		Enabled: false,
	}
}

// generateScrapeConfigsTyped creates Prometheus scrape configurations using typed structs
func (t *ThanosStack) generateScrapeConfigsTyped(config *MonitoringConfig) []types.ScrapeConfig {
	scrapeConfigs := []types.ScrapeConfig{}

	// OP Stack components scrape configs
	components := map[string]map[string]interface{}{
		"op-node": {
			"port": 7300,
			"path": "/metrics",
		},
		"op-batcher": {
			"port": 7300,
			"path": "/metrics",
		},
		"op-proposer": {
			"port": 7300,
			"path": "/metrics",
		},
		"op-geth": {
			"port": 6060,
			"path": "/debug/metrics/prometheus",
		},
		"blockscout": {
			"port": 3000,
			"path": "/metrics",
		},
		"block-explorer-frontend": {
			"port": 80,
			"path": "/api/healthz",
		},
	}

	for component, settings := range components {
		if serviceName, exists := config.ServiceNames[component]; exists && serviceName != "" {
			// Use FQDN for cross-namespace service access
			serviceTarget := fmt.Sprintf("%s.%s.svc.cluster.local:%d",
				serviceName,
				t.deployConfig.K8s.Namespace, // Original Thanos Stack namespace
				settings["port"])

			scrapeConfig := types.ScrapeConfig{
				JobName: component,
				StaticConfigs: []types.StaticConfig{
					{
						Targets: []string{serviceTarget},
					},
				},
				ScrapeInterval: "30s",
				MetricsPath:    settings["path"].(string),
			}
			scrapeConfigs = append(scrapeConfigs, scrapeConfig)
		}
	}

	// Blackbox exporter scrape configs for L1 RPC health checks
	blackboxConfigs := []types.ScrapeConfig{
		{
			JobName: "blackbox-eth-node-synced",
			StaticConfigs: []types.StaticConfig{
				{
					Targets: []string{config.L1RpcUrl},
				},
			},
			MetricsPath: "/probe",
			Params: map[string][]string{
				"module": {"http_post_eth_node_synced_2xx"},
			},
			RelabelConfigs: []types.RelabelConfig{
				{
					SourceLabels: []string{"__address__"},
					TargetLabel:  "__param_target",
				},
				{
					SourceLabels: []string{"__param_target"},
					TargetLabel:  "instance",
				},
				{
					TargetLabel: "__address__",
					Replacement: fmt.Sprintf("%s-prometheus-blackbox-exporter:9115", config.HelmReleaseName),
				},
			},
		},
		{
			JobName: "blackbox-eth-block-number",
			StaticConfigs: []types.StaticConfig{
				{
					Targets: []string{config.L1RpcUrl},
				},
			},
			MetricsPath: "/probe",
			Params: map[string][]string{
				"module": {"http_post_eth_block_number_2xx"},
			},
			RelabelConfigs: []types.RelabelConfig{
				{
					SourceLabels: []string{"__address__"},
					TargetLabel:  "__param_target",
				},
				{
					SourceLabels: []string{"__param_target"},
					TargetLabel:  "instance",
				},
				{
					TargetLabel: "__address__",
					Replacement: fmt.Sprintf("%s-prometheus-blackbox-exporter:9115", config.HelmReleaseName),
				},
			},
		},
	}

	scrapeConfigs = append(scrapeConfigs, blackboxConfigs...)

	return scrapeConfigs
}

// getServiceNames discovers service names in the namespace using trh-sdk patterns
func (t *ThanosStack) getServiceNames(namespace, chainName string) (map[string]string, error) {
	serviceNames := map[string]string{
		"op-node":                 "",
		"op-batcher":              "",
		"op-proposer":             "",
		"op-geth":                 "",
		"blockscout":              "",
		"block-explorer-frontend": "",
	}

	// Get all services in the namespace
	services, err := utils.ExecuteCommand("kubectl", []string{
		"get", "services",
		"-n", namespace,
		"-o", "jsonpath={.items[*].metadata.name}",
	}...)
	if err != nil {
		fmt.Printf("⚠️  Could not get services from namespace %s: %s\n", namespace, err)
		fmt.Println("📝 Using default trh-sdk service naming pattern")
	} else {
		serviceList := strings.Split(strings.TrimSpace(services), " ")

		// Match services to components
		for _, service := range serviceList {
			service = strings.TrimSpace(service)
			if service == "" {
				continue
			}

			switch {
			case strings.Contains(service, "op-node") && !strings.Contains(service, "op-geth"):
				serviceNames["op-node"] = service
			case strings.Contains(service, "op-batcher"):
				serviceNames["op-batcher"] = service
			case strings.Contains(service, "op-proposer"):
				serviceNames["op-proposer"] = service
			case strings.Contains(service, "op-geth") || strings.Contains(service, "geth"):
				serviceNames["op-geth"] = service
			case strings.Contains(service, "blockscout") && !strings.Contains(service, "frontend"):
				serviceNames["blockscout"] = service
			case strings.Contains(service, "block-explorer-fe") && strings.Contains(service, "frontend"):
				serviceNames["block-explorer-frontend"] = service
			}
		}
	}

	// Set trh-sdk default names if not found (without namespace suffix as per requirements)
	if serviceNames["op-node"] == "" {
		serviceNames["op-node"] = fmt.Sprintf("%s-thanos-stack-op-node", chainName)
	}
	if serviceNames["op-batcher"] == "" {
		serviceNames["op-batcher"] = fmt.Sprintf("%s-thanos-stack-op-batcher", chainName)
	}
	if serviceNames["op-proposer"] == "" {
		serviceNames["op-proposer"] = fmt.Sprintf("%s-thanos-stack-op-proposer", chainName)
	}
	if serviceNames["op-geth"] == "" {
		serviceNames["op-geth"] = fmt.Sprintf("%s-thanos-stack-op-geth", chainName)
	}
	if serviceNames["blockscout"] == "" {
		serviceNames["blockscout"] = fmt.Sprintf("%s-thanos-stack-blockscout", chainName)
	}
	if serviceNames["block-explorer-frontend"] == "" {
		serviceNames["block-explorer-frontend"] = fmt.Sprintf("%s-thanos-stack-block-explorer-frontend", chainName)
	}

	fmt.Printf("📋 Discovered service names:\n")
	for component, serviceName := range serviceNames {
		if serviceName != "" {
			fmt.Printf("   - %s: %s\n", component, serviceName)
		}
	}

	return serviceNames, nil
}

func (t *ThanosStack) uninstallMonitoring(ctx context.Context) error {
	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	_, _, err := t.loginAWS(ctx)
	if err != nil {
		fmt.Println("Error to login in AWS:", err)
		return err
	}

	if t.deployConfig.AWS == nil {
		return fmt.Errorf("AWS configuration is not set. Please run the deploy command first")
	}

	// Find monitoring releases
	releases, err := utils.FilterHelmReleases(namespace, "monitoring")
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
			namespace,
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
func (t *ThanosStack) getEFSFileSystemId(ctx context.Context) (string, error) {
	// Check if EFS is available in the region
	efsOutput, err := utils.ExecuteCommand("aws", []string{
		"efs", "describe-file-systems",
		"--region", t.deployConfig.AWS.Region,
		"--query", "FileSystems[?Tags[?Key=='Name' && Value=='thanos-monitoring']].FileSystemId",
		"--output", "text",
	}...)
	if err != nil {
		fmt.Printf("⚠️  Could not find existing EFS file system: %s\n", err)
		return "", fmt.Errorf("failed to check EFS: %s", err)
	}

	efsId := strings.TrimSpace(efsOutput)
	if efsId == "" || efsId == "None" {
		// Try to create a new EFS file system
		fmt.Println("🔧 Creating new EFS file system for monitoring...")
		createOutput, createErr := utils.ExecuteCommand("aws", []string{
			"efs", "create-file-system",
			"--region", t.deployConfig.AWS.Region,
			"--performance-mode", "generalPurpose",
			"--throughput-mode", "provisioned",
			"--provisioned-throughput-in-mibps", "100",
			"--tags", "Key=Name,Value=thanos-monitoring",
			"--query", "FileSystemId",
			"--output", "text",
		}...)
		if createErr != nil {
			fmt.Printf("❌ Failed to create EFS file system: %s\n", createErr)
			return "", fmt.Errorf("failed to create EFS: %s", createErr)
		}

		efsId = strings.TrimSpace(createOutput)
		fmt.Printf("✅ Created new EFS file system: %s\n", efsId)

		// Wait for EFS to be available
		fmt.Println("⏳ Waiting for EFS to become available...")
		for i := 0; i < 30; i++ {
			statusOutput, _ := utils.ExecuteCommand("aws", []string{
				"efs", "describe-file-systems",
				"--region", t.deployConfig.AWS.Region,
				"--file-system-id", efsId,
				"--query", "FileSystems[0].LifeCycleState",
				"--output", "text",
			}...)

			if strings.TrimSpace(statusOutput) == "available" {
				fmt.Printf("✅ EFS file system %s is now available\n", efsId)

				// Create mount targets if needed
				t.ensureEFSMountTargets(efsId)

				return efsId, nil
			}

			fmt.Printf("⏳ EFS status: %s (waiting...)\n", strings.TrimSpace(statusOutput))
			time.Sleep(10 * time.Second)
		}

		return efsId, nil
	}

	fmt.Printf("✅ Found existing EFS file system: %s\n", efsId)
	return efsId, nil
}

// ensureEFSMountTargets creates EFS mount targets for all subnets if they don't exist
func (t *ThanosStack) ensureEFSMountTargets(efsId string) {
	fmt.Println("🔧 Checking EFS mount targets...")

	// Get existing mount targets
	mountTargetsOutput, err := utils.ExecuteCommand("aws", "efs", "describe-mount-targets",
		"--region", t.deployConfig.AWS.Region,
		"--file-system-id", efsId,
		"--query", "MountTargets[].SubnetId",
		"--output", "text")

	if err != nil {
		fmt.Printf("⚠️  Could not check mount targets: %s\n", err)
		return
	}

	existingSubnets := strings.Fields(strings.TrimSpace(mountTargetsOutput))
	fmt.Printf("📍 Found %d existing mount targets\n", len(existingSubnets))

	if len(existingSubnets) > 0 {
		fmt.Printf("✅ EFS mount targets already exist in %d subnets\n", len(existingSubnets))
	} else {
		fmt.Println("💡 EFS mount targets will be created automatically by the EFS CSI driver")
		fmt.Println("💡 This may take a few minutes during first PVC creation")
	}
}

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

			// Test if the endpoint is actually accessible
			fmt.Println("   🔍 Testing endpoint accessibility...")
			if t.testEndpointAccessibility(grafanaURL) {
				fmt.Println("   ✅ Endpoint is accessible!")
				return grafanaURL
			} else {
				fmt.Println("   ⚠️  Endpoint created but not yet accessible, continuing to wait...")
			}
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

// testEndpointAccessibility tests if the endpoint is accessible
func (t *ThanosStack) testEndpointAccessibility(url string) bool {
	// Simple curl test to check if endpoint responds
	_, err := utils.ExecuteCommand("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
		"--connect-timeout", "5", "--max-time", "10", url)
	return err == nil
}

// monitorInstallationErrors monitors installation progress and reports errors in real-time
func (t *ThanosStack) monitorInstallationErrors(config *MonitoringConfig, errorChan chan error) {
	fmt.Println("🔍 Starting installation error monitoring...")

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
