package main

import (
	"bytes"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

var rvOrig = []string{"a x", "b", "x x", "c", "x"}

func rvFile(t *testing.T, lines []string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rv.txt")
	if err := writeLines(p, lines); err != nil {
		t.Fatal(err)
	}
	return p
}

func rvNeed(t *testing.T, s string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(s, w) {
			t.Errorf("output lacks %q:\n%s", w, s)
		}
	}
}

func TestRV_Pass(t *testing.T) {
	p := rvFile(t, rvOrig)
	var out bytes.Buffer
	code := runReplaceAllVerified(rvOrig, p, "x", "Y", false, &out, writeLines)
	if code != 0 {
		t.Fatalf("code = %d:\n%s", code, out.String())
	}
	got, _ := readLines(p)
	if strings.Join(got, "|") != "a Y|b|Y Y|c|Y" {
		t.Errorf("file = %q", got)
	}
	rvNeed(t, out.String(), "Replaced 'x' on 3 line(s) (5 total)", "VERIFY: PASS", "4 occurrence(s)", "changed lines: 1,3,5")
	if strings.Contains(out.String(), "=== VERIFY ===") {
		t.Errorf("windows must only print with verbose:\n%s", out.String())
	}
}

func TestRV_PassVerboseShowsEveryHit(t *testing.T) {
	p := rvFile(t, rvOrig)
	var out bytes.Buffer
	if code := runReplaceAllVerified(rvOrig, p, "x", "Y", true, &out, writeLines); code != 0 {
		t.Fatalf("code = %d:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "=== VERIFY ===")
	if n := strings.Count(out.String(), "\n> "); n != 3 {
		t.Errorf("marked lines = %d, want 3:\n%s", n, out.String())
	}
}

func TestRV_ReplacementContainsMatchPasses(t *testing.T) {
	in := []string{"a a", "b"}
	p := rvFile(t, in)
	var out bytes.Buffer
	if code := runReplaceAllVerified(in, p, "a", "aa", false, &out, writeLines); code != 0 {
		t.Fatalf("code = %d:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "VERIFY: PASS", "remaining 4, expected 4")
}

func TestRV_SkippedHitFailsWithExit3(t *testing.T) {
	p := rvFile(t, rvOrig)
	skip := func(path string, ls []string) error {
		cp := append([]string(nil), ls...)
		cp[2] = "x"
		return writeLines(path, cp)
	}
	var out bytes.Buffer
	code := runReplaceAllVerified(rvOrig, p, "x", "Y", true, &out, skip)
	if code != 3 {
		t.Fatalf("code = %d, want 3:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "VERIFY: FAIL", "mismatched lines: 3", "   3 | x\n")
}

func TestRV_TruncatedWriteFailsWithExit3(t *testing.T) {
	p := rvFile(t, rvOrig)
	cut := func(path string, ls []string) error { return writeLines(path, ls[:2]) }
	var out bytes.Buffer
	code := runReplaceAllVerified(rvOrig, p, "x", "Y", false, &out, cut)
	if code != 3 {
		t.Fatalf("code = %d, want 3:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "VERIFY: FAIL", "lines on disk 2, expected 5")
}

func TestRV_NoMatchExits1AndLeavesFile(t *testing.T) {
	p := rvFile(t, rvOrig)
	var out bytes.Buffer
	if code := runReplaceAllVerified(rvOrig, p, "zzz", "Y", false, &out, writeLines); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	rvNeed(t, out.String(), "No lines contain: zzz")
	got, _ := readLines(p)
	if strings.Join(got, "|") != strings.Join(rvOrig, "|") {
		t.Errorf("file changed: %q", got)
	}
}

func TestRV_WriteErrorExits1(t *testing.T) {
	p := rvFile(t, rvOrig)
	boom := func(string, []string) error { return errors.New("boom") }
	var out bytes.Buffer
	if code := runReplaceAllVerified(rvOrig, p, "x", "Y", false, &out, boom); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	rvNeed(t, out.String(), "Error writing file: boom")
}

func TestRV_UnreadableAfterWriteExits3(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "never_written.txt")
	nop := func(string, []string) error { return nil }
	var out bytes.Buffer
	code := runReplaceAllVerified(rvOrig, missing, "x", "Y", false, &out, nop)
	if code != 3 {
		t.Fatalf("code = %d, want 3:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "VERIFY:", "could not re-read")
}
