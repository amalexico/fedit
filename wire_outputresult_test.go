package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// orxRun runs the fedit binary and returns stdout, stderr and the exit code
// separately, because -outputresult changes what goes to each stream.
func orxRun(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("run %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

func orxWrite(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

func orxRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func orxNorm(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// A successful run writes the operation's report to FILE (UTF-8, no BOM) and
// leaves exactly one summary line on the terminal and nothing on stderr.
func TestOR_Success(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\nbeta\nalpha\n")
	out := filepath.Join(dir, "result.txt")

	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma", "-outputresult", out)
	if code != 0 {
		t.Fatalf("exit %d, want 0\nstdout: %q\nstderr: %q", code, so, se)
	}
	if se != "" {
		t.Errorf("stderr should be empty on success, got %q", se)
	}
	lines := strings.Split(strings.TrimRight(so, "\r\n"), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], out) {
		t.Errorf("terminal should get one summary line naming the result file, got %q", so)
	}
	if got := orxNorm(orxRead(t, f)); got != "gamma\nbeta\ngamma\n" {
		t.Errorf("target file not edited as expected, got %q", got)
	}
	res := orxRead(t, out)
	if strings.HasPrefix(res, "\xef\xbb\xbf") {
		t.Errorf("result file must not start with a BOM")
	}
	if !strings.Contains(res, "VERIFY: PASS") {
		t.Errorf("result file should hold the report with VERIFY: PASS, got %q", res)
	}
}

// FILE must never be an edited file: refuse before anything is changed.
func TestOR_RefuseSameFile(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\n")

	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma", "-outputresult", f)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout: %q\nstderr: %q", code, so, se)
	}
	if !strings.Contains(se, "-outputresult") {
		t.Errorf("stderr should explain the -outputresult refusal, got %q", se)
	}
	if got := orxNorm(orxRead(t, f)); got != "alpha\n" {
		t.Errorf("file must be unchanged after the refusal, got %q", got)
	}
}

// FILE is overwritten by default and appended to with -outappend.
func TestOR_OverwriteAndAppend(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\n")
	out := filepath.Join(dir, "result.txt")

	steps := []struct {
		old, new string
		extra    []string
		wantPass int
	}{
		{"alpha", "gamma", nil, 1},
		{"gamma", "delta", nil, 1},
		{"delta", "omega", []string{"-outappend"}, 2},
	}
	for i, s := range steps {
		args := []string{"-file", f, "-op", "replaceall", "-match", s.old, "-text", s.new, "-outputresult", out}
		args = append(args, s.extra...)
		so, se, code := orxRun(t, bin, args...)
		if code != 0 {
			t.Fatalf("step %d: exit %d, want 0\nstdout: %q\nstderr: %q", i+1, code, so, se)
		}
		if n := strings.Count(orxRead(t, out), "VERIFY: PASS"); n != s.wantPass {
			t.Errorf("step %d: result file has %d VERIFY: PASS lines, want %d", i+1, n, s.wantPass)
		}
	}
}
