package thanos

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// Production terraform invocations from non-TTY contexts (trh-backend
// container, CI) must pass -input=false so a missing required variable or
// any other interactive prompt fails fast instead of sleeping on stdin
// indefinitely. We learned this the hard way: see trh-wiki
// `block-explorer-envrc-thanos-stack-name-stripped`. This guard scans the
// bash command literals (backtick raw-strings) in the production source
// files and rejects any `terraform <init|plan|apply>` line that omits
// the flag.
func TestTerraformInvocations_RequireInputFalse(t *testing.T) {
	files := []string{
		"block_explorer.go",
		"deploy_chain.go",
	}

	// Backtick-delimited Go raw strings hold the bash command templates.
	rawStringPattern := regexp.MustCompile("`[^`]*`")
	// Inside a bash command, match `terraform <verb>` up to the next && /
	// newline / closing backtick.
	verbPattern := regexp.MustCompile(`terraform (init|plan|apply)\b[^\n&]*`)

	for _, f := range files {
		t.Run(f, func(t *testing.T) {
			b, err := os.ReadFile(f)
			if err != nil {
				t.Fatalf("read %s: %v", f, err)
			}

			rawStrings := rawStringPattern.FindAllString(string(b), -1)
			if len(rawStrings) == 0 {
				t.Fatalf("%s: expected backtick command literals, found none", f)
			}

			var found int
			for _, raw := range rawStrings {
				for _, m := range verbPattern.FindAllString(raw, -1) {
					found++
					if !strings.Contains(m, "-input=false") {
						t.Errorf("%s: terraform invocation missing -input=false: %q", f, strings.TrimSpace(m))
					}
				}
			}
			if found == 0 {
				t.Fatalf("%s: no terraform init/plan/apply invocations found in bash literals", f)
			}
		})
	}
}
