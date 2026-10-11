package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mcpMMFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"a.txt": "alpha\nmarker\nomega\n",
		"b.txt": "beta\nmarker\nomega\n",
		"c.txt": "gamma\nomega\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func mcpMMRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func mcpMMSpecAB(dir string) string {
	return filepath.Join(dir, "a.txt") + "," + filepath.Join(dir, "b.txt")
}

func TestMCPMultiMut_ReplaceAllGlob(t *testing.T) {
	dir := mcpMMFixture(t)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": filepath.Join(dir, "*.txt"), "match": "omega", "text": "OMEGA"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "3/3 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	for _, n := range []string{"a.txt", "b.txt", "c.txt"} {
		if !strings.Contains(mcpMMRead(t, filepath.Join(dir, n)), "OMEGA") {
			t.Errorf("%s not updated", n)
		}
	}
}

func TestMCPMultiMut_ReplaceAllRegex(t *testing.T) {
	dir := t.TempDir()
	p1 := filepath.Join(dir, "r1.txt")
	p2 := filepath.Join(dir, "r2.txt")
	os.WriteFile(p1, []byte("foo1\nbar\n"), 0o644)
	os.WriteFile(p2, []byte("foo22\n"), 0o644)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": p1 + "," + p2, "match_regex": `foo(\d+)`, "text": "baz$1"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "2/2 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	if !strings.Contains(mcpMMRead(t, p1), "baz1") || !strings.Contains(mcpMMRead(t, p2), "baz22") {
		t.Errorf("regex replace not applied: %q %q", mcpMMRead(t, p1), mcpMMRead(t, p2))
	}
}

func TestMCPMultiMut_InsertAfter(t *testing.T) {
	dir := mcpMMFixture(t)
	res := mcpExecTool("fedit_insertafter", map[string]any{"file": mcpMMSpecAB(dir), "match": "marker", "text": "NEW"})
	if res.IsError {
		t.Fatalf("unexpected error: %s", mcpMultiText(res))
	}
	for _, n := range []string{"a.txt", "b.txt"} {
		if !strings.Contains(mcpMMRead(t, filepath.Join(dir, n)), "marker\nNEW\nomega") {
			t.Errorf("%s: insert not in place:\n%s", n, mcpMMRead(t, filepath.Join(dir, n)))
		}
	}
}

func TestMCPMultiMut_DeleteSkipsFileWithoutMatch(t *testing.T) {
	dir := mcpMMFixture(t)
	spec := mcpMMSpecAB(dir) + "," + filepath.Join(dir, "c.txt")
	res := mcpExecTool("fedit_delete", map[string]any{"file": spec, "match": "marker"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "2/3 file(s) succeeded") || !strings.Contains(text, "SKIP") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	if strings.Contains(mcpMMRead(t, filepath.Join(dir, "a.txt")), "marker") {
		t.Error("a.txt still has the marker")
	}
	if mcpMMRead(t, filepath.Join(dir, "c.txt")) != "gamma\nomega\n" {
		t.Error("c.txt must be left unmodified")
	}
}

func TestMCPMultiMut_Replace(t *testing.T) {
	dir := mcpMMFixture(t)
	res := mcpExecTool("fedit_replace", map[string]any{"file": mcpMMSpecAB(dir), "match": "marker", "text": "REPLACED"})
	if res.IsError {
		t.Fatalf("unexpected error: %s", mcpMultiText(res))
	}
	if !strings.Contains(mcpMMRead(t, filepath.Join(dir, "a.txt")), "alpha\nREPLACED\nomega") {
		t.Errorf("replace not applied:\n%s", mcpMMRead(t, filepath.Join(dir, "a.txt")))
	}
}

func TestMCPMultiMut_MissingFileIsSkipped(t *testing.T) {
	dir := mcpMMFixture(t)
	spec := filepath.Join(dir, "a.txt") + "," + filepath.Join(dir, "missing.txt")
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": spec, "match": "alpha", "text": "ALPHA"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "SKIP") || !strings.Contains(text, "1/2 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
}

func TestMCPMultiMut_NoFileSucceededIsError(t *testing.T) {
	dir := mcpMMFixture(t)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": mcpMMSpecAB(dir), "match": "zzz-nothing", "text": "x"})
	text := mcpMultiText(res)
	if !res.IsError || !strings.Contains(text, "0/2 file(s) succeeded") {
		t.Fatalf("want error with 0/2, got:\n%s", text)
	}
}

func TestMCPMultiMut_RejectedRequestsModifyNothing(t *testing.T) {
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"block", "fedit_insertafter", map[string]any{"match": "marker", "text": "x", "block": "Foo"}, "block is not supported"},
		{"line", "fedit_delete", map[string]any{"line": 2}, "line numbers are not supported"},
		{"stream", "fedit_replaceall", map[string]any{"match": "marker", "text": "x", "stream": true}, "stream and files are not supported"},
		{"nomatch", "fedit_replaceall", map[string]any{"text": "x"}, "requires match"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := mcpMMFixture(t)
			before := mcpMMRead(t, filepath.Join(dir, "a.txt")) + mcpMMRead(t, filepath.Join(dir, "b.txt"))
			c.args["file"] = mcpMMSpecAB(dir)
			res := mcpExecTool(c.tool, c.args)
			text := mcpMultiText(res)
			if !res.IsError || !strings.Contains(text, c.want) {
				t.Errorf("want error containing %q, got:\n%s", c.want, text)
			}
			after := mcpMMRead(t, filepath.Join(dir, "a.txt")) + mcpMMRead(t, filepath.Join(dir, "b.txt"))
			if before != after {
				t.Error("a rejected request must not modify any file")
			}
		})
	}
}

func TestMCPMultiMut_ReadOnlyFileIsSkipped(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission bits")
	}
	dir := mcpMMFixture(t)
	ro := filepath.Join(dir, "a.txt")
	if err := os.Chmod(ro, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(ro, 0o644) })
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": mcpMMSpecAB(dir), "match": "marker", "text": "MARKER"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "SKIP") || !strings.Contains(text, "1/2 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	if strings.Contains(mcpMMRead(t, ro), "MARKER") {
		t.Error("the read-only file was modified")
	}
	if !strings.Contains(mcpMMRead(t, filepath.Join(dir, "b.txt")), "MARKER") {
		t.Error("b.txt should still have been updated")
	}
}

func TestMCPMultiMut_SingleFileUnchanged(t *testing.T) {
	dir := mcpMMFixture(t)
	res := mcpExecTool("fedit_replaceall", map[string]any{"file": filepath.Join(dir, "a.txt"), "match": "alpha", "text": "ALPHA"})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "Replaced") || strings.Contains(text, "file(s) succeeded") {
		t.Errorf("single-file path changed:\n%s", text)
	}
}
