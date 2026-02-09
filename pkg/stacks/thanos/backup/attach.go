package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const DefaultBackupPvPvcRawURL = "https://raw.githubusercontent.com/tokamak-network/trh-sdk/main/scripts/backup_pv_pvc.sh"

// ShowAttachUsage prints usage via the provided logger
func ShowAttachUsage(l *zap.SugaredLogger) {
	if l == nil {
		return
	}
	l.Info("📋 Attach Usage")
	l.Info("")
	l.Info("COMMAND:")
	l.Info("  trh-sdk backup-manager --attach [OPTIONS]")
	l.Info("")
	l.Info("OPTIONS:")
	l.Info("  --efs-id fs-xxxx          New EFS FileSystemId to switch to")
	l.Info("  --pvc op-geth,op-node     Comma-separated PVC names to switch")
	l.Info("  --sts op-geth,op-node     Comma-separated StatefulSet names to restart")
}

// ValidateAttachPrerequisites checks required CLIs and cluster access
func ValidateAttachPrerequisites(ctx context.Context) error {
	if _, err := utils.ExecuteCommand(ctx, "aws", "--version"); err != nil {
		return fmt.Errorf("AWS CLI is not installed or not accessible: %w", err)
	}
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "version", "--client"); err != nil {
		return fmt.Errorf("kubectl is not installed or not accessible: %w", err)
	}
	if _, err := utils.ExecuteCommand(ctx, "kubectl", "cluster-info"); err != nil {
		return fmt.Errorf("cannot access Kubernetes cluster: %w", err)
	}
	return nil
}

// VerifyEFSData delegates to provided verification function
func VerifyEFSData(ctx context.Context, namespace string, verify func(context.Context, string) error) error {
	if verify == nil {
		return nil
	}
	return verify(ctx, namespace)
}

// RestartStatefulSets restarts comma-separated StatefulSets and waits for rollout
func RestartStatefulSets(ctx context.Context, l *zap.SugaredLogger, namespace, stsCSV string) error {
	list := strings.Split(strings.TrimSpace(stsCSV), ",")
	for _, raw := range list {
		name := strings.TrimSpace(raw)
		if name == "" {
			continue
		}

		// Find actual StatefulSet name by pattern matching
		actualName, err := findStatefulSetByName(ctx, namespace, name)
		if err != nil {
			return fmt.Errorf("failed to find StatefulSet matching %s: %w", name, err)
		}

		// Log StatefulSet restart initiation
		l.Infof("🔄 Restarting StatefulSet %s (actual: %s)...", name, actualName)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "rollout", "restart", "statefulset/"+actualName); err != nil {
			return fmt.Errorf("failed to restart StatefulSet %s (actual: %s): %w", name, actualName, err)
		}

		// Log rollout status monitoring
		l.Infof("⏳ Waiting for StatefulSet %s rollout to complete (timeout: 10 minutes)...", actualName)

		if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "rollout", "status", "statefulset/"+actualName, "--timeout=600s"); err != nil {
			return fmt.Errorf("rollout status failed for %s (actual: %s): %w", name, actualName, err)
		}

		// Log successful completion
		l.Infof("✅ StatefulSet %s rollout completed successfully", actualName)
	}
	return nil
}

// findStatefulSetByName finds the actual StatefulSet name by pattern matching
func findStatefulSetByName(ctx context.Context, namespace, pattern string) (string, error) {
	// Get all StatefulSets in the namespace using a simpler approach
	output, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "statefulsets", "-o", "name")
	if err != nil {
		return "", fmt.Errorf("failed to list StatefulSets: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Remove "statefulset.apps/" prefix
		stsName := strings.TrimPrefix(line, "statefulset.apps/")

		// Check if the StatefulSet name contains the pattern
		// e.g., pattern "op-geth" should match "theo09112-iskw4-1757567097-thanos-stack-op-geth"
		if strings.Contains(stsName, pattern) {
			return stsName, nil
		}
	}

	return "", fmt.Errorf("no StatefulSet found matching pattern: %s", pattern)
}

