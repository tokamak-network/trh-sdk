package utils

import "regexp"

// IsValidAWSAccessKey checks if the AWS Access Key matches the correct format
func IsValidAWSAccessKey(accessKey string) bool {
	matched, _ := regexp.MatchString(`^(AKIA|ASIA)[A-Z0-9]{16}$`, accessKey)
	return matched
}

// IsValidAWSSecretKey checks if the AWS Secret Key matches the correct format
func IsValidAWSSecretKey(secretKey string) bool {
	matched, _ := regexp.MatchString(`^[A-Za-z0-9/+]{40}$`, secretKey)
	return matched
}
