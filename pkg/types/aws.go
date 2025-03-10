package types

type AWSConfig struct {
	SecretKey     string `json:"secret_key"`
	AccessKey     string `json:"access_key"`
	Region        string `json:"region"`
	DefaultFormat string `json:"default_format" default:"json"`
	VpcID         string `json:"vpc_id"`
}
