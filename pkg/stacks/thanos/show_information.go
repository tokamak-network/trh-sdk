package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

var SupportedLogsComponents = map[string]bool{
	"op-batcher":        true,
	"op-proposer":       true,
	"op-geth":           true,
	"op-node":           true,
	"block-explorer-fe": true,
	"block-explorer-be": true,
	"bridge":            true,
}

func (t *ThanosStack) ShowInformation(ctx context.Context, config *types.Config) error {
	namespace := config.K8s.Namespace

	// Step 1: Get pods
	runningPods, err := t.getRunningPods(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	status := map[string]bool{
		"chain":             false,
		"bridge":            false,
		"block-explorer-fe": false,
	}

	for _, pod := range runningPods {
		if strings.Contains(pod, namespace) {
			status["chain"] = true
		}
		if strings.Contains(pod, "bridge") {
			status["bridge"] = true
		}
		if strings.Contains(pod, "block-explorer-fe") {
			status["block-explorer-fe"] = true
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
		case strings.Contains(ingressName, "block-explorer-fe") && status["block-explorer-fe"]:
			fmt.Printf("✅ Block Explorer is running on http://%s\n", ingress)
		}
	}

	return nil
}

func (t *ThanosStack) ShowLogs(ctx context.Context, config *types.Config, component string) error {
	namespace := config.K8s.Namespace

	if !SupportedLogsComponents[component] {
		return fmt.Errorf("unsupported component: %s", component)
	}

	runningPods, err := t.getRunningPods(ctx, config)
	if err != nil {
		fmt.Printf("failed to get running pods: %s \n", err.Error())
		return err
	}

	var (
		runningPodName string
	)
	for _, pod := range runningPods {
		if !strings.Contains(pod, component) {
			continue
		}
		runningPodName = pod
	}

	err = utils.ExecuteCommandStream("bash", "-c", fmt.Sprintf("kubectl -n %s logs %s -f", namespace, runningPodName))
	if err != nil {
		fmt.Printf("failed to show logs: %s \n", err.Error())
		return err
	}

	return nil
}

func (t *ThanosStack) getRunningPods(ctx context.Context, config *types.Config) ([]string, error) {
	namespace := config.K8s.Namespace

	// Step 1: Login AWS
	if _, _, err := t.loginAWS(ctx, config); err != nil {
		return nil, fmt.Errorf("failed to login to AWS: %w", err)
	}

	// Step 2: Get pods
	runningPods, err := utils.GetK8sPods(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	return runningPods, nil
}
