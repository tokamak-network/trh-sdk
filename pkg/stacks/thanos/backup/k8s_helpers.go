package backup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// writeTempFile writes data to a temporary YAML file and returns its path.
// The caller is responsible for removing the file when done.
func writeTempFile(data []byte) (string, error) {
	f, err := os.CreateTemp("", "trh-backup-*.yaml")
	if err != nil {
		return "", fmt.Errorf("writeTempFile: failed to create temp file: %w", err)
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		os.Remove(f.Name())
		return "", fmt.Errorf("writeTempFile: failed to write temp file: %w", err)
	}
	return f.Name(), nil
}

// k8sListStatefulSetNames returns all StatefulSet names in the namespace.
func (b *BackupClient) k8sListStatefulSetNames(ctx context.Context, namespace string) ([]string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.List(ctx, "statefulsets", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sListStatefulSetNames: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListStatefulSetNames: failed to parse JSON: %w", err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "statefulsets", "-o", "name")
	if err != nil {
		return nil, fmt.Errorf("k8sListStatefulSetNames: failed to list StatefulSets: %w", err)
	}
	var names []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		names = append(names, strings.TrimPrefix(line, "statefulset.apps/"))
	}
	return names, nil
}

// k8sDeletePod deletes a pod by name, ignoring not-found errors.
func (b *BackupClient) k8sDeletePod(ctx context.Context, name, namespace string) error {
	if b.k8sRunner != nil {
		return b.k8sRunner.Delete(ctx, "pod", name, namespace, true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pod", name, "--ignore-not-found=true")
	return err
}

// k8sListPVCNames returns all PVC names in the namespace.
func (b *BackupClient) k8sListPVCNames(ctx context.Context, namespace string) ([]string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.List(ctx, "pvc", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sListPVCNames: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListPVCNames: failed to parse JSON: %w", err)
		}
		names := make([]string, 0, len(list.Items))
		for _, item := range list.Items {
			names = append(names, item.Metadata.Name)
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", "-o", "jsonpath={range .items[*]}{.metadata.name}{\"\\n\"}{end}")
	if err != nil {
		return nil, fmt.Errorf("k8sListPVCNames: %w", err)
	}
	return strings.Fields(out), nil
}

// k8sGetPVCExists returns true if the named PVC exists in the namespace.
func (b *BackupClient) k8sGetPVCExists(ctx context.Context, name, namespace string) (bool, error) {
	if b.k8sRunner != nil {
		_, err := b.k8sRunner.Get(ctx, "pvc", name, namespace)
		if err != nil {
			// treat not-found as false
			return false, nil
		}
		return true, nil
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", name)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// k8sGetPVCVolumeName returns the spec.volumeName of the named PVC.
func (b *BackupClient) k8sGetPVCVolumeName(ctx context.Context, pvcName, namespace string) (string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.Get(ctx, "pvc", pvcName, namespace)
		if err != nil {
			return "", fmt.Errorf("k8sGetPVCVolumeName: %w", err)
		}
		var obj struct {
			Spec struct {
				VolumeName string `json:"volumeName"`
			} `json:"spec"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sGetPVCVolumeName: failed to parse JSON: %w", err)
		}
		return obj.Spec.VolumeName, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", pvcName, "-o", "jsonpath={.spec.volumeName}")
	if err != nil {
		return "", fmt.Errorf("k8sGetPVCVolumeName: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// k8sListPodsUsingPVC returns pod names in the namespace whose volumes reference claimName.
func (b *BackupClient) k8sListPodsUsingPVC(ctx context.Context, claimName, namespace string) ([]string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.List(ctx, "pods", namespace, "")
		if err != nil {
			return nil, fmt.Errorf("k8sListPodsUsingPVC: %w", err)
		}
		var list struct {
			Items []struct {
				Metadata struct {
					Name string `json:"name"`
				} `json:"metadata"`
				Spec struct {
					Volumes []struct {
						PVC *struct {
							ClaimName string `json:"claimName"`
						} `json:"persistentVolumeClaim"`
					} `json:"volumes"`
				} `json:"spec"`
			} `json:"items"`
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			return nil, fmt.Errorf("k8sListPodsUsingPVC: failed to parse JSON: %w", err)
		}
		var names []string
		for _, pod := range list.Items {
			for _, vol := range pod.Spec.Volumes {
				if vol.PVC != nil && vol.PVC.ClaimName == claimName {
					names = append(names, pod.Metadata.Name)
					break
				}
			}
		}
		return names, nil
	}
	out, err := utils.ExecuteCommand(ctx, "sh", "-c",
		fmt.Sprintf("kubectl -n %s get pods -o json | jq -r '.items[] | select(.spec.volumes[]?.persistentVolumeClaim.claimName == \"%s\") | .metadata.name'",
			namespace, claimName))
	if err != nil || strings.TrimSpace(out) == "" {
		return nil, nil
	}
	return strings.Fields(strings.TrimSpace(out)), nil
}

// k8sDeletePVCNoWait deletes the named PVC without waiting, ignoring not-found errors.
func (b *BackupClient) k8sDeletePVCNoWait(ctx context.Context, name, namespace string) error {
	if b.k8sRunner != nil {
		return b.k8sRunner.Delete(ctx, "pvc", name, namespace, true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "delete", "pvc", name, "--wait=false", "--ignore-not-found=true")
	return err
}

// k8sDeletePV deletes a cluster-scoped PersistentVolume, ignoring not-found errors.
func (b *BackupClient) k8sDeletePV(ctx context.Context, name string) error {
	if b.k8sRunner != nil {
		return b.k8sRunner.Delete(ctx, "pv", name, "", true)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "pv", name, "--ignore-not-found=true")
	return err
}

// k8sApplyManifest applies a YAML manifest (bytes).
func (b *BackupClient) k8sApplyManifest(ctx context.Context, manifest []byte) error {
	if b.k8sRunner != nil {
		return b.k8sRunner.Apply(ctx, manifest)
	}
	// Write to a temp file and kubectl apply -f
	tmpFile, err := writeTempFile(manifest)
	if err != nil {
		return fmt.Errorf("k8sApplyManifest: %w", err)
	}
	defer os.Remove(tmpFile)
	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmpFile)
	return err
}

// k8sGetPVCPhase returns the status.phase of a PVC ("Bound", "Pending", …).
func (b *BackupClient) k8sGetPVCPhase(ctx context.Context, name, namespace string) (string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.Get(ctx, "pvc", name, namespace)
		if err != nil {
			return "", fmt.Errorf("k8sGetPVCPhase: %w", err)
		}
		var obj struct {
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sGetPVCPhase: failed to parse JSON: %w", err)
		}
		return obj.Status.Phase, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pvc", name, "-o", "jsonpath={.status.phase}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sGetPodPhase returns the status.phase of the named pod.
func (b *BackupClient) k8sGetPodPhase(ctx context.Context, name, namespace string) (string, error) {
	if b.k8sRunner != nil {
		raw, err := b.k8sRunner.Get(ctx, "pod", name, namespace)
		if err != nil {
			return "", fmt.Errorf("k8sGetPodPhase: %w", err)
		}
		var obj struct {
			Status struct {
				Phase string `json:"phase"`
			} `json:"status"`
		}
		if err := json.Unmarshal(raw, &obj); err != nil {
			return "", fmt.Errorf("k8sGetPodPhase: failed to parse JSON: %w", err)
		}
		return obj.Status.Phase, nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pod", name, "-o", "jsonpath={.status.phase}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// k8sGetPodLogs returns the logs of the named pod as a string.
func (b *BackupClient) k8sGetPodLogs(ctx context.Context, name, namespace string) (string, error) {
	if b.k8sRunner != nil {
		rc, err := b.k8sRunner.Logs(ctx, name, namespace, "", false)
		if err != nil {
			return "", fmt.Errorf("k8sGetPodLogs: %w", err)
		}
		defer rc.Close()
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, rc); err != nil {
			return "", fmt.Errorf("k8sGetPodLogs: failed to read logs: %w", err)
		}
		return buf.String(), nil
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", "-n", namespace, "logs", name)
	if err != nil {
		return "", err
	}
	return out, nil
}
