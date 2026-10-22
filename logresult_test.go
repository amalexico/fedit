package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// lnxStart installs a permissive logging session that writes to a fake log
// and returns the lines it receives. The globals are reset when the test ends.
func lnxStart(t *testing.T) *[]string {
	t.Helper()
	var lines []string
	fake := logIO{
		Append: func(path, line string) error { lines = append(lines, line); return nil },
	}
	s := logSettings{Mode: logPermissive, File: "x.log", Level: logThresholdAll, Scope: logScopeAll}
	logRun = newLogSession(s, "replaceall", time.Now, fake, func() {}, nil)
	logRunStrict, logRunEnded, logWarned = false, false, false
	logNotes = nil
	t.Cleanup(func() {
		logRun, logRunStrict, logRunEnded, logWarned = nil, false, false, false
		logNotes = nil
	})
	return &lines
}

func lnxWant(t *testing.T, key, want string) {
	t.Helper()
	got, ok := logNoteValue(key)
	if !ok || got != want {
		t.Errorf("note %s = %q (present %v), want %q", key, got, ok, want)
	}
}

func TestLN_NoSessionIsNoop(t *testing.T) {
	logRun, logNotes = nil, nil
	logNote("replacements", "3")
	logNoteReplace(3, 2)
	logNoteBatch([]batchResult{{"a.go", "OK", ""}}, true)
	if len(logNotes) != 0 || logTakeNotes() != nil {
		t.Fatalf("notes were kept without a session: %v", logNotes)
	}
	if got := logFileSHA256("whatever"); got != "" {
		t.Errorf("logFileSHA256 without a session = %q, want empty", got)
	}
}

func TestLN_FinishWritesNotes(t *testing.T) {
	lines := lnxStart(t)
	logNote("replacements", "3")
	logNoteVerify("PASS")
	if code := logFinish(0); code != 0 {
		t.Fatalf("logFinish = %d, want 0", code)
	}
	if len(*lines) != 1 {
		t.Fatalf("got %d records, want 1: %v", len(*lines), *lines)
	}
	rec := (*lines)[0]
	for _, w := range []string{"phase=end", "replacements=3", "verify=PASS"} {
		if !strings.Contains(rec, w) {
			t.Errorf("end record lacks %s: %s", w, rec)
		}
	}
	if len(logNotes) != 0 {
		t.Errorf("notes not consumed: %v", logNotes)
	}
}

func TestLN_LastValueWins(t *testing.T) {
	lnxStart(t)
	logNote("verify", "FAIL")
	logNote("verify", "PASS")
	if len(logNotes) != 1 {
		t.Fatalf("got %d notes, want 1", len(logNotes))
	}
	lnxWant(t, "verify", "PASS")
}

func TestLN_ReplaceAndHashes(t *testing.T) {
	lnxStart(t)
	logNoteReplace(7, 4)
	logNoteHashes("aa", "")
	lnxWant(t, "replacements", "7")
	lnxWant(t, "lines_changed", "4")
	lnxWant(t, "sha256_before", "aa")
	if _, ok := logNoteValue("sha256_after"); ok {
		t.Error("an empty after hash must not be noted")
	}
	logNoteHashes("", "bb")
	lnxWant(t, "sha256_after", "bb")
}

func TestLN_Batch(t *testing.T) {
	lnxStart(t)
	logNoteBatch([]batchResult{
		{"a.go", "OK", "x"}, {"b.go", "SKIP", "no match"}, {"c.go", "FAIL", "y"}, {"d.go", "ERROR", "z"},
	}, true)
	lnxWant(t, "files_total", "4")
	lnxWant(t, "files_ok", "1")
	lnxWant(t, "files_skip", "1")
	lnxWant(t, "files_fail", "2")
	lnxWant(t, "verify", "FAIL")
	lnxWant(t, "results", "OK a.go; SKIP b.go; FAIL c.go; ERROR d.go")
}

func TestLN_BatchVerifyStates(t *testing.T) {
	lnxStart(t)
	logNoteBatch([]batchResult{{"a.go", "OK", ""}}, true)
	lnxWant(t, "verify", "PASS")
	logNoteBatch([]batchResult{{"a.go", "OK", ""}}, false)
	lnxWant(t, "verify", "none")
	logNoteBatch([]batchResult{{"a.go", "SKIP", ""}}, true)
	lnxWant(t, "verify", "none")
}

func TestLN_FileSHA256(t *testing.T) {
	lnxStart(t)
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := logFileSHA256(p); got != want {
		t.Errorf("logFileSHA256 = %q, want %q", got, want)
	}
	if got := logFileSHA256(filepath.Join(t.TempDir(), "missing")); got != "" {
		t.Errorf("a missing file must give an empty hash, got %q", got)
	}
}
