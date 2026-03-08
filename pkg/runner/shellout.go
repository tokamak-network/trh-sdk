package runner

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShellOutK8sRunner implements K8sRunner by shelling out to the kubectl binary.
// It preserves the exact existing behaviour and serves as a fallback when
// NativeK8sRunner is not available or when --legacy mode is requested.
type ShellOutK8sRunner struct{}

func (r *ShellOutK8sRunner) Apply(ctx context.Context, manifest []byte) error {
	// Write to a temp file and call kubectl apply -f <file>.
	// Piping via stdin would require --filename=- which some older kubectl versions reject.
	tmp, err := writeTempManifest(manifest)
	if err != nil {
		return fmt.Errorf("shellout apply: write temp manifest: %w", err)
	}
	defer removeTempFile(tmp)

	_, err = utils.ExecuteCommand(ctx, "kubectl", "apply", "-f", tmp)
	if err != nil {
		return fmt.Errorf("shellout apply: %w", err)
	}
	return nil
}

func (r *ShellOutK8sRunner) Delete(ctx context.Context, resource, name, namespace string, ignoreNotFound bool) error {
	args := []string{"delete", resource, name}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if ignoreNotFound {
		args = append(args, "--ignore-not-found=true")
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	if err != nil {
		return fmt.Errorf("shellout delete %s/%s: %w", resource, name, err)
	}
	return nil
}

func (r *ShellOutK8sRunner) Get(ctx context.Context, resource, name, namespace string) ([]byte, error) {
	args := []string{"get", resource, name, "-o", "json"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	if err != nil {
		return nil, fmt.Errorf("shellout get %s/%s: %w", resource, name, err)
	}
	return []byte(out), nil
}

func (r *ShellOutK8sRunner) List(ctx context.Context, resource, namespace, labelSelector string) ([]byte, error) {
	args := []string{"get", resource, "-o", "json"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	if labelSelector != "" {
		args = append(args, "-l", labelSelector)
	}
	out, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	if err != nil {
		return nil, fmt.Errorf("shellout list %s: %w", resource, err)
	}
	return []byte(out), nil
}

func (r *ShellOutK8sRunner) Patch(ctx context.Context, resource, name, namespace string, patch []byte) error {
	args := []string{"patch", resource, name, "-p", string(patch), "--type=merge"}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	if err != nil {
		return fmt.Errorf("shellout patch %s/%s: %w", resource, name, err)
	}
	return nil
}

func (r *ShellOutK8sRunner) Wait(ctx context.Context, resource, name, namespace, condition string, timeout time.Duration) error {
	args := []string{
		"wait", resource + "/" + name,
		"--for=condition=" + condition,
		fmt.Sprintf("--timeout=%ds", int(timeout.Seconds())),
	}
	if namespace != "" {
		args = append(args, "-n", namespace)
	}
	_, err := utils.ExecuteCommand(ctx, "kubectl", args...)
	if err != nil {
		return fmt.Errorf("shellout wait %s/%s condition=%s: %w", resource, name, condition, err)
	}
	return nil
}

func (r *ShellOutK8sRunner) EnsureNamespace(ctx context.Context, namespace string) error {
	exists, err := r.NamespaceExists(ctx, namespace)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	_, err = utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace)
	if err != nil {
		return fmt.Errorf("shellout create namespace %s: %w", namespace, err)
	}
	return nil
}

func (r *ShellOutK8sRunner) NamespaceExists(ctx context.Context, namespace string) (bool, error) {
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return false, fmt.Errorf("shellout get namespace %s: %w", namespace, err)
	}
	return strings.TrimSpace(out) != "", nil
}

func (r *ShellOutK8sRunner) Logs(ctx context.Context, pod, namespace, container string, follow bool) (io.ReadCloser, error) {
	// Streaming logs via subprocess is complex; return an error for now.
	// Full streaming will be implemented in NativeK8sRunner.
	return nil, fmt.Errorf("shellout logs: streaming not supported in legacy mode; use kubectl logs %s -n %s directly", pod, namespace)
}
