package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// -quiet must silence the per-file batch report when every file succeeded, and
// must still report skips. Builds and runs the real binary.

func buildQuietBin(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the binary; skipped with -short")
	}
	name := "fedit_quiet_test"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	return bin
}

func runQuiet(t *testing.T, bin, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = ee.ExitCode()
	}
	return string(out), code
}

func TestQuietDestReport(t *testing.T) {
	bin := buildQuietBin(t)
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.go", "package a\n\nfunc one() {\n\tprintln(1)\n}\n")
	base := []string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one"}
	with := func(extra ...string) []string { return append(append([]string{}, base...), extra...) }

	write("b.go", "package b\n")
	out, code := runQuiet(t, bin, dir, with("-dest", "b.go", "-after", "1")...)
	if code != 0 || !strings.Contains(out, "file(s) succeeded") {
		t.Errorf("without -quiet: want the report and exit 0, got code %d: %q", code, out)
	}

	write("b.go", "package b\n")
	out, code = runQuiet(t, bin, dir, with("-dest", "b.go", "-after", "1", "-quiet")...)
	if code != 0 || out != "" {
		t.Errorf("-quiet, all OK: want no output and exit 0, got code %d: %q", code, out)
	}

	write("b.go", "package b\n")
	out, code = runQuiet(t, bin, dir, with("-dest", "b.go,nope.go", "-after", "1", "-quiet")...)
	if code != 0 || !strings.Contains(out, "SKIP") || !strings.Contains(out, "nope.go") {
		t.Errorf("-quiet with a skipped file: want the SKIP line and exit 0, got code %d: %q", code, out)
	}
}
