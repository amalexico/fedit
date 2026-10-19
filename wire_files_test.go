package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Binary tests for -files GLOB with replaceall: the glob runs through the
// same verified multi-file runner as a comma list, so every written file is
// read back and reports VERIFY: PASS in its own report line.

func wrfDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestWRF_Binary(t *testing.T) {
	bin := clhBuild(t)

	t.Run("literal_glob_verified", func(t *testing.T) {
		dir := wrfDir(t, map[string]string{"a.txt": "a x\nb\nx x\n", "b.txt": "nothing\n", "c.txt": "x\n"})
		out, code := wraRun(t, bin, "-files", filepath.Join(dir, "*.txt"), "-op", "replaceall", "-match", "x", "-text", "Y")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		if n := strings.Count(out, "VERIFY: PASS"); n != 2 {
			t.Errorf("VERIFY: PASS printed %d times, want 2:\n%s", n, out)
		}
		if got := wrrRead(t, filepath.Join(dir, "a.txt")); got != "a Y\nb\nY Y\n" {
			t.Errorf("a.txt = %q", got)
		}
		if got := wrrRead(t, filepath.Join(dir, "b.txt")); got != "nothing\n" {
			t.Errorf("b.txt changed: %q", got)
		}
		if got := wrrRead(t, filepath.Join(dir, "c.txt")); got != "Y\n" {
			t.Errorf("c.txt = %q", got)
		}
	})

	t.Run("regex_glob_verified", func(t *testing.T) {
		dir := wrfDir(t, map[string]string{"a.txt": "id=1 name=ann\nskip\n", "b.txt": "id=2 name=bob\n"})
		out, code := wraRun(t, bin, "-files", filepath.Join(dir, "*.txt"), "-op", "replaceall", "-match-regex", `id=(\d+) name=(\w+)`, "-text", "$2:$1")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		if n := strings.Count(out, "VERIFY: PASS"); n != 2 {
			t.Errorf("VERIFY: PASS printed %d times, want 2:\n%s", n, out)
		}
		if got := wrrRead(t, filepath.Join(dir, "a.txt")); got != "ann:1\nskip\n" {
			t.Errorf("a.txt = %q", got)
		}
		if got := wrrRead(t, filepath.Join(dir, "b.txt")); got != "bob:2\n" {
			t.Errorf("b.txt = %q", got)
		}
	})

	t.Run("no_files_matched_exit1", func(t *testing.T) {
		dir := wrfDir(t, map[string]string{"a.txt": "x\n"})
		out, code := wraRun(t, bin, "-files", filepath.Join(dir, "*.nomatch"), "-op", "replaceall", "-match", "x", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, filepath.Join(dir, "a.txt")); got != "x\n" {
			t.Errorf("a.txt changed: %q", got)
		}
	})
}
