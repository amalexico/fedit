package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// A failed run must not touch the result file: the captured report goes to
// stderr, the exit code is kept and no summary line is printed.
func TestORF_NoMatchLeavesResultFile(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\n")
	out := orxWrite(t, dir, "result.txt", "OLD\n")

	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "zzz", "-text", "gamma", "-outputresult", out)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout: %q\nstderr: %q", code, so, se)
	}
	if so != "" {
		t.Errorf("no summary line expected on failure, got %q", so)
	}
	if strings.TrimSpace(se) == "" {
		t.Errorf("the captured report should be replayed on stderr, got nothing")
	}
	if got := orxNorm(orxRead(t, out)); got != "OLD\n" {
		t.Errorf("result file must be left alone on failure, got %q", got)
	}
	if got := orxNorm(orxRead(t, f)); got != "alpha\n" {
		t.Errorf("target file must be unchanged, got %q", got)
	}
}

// -outappend only makes sense with -outputresult.
func TestORF_OutappendAlone(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\n")

	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma", "-outappend")
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout: %q\nstderr: %q", code, so, se)
	}
	if !strings.Contains(se, "-outappend requires -outputresult") {
		t.Errorf("stderr should explain the problem, got %q", se)
	}
	if got := orxNorm(orxRead(t, f)); got != "alpha\n" {
		t.Errorf("file must be unchanged, got %q", got)
	}
}

// A result file in a directory that does not exist is refused before any edit.
func TestORF_MissingResultDir(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	f := orxWrite(t, dir, "a.txt", "alpha\n")
	out := filepath.Join(dir, "nope", "result.txt")

	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma", "-outputresult", out)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout: %q\nstderr: %q", code, so, se)
	}
	if !strings.Contains(se, "-outputresult") {
		t.Errorf("stderr should mention -outputresult, got %q", se)
	}
	if got := orxNorm(orxRead(t, f)); got != "alpha\n" {
		t.Errorf("file must be unchanged, got %q", got)
	}
}

// The result file may not be any entry of a comma list in -file.
func TestORF_RefuseListEntry(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	a := orxWrite(t, dir, "a.txt", "alpha\n")
	b := orxWrite(t, dir, "b.txt", "alpha\n")

	so, se, code := orxRun(t, bin, "-file", a+","+b, "-op", "replaceall", "-match", "alpha", "-text", "gamma", "-outputresult", b)
	if code != 1 {
		t.Fatalf("exit %d, want 1\nstdout: %q\nstderr: %q", code, so, se)
	}
	if !strings.Contains(se, "-outputresult") {
		t.Errorf("stderr should explain the refusal, got %q", se)
	}
	for _, p := range []string{a, b} {
		if got := orxNorm(orxRead(t, p)); got != "alpha\n" {
			t.Errorf("%s must be unchanged, got %q", filepath.Base(p), got)
		}
	}
}
