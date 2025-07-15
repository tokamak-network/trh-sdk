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
