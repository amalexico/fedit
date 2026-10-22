package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func lhxSum(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// lhxFile writes a file with the given text and returns its path.
func lhxFile(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func lhxRes() replaceAllResult {
	return replaceAllResult{Lines: []string{"a bar", "b"}, Changed: []int{1}, Occurrences: 2}
}

func lhxVerify(v replaceVerify, err error) func(string) (replaceVerify, error) {
	return func(string) (replaceVerify, error) { return v, err }
}

func TestLH_FinishVerifiedNotesPass(t *testing.T) {
	lnxStart(t)
	p := lhxFile(t, "a foo\nb\n")
	code := finishVerified(lhxRes(), p, "done\n", false, io.Discard, writeLines, lhxVerify(replaceVerify{OK: true}, nil))
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	lnxWant(t, "replacements", "2")
	lnxWant(t, "lines_changed", "1")
	lnxWant(t, "verify", "PASS")
	lnxWant(t, "sha256_before", lhxSum([]byte("a foo\nb\n")))
	after, _ := os.ReadFile(p)
	lnxWant(t, "sha256_after", lhxSum(after))
}

func TestLH_FinishVerifiedNotesFail(t *testing.T) {
	lnxStart(t)
	p := lhxFile(t, "a foo\nb\n")
	code := finishVerified(lhxRes(), p, "done\n", false, io.Discard, writeLines, lhxVerify(replaceVerify{OK: false}, nil))
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	lnxWant(t, "verify", "FAIL")
	lnxWant(t, "replacements", "2")
}

func TestLH_FinishVerifiedNotesUnreadable(t *testing.T) {
	lnxStart(t)
	p := lhxFile(t, "a foo\nb\n")
	code := finishVerified(lhxRes(), p, "done\n", false, io.Discard, writeLines, lhxVerify(replaceVerify{}, errors.New("gone")))
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	lnxWant(t, "verify", "FAIL")
}

func TestLH_FinishVerifiedWriteFailureNotesNothing(t *testing.T) {
	lnxStart(t)
	p := lhxFile(t, "a foo\nb\n")
	bad := func(string, []string) error { return errors.New("disk full") }
	code := finishVerified(lhxRes(), p, "done\n", false, io.Discard, bad, lhxVerify(replaceVerify{OK: true}, nil))
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(logNotes) != 0 {
		t.Errorf("a failed write must note nothing, got %v", logNotes)
	}
}

func TestLH_FinishVerifiedNoSessionNotesNothing(t *testing.T) {
	logRun, logNotes = nil, nil
	p := lhxFile(t, "a foo\nb\n")
	code := finishVerified(lhxRes(), p, "done\n", false, io.Discard, writeLines, lhxVerify(replaceVerify{OK: true}, nil))
	if code != 0 || len(logNotes) != 0 {
		t.Fatalf("code = %d, notes = %v", code, logNotes)
	}
}
