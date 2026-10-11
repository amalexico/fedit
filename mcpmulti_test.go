package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mcpMultiText(res mcpCallResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	return res.Content[0].Text
}

func mcpMultiFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"a.go": "package a\n\nfunc Alpha() {}\n// marker one\n",
		"b.go": "package b\n\nfunc Beta() {}\n// marker two\n",
		"c.go": "package c\n\nfunc Gamma() {}\n",
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestMCPMulti_FindCommaList(t *testing.T) {
	dir := mcpMultiFixture(t)
	spec := filepath.Join(dir, "a.go") + "," + filepath.Join(dir, "b.go") + "," + filepath.Join(dir, "c.go")
	res := mcpExecTool("fedit_find", map[string]any{"file": spec, "match": "marker"})
	text := mcpMultiText(res)
	if res.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if strings.Count(text, "==> ") != 2 {
		t.Errorf("want 2 headers, got:\n%s", text)
	}
	if !strings.Contains(text, "2 with results, 1 without, 0 skipped") {
		t.Errorf("summary wrong:\n%s", text)
	}
}

func TestMCPMulti_FindGlob(t *testing.T) {
	dir := mcpMultiFixture(t)
	res := mcpExecTool("fedit_find", map[string]any{"file": filepath.Join(dir, "*.go"), "match": "marker"})
	text := mcpMultiText(res)
	if res.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !strings.Contains(text, "2 with results, 1 without, 0 skipped") {
		t.Errorf("summary wrong:\n%s", text)
	}
}

func TestMCPMulti_FindNoHitsAnywhere(t *testing.T) {
	dir := mcpMultiFixture(t)
	res := mcpExecTool("fedit_find", map[string]any{"file": filepath.Join(dir, "*.go"), "match": "zzz-nothing"})
	text := mcpMultiText(res)
	if !res.IsError {
		t.Errorf("want error when nothing matched anywhere, got:\n%s", text)
	}
	if !strings.Contains(text, "0 with results, 3 without") {
		t.Errorf("summary wrong:\n%s", text)
	}
}

func TestMCPMulti_MissingFileIsSkipped(t *testing.T) {
	dir := mcpMultiFixture(t)
	spec := filepath.Join(dir, "a.go") + "," + filepath.Join(dir, "missing.go") + "," + filepath.Join(dir, "b.go")
	res := mcpExecTool("fedit_find", map[string]any{"file": spec, "match": "marker"})
	text := mcpMultiText(res)
	if res.IsError {
		t.Fatalf("a skipped file must not fail the batch: %s", text)
	}
	if !strings.Contains(text, "SKIP") || !strings.Contains(text, "2 with results, 0 without, 1 skipped") {
		t.Errorf("skip not reported:\n%s", text)
	}
}

func TestMCPMulti_Map(t *testing.T) {
	dir := mcpMultiFixture(t)
	spec := filepath.Join(dir, "a.go") + "," + filepath.Join(dir, "b.go")
	res := mcpExecTool("fedit_map", map[string]any{"file": spec})
	text := mcpMultiText(res)
	if res.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if strings.Count(text, "==> ") != 2 {
		t.Errorf("want 2 headers, got:\n%s", text)
	}
}

func TestMCPMulti_Show(t *testing.T) {
	dir := mcpMultiFixture(t)
	spec := filepath.Join(dir, "a.go") + "," + filepath.Join(dir, "b.go")
	res := mcpExecTool("fedit_show", map[string]any{"file": spec, "line": 1, "end": 1})
	text := mcpMultiText(res)
	if res.IsError {
		t.Fatalf("unexpected error: %s", text)
	}
	if !strings.Contains(text, "package a") || !strings.Contains(text, "package b") {
		t.Errorf("both files should be shown:\n%s", text)
	}
}

func TestMCPMulti_SingleFileUnchanged(t *testing.T) {
	dir := mcpMultiFixture(t)
	res := mcpExecTool("fedit_find", map[string]any{"file": filepath.Join(dir, "a.go"), "match": "marker"})
	text := mcpMultiText(res)
	if res.IsError || strings.Contains(text, "==> ") || !strings.Contains(text, "Found 1 match") {
		t.Errorf("single-file path changed:\n%s", text)
	}
}

func TestMCPMulti_WriteWithCommaInNameStillWorks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "x,y.txt")
	res := mcpExecTool("fedit_write", map[string]any{"file": target, "text": "hi"})
	if res.IsError {
		t.Fatalf("write failed: %s", mcpMultiText(res))
	}
	if _, err := os.Stat(target); err != nil {
		t.Errorf("file with a comma in its name was not created: %v", err)
	}
}
