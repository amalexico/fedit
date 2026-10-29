package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// mogBegins returns the begin records of the audit log, in order.
func mogBegins(t *testing.T, logp string) []string {
	t.Helper()
	var out []string
	for _, l := range lwxLog(t, logp) {
		if strings.Contains(l, "phase=begin") {
			out = append(out, l)
		}
	}
	return out
}

// TestMCPOutputLogged: the begin record of a logged MCP call names the result file (output_file, and output_append when it is set); a call without them logs neither.
func TestMCPOutputLogged(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	res := filepath.Join(dir, "result.txt")
	logp := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	r := mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "marker", "output_file": res})
	if r.IsError {
		t.Fatalf("output call failed: %s", mcpText(r))
	}
	if got := mcpMMRead(t, res); !strings.Contains(got, "marker") {
		t.Errorf("the result file should hold the find result: %q", got)
	}
	r = mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "alpha", "output_file": res, "output_append": true})
	if r.IsError {
		t.Fatalf("append call failed: %s", mcpText(r))
	}
	r = mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "omega"})
	if r.IsError {
		t.Fatalf("plain call failed: %s", mcpText(r))
	}
	begins := mogBegins(t, logp)
	if len(begins) != 3 {
		t.Fatalf("want 3 begin records, got %d: %q", len(begins), begins)
	}
	if !strings.Contains(begins[0], " output_file=") || !strings.Contains(begins[0], "result.txt") {
		t.Errorf("no output_file in the begin record: %q", begins[0])
	}
	if strings.Contains(begins[0], "output_append") {
		t.Errorf("output_append must not be logged when it is not set: %q", begins[0])
	}
	if !strings.Contains(begins[1], " output_file=") || !strings.Contains(begins[1], " output_append=true") {
		t.Errorf("no output_file and output_append=true in the begin record: %q", begins[1])
	}
	if strings.Contains(begins[2], "output_file") || strings.Contains(begins[2], "output_append") {
		t.Errorf("a call without output_file must log neither: %q", begins[2])
	}
}
