package thanos

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
	"github.com/tokamak-network/trh-sdk/pkg/logging"
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

func (t *ThanosStack) ShowInformation(ctx context.Context) error {
	fileName := fmt.Sprintf("logs/show_information_%s_%s_%d.log", t.stack, t.network, time.Now().Unix())
	logging.InitLogger(fileName)

	if t.network == constants.LocalDevnet {
		// Check the devnet network running
		runningContainers, err := utils.GetDockerContainers(ctx)
		if err != nil {
			return fmt.Errorf("failed to get docker containers: %w", err)
		}
		if len(runningContainers) == 0 {
			fmt.Println("No running containers found. Please run the deploy command first")
			return nil
		}
		fmt.Println("✅ L1 and L2 networks are running on local devnet")
		fmt.Println("L1 network is running on http://localhost:8545")
		fmt.Println("L2 network is running on http://localhost:9545")
		return nil
	}

	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	namespace := t.deployConfig.K8s.Namespace

	// Step 1: Get pods
	runningPods, err := t.getRunningPods(ctx)
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

func (t *ThanosStack) ShowLogs(ctx context.Context, config *types.Config, component string, isTroubleshoot bool) error {
	fileName := fmt.Sprintf("logs/show_logs_%s_%s_%s_%d.log", t.stack, t.network, component, time.Now().Unix())
	logging.InitLogger(fileName)

	if config.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = config.K8s.Namespace
	)

	if !SupportedLogsComponents[component] {
		return fmt.Errorf("unsupported component: %s", component)
	}

	runningPods, err := t.getRunningPods(ctx)
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

	if isTroubleshoot {
		err = utils.ExecuteCommandStream("bash", "-c", fmt.Sprintf("kubectl -n %s logs %s -f | grep -iE 'error|fail|panic|critical'", namespace, runningPodName))
		if err != nil {
			fmt.Printf("failed to show logs: %s \n", err.Error())
			return err
		}
	} else {
		err = utils.ExecuteCommandStream("bash", "-c", fmt.Sprintf("kubectl -n %s logs %s -f", namespace, runningPodName))
		if err != nil {
			fmt.Printf("failed to show logs: %s \n", err.Error())
			return err
		}
	}

	return nil
}

func (t *ThanosStack) getRunningPods(ctx context.Context) ([]string, error) {
	if t.deployConfig.K8s == nil {
		return nil, fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	namespace := t.deployConfig.K8s.Namespace

	// Step 2: Get pods
	runningPods, err := utils.GetK8sPods(namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	return runningPods, nil
}
