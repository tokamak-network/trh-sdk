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

func GetK8sPods(namespace string) (string, error) {
	return ExecuteCommand("kubectl", "-n", namespace, "get", "pods", "-o", "json")
}

func GetPodsByName(namespace string, podName string) ([]string, error) {
	output, err := GetK8sPods(namespace)
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
		if strings.HasPrefix(item.Metadata.Name, podName) {
			pods = append(pods, item.Metadata.Name)
		}
	}
	return pods, nil
}

func GetK8sIngresses(namespace string) (string, error) {
	return ExecuteCommand("kubectl", "-n", namespace, "get", "ingress", "-o", "json")
}

func GetAddressByIngress(namespace string, ingressName string) ([]string, error) {
	output, err := GetK8sIngresses(namespace)
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
		if strings.HasPrefix(item.Metadata.Name, ingressName) {
			// Extract IP or Hostname
			for _, ingress := range item.Status.LoadBalancer.Ingress {
				addresses = append(addresses, ingress.Hostname)
			}
		}
	}
	return addresses, nil
}
