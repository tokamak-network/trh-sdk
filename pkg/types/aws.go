package types

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type AccountProfile struct {
	UserId            string   `json:"UserId"`
	Account           string   `json:"Account"`
	Arn               string   `json:"Arn"`
	AvailabilityZones []string `json:"AvailabilityZones"`
}

type AWSConfig struct {
	SecretKey     string `json:"secret_key"`
	AccessKey     string `json:"access_key"`
	Region        string `json:"region"`
	DefaultFormat string `json:"default_format" default:"json"`
	VpcID         string `json:"vpc_id,omitempty"`
}

type AvailabilityZone struct {
	ZoneName string `json:"ZoneName"`
	State    string `json:"State"`
}

type AWSAvailabilityZoneResponse struct {
	AvailabilityZones []AvailabilityZone `json:"AvailabilityZones"`
}

type AWSTableListResponse struct {
	TableNames []string `json:"TableNames"`
}

type AWSProfile struct {
	S3Client       *s3.Client
	AccountProfile *AccountProfile
	AwsConfig      *AWSConfig
}
