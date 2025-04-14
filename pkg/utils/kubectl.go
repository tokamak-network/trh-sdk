package utils

import (
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
				TargetPort int    `json:"targetPort"`
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

func getK8sPods(namespace string) (string, error) {
	return ExecuteCommand("kubectl", "-n", namespace, "get", "pods", "-o", "json")
}

func GetK8sPods(namespace string) ([]string, error) {
	output, err := getK8sPods(namespace)
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

func GetPodsByName(namespace string, podName string) ([]string, error) {
	output, err := getK8sPods(namespace)
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

func getK8sIngresses(namespace string) (string, error) {
	return ExecuteCommand("kubectl", "-n", namespace, "get", "ingress", "-o", "json")
}
func getK8sSVC(namespace string) (string, error) {
	return ExecuteCommand("kubectl", "-n", namespace, "get", "svc", "-o", "json")
}

func GetIngresses(namespace string) (map[string][]string, error) {
	output, err := getK8sIngresses(namespace)
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

func GetAddressByIngress(namespace string, ingressName string) ([]string, error) {
	output, err := getK8sIngresses(namespace)
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

func GetServiceNames(namespace string, serviceName string) ([]string, error) {
	output, err := getK8sSVC(namespace)
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
