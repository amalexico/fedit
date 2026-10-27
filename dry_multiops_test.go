package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drbDryPair writes a.txt and b.txt in a fresh temp dir and returns their
// paths with the content each must still have after a -dry run.
func drbDryPair(t *testing.T) (string, string, map[string]string) {
	dir := t.TempDir()
	a := orxWrite(t, dir, "a.txt", "alpha\nbeta\n")
	b := orxWrite(t, dir, "b.txt", "gamma\nalpha\n")
	return a, b, map[string]string{a: "alpha\nbeta\n", b: "gamma\nalpha\n"}
}

// drbDryUnchanged fails the test for every file that no longer has its content.
func drbDryUnchanged(t *testing.T, want map[string]string) {
	for p, txt := range want {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != txt {
			t.Errorf("-dry changed %s: %q -> %q", p, txt, got)
		}
	}
}

// drbBinDryMultiOps: guard tests for the writers that all go through
// writeLines. Under -dry every one of them must change no file, print a diff
// for each file it would change, exit 0 and say applied=false. They cover
// multi -file insertafter, insertbefore, delete, replace and write (also onto
// a file that does not exist), and -dest copy and -dest replace.
func drbBinDryMultiOps(t *testing.T, bin string) {
	cases := []struct {
		name  string
		args  []string
		gone  string
		added string
	}{
		{"insertafter", []string{"-op", "insertafter", "-match", "alpha", "-text", "NEWLINE"}, "", "NEWLINE"},
		{"insertbefore", []string{"-op", "insertbefore", "-match", "alpha", "-text", "NEWLINE"}, "", "NEWLINE"},
		{"delete", []string{"-op", "delete", "-match", "alpha"}, "alpha", ""},
		{"replace", []string{"-op", "replace", "-match", "alpha", "-text", "omega"}, "alpha", "omega"},
		{"write", []string{"-op", "write", "-text", "zzz"}, "", "zzz"},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			a, b, want := drbDryPair(t)
			args := append([]string{"-dry", "-file", a + "," + b}, c.args...)
			so, se, code := orxRun(t, bin, args...)
			if code != 0 {
				t.Errorf("exit %d, want 0 (stderr %q)", code, se)
			}
			drbDryUnchanged(t, want)
			for p := range want {
				if !strings.Contains(so, "+++ "+p+"\n") {
					t.Errorf("stdout has no diff for %s: %q", p, so)
				}
			}
			if c.gone != "" {
				if n := strings.Count(so, "\n-"+c.gone+"\n"); n != 2 {
					t.Errorf("want 2 removed %q lines, got %d in %q", c.gone, n, so)
				}
			}
			if c.added != "" {
				if n := strings.Count(so, "\n+"+c.added+"\n"); n != 2 {
					t.Errorf("want 2 added %q lines, got %d in %q", c.added, n, so)
				}
			}
			if !strings.Contains(se, "2/2 file(s) would succeed") {
				t.Errorf("stderr lacks the batch total: %q", se)
			}
			if !strings.Contains(se, "dry-run: applied=false") {
				t.Errorf("stderr lacks the applied=false note: %q", se)
			}
		})
	}
	t.Run("writemissing", func(t *testing.T) {
		a, _, want := drbDryPair(t)
		w := map[string]string{a: want[a]}
		c := filepath.Join(filepath.Dir(a), "c.txt")
		orxRun(t, bin, "-dry", "-file", a+","+c, "-op", "write", "-text", "zzz")
		drbDryUnchanged(t, w)
		if _, err := os.Stat(c); err == nil {
			t.Errorf("-dry created %s", c)
		}
	})
	t.Run("destcopy", func(t *testing.T) {
		dir := t.TempDir()
		a := orxWrite(t, dir, "a.txt", "one\ntwo\n")
		b := orxWrite(t, dir, "b.txt", "x\ny\n")
		want := map[string]string{a: "one\ntwo\n", b: "x\ny\n"}
		so, se, code := orxRun(t, bin, "-dry", "-op", "copy", "-file", a, "-line", "1", "-end", "1", "-dest", b, "-aftermatch", "x")
		if code != 0 {
			t.Errorf("exit %d, want 0 (stderr %q)", code, se)
		}
		drbDryUnchanged(t, want)
		if !strings.Contains(so, "+++ "+b+"\n") || !strings.Contains(so, "\n+one\n") {
			t.Errorf("stdout lacks the destination diff: %q", so)
		}
		if strings.Contains(so, "+++ "+a+"\n") {
			t.Errorf("a copy printed a diff for the source: %q", so)
		}
		if !strings.Contains(se, "dry-run: applied=false") {
			t.Errorf("stderr lacks the applied=false note: %q", se)
		}
	})
	t.Run("destreplace", func(t *testing.T) {
		dir := t.TempDir()
		srcTxt := "package p\n\nfunc Helper() int {\n\treturn 2\n}\n"
		dstTxt := "package p\n\nfunc Helper() int {\n\treturn 1\n}\n"
		a := orxWrite(t, dir, "a.go", srcTxt)
		b := orxWrite(t, dir, "b.go", dstTxt)
		want := map[string]string{a: srcTxt, b: dstTxt}
		so, se, code := orxRun(t, bin, "-dry", "-op", "replace", "-file", a, "-block", "Helper", "-lang", "go", "-dest", b, "-destblock", "Helper")
		if code != 0 {
			t.Errorf("exit %d, want 0 (stderr %q)", code, se)
		}
		drbDryUnchanged(t, want)
		if !strings.Contains(so, "+++ "+b+"\n") || !strings.Contains(so, "\n-\treturn 1\n") || !strings.Contains(so, "\n+\treturn 2\n") {
			t.Errorf("stdout lacks the destination diff: %q", so)
		}
		if strings.Contains(so, "+++ "+a+"\n") {
			t.Errorf("a replace printed a diff for the source: %q", so)
		}
		if !strings.Contains(se, "dry-run: applied=false") {
			t.Errorf("stderr lacks the applied=false note: %q", se)
		}
	})
}
