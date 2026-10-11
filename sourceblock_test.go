package main

import (
	"os"
	"path/filepath"
	"testing"
)

func sbtWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0644); err != nil {
		t.Fatal(err)
	}
}

const sbtSourceContent = "package main\n\nfunc SrcFunc() {\n\tx := 1\n\ty := 2\n}\n"

func TestResolveTextFull_SourceBlockFound(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	sbtWrite(t, src, sbtSourceContent)

	got, err := resolveTextFull("", "", nil, src, "SrcFunc", "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"func SrcFunc() {", "\tx := 1", "\ty := 2", "}"}
	if len(got) != len(want) {
		t.Fatalf("got %d line(s), want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestResolveTextFull_SourceFileMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.go")

	_, err := resolveTextFull("", "", nil, missing, "SrcFunc", "go")
	if err == nil {
		t.Fatal("expected error for missing -sourcefile, got nil")
	}
}

func TestResolveTextFull_SourceBlockMissing(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	sbtWrite(t, src, sbtSourceContent)

	_, err := resolveTextFull("", "", nil, src, "NoSuchFunc", "go")
	if err == nil {
		t.Fatal("expected error for missing -sourceblock, got nil")
	}
}

func TestResolveTextFull_SourceBlockRequiresLang(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	sbtWrite(t, src, sbtSourceContent)

	_, err := resolveTextFull("", "", nil, src, "SrcFunc", "")
	if err == nil {
		t.Fatal("expected error when -lang is empty, got nil")
	}
}

func TestResolveTextFull_SourceBlockDoesNotTouchSourceFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	sbtWrite(t, src, sbtSourceContent)

	if _, err := resolveTextFull("", "", nil, src, "SrcFunc", "go"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != sbtSourceContent {
		t.Errorf("-sourcefile was modified: %q", string(b))
	}
}

// TestSourceBlockIntegration_InsertAfter exercises the full path a real
// `fedit -op insertafter -sourcefile ... -sourceblock ...` command takes:
// resolve the block from the source file, then insert it into the
// destination via the same doInsertMatch used by plain insertafter.
func TestSourceBlockIntegration_InsertAfter(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.go")
	dst := filepath.Join(dir, "dst.go")
	sbtWrite(t, src, sbtSourceContent)
	sbtWrite(t, dst, "package main\n\nfunc Anchor() {}\n")

	newText, err := resolveTextFull("", "", nil, src, "SrcFunc", "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lines, err := readLines(dst)
	if err != nil {
		t.Fatal(err)
	}
	doInsertMatch(lines, dst, "func Anchor", 1, newText, false)

	got, err := readLines(dst)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"package main", "", "func Anchor() {}", "func SrcFunc() {", "\tx := 1", "\ty := 2", "}",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d line(s), want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d: got %q, want %q", i, got[i], want[i])
		}
	}
	// Source file must remain untouched -- -sourcefile is read-only.
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != sbtSourceContent {
		t.Errorf("-sourcefile was modified: %q", string(b))
	}
}
