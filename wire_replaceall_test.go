package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// These tests run the real binary, so they cover the wiring in main(): the
// literal replaceall now verifies the written file, prints PASS or FAIL, and
// under -v shows a window for every changed line instead of the first hit.

func wraRun(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var se bytes.Buffer
	cmd.Stderr = &se
	code := 0
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run failed: %v", err)
		}
		code = ee.ExitCode()
	}
	return se.String(), code
}

func wraFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "wra.txt")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestWRA_Binary(t *testing.T) {
	bin := clhBuild(t)
	const orig = "a x\nb\nx x\nc\nx\n"

	t.Run("pass_verbose", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match", "x", "-text", "Y", "-v")
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		for _, w := range []string{"VERIFY: PASS", "4 occurrence(s)", "changed lines: 1,3,5", "=== STATS ==="} {
			if !strings.Contains(out, w) {
				t.Errorf("stderr lacks %q:\n%s", w, out)
			}
		}
		if n := strings.Count(out, "=== VERIFY ==="); n != 1 {
			t.Errorf("VERIFY windows printed %d times, want 1:\n%s", n, out)
		}
		if n := strings.Count(out, "\n> "); n != 3 {
			t.Errorf("marked lines = %d, want 3:\n%s", n, out)
		}
		b, _ := os.ReadFile(p)
		if got := strings.ReplaceAll(string(b), "\r\n", "\n"); got != "a Y\nb\nY Y\nc\nY\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("pass_without_v", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match", "x", "-text", "Y")
		if code != 0 {
			t.Fatalf("exit %d:\n%s", code, out)
		}
		if !strings.Contains(out, "VERIFY: PASS") {
			t.Errorf("verification must run without -v:\n%s", out)
		}
		if strings.Contains(out, "=== VERIFY ===") {
			t.Errorf("windows must only print with -v:\n%s", out)
		}
	})

	t.Run("no_match_exit1", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match", "zzz", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if !strings.Contains(out, "No lines contain: zzz") {
			t.Errorf("stderr:\n%s", out)
		}
	})
}
