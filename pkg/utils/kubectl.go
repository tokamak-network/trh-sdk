package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type PodsJSON struct {
	APIVersion string `json:"apiVersion"`
	Items      []struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			Name      string `json:"name"`
			Namespace string `json:"namespace"`
			UID       string `json:"uid"`
		} `json:"metadata"`
		Status struct {
			Conditions []struct {
				LastProbeTime      any       `json:"lastProbeTime"`
				LastTransitionTime time.Time `json:"lastTransitionTime"`
				Status             string    `json:"status"`
				Type               string    `json:"type"`
			} `json:"conditions"`
			ContainerStatuses []struct {
				ContainerID string `json:"containerID"`
				Image       string `json:"image"`
				ImageID     string `json:"imageID"`
				LastState   struct {
				} `json:"lastState"`
				Name         string `json:"name"`
				Ready        bool   `json:"ready"`
				RestartCount int    `json:"restartCount"`
				Started      bool   `json:"started"`
				State        struct {
					Running struct {
						StartedAt time.Time `json:"startedAt"`
					} `json:"running"`
				} `json:"state"`
			} `json:"containerStatuses"`
			HostIP  string `json:"hostIP"`
			HostIPs []struct {
				IP string `json:"ip"`
			} `json:"hostIPs"`
			Phase  string `json:"phase"`
			PodIP  string `json:"podIP"`
			PodIPs []struct {
				IP string `json:"ip"`
			} `json:"podIPs"`
			QosClass  string    `json:"qosClass"`
			StartTime time.Time `json:"startTime"`
		} `json:"status,omitempty"`
	} `json:"items"`
	Kind     string `json:"kind"`
	Metadata struct {
		ResourceVersion string `json:"resourceVersion"`
	} `json:"metadata"`
}

type SvcJSON struct {
	APIVersion string `json:"apiVersion"`
	Items      []struct {
		APIVersion string `json:"apiVersion"`
		Kind       string `json:"kind"`
		Metadata   struct {
			Annotations struct {
				MetaHelmShReleaseName      string `json:"meta.helm.sh/release-name"`
				MetaHelmShReleaseNamespace string `json:"meta.helm.sh/release-namespace"`
			} `json:"annotations"`
			CreationTimestamp time.Time `json:"creationTimestamp"`
			Labels            struct {
				AppKubernetesIoInstance    string `json:"app.kubernetes.io/instance"`
				AppKubernetesIoManagedBy   string `json:"app.kubernetes.io/managed-by"`
				AppKubernetesIoName        string `json:"app.kubernetes.io/name"`
				AppKubernetesIoVersion     string `json:"app.kubernetes.io/version"`
				ExternalSecretsIoComponent string `json:"external-secrets.io/component"`
				HelmShChart                string `json:"helm.sh/chart"`
			} `json:"labels"`
			Name            string `json:"name"`
			Namespace       string `json:"namespace"`
			ResourceVersion string `json:"resourceVersion"`
			UID             string `json:"uid"`
		} `json:"metadata"`
		Spec struct {
			ClusterIP             string   `json:"clusterIP"`
			ClusterIPs            []string `json:"clusterIPs"`
			InternalTrafficPolicy string   `json:"internalTrafficPolicy"`
			IPFamilies            []string `json:"ipFamilies"`
			IPFamilyPolicy        string   `json:"ipFamilyPolicy"`
			Ports                 []struct {
				Name       string `json:"name"`
				Port       int    `json:"port"`
				Protocol   string `json:"protocol"`
				TargetPort any    `json:"targetPort"`
			} `json:"ports"`
			Selector struct {
				AppKubernetesIoInstance string `json:"app.kubernetes.io/instance"`
				AppKubernetesIoName     string `json:"app.kubernetes.io/name"`
			} `json:"selector"`
			SessionAffinity string `json:"sessionAffinity"`
			Type            string `json:"type"`
		} `json:"spec"`
		Status struct {
			LoadBalancer struct {
				Ingress []struct {
					IP       string `json:"ip,omitempty"`
					Hostname string `json:"hostname,omitempty"`
				} `json:"ingress"`
			} `json:"loadBalancer"`
		} `json:"status"`
	} `json:"items"`
	Kind     string `json:"kind"`
	Metadata struct {
		ResourceVersion string `json:"resourceVersion"`
	} `json:"metadata"`
}

