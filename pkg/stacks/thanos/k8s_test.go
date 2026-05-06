package thanos

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildNamespaceFinalizeBody(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name: "clears existing kubernetes finalizer",
			input: `{
				"apiVersion":"v1",
				"kind":"Namespace",
				"metadata":{"name":"foo"},
				"spec":{"finalizers":["kubernetes"]},
				"status":{"phase":"Terminating"}
			}`,
		},
		{
			name: "no-op when spec has no finalizers",
			input: `{
				"apiVersion":"v1",
				"kind":"Namespace",
				"metadata":{"name":"bar"},
				"spec":{},
				"status":{"phase":"Active"}
			}`,
		},
		{
			name: "tolerates missing spec",
			input: `{
				"apiVersion":"v1",
				"kind":"Namespace",
				"metadata":{"name":"baz"},
				"status":{"phase":"Active"}
			}`,
		},
		{
			name:    "rejects invalid json",
			input:   `not json`,
			wantErr: true,
		},
		{
			name: "stuck namespace with real conditions array",
			input: `{
				"apiVersion":"v1",
				"kind":"Namespace",
				"metadata":{"name":"foo"},
				"spec":{"finalizers":["kubernetes"]},
				"status":{
					"phase":"Terminating",
					"conditions":[
						{"type":"NamespaceContentRemaining","status":"True"},
						{"type":"NamespaceFinalizersRemaining","status":"True"}
					]
				}
			}`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out, err := buildNamespaceFinalizeBody([]byte(tc.input))
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}

			var got map[string]interface{}
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("output is not valid namespace JSON: %v", err)
			}

			spec, ok := got["spec"].(map[string]interface{})
			if !ok {
				t.Fatalf("spec missing or wrong type in output: %s", out)
			}
			arr, ok := spec["finalizers"].([]interface{})
			if !ok {
				t.Fatalf("finalizers type = %T, want []interface{}", spec["finalizers"])
			}
			if len(arr) != 0 {
				t.Fatalf("finalizers len = %d, want 0", len(arr))
			}
		})
	}
}

func TestBuildNamespaceFinalizeBody_PreservesIdentity(t *testing.T) {
	// The /finalize subresource validates apiVersion/kind/metadata.name —
	// our transform must not drop them.
	input := `{
		"apiVersion":"v1",
		"kind":"Namespace",
		"metadata":{"name":"keep-me","uid":"abc"},
		"spec":{"finalizers":["kubernetes"]},
		"status":{"phase":"Terminating"}
	}`

	out, err := buildNamespaceFinalizeBody([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	body := string(out)
	for _, want := range []string{`"apiVersion": "v1"`, `"kind": "Namespace"`, `keep-me`} {
		if !strings.Contains(body, want) {
			t.Fatalf("output missing %q:\n%s", want, body)
		}
	}
}

func TestExtractNamespacePhase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Active",
			input: `{"status":{"phase":"Active"}}`,
			want:  "Active",
		},
		{
			name:  "Terminating with conditions array",
			input: `{"status":{"phase":"Terminating","conditions":[{"type":"NamespaceContentRemaining"}]}}`,
			want:  "Terminating",
		},
		{
			name:  "missing status",
			input: `{"metadata":{"name":"x"}}`,
			want:  "",
		},
		{
			name:  "invalid json",
			input: `bad`,
			want:  "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractNamespacePhase([]byte(tc.input)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