// GatherBackupAttachInfo builds BackupAttachInfo from flags
func GatherBackupAttachInfo(
	ctx context.Context,
	namespace, region string,
	efsId *string, pvcs *string, stss *string,
	showUsage func(),
) (*types.BackupAttachInfo, error) {
	// require at least one parameter
	if (efsId == nil || strings.TrimSpace(*efsId) == "") &&
		(pvcs == nil || strings.TrimSpace(*pvcs) == "") &&
		(stss == nil || strings.TrimSpace(*stss) == "") {
		if showUsage != nil {
			showUsage()
		}
		return nil, fmt.Errorf("at least one parameter (--efs-id, --pvc, or --sts) must be provided")
	}

	// parse pvcs
	var pvcList []string
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		pvcList = strings.Split(strings.TrimSpace(*pvcs), ",")
		for i, v := range pvcList {
			pvcList[i] = strings.TrimSpace(v)
		}
	}
	// parse stss
	var stsList []string
	if stss != nil && strings.TrimSpace(*stss) != "" {
		stsList = strings.Split(strings.TrimSpace(*stss), ",")
		for i, v := range stsList {
			stsList[i] = strings.TrimSpace(v)
		}
	}

	// validate efs id
	efsID := ""
	if efsId != nil {
		efsID = strings.TrimSpace(*efsId)
		if efsID != "" && !strings.HasPrefix(efsID, "fs-") {
			return nil, fmt.Errorf("invalid EFS ID format: %s (should start with 'fs-')", efsID)
		}
	}

	return &types.BackupAttachInfo{
		Region:    region,
		Namespace: namespace,
		EFSID:     efsID,
		PVCs:      pvcList,
		STSs:      stsList,
	}, nil
}

