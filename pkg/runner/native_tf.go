package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	version "github.com/hashicorp/go-version"
	install "github.com/hashicorp/hc-install"
	"github.com/hashicorp/hc-install/product"
	"github.com/hashicorp/hc-install/releases"
	"github.com/hashicorp/hc-install/src"
	"github.com/hashicorp/terraform-exec/tfexec"
)

// NativeTFRunner implements TFRunner using the terraform-exec library.
// No terraform binary installation is required — hc-install locates or
// downloads the binary automatically.
type NativeTFRunner struct {
	execPath string
	stdout   io.Writer
}

// terraformVersion pins the exact Terraform binary for reproducible deploys.
// Verified against TRH integration tests as of 2026-03. Bump requires re-testing.
const terraformVersion = "1.9.8"

// execLookPath and tfCheckVersion are package-level vars so unit tests can inject
// fakes without spawning real processes.
var (
	execLookPath = exec.LookPath

	// tfCheckVersion returns the version reported by the terraform binary at path.
	tfCheckVersion = func(ctx context.Context, path string) (*version.Version, error) {
		tf, err := tfexec.NewTerraform(os.TempDir(), path)
		if err != nil {
			return nil, err
		}
		v, _, err := tf.Version(ctx, false)
		return v, err
	}
)

// newNativeTFRunner creates a NativeTFRunner by locating or installing terraform.
// hc-install v0.9+ moved the Installer type to the root package (install alias).
// ctx is forwarded to hc-install so callers can cancel long downloads.
func newNativeTFRunner(ctx context.Context) (*NativeTFRunner, error) {
	pinnedVersion, err := version.NewVersion(terraformVersion)
	if err != nil {
		return nil, fmt.Errorf("native tf: invalid terraform version %q: %w", terraformVersion, err)
	}

	// Try to find a matching terraform version in PATH first.
	if path := findPinnedTerraformInPath(ctx, pinnedVersion); path != "" {
		return &NativeTFRunner{execPath: path, stdout: os.Stdout}, nil
	}

	// Fall back to hc-install to download the pinned terraform version.
	i := install.NewInstaller()
	execPath, err := i.Ensure(ctx, []src.Source{
		&releases.ExactVersion{
			Product:    product.Terraform,
			Version:    pinnedVersion,
			InstallDir: os.TempDir(),
		},
	})
	if err != nil {
		_ = i.Remove(context.Background())
		return nil, fmt.Errorf("native tf: locate/install terraform %s: %w", terraformVersion, err)
	}
	return &NativeTFRunner{execPath: execPath, stdout: os.Stdout}, nil
}

// findPinnedTerraformInPath looks for a terraform binary in PATH whose version
// matches pinnedVersion. Returns the binary path on a match, or "" otherwise.
// A diagnostic is written to stderr when a binary is found but skipped, so
// operators know why hc-install was invoked.
func findPinnedTerraformInPath(ctx context.Context, pinnedVersion *version.Version) string {
	path, lookErr := execLookPath("terraform")
	if lookErr != nil {
		return ""
	}

	v, vErr := tfCheckVersion(ctx, path)
	if vErr != nil || !v.Equal(pinnedVersion) {
		var reason string
		if vErr != nil {
			reason = vErr.Error()
		} else {
			reason = fmt.Sprintf("version %s != pinned %s", v, pinnedVersion)
		}
		fmt.Fprintf(os.Stderr, "native tf: PATH terraform skipped (%s) — falling back to hc-install\n", reason)
		return ""
	}
	return path
}

func (r *NativeTFRunner) SetStdout(w io.Writer) { r.stdout = w }

func (r *NativeTFRunner) Init(ctx context.Context, workDir string, env []string, backendConfigs []string) error {
	tf, err := tfexec.NewTerraform(workDir, r.execPath)
	if err != nil {
		return fmt.Errorf("native tf init: new terraform: %w", err)
	}
	if err := tf.SetEnv(envSliceToMap(env)); err != nil {
		return fmt.Errorf("native tf init: set env: %w", err)
	}
	if r.stdout != nil {
		tf.SetStdout(r.stdout)
	}

	var opts []tfexec.InitOption
	for _, bc := range backendConfigs {
		opts = append(opts, tfexec.BackendConfig(bc))
	}
	if err := tf.Init(ctx, opts...); err != nil {
		return fmt.Errorf("native tf init %s: %w", workDir, err)
	}
	return nil
}

func (r *NativeTFRunner) Apply(ctx context.Context, workDir string, env []string) error {
	tf, err := tfexec.NewTerraform(workDir, r.execPath)
	if err != nil {
		return fmt.Errorf("native tf apply: new terraform: %w", err)
	}
	if err := tf.SetEnv(envSliceToMap(env)); err != nil {
		return fmt.Errorf("native tf apply: set env: %w", err)
	}
	if r.stdout != nil {
		tf.SetStdout(r.stdout)
	}
	if err := tf.Apply(ctx); err != nil {
		return fmt.Errorf("native tf apply %s: %w", workDir, err)
	}
	return nil
}

func (r *NativeTFRunner) Destroy(ctx context.Context, workDir string, env []string) error {
	tf, err := tfexec.NewTerraform(workDir, r.execPath)
	if err != nil {
		return fmt.Errorf("native tf destroy: new terraform: %w", err)
	}
	if err := tf.SetEnv(envSliceToMap(env)); err != nil {
		return fmt.Errorf("native tf destroy: set env: %w", err)
	}
	if r.stdout != nil {
		tf.SetStdout(r.stdout)
	}
	if err := tf.Destroy(ctx); err != nil {
		return fmt.Errorf("native tf destroy %s: %w", workDir, err)
	}
	return nil
}

func (r *NativeTFRunner) CheckVersion(ctx context.Context) error {
	tf, err := tfexec.NewTerraform(os.TempDir(), r.execPath)
	if err != nil {
		return fmt.Errorf("native tf version: %w", err)
	}
	_, _, err = tf.Version(ctx, false)
	return err
}

// envSliceToMap converts []string{"KEY=VALUE"} to map[string]string{"KEY":"VALUE"}.
func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		idx := strings.Index(e, "=")
		if idx < 0 {
			m[e] = ""
			continue
		}
		m[e[:idx]] = e[idx+1:]
	}
	return m
}
