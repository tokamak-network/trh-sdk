package utils

import (
	"encoding/json"
	"fmt"
	"strings"
)

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
	return ExecuteCommand("kubectl", "-n", namespace, "get", "pods")
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
