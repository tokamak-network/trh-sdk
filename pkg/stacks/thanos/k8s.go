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

func (t *ThanosStack) tryToDeleteK8sNamespace(ctx context.Context, namespace string, logFileName string) error {
	if namespace == "" {
		return nil
	}
	output, err := utils.ExecuteCommand("kubectl", logFileName, "get", "namespace", namespace, "-o", "json")
	if err != nil {
		utils.LogToFile(logFileName, fmt.Sprintf("Error getting namespace status: %s", err), true)
	}

	var status K8sNamespaceStatus
	if err := json.Unmarshal([]byte(output), &status); err != nil {
		return err
	}

	if status.Status.Phase == "Terminating" {
		utils.LogToFile(logFileName, "Namespace is already terminating", true)
		// Replace the state to pass on stucking deleted namespace
		status.Spec["finalizers"] = make([]string, 0)

		// write to the temporary file
		tmpFile, err := os.Create("/tmp/namespace.json")
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error creating temporary file: %s", err), true)
			return err
		}
		defer tmpFile.Close()

		encoder := json.NewEncoder(tmpFile)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(status)
		if err != nil {
			return err
		}

		// apply the changes
		_, err = utils.ExecuteCommand("kubectl", logFileName, "replace", "--raw", fmt.Sprintf("/api/v1/namespaces/%s/finalize", namespace), "-f", "/tmp/namespace.json")
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error applying changes to namespace: %s", err), true)
			return err
		}

		// Delete the temporary file
		if err := os.Remove("/tmp/namespace.json"); err != nil {
			fmt.Println("Error deleting temporary file:", err)
			return err
		}
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := utils.ExecuteCommand("kubectl", logFileName, "delete", "namespace", namespace)
		if err != nil {
			utils.LogToFile(logFileName, fmt.Sprintf("Error deleting namespace: %s", err), true)
			done <- err
			return
		}
		done <- nil
	}()

	select {
	case err := <-done:
		return err
	case <-ctxWithTimeout.Done():
		utils.LogToFile(logFileName, "Timeout reached while deleting namespace", true)
		return ctxWithTimeout.Err()
	}
}
