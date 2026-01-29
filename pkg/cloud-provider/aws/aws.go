package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func LoginAWS(ctx context.Context, awsConfig *types.AWSConfig) (*types.AWSProfile, error) {
	var (
		err error
	)

	if awsConfig == nil {
		fmt.Println("You aren't logged into your AWS account.")
		return nil, fmt.Errorf("you aren't logged into your AWS account")
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

func loginAWS(accessKey, secretKey, region, formatFile string) (*types.AccountProfile, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("accessKey can't be empty")
	}

	if secretKey == "" {
		return nil, fmt.Errorf("secretKey can't be empty")
	}

	if region == "" {
		return nil, fmt.Errorf("region can't be empty")
	}

	if formatFile == "" {
		formatFile = "json"
	}

	// Prefer per-deployment credential/config files when provided.
	if credPath := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); credPath != "" {
		if err := utils.WriteAWSCredentialsFile(credPath, accessKey, secretKey); err != nil {
			return nil, err
		}
		if cfgPath := os.Getenv("AWS_CONFIG_FILE"); cfgPath != "" {
			if err := utils.WriteAWSConfigFile(cfgPath, region, formatFile); err != nil {
				return nil, err
			}
		}
		os.Setenv("AWS_REGION", region)
		os.Setenv("AWS_DEFAULT_REGION", region)
	} else {
		configureAWS("aws", "configure", "set", "aws_access_key_id", accessKey)
		configureAWS("aws", "configure", "set", "aws_secret_access_key", secretKey)
		configureAWS("aws", "configure", "set", "region", region)
		configureAWS("aws", "configure", "set", "output", formatFile)
	}

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
		return nil, fmt.Errorf("error getting tables: %s", err)
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
			return nil, fmt.Errorf("error creating terraform-lock table: %s", err)
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

	// EKS unsupported availability zones by region
	// Reference: https://docs.aws.amazon.com/eks/latest/userguide/network_reqs.html
	eksUnsupportedAZs := map[string][]string{
		"us-east-1": {"us-east-1e"}, // EKS control plane not supported
	}

	unsupportedZones := eksUnsupportedAZs[region]
	isUnsupported := func(zoneName string) bool {
		for _, unsupported := range unsupportedZones {
			if zoneName == unsupported {
				return true
			}
		}
		return false
	}

	// Extract only available zones and filter out EKS unsupported zones
	availabilityZones := make([]string, 0)
	for _, zone := range awsResponse.AvailabilityZones {
		if zone.State == "available" && !isUnsupported(zone.ZoneName) {
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

func GetAvailableRegions(accessKey string, secretKey string, region string) ([]string, error) {
	configureAWS("aws", "configure", "set", "region", region)
	configureAWS("aws", "configure", "set", "aws_access_key_id", accessKey)
	configureAWS("aws", "configure", "set", "aws_secret_access_key", secretKey)

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
	availableRegions, err := GetAvailableRegions(accessKey, secretKey, region)
	if err != nil {
		return false
	}
	return utils.IsAvailableRegion(region, availableRegions)
}
