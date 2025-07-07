package types

// MonitoringConfig holds all configuration needed for monitoring installation
type MonitoringConfig struct {
	Namespace         string
	HelmReleaseName   string
	AdminPassword     string
	L1RpcUrl          string
	ServiceNames      map[string]string
	EnablePersistence bool
	EFSFileSystemId   string
	ChartsPath        string
	ValuesFilePath    string
	ResourceName      string
	// AlertManager configuration
	AlertManager AlertManagerConfig
}

// AlertManagerConfig holds alertmanager-specific configuration
type AlertManagerConfig struct {
	Telegram TelegramConfig
	Email    EmailConfig
}

// TelegramConfig holds Telegram notification configuration
type TelegramConfig struct {
	Enabled           bool
	ApiToken          string
	CriticalReceivers []TelegramReceiver
}

// TelegramReceiver represents a Telegram chat recipient
type TelegramReceiver struct {
	ChatId string
}

// EmailConfig holds email notification configuration
type EmailConfig struct {
	Enabled           bool
	SmtpSmarthost     string
	SmtpFrom          string
	SmtpAuthUsername  string
	SmtpAuthPassword  string
	DefaultReceivers  []string
	CriticalReceivers []string
}

// ThanosStackMonitoringConfig represents the new Helm dependencies-based monitoring configuration
type ThanosStackMonitoringConfig struct {
	Global                     GlobalConfig              `yaml:"global"`
	ThanosStack                ThanosStackConfig         `yaml:"thanosStack"`
	KubePrometheusStack        KubePrometheusStackConfig `yaml:"kube-prometheus-stack"`
	PrometheusBlackboxExporter BlackboxExporterConfig    `yaml:"prometheus-blackbox-exporter"`
}

// GlobalConfig represents global configuration values
type GlobalConfig struct {
	L1RpcUrl string        `yaml:"l1RpcUrl"`
	Storage  StorageConfig `yaml:"storage"`
}

// StorageConfig represents global storage configuration
type StorageConfig struct {
	Enabled      bool                    `yaml:"enabled"`
	StorageClass string                  `yaml:"storageClass"`
	Prometheus   PrometheusStorageConfig `yaml:"prometheus"`
	Grafana      GrafanaStorageConfig    `yaml:"grafana"`
}

// PrometheusStorageConfig represents Prometheus storage configuration
type PrometheusStorageConfig struct {
	Size string `yaml:"size"`
}

// GrafanaStorageConfig represents Grafana storage configuration
type GrafanaStorageConfig struct {
	Size string `yaml:"size"`
}

// ThanosStackConfig represents trh-sdk specific configuration
type ThanosStackConfig struct {
	ReleaseName string `yaml:"releaseName"`
	Namespace   string `yaml:"namespace"`
	ChainName   string `yaml:"chainName"`
}

// KubePrometheusStackConfig represents kube-prometheus-stack subchart configuration
type KubePrometheusStackConfig struct {
	Enabled          bool                   `yaml:"enabled"`
	Prometheus       PrometheusConfig       `yaml:"prometheus"`
	Grafana          GrafanaConfig          `yaml:"grafana"`
	Alertmanager     AlertmanagerConfig     `yaml:"alertmanager"`
	NodeExporter     NodeExporterConfig     `yaml:"nodeExporter"`
	KubeStateMetrics KubeStateMetricsConfig `yaml:"kubeStateMetrics"`
}

// PrometheusConfig represents Prometheus configuration
type PrometheusConfig struct {
	PrometheusSpec PrometheusSpecConfig `yaml:"prometheusSpec"`
}

