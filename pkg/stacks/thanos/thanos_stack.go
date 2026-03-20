package thanos

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/aws"
	"github.com/tokamak-network/trh-sdk/pkg/cloud-provider/digitalocean"
	"github.com/tokamak-network/trh-sdk/pkg/constants"
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
	tfRunner          runner.TFRunner   // optional; when nil, falls back to shellout
	awsRunner         runner.AWSRunner  // optional; when nil, falls back to shellout
}

// isLocal returns true when the stack targets a local kind cluster rather than cloud infrastructure.
func (t *ThanosStack) isLocal() bool {
	return t.network == constants.LocalTestnet
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

// SetTFRunner injects a TFRunner for native Terraform operations.
func (t *ThanosStack) SetTFRunner(tr runner.TFRunner) {
	t.tfRunner = tr
}

// SetAWSRunner injects an AWSRunner for native AWS operations.
func (t *ThanosStack) SetAWSRunner(ar runner.AWSRunner) {
	t.awsRunner = ar
}

// maxLogBytes caps the amount of log data read into memory by PodLogs.
const maxLogBytes = 100 << 20 // 100 MiB

// PodLogs reads logs from the named pod. Uses K8sRunner when since is zero;
// falls back to shell-out when since > 0 because K8sRunner.Logs does not
// support time-windowed retrieval. container may be empty to select the
// pod's first container.
func (t *ThanosStack) PodLogs(ctx context.Context, pod, namespace, container string, since time.Duration) ([]byte, error) {
	if t.k8sRunner != nil && since == 0 {
		rc, err := t.k8sRunner.Logs(ctx, pod, namespace, container, false)
		if err != nil {
			return nil, fmt.Errorf("pod logs %s/%s: %w", namespace, pod, err)
		}
		defer rc.Close() //nolint:errcheck
		return io.ReadAll(io.LimitReader(rc, maxLogBytes))
	}
	args := []string{"logs", pod, "-n", namespace}
	if since > 0 {
		args = append(args, "--since", since.String())
	}
	raw, err := exec.CommandContext(ctx, "kubectl", args...).CombinedOutput()
	if int64(len(raw)) > maxLogBytes {
		raw = raw[:maxLogBytes]
	}
	return raw, err
}

// tfInit runs terraform init in workDir. Uses TFRunner when available.
func (t *ThanosStack) tfInit(ctx context.Context, workDir string, env []string, backendConfigs []string) error {
	if t.tfRunner != nil {
		return t.tfRunner.Init(ctx, workDir, env, backendConfigs)
	}
	args := []string{"init"}
	for _, bc := range backendConfigs {
		args = append(args, "-backend-config="+bc)
	}
	return utils.ExecuteCommandStreamWithEnvInDir(ctx, t.logger, workDir, env, "terraform", args...)
}

// tfApply runs terraform apply -auto-approve in workDir. Uses TFRunner when available.
func (t *ThanosStack) tfApply(ctx context.Context, workDir string, env []string) error {
	if t.tfRunner != nil {
		return t.tfRunner.Apply(ctx, workDir, env)
	}
	return utils.ExecuteCommandStreamWithEnvInDir(ctx, t.logger, workDir, env, "terraform", "apply", "-auto-approve")
}

// tfDestroy runs terraform destroy -auto-approve in workDir. Uses TFRunner when available.
func (t *ThanosStack) tfDestroy(ctx context.Context, workDir string, env []string) error {
	if t.tfRunner != nil {
		return t.tfRunner.Destroy(ctx, workDir, env)
	}
	return utils.ExecuteCommandStreamWithEnvInDir(ctx, t.logger, workDir, env, "terraform", "destroy", "-auto-approve")
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
// When HelmRunner is set, extraArgs are not supported and must be empty.
func (t *ThanosStack) helmUpgradeInstallWithFiles(ctx context.Context, release, chart, namespace string, valueFiles []string, extraArgs ...string) error {
	if len(valueFiles) == 0 {
		return fmt.Errorf("helmUpgradeInstallWithFiles: valueFiles cannot be empty")
	}
	if t.helmRunner != nil {
		if len(extraArgs) > 0 {
			return fmt.Errorf("helmUpgradeInstallWithFiles: extraArgs %v not supported with helmRunner; remove extra args or use shellout mode", extraArgs)
		}
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
	var kubeconfigPath string

	if awsConfig != nil {
		if _, err := utils.SetAWSConfigFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS config file", "err", err)
			return nil, err
		}
		if _, err := utils.SetAWSCredentialsFile(deploymentPath); err != nil {
			l.Error("Failed to set AWS credentials file", "err", err)
			return nil, err
		}
		kubeconfigPath, err = utils.SetKubeconfigFile(deploymentPath)
		if err != nil {
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

	stack := &ThanosStack{
		network:        network,
		usePromptInput: usePromptInput,
		awsProfile:     awsProfile,
		doProfile:      doProfile,
		logger:         l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
	}

	// Only attempt runner wiring when an infra provider is configured.
	// Callers that pass nil/nil (e.g. shutdown introspection) do not need runners.
	if awsConfig != nil || doConfig != nil {
		injectRunners(stack, l, kubeconfigPath)
	}

	return stack, nil
}

// NewLocalTestnetThanosStack creates a ThanosStack for a local kind-cluster deployment.
// It reads settings.json when available (nil config is tolerated for fresh deployments),
// wires native runners with the caller-supplied kubeconfigPath, and skips all AWS/DO setup.
func NewLocalTestnetThanosStack(
	ctx context.Context,
	l *zap.SugaredLogger,
	deploymentPath string,
	kubeconfigPath string,
) (*ThanosStack, error) {
	l.Infof("Deployment Path: %s", deploymentPath)
	l.Infof("Network: LocalTestnet")

	config, err := utils.ReadConfigFromJSONFile(deploymentPath)
	if err != nil {
		l.Error("Error reading settings.json", "err", err)
		return nil, err
	}

	stack := &ThanosStack{
		network:        "LocalTestnet",
		usePromptInput: false,
		logger:         l,
		deploymentPath: deploymentPath,
		deployConfig:   config,
	}

	injectRunners(stack, l, kubeconfigPath)

	return stack, nil
}

// injectRunners initialises native runners and injects them into stack.
// On failure it logs a warning and leaves all runner fields nil so that
// each helper method falls back to shell-out transparently.
func injectRunners(stack *ThanosStack, l *zap.SugaredLogger, kubeconfigPath string) {
	tr, err := runner.New(runner.RunnerConfig{UseNative: true, KubeconfigPath: kubeconfigPath})
	if err != nil {
		l.Warnf("Native runner init failed, falling back to shell-out: %v", err)
		return
	}
	stack.helmRunner = tr.Helm()
	stack.k8sRunner = tr.K8s()
	stack.tfRunner = tr.TF()
	stack.awsRunner = tr.AWS()
}
