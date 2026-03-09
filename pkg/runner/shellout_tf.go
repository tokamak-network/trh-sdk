package runner

import (
	"context"
	"fmt"
	"io"

	"github.com/tokamak-network/trh-sdk/pkg/utils"
)

// ShellOutTFRunner implements TFRunner by shelling out to the terraform binary.
// It preserves the exact existing behaviour and serves as a fallback when
// NativeTFRunner is not available or when --legacy mode is requested.
type ShellOutTFRunner struct {
	stdout io.Writer
}

func (r *ShellOutTFRunner) SetStdout(w io.Writer) { r.stdout = w }

func (r *ShellOutTFRunner) Init(ctx context.Context, workDir string, env []string, backendConfigs []string) error {
	args := []string{"init"}
	for _, bc := range backendConfigs {
		args = append(args, "-backend-config="+bc)
	}
	if err := utils.ExecuteCommandStreamWithEnvInDir(ctx, nil, workDir, env, "terraform", args...); err != nil {
		return fmt.Errorf("shellout tf init %s: %w", workDir, err)
	}
	return nil
}

func (r *ShellOutTFRunner) Apply(ctx context.Context, workDir string, env []string) error {
	if err := utils.ExecuteCommandStreamWithEnvInDir(ctx, nil, workDir, env, "terraform", "apply", "-auto-approve"); err != nil {
		return fmt.Errorf("shellout tf apply %s: %w", workDir, err)
	}
	return nil
}

func (r *ShellOutTFRunner) Destroy(ctx context.Context, workDir string, env []string) error {
	if err := utils.ExecuteCommandStreamWithEnvInDir(ctx, nil, workDir, env, "terraform", "destroy", "-auto-approve"); err != nil {
		return fmt.Errorf("shellout tf destroy %s: %w", workDir, err)
	}
	return nil
}

func (r *ShellOutTFRunner) CheckVersion(ctx context.Context) error {
	_, err := utils.ExecuteCommand(ctx, "terraform", "version")
	if err != nil {
		return fmt.Errorf("shellout tf version: %w", err)
	}
	return nil
}
