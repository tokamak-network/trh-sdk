package thanos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// makeBlockExplorerEnvs is invoked after makeTerraformEnvFile has already
// produced an .envrc that exports TF_VAR_thanos_stack_name. The function must
// not regress that variable: block-explorer/variables.tf declares it as a
// required input and `terraform plan` runs without -input=false, so a missing
// value silently hangs on an interactive stdin prompt for hours.
//
// See trh-wiki: block-explorer-envrc-thanos-stack-name-stripped.
func TestMakeBlockExplorerEnvs_PreservesThanosStackName(t *testing.T) {
	tmp := t.TempDir()
	existing := strings.Join([]string{
		`export TF_VAR_thanos_stack_name="my-stack"`,
		`export TF_VAR_aws_region="ap-northeast-2"`,
		`export TF_VAR_vpc_name="${TF_VAR_thanos_stack_name}/VPC"`,
		``,
	}, "\n")
	if err := os.WriteFile(filepath.Join(tmp, ".envrc"), []byte(existing), 0o600); err != nil {
		t.Fatalf("seed envrc: %v", err)
	}

	t.Run("explicit StackName re-exports the variable", func(t *testing.T) {
		err := makeBlockExplorerEnvs(tmp, ".envrc", types.AwsDatabaseEnvs{
			DatabaseUserName: "blockscout",
			DatabasePassword: "pw",
			DatabaseName:     "blockscout",
			VpcId:            "vpc-1",
			AwsRegion:        "ap-northeast-2",
			StackName:        "my-stack",
		})
		if err != nil {
			t.Fatalf("makeBlockExplorerEnvs: %v", err)
		}
		body := readFile(t, filepath.Join(tmp, ".envrc"))
		if !strings.Contains(body, `export TF_VAR_thanos_stack_name="my-stack"`) {
			t.Fatalf("envrc must export TF_VAR_thanos_stack_name; got:\n%s", body)
		}
	})

	t.Run("empty StackName must not drop the existing export", func(t *testing.T) {
		// Re-seed because the previous subtest mutated the file.
		if err := os.WriteFile(filepath.Join(tmp, ".envrc"), []byte(existing), 0o600); err != nil {
			t.Fatalf("seed envrc: %v", err)
		}
		err := makeBlockExplorerEnvs(tmp, ".envrc", types.AwsDatabaseEnvs{
			DatabaseUserName: "blockscout",
			DatabasePassword: "pw",
			DatabaseName:     "blockscout",
			VpcId:            "vpc-1",
			AwsRegion:        "ap-northeast-2",
			// StackName intentionally empty — defensive guard.
		})
		if err != nil {
			t.Fatalf("makeBlockExplorerEnvs: %v", err)
		}
		body := readFile(t, filepath.Join(tmp, ".envrc"))
		if !strings.Contains(body, `export TF_VAR_thanos_stack_name="my-stack"`) {
			t.Fatalf("envrc dropped TF_VAR_thanos_stack_name; got:\n%s", body)
		}
	})
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}
