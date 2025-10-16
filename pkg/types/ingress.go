package types

type TLS struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
}

type Ingress struct {
	Enabled     bool              `yaml:"enabled" json:"enabled"`
	ClassName   string            `yaml:"className" json:"className"`
	Annotations map[string]string `yaml:"annotations" json:"annotations"`
	TLS         TLS               `yaml:"tls" json:"tls"`
}
