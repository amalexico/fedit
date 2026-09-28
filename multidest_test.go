package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mdtWrite(t *testing.T, p, s string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(s), 0644); err != nil {
		t.Fatal(err)
	}
}

func mdtRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func mdtOpts(op, src, dests string) multiDestOpts {
	return multiDestOpts{
		op: op, srcFile: src, dests: dests,
		match: "beta", endmatch: "gamma", nth: 1, times: 1,
		after: -1, before: -1, aftermatch: "ANCHOR",
	}
}

// mdtSetup creates src.txt plus a.txt and b.txt (with ANCHOR) and c.txt (without).
func mdtSetup(t *testing.T) (dir string) {
	t.Helper()
	dir = t.TempDir()
	mdtWrite(t, filepath.Join(dir, "src.txt"), "alpha\nbeta\ngamma\ndelta\n")
	mdtWrite(t, filepath.Join(dir, "a.txt"), "A1\nANCHOR\nA2\n")
	mdtWrite(t, filepath.Join(dir, "b.txt"), "B1\nANCHOR\nB2\n")
	mdtWrite(t, filepath.Join(dir, "c.txt"), "C1\nno anchor here\nC2\n")
	return dir
}

const mdtSrcOrig = "alpha\nbeta\ngamma\ndelta\n"

func TestMultiDestCopyFanOut(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	code := runMultiDest(mdtOpts("copy", p("src.txt"), p("a.txt")+","+p("b.txt")+","+p("c.txt")+","+p("missing.txt")))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := mdtRead(t, p("src.txt")); got != mdtSrcOrig {
		t.Errorf("source changed by copy: %q", got)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nbeta\ngamma\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
	if got := mdtRead(t, p("b.txt")); got != "B1\nANCHOR\nbeta\ngamma\nB2\n" {
		t.Errorf("b.txt = %q", got)
	}
	if got := mdtRead(t, p("c.txt")); got != "C1\nno anchor here\nC2\n" {
		t.Errorf("c.txt should be untouched: %q", got)
	}
	if _, err := os.Stat(p("missing.txt")); err == nil {
		t.Error("missing.txt must not be created")
	}
}

func TestMultiDestMoveAllFailKeepsSource(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	code := runMultiDest(mdtOpts("move", p("src.txt"), p("c.txt")+","+p("missing.txt")))
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if got := mdtRead(t, p("src.txt")); got != mdtSrcOrig {
		t.Errorf("source must be intact when every destination fails: %q", got)
	}
}

func TestMultiDestMoveMixedTrimsSource(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	code := runMultiDest(mdtOpts("move", p("src.txt"), p("a.txt")+","+p("c.txt")))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := mdtRead(t, p("src.txt")); got != "alpha\ndelta\n" {
		t.Errorf("source = %q, want alpha/delta", got)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nbeta\ngamma\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
	if got := mdtRead(t, p("c.txt")); got != "C1\nno anchor here\nC2\n" {
		t.Errorf("c.txt should be untouched: %q", got)
	}
}

func TestMultiDestMoveReadOnlyDestSkipped(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	ro := p("ro.txt")
	mdtWrite(t, ro, "R1\nANCHOR\nR2\n")
	if err := os.Chmod(ro, 0444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(ro, 0644) })
	if f, err := os.OpenFile(ro, os.O_WRONLY, 0); err == nil {
		f.Close()
		t.Skip("read-only file is writable here (running privileged); permission behavior cannot be tested")
	}
	code := runMultiDest(mdtOpts("move", p("src.txt"), p("a.txt")+","+ro))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := mdtRead(t, ro); got != "R1\nANCHOR\nR2\n" {
		t.Errorf("read-only file changed: %q", got)
	}
	if got := mdtRead(t, p("src.txt")); got != "alpha\ndelta\n" {
		t.Errorf("source = %q", got)
	}
}

func TestMultiDestSameFileAsDestSkipped(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	code := runMultiDest(mdtOpts("move", p("src.txt"), p("src.txt")+","+p("a.txt")))
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if got := mdtRead(t, p("src.txt")); got != "alpha\ndelta\n" {
		t.Errorf("source = %q", got)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nbeta\ngamma\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
}

func TestMultiDestTimes(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	o := mdtOpts("copy", p("src.txt"), p("a.txt"))
	o.times = 2
	if code := runMultiDest(o); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nbeta\ngamma\nbeta\ngamma\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
}

func TestMultiDestBeforeMatch(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	o := mdtOpts("copy", p("src.txt"), p("a.txt"))
	o.aftermatch = ""
	o.beforematch = "ANCHOR"
	if code := runMultiDest(o); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nbeta\ngamma\nANCHOR\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
}

func TestMultiDestLineRangeSource(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	o := mdtOpts("copy", p("src.txt"), p("a.txt"))
	o.match, o.endmatch = "", ""
	o.line, o.end = 1, 2
	if code := runMultiDest(o); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nalpha\nbeta\nA2\n" {
		t.Errorf("a.txt = %q", got)
	}
}

func TestMultiDestDuplicateDestOnce(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }
	if code := runMultiDest(mdtOpts("copy", p("src.txt"), p("a.txt")+","+p("a.txt"))); code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nbeta\ngamma\nA2\n" {
		t.Errorf("duplicate -dest entry applied twice: %q", got)
	}
}

func TestMultiDestRejectsBadInput(t *testing.T) {
	d := mdtSetup(t)
	p := func(n string) string { return filepath.Join(d, n) }

	o := mdtOpts("delete", p("src.txt"), p("a.txt"))
	if code := runMultiDest(o); code != 1 {
		t.Errorf("bad op: exit code = %d, want 1", code)
	}
	o = mdtOpts("copy", p("src.txt"), "")
	if code := runMultiDest(o); code != 1 {
		t.Errorf("no dest: exit code = %d, want 1", code)
	}
	o = mdtOpts("copy", p("src.txt"), p("a.txt"))
	o.times = 0
	if code := runMultiDest(o); code != 1 {
		t.Errorf("times 0: exit code = %d, want 1", code)
	}
	o = mdtOpts("move", p("src.txt"), p("a.txt"))
	o.match = "nomatchanywhere"
	if code := runMultiDest(o); code != 1 {
		t.Errorf("no source match: exit code = %d, want 1", code)
	}
	if got := mdtRead(t, p("a.txt")); got != "A1\nANCHOR\nA2\n" {
		t.Errorf("a.txt changed by a rejected call: %q", got)
	}
	if got := mdtRead(t, p("src.txt")); got != mdtSrcOrig {
		t.Errorf("source changed by a rejected call: %q", got)
	}
}
