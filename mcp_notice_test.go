package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// mntNotices returns the text items of an MCP result that carry the not-logged notice.
func mntNotices(res mcpCallResult) []string {
	var out []string
	for _, c := range res.Content {
		if strings.Contains(c.Text, "The action was not logged") {
			out = append(out, c.Text)
		}
	}
	return out
}

// TestMCPNotLoggedNotice: in permissive mode, when the main log cannot be written, the tool result carries the not-logged notice as one extra text item; the tool result itself stays first and the call is not an error.
func TestMCPNotLoggedNotice(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "la"))
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", t.TempDir())
	res := mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "marker"})
	if res.IsError {
		t.Fatalf("the call must not fail in permissive mode: %q", res.Content)
	}
	if len(res.Content) == 0 || !strings.Contains(res.Content[0].Text, "marker") {
		t.Fatalf("the first item must be the find result: %q", res.Content)
	}
	if n := mntNotices(res); len(n) != 1 {
		t.Errorf("want exactly 1 not-logged notice in the result, got %d: %q", len(n), res.Content)
	}
}

// TestMCPNoNoticeWhenLogged: a call whose log is written carries no notice, also right after a call that had one.
func TestMCPNoNoticeWhenLogged(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	t.Setenv("LOCALAPPDATA", filepath.Join(t.TempDir(), "la"))
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", t.TempDir())
	mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "marker"})
	t.Setenv("FEDIT_LOG_FILE", filepath.Join(t.TempDir(), "audit.log"))
	res := mcpLoggedExec("fedit_find", map[string]any{"file": a, "match": "marker"})
	if res.IsError {
		t.Fatalf("the call failed: %q", res.Content)
	}
	if n := mntNotices(res); len(n) != 0 {
		t.Errorf("a logged call must carry no notice, got %q", n)
	}
}
