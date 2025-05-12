package aws

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type AccountProfile struct {
	UserId            string   `json:"UserId"`
	Account           string   `json:"Account"`
	Arn               string   `json:"Arn"`
	AvailabilityZones []string `json:"AvailabilityZones"`
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

func LoginAWS(accessKey, secretKey, region, formatFile string) (*AccountProfile, error) {
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

	var profile AccountProfile
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
	var awsResponse AWSAvailabilityZoneResponse
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
	var awsResponse AWSTableListResponse
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
