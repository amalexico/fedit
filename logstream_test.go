package main

import (
	"io"
	"os"
	"testing"
)

func TestLK_StreamLiteralNotes(t *testing.T) {
	lnxStart(t)
	orig := "foo a\nb foo foo\nc\n"
	p := lhxFile(t, orig)
	if code := runStreamLiteralVerified(p, "foo", "bar", io.Discard); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	lnxWant(t, "replacements", "3")
	lnxWant(t, "lines_changed", "2")
	lnxWant(t, "verify", "PASS")
	lnxWant(t, "sha256_before", lhxSum([]byte(orig)))
	after, _ := os.ReadFile(p)
	lnxWant(t, "sha256_after", lhxSum(after))
}

func TestLK_StreamRegexNotes(t *testing.T) {
	lnxStart(t)
	orig := "a1\nb\nc22\n"
	p := lhxFile(t, orig)
	if code := runStreamRegexVerified(p, "[0-9]+", "N", io.Discard); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	lnxWant(t, "replacements", "2")
	lnxWant(t, "lines_changed", "2")
	lnxWant(t, "verify", "PASS")
	lnxWant(t, "sha256_before", lhxSum([]byte(orig)))
}

func TestLK_StreamNoMatchNotesNothing(t *testing.T) {
	lnxStart(t)
	p := lhxFile(t, "a\nb\n")
	if code := runStreamLiteralVerified(p, "zzz", "y", io.Discard); code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if len(logNotes) != 0 {
		t.Errorf("a run that changed nothing must note nothing, got %v", logNotes)
	}
}

func TestLK_StreamNoSessionNotesNothing(t *testing.T) {
	logRun, logNotes = nil, nil
	p := lhxFile(t, "foo\n")
	if code := runStreamLiteralVerified(p, "foo", "bar", io.Discard); code != 0 || len(logNotes) != 0 {
		t.Fatalf("code = %d, notes = %v", code, logNotes)
	}
}
