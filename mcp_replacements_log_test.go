package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// The MCP fedit_replaceall end record must note the same result fields as the
// CLI: replacements (occurrences) as well as lines_changed (lines). A dry run
// notes neither.
func TestMRP_ReplacementsInEndRecord(t *testing.T) {
	run := func(t *testing.T, args func(f string) map[string]any) string {
		t.Helper()
		logp := filepath.Join(t.TempDir(), "audit.log")
		lmcxEnv(t, "permissive", logp)
		f := lhxFile(t, "alpha alpha\nalpha\nx\n")
		if res := mcpLoggedExec("fedit_replaceall", args(f)); res.IsError {
			t.Fatalf("unexpected error result: %s", mcpText(res))
		}
		lines := lmcxLines(t, logp)
		if len(lines) != 2 {
			t.Fatalf("want a begin and an end record, got %q", lines)
		}
		return lines[1]
	}
	t.Run("literal", func(t *testing.T) {
		end := run(t, func(f string) map[string]any {
			return map[string]any{"file": f, "match": "alpha", "text": "beta"}
		})
		lmcxHas(t, end, "exit=0", "verify=PASS", "replacements=3", "lines_changed=2")
	})
	t.Run("regex", func(t *testing.T) {
		end := run(t, func(f string) map[string]any {
			return map[string]any{"file": f, "match_regex": "alph[a]", "text": "beta"}
		})
		lmcxHas(t, end, "exit=0", "verify=PASS", "replacements=3", "lines_changed=2")
	})
	t.Run("dry_run", func(t *testing.T) {
		end := run(t, func(f string) map[string]any {
			return map[string]any{"file": f, "match": "alpha", "text": "beta", "dry_run": true}
		})
		lmcxHas(t, end, "exit=0", "dry=true")
		for _, key := range []string{"replacements=", "lines_changed="} {
			if strings.Contains(end, key) {
				t.Errorf("a dry run must note no %s: %s", key, end)
			}
		}
	})
}
