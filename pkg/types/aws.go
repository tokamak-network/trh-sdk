package types

type AWSLogin struct {
	SecretKey     string `json:"secret_key"`
	AccessKey     string `json:"access_key"`
	Region        string `json:"region"`
	DefaultFormat string `json:"default_format" default:"json"`
}
