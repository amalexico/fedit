package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func sbhMultiRun(t *testing.T, srcFile, srcBlock string) (map[string]batchResult, int) {
	t.Helper()
	dir := t.TempDir()
	a := mmFile(t, dir, "a.go", `package main
`)
	b := mmFile(t, dir, "b.go", `package main

import "strings"
`)
	c := mmFile(t, dir, "c.txt", `package main
`)
	d := mmFile(t, dir, "d.go", `nothing here
`)
	o := multiMutOpts{
		spec: mmSpec(a, b, c, d), op: "insertafter", match: "package main", nth: 1,
		newLines: sbhInserted, srcFile: srcFile, srcBlock: srcBlock,
	}
	results, ok, err := collectMultiMutate(o)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]batchResult{}
	for _, r := range results {
		byName[filepath.Base(r.path)] = r
	}
	return byName, ok
}

func TestSbhMulti_PerFileHint(t *testing.T) {
	got, ok := sbhMultiRun(t, "src.go", "Up")
	if ok != 3 {
		t.Errorf("ok count %d, want 3", ok)
	}
	if r := got["a.go"]; r.status != "OK" || !strings.Contains(r.detail, "hint:") || !strings.Contains(r.detail, "strings") {
		t.Errorf("a.go: want OK with a strings hint, got %s %q", r.status, r.detail)
	}
	if r := got["b.go"]; r.status != "OK" || strings.Contains(r.detail, "hint:") {
		t.Errorf("b.go: want OK without a hint, got %s %q", r.status, r.detail)
	}
	if r := got["c.txt"]; r.status != "OK" || strings.Contains(r.detail, "hint:") {
		t.Errorf("c.txt: want OK without a hint, got %s %q", r.status, r.detail)
	}
	if r := got["d.go"]; r.status != "SKIP" || strings.Contains(r.detail, "hint:") {
		t.Errorf("d.go: want SKIP without a hint, got %s %q", r.status, r.detail)
	}
}

func TestSbhMulti_NoSourceFlags_NoHint(t *testing.T) {
	got, ok := sbhMultiRun(t, "", "")
	if ok != 3 {
		t.Errorf("ok count %d, want 3", ok)
	}
	for name, r := range got {
		if strings.Contains(r.detail, "hint:") {
			t.Errorf("%s: want no hint without sourcefile and sourceblock, got %q", name, r.detail)
		}
	}
}
