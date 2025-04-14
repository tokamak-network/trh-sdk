package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

func (t *ThanosStack) ShowInformation(ctx context.Context, config *types.Config) error {
	namespace := config.K8s.Namespace

	// Step 1: Login AWS
	if _, _, err := t.loginAWS(ctx, config); err != nil {
		return fmt.Errorf("failed to login to AWS: %w", err)
	}

	// Step 2: Get pods
	runningPods, err := utils.GetK8sPods(namespace)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	status := map[string]bool{
		"chain":          false,
		"bridge":         false,
		"block-explorer": false,
	}

	for _, pod := range runningPods {
		if strings.Contains(pod, namespace) {
			status["chain"] = true
		}
		if strings.Contains(pod, "bridge") {
			status["bridge"] = true
		}
		if strings.Contains(pod, "block-explorer") {
			status["block-explorer"] = true
		}
	}

	ingresses, err := utils.GetIngresses(namespace)
	if err != nil {
		return fmt.Errorf("failed to get ingresses: %w", err)
	}

	for ingressName, addresses := range ingresses {
		if len(addresses) == 0 {
			continue
		}
		ingress := addresses[0]
		switch {
		case strings.Contains(ingressName, namespace) && status["chain"]:
			fmt.Printf("✅ L2 network is running on http://%s\n", ingress)
		case strings.Contains(ingressName, "bridge") && status["bridge"]:
			fmt.Printf("✅ Bridge is running on http://%s\n", ingress)
		case strings.Contains(ingressName, "block-explorer") && status["block-explorer"]:
			fmt.Printf("✅ Block Explorer is running on http://%s\n", ingress)
		}
	}

	return nil
}
