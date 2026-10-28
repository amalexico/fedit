package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mdrCall runs an MCP tool with dry_run set and returns the result and its
// text. It also fails the test when anything reaches stdout, which is the
// JSON-RPC channel under MCP.
func mdrCall(t *testing.T, name string, args map[string]any) (mcpCallResult, string) {
	t.Helper()
	args["dry_run"] = true
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	res := mcpExecTool(name, args)
	os.Stdout = old
	w.Close()
	out, _ := io.ReadAll(r)
	if len(out) > 0 {
		t.Errorf("%s wrote to stdout under dry_run: %q", name, out)
	}
	return res, mcpText(res)
}

func mdrNames(t *testing.T, dir string) int {
	t.Helper()
	ents, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	return len(ents)
}

func TestMCPDryRun_Tools(t *testing.T) {
	cases := []struct {
		name string
		tool string
		args func(a string) map[string]any
		want []string
	}{
		{"insert", "fedit_insert", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "text": "NEWLINE"} }, []string{"+NEWLINE"}},
		{"delete", "fedit_delete", func(a string) map[string]any { return map[string]any{"file": a, "line": 2, "end": 2} }, []string{"-marker"}},
		{"replace", "fedit_replace", func(a string) map[string]any { return map[string]any{"file": a, "line": 2, "end": 2, "text": "MARK"} }, []string{"-marker", "+MARK"}},
		{"write", "fedit_write", func(a string) map[string]any { return map[string]any{"file": a, "text": "zzz"} }, []string{"+zzz"}},
		{"writeraw", "fedit_writeraw", func(a string) map[string]any { return map[string]any{"file": a, "text": "zzz"} }, []string{"+zzz"}},
		{"replaceall", "fedit_replaceall", func(a string) map[string]any { return map[string]any{"file": a, "match": "omega", "text": "OMEGA"} }, []string{"-omega", "+OMEGA"}},
		{"insertafter", "fedit_insertafter", func(a string) map[string]any { return map[string]any{"file": a, "match": "marker", "text": "NEWLINE"} }, []string{"+NEWLINE"}},
		{"insertbefore", "fedit_insertbefore", func(a string) map[string]any { return map[string]any{"file": a, "match": "marker", "text": "NEWLINE"} }, []string{"+NEWLINE"}},
		{"move", "fedit_move", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "end": 1, "after": 3} }, []string{"+++ "}},
		{"copy", "fedit_copy", func(a string) map[string]any { return map[string]any{"file": a, "line": 1, "end": 1, "after": 3} }, []string{"+++ "}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			dir := mcpMMFixture(t)
			a := filepath.Join(dir, "a.txt")
			before := mcpMMRead(t, a)
			res, text := mdrCall(t, c.tool, c.args(a))
			if res.IsError {
				t.Fatalf("error result: %s", text)
			}
			if got := mcpMMRead(t, a); got != before {
				t.Errorf("dry_run changed a.txt: %q -> %q", before, got)
			}
			if mdrNames(t, dir) != 3 {
				t.Errorf("dry_run created a file in %s", dir)
			}
			if !strings.HasPrefix(text, "dry-run: applied=false") {
				t.Errorf("the result should start with the applied=false line: %q", text)
			}
			for _, w := range c.want {
				if !strings.Contains(text, w) {
					t.Errorf("result lacks %q: %q", w, text)
				}
			}
		})
	}
}

func TestMCPDryRun_WriteNewFileInNewDir(t *testing.T) {
	dir := mcpMMFixture(t)
	p := filepath.Join(dir, "sub", "new.txt")
	res, text := mdrCall(t, "fedit_write", map[string]any{"file": p, "text": "x"})
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub")); err == nil {
		t.Error("dry_run created the directory")
	}
	if !strings.Contains(text, "+x") {
		t.Errorf("result lacks the new content: %q", text)
	}
}

