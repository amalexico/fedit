package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mopFix returns the fixture folder, the path of a.txt and a result file path
// in the same folder (not created).
func mopFix(t *testing.T) (string, string, string) {
	t.Helper()
	dir := mcpMMFixture(t)
	return dir, filepath.Join(dir, "a.txt"), filepath.Join(dir, "result.out")
}

func mopRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mopMissing(t *testing.T, p string) {
	t.Helper()
	if _, err := os.Stat(p); err == nil {
		t.Errorf("%s should not exist", p)
	}
}

// find with x=true gives bare line numbers, so two calls return the same text.
func mopFind(a, out string, app bool) mcpCallResult {
	args := map[string]any{"file": a, "match": "marker", "x": true}
	if out != "" {
		args["output_file"] = out
	}
	if app {
		args["output_append"] = true
	}
	return mcpExecTool("fedit_find", args)
}

func TestMCPOutput_Find(t *testing.T) {
	_, a, out := mopFix(t)
	want := mcpText(mopFind(a, "", false))
	res := mopFind(a, out, false)
	text := mcpText(res)
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	if !strings.HasPrefix(text, "result written to "+out+" (") || !strings.Contains(text, " lines)") {
		t.Errorf("unexpected summary: %q", text)
	}
	if got := mopRead(t, out); strings.TrimRight(got, "\n") != strings.TrimRight(want, "\n") {
		t.Errorf("result file = %q, want %q", got, want)
	}
}

func TestMCPOutput_OverwriteAndAppend(t *testing.T) {
	_, a, out := mopFix(t)
	want := strings.TrimRight(mcpText(mopFind(a, "", false)), "\n")
	if err := os.WriteFile(out, []byte("old content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mopFind(a, out, false)
	if got := strings.TrimRight(mopRead(t, out), "\n"); got != want {
		t.Errorf("overwrite: result file = %q, want %q", got, want)
	}
	text := mcpText(mopFind(a, out, true))
	if !strings.HasPrefix(text, "result appended to "+out) {
		t.Errorf("unexpected summary: %q", text)
	}
	if got := mopRead(t, out); strings.Count(got, want) != 2 {
		t.Errorf("append: want the result twice, got %q", got)
	}
}

func TestMCPOutput_ErrorLeavesFile(t *testing.T) {
	_, a, out := mopFix(t)
	before := mopRead(t, a)
	args := map[string]any{"file": a, "match": "no such text", "text": "x", "output_file": out}
	res := mcpExecTool("fedit_replace", args)
	if !res.IsError {
		t.Errorf("a failing call must stay an error: %q", mcpText(res))
	}
	mopMissing(t, out)
	if err := os.WriteFile(out, []byte("keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mcpExecTool("fedit_replace", args)
	if got := mopRead(t, out); got != "keep\n" {
		t.Errorf("an error result must leave the result file alone: %q", got)
	}
	if mopRead(t, a) != before {
		t.Error("a.txt changed")
	}
}

func TestMCPOutput_RefusesEditedFile(t *testing.T) {
	dir, a, _ := mopFix(t)
	b := filepath.Join(dir, "b.txt")
	beforeA, beforeB := mopRead(t, a), mopRead(t, b)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA", "output_file": a})
	if !res.IsError || !strings.Contains(mcpText(res), "also an edited file") {
		t.Errorf("output_file equal to file must be refused: %q", mcpText(res))
	}
	res = mcpExecTool("fedit_copy", map[string]any{"file": a, "line": 1, "end": 1, "dest": b, "aftermatch": "beta", "output_file": b})
	if !res.IsError || !strings.Contains(mcpText(res), "also an edited file") {
		t.Errorf("output_file equal to dest must be refused: %q", mcpText(res))
	}
	if mopRead(t, a) != beforeA || mopRead(t, b) != beforeB {
		t.Error("a refused call changed a file")
	}
}

func TestMCPOutput_MissingDir(t *testing.T) {
	dir, a, _ := mopFix(t)
	before := mopRead(t, a)
	out := filepath.Join(dir, "nosuch", "r.out")
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA", "output_file": out})
	if !res.IsError || !strings.Contains(mcpText(res), "does not exist") {
		t.Errorf("a missing folder must be an error: %q", mcpText(res))
	}
	if mopRead(t, a) != before {
		t.Error("the edit ran although the result file could not be created")
	}
}

func TestMCPOutput_AppendNeedsFile(t *testing.T) {
	_, a, _ := mopFix(t)
	res := mopFind(a, "", true)
	if !res.IsError || !strings.Contains(mcpText(res), "output_append requires output_file") {
		t.Errorf("output_append alone must be an error: %q", mcpText(res))
	}
}

func TestMCPOutput_DryRun(t *testing.T) {
	_, a, out := mopFix(t)
	before := mopRead(t, a)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA", "dry_run": true, "output_file": out})
	text := mcpText(res)
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	if mopRead(t, a) != before {
		t.Error("dry_run changed a.txt")
	}
	if !strings.HasPrefix(text, "result written to ") {
		t.Errorf("unexpected summary: %q", text)
	}
	got := mopRead(t, out)
	if !strings.HasPrefix(got, "dry-run: applied=false") || !strings.Contains(got, "+OMEGA") {
		t.Errorf("the result file should hold the dry-run text: %q", got)
	}
}

func TestMCPOutput_MutationRuns(t *testing.T) {
	_, a, out := mopFix(t)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA", "output_file": out})
	text := mcpText(res)
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	if got := mopRead(t, a); !strings.Contains(got, "OMEGA") {
		t.Errorf("the edit must still run: %q", got)
	}
	if !strings.HasPrefix(text, "result written to ") {
		t.Errorf("unexpected summary: %q", text)
	}
	if strings.TrimSpace(mopRead(t, out)) == "" {
		t.Error("the result file is empty")
	}
}

func TestMCPOutput_Schema(t *testing.T) {
	n := 0
	for _, d := range mcpToolDefs() {
		n++
		s := fmt.Sprintf("%s", d.InputSchema)
		for _, k := range []string{"\"output_file\"", "\"output_append\""} {
			if !strings.Contains(s, k) {
				t.Errorf("%s: schema lacks %s", d.Name, k)
			}
		}
	}
	if n != 14 {
		t.Errorf("found %d tools, want 14", n)
	}
}
