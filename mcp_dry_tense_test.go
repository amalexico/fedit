package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestMCPDryRun_SummaryTense: under dry_run nothing was done, so the tool summary reads as a preview (Would insert ...), never as a past tense report, and it does not claim a total now.
func TestMCPDryRun_SummaryTense(t *testing.T) {
	past := []string{"Inserted ", "Deleted ", "Replaced ", "Moved ", "Copied ", "Wrote "}
	cases := []struct {
		name string
		tool string
		args func(a string) map[string]any
		want string
	}{
		{"insert", "fedit_insert", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "text": "NEWLINE"} }, "Would insert "},
		{"delete", "fedit_delete", func(a string) map[string]any { return map[string]any{"file": a, "line": 2, "end": 2} }, "Would delete lines"},
		{"replace", "fedit_replace", func(a string) map[string]any { return map[string]any{"file": a, "line": 2, "end": 2, "text": "MARK"} }, "Would replace lines"},
		{"write", "fedit_write", func(a string) map[string]any { return map[string]any{"file": a, "text": "zzz"} }, "Would write "},
		{"writeraw", "fedit_writeraw", func(a string) map[string]any { return map[string]any{"file": a, "text": "zzz"} }, "Would write "},
		{"replaceall", "fedit_replaceall", func(a string) map[string]any { return map[string]any{"file": a, "match": "omega", "text": "OMEGA"} }, "Would replace "},
		{"insertafter", "fedit_insertafter", func(a string) map[string]any { return map[string]any{"file": a, "match": "marker", "text": "NEWLINE"} }, "Would insert "},
		{"insertbefore", "fedit_insertbefore", func(a string) map[string]any { return map[string]any{"file": a, "match": "marker", "text": "NEWLINE"} }, "Would insert "},
		{"move", "fedit_move", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "end": 1, "after": 3} }, "Would move lines"},
		{"copy", "fedit_copy", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "end": 1, "after": 3} }, "Would copy lines"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dir := mcpMMFixture(t)
			a := filepath.Join(dir, "a.txt")
			res, text := mdrCall(t, c.tool, c.args(a))
			if res.IsError {
				t.Fatalf("error result: %s", text)
			}
			found := false
			for _, l := range strings.Split(text, "\n") {
				for _, p := range past {
					if strings.HasPrefix(l, p) {
						t.Errorf("past tense summary under dry_run: %q", l)
					}
				}
				if strings.HasPrefix(l, c.want) {
					found = true
				}
			}
			if !found {
				t.Errorf("no summary line starting with %q: %q", c.want, text)
			}
			if strings.Contains(text, "total now)") {
				t.Errorf("the summary says now for a file that did not change: %q", text)
			}
		})
	}
}

// TestMCPDryRun_RealSummaryStaysPast: a real call keeps its past tense summary.
func TestMCPDryRun_RealSummaryStaysPast(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	res := mcpExecTool("fedit_insert", map[string]any{"file": a, "line": 1, "text": "NEWLINE"})
	if res.IsError {
		t.Fatalf("real call failed: %s", mcpText(res))
	}
	if text := mcpText(res); !strings.HasPrefix(text, "Inserted ") || !strings.Contains(text, "total now)") {
		t.Errorf("a real call must keep its summary: %q", text)
	}
}
