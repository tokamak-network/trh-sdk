package runner

import (
	"context"
	"io"
)

// TFRunner defines Terraform operations used across TRH SDK.
// It replaces terraform subprocess calls (init, apply, destroy).
//
// NativeTFRunner uses github.com/hashicorp/terraform-exec/tfexec;
// ShellOutTFRunner shells out to the terraform binary as a fallback.
type TFRunner interface {
	// Init runs terraform init in the given working directory.
	// backendConfigs are key=value pairs passed via -backend-config.
	Init(ctx context.Context, workDir string, env []string, backendConfigs []string) error

	// Apply runs terraform apply -auto-approve in the given working directory.
	Apply(ctx context.Context, workDir string, env []string) error

	// Destroy runs terraform destroy -auto-approve in the given working directory.
	Destroy(ctx context.Context, workDir string, env []string) error

	// SetStdout sets the writer for terraform stdout streaming.
	SetStdout(w io.Writer)

	// CheckVersion verifies terraform is available (for legacy/shellout mode).
	CheckVersion(ctx context.Context) error
}
