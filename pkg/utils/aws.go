package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

// SetAWSConfigFile points AWS CLI config to a per-deployment path to avoid touching ~/.aws/config.
func SetAWSConfigFile(basePath string) (string, error) {
	if basePath == "" {
		return "", fmt.Errorf("base path is required")
	}
	if existing := os.Getenv("AWS_CONFIG_FILE"); existing != "" {
		return existing, nil
	}
	cfgDir := filepath.Join(basePath, ".aws")
	if err := os.MkdirAll(cfgDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create aws config dir: %w", err)
	}
	cfgPath := filepath.Join(cfgDir, "config")
	if err := os.Setenv("AWS_CONFIG_FILE", cfgPath); err != nil {
		return "", fmt.Errorf("failed to set AWS_CONFIG_FILE: %w", err)
	}
	return cfgPath, nil
}

// SetAWSCredentialsFile points AWS shared credentials to a per-deployment path to avoid touching ~/.aws/credentials.
func SetAWSCredentialsFile(basePath string) (string, error) {
	if basePath == "" {
		return "", fmt.Errorf("base path is required")
	}
	if existing := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); existing != "" {
		return existing, nil
	}
	credDir := filepath.Join(basePath, ".aws")
	if err := os.MkdirAll(credDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create aws credentials dir: %w", err)
	}
	credPath := filepath.Join(credDir, "credentials")
	if err := os.Setenv("AWS_SHARED_CREDENTIALS_FILE", credPath); err != nil {
		return "", fmt.Errorf("failed to set AWS_SHARED_CREDENTIALS_FILE: %w", err)
	}
	return credPath, nil
}

// SetKubeconfigFile points kubeconfig to a per-deployment path to avoid touching ~/.kube/config.
func SetKubeconfigFile(basePath string) (string, error) {
	if basePath == "" {
		return "", fmt.Errorf("base path is required")
	}
	if existing := os.Getenv("KUBECONFIG"); existing != "" {
		return existing, nil
	}
	kubeDir := filepath.Join(basePath, ".kube")
	if err := os.MkdirAll(kubeDir, 0o700); err != nil {
		return "", fmt.Errorf("failed to create kubeconfig dir: %w", err)
	}
	kubePath := filepath.Join(kubeDir, "config")
	if err := os.Setenv("KUBECONFIG", kubePath); err != nil {
		return "", fmt.Errorf("failed to set KUBECONFIG: %w", err)
	}
	return kubePath, nil
}

// WriteAWSCredentialsFile writes a minimal default profile credentials file.
func WriteAWSCredentialsFile(path string, accessKey string, secretKey string) error {
	if path == "" {
		return fmt.Errorf("credentials path is required")
	}
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("access key and secret key are required")
	}
	content := fmt.Sprintf("[default]\naws_access_key_id = %s\naws_secret_access_key = %s\n", accessKey, secretKey)
	return os.WriteFile(path, []byte(content), 0o600)
}

// WriteAWSConfigFile writes a minimal default profile config file.
func WriteAWSConfigFile(path string, region string, output string) error {
	if path == "" {
		return fmt.Errorf("config path is required")
	}
	if region == "" {
		return fmt.Errorf("region is required")
	}
	if output == "" {
		output = "json"
	}
	content := fmt.Sprintf("[default]\nregion = %s\noutput = %s\n", region, output)
	return os.WriteFile(path, []byte(content), 0o644)
}

func writeAWSConfigFile(path string, region string) error {
	if path == "" {
		return fmt.Errorf("config path is required")
	}
	content := fmt.Sprintf("[default]\nregion = %s\noutput = json\n", region)
	return os.WriteFile(path, []byte(content), 0o644)
}

// SwitchAWSRegion sets the AWS CLI default region configuration
func SwitchAWSRegion(ctx context.Context, region string) error {
	os.Setenv("AWS_REGION", region)
	os.Setenv("AWS_DEFAULT_REGION", region)

	if cfgPath := os.Getenv("AWS_CONFIG_FILE"); cfgPath != "" {
		if err := writeAWSConfigFile(cfgPath, region); err != nil {
			fmt.Println("Error setting AWS region:", err, "details:", cfgPath)
			return fmt.Errorf("error setting AWS region: %w, details: %s", err, cfgPath)
		}
		fmt.Println("AWS region updated to:", region)
		return nil
	}

	regionSetup, err := ExecuteCommand(ctx, "aws", []string{
		"configure",
		"set",
		"region", region,
	}...)
	if err != nil {
		fmt.Println("Error setting AWS region:", err, "details:", regionSetup)
		return fmt.Errorf("error setting AWS region: %w, details: %s", err, regionSetup)
	}

	fmt.Println("AWS region updated to:", region)
	return nil
}

func SwitchKubernetesContext(ctx context.Context, namespace string, region string) error {
	// First, set the AWS region configuration
	if err := SwitchAWSRegion(ctx, region); err != nil {
		return fmt.Errorf("failed to set AWS region: %w", err)
	}

	// Then update the EKS kubeconfig
	args := []string{
		"eks",
		"update-kubeconfig",
		"--region", region,
		"--name", namespace,
	}
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		args = append(args, "--kubeconfig", kubeconfig)
	}
	eksSetup, err := ExecuteCommand(ctx, "aws", args...)
	if err != nil {
		fmt.Println("Error configuring EKS access:", err, "details:", eksSetup)
		return fmt.Errorf("error configuring EKS access: %w, details: %s", err, eksSetup)
	}

	fmt.Println("EKS configuration updated:", eksSetup)

	return nil
}
