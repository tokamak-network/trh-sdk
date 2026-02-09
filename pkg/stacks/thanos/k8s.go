package thanos

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

type K8sNamespaceStatus struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   interface{}            `json:"metadata"`
	Spec       map[string]interface{} `json:"spec"`
	Status     struct {
		Phase      string `json:"phase"`
		Conditions string `json:"conditions"`
	} `json:"status"`
}

func (t *ThanosStack) tryToDeleteK8sNamespace(ctx context.Context, namespace string) error {
	if namespace == "" {
		t.logger.Warn("Namespace is empty, skipping namespace deletion")
		return nil
	}

	exists, err := utils.CheckNamespaceExists(ctx, namespace)
	if err != nil {
		t.logger.Error("Failed to check namespace existence", "namespace", namespace, "err", err)
		return err
	}
	if !exists {
		t.logger.Info("Namespace does not exist, skipping deletion")
		return nil
	}

	output, err := utils.ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "-o", "json")
	if err != nil {
		t.logger.Error("Error getting namespace status", "err", err)
		return err
	}

	var status K8sNamespaceStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		t.logger.Error("Error unmarshalling namespace", "err", err)
		return err
	}

	if status.Status.Phase == "Terminating" {
		t.logger.Info("Namespace is already terminating")
		// Replace the state to pass on stucking deleted namespace
		status.Spec["finalizers"] = make([]string, 0)

		// write to the temporary file
		tmpFile, err := os.Create("/tmp/namespace.json")
		if err != nil {
			t.logger.Error("Error creating temporary file", "err", err)
			return err
		}
		defer tmpFile.Close()

		encoder := json.NewEncoder(tmpFile)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(status)
		if err != nil {
			t.logger.Error("Error encoding namespace", "err", err)
			return err
		}

		// apply the changes
		_, err = utils.ExecuteCommand(ctx, "kubectl", "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", "/tmp/namespace.json")
		if err != nil {
			t.logger.Error("Error applying changes to namespace", "err", err)
			return err
		}

		// Delete the temporary file
		if err := os.Remove("/tmp/namespace.json"); err != nil {
			t.logger.Error("Error deleting temporary file", "err", err)
			return err
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := utils.ExecuteCommand(ctx, "kubectl", "delete", "namespace", namespace)
		if err != nil {
			t.logger.Error("Error deleting namespace", "err", err)
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctxWithTimeout.Done():
		t.logger.Error("Timeout reached while deleting namespace")
		return ctxWithTimeout.Err()
	}
}