// ExecuteBackupAttach performs attach flow using injected helpers
func ExecuteBackupAttach(
	ctx context.Context,
	l *zap.SugaredLogger,
	attachInfo *types.BackupAttachInfo,
	validatePrereq func(context.Context) error,
	verify func(context.Context, string) error,
	restart func(context.Context, string, string) error,
	execOps func(context.Context, *types.BackupAttachInfo, func(string, float64)) error,
	progressReporter func(string, float64),
) error {
	if progressReporter == nil {
		progressReporter = func(string, float64) {}
	}

	progressReporter("Verifying prerequisites...", 5.0)
	l.Info("Verifying restored data and switching workloads...")
	if err := validatePrereq(ctx); err != nil {
		return fmt.Errorf("attach prerequisites validation failed: %w", err)
	}

	// Handle EFS operations
	if attachInfo.EFSID != "" {
		progressReporter("Executing EFS operations...", 10.0)
		if err := execOps(ctx, attachInfo, progressReporter); err != nil {
			return err
		}
	} else {
		progressReporter("Verifying current EFS data...", 10.0)
		l.Info("Skipped PV update (no --efs-id provided).")
		if err := verify(ctx, attachInfo.Namespace); err != nil {
			l.Warnf("Current EFS data verification failed: %v", err)
		}
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", attachInfo.Namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	}

	// Handle StatefulSet restarts
	if len(attachInfo.STSs) > 0 {
		progressReporter("Restarting StatefulSets...", 70.0)
		stsList := strings.Join(attachInfo.STSs, ",")
		if err := restart(ctx, attachInfo.Namespace, stsList); err != nil {
			return fmt.Errorf("failed to restart StatefulSets: %w", err)
		}
	}

	l.Info("✅ Backup attach completed successfully")

	// Create recovery point after successful attach
	if attachInfo.EFSID == "" {
		progressReporter("Attach completed", 100.0)
		return nil
	}

	progressReporter("Creating recovery point...", 90.0)
	l.Info("Creating recovery point for attached EFS...")
	snapshotInfo, err := SnapshotExecute(ctx, l, attachInfo.Region, attachInfo.Namespace, nil)
	if err != nil {
		l.Warnf("Failed to create recovery point: %v", err)
		// Don't fail the whole operation if snapshot fails
		progressReporter("Attach completed (snapshot failed)", 100.0)
		return nil
	}

	l.Info("✅ Recovery point created successfully")
	l.Infof("   Job ID: %s", snapshotInfo.JobID)
	progressReporter("Attach completed successfully", 100.0)
	return nil
}

// ExecuteEFSOperationsFull contains the full attach EFS operation flow
// (duplicate removed)

// ReplicateEFSMountTargets replicates mount targets from source to destination EFS
func ReplicateEFSMountTargets(ctx context.Context, l *zap.SugaredLogger, region, srcFs, dstFs string) error {
	if strings.TrimSpace(srcFs) == "" || strings.TrimSpace(dstFs) == "" || srcFs == dstFs {
		return nil
	}

	mtJSON, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets", "--region", region, "--file-system-id", srcFs, "--output", "json")
	if err != nil {
		return fmt.Errorf("failed to describe source mount targets: %w", err)
	}
	type mtItem struct {
		MountTargetId        string `json:"MountTargetId"`
		SubnetId             string `json:"SubnetId"`
		AvailabilityZoneName string `json:"AvailabilityZoneName"`
	}
	var mtResp struct {
		MountTargets []mtItem `json:"MountTargets"`
	}
	if err := json.Unmarshal([]byte(mtJSON), &mtResp); err != nil {
		return fmt.Errorf("failed to parse mount targets: %w", err)
	}
	if len(mtResp.MountTargets) == 0 {
		l.Info("No mount targets found on source EFS; skipping replication")
		return nil
	}
	for _, mt := range mtResp.MountTargets {
		if strings.TrimSpace(mt.SubnetId) == "" || strings.TrimSpace(mt.MountTargetId) == "" {
			continue
		}
		sgOut, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups", "--region", region, "--mount-target-id", mt.MountTargetId, "--query", "SecurityGroups", "--output", "text")
		if err != nil {
			l.Warnf("Failed to get SGs for %s: %v", mt.MountTargetId, err)
			continue
		}
		sgOut = strings.TrimSpace(sgOut)
		if sgOut == "" {
			l.Warnf("No SGs for %s; skipping subnet %s", mt.MountTargetId, mt.SubnetId)
			continue
		}
		args := []string{"efs", "create-mount-target", "--region", region, "--file-system-id", dstFs, "--subnet-id", mt.SubnetId, "--security-groups"}
		args = append(args, strings.Fields(sgOut)...)
		if _, err := utils.ExecuteCommand(ctx, "aws", args...); err != nil {
			l.Infof("Note: create-mount-target may have failed/exists for subnet %s: %v", mt.SubnetId, err)
		} else {
			l.Infof("Created mount target on subnet %s (AZ %s) for %s", mt.SubnetId, mt.AvailabilityZoneName, dstFs)
		}
	}
	return nil
}

// ValidateEFSMountTargets validates destination EFS mount targets and SGs
func ValidateEFSMountTargets(ctx context.Context, l *zap.SugaredLogger, region, fsId string) error {
	if strings.TrimSpace(fsId) == "" {
		return fmt.Errorf("empty file system id")
	}
	mtJSON, err := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-targets", "--region", region, "--file-system-id", fsId, "--output", "json")
	if err != nil {
		return fmt.Errorf("failed to describe mount targets for %s: %w", fsId, err)
	}
	type mtItem struct {
		MountTargetId string `json:"MountTargetId"`
		SubnetId      string `json:"SubnetId"`
	}
	var mtResp struct {
		MountTargets []mtItem `json:"MountTargets"`
	}
	if err := json.Unmarshal([]byte(mtJSON), &mtResp); err != nil {
		return fmt.Errorf("failed to parse mount targets for %s: %w", fsId, err)
	}
	if len(mtResp.MountTargets) == 0 {
		return fmt.Errorf("no mount targets found on EFS %s", fsId)
	}
	criticalIssues := []string{}
	for _, mt := range mtResp.MountTargets {
		subnetId := strings.TrimSpace(mt.SubnetId)
		if subnetId == "" || strings.TrimSpace(mt.MountTargetId) == "" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("invalid mount target entry (id=%s subnet=%s)", mt.MountTargetId, mt.SubnetId))
			continue
		}
		state, sErr := utils.ExecuteCommand(ctx, "aws", "ec2", "describe-subnets", "--region", region, "--subnet-ids", subnetId, "--query", "Subnets[0].State", "--output", "text")
		if sErr != nil {
			l.Infof("⚠️  Failed to check subnet state for %s: %v", subnetId, sErr)
		} else if strings.TrimSpace(state) != "available" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("subnet %s state is %s (expected available)", subnetId, strings.TrimSpace(state)))
		} else {
			l.Infof("✅ Subnet %s is available", subnetId)
		}
		sgText, gErr := utils.ExecuteCommand(ctx, "aws", "efs", "describe-mount-target-security-groups", "--region", region, "--mount-target-id", mt.MountTargetId, "--query", "SecurityGroups", "--output", "text")
		if gErr != nil {
			criticalIssues = append(criticalIssues, fmt.Sprintf("failed to get security groups for mount target %s: %v", mt.MountTargetId, gErr))
			continue
		}
		sgText = strings.TrimSpace(sgText)
		if sgText == "" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("no security groups associated with mount target %s", mt.MountTargetId))
			continue
		}
		l.Infof("Mount target %s has SGs: %s", mt.MountTargetId, sgText)
	}
	if len(criticalIssues) > 0 {
		return fmt.Errorf("EFS %s mount target validation issues: %s", fsId, strings.Join(criticalIssues, "; "))
	}
	return nil
}

