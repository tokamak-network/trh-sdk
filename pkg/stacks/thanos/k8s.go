package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

const (
	namespaceTerminatingPhase           = "Terminating"
	namespaceTerminatePollInterval      = 5 * time.Second
	namespaceTerminatePhasePollDeadline = 30 * time.Second
	// namespaceDeleteSafetyDeadline caps the total wall-clock spent inside
	// tryToDeleteK8sNamespace when the caller did not supply a ctx deadline.
	namespaceDeleteSafetyDeadline = 5 * time.Minute
)

// buildNamespaceFinalizeBody takes a namespace's JSON representation and
// returns a body suitable for the /finalize subresource that clears every
// finalizer entry. Uses a generic map so unknown / schema-evolving fields
// (notably status.conditions, which is an array on a real Terminating
// namespace) round-trip untouched. Pure logic — unit-testable independently
// of kubectl.
func buildNamespaceFinalizeBody(raw []byte) ([]byte, error) {
	var ns map[string]interface{}
	if err := json.Unmarshal(raw, &ns); err != nil {
		return nil, fmt.Errorf("namespace json: %w", err)
	}
	spec, _ := ns["spec"].(map[string]interface{})
	if spec == nil {
		spec = map[string]interface{}{}
		ns["spec"] = spec
	}
	spec["finalizers"] = []string{}
	return json.MarshalIndent(ns, "", "  ")
}

// extractNamespacePhase reads .status.phase from a namespace JSON payload
// without imposing a strict schema on the rest of the document.
func extractNamespacePhase(raw []byte) string {
	var ns map[string]interface{}
	if err := json.Unmarshal(raw, &ns); err != nil {
		return ""
	}
	status, _ := ns["status"].(map[string]interface{})
	if status == nil {
		return ""
	}
	phase, _ := status["phase"].(string)
	return phase
}

// tryToDeleteK8sNamespace deletes a namespace using a self-healing pattern:
//  1. Issue a non-blocking delete (`--wait=false`) so the apiserver flips the
//     namespace to Terminating without us holding kubectl open.
//  2. Wait briefly for the controller to reach Terminating (or for the
//     namespace to disappear).
//  3. If the namespace is still around, best-effort clear finalizers on
//     resources that commonly stall termination (LoadBalancer Services, PVCs,
//     bound PVs) and force-clear the namespace's own finalizers via the
//     /finalize subresource.
//  4. Poll until the caller-supplied ctx deadline for the namespace to vanish.
//
// Caller controls the overall deadline via ctx; if ctx has no deadline a
// safety-net timeout is applied so a missing controller cannot wedge this
// call indefinitely.
func (t *ThanosStack) tryToDeleteK8sNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		t.logger.Warn("Namespace is empty, skipping namespace deletion")
		return nil
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, namespaceDeleteSafetyDeadline)
		defer cancel()
	}

	exists, err := utils.CheckNamespaceExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("check namespace %q exists: %w", namespace, err)
	}
	if !exists {
		t.logger.Infow("Namespace does not exist, skipping deletion", "namespace", namespace)
		return nil
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace, "--wait=false", "--ignore-not-found=true"); err != nil {
		t.logger.Warnw("Initial namespace delete failed; will continue with self-heal", "namespace", namespace, "err", err)
	}

	t.waitForNamespaceGoneOrTerminating(ctx, namespace, namespaceTerminatePhasePollDeadline)

	stillExists, _ := utils.CheckNamespaceExists(ctx, namespace)
	if !stillExists {
		return nil
	}

	t.clearStuckResourceFinalizers(ctx, namespace)

	if err := t.forceFinalizeNamespace(ctx, namespace); err != nil {
		t.logger.Warnw("Force-finalize namespace failed", "namespace", namespace, "err", err)
	}

	return t.waitForNamespaceGone(ctx, namespace)
}

