package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tokamak-network/trh-sdk/pkg/scanner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func LoginAWS(ctx context.Context, deployConfig *types.Config) (*types.AWSProfile, error) {
	var (
		awsConfig *types.AWSConfig
		err       error
	)

	if deployConfig != nil && deployConfig.AWS != nil {
		awsConfig = deployConfig.AWS
	}

	// If AWS config is not provided, prompt the user for AWS credentials
	if awsConfig == nil {
		fmt.Println("You aren't logged into your AWS account.")
		awsConfig, err = inputAWSLogin()
		if err != nil {
			fmt.Println("Error collecting AWS credentials:", err)
			return nil, err
		}

		if awsConfig == nil {
			return nil, fmt.Errorf("AWS config is nil")
		}
	}

	// Login to AWS account
	fmt.Println("Authenticating AWS account...")
	profileAccount, err := loginAWS(awsConfig.AccessKey, awsConfig.SecretKey, awsConfig.Region, awsConfig.DefaultFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to log in into AWS: %v", err)
	}

	if profileAccount == nil {
		return nil, fmt.Errorf("failed to get AWS profile account")
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(awsConfig.Region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS configuration: %v", err)
	}

	s3Client := s3.NewFromConfig(cfg)

	return &types.AWSProfile{
		S3Client:       s3Client,
		AccountProfile: profileAccount,
		AwsConfig:      awsConfig,
	}, nil
}

func inputAWSLogin() (*types.AWSConfig, error) {
	var (
		awsAccessKeyID, awsSecretKey, awsRegion string
		err                                     error
	)
	for {
		fmt.Print("Please enter your AWS access key: ")
		awsAccessKeyID, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS access key")
			return nil, err
		}
		if awsAccessKeyID == "" {
			fmt.Println("Error: AWS access key ID cannot be empty")
			continue
		}
		if !utils.IsValidAWSAccessKey(awsAccessKeyID) {
			fmt.Println("Error: The AWS access key ID format is invalid")
			continue
		}
		break
	}

	for {
		fmt.Print("Please enter your AWS secret key: ")
		awsSecretKey, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS secret key")
			return nil, err
		}
		if awsSecretKey == "" {
			fmt.Println("Error: AWS secret key cannot be empty")
			continue
		}
		if !utils.IsValidAWSSecretKey(awsSecretKey) {
			fmt.Println("Error: The AWS secret key format is invalid")
			continue
		}
		break
	}

	for {
		fmt.Print("Please enter your AWS region (default: ap-northeast-2): ")
		awsRegion, err = scanner.ScanString()
		if err != nil {
			fmt.Println("Error while reading AWS region")
			return nil, err
		}
		if awsRegion == "" {
			awsRegion = "ap-northeast-2"
		}
		fmt.Println("Verifying region availability...")
		if !IsAvailableRegion(awsAccessKeyID, awsSecretKey, awsRegion) {
			fmt.Println("Error: The AWS region is not available. Please try again.")
			continue
		}
		break
	}

	return &types.AWSConfig{
		SecretKey:     awsSecretKey,
		Region:        awsRegion,
		AccessKey:     awsAccessKeyID,
		DefaultFormat: "json",
	}, nil
}

func loginAWS(accessKey, secretKey, region, formatFile string) (*types.AccountProfile, error) {
	configureAWS("aws", "configure", "set", "aws_access_key_id", accessKey)
	configureAWS("aws", "configure", "set", "aws_secret_access_key", secretKey)
	configureAWS("aws", "configure", "set", "region", region)
	configureAWS("aws", "configure", "set", "output", formatFile)

	cmd := exec.Command("aws", "sts", "get-caller-identity")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error fetching AWS caller identity:", err)
		return nil, err
	}

	var profile types.AccountProfile
	if err := json.Unmarshal(output, &profile); err != nil {
		return nil, err
	}

	availabilityZones, err := getAvailabilityZones(region)
	if err != nil {
		fmt.Println("Error fetching AWS availability zones:", err)
		return nil, err
	}

	profile.AvailabilityZones = availabilityZones

	// Check requirements
	// before making the thanos-stack terraform up, check the `terraform-lock` table creation first
	// https://github.com/tokamak-network/tokamak-thanos-stack/blob/main/terraform/thanos-stack/backend.tf#L7
	// Step 1: get the table list by the region
	tables, err := getTablesByRegion(region)
	if err != nil {
		return nil, fmt.Errorf("Error getting tables: %s", err)
	}

	existTerraformLockTable := false
	for _, table := range tables {
		if table == "terraform-lock" {
			existTerraformLockTable = true
		}
	}

	if !existTerraformLockTable {
		err = createDynamoDBTable(region, "terraform-lock")
		if err != nil {
			return nil, fmt.Errorf("Error creating terraform-lock table: %s", err)
		}
	}

	return &profile, nil
}

func configureAWS(command ...string) {
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("Error:", err)
	}
}

func getAvailabilityZones(region string) ([]string, error) {
	cmd := exec.Command("aws", "ec2", "describe-availability-zones", "--region", region, "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error fetching AWS availability zones:", err)
		return nil, err
	}
	var awsResponse types.AWSAvailabilityZoneResponse
	err = json.Unmarshal([]byte(output), &awsResponse)
	if err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return nil, err
	}

	// Extract and print only available zones
	availabilityZones := make([]string, 0)
	for _, zone := range awsResponse.AvailabilityZones {
		if zone.State == "available" {
			availabilityZones = append(availabilityZones, zone.ZoneName)
		}
	}

	return availabilityZones, nil
}

func getTablesByRegion(region string) ([]string, error) {
	cmd := exec.Command("aws", "dynamodb", "list-tables", "--region", region, "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error fetching the table list:", err)
		return nil, err
	}
	var awsResponse types.AWSTableListResponse
	err = json.Unmarshal(output, &awsResponse)
	if err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return nil, err
	}

	return awsResponse.TableNames, nil
}

func createDynamoDBTable(region, tableName string) error {
	cmd := exec.Command(
		"aws", "dynamodb", "create-table",
		"--table-name", tableName,
		"--attribute-definitions", "AttributeName=LockID,AttributeType=S",
		"--key-schema", "AttributeName=LockID,KeyType=HASH",
		"--billing-mode", "PAY_PER_REQUEST",
		"--region", region,
		"--output", "json",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("❌ Error creating table: %v\nDetails: %s\n", err, string(output))
		return err
	}

	fmt.Println("✅ Table created successfully:", string(output))
	return nil
}

func getAvailableRegions() ([]string, error) {
	cmd := exec.Command("aws", "ec2", "describe-regions", "--query", "Regions[].RegionName", "--output", "json")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error fetching AWS regions:", err)
		return nil, err
	}

	var availableRegions []string
	err = json.Unmarshal(output, &availableRegions)
	if err != nil {
		fmt.Printf("❌ Error parsing JSON: %v\n", err)
		return nil, err
	}

	return availableRegions, nil
}

func IsAvailableRegion(accessKey, secretKey, region string) bool {
	configureAWS("aws", "configure", "set", "aws_access_key_id", accessKey)
	configureAWS("aws", "configure", "set", "aws_secret_access_key", secretKey)
	configureAWS("aws", "configure", "set", "region", region)
	availableRegions, err := getAvailableRegions()
	if err != nil {
		return false
	}
	return utils.IsAvailableRegion(region, availableRegions)
}