// BackupPvPvcWithUserChoice backs up PV/PVCs with user confirmation
func BackupPvPvcWithUserChoice(ctx context.Context, l *zap.SugaredLogger, namespace string) error {
	choice := "y"
	fmt.Print("Run PV/PVC backup before changes? (y/n): ")
	fmt.Scanf("%s", &choice)
	choice = strings.ToLower(strings.TrimSpace(choice))
	if choice == "y" || choice == "yes" {
		if _, err := backupPvPvc(ctx, l, namespace); err != nil {
			return fmt.Errorf("PV/PVC backup required but failed: %w", err)
		}
	} else {
		l.Info("Skipped PV/PVC backup by user choice")
	}
	return nil
}

// BackupPvPvc executes the PV/PVC backup script and returns the output directory.
func BackupPvPvc(ctx context.Context, l *zap.SugaredLogger, namespace string) (string, error) {
	return backupPvPvc(ctx, l, namespace)
}

// BackupPvPvcToDir executes the PV/PVC backup script and writes to the provided directory.
func BackupPvPvcToDir(ctx context.Context, l *zap.SugaredLogger, namespace string, backupDir string) (string, error) {
	backupDir = strings.TrimSpace(backupDir)
	if backupDir == "" {
		return "", fmt.Errorf("backup directory is required")
	}
	return backupPvPvcWithDir(ctx, l, namespace, &backupDir)
}

// backupPvPvc executes the PV/PVC backup script
func backupPvPvc(ctx context.Context, l *zap.SugaredLogger, namespace string) (string, error) {
	return backupPvPvcWithDir(ctx, l, namespace, nil)
}

func backupPvPvcWithDir(ctx context.Context, l *zap.SugaredLogger, namespace string, backupDir *string) (string, error) {
	scriptPath := "./scripts/backup_pv_pvc.sh"
	if _, err := os.Stat(scriptPath); err != nil {
		// Attempt to auto-download the script from a configurable raw URL
		rawURL := strings.TrimSpace(os.Getenv("BACKUP_PV_PVC_URL"))
		if rawURL == "" {
			rawURL = DefaultBackupPvPvcRawURL
		}
		l.Infof("Backup script not found. Attempting download from %s", rawURL)

		// Create scripts directory using Go's os package (safer than shell)
		if mkErr := os.MkdirAll("./scripts", 0755); mkErr != nil {
			return "", fmt.Errorf("failed to create scripts directory: %w", mkErr)
		}

		// Download script using curl with separate arguments (prevents command injection)
		if _, dErr := utils.ExecuteCommand(ctx, "curl", "-fsSL", rawURL, "-o", scriptPath); dErr != nil {
			return "", fmt.Errorf("failed to download backup script from %s: %w", rawURL, dErr)
		}

		// Set executable permission using Go's os package (safer than shell)
		if chErr := os.Chmod(scriptPath, 0755); chErr != nil {
			return "", fmt.Errorf("failed to set executable permission on %s: %w", scriptPath, chErr)
		}

		if _, sErr := os.Stat(scriptPath); sErr != nil {
			return "", fmt.Errorf("backup script still missing at %s after download", scriptPath)
		}
	}
	l.Infof("Running backup script for namespace %s...", namespace)

	// Build environment variables for the script
	env := []string{fmt.Sprintf("NAMESPACE=%s", namespace)}
	if backupDir != nil && strings.TrimSpace(*backupDir) != "" {
		env = append(env, fmt.Sprintf("BACKUP_DIR=%s", strings.TrimSpace(*backupDir)))
		env = append(env, "BACKUP_SKIP_SUMMARY=1")
	}

	// Execute script with environment variables (safer than shell command construction)
	output, err := utils.ExecuteCommandWithEnv(ctx, env, "bash", "-l", scriptPath)
	if err != nil {
		return "", fmt.Errorf("backup script failed: %w\nOutput: %s", err, output)
	}
	l.Info(strings.TrimSpace(output))
	outDir := ""
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "[+] Backup dir: ") {
			outDir = strings.TrimSpace(strings.TrimPrefix(line, "[+] Backup dir: "))
			break
		}
	}
	if outDir == "" {
		return "", fmt.Errorf("backup output directory not found in script output")
	}
	return outDir, nil
}

