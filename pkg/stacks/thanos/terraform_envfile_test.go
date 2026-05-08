package thanos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tokamak-network/trh-sdk/pkg/types"
)

// TestMakeTerraformEnvFile_WritesBackendBucketName verifies that makeTerraformEnvFile
// writes config.BackendBucketName to TF_VAR_backend_bucket_name, not a hardcoded "".
//
// Stage A legitimately passes "" (bucket doesn't exist yet; bucket_name.sh fills it in later).
// Stage B must pass the real bucket name so terraform init reuses the same S3 backend.
func TestMakeTerraformEnvFile_WritesBackendBucketName(t *testing.T) {
	minimalConfig := func(bucket string) types.TerraformEnvConfig {
		return types.TerraformEnvConfig{
			Namespace:          "full-eu5up",
			AwsRegion:          "ap-northeast-2",
			BackendBucketName:  bucket,
			EksClusterAdmins:   "arn:aws:iam::123456789012:user/test",
			Azs:                []string{"ap-northeast-2a"},
			SequencerKey:       "deadbeef",
			BatcherKey:         "deadbeef",
			ProposerKey:        "deadbeef",
			ChallengerKey:      "deadbeef",
			MaxChannelDuration: 100,
		}
	}

	t.Run("stage A: empty bucket writes empty value", func(t *testing.T) {
		tmp := t.TempDir()
		if err := makeTerraformEnvFile(tmp, minimalConfig("")); err != nil {
			t.Fatalf("makeTerraformEnvFile: %v", err)
		}
		body := readFile(t, filepath.Join(tmp, ".envrc"))
		if !strings.Contains(body, `export TF_VAR_backend_bucket_name=""`) {
			t.Fatalf("expected empty bucket name in envrc; got:\n%s", body)
		}
	})

	t.Run("stage B: real bucket name is written verbatim", func(t *testing.T) {
		const realBucket = "full-eu5up-thanos-stack-tfstate-flea1wn6"
		tmp := t.TempDir()
		if err := makeTerraformEnvFile(tmp, minimalConfig(realBucket)); err != nil {
			t.Fatalf("makeTerraformEnvFile: %v", err)
		}
		body := readFile(t, filepath.Join(tmp, ".envrc"))
		want := `export TF_VAR_backend_bucket_name="` + realBucket + `"`
		if !strings.Contains(body, want) {
			t.Fatalf("expected %q in envrc; got:\n%s", want, body)
		}
	})
}

// TestReadBackendStateFromTfstate verifies that readBackendStateFromTfstate correctly
// parses .terraform/terraform.tfstate and extracts the S3 bucket name and derives the namespace.
func TestReadBackendStateFromTfstate(t *testing.T) {
	makeTfstate := func(bucket string) []byte {
		state := map[string]any{
			"version":           4,
			"terraform_version": "1.5.0",
			"backend": map[string]any{
				"type": "s3",
				"config": map[string]any{
					"bucket": bucket,
					"region": "ap-northeast-2",
					"key":    "terraform.tfstate",
				},
				"hash": 133827910,
			},
		}
		b, _ := json.Marshal(state)
		return b
	}

	writeTfstate := func(t *testing.T, terraformDir string, content []byte) {
		t.Helper()
		dotTf := filepath.Join(terraformDir, "thanos-stack", ".terraform")
		if err := os.MkdirAll(dotTf, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dotTf, "terraform.tfstate"), content, 0o644); err != nil {
			t.Fatalf("write tfstate: %v", err)
		}
	}

	t.Run("parses bucket and derives namespace", func(t *testing.T) {
		tmp := t.TempDir()
		writeTfstate(t, tmp, makeTfstate("full-eu5up-thanos-stack-tfstate-flea1wn6"))

		namespace, bucket, err := readBackendStateFromTfstate(tmp)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if namespace != "full-eu5up" {
			t.Errorf("namespace = %q, want %q", namespace, "full-eu5up")
		}
		if bucket != "full-eu5up-thanos-stack-tfstate-flea1wn6" {
			t.Errorf("bucket = %q, want %q", bucket, "full-eu5up-thanos-stack-tfstate-flea1wn6")
		}
	})

	t.Run("missing tfstate returns error", func(t *testing.T) {
		tmp := t.TempDir()
		_, _, err := readBackendStateFromTfstate(tmp)
		if err == nil {
			t.Fatal("expected error for missing tfstate file")
		}
	})

	t.Run("empty bucket in tfstate returns error", func(t *testing.T) {
		tmp := t.TempDir()
		writeTfstate(t, tmp, makeTfstate(""))

		_, _, err := readBackendStateFromTfstate(tmp)
		if err == nil {
			t.Fatal("expected error for empty bucket")
		}
	})

	t.Run("bucket without thanos-stack-tfstate separator returns error", func(t *testing.T) {
		tmp := t.TempDir()
		writeTfstate(t, tmp, makeTfstate("some-other-bucket"))

		_, _, err := readBackendStateFromTfstate(tmp)
		if err == nil {
			t.Fatal("expected error for unrecognized bucket format")
		}
	})
}