type IngressJSON struct {
	Items []struct {
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Status struct {
			LoadBalancer struct {
				Ingress []struct {
					IP       string `json:"ip,omitempty"`
					Hostname string `json:"hostname,omitempty"`
				} `json:"ingress"`
			} `json:"loadBalancer"`
		} `json:"status"`
	} `json:"items"`
}

func getK8sPods(ctx context.Context, namespace string) (string, error) {
	return ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pods", "-o", "json")
}

func getK8sPodsByLabel(ctx context.Context, namespace string, labelSelector string) (string, error) {
	if strings.TrimSpace(labelSelector) == "" {
		return "", fmt.Errorf("labelSelector must not be empty")
	}
	return ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "pods", "-l", labelSelector, "-o", "json")
}

func GetK8sPods(ctx context.Context, namespace string) ([]string, error) {
	output, err := getK8sPods(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var podData PodsJSON
	if err := json.Unmarshal([]byte(output), &podData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	pods := make([]string, 0)
	for _, item := range podData.Items {
		if item.Status.Phase == "Running" || item.Status.Phase == "Pending" || item.Status.Phase == "Succeeded" {
			pods = append(pods, item.Metadata.Name)
		}
	}
	return pods, nil
}

func GetPodsByName(ctx context.Context, namespace string, podName string) ([]string, error) {
	output, err := getK8sPods(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var podData PodsJSON
	if err := json.Unmarshal([]byte(output), &podData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	pods := make([]string, 0)
	for _, item := range podData.Items {
		if item.Status.Phase != "Running" && item.Status.Phase != "Pending" && item.Status.Phase != "Succeeded" {
			continue
		}
		if strings.HasPrefix(item.Metadata.Name, podName) {
			pods = append(pods, item.Metadata.Name)
		}
	}
	return pods, nil
}

// GetPodNamesByLabel returns pod names filtered by the given label selector in a namespace
func GetPodNamesByLabel(ctx context.Context, namespace string, labelSelector string) ([]string, error) {
	output, err := getK8sPodsByLabel(ctx, namespace, labelSelector)
	if err != nil {
		return nil, err
	}

	var podData PodsJSON
	if err := json.Unmarshal([]byte(output), &podData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	allowed := map[string]bool{"Running": true, "Pending": true, "Succeeded": true}
	pods := make([]string, 0, len(podData.Items))
	for _, item := range podData.Items {
		if allowed[item.Status.Phase] {
			pods = append(pods, item.Metadata.Name)
		}
	}
	return pods, nil
}

func getK8sIngresses(ctx context.Context, namespace string) (string, error) {
	return ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "ingress", "-o", "json")
}
func getK8sSVC(ctx context.Context, namespace string) (string, error) {
	return ExecuteCommand(ctx, "kubectl", "-n", namespace, "get", "svc", "-o", "json")
}

func GetIngresses(ctx context.Context, namespace string) (map[string][]string, error) {
	output, err := getK8sIngresses(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var ingressData IngressJSON
	if err := json.Unmarshal([]byte(output), &ingressData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	addresses := make(map[string][]string)
	for _, item := range ingressData.Items {
		// Extract IP or Hostname
		for _, ingress := range item.Status.LoadBalancer.Ingress {
			if addresses[item.Metadata.Name] == nil {
				addresses[item.Metadata.Name] = make([]string, 0)
			}
			if ingress.IP != "" {
				addresses[item.Metadata.Name] = append(addresses[item.Metadata.Name], ingress.IP)
			}
			if ingress.Hostname != "" {
				addresses[item.Metadata.Name] = append(addresses[item.Metadata.Name], ingress.Hostname)
			}
		}
	}
	return addresses, nil
}

func GetAddressByIngress(ctx context.Context, namespace string, ingressName string) ([]string, error) {
	output, err := getK8sIngresses(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var ingressData IngressJSON
	if err := json.Unmarshal([]byte(output), &ingressData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	addresses := make([]string, 0)
	for _, item := range ingressData.Items {
		if strings.Contains(item.Metadata.Name, ingressName) {
			// Extract IP or Hostname
			for _, ingress := range item.Status.LoadBalancer.Ingress {
				addresses = append(addresses, ingress.Hostname)
			}
		}
	}
	return addresses, nil
}

func GetAddressByService(ctx context.Context, namespace string, serviceName string) ([]string, error) {
	output, err := getK8sSVC(ctx, namespace)
	if err != nil {
		return nil, err
	}
	var serviceData SvcJSON
	if err := json.Unmarshal([]byte(output), &serviceData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	addresses := make([]string, 0)
	for _, item := range serviceData.Items {
		if strings.Contains(item.Metadata.Name, serviceName) && item.Spec.Type == "LoadBalancer" {
			// Extract hostname from LoadBalancer status
			for _, ingress := range item.Status.LoadBalancer.Ingress {
				if ingress.Hostname != "" {
					addresses = append(addresses, ingress.Hostname)
				}
			}
		}
	}
	return addresses, nil
}

func GetServiceNames(ctx context.Context, namespace string, serviceName string) ([]string, error) {
	output, err := getK8sSVC(ctx, namespace)
	if err != nil {
		return nil, err
	}

	var serviceData SvcJSON
	if err := json.Unmarshal([]byte(output), &serviceData); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	addresses := make([]string, 0)
	for _, item := range serviceData.Items {
		if strings.Contains(item.Metadata.Name, serviceName) {
			addresses = append(addresses, item.Metadata.Name)
		}
	}
	return addresses, nil
}

func CheckPVCStatus(ctx context.Context, namespace string) (bool, error) {
	cmd := []string{
		"get", "pvc",
		"-n", namespace,
		"-o", "jsonpath={range .items[*]}{.status.phase}{\"\\n\"}{end}",
	}
	output, err := ExecuteCommand(ctx, "kubectl", cmd...)
	if err != nil {
		return false, fmt.Errorf("failed to get PVC status: %w", err)
	}
	// Split output into lines and check each PVC status
	pvcStatuses := strings.Split(strings.TrimSpace(output), "\n")
	if len(pvcStatuses) == 0 {
		return false, fmt.Errorf("no PVCs found in namespace %s", namespace)
	}
	fmt.Println("PVC statuses:", pvcStatuses)
	for _, status := range pvcStatuses {
		if status != "Bound" {
			fmt.Printf("⚠️ Found PVC with status: %s\n", status)
			return false, nil
		}
	}
	fmt.Println("✅ All PVCs are bound")
	return true, nil
}

func WaitPVCReady(ctx context.Context, namespace string) error {
	maxRetries := 5
	for i := 0; i < maxRetries; i++ {
		isReady, err := CheckPVCStatus(ctx, namespace)
		if err != nil {
			fmt.Println("Error checking PVC status:", err)
			return err
		}
		if isReady {
			return nil
		}
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("PVC not ready after %d attempts", maxRetries)
}

// CheckNamespaceExists checks if a namespace exists in Kubernetes
func CheckNamespaceExists(ctx context.Context, namespace string) (bool, error) {
	output, err := ExecuteCommand(ctx, "kubectl", "get", "namespace", namespace, "--ignore-not-found=true")
	if err != nil {
		return false, fmt.Errorf("failed to check namespace existence: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

// EnsureNamespaceExists checks if namespace exists and creates it if needed
func EnsureNamespaceExists(ctx context.Context, namespace string) error {
	exists, err := CheckNamespaceExists(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to check namespace existence: %w", err)
	}

	if !exists {
		if _, err := ExecuteCommand(ctx, "kubectl", "create", "namespace", namespace); err != nil {
			return fmt.Errorf("failed to create namespace: %w", err)
		}
	}

	return nil
}