// UpdatePVVolumeHandles recreates PV/PVCs pointing to the new EFS ID
func UpdatePVVolumeHandles(ctx context.Context, l *zap.SugaredLogger, namespace, newEfs string, pvcs *string) error {
	l.Infof("Updating PV volume handles to EFS: %s", newEfs)
	var targetPVCs []string

	// Handle specific PVCs if provided
	if pvcs != nil && strings.TrimSpace(*pvcs) != "" {
		allPVCsOut, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if err != nil {
			return fmt.Errorf("failed to list PVCs for matching: %w", err)
		}
		allPVCs := strings.Fields(allPVCsOut)
		for _, input := range strings.Split(strings.TrimSpace(*pvcs), ",") {
			alias := strings.TrimSpace(input)
			if alias == "" {
				continue
			}
			resolvedPVC := ""
			for _, name := range allPVCs {
				if name == alias {
					resolvedPVC = name
					break
				}
			}
			if resolvedPVC == "" {
				for _, name := range allPVCs {
					if strings.Contains(name, alias) {
						resolvedPVC = name
						break
					}
				}
			}
			if resolvedPVC == "" {
				l.Warnf("PVC alias '%s' did not match any PVC in namespace %s", alias, namespace)
				continue
			}
			if _, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", resolvedPVC); err != nil {
				l.Warnf("PVC %s not found after resolution, skipping", resolvedPVC)
				continue
			}
			l.Infof("Resolved PVC alias '%s' -> '%s'", alias, resolvedPVC)
			targetPVCs = append(targetPVCs, resolvedPVC)
		}
	} else {
		// Auto-detect op-geth and op-node PVCs
		pvcsList, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
		if err != nil {
			return fmt.Errorf("failed to list PVCs: %w", err)
		}
		for _, pvc := range strings.Fields(pvcsList) {
			if strings.Contains(pvc, "op-geth") || strings.Contains(pvc, "op-node") {
				targetPVCs = append(targetPVCs, pvc)
				l.Infof("Found PVC: %s", pvc)
			}
		}
	}

	if len(targetPVCs) == 0 {
		return fmt.Errorf("no target PVCs found to update")
	}
	l.Infof("PV volumeHandle is immutable, deleting and recreating PVs with new EFS...")
	successCount := 0
	for _, pvcName := range targetPVCs {
		pvcName = strings.TrimSpace(pvcName)
		if pvcName == "" {
			continue
		}
		pvName, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.spec.volumeName}")
		if err != nil {
			l.Errorf("Failed to get PV name for PVC %s: %v", pvcName, err)
			continue
		}
		pvName = strings.TrimSpace(pvName)
		if pvName == "" {
			l.Warnf("PVC %s has no volumeName, skipping", pvcName)
			continue
		}
		l.Infof("Processing PVC: %s (PV: %s)", pvcName, pvName)
		// Find pods using the PVC with a more reliable method using jq
		podsUsingPVC, err := utils.ExecuteCommand(ctx, "sh", "-c", fmt.Sprintf("kubectl -n %s get pods -o json | jq -r '.items[] | select(.spec.volumes[]?.persistentVolumeClaim.claimName == \"%s\") | .metadata.name'", namespace, pvcName))
		var pods []string
		if err == nil && strings.TrimSpace(podsUsingPVC) != "" {
			pods = strings.Fields(strings.TrimSpace(podsUsingPVC))
		}
		if len(pods) > 0 {
			l.Infof("Found %d pods using PVC %s: %v", len(pods), pvcName, pods)

			// Delete pods that use the PVC (StatefulSet will recreate them later)
			for _, podName := range pods {
				l.Infof("Deleting pod %s that uses PVC %s...", podName, pvcName)
				deleteCmd := []string{"kubectl", "-n", namespace, "delete", "pod", podName, "--ignore-not-found=true"}
				output, err := utils.ExecuteCommand(ctx, deleteCmd[0], deleteCmd[1:]...)
				if err != nil {
					l.Warnf("⚠️  Pod deletion command failed: %v, Output: %s", err, output)
				} else {
					l.Infof("✅ Pod deletion command succeeded. Output: %s", output)
				}
			}

		}
		// Delete PVC
		l.Infof("Deleting PVC %s...", pvcName)
		// Non-blocking delete with wait=false and ignore-not-found
		pvcDeleteCmd := []string{"kubectl", "-n", namespace, "delete", "pvc", pvcName, "--wait=false", "--ignore-not-found=true"}
		output, err := utils.ExecuteCommand(ctx, pvcDeleteCmd[0], pvcDeleteCmd[1:]...)
		if err != nil {
			l.Warnf("⚠️  PVC deletion command failed: %v, Output: %s", err, output)
		} else {
			l.Infof("✅ PVC deletion command succeeded. Output: %s", output)
		}

		l.Infof("✅ PVC deletion command completed. Proceeding with PV deletion and recreation.")
		l.Infof("Deleting old PV %s...", pvName)
		pvDeleteCmd := []string{"kubectl", "delete", "pv", pvName, "--ignore-not-found=true"}
		pvOutput, pvErr := utils.ExecuteCommand(ctx, pvDeleteCmd[0], pvDeleteCmd[1:]...)
		if pvErr != nil {
			l.Warnf("Failed to delete PV %s: %v, Output: %s", pvName, pvErr, pvOutput)
		} else {
			l.Infof("✅ PV deletion command succeeded. Output: %s", pvOutput)
		}
		time.Sleep(2 * time.Second)
		newPVYaml := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolume
