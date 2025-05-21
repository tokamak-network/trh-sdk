package thanos

import (
	"context"
	"fmt"

	"github.com/tokamak-network/trh-sdk/pkg/constants"
)

// ------------------------------------------ Install plugins ---------------------------

func (t *ThanosStack) InstallPlugins(ctx context.Context, pluginNames []string) error {
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
	}

	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
			continue
		}

		fmt.Printf("Installing plugin: %s in namespace: %s...\n", pluginName, namespace)

		switch pluginName {
		case constants.PluginBlockExplorer:
			err := t.installBlockExplorer(ctx)
			if err != nil {
				return t.uninstallBlockExplorer(ctx)
			}
			return nil
		case constants.PluginBridge:
			err := t.installBridge(ctx)
			if err != nil {
				return t.uninstallBridge(ctx)
			}
			return nil
		}
	}
	return nil
}

// ------------------------------------------ Uninstall plugins ---------------------------

func (t *ThanosStack) UninstallPlugins(ctx context.Context, pluginNames []string) error {
	if t.network == constants.LocalDevnet {
		return fmt.Errorf("network %s does not support plugin installation", constants.LocalDevnet)
	}

	if t.deployConfig.K8s == nil {
		return fmt.Errorf("K8s configuration is not set. Please run the deploy command first")
	}

	var (
		namespace = t.deployConfig.K8s.Namespace
	)

	for _, pluginName := range pluginNames {
		if !constants.SupportedPlugins[pluginName] {
			fmt.Printf("Plugin %s is not supported for this stack.\n", pluginName)
			continue
		}

		fmt.Printf("Uninstalling plugin: %s in namespace: %s...\n", pluginName, namespace)

		switch pluginName {
		case constants.PluginBridge:
			return t.uninstallBridge(ctx)
		case constants.PluginBlockExplorer:
			return t.uninstallBlockExplorer(ctx)
		}
	}
	return nil
}