// waitForNamespaceGoneOrTerminating polls until the namespace disappears or
// reaches the Terminating phase, or the deadline elapses. Returns true if
// either condition was observed before the deadline.
func (t *ThanosStack) waitForNamespaceGoneOrTerminating(ctx context.Context, namespace string, deadline time.Duration) bool {
	deadlineCtx, cancel := context.WithTimeout(ctx, deadline)
	defer cancel()

	for {
		exists, err := utils.CheckNamespaceExists(deadlineCtx, namespace)
		if err == nil && !exists {
			return true
		}
		if phase, _ := t.getNamespacePhase(deadlineCtx, namespace); phase == namespaceTerminatingPhase {
			return true
		}

		select {
		case <-deadlineCtx.Done():
			return false
		case <-time.After(namespaceTerminatePollInterval):
		}
	}
}

// waitForNamespaceGone polls until the namespace is gone or ctx expires.
func (t *ThanosStack) waitForNamespaceGone(ctx context.Context, namespace string) error {
	for {
		exists, err := utils.CheckNamespaceExists(ctx, namespace)
		if err != nil {
			t.logger.Debugw("namespace existence check transient error", "namespace", namespace, "err", err)
		} else if !exists {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("namespace %q still present after deadline: %w", namespace, ctx.Err())
		case <-time.After(namespaceTerminatePollInterval):
		}
	}
}

func (t *ThanosStack) getNamespacePhase(ctx context.Context, namespace string) (string, error) {
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "-o", "jsonpath={.status.phase}")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// clearStuckResourceFinalizers patches Services, PVCs, and bound PVs in the
// namespace to drop their finalizers. Best-effort: each failure is logged but
// not returned. The intent is to unblock namespace termination when the
// controllers responsible for those finalizers (AWS LB Controller, EFS CSI,
// etc.) are no longer reconciling — for example, after their helm releases
// were uninstalled earlier in the destroy flow.
func (t *ThanosStack) clearStuckResourceFinalizers(ctx context.Context, namespace string) {
	for _, kind := range []string{"service", "pvc"} {
		for _, name := range t.listResourceNames(ctx, kind, namespace) {
			if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", kind, name, "-n", namespace, "--type", "merge", "-p", `{"metadata":{"finalizers":null}}`); err != nil {
				t.logger.Debugw("clear finalizers", "kind", kind, "namespace", namespace, "name", name, "err", err)
			}
		}
	}

	for _, name := range t.listPVsBoundToNamespace(ctx, namespace) {
		if _, err := utils.ExecuteCommand(ctx, "kubectl", "patch", "pv", name, "--type", "merge", "-p", `{"metadata":{"finalizers":null}}`); err != nil {
			t.logger.Debugw("clear pv finalizers", "name", name, "err", err)
		}
	}
}

func (t *ThanosStack) listResourceNames(ctx context.Context, kind, namespace string) []string {
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", kind, "-n", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	if err != nil {
		return nil
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil
	}
	return strings.Fields(out)
}

func (t *ThanosStack) listPVsBoundToNamespace(ctx context.Context, namespace string) []string {
	out, err := utils.ExecuteCommand(ctx, "kubectl", "get", "pv", "-o", "json")
	if err != nil {
		return nil
	}
	var list struct {
		Items []struct {
			Metadata struct {
				Name string `json:"name"`
			} `json:"metadata"`
			Spec struct {
				ClaimRef struct {
					Namespace string `json:"namespace"`
				} `json:"claimRef"`
			} `json:"spec"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		return nil
	}
	names := make([]string, 0, len(list.Items))
	for _, it := range list.Items {
		if it.Spec.ClaimRef.Namespace == namespace {
			names = append(names, it.Metadata.Name)
		}
	}
	return names
}

// forceFinalizeNamespace clears finalizers on the namespace itself via the
// /finalize subresource. Last resort when controller-managed finalizers
// cannot be processed.
func (t *ThanosStack) forceFinalizeNamespace(ctx context.Context, namespace string) error {
	raw, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "-o", "json")
	if err != nil {
		return fmt.Errorf("get namespace json: %w", err)
	}

	body, err := buildNamespaceFinalizeBody([]byte(raw))
	if err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp("", "ns-finalize-*.json")
	if err != nil {
		return fmt.Errorf("create temp finalize file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(body); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write finalize body: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close finalize file: %w", err)
	}

	if _, err := utils.ExecuteCommand(ctx, "kubectl", "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", tmpPath); err != nil {
		return fmt.Errorf("replace finalize subresource: %w", err)
	}
	return nil
}