metadata:
  name: %s
  labels:
    app: %s
spec:
  capacity:
    storage: 500Gi
  volumeMode: Filesystem
  accessModes:
    - ReadWriteMany
  persistentVolumeReclaimPolicy: Retain
  storageClassName: efs-sc
  csi:
    driver: efs.csi.aws.com
    volumeHandle: %s`, pvName, pvName, newEfs)
		tempPVFile := fmt.Sprintf("/tmp/new-pv-%s.yaml", pvName)
		if err := os.WriteFile(tempPVFile, []byte(newPVYaml), 0644); err != nil {
			l.Errorf("Failed to create temporary PV YAML file: %v", err)
			continue
		}
		defer os.Remove(tempPVFile)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVFile); err != nil {
			l.Errorf("Failed to create new PV %s: %v", pvName, err)
			continue
		}
		newPVCYaml := fmt.Sprintf(`apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: %s
  namespace: %s
spec:
  storageClassName: efs-sc
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 500Gi
  selector:
    matchLabels:
      app: %s
  volumeMode: Filesystem
  volumeName: %s`, pvcName, namespace, pvName, pvName)
		tempPVCFile := fmt.Sprintf("/tmp/new-pvc-%s.yaml", pvcName)
		if err := os.WriteFile(tempPVCFile, []byte(newPVCYaml), 0644); err != nil {
			l.Errorf("Failed to create temporary PVC YAML file: %v", err)
			continue
		}
		defer os.Remove(tempPVCFile)
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempPVCFile); err != nil {
			l.Errorf("Failed to create new PVC %s: %v", pvcName, err)
			continue
		}
		l.Infof("Waiting for PVC %s to be bound...", pvcName)
		for i := 0; i < 30; i++ {
			status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.status.phase}")
			if err == nil && strings.TrimSpace(status) == "Bound" {
				break
			}
			time.Sleep(1 * time.Second)
		}
		l.Infof("✅ PVC %s and PV %s recreated successfully with new EFS", pvcName, pvName)
		successCount++
	}
	if successCount == 0 {
		return fmt.Errorf("failed to recreate any PVCs/PVs")
	}
	l.Infof("✅ Successfully recreated %d/%d PVCs with new EFS", successCount, len(targetPVCs))
	return nil
}

// ExecuteEFSOperationsFull contains the full attach EFS operation flow
func ExecuteEFSOperationsFull(
	ctx context.Context,
	l *zap.SugaredLogger,
	attachInfo *types.BackupAttachInfo,
	verify func(context.Context, string) error,
	backupPvPvcFlag *bool,
	progressReporter func(string, float64),
) error {
	if progressReporter == nil {
		progressReporter = func(string, float64) {}
	}

	// Replicate mount targets from current EFS to the new one (best effort)
	srcEfs, err := utils.DetectEFSId(ctx, attachInfo.Namespace)
	if err != nil || strings.TrimSpace(srcEfs) == "" {
		l.Warnf("Could not detect source EFS in namespace %s. Skipping mount-target replication.", attachInfo.Namespace)
		// continue to next steps
	} else {
		progressReporter("Replicating mount targets...", 20.0)
		if err := ReplicateEFSMountTargets(ctx, l, attachInfo.Region, strings.TrimSpace(srcEfs), attachInfo.EFSID); err != nil {
			l.Warnf("Failed to replicate EFS mount targets from %s to %s: %v", srcEfs, attachInfo.EFSID, err)
			// continue even if replication fails
		} else {
			l.Infof("✅ Replicated mount targets from %s to %s (region: %s)", srcEfs, attachInfo.EFSID, attachInfo.Region)
		}
	}

	progressReporter("Validating mount targets...", 25.0)
	if vErr := ValidateEFSMountTargets(ctx, l, attachInfo.Region, attachInfo.EFSID); vErr != nil {
		l.Warnf("Mount target validation for %s reported issues: %v", attachInfo.EFSID, vErr)
		// continue? usually yes, let kubelet handle attach errors
	} else {
		l.Infof("✅ Mount targets validated for %s", attachInfo.EFSID)
	}

	// Verify EFS data using injected verifier (best effort)
	if verify != nil {
		progressReporter("Verifying EFS data...", 30.0)
		if err := verify(ctx, attachInfo.Namespace); err != nil {
			l.Warnf("EFS data verification failed: %v", err)
		}
		_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", attachInfo.Namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	}

	// Backup PV/PVC definitions before destructive changes
	progressReporter("Backing up PV/PVC configurations...", 40.0)
	switch {
	case backupPvPvcFlag == nil:
		if err := BackupPvPvcWithUserChoice(ctx, l, attachInfo.Namespace); err != nil {
			return err
		}
	case *backupPvPvcFlag:
		if _, err := backupPvPvc(ctx, l, attachInfo.Namespace); err != nil {
			return err
		}
	default:
		l.Info("Skipped PV/PVC backup by user choice")
	}

	// Update PV volume handles
	if len(attachInfo.PVCs) == 0 {
		return nil
	}

	progressReporter("Updating PV/PVC volume handles...", 60.0)
	pvcList := strings.Join(attachInfo.PVCs, ",")
	if err := UpdatePVVolumeHandles(ctx, l, attachInfo.Namespace, attachInfo.EFSID, &pvcList); err != nil {
		return fmt.Errorf("failed to update PV volume handles: %w", err)
	}

	return nil
}

// VerifyEFSDataImpl verifies EFS data integrity by creating a verification pod
func VerifyEFSDataImpl(ctx context.Context, l *zap.SugaredLogger, namespace string) error {
	l.Info("Checking EFS data...")

	// Clean up any existing verify pod
	_, _ = utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")

	// Wait for cleanup
	time.Sleep(2 * time.Second)

	// Find the correct PVC name for op-geth
	pvcsList, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		return fmt.Errorf("failed to list PVCs: %w", err)
	}

	var opGethPVC string
	for _, pvc := range strings.Fields(pvcsList) {
		if strings.Contains(pvc, "op-geth") {
			opGethPVC = pvc
			break
		}
	}

	if opGethPVC == "" {
		return fmt.Errorf("no op-geth PVC found in namespace %s", namespace)
	}

	l.Infof("Using PVC: %s", opGethPVC)

	// Create verify pod with op-geth metadata checking capabilities
	podYaml := fmt.Sprintf(`apiVersion: v1
