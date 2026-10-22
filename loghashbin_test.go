package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Binary-level checks of the result fields in the end record. They run as
// subtests of TestLW_Bin, which builds the binary once.

// lhbSetup starts a permissive audit log next to a fresh test file.
func lhbSetup(t *testing.T) (logp, f string) {
	t.Helper()
	dir, f := lwxSetup(t)
	logp = filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	return logp, f
}

// lhbEnd returns the end record of a run that wrote a begin and an end record.
func lhbEnd(t *testing.T, logp string) string {
	t.Helper()
	lines := lwxLog(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	return lines[1]
}

func lhbField(t *testing.T, rec, key, want string) {
	t.Helper()
	if got, ok := spoolLogField(rec, key); !ok || got != want {
		t.Errorf("%s = %q (present %v), want %q in %q", key, got, ok, want, rec)
	}
}

func lwxBinWriteOpHashes(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	before, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	_, se, code := orxRun(t, bin, "-file", f, "-op", "insertafter", "-match", "alpha", "-text", "zzzline")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	after, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	e := lhbEnd(t, logp)
	lhbField(t, e, "sha256_before", lhxSum(before))
	lhbField(t, e, "sha256_after", lhxSum(after))
	if lhxSum(before) == lhxSum(after) {
		t.Error("the file did not change")
	}
}

func lwxBinReadOpNoHashes(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	if _, se, code := orxRun(t, bin, "-file", f, "-op", "find", "-match", "alpha"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if e := lhbEnd(t, logp); strings.Contains(e, "sha256_") {
		t.Errorf("a read operation must not log hashes: %q", e)
	}
}

func lwxBinReplaceAllResult(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	before, _ := os.ReadFile(f)
	if _, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "beta"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	after, _ := os.ReadFile(f)
	e := lhbEnd(t, logp)
	lhbField(t, e, "verify", "PASS")
	lhbField(t, e, "sha256_before", lhxSum(before))
	lhbField(t, e, "sha256_after", lhxSum(after))
	for _, key := range []string{"replacements", "lines_changed"} {
		if _, ok := spoolLogField(e, key); !ok {
			t.Errorf("end record lacks %s: %q", key, e)
		}
	}
	if n := strings.Count(e, "sha256_before="); n != 1 {
		t.Errorf("sha256_before appears %d times: %q", n, e)
	}
}

func lwxBinMultiFileResult(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	g := orxWrite(t, filepath.Dir(f), "b.txt", lwxOrig)
	if _, se, code := orxRun(t, bin, "-file", f+","+g, "-op", "replaceall", "-match", "alpha", "-text", "beta"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	e := lhbEnd(t, logp)
	lhbField(t, e, "files_total", "2")
	lhbField(t, e, "files_ok", "2")
	lhbField(t, e, "verify", "PASS")
	if strings.Contains(e, "sha256_") {
		t.Errorf("a multi-file run reports per-file results, not hashes: %q", e)
	}
}
