package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

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

// newNativeTFRunner creates a NativeTFRunner by locating or installing terraform.
// hc-install v0.9+ moved the Installer type to the root package (install alias).
func newNativeTFRunner() (*NativeTFRunner, error) {
	// Try to find terraform in PATH first.
	if path, err := exec.LookPath("terraform"); err == nil {
		return &NativeTFRunner{execPath: path, stdout: os.Stdout}, nil
	}
	// Fall back to hc-install to download terraform.
	i := install.NewInstaller()
	execPath, err := i.Ensure(context.Background(), []src.Source{
		&releases.LatestVersion{
			Product:    product.Terraform,
			InstallDir: os.TempDir(),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("native tf: locate/install terraform: %w", err)
	}
	return &NativeTFRunner{execPath: execPath, stdout: os.Stdout}, nil
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
	tf, err := tfexec.NewTerraform(".", r.execPath)
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
