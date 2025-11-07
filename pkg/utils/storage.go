package utils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// GetEFSFileSystemId extracts EFS filesystem ID from existing PV
func GetEFSFileSystemId(ctx context.Context, chainName string, awsRegion string) (string, error) {
	// First try to get EFS filesystem ID from existing op-geth PV using a more reliable method
	output, err := ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name,VOLUMEHANDLE:.spec.csi.volumeHandle", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, chainName) && strings.Contains(line, "thanos-stack-op-geth") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				volumeHandle := fields[1]
				if strings.HasPrefix(volumeHandle, "fs-") {
					return volumeHandle, nil
				}
			}
		}
	}

	// Fallback: try to get from AWS EFS directly
	output, err = ExecuteCommand(ctx, "aws", "efs", "describe-file-systems", "--query", "FileSystems[0].FileSystemId", "--output", "text", "--region", awsRegion)
	if err != nil {
		return "", fmt.Errorf("failed to get EFS filesystem ID from AWS: %w", err)
	}

	efsId := strings.TrimSpace(output)
	if efsId == "" || efsId == "None" {
		return "", fmt.Errorf("no EFS filesystem found in region %s", awsRegion)
	}

	return efsId, nil
}

// GetTimestampFromExistingPV extracts timestamp from existing monitoring PVs
func GetTimestampFromExistingPV(ctx context.Context, chainName string) (string, error) {
	output, err := ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "custom-columns=NAME:.metadata.name", "--no-headers")
	if err != nil {
		return "", fmt.Errorf("failed to get PVs: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		// Look for existing monitoring PVs (grafana or prometheus) that are Released
		if strings.Contains(line, chainName) &&
			(strings.Contains(line, "thanos-stack-grafana") || strings.Contains(line, "thanos-stack-prometheus")) {
			// Extract timestamp from PV
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	// Fallback to op-geth PV if no monitoring PVs found
	for _, line := range lines {
		if strings.Contains(line, chainName) && strings.Contains(line, "thanos-stack-op-geth") {
			parts := strings.Split(line, "-")
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	return "", fmt.Errorf("could not find existing PV to extract timestamp")
}

// GenerateStaticPVManifest generates PV manifest using StorageConfig interface
func GenerateStaticPVManifest(component string, config types.StorageConfig, size string, timestamp string) string {
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.GetChainName(), timestamp, component)
	volumeHandle := config.GetEFSFileSystemId()

	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: %s
  labels:
    app: %s
spec:
  capacity:
    storage: %s
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: efs-sc
  csi:
    driver: efs.csi.aws.com
    volumeHandle: %s
`, pvName, pvName, size, volumeHandle)
}

// ApplyPVManifest applies PV manifest using kubectl (with prefix parameter)
func ApplyPVManifest(ctx context.Context, deploymentPath string, component string, manifest string, prefix string) error {
	tempFile := filepath.Join(deploymentPath, fmt.Sprintf("%s-%s-pv.yaml", prefix, component))
	if err := os.WriteFile(tempFile, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write PV manifest: %w", err)
	}
	defer os.Remove(tempFile)

	_, err := ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
	if err != nil {
		return fmt.Errorf("failed to apply PV manifest: %w", err)
	}

	return nil
}

// GenerateStaticPVCManifest generates PVC manifest using StorageConfig interface
func GenerateStaticPVCManifest(component string, config types.StorageConfig, size string, timestamp string) string {
	pvcName := fmt.Sprintf("%s-pvc", config.GetHelmReleaseName())
	pvName := fmt.Sprintf("%s-%s-thanos-stack-%s", config.GetChainName(), timestamp, component)

	return fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: %s
  selector:
    matchLabels:
      app: %s
  storageClassName: efs-sc
  volumeMode: Filesystem
  volumeName: %s
`, pvcName, config.GetNamespace(), size, pvName, pvName)
}

// ApplyPVCManifest applies PVC manifest using kubectl (with prefix parameter)
func ApplyPVCManifest(ctx context.Context, deploymentPath string, component string, manifest string, prefix string) error {
	tempFile := filepath.Join(deploymentPath, fmt.Sprintf("%s-pvc.yaml", prefix))
	if err := os.WriteFile(tempFile, []byte(manifest), 0644); err != nil {
		return fmt.Errorf("failed to write PVC manifest: %w", err)
	}
	defer os.Remove(tempFile)

	_, err := ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile)
	if err != nil {
		return fmt.Errorf("failed to apply PVC manifest: %w", err)
	}

	return nil
}
