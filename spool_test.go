package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func spWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", p, err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
}

func spRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func spGlob(t *testing.T, pattern string) []string {
	t.Helper()
	m, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("glob %s: %v", pattern, err)
	}
	return m
}

func spFailing() logIO {
	return logIO{
		Append: func(path, line string) error { return errors.New("locked") },
		Size:   func(path string) (int64, error) { return 42, nil },
		Rotate: func(path string) error { return nil },
	}
}

const (
	spA1Begin = "2026-10-18T08:00:00.000Z [INFO] id=a1 phase=begin op=x logged=spool"
	spA1End   = "2026-10-18T08:00:00.250Z [INFO] id=a1 phase=end exit=0 ms=250 logged=spool"
	spB2Begin = "2026-10-18T09:00:00.000Z [INFO] id=b2 phase=begin op=y logged=spool"
	spMainOld = "2026-10-18T07:00:00.000Z [INFO] id=zz phase=begin op=y"
)

func TestSP_SpoolOnFailure(t *testing.T) {
	dir := t.TempDir()
	notices := 0
	lio := newSpoolIO(spFailing(), dir, "abc123", func() { notices++ })
	for _, line := range []string{"L1", "L2"} {
		if err := lio.Append("main.log", line); err != nil {
			t.Fatalf("Append(%s) should succeed through the spool, got %v", line, err)
		}
	}
	if got := spRead(t, filepath.Join(dir, "abc123.log")); got != "L1 logged=spool\nL2 logged=spool\n" {
		t.Errorf("spool file = %q", got)
	}
	if notices != 1 {
		t.Errorf("the notice must fire exactly once, fired %d times", notices)
	}
}

func TestSP_PrimaryOK(t *testing.T) {
	dir := t.TempDir()
	notices := 0
	var got []string
	primary := logIO{
		Append: func(path, line string) error { got = append(got, path+"|"+line); return nil },
		Size:   func(path string) (int64, error) { return 0, nil },
		Rotate: func(path string) error { return nil },
	}
	lio := newSpoolIO(primary, dir, "abc123", func() { notices++ })
	if err := lio.Append("main.log", "L1"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if len(got) != 1 || got[0] != "main.log|L1" {
		t.Errorf("the primary should receive the line unchanged, got %q", got)
	}
	if m := spGlob(t, filepath.Join(dir, "*")); len(m) != 0 {
		t.Errorf("nothing should be spooled while the primary works, found %v", m)
	}
	if notices != 0 {
		t.Errorf("no notice expected, got %d", notices)
	}
}

func TestSP_SpoolFailsToo(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	spWrite(t, blocker, "x")
	lio := newSpoolIO(spFailing(), filepath.Join(blocker, "spool"), "abc123", func() {})
	if err := lio.Append("main.log", "L1"); err == nil {
		t.Errorf("when the spool cannot be written either, Append must return an error")
	}
}

func TestSP_Delegates(t *testing.T) {
	rotated := 0
	primary := spFailing()
	primary.Rotate = func(path string) error { rotated++; return nil }
	lio := newSpoolIO(primary, t.TempDir(), "abc123", func() {})
	if size, err := lio.Size("main.log"); err != nil || size != 42 {
		t.Errorf("Size = %d, %v; want 42, nil", size, err)
	}
	if err := lio.Rotate("main.log"); err != nil || rotated != 1 {
		t.Errorf("Rotate: err = %v, primary rotated %d times; want nil and 1", err, rotated)
	}
}

func TestSP_Merge(t *testing.T) {
	dir := t.TempDir()
	spool := filepath.Join(dir, "spool")
	main := filepath.Join(dir, "fedit.log")
	spWrite(t, main, spMainOld+"\n")
	spWrite(t, filepath.Join(spool, "a1.log"), spA1Begin+"\n"+spA1End+"\n")
	spWrite(t, filepath.Join(spool, "b2.log"), spB2Begin+"\n")
	spWrite(t, filepath.Join(spool, "readme.txt"), "keep me")

	n, err := mergeSpool(spool, main, fileLogIO())
	if err != nil || n != 3 {
		t.Fatalf("mergeSpool = %d, %v; want 3, nil", n, err)
	}
	want := spMainOld + "\n" +
		spA1Begin + " late=true\n" +
		spA1End + " late=true\n" +
		spB2Begin + " late=true\n"
	if got := spRead(t, main); got != want {
		t.Errorf("main log =\n%q\nwant\n%q", got, want)
	}
	left := spGlob(t, filepath.Join(spool, "*"))
	if len(left) != 1 || filepath.Base(left[0]) != "readme.txt" {
		t.Errorf("only readme.txt should remain in the spool folder, found %v", left)
	}
}

func TestSP_MergeSkipsDuplicates(t *testing.T) {
	dir := t.TempDir()
	spool := filepath.Join(dir, "spool")
	main := filepath.Join(dir, "fedit.log")
	spWrite(t, main, spA1Begin+" late=true\n")
	spWrite(t, filepath.Join(spool, "a1.log.merging"), spA1Begin+"\n"+spA1End+"\n")

	n, err := mergeSpool(spool, main, fileLogIO())
	if err != nil || n != 1 {
		t.Fatalf("mergeSpool = %d, %v; want 1, nil", n, err)
	}
	got := spRead(t, main)
	if c := strings.Count(got, "id=a1 phase=begin"); c != 1 {
		t.Errorf("the begin record must not be duplicated, found %d copies in %q", c, got)
	}
	if !strings.Contains(got, "id=a1 phase=end") {
		t.Errorf("the end record should have been merged, got %q", got)
	}
	if m := spGlob(t, filepath.Join(spool, "*")); len(m) != 0 {
		t.Errorf("the spool folder should be empty, found %v", m)
	}
}

func TestSP_MergeKeepsFileOnFailure(t *testing.T) {
	dir := t.TempDir()
	spool := filepath.Join(dir, "spool")
	main := filepath.Join(dir, "fedit.log")
	spWrite(t, filepath.Join(spool, "a1.log"), spA1Begin+"\n"+spA1End+"\n")

	bad := fileLogIO()
	bad.Append = func(path, line string) error { return errors.New("locked") }
	n, err := mergeSpool(spool, main, bad)
	if err == nil || n != 0 {
		t.Fatalf("mergeSpool with a failing log = %d, %v; want 0 and an error", n, err)
	}
	if m := spGlob(t, filepath.Join(spool, "*.merging")); len(m) != 1 {
		t.Errorf("the records must be kept in a .merging file, found %v", m)
	}
	if m := spGlob(t, filepath.Join(spool, "a1.log")); len(m) != 0 {
		t.Errorf("the original spool file should have been renamed, found %v", m)
	}

	n, err = mergeSpool(spool, main, fileLogIO())
	if err != nil || n != 2 {
		t.Fatalf("retry = %d, %v; want 2, nil", n, err)
	}
	if m := spGlob(t, filepath.Join(spool, "*")); len(m) != 0 {
		t.Errorf("the spool folder should be empty after the retry, found %v", m)
	}
}

func TestSP_MergeNoSpoolDir(t *testing.T) {
	dir := t.TempDir()
	n, err := mergeSpool(filepath.Join(dir, "nope"), filepath.Join(dir, "fedit.log"), fileLogIO())
	if err != nil || n != 0 {
		t.Errorf("a missing spool folder is not an error: got %d, %v", n, err)
	}
}
