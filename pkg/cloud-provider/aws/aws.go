package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// awsEnv returns environment variables for AWS CLI commands
// This avoids modifying ~/.aws/credentials file
func awsEnv(accessKey, secretKey, region string) []string {
	return append(os.Environ(),
		"AWS_ACCESS_KEY_ID="+accessKey,
		"AWS_SECRET_ACCESS_KEY="+secretKey,
		"AWS_DEFAULT_REGION="+region,
		"AWS_REGION="+region,
	)
}

// newAWSCommand creates an exec.Command with AWS credentials set via environment variables
func newAWSCommand(accessKey, secretKey, region string, args ...string) *exec.Cmd {
	cmd := exec.Command("aws", args...)
	cmd.Env = awsEnv(accessKey, secretKey, region)
	return cmd
}

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

	// Use static credentials provider instead of default config
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(awsConfig.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			awsConfig.AccessKey,
			awsConfig.SecretKey,
			"",
		)),
	)
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

	// Set environment variables for the current process
	// This affects subsequent AWS SDK calls but doesn't modify ~/.aws/credentials
	os.Setenv("AWS_ACCESS_KEY_ID", accessKey)
	os.Setenv("AWS_SECRET_ACCESS_KEY", secretKey)
	os.Setenv("AWS_REGION", region)
	os.Setenv("AWS_DEFAULT_REGION", region)

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
	}

	// Use environment variables for AWS CLI commands instead of modifying ~/.aws/credentials
	cmd := newAWSCommand(accessKey, secretKey, region, "sts", "get-caller-identity")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error fetching AWS caller identity:", err)
		return nil, err
	}

	var profile types.AccountProfile
	if err := json.Unmarshal(output, &profile); err != nil {
		return nil, err
	}

	availabilityZones, err := getAvailabilityZonesWithCreds(accessKey, secretKey, region)
	if err != nil {
		fmt.Println("Error fetching AWS availability zones:", err)
		return nil, err
	}

	profile.AvailabilityZones = availabilityZones

	// Check requirements
	// before making the thanos-stack terraform up, check the `terraform-lock` table creation first
	// https://github.com/tokamak-network/tokamak-thanos-stack/blob/main/terraform/thanos-stack/backend.tf#L7
	// Step 1: get the table list by the region
	tables, err := getTablesByRegionWithCreds(accessKey, secretKey, region)
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
		err = createDynamoDBTableWithCreds(accessKey, secretKey, region, "terraform-lock")
		if err != nil {
			return nil, fmt.Errorf("error creating terraform-lock table: %s", err)
		}
	}

	return &profile, nil
}

// getAvailabilityZonesWithCreds fetches availability zones using explicit credentials
func getAvailabilityZonesWithCreds(accessKey, secretKey, region string) ([]string, error) {
	cmd := newAWSCommand(accessKey, secretKey, region,
		"ec2", "describe-availability-zones", "--region", region, "--output", "json")
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

// getTablesByRegionWithCreds fetches DynamoDB tables using explicit credentials
func getTablesByRegionWithCreds(accessKey, secretKey, region string) ([]string, error) {
	cmd := newAWSCommand(accessKey, secretKey, region,
		"dynamodb", "list-tables", "--region", region, "--output", "json")
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

// createDynamoDBTableWithCreds creates a DynamoDB table using explicit credentials
func createDynamoDBTableWithCreds(accessKey, secretKey, region, tableName string) error {
	cmd := newAWSCommand(accessKey, secretKey, region,
		"dynamodb", "create-table",
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

// GetAvailableRegions fetches available AWS regions using explicit credentials
// This function uses environment variables instead of modifying ~/.aws/credentials
func GetAvailableRegions(accessKey string, secretKey string, region string) ([]string, error) {
	cmd := newAWSCommand(accessKey, secretKey, region,
		"ec2", "describe-regions", "--query", "Regions[].RegionName", "--output", "json")
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
