package main

import (
	"strings"
	"testing"
)

// -afterblock and -beforeblock are destination anchors for move and copy only.
// With insertafter/insertbefore they used to be ignored, so the user saw a
// misleading "-match is required" error. They are now rejected with a pointer
// to -block.

func TestCLIBlockAnchorGuard_InsertOps(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct{ op, flag string }{
		{"insertafter", "-afterblock"},
		{"insertbefore", "-beforeblock"},
		{"insertafter", "-beforeblock"},
		{"insertbefore", "-afterblock"},
	}
	const src = "package main\n\nfunc A() {\n}\n"
	for _, c := range cases {
		dir := t.TempDir()
		p := clhWrite(t, dir, "a.go", src)
		_, se, code := clhRun(t, bin, dir, "-file", "a.go", "-op", c.op, c.flag, "A", "-lang", "go", "-text", "x")
		if code != 1 {
			t.Errorf("%s %s: want exit 1, got %d (stderr %q)", c.op, c.flag, code, se)
		}
		if !strings.Contains(se, "use -block") || strings.Contains(se, "-match is required") {
			t.Errorf("%s %s: want the destination-anchor error pointing to -block, got %q", c.op, c.flag, se)
		}
		if got := clhRead(t, p); got != src {
			t.Errorf("%s %s: the file must stay unchanged, got %q", c.op, c.flag, got)
		}
	}
}

func TestCLIBlockAnchorGuard_CopyStillWorks(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	p := clhWrite(t, dir, "a.go", "package main\n\nfunc A() {\n}\n\nfunc B() {\n}\n")
	_, se, code := clhRun(t, bin, dir, "-file", "a.go", "-op", "copy", "-block", "A", "-lang", "go", "-afterblock", "B")
	if code != 0 {
		t.Fatalf("copy -afterblock: exit %d, stderr %q", code, se)
	}
	if n := strings.Count(clhRead(t, p), "func A()"); n != 2 {
		t.Errorf("copy -afterblock: want 2 copies of func A, got %d", n)
	}
}
