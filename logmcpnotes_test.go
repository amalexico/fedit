package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLMN_MCPReplaceAllNotes(t *testing.T) {
	cases := []struct {
		name string
		args func(f string) map[string]any
	}{
		{"literal", func(f string) map[string]any {
			return map[string]any{"file": f, "match": "alpha", "text": "beta"}
		}},
		{"regex", func(f string) map[string]any {
			return map[string]any{"file": f, "match_regex": "alp[a-z]+", "text": "beta"}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			logp := filepath.Join(t.TempDir(), "audit.log")
			lmcxEnv(t, "permissive", logp)
			f := lhxFile(t, "alpha\nalpha two\nx\n")
			before, _ := os.ReadFile(f)
			if res := mcpLoggedExec("fedit_replaceall", c.args(f)); res.IsError {
				t.Fatalf("unexpected error result: %+v", res)
			}
			after, _ := os.ReadFile(f)
			lines := lmcxLines(t, logp)
			if len(lines) != 2 {
				t.Fatalf("want a begin and an end record, got %q", lines)
			}
			lmcxHas(t, lines[1], "exit=0", "verify=PASS", "lines_changed=2",
				"sha256_before="+lhxSum(before), "sha256_after="+lhxSum(after))
		})
	}
}
