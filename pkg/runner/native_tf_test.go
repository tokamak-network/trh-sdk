package runner

import (
	"testing"
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
