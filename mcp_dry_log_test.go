package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// mlgEnds returns the end records of the audit log, in order.
func mlgEnds(t *testing.T, logp string) []string {
	t.Helper()
	var ends []string
	for _, l := range lwxLog(t, logp) {
		if strings.Contains(l, "phase=end") {
			ends = append(ends, l)
		}
	}
	return ends
}

// TestMCPDryRunLogged: a logged MCP call with dry_run changes no file and its
// end record carries dry=true; the next real call must not inherit the flag.
func TestMCPDryRunLogged(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	logp := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	before := mcpMMRead(t, a)
	res := mcpLoggedExec("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA", "dry_run": true})
	if res.IsError {
		t.Fatalf("dry call failed: %s", mcpText(res))
	}
	if got := mcpMMRead(t, a); got != before {
		t.Errorf("dry_run changed a.txt: %q -> %q", before, got)
	}
	ends := mlgEnds(t, logp)
	if len(ends) != 1 {
		t.Fatalf("want 1 end record, got %d: %q", len(ends), ends)
	}
	if !strings.Contains(ends[0], "exit=0") || !strings.Contains(ends[0], " dry=true") {
		t.Errorf("the end record should carry exit=0 and dry=true: %q", ends[0])
	}
	if !strings.Contains(strings.Join(lwxLog(t, logp), "\n"), "source=mcp") {
		t.Errorf("the log has no source=mcp record")
	}
	res = mcpLoggedExec("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA"})
	if res.IsError {
		t.Fatalf("real call failed: %s", mcpText(res))
	}
	if got := mcpMMRead(t, a); !strings.Contains(got, "OMEGA") {
		t.Errorf("the real call must write: %q", got)
	}
	ends = mlgEnds(t, logp)
	if len(ends) != 2 {
		t.Fatalf("want 2 end records, got %d: %q", len(ends), ends)
	}
	if strings.Contains(ends[1], "dry=true") {
		t.Errorf("the dry flag leaked into the next record: %q", ends[1])
	}
}
