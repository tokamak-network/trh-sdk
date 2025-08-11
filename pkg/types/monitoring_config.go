package types

// MonitoringConfig holds all configuration needed for monitoring installation
type MonitoringConfig struct {
	// Basic configuration
	Namespace       string
	HelmReleaseName string
	ChainName       string
	AdminPassword   string
	L1RpcUrl        string
	ServiceNames    map[string]string

	// Storage configuration
	EnablePersistence bool
	EFSFileSystemId   string
	ChartsPath        string
	ValuesFilePath    string
	ResourceName      string

	// AlertManager configuration
	AlertManager AlertManagerConfig
	AlertRules   map[string]AlertRule

	// Logging enabled flag
	LoggingEnabled bool
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

// EmailReceiver represents an email recipient configuration
type EmailReceiver struct {
	To string `json:"to" yaml:"to"`
}

// EmailConfigList represents a list of email configurations
type EmailConfigList []EmailReceiver

// AlertManagerReceiver represents a receiver in AlertManager configuration
type AlertManagerReceiver struct {
	Name            string          `json:"name" yaml:"name"`
	EmailConfigs    EmailConfigList `json:"email_configs,omitempty" yaml:"email_configs,omitempty"`
	TelegramConfigs []interface{}   `json:"telegram_configs,omitempty" yaml:"telegram_configs,omitempty"`
}

// AlertManagerParsedConfig represents the parsed AlertManager configuration structure
type AlertManagerParsedConfig struct {
	Global struct {
		SmtpSmarthost string `yaml:"smtp_smarthost"`
		SmtpFrom      string `yaml:"smtp_from"`
	} `yaml:"global"`
	Receivers []AlertManagerParsedReceiver `yaml:"receivers"`
}

// AlertManagerParsedReceiver represents a receiver in parsed AlertManager configuration
type AlertManagerParsedReceiver struct {
	Name         string `yaml:"name"`
	EmailConfigs []struct {
		To string `yaml:"to"`
	} `yaml:"email_configs"`
	TelegramConfigs []struct {
		BotToken string `yaml:"bot_token"`
		ChatID   string `yaml:"chat_id"`
	} `yaml:"telegram_configs"`
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
	Alert       string            `json:"alert" yaml:"alert"`
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Severity    string            `json:"severity" yaml:"severity"`
	Threshold   string            `json:"threshold" yaml:"threshold"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Expr        string            `json:"expr" yaml:"expr"`
	For         string            `json:"for" yaml:"for"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
}

// PrometheusRule represents a PrometheusRule Kubernetes resource
type PrometheusRule struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace"`
	} `yaml:"metadata"`
	Spec PrometheusRuleSpec `yaml:"spec"`
}

// PrometheusRuleSpec represents the spec section of PrometheusRule
type PrometheusRuleSpec struct {
	Groups []PrometheusRuleGroup `yaml:"groups"`
}

// PrometheusRuleGroup represents a group of alert rules in PrometheusRule
type PrometheusRuleGroup struct {
	Name  string      `yaml:"name"`
	Rules []AlertRule `yaml:"rules"`
}

// PrometheusRuleList represents a list of PrometheusRule resources
type PrometheusRuleList struct {
	APIVersion string           `yaml:"apiVersion"`
	Kind       string           `yaml:"kind"`
	Items      []PrometheusRule `yaml:"items"`
}

// AlertRuleConfig represents the configuration for alert rule operations
type AlertRuleConfig struct {
	AlertName   string            `json:"alertName" yaml:"alertName"`
	Expr        string            `json:"expr" yaml:"expr"`
	For         string            `json:"for" yaml:"for"`
	Labels      map[string]string `json:"labels" yaml:"labels"`
	Annotations map[string]string `json:"annotations" yaml:"annotations"`
}

// EmailConfiguration represents email configuration status
type EmailConfiguration struct {
	Enabled bool   `json:"enabled" yaml:"enabled"`
	SmtpURL string `json:"smtp_url" yaml:"smtp_url"`
	From    string `json:"from" yaml:"from"`
	To      string `json:"to" yaml:"to"`
}

// TelegramConfiguration represents telegram configuration status
type TelegramConfiguration struct {
	Enabled  bool   `json:"enabled" yaml:"enabled"`
	BotToken string `json:"bot_token" yaml:"bot_token"`
	ChatID   string `json:"chat_id" yaml:"chat_id"`
}
