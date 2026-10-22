package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func lhtxReset(t *testing.T) {
	t.Helper()
	logHashPath, logHashBefore = "", ""
	t.Cleanup(func() { logHashPath, logHashBefore = "", "" })
}

func lhtxMissing(t *testing.T, key string) {
	t.Helper()
	if v, ok := logNoteValue(key); ok {
		t.Errorf("note %s = %q, want none", key, v)
	}
}

func TestLHT_Target(t *testing.T) {
	cwd := t.TempDir()
	abs := filepath.Join(cwd, "sub", "b.txt")
	cases := []struct {
		name string
		op   string
		in   logCLIInput
		want string
	}{
		{"relative file", "replace", logCLIInput{File: "a.txt", Cwd: cwd}, filepath.Join(cwd, "a.txt")},
		{"absolute file", "insertafter", logCLIInput{File: abs, Cwd: cwd}, abs},
		{"write op that may create the file", "write", logCLIInput{File: "n.txt", Cwd: cwd}, filepath.Join(cwd, "n.txt")},
		{"read op", "show", logCLIInput{File: "a.txt", Cwd: cwd}, ""},
		{"guided parent", "guided", logCLIInput{File: "a.txt", Cwd: cwd}, ""},
		{"file list", "replaceall", logCLIInput{File: "a.txt,b.txt", Cwd: cwd}, ""},
		{"glob in file", "replaceall", logCLIInput{File: "*.go", Cwd: cwd}, ""},
		{"files glob", "replaceall", logCLIInput{Files: "*.go", Cwd: cwd}, ""},
		{"dest list", "copy", logCLIInput{File: "a.txt", Dest: "b.txt", Cwd: cwd}, ""},
		{"no file", "replace", logCLIInput{Cwd: cwd}, ""},
		{"unknown op", "frobnicate", logCLIInput{File: "a.txt", Cwd: cwd}, ""},
	}
	for _, c := range cases {
		if got := logHashTarget(c.op, c.in); got != c.want {
			t.Errorf("%s: target = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestLHT_StartAndFinishNoteHashes(t *testing.T) {
	lnxStart(t)
	lhtxReset(t)
	p := lhxFile(t, "one\n")
	logHashStart("insertafter", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	if err := os.WriteFile(p, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logHashFinish()
	lnxWant(t, "sha256_before", lhxSum([]byte("one\n")))
	lnxWant(t, "sha256_after", lhxSum([]byte("two\n")))
	if logHashPath != "" {
		t.Errorf("logHashPath not cleared: %q", logHashPath)
	}
}

func TestLHT_HashesAlreadyNotedAreKept(t *testing.T) {
	lnxStart(t)
	lhtxReset(t)
	p := lhxFile(t, "one\n")
	logHashStart("replaceall", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	logNoteHashes("B", "A")
	if err := os.WriteFile(p, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logHashFinish()
	lnxWant(t, "sha256_before", "B")
	lnxWant(t, "sha256_after", "A")
}

func TestLHT_NewFileHasOnlyAfter(t *testing.T) {
	lnxStart(t)
	lhtxReset(t)
	p := filepath.Join(t.TempDir(), "new.txt")
	logHashStart("write", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	if err := os.WriteFile(p, []byte("fresh\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logHashFinish()
	lhtxMissing(t, "sha256_before")
	lnxWant(t, "sha256_after", lhxSum([]byte("fresh\n")))
}

func TestLHT_ReadOpHashesNothing(t *testing.T) {
	lnxStart(t)
	lhtxReset(t)
	p := lhxFile(t, "one\n")
	logHashStart("show", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	logHashFinish()
	if len(logNotes) != 0 {
		t.Errorf("a read operation must note nothing, got %v", logNotes)
	}
}

func TestLHT_NoSessionDoesNothing(t *testing.T) {
	logRun, logNotes = nil, nil
	lhtxReset(t)
	p := lhxFile(t, "one\n")
	logHashStart("insertafter", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	if logHashPath != "" || logHashBefore != "" {
		t.Errorf("state kept without a session: %q %q", logHashPath, logHashBefore)
	}
	logHashFinish()
	if len(logNotes) != 0 {
		t.Errorf("notes without a session: %v", logNotes)
	}
}

func TestLHT_FinishRecordCarriesHashes(t *testing.T) {
	lines := lnxStart(t)
	lhtxReset(t)
	p := lhxFile(t, "one\n")
	logHashStart("delete", logCLIInput{File: p, Cwd: filepath.Dir(p)})
	if err := os.WriteFile(p, []byte("two\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logFinish(0)
	if len(*lines) != 1 {
		t.Fatalf("got %d records: %v", len(*lines), *lines)
	}
	rec := (*lines)[0]
	for _, w := range []string{"sha256_before=" + lhxSum([]byte("one\n")), "sha256_after=" + lhxSum([]byte("two\n"))} {
		if !strings.Contains(rec, w) {
			t.Errorf("end record lacks %s: %s", w, rec)
		}
	}
}