// PrometheusSpecConfig represents Prometheus spec configuration
type PrometheusSpecConfig struct {
	Resources               ResourcesConfig     `yaml:"resources"`
	Retention               string              `yaml:"retention"`
	RetentionSize           string              `yaml:"retentionSize"`
	ScrapeInterval          string              `yaml:"scrapeInterval"`
	EvaluationInterval      string              `yaml:"evaluationInterval"`
	StorageSpec             *StorageSpecConfig  `yaml:"storageSpec,omitempty"`
	AdditionalScrapeConfigs []ScrapeConfig      `yaml:"additionalScrapeConfigs"`
	SecurityContext         *SecurityContext    `yaml:"securityContext,omitempty"`
	PodSecurityContext      *PodSecurityContext `yaml:"podSecurityContext,omitempty"`
}

// GrafanaConfig represents Grafana configuration
type GrafanaConfig struct {
	Enabled            bool                `yaml:"enabled"`
	AdminUser          string              `yaml:"adminUser"`
	AdminPassword      string              `yaml:"adminPassword"`
	Resources          ResourcesConfig     `yaml:"resources"`
	Persistence        PersistenceConfig   `yaml:"persistence"`
	Ingress            IngressConfig       `yaml:"ingress"`
	Sidecar            SidecarConfig       `yaml:"sidecar"`
	SecurityContext    *SecurityContext    `yaml:"securityContext,omitempty"`
	PodSecurityContext *PodSecurityContext `yaml:"podSecurityContext,omitempty"`
}

// BlackboxExporterConfig represents prometheus-blackbox-exporter subchart configuration
type BlackboxExporterConfig struct {
	Enabled        bool                         `yaml:"enabled"`
	Config         BlackboxConfig               `yaml:"config"`
	Resources      ResourcesConfig              `yaml:"resources"`
	ServiceMonitor BlackboxServiceMonitorConfig `yaml:"serviceMonitor"`
}

// BlackboxConfig represents blackbox exporter configuration
type BlackboxConfig struct {
	Modules map[string]BlackboxModule `yaml:"modules"`
}

// BlackboxModule represents a blackbox module configuration
type BlackboxModule struct {
	Prober string              `yaml:"prober"`
	HTTP   *BlackboxHTTPConfig `yaml:"http,omitempty"`
}

// BlackboxHTTPConfig represents HTTP probe configuration
type BlackboxHTTPConfig struct {
	Method                     string            `yaml:"method"`
	Headers                    map[string]string `yaml:"headers"`
	Body                       string            `yaml:"body"`
	ValidStatusCodes           []int             `yaml:"valid_status_codes"`
	FailIfBodyNotMatchesRegexp []string          `yaml:"fail_if_body_not_matches_regexp"`
}

// BlackboxServiceMonitorConfig represents blackbox ServiceMonitor configuration
type BlackboxServiceMonitorConfig struct {
	Enabled  bool                           `yaml:"enabled"`
	Defaults BlackboxServiceMonitorDefaults `yaml:"defaults"`
}

// BlackboxServiceMonitorDefaults represents default ServiceMonitor settings
type BlackboxServiceMonitorDefaults struct {
	Labels        map[string]string `yaml:"labels"`
	Interval      string            `yaml:"interval"`
	ScrapeTimeout string            `yaml:"scrapeTimeout"`
	Targets       []BlackboxTarget  `yaml:"targets"`
}

// BlackboxTarget represents a blackbox monitoring target
type BlackboxTarget struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Module string `yaml:"module"`
}

// ResourcesConfig represents resource configuration
type ResourcesConfig struct {
	Requests ResourceRequests `yaml:"requests"`
	Limits   ResourceLimits   `yaml:"limits"`
}

// ResourceRequests represents resource requests
type ResourceRequests struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

// ResourceLimits represents resource limits
type ResourceLimits struct {
	CPU    string `yaml:"cpu"`
	Memory string `yaml:"memory"`
}

// StorageSpecConfig represents storage specification
type StorageSpecConfig struct {
	VolumeClaimTemplate VolumeClaimTemplateConfig `yaml:"volumeClaimTemplate"`
}

// VolumeClaimTemplateConfig represents volume claim template
type VolumeClaimTemplateConfig struct {
	Spec VolumeClaimSpec `yaml:"spec"`
}

