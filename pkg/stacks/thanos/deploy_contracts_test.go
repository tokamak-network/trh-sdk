package thanos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Sample Makefile snippet that contains the cannon-prestate patterns we need to patch.
const testMakefileContent = `.PHONY: cannon-prestate
cannon-prestate:
	cd cannon && make cannon64-impl
	cannon/bin/cannon load-elf --path op-program/bin/op-program-client.elf --out op-program/bin/prestate.json
	cannon/bin/cannon run --proof-at '%(%)' --stop-at '=%(%)' --input op-program/bin/prestate.json --meta "" --proof-fmt 'op-program/bin/%d.json' --output ""
`

func TestPatchMakefileCannonPrestate_SuccessfulPatch(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")
	if err := os.WriteFile(makefilePath, []byte(testMakefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	if err := patchMakefileCannonPrestate(dir); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	data, err := os.ReadFile(makefilePath)
	if err != nil {
		t.Fatal(err)
	}
	patched := string(data)

	// Verify old patterns are gone
	if strings.Contains(patched, "cannon/bin/cannon load-elf") {
		t.Error("old load-elf pattern still present after patching")
	}
	if strings.Contains(patched, "cannon/bin/cannon run") {
		t.Error("old run pattern still present after patching")
	}

	// Verify new patterns are present
	if !strings.Contains(patched, "cannon64-impl load-elf --type multithreaded64-5") {
		t.Error("new load-elf pattern not found")
	}
	if !strings.Contains(patched, "cannon64-impl run --type multithreaded64-5") {
		t.Error("new run pattern not found")
	}
	if !strings.Contains(patched, "cp op-program/bin/0.json op-program/bin/prestate-proof.json") {
		t.Error("cp prestate-proof command not found")
	}
	if !strings.Contains(patched, "cp op-program/bin/prestate-proof.json op-program/bin/prestate.json") {
		t.Error("cp prestate.json command not found")
	}
}

func TestPatchMakefileCannonPrestate_Idempotent(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")
	if err := os.WriteFile(makefilePath, []byte(testMakefileContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Patch once
	if err := patchMakefileCannonPrestate(dir); err != nil {
		t.Fatalf("first patch failed: %v", err)
	}
	first, _ := os.ReadFile(makefilePath)

	// Patch again — should be a no-op
	if err := patchMakefileCannonPrestate(dir); err != nil {
		t.Fatalf("second patch should succeed (idempotent), got: %v", err)
	}
	second, _ := os.ReadFile(makefilePath)

	if string(first) != string(second) {
		t.Error("second patch modified the file — not idempotent")
	}
}

func TestPatchMakefileCannonPrestate_NoMatchReturnsError(t *testing.T) {
	dir := t.TempDir()
	makefilePath := filepath.Join(dir, "Makefile")
	// Makefile with no matching patterns
	if err := os.WriteFile(makefilePath, []byte("all:\n\techo hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	err := patchMakefileCannonPrestate(dir)
	if err == nil {
		t.Fatal("expected error for non-matching Makefile, got nil")
	}
	if !strings.Contains(err.Error(), "no patterns matched") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestPatchMakefileCannonPrestate_MissingMakefile(t *testing.T) {
	dir := t.TempDir()
	err := patchMakefileCannonPrestate(dir)
	if err == nil {
		t.Fatal("expected error for missing Makefile, got nil")
	}
}
