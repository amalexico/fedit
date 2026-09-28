package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mmFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

// mmRead returns file content with CRLF normalized and trailing newlines trimmed.
func mmRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimRight(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n")
}

func mmSpec(paths ...string) string { return strings.Join(paths, ",") }

func mmWant(t *testing.T, p, want string) {
	t.Helper()
	if got := mmRead(t, p); got != want {
		t.Errorf("%s: got %q, want %q", filepath.Base(p), got, want)
	}
}

func TestMultiMutateReplaceAll(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "one foo\ntwo foo\n")
	b := mmFile(t, dir, "b.txt", "foo\n")
	n := mmFile(t, dir, "n.txt", "nothing here\n")
	code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b, n), op: "replaceall", match: "foo", replacement: "BAR", nth: 1})
	if code != 0 {
		t.Errorf("exit code %d, want 0", code)
	}
	mmWant(t, a, "one BAR\ntwo BAR")
	mmWant(t, b, "BAR")
	mmWant(t, n, "nothing here")
}

func TestMultiMutateAllSkippedExits1(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "alpha\n")
	b := mmFile(t, dir, "b.txt", "beta\n")
	code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b), op: "replaceall", match: "foo", replacement: "X", nth: 1})
	if code != 1 {
		t.Errorf("exit code %d, want 1 when no file succeeded", code)
	}
	mmWant(t, a, "alpha")
	mmWant(t, b, "beta")
}

func TestMultiMutateReadOnlySkip(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "one foo\ntwo foo\n")
	ro1 := mmFile(t, dir, "ro1.txt", "one foo\ntwo foo\n")
	ro2 := mmFile(t, dir, "ro2.txt", "one foo\ntwo foo\n")
	for _, p := range []string{ro1, ro2} {
		p := p
		if err := os.Chmod(p, 0444); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(p, 0644) })
	}
	if f, err := os.OpenFile(ro1, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("read-only file is still writable here (running as admin or root?)")
	}
	code := runMultiMutate(multiMutOpts{spec: mmSpec(a, ro1), op: "replaceall", match: "foo", replacement: "BAR", nth: 1})
	if code != 0 {
		t.Errorf("exit code %d, want 0 (one file succeeded)", code)
	}
	mmWant(t, a, "one BAR\ntwo BAR")
	mmWant(t, ro1, "one foo\ntwo foo")

	code = runMultiMutate(multiMutOpts{spec: mmSpec(ro1, ro2), op: "replaceall", match: "foo", replacement: "BAR", nth: 1})
	if code != 1 {
		t.Errorf("exit code %d, want 1 (every file skipped)", code)
	}
	mmWant(t, ro1, "one foo\ntwo foo")
	mmWant(t, ro2, "one foo\ntwo foo")
}

func TestMultiMutateInsertAfterAndBefore(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "one\ntwo\n")
	b := mmFile(t, dir, "b.txt", "one\ntwo\n")
	if code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b), op: "insertafter", match: "one", nth: 1, newLines: []string{"INS"}}); code != 0 {
		t.Errorf("insertafter exit code %d, want 0", code)
	}
	mmWant(t, a, "one\nINS\ntwo")
	mmWant(t, b, "one\nINS\ntwo")
	if code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b), op: "insertbefore", match: "two", nth: 1, newLines: []string{"PRE"}}); code != 0 {
		t.Errorf("insertbefore exit code %d, want 0", code)
	}
	mmWant(t, a, "one\nINS\nPRE\ntwo")
	mmWant(t, b, "one\nINS\nPRE\ntwo")
}

func TestMultiMutateDelete(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "one\ntwo\nthree\n")
	b := mmFile(t, dir, "b.txt", "two\nfour\n")
	n := mmFile(t, dir, "n.txt", "nothing\n")
	if code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b, n), op: "delete", match: "two", nth: 1}); code != 0 {
		t.Errorf("delete exit code %d, want 0", code)
	}
	mmWant(t, a, "one\nthree")
	mmWant(t, b, "four")
	mmWant(t, n, "nothing")
}

func TestMultiMutateDeleteRange(t *testing.T) {
	dir := t.TempDir()
	a := mmFile(t, dir, "a.txt", "a\nb\nc\nd\n")
	b := mmFile(t, dir, "b.txt", "a\nb\nc\nd\n")
	code := runMultiMutate(multiMutOpts{spec: mmSpec(a, b), op: "delete", match: "b", endmatch: "c", nth: 1})
	if code != 0 {
		t.Errorf("delete range exit code %d, want 0", code)
	}
	for _, p := range []string{a, b} {
		got := mmRead(t, p)
		if !strings.HasPrefix(got, "a\n") || !strings.HasSuffix(got, "d") || strings.Contains(got, "b") {
			t.Errorf("%s: got %q, want a first, d last and b gone", filepath.Base(p), got)
		}
	}
}

func TestMultiMutateRejectsBadRequests(t *testing.T) {
	cases := []struct {
		name string
		o    multiMutOpts
	}{
		{"unsupported op", multiMutOpts{spec: "x,y", op: "replace", match: "a", nth: 1}},
		{"missing match", multiMutOpts{spec: "x,y", op: "replaceall", nth: 1}},
		{"insert without content", multiMutOpts{spec: "x,y", op: "insertafter", match: "a", nth: 1}},
	}
	for _, c := range cases {
		if code := runMultiMutate(c.o); code != 1 {
			t.Errorf("%s: exit code %d, want 1", c.name, code)
		}
	}
}
