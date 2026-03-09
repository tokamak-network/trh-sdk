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
//
// SECURITY NOTE: resource, name, namespace, condition, and patch parameters are
// passed as discrete arguments to utils.ExecuteCommand, which uses os/exec.Command
// directly (not via a shell). Shell metacharacters in these values therefore have
// no effect. Callers must still validate inputs for semantic correctness before
// passing them here.
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
	if resource == "" {
		return fmt.Errorf("shellout delete: resource name cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("shellout delete: object name cannot be empty")
	}
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
	if resource == "" {
		return nil, fmt.Errorf("shellout get: resource name cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("shellout get: object name cannot be empty")
	}
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
	if resource == "" {
		return nil, fmt.Errorf("shellout list: resource name cannot be empty")
	}
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
	if resource == "" {
		return fmt.Errorf("shellout patch: resource name cannot be empty")
	}
	if name == "" {
		return fmt.Errorf("shellout patch: object name cannot be empty")
	}
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
	// Optimistic create: attempt first, then inspect the error.
	// This avoids the TOCTOU race inherent in a check-then-create pattern.
	_, err := utils.ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace)
	if err != nil {
		if isKubectlAlreadyExistsErr(err) {
			return nil
		}
		return fmt.Errorf("shellout create namespace %s: %w", namespace, err)
	}
	return nil
}

// isKubectlAlreadyExistsErr reports whether err came from kubectl reporting that
// a resource already exists. kubectl always uses English error messages regardless
// of locale, so string matching is reliable here.
// Example kubectl output: "Error from server (AlreadyExists): namespaces "foo" already exists"
func isKubectlAlreadyExistsErr(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "AlreadyExists") || strings.Contains(msg, "already exists")
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
	return nil, fmt.Errorf("shellout logs: streaming is not supported in legacy mode (TRHS_LEGACY=1 or UseNative=false); " +
		"switch to native mode (RunnerConfig{UseNative: true}) or run 'kubectl logs' directly")
}
