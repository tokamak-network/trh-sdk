package thanos

import (
	"context"
	"fmt"
	"strings"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/digitalocean"
	"github.com/tokamak-network/trh-sdk/pkg/runner"
	"github.com/tokamak-network/trh-sdk/pkg/types"
	"github.com/tokamak-network/trh-sdk/pkg/utils"

	"go.uber.org/zap"
)

type ThanosStack struct {
	network           string
	deployConfig      *types.Config
	usePromptInput    bool
	awsProfile        *types.AWSProfile
	doProfile         *types.DigitalOceanProfile
	logger            *zap.SugaredLogger
	deploymentPath    string
	registerCandidate bool
	helmRunner        runner.HelmRunner // optional; when nil, falls back to shellout
	k8sRunner         runner.K8sRunner  // optional; when nil, falls back to shellout
}

// SetHelmRunner injects a HelmRunner for native Helm operations.
// When set, Helm calls use the runner instead of shelling out to the helm binary.
func (t *ThanosStack) SetHelmRunner(hr runner.HelmRunner) {
	t.helmRunner = hr
}

// SetK8sRunner injects a K8sRunner for native Kubernetes operations.
// When set, kubectl calls use the runner instead of shelling out to kubectl.
func (t *ThanosStack) SetK8sRunner(kr runner.K8sRunner) {
	t.k8sRunner = kr
}

// helmList returns all release names in a namespace. Uses HelmRunner when available.
func (t *ThanosStack) helmList(ctx context.Context, namespace string) ([]string, error) {
	if t.helmRunner != nil {
		return t.helmRunner.List(ctx, namespace)
	}
	return utils.GetHelmReleases(ctx, namespace)
}

// helmUninstall uninstalls a Helm release. Uses HelmRunner when available.
func (t *ThanosStack) helmUninstall(ctx context.Context, release, namespace string) error {
	if t.helmRunner != nil {
		return t.helmRunner.Uninstall(ctx, release, namespace)
	}
	_, err := utils.ExecuteCommand(ctx, "helm", "uninstall", release, "--namespace", namespace)
	return err
}

// helmInstallWithFiles installs a Helm chart using values files. Uses HelmRunner when available.
// NOTE: when HelmRunner is set, UpgradeWithFiles is used (upsert semantics) because the runner
// interface does not expose a pure-install method.
func (t *ThanosStack) helmInstallWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error {
	if len(valueFiles) == 0 {
		return fmt.Errorf("helmInstallWithFiles: valueFiles cannot be empty")
	}
	if t.helmRunner != nil {
		return t.helmRunner.UpgradeWithFiles(ctx, release, chart, namespace, valueFiles)
	}
	args := []string{"install", release, chart, "--values", valueFiles[0], "--namespace", namespace}
	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	return err
}

// helmUpgradeWithFiles upgrades a Helm release using values files. Uses HelmRunner when available.
func (t *ThanosStack) helmUpgradeWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string) error {
	if len(valueFiles) == 0 {
		return fmt.Errorf("helmUpgradeWithFiles: valueFiles cannot be empty")
	}
	if t.helmRunner != nil {
		return t.helmRunner.UpgradeWithFiles(ctx, release, chart, namespace, valueFiles)
	}
	args := []string{"upgrade", release, chart, "--values", valueFiles[0], "--namespace", namespace}
	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	return err
}

// helmDependencyUpdate updates chart dependencies. Uses HelmRunner when available.
func (t *ThanosStack) helmDependencyUpdate(ctx context.Context, chartPath string) error {
	if t.helmRunner != nil {
		return t.helmRunner.DependencyUpdate(ctx, chartPath)
	}
	_, err := utils.ExecuteCommand(ctx, "helm", "dependency", "update", chartPath)
	return err
}

// helmUpgradeInstallWithFiles performs helm upgrade --install using values files and extra args.
// Uses HelmRunner when available; extra args are only used in shellout mode.
func (t *ThanosStack) helmUpgradeInstallWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string, extraArgs ...string) error {
	if len(valueFiles) == 0 {
		return fmt.Errorf("helmUpgradeInstallWithFiles: valueFiles cannot be empty")
	}
	if t.helmRunner != nil {
		return t.helmRunner.UpgradeWithFiles(ctx, release, chart, namespace, valueFiles)
	}
	args := []string{"upgrade", "--install", release, chart, "--values", valueFiles[0], "--namespace", namespace}
	args = append(args, extraArgs...)
	_, err := utils.ExecuteCommand(ctx, "helm", args...)
	return err
}

// helmRepoAdd adds a Helm repository. Uses HelmRunner when available.
func (t *ThanosStack) helmRepoAdd(ctx context.Context, name, url string) error {
	if t.helmRunner != nil {
		return t.helmRunner.RepoAdd(ctx, name, url)
	}
	_, err := utils.ExecuteCommand(ctx, "helm", "repo", "add", name, url)
	return err
}

// helmSearch searches a Helm repository. Uses HelmRunner when available.
func (t *ThanosStack) helmSearch(ctx context.Context, keyword string) (string, error) {
	if t.helmRunner != nil {
		return t.helmRunner.Search(ctx, keyword)
	}
	return utils.ExecuteCommand(ctx, "helm", "search", "repo", keyword)
}

// helmFilterReleases lists releases and filters by name substring. Uses HelmRunner when available.
func (t *ThanosStack) helmFilterReleases(ctx context.Context, namespace, releaseName string) ([]string, error) {
	releases, err := t.helmList(ctx, namespace)
	if err != nil {
		return nil, err
	}
	filtered := make([]string, 0)
	for _, r := range releases {
		if strings.Contains(r, releaseName) {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

func NewThanosStack(
	ctx context.Context,
	l *zap.SugaredLogger,
	network string,
	usePromptInput bool,
	deploymentPath string,
	awsConfig *types.AWSConfig,
	doConfig *types.DigitalOceanConfig,
) (*ThanosStack, error) {
	l.Infof("Deployment Path: %s", deploymentPath)
	l.Infof("Network: %s", network)

	// get the config file
	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		l.Error("Error reading settings.json", "err", err)
		return nil, err
	}

	// Login AWS
	var awsProfile *types.AWSProfile

	if awsConfig != nil {
		if _, err := utils.SetAWSConfigFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS config file", "err", err)
			return nil, err
		}
		if _, err := utils.SetAWSCredentialsFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS credentials file", "err", err)
			return nil, err
		}
		if _, err := utils.SetKubeconfigFile(deploymentPath); err != nil {
			l.Error("Failed to set kubeconfig file", "err", err)
			return nil, err
		}

		awsProfile, err = aws.LoginAWS(ctx, awsConfig)
		if err != nil {
			l.Error("Failed to login aws", "err", err)
			return nil, err
		}

		// Switch to this context
		if config != nil && config.K8s != nil {
			err = utils.SwitchKubernetesContext(ctx, config.K8s.Namespace, awsConfig.Region)
			if err != nil {
				return nil, err
			}
		}
	}

	// Login DigitalOcean
	var doProfile *types.DigitalOceanProfile

	if doConfig != nil {
		if err := digitalocean.ValidateToken(ctx, doConfig.Token); err != nil {
			l.Error("Failed to validate DigitalOcean token", "err", err)
			return nil, err
		}
		doProfile = &types.DigitalOceanProfile{
			Config: doConfig,
		}
	}

	return &ThanosStack{
		network:        network,
		usePromptInput: usePromptInput,
		awsProfile:     awsProfile,
		doProfile:      doProfile,
		logger:         l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
	}, nil
}