// VolumeClaimSpec represents volume claim specification
type VolumeClaimSpec struct {
	StorageClassName string               `yaml:"storageClassName"`
	AccessModes      []string             `yaml:"accessModes"`
	Resources        VolumeClaimResources `yaml:"resources"`
}

// VolumeClaimResources represents volume claim resources
type VolumeClaimResources struct {
	Requests VolumeClaimRequests `yaml:"requests"`
}

// VolumeClaimRequests represents volume claim requests
type VolumeClaimRequests struct {
	Storage string `yaml:"storage"`
}

// PersistenceConfig represents persistence configuration
type PersistenceConfig struct {
	Enabled      bool   `yaml:"enabled"`
	Size         string `yaml:"size,omitempty"`
	StorageClass string `yaml:"storageClass,omitempty"`
}

// IngressConfig represents ingress configuration
type IngressConfig struct {
	Enabled     bool              `yaml:"enabled"`
	ClassName   string            `yaml:"className"`
	Annotations map[string]string `yaml:"annotations"`
}

// SidecarConfig represents sidecar configuration
type SidecarConfig struct {
	Dashboards DashboardSidecarConfig `yaml:"dashboards"`
}

// DashboardSidecarConfig represents dashboard sidecar configuration
type DashboardSidecarConfig struct {
	Enabled         bool   `yaml:"enabled"`
	Label           string `yaml:"label"`
	LabelValue      string `yaml:"labelValue"`
	SearchNamespace string `yaml:"searchNamespace"`
}

// AlertmanagerConfig represents Alertmanager configuration
type AlertmanagerConfig struct {
	Enabled bool `yaml:"enabled"`
}

// NodeExporterConfig represents Node Exporter configuration
type NodeExporterConfig struct {
	Enabled bool `yaml:"enabled"`
}

// KubeStateMetricsConfig represents Kube State Metrics configuration
type KubeStateMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

// ScrapeConfig represents a Prometheus scrape configuration
type ScrapeConfig struct {
	JobName        string              `yaml:"job_name"`
	StaticConfigs  []StaticConfig      `yaml:"static_configs,omitempty"`
	ScrapeInterval string              `yaml:"scrape_interval,omitempty"`
	MetricsPath    string              `yaml:"metrics_path,omitempty"`
	Params         map[string][]string `yaml:"params,omitempty"`
	RelabelConfigs []RelabelConfig     `yaml:"relabel_configs,omitempty"`
}

// StaticConfig represents a static configuration for scraping
type StaticConfig struct {
	Targets []string `yaml:"targets"`
}

// RelabelConfig represents a relabel configuration
type RelabelConfig struct {
	SourceLabels []string `yaml:"source_labels,omitempty"`
	TargetLabel  string   `yaml:"target_label,omitempty"`
	Replacement  string   `yaml:"replacement,omitempty"`
}

// SecurityContext represents container security context (Fargate compatible)
type SecurityContext struct {
	RunAsNonRoot             *bool  `yaml:"runAsNonRoot,omitempty"`
	RunAsUser                *int64 `yaml:"runAsUser,omitempty"`
	RunAsGroup               *int64 `yaml:"runAsGroup,omitempty"`
	ReadOnlyRootFilesystem   *bool  `yaml:"readOnlyRootFilesystem,omitempty"`
	AllowPrivilegeEscalation *bool  `yaml:"allowPrivilegeEscalation,omitempty"`
}

// PodSecurityContext represents pod security context (Fargate compatible)
type PodSecurityContext struct {
	RunAsNonRoot *bool  `yaml:"runAsNonRoot,omitempty"`
	RunAsUser    *int64 `yaml:"runAsUser,omitempty"`
	RunAsGroup   *int64 `yaml:"runAsGroup,omitempty"`
	FSGroup      *int64 `yaml:"fsGroup,omitempty"`
}
