package thanos

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/types"
)

func (t *ThanosStack) loginAWS(ctx context.Context) (*aws.AccountProfile, *types.AWSConfig, error) {
	var (
		awsConfig *types.AWSConfig
		err       error
	)

	if t.deployConfig != nil {
		awsConfig = t.deployConfig.AWS
	}

	// If AWS config is not provided, prompt the user for AWS credentials
	if awsConfig == nil {
		fmt.Println("You aren't logged into your AWS account.")
		awsConfig, err = t.inputAWSLogin()
		if err != nil {
			fmt.Println("Error collecting AWS credentials:", err)
			return nil, nil, err
		}

		if awsConfig == nil {
			return nil, nil, fmt.Errorf("AWS config is nil")
		}
	}

	// Login to AWS account
	fmt.Println("Authenticating AWS account...")
	awsProfileAccount, err := aws.LoginAWS(awsConfig.AccessKey, awsConfig.SecretKey, awsConfig.Region, awsConfig.DefaultFormat)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to log in into AWS: %v", err)
	}

	if awsProfileAccount == nil {
		return nil, nil, fmt.Errorf("failed to get AWS profile account")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsConfig.Region))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	t.s3Client = s3.NewFromConfig(cfg)

	return awsProfileAccount, awsConfig, nil
}
