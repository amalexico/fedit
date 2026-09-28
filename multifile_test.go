package main

import (
	"os"
	"path/filepath"
	"testing"
)

func mfWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("mfWrite %s: %v", name, err)
	}
	return p
}

func TestIsMultiSpec(t *testing.T) {
	dir := t.TempDir()
	real := mfWrite(t, dir, "x[1].txt", "hi\n")
	cases := []struct {
		spec string
		want bool
	}{
		{"a.go", false},
		{"a.go,b.go", true},
		{"*.go", true},
		{"internal/*.go", true},
		{real, false}, // a real file with glob characters in its name wins
	}
	for _, c := range cases {
		if got := isMultiSpec(c.spec); got != c.want {
			t.Errorf("isMultiSpec(%q) = %v, want %v", c.spec, got, c.want)
		}
	}
}

func TestExpandMultiSpec_CommaGlobDedup(t *testing.T) {
	dir := t.TempDir()
	a := mfWrite(t, dir, "a.go", "package a\n")
	b := mfWrite(t, dir, "b.go", "package b\n")
	mfWrite(t, dir, "c.txt", "x\n")
	got, err := expandMultiSpec(filepath.Join(dir, "*.go") + "," + a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[0] != a || got[1] != b {
		t.Errorf("got %v, want [%s %s]", got, a, b)
	}
}

func TestExpandMultiSpec_NoMatch(t *testing.T) {
	dir := t.TempDir()
	if _, err := expandMultiSpec(filepath.Join(dir, "*.nothing")); err == nil {
		t.Error("expected an error when nothing matches")
	}
}

func TestExpandMultiSpec_KeepsLiteralPaths(t *testing.T) {
	got, err := expandMultiSpec("no_such_a.go, no_such_b.go")
	if err != nil || len(got) != 2 || got[1] != "no_such_b.go" {
		t.Errorf("got %v, %v; want both literal paths kept, trimmed", got, err)
	}
}

func TestDetectLang(t *testing.T) {
	cases := map[string]string{
		"a.go": "go", "b.PY": "python", "c.jsx": "javascript", "d.tsx": "typescript",
		"e.cs": "csharp", "f.yml": "yaml", "g.md": "markdown", "Dockerfile": "dockerfile",
		"h.mk": "makefile", "i.json": "json", "j.csv": "csv",
	}
	for name, want := range cases {
		got, ok := detectLang(name)
		if !ok || got != want {
			t.Errorf("detectLang(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
	if _, ok := detectLang("notes.txt"); ok {
		t.Error("detectLang(notes.txt) should not detect a language")
	}
	if _, ok := detectLang("infra.tf"); ok {
		t.Error("detectLang(infra.tf) should not detect a language (doMap has no HCL)")
	}
}

func TestRunMultiFile_RejectsMutatingOp(t *testing.T) {
	dir := t.TempDir()
	a := mfWrite(t, dir, "a.txt", "one\ntwo\n")
	mfWrite(t, dir, "b.txt", "one\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "delete"}); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if lines, _ := readLines(a); len(lines) != 2 {
		t.Errorf("file was modified: %v", lines)
	}
}

func TestRunMultiFile_RejectsStream(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.txt", "one\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "find", match: "one", stream: true}); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestRunMultiFile_FindRequiresMatch(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.txt", "one\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "find"}); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestRunMultiFile_UnknownMapLang(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.txt", "one\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "map", lang: "cobol"}); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

// Files without hits must not reach doFind (which would os.Exit and kill the test binary).
func TestRunMultiFile_FindNoHitsAnywhere(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.txt", "one\n")
	mfWrite(t, dir, "b.txt", "two\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "find", match: "zzz", nth: 1}); code != 1 {
		t.Errorf("code = %d, want 1 when nothing matches anywhere", code)
	}
}

func TestRunMultiFile_FindSomeHits(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.txt", "needle here\n")
	mfWrite(t, dir, "b.txt", "nothing\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*.txt"), op: "find", match: "needle", nth: 1, x: true}); code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}

func TestRunMultiFile_SkipsUnreadable(t *testing.T) {
	dir := t.TempDir()
	good := mfWrite(t, dir, "good.txt", "one\n")
	spec := filepath.Join(dir, "missing.txt") + "," + good
	if code := runMultiFile(multiOpts{spec: spec, op: "find", match: "one", nth: 1, x: true}); code != 0 {
		t.Errorf("code = %d, want 0 (a missing file is skipped, not fatal)", code)
	}
}

// If the unsupported file were not skipped, doMap would os.Exit and this test binary would die.
func TestRunMultiFile_MapContinuesPastUnknownExt(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.go", "package a\n\nfunc A() {}\n")
	mfWrite(t, dir, "b.xyz", "whatever\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*"), op: "map"}); code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}

func TestRunMultiFile_ShowBlockUsesPerFileLang(t *testing.T) {
	dir := t.TempDir()
	mfWrite(t, dir, "a.js", "function beta() {\n  return 1;\n}\n")
	mfWrite(t, dir, "b.rs", "fn beta() {\n    1\n}\n")
	if code := runMultiFile(multiOpts{spec: filepath.Join(dir, "*"), op: "show", block: "beta"}); code != 0 {
		t.Errorf("code = %d, want 0", code)
	}
}
