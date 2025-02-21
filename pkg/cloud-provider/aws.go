package cloud_provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

	fmt.Println("Available AWS availability zones:", availabilityZones)

	profile.AvailabilityZones = availabilityZones
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
