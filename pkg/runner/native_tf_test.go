package runner

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	version "github.com/hashicorp/go-version"
)

// ─── envSliceToMap ────────────────────────────────────────────────────────

func TestEnvSliceToMap_Normal(t *testing.T) {
	m := envSliceToMap([]string{"KEY=VALUE", "FOO=BAR"})
	if m["KEY"] != "VALUE" {
		t.Fatalf("expected KEY=VALUE, got %q", m["KEY"])
	}
	if m["FOO"] != "BAR" {
		t.Fatalf("expected FOO=BAR, got %q", m["FOO"])
	}
}

// TestEnvSliceToMap_EmptyValue verifies that "KEY=" produces an empty string value.
func TestEnvSliceToMap_EmptyValue(t *testing.T) {
	m := envSliceToMap([]string{"KEY="})
	v, ok := m["KEY"]
	if !ok {
		t.Fatal("expected KEY to be present")
	}
	if v != "" {
		t.Fatalf("expected empty value, got %q", v)
	}
}

// TestEnvSliceToMap_ValueContainsEquals verifies that only the first '=' is used
// as the delimiter, preserving '=' characters in the value.
func TestEnvSliceToMap_ValueContainsEquals(t *testing.T) {
	m := envSliceToMap([]string{"URL=http://example.com?a=1&b=2"})
	if m["URL"] != "http://example.com?a=1&b=2" {
		t.Fatalf("expected full URL as value, got %q", m["URL"])
	}
}

// TestEnvSliceToMap_NoEquals verifies that entries without '=' are treated as
// keys with an empty value (consistent with os.Getenv behaviour for bare names).
func TestEnvSliceToMap_NoEquals(t *testing.T) {
	m := envSliceToMap([]string{"BARE"})
	v, ok := m["BARE"]
	if !ok {
		t.Fatal("expected BARE to be present")
	}
	if v != "" {
		t.Fatalf("expected empty value for bare key, got %q", v)
	}
}

// TestEnvSliceToMap_Empty verifies that an empty slice produces an empty map.
func TestEnvSliceToMap_Empty(t *testing.T) {
	m := envSliceToMap(nil)
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %v", m)
	}
}

// ─── findPinnedTerraformInPath ────────────────────────────────────────────
//
// These tests swap the injectable package vars to avoid real process execution.

// mustParseVersion parses a version string and fails the test on error.
func mustParseVersion(t *testing.T, v string) *version.Version {
	t.Helper()
	parsed, err := version.NewVersion(v)
	if err != nil {
		t.Fatalf("mustParseVersion(%q): %v", v, err)
	}
	return parsed
}

// captureStderr redirects os.Stderr to a buffer for the duration of fn,
// returning whatever was written to stderr.
func captureStderr(fn func()) string {
	r, w, err := os.Pipe()
	if err != nil {
		panic("captureStderr: os.Pipe: " + err.Error())
	}
	old := os.Stderr
	os.Stderr = w
	fn()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck // pipe read never fails after write end closed
	return buf.String()
}

func TestFindPinnedTerraformInPath_NotFound(t *testing.T) {
	orig := execLookPath
	t.Cleanup(func() { execLookPath = orig })
	execLookPath = func(string) (string, error) {
		return "", errors.New("terraform: executable file not found in $PATH")
	}

	pinnedVersion := mustParseVersion(t, terraformVersion)
	if got := findPinnedTerraformInPath(context.Background(), pinnedVersion); got != "" {
		t.Fatalf("expected empty path when terraform not in PATH, got %q", got)
	}
}

func TestFindPinnedTerraformInPath_VersionMatch(t *testing.T) {
	origLook := execLookPath
	origCheck := tfCheckVersion
	t.Cleanup(func() {
		execLookPath = origLook
		tfCheckVersion = origCheck
	})

	const fakePath = "/usr/local/bin/terraform"
	execLookPath = func(string) (string, error) { return fakePath, nil }

	pinnedVersion := mustParseVersion(t, terraformVersion)
	tfCheckVersion = func(_ context.Context, _ string) (*version.Version, error) {
		return pinnedVersion, nil
	}

	if got := findPinnedTerraformInPath(context.Background(), pinnedVersion); got != fakePath {
		t.Fatalf("expected %q, got %q", fakePath, got)
	}
}

// TestFindPinnedTerraformInPath_VersionMismatch verifies that a PATH terraform
// with a different version is skipped (returns "") and emits a stderr diagnostic
// containing the reason.
func TestFindPinnedTerraformInPath_VersionMismatch(t *testing.T) {
	origLook := execLookPath
	origCheck := tfCheckVersion
	t.Cleanup(func() {
		execLookPath = origLook
		tfCheckVersion = origCheck
	})

	execLookPath = func(string) (string, error) { return "/usr/local/bin/terraform", nil }

	differentVersion := mustParseVersion(t, "1.8.0")
	tfCheckVersion = func(_ context.Context, _ string) (*version.Version, error) {
		return differentVersion, nil
	}

	pinnedVersion := mustParseVersion(t, terraformVersion)
	var got string
	stderr := captureStderr(func() {
		got = findPinnedTerraformInPath(context.Background(), pinnedVersion)
	})

	if got != "" {
		t.Fatalf("expected empty path on version mismatch, got %q", got)
	}
	if !strings.Contains(stderr, "PATH terraform skipped") {
		t.Fatalf("expected stderr diagnostic, got: %q", stderr)
	}
}

// TestFindPinnedTerraformInPath_VersionCheckError verifies that a version-check
// failure (e.g. binary not executable) causes the PATH branch to be skipped.
func TestFindPinnedTerraformInPath_VersionCheckError(t *testing.T) {
	origLook := execLookPath
	origCheck := tfCheckVersion
	t.Cleanup(func() {
		execLookPath = origLook
		tfCheckVersion = origCheck
	})

	execLookPath = func(string) (string, error) { return "/usr/local/bin/terraform", nil }
	tfCheckVersion = func(_ context.Context, _ string) (*version.Version, error) {
		return nil, errors.New("version check failed: binary not executable")
	}

	pinnedVersion := mustParseVersion(t, terraformVersion)
	if got := findPinnedTerraformInPath(context.Background(), pinnedVersion); got != "" {
		t.Fatalf("expected empty path on version check error, got %q", got)
	}
}
