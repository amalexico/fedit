package main

import (
	"os"
	"strings"
	"testing"
)

func mraRead(t *testing.T, path string) []string {
	t.Helper()
	lines, err := readLines(path)
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

func mraWant(t *testing.T, path string, want ...string) {
	t.Helper()
	lines := mraRead(t, path)
	if len(lines) < len(want) {
		t.Fatalf("got %d lines, want at least %d: %q", len(lines), len(want), lines)
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d = %q, want %q", i+1, lines[i], w)
		}
	}
}

func TestMCPExec_ReplaceAll_Regex(t *testing.T) {
	path := mcpTempFile(t, "foo bar\nfoo baz\nqux\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":        path,
		"match_regex": "^foo.*$",
		"text":        "X",
	})
	if r.IsError {
		t.Fatalf("regex replaceall error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "Replaced /^foo.*$/ on 2 line(s)") {
		t.Errorf("unexpected message: %s", mcpText(r))
	}
	mraWant(t, path, "X", "X", "qux")
}

func TestMCPExec_ReplaceAll_RegexNoMatch(t *testing.T) {
	path := mcpTempFile(t, "foo\nbar\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":        path,
		"match_regex": "^zzz",
		"text":        "X",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "No lines match: ^zzz") {
		t.Fatalf("want no-match error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, path, "foo", "bar")
}

func TestMCPExec_ReplaceAll_RegexInvalid(t *testing.T) {
	path := mcpTempFile(t, "foo\nbar\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":        path,
		"match_regex": "(",
		"text":        "X",
	})
	if !r.IsError {
		t.Fatalf("want error for invalid regex, got: %s", mcpText(r))
	}
	mraWant(t, path, "foo", "bar")
}

func TestMCPExec_ReplaceAll_MatchAndRegex(t *testing.T) {
	path := mcpTempFile(t, "foo\nbar\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":        path,
		"match":       "foo",
		"match_regex": "^foo",
		"text":        "X",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "match or match_regex, not both") {
		t.Fatalf("want not-both error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, path, "foo", "bar")
}

func TestMCPExec_ReplaceAll_NeedsMatch(t *testing.T) {
	path := mcpTempFile(t, "foo\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file": path,
		"text": "X",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "requires match") {
		t.Fatalf("want requires-match error, got error=%v text=%s", r.IsError, mcpText(r))
	}
}

func TestMCPExec_ReplaceAll_TextFile(t *testing.T) {
	path := mcpTempFile(t, "foo bar\nfoo baz\nqux\n")
	defer os.Remove(path)
	repl := mcpTempFile(t, "new\n\n")
	defer os.Remove(repl)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":     path,
		"match":    "foo",
		"textfile": repl,
	})
	if r.IsError {
		t.Fatalf("textfile replaceall error: %s", mcpText(r))
	}
	mraWant(t, path, "new bar", "new baz", "qux")
}

func TestMCPExec_ReplaceAll_TextFileAndText(t *testing.T) {
	path := mcpTempFile(t, "foo\n")
	defer os.Remove(path)
	repl := mcpTempFile(t, "new\n")
	defer os.Remove(repl)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":     path,
		"match":    "foo",
		"text":     "X",
		"textfile": repl,
	})
	if !r.IsError || !strings.Contains(mcpText(r), "text or textfile, not both") {
		t.Fatalf("want not-both error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, path, "foo")
}

func TestMCPExec_ReplaceAll_TextFileMissing(t *testing.T) {
	path := mcpTempFile(t, "foo\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":     path,
		"match":    "foo",
		"textfile": path + ".does_not_exist",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "Error reading textfile") {
		t.Fatalf("want textfile read error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, path, "foo")
}

func TestMCPExec_ReplaceAll_LiteralMessage(t *testing.T) {
	path := mcpTempFile(t, "foo bar\nfoo baz\nqux\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":  path,
		"match": "foo",
		"text":  "Y",
	})
	if r.IsError {
		t.Fatalf("literal replaceall error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "Replaced 'foo' on 2 line(s)") {
		t.Errorf("unexpected message: %s", mcpText(r))
	}
	mraWant(t, path, "Y bar", "Y baz", "qux")
}

func TestMCPExec_ReplaceAll_FilesRejected(t *testing.T) {
	path := mcpTempFile(t, "foo\nbar\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":  path,
		"match": "foo",
		"text":  "X",
		"files": "*.go",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "files parameter") {
		t.Fatalf("want files-rejected error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, path, "foo", "bar")
}

func TestMCPExec_ReplaceAll_MultiTextFile(t *testing.T) {
	p1 := mcpTempFile(t, "foo one\nkeep\n")
	defer os.Remove(p1)
	p2 := mcpTempFile(t, "two foo\n")
	defer os.Remove(p2)
	repl := mcpTempFile(t, "new\n\n")
	defer os.Remove(repl)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":     p1 + "," + p2,
		"match":    "foo",
		"textfile": repl,
	})
	if r.IsError {
		t.Fatalf("multi textfile replaceall error: %s", mcpText(r))
	}
	mraWant(t, p1, "new one", "keep")
	mraWant(t, p2, "two new")
}

func TestMCPExec_ReplaceAll_MultiTextFileAndText(t *testing.T) {
	p1 := mcpTempFile(t, "foo\n")
	defer os.Remove(p1)
	p2 := mcpTempFile(t, "foo\n")
	defer os.Remove(p2)
	repl := mcpTempFile(t, "new\n")
	defer os.Remove(repl)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":     p1 + "," + p2,
		"match":    "foo",
		"text":     "X",
		"textfile": repl,
	})
	if !r.IsError || !strings.Contains(mcpText(r), "text or textfile, not both") {
		t.Fatalf("want not-both error, got error=%v text=%s", r.IsError, mcpText(r))
	}
	mraWant(t, p1, "foo")
	mraWant(t, p2, "foo")
}
