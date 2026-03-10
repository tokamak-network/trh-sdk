package runner

import (
	"reflect"
	"testing"
)

// TestFlattenValues_Empty verifies nil is returned for an empty map.
func TestFlattenValues_Empty(t *testing.T) {
	if got := flattenValues(nil); got != nil {
		t.Fatalf("expected nil for nil map, got %v", got)
	}
	if got := flattenValues(map[string]interface{}{}); got != nil {
		t.Fatalf("expected nil for empty map, got %v", got)
	}
}

// TestFlattenValues_Deterministic verifies that keys are sorted and output is stable.
func TestFlattenValues_Deterministic(t *testing.T) {
	vals := map[string]interface{}{
		"replicaCount": 3,
		"image.tag":    "latest",
		"enabled":      true,
	}
	want := []string{
		"--set", "enabled=true",
		"--set", "image.tag=latest",
		"--set", "replicaCount=3",
	}
	// Run multiple times to catch non-determinism.
	for i := 0; i < 20; i++ {
		got := flattenValues(vals)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: got %v, want %v", i, got, want)
		}
	}
}

// TestFlattenValues_SingleEntry verifies a single key-value pair.
func TestFlattenValues_SingleEntry(t *testing.T) {
	vals := map[string]interface{}{"key": "value"}
	got := flattenValues(vals)
	want := []string{"--set", "key=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
