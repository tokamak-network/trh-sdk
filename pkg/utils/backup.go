package utils

import (
	"context"
	"fmt"
	"strings"
)

// DetectEFSId scans PVC-bound PVs in the given namespace to find an EFS CSI volume handle (fs-xxxx).
func DetectEFSId(ctx context.Context, namespace string) (string, error) {
	// Try PVC-bound PVs in namespace
	out, _ := ExecuteCommand(ctx, "kubectl", "get", "pvc", "-n", namespace, "-o", "jsonpath={range .items[*]}{.spec.volumeName}{\"\\n\"}{end}")
	pvs := strings.Fields(out)
	for _, pv := range pvs {
		handle, _ := ExecuteCommand(ctx, "kubectl", "get", "pv", pv, "-o", "jsonpath={.spec.csi.volumeHandle}")
		handle = strings.TrimSpace(handle)
		if strings.HasPrefix(handle, "fs-") {
			return handle, nil
		}
	}
	// Fallback: scan all PVs
	all, _ := ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	for _, pv := range strings.Fields(all) {
		handle, _ := ExecuteCommand(ctx, "kubectl", "get", "pv", pv, "-o", "jsonpath={.spec.csi.volumeHandle}")
		handle = strings.TrimSpace(handle)
		if strings.HasPrefix(handle, "fs-") {
			return handle, nil
		}
	}
	return "", fmt.Errorf("failed to detect EFS FileSystemId from PVs")
}

// BuildEFSArn builds the EFS ARN from region, account id and filesystem id.
func BuildEFSArn(region, accountID, efsID string) string {
	return fmt.Sprintf("arn:aws:elasticfilesystem:%s:%s:file-system/%s", region, accountID, efsID)
}

// DetectAWSAccountID returns current caller account id via STS.
func DetectAWSAccountID(ctx context.Context) (string, error) {
	out, err := ExecuteCommand(ctx, "aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// RDSIdentifierFromNamespace returns the expected RDS instance identifier for the namespace.
func RDSIdentifierFromNamespace(namespace string) string {
	return fmt.Sprintf("%s-rds", namespace)
}
