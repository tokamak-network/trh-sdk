package cloud_provider

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type AccountProfile struct {
	UserId  string `json:"UserId"`
	Account string `json:"Account"`
	Arn     string `json:"Arn"`
}

func LoginAWS(accessKey, secretKey, region, formatFile string) (*AccountProfile, error) {
	configureAWS("aws", "configure", "set", "aws_access_key_id", accessKey)
	configureAWS("aws", "configure", "set", "aws_secret_access_key", secretKey)
	configureAWS("aws", "configure", "set", "region", region)
	configureAWS("aws", "configure", "set", "output", formatFile)

	cmd := exec.Command("aws", "sts", "get-caller-identity")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Failed to fetch AWS caller identity:", err)
		return nil, err
	}

	var profile AccountProfile
	if err := json.Unmarshal(output, &profile); err != nil {
		return nil, err
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
