package main

import (
	"strings"
	"testing"
)

// Binary tests for the multi -file replaceall: every file that is written is
// read back and reports VERIFY: PASS in its own report line; a file without a
// match stays a SKIP and the run keeps its exit code (0 when at least one file
// succeeded, 1 when none did).

func TestWRM_Binary(t *testing.T) {
	bin := clhBuild(t)

	t.Run("verified_pass_per_file", func(t *testing.T) {
		a := wraFile(t, "a x\nb\nx x\n")
		b := wraFile(t, "nothing\n")
		c := wraFile(t, "x\n")
		out, code := wraRun(t, bin, "-file", a+","+b+","+c, "-op", "replaceall", "-match", "x", "-text", "Y")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		if n := strings.Count(out, "VERIFY: PASS"); n != 2 {
			t.Errorf("VERIFY: PASS printed %d times, want 2 (one per written file):\n%s", n, out)
		}
		if !strings.Contains(out, "2/3 file(s) succeeded") {
			t.Errorf("summary missing:\n%s", out)
		}
		if got := wrrRead(t, a); got != "a Y\nb\nY Y\n" {
			t.Errorf("a = %q", got)
		}
		if got := wrrRead(t, b); got != "nothing\n" {
			t.Errorf("b changed: %q", got)
		}
		if got := wrrRead(t, c); got != "Y\n" {
			t.Errorf("c = %q", got)
		}
	})

	t.Run("no_file_matches_exit1", func(t *testing.T) {
		a := wraFile(t, "alpha\n")
		b := wraFile(t, "beta\n")
		out, code := wraRun(t, bin, "-file", a+","+b, "-op", "replaceall", "-match", "zzz", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, a); got != "alpha\n" {
			t.Errorf("a changed: %q", got)
		}
		if got := wrrRead(t, b); got != "beta\n" {
			t.Errorf("b changed: %q", got)
		}
	})
}
