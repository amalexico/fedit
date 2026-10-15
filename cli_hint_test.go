package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// These tests build the real fedit binary and run it, so they cover the two
// hint sites in main() (insertafter and insertbefore with -sourcefile and
// -sourceblock) that the pure-function tests cannot reach. The hint goes to
// stderr; the insert itself is reported on stdout.

const clhSrc = "package main\n\nfunc Up(s string) string {\n\treturn strings.ToUpper(s)\n}\n"

const clhDestNoImport = "package main\n\nfunc Anchor() {\n}\n"

const clhDestWithImport = "package main\n\nimport \"strings\"\n\nfunc Anchor() {\n}\n"

func clhBuild(t *testing.T) string {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping binary build in -short mode")
	}
	name := "fedit_clh"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	bin := filepath.Join(t.TempDir(), name)
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
	return bin
}

func clhWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func clhRun(t *testing.T, bin, dir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run failed: %v", err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

func clhRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCLIHint_InsertAfter_MissingImport(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	clhWrite(t, dir, "src.go", clhSrc)
	dest := clhWrite(t, dir, "dest.go", clhDestNoImport)
	_, se, code := clhRun(t, bin, dir, "-file", "dest.go", "-op", "insertafter", "-match", "Anchor", "-sourcefile", "src.go", "-sourceblock", "Up", "-lang", "go")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, se)
	}
	if !strings.Contains(se, "hint:") || !strings.Contains(se, "strings") || !strings.Contains(se, "dest.go") {
		t.Errorf("want a strings import hint naming dest.go on stderr, got %q", se)
	}
	if !strings.Contains(clhRead(t, dest), "strings.ToUpper") {
		t.Errorf("the block was not inserted into dest.go")
	}
}

func TestCLIHint_InsertBefore_MissingImport(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	clhWrite(t, dir, "src.go", clhSrc)
	dest := clhWrite(t, dir, "dest.go", clhDestNoImport)
	_, se, code := clhRun(t, bin, dir, "-file", "dest.go", "-op", "insertbefore", "-match", "Anchor", "-sourcefile", "src.go", "-sourceblock", "Up", "-lang", "go")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, se)
	}
	if !strings.Contains(se, "hint:") || !strings.Contains(se, "strings") || !strings.Contains(se, "dest.go") {
		t.Errorf("want a strings import hint naming dest.go on stderr, got %q", se)
	}
	if !strings.Contains(clhRead(t, dest), "strings.ToUpper") {
		t.Errorf("the block was not inserted into dest.go")
	}
}

func TestCLIHint_BlockBranch_BothOps(t *testing.T) {
	bin := clhBuild(t)
	for _, op := range []string{"insertafter", "insertbefore"} {
		dir := t.TempDir()
		clhWrite(t, dir, "src.go", clhSrc)
		dest := clhWrite(t, dir, "dest.go", clhDestNoImport)
		_, se, code := clhRun(t, bin, dir, "-file", "dest.go", "-op", op, "-block", "Anchor", "-lang", "go", "-sourcefile", "src.go", "-sourceblock", "Up")
		if code != 0 {
			t.Fatalf("%s: exit code %d, stderr: %s", op, code, se)
		}
		if !strings.Contains(se, "hint:") || !strings.Contains(se, "strings") {
			t.Errorf("%s -block: want a strings import hint on stderr, got %q", op, se)
		}
		if !strings.Contains(clhRead(t, dest), "strings.ToUpper") {
			t.Errorf("%s -block: the block was not inserted", op)
		}
	}
}

func TestCLIHint_AlreadyImported_NoHint(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	clhWrite(t, dir, "src.go", clhSrc)
	clhWrite(t, dir, "dest.go", clhDestWithImport)
	_, se, code := clhRun(t, bin, dir, "-file", "dest.go", "-op", "insertafter", "-match", "Anchor", "-sourcefile", "src.go", "-sourceblock", "Up", "-lang", "go")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, se)
	}
	if strings.Contains(se, "hint:") {
		t.Errorf("want no hint when strings is already imported, got %q", se)
	}
}

func TestCLIHint_TxtDestination_NoHint(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	clhWrite(t, dir, "src.go", clhSrc)
	dest := clhWrite(t, dir, "dest.txt", "first\nAnchor\nlast\n")
	_, se, code := clhRun(t, bin, dir, "-file", "dest.txt", "-op", "insertafter", "-match", "Anchor", "-sourcefile", "src.go", "-sourceblock", "Up", "-lang", "go")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, se)
	}
	if strings.Contains(se, "hint:") {
		t.Errorf("want no hint for a .txt destination, got %q", se)
	}
	if !strings.Contains(clhRead(t, dest), "strings.ToUpper") {
		t.Errorf("the block was not inserted into dest.txt")
	}
}

func TestCLIHint_PlainText_NoHint(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	dest := clhWrite(t, dir, "dest.go", clhDestNoImport)
	_, se, code := clhRun(t, bin, dir, "-file", "dest.go", "-op", "insertafter", "-match", "Anchor", "-text", "var _ = strings.ToUpper")
	if code != 0 {
		t.Fatalf("exit code %d, stderr: %s", code, se)
	}
	if strings.Contains(se, "hint:") {
		t.Errorf("want no hint without -sourcefile and -sourceblock, got %q", se)
	}
	if !strings.Contains(clhRead(t, dest), "strings.ToUpper") {
		t.Errorf("the text was not inserted into dest.go")
	}
}
