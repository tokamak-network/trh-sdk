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
	ChainName         string
	// AlertManager configuration
	AlertManager AlertManagerConfig
	// Alert rules configuration
	AlertRules map[string]AlertRule
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

// MonitoringInfo holds information about the installed monitoring stack
type MonitoringInfo struct {
	GrafanaURL   string             `json:"grafanaUrl"`
	Username     string             `json:"username"`
	Password     string             `json:"password"`
	Namespace    string             `json:"namespace"`
	ReleaseName  string             `json:"releaseName"`
	ChainName    string             `json:"chainName"`
	AlertManager AlertManagerConfig `json:"alertManager"`
}

// MonitoringStatus holds current monitoring status information
type MonitoringStatus struct {
	NamespaceExists     bool `json:"namespaceExists"`
	AlertManagerRunning bool `json:"alertManagerRunning"`
	PrometheusRunning   bool `json:"prometheusRunning"`
	EmailEnabled        bool `json:"emailEnabled"`
	TelegramEnabled     bool `json:"telegramEnabled"`
}

// AlertRule represents a single alert rule configuration
type AlertRule struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Severity    string            `json:"severity"`
	Threshold   string            `json:"threshold"`
	Enabled     bool              `json:"enabled"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}