func TestMCPDryRun_MultiReplaceAll(t *testing.T) {
	dir := mcpMMFixture(t)
	before := map[string]string{}
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		before[n] = mcpMMRead(t, filepath.Join(dir, n))
	}
	res, text := mdrCall(t, "fedit_replaceall", map[string]any{"file": filepath.Join(dir, "*.txt"), "match": "omega", "text": "OMEGA"})
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	for n, b := range before {
		if got := mcpMMRead(t, filepath.Join(dir, n)); got != b {
			t.Errorf("dry_run changed %s: %q -> %q", n, b, got)
		}
	}
	if n := strings.Count(text, "+++ "); n != 3 {
		t.Errorf("want 3 file diffs, got %d in %q", n, text)
	}
	if !strings.Contains(text, "3/3 file(s) would succeed") || strings.Contains(text, "succeeded") {
		t.Errorf("the batch total should read would succeed: %q", text)
	}
}

func TestMCPDryRun_Dest(t *testing.T) {
	dir := mcpMMFixture(t)
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	wa, wb := mcpMMRead(t, a), mcpMMRead(t, b)
	res, text := mdrCall(t, "fedit_copy", map[string]any{"file": a, "line": 1, "end": 1, "dest": b, "aftermatch": "beta"})
	if res.IsError {
		t.Fatalf("error result: %s", text)
	}
	if mcpMMRead(t, a) != wa || mcpMMRead(t, b) != wb {
		t.Error("dry_run changed a file")
	}
	if !strings.Contains(text, "+++ "+b) || !strings.Contains(text, "1/1 file(s) would succeed") {
		t.Errorf("result lacks the destination diff or the dry total: %q", text)
	}
}

func TestMCPDryRun_ErrorStaysError(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	before := mcpMMRead(t, a)
	res, text := mdrCall(t, "fedit_replace", map[string]any{"file": a, "match": "no such text", "text": "x"})
	if !res.IsError {
		t.Errorf("a failing call must stay an error: %q", text)
	}
	if mcpMMRead(t, a) != before {
		t.Error("dry_run changed a.txt")
	}
}

func TestMCPDryRun_ReadOnlyToolsIgnoreIt(t *testing.T) {
	dir := mcpMMFixture(t)
	res, text := mdrCall(t, "fedit_find", map[string]any{"file": filepath.Join(dir, "a.txt"), "match": "marker"})
	if res.IsError || strings.Contains(text, "dry-run") {
		t.Errorf("a read-only tool must ignore dry_run: %q", text)
	}
}

func TestMCPDryRun_GlobalsRestored(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	before := mcpMMRead(t, a)
	mdrCall(t, "fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA"})
	if mcpMMRead(t, a) != before {
		t.Fatal("dry_run changed a.txt")
	}
	if dryRun || dryOut != nil {
		t.Errorf("the dry state leaked: dryRun=%v dryOut=%v", dryRun, dryOut)
	}
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": a, "match": "omega", "text": "OMEGA"})
	if res.IsError {
		t.Fatalf("real call failed: %s", mcpText(res))
	}
	if got := mcpMMRead(t, a); !strings.Contains(got, "OMEGA") {
		t.Errorf("the next real call must write: %q", got)
	}
}

func TestMCPDryRun_Schema(t *testing.T) {
	want := map[string]bool{
		"fedit_insert": true, "fedit_delete": true, "fedit_replace": true, "fedit_write": true,
		"fedit_writeraw": true, "fedit_replaceall": true, "fedit_insertafter": true,
		"fedit_insertbefore": true, "fedit_move": true, "fedit_copy": true,
		"fedit_show": false, "fedit_find": false, "fedit_map": false, "fedit_fields": false,
	}
	seen := 0
	for _, d := range mcpToolDefs() {
		w, ok := want[d.Name]
		if !ok {
			continue
		}
		seen++
		if has := strings.Contains(fmt.Sprintf("%s", d.InputSchema), "\"dry_run\""); has != w {
			t.Errorf("%s: dry_run in schema = %v, want %v", d.Name, has, w)
		}
	}
	if seen != len(want) {
		t.Errorf("found %d of %d tools", seen, len(want))
	}
}