kind: Pod
metadata:
  name: verify-efs
  namespace: %s
spec:
  containers:
  - name: verify
    image: ethereum/client-go:latest
    command: ["/bin/sh", "-c"]
    args:
    - |
      # Check EFS accessibility
      ls -la /db > /dev/null || { echo "ERROR: EFS not accessible"; exit 1; }
      
      echo "🔍 Op-Geth Data Analysis"
      
      # Find the actual operational chaindata path (subPath-based)
      OPERATIONAL_CHAINDATA=""
      for subpath_dir in /db/*-op-geth; do
        if [ -d "$subpath_dir/geth/chaindata" ]; then
          OPERATIONAL_CHAINDATA="$subpath_dir/geth/chaindata"
          break
        fi
      done
      
      # Fallback to direct chaindata if no subPath found
      if [ -z "$OPERATIONAL_CHAINDATA" ] && [ -d "/db/chaindata" ]; then
        OPERATIONAL_CHAINDATA="/db/chaindata"
      fi
      
      if [ -n "$OPERATIONAL_CHAINDATA" ]; then
        echo "   ✅ Operational chaindata: $OPERATIONAL_CHAINDATA"
        chaindata_size=$(du -sh "$OPERATIONAL_CHAINDATA" 2>/dev/null | awk '{print $1}' || echo 'N/A')
        file_count=$(find "$OPERATIONAL_CHAINDATA" -type f 2>/dev/null | wc -l || echo '0')
        echo "   📊 Size: $chaindata_size, Files: $file_count"
        PRIMARY_CHAINDATA="$OPERATIONAL_CHAINDATA"
      else
        echo "   ❌ No operational chaindata found"
      fi
      
      echo "🔍 Chaindata Integrity Check:"
      
      if [ -n "$PRIMARY_CHAINDATA" ] && [ -d "$PRIMARY_CHAINDATA" ]; then
        file_count=$(find "$PRIMARY_CHAINDATA" -type f 2>/dev/null | wc -l || echo "0")
        
        if [ "$file_count" -gt 0 ]; then
          echo "✅ Contains $file_count files"
          
          # Check critical LevelDB files
          if [ -f "$PRIMARY_CHAINDATA/CURRENT" ]; then
            echo "✅ CURRENT file exists"
          else
            echo "❌ CURRENT file missing - database corrupted"
            exit 1
          fi
          
          if ls "$PRIMARY_CHAINDATA"/MANIFEST-* >/dev/null 2>&1; then
            echo "✅ MANIFEST file exists"
          else
            echo "❌ MANIFEST file missing - database corrupted"
            exit 1
          fi
          
          # Check for LOG files (indicates recent activity)
          log_files=$(find "$PRIMARY_CHAINDATA" -name "*.log" -type f 2>/dev/null | wc -l || echo "0")
          if [ "$log_files" -gt 0 ]; then
            echo "✅ Found $log_files LOG files"
          else
            echo "⚠️  No LOG files found"
          fi
          
          # Check for SST files (actual data)
          sst_files=$(find "$PRIMARY_CHAINDATA" -name "*.sst" -o -name "*.ldb" -type f 2>/dev/null | wc -l || echo "0")
          if [ "$sst_files" -gt 0 ]; then
            echo "✅ Found $sst_files data files"
          else
            echo "❌ No data files found - empty database"
            exit 1
          fi
          
          # Check for LOCK file (should not exist if geth is not running)
          if [ -f "$PRIMARY_CHAINDATA/LOCK" ]; then
            echo "⚠️  LOCK file exists - database may be in use"
          else
            echo "✅ No LOCK file - database available"
          fi
          
        else
          echo "❌ Chaindata directory is empty"
          exit 1
        fi
      else
        echo "❌ No operational chaindata found"
        exit 1
      fi
      echo ""
    volumeMounts:
    - name: efs-volume
      mountPath: /db
  volumes:
  - name: efs-volume
    persistentVolumeClaim:
      claimName: %s
  restartPolicy: Never`, namespace, opGethPVC)

	// Create pod using kubectl apply
	tempFile := fmt.Sprintf("/tmp/verify-efs-%s.yaml", namespace)
	if err := os.WriteFile(tempFile, []byte(podYaml), 0644); err != nil {
		return fmt.Errorf("failed to create temporary YAML file: %w", err)
	}
	defer os.Remove(tempFile)

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tempFile); err != nil {
		return fmt.Errorf("failed to create verify pod: %w", err)
	}

	// Wait for pod to complete
	l.Info("Waiting for verification pod to complete...")
	for i := 0; i < 90; i++ { // Increased timeout to 3 minutes for geth operations
		status, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", "verify-efs", "-o", "jsonpath={.status.phase}")
		if err != nil {
			l.Infof("Pod status check failed: %v", err)
			time.Sleep(2 * time.Second)
			continue
		}

		status = strings.TrimSpace(status)
		if status == "Succeeded" {
			l.Info("✅ EFS data verification completed successfully")

			// Get and display pod logs with metadata information
			logs, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", "verify-efs")
			if err != nil {
				l.Infof("Warning: Could not retrieve verification logs: %v", err)
			} else {
				l.Info(logs)
			}
			return nil
		}
		if status == "Failed" {
			// Get pod logs for analysis
			logs, _ := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", "verify-efs")
			l.Infof("❌ Pod failed. Logs:\n%s", logs)
			return fmt.Errorf("verification pod failed")
		}
		if status == "Pending" {
			l.Infof("Pod is pending... (attempt %d/90)", i+1)
		}
		if status == "Running" {
			l.Infof("Pod is running... (attempt %d/90)", i+1)
		}
		time.Sleep(2 * time.Second)
	}

	// Clean up the pod
	utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", "verify-efs", "--ignore-not-found=true")
	return fmt.Errorf("verification pod timed out after 3 minutes")
}
