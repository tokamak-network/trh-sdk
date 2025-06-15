package utils

import (
	"context"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

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

func IsValidRDSUsername(username string) bool {
	rdsUsernamePattern := `^[a-zA-Z][a-zA-Z0-9_]{0,62}$`
	matched, _ := regexp.MatchString(rdsUsernamePattern, username)
	return matched
}

func IsValidRDSPassword(password string) bool {
	rdsPasswordPattern := `^[a-zA-Z0-9@\$#%&*\(\)_\+\-!]{8,128}$`
	matched, _ := regexp.MatchString(rdsPasswordPattern, password)
	return matched
}

func BucketExists(ctx context.Context, client *s3.Client, bucketName string) bool {
	if bucketName == "" {
		return false
	}
	// Try to get bucket's HEAD (metadata)
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	return err == nil
}

func IsAvailableRegion(region string, availableRegions []string) bool {
	for _, availableRegion := range availableRegions {
		if region == availableRegion {
			return true
		}
	}
	return false
}

func SwitchKubernetesContext(ctx context.Context, namespace string, region string) error {
	eksSetup, err := ExecuteCommand(ctx, "aws", []string{
		"eks",
		"update-kubeconfig",
		"--region", region,
		"--name", namespace,
	}...)
	if err != nil {
		fmt.Println("Error configuring EKS access:", err, "details:", eksSetup)
		return fmt.Errorf("error configuring EKS access: %w, details: %s", err, eksSetup)
	}

	fmt.Println("EKS configuration updated:", eksSetup)

	return nil
}
