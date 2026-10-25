package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func drxLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "l" + strconv.Itoa(i+1)
	}
	return out
}

func drxWith(base []string, repl map[int]string) []string {
	out := append([]string(nil), base...)
	for n, v := range repl {
		out[n-1] = v
	}
	return out
}

// drxSetup points dryOut at a buffer and restores dryRun and dryOut afterwards.
func drxSetup(t *testing.T, on bool) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	prevRun, prevOut := dryRun, dryOut
	dryRun, dryOut = on, buf
	t.Cleanup(func() { dryRun, dryOut = prevRun, prevOut })
	return buf
}

func drxFile(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f.txt")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func drxHunks(d string) int { return strings.Count(d, "\n@@ ") }

func TestDRY_DiffEqual(t *testing.T) {
	if d := dryDiff("f", "f", []string{"a", "b"}, []string{"a", "b"}); d != "" {
		t.Fatalf("equal input must give no diff, got %q", d)
	}
	if d := dryDiff("f", "f", nil, []string{}); d != "" {
		t.Fatalf("nil and empty must give no diff, got %q", d)
	}
}

func TestDRY_DiffOneChange(t *testing.T) {
	got := dryDiff("f.txt", "f.txt", []string{"a", "b", "c"}, []string{"a", "B", "c"})
	want := "--- f.txt\n+++ f.txt\n@@ -1,3 +1,3 @@\n a\n-b\n+B\n c\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDRY_DiffAppend(t *testing.T) {
	got := dryDiff("f", "f", []string{"a", "b"}, []string{"a", "b", "c"})
	want := "--- f\n+++ f\n@@ -1,2 +1,3 @@\n a\n b\n+c\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDRY_DiffDeleteAll(t *testing.T) {
	got := dryDiff("f", "f", []string{"a"}, nil)
	want := "--- f\n+++ f\n@@ -1 +0,0 @@\n-a\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDRY_DiffNewFileHeader(t *testing.T) {
	got := dryDiff("/dev/null", "f", nil, []string{"x", "y"})
	want := "--- /dev/null\n+++ f\n@@ -0,0 +1,2 @@\n+x\n+y\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDRY_DiffContext(t *testing.T) {
	a := drxLines(10)
	got := dryDiff("f", "f", a, drxWith(a, map[int]string{5: "X"}))
	want := "--- f\n+++ f\n@@ -2,7 +2,7 @@\n l2\n l3\n l4\n-l5\n+X\n l6\n l7\n l8\n"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestDRY_DiffHunkSplit(t *testing.T) {
	a := drxLines(20)
	d := dryDiff("f", "f", a, drxWith(a, map[int]string{2: "X", 19: "Y"}))
	if drxHunks(d) != 2 {
		t.Fatalf("want 2 hunks, got %d:\n%s", drxHunks(d), d)
	}
	for _, h := range []string{"@@ -1,5 +1,5 @@", "@@ -16,5 +16,5 @@"} {
		if !strings.Contains(d, h) {
			t.Fatalf("missing %s in:\n%s", h, d)
		}
	}
}

func TestDRY_DiffHunkMergeBoundary(t *testing.T) {
	a := drxLines(20)
	d := dryDiff("f", "f", a, drxWith(a, map[int]string{5: "X", 12: "Y"}))
	if drxHunks(d) != 1 || !strings.Contains(d, "@@ -2,14 +2,14 @@") {
		t.Fatalf("a gap of 6 unchanged lines must merge into one hunk:\n%s", d)
	}
	d = dryDiff("f", "f", a, drxWith(a, map[int]string{5: "X", 13: "Y"}))
	if drxHunks(d) != 2 {
		t.Fatalf("a gap of 7 unchanged lines must give two hunks:\n%s", d)
	}
}

func TestDRY_DiffLargeNoCommonLines(t *testing.T) {
	a := make([]string, 3000)
	b := make([]string, 3000)
	for i := range a {
		a[i] = "a" + strconv.Itoa(i)
		b[i] = "b" + strconv.Itoa(i)
	}
	d := dryDiff("f", "f", a, b)
	minus, plus := 0, 0
	for _, l := range strings.Split(d, "\n")[3:] {
		switch {
		case strings.HasPrefix(l, "-"):
			minus++
		case strings.HasPrefix(l, "+"):
			plus++
		}
	}
	if minus != 3000 || plus != 3000 {
		t.Fatalf("want 3000 removed and 3000 added lines, got %d and %d", minus, plus)
	}
}

func TestDRY_WriteLinesDryKeepsFile(t *testing.T) {
	buf := drxSetup(t, true)
	p := drxFile(t, "a\nb\n")
	if err := writeLines(p, []string{"a", "c"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "a\nb\n" {
		t.Fatalf("dry run changed the file: %q", b)
	}
	out := buf.String()
	if !strings.Contains(out, "--- "+p+"\n+++ "+p+"\n") || !strings.Contains(out, "-b\n+c\n") {
		t.Fatalf("diff missing or wrong:\n%s", out)
	}
}

func TestDRY_WriteLinesDryNewFile(t *testing.T) {
	buf := drxSetup(t, true)
	p := filepath.Join(t.TempDir(), "new.txt")
	if err := writeLines(p, []string{"x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err == nil {
		t.Fatal("dry run created the file")
	}
	out := buf.String()
	if !strings.Contains(out, "--- /dev/null\n+++ "+p+"\n") || !strings.Contains(out, "+x\n") {
		t.Fatalf("new-file diff missing or wrong:\n%s", out)
	}
}

func TestDRY_WriteLinesDryNoChange(t *testing.T) {
	buf := drxSetup(t, true)
	p := drxFile(t, "a\nb\n")
	if err := writeLines(p, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("no change must print no diff, got %q", buf.String())
	}
}

func TestDRY_WriteLinesRealStillWrites(t *testing.T) {
	buf := drxSetup(t, false)
	p := drxFile(t, "a\nb\n")
	if err := writeLines(p, []string{"a", "c"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(p); string(b) != "a\nc\n" {
		t.Fatalf("normal write broken: %q", b)
	}
	if buf.Len() != 0 {
		t.Fatalf("normal write must print no diff, got %q", buf.String())
	}
}
