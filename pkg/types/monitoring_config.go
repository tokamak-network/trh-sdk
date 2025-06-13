package types

// MonitoringValuesConfig represents the new values.yaml structure for monitoring
type MonitoringValuesConfig struct {
	Global struct {
		L1RpcUrl   string `yaml:"l1RpcUrl"`
		Dashboards struct {
			AutoImport bool `yaml:"autoImport"`
		} `yaml:"dashboards"`
	} `yaml:"global"`

	EnablePersistence bool `yaml:"enablePersistence"`

	Prometheus struct {
		Enabled   bool `yaml:"enabled"`
		Resources struct {
			Requests struct {
				CPU    string `yaml:"cpu"`
				Memory string `yaml:"memory"`
			} `yaml:"requests"`
		} `yaml:"resources"`
		Retention          string `yaml:"retention"`
		RetentionSize      string `yaml:"retentionSize"`
		ScrapeInterval     string `yaml:"scrapeInterval"`
		EvaluationInterval string `yaml:"evaluationInterval"`
		Persistence        struct {
			Enabled      bool   `yaml:"enabled"`
			StorageClass string `yaml:"storageClass"`
			Size         string `yaml:"size"`
		} `yaml:"persistence"`
		Volume struct {
			Capacity         string `yaml:"capacity"`
			StorageClassName string `yaml:"storageClassName"`
			CSI              struct {
				Driver       string `yaml:"driver"`
				VolumeHandle string `yaml:"volumeHandle"`
			} `yaml:"csi"`
		} `yaml:"volume"`
		ScrapeConfigs []ScrapeConfig `yaml:"scrapeConfigs"`
	} `yaml:"prometheus"`

	Grafana struct {
		Enabled       bool   `yaml:"enabled"`
		AdminUser     string `yaml:"adminUser"`
		AdminPassword string `yaml:"adminPassword"`
		Resources     struct {
			Requests struct {
				CPU    string `yaml:"cpu"`
				Memory string `yaml:"memory"`
			} `yaml:"requests"`
		} `yaml:"resources"`
		Persistence struct {
			Enabled      bool   `yaml:"enabled"`
			Size         string `yaml:"size"`
			StorageClass string `yaml:"storageClass"`
		} `yaml:"persistence"`
		Volume struct {
			Capacity         string `yaml:"capacity"`
			StorageClassName string `yaml:"storageClassName"`
			CSI              struct {
				Driver       string `yaml:"driver"`
				VolumeHandle string `yaml:"volumeHandle"`
			} `yaml:"csi"`
		} `yaml:"volume"`
		Service struct {
			Type       string `yaml:"type"`
			Port       int    `yaml:"port"`
			TargetPort int    `yaml:"targetPort"`
		} `yaml:"service"`
		Ingress struct {
			Enabled     bool              `yaml:"enabled"`
			ClassName   string            `yaml:"className"`
			Annotations map[string]string `yaml:"annotations"`
			Hosts       []IngressHost     `yaml:"hosts"`
		} `yaml:"ingress"`
		Datasources []Datasource `yaml:"datasources"`
		Dashboards  struct {
			Enabled string `yaml:"enabled"`
		} `yaml:"dashboards"`
	} `yaml:"grafana"`

	BlackboxExporter struct {
		Enabled bool `yaml:"enabled"`
		Service struct {
			Type string `yaml:"type"`
			Port int    `yaml:"port"`
		} `yaml:"service"`
	} `yaml:"blackboxExporter"`
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

// IngressHost represents an ingress host configuration
type IngressHost struct {
	Host  string        `yaml:"host"`
	Paths []IngressPath `yaml:"paths"`
}

// IngressPath represents an ingress path configuration
type IngressPath struct {
	Path     string `yaml:"path"`
	PathType string `yaml:"pathType"`
}

// Datasource represents a Grafana datasource configuration
type Datasource struct {
	Name      string `yaml:"name"`
	Type      string `yaml:"type"`
	URL       string `yaml:"url"`
	Access    string `yaml:"access"`
	IsDefault bool   `yaml:"isDefault"`
}
