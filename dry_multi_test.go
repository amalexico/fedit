package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drbBinDryMultiReplaceAll: a multi -file replaceall under -dry changes no
// file, prints a unified diff for every file on stdout, skips the per-file
// verify (it would re-read a file that was not written) and exits 0.
func drbBinDryMultiReplaceAll(t *testing.T, bin string) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	want := map[string][]byte{a: []byte("alpha\nbeta\n"), b: []byte("gamma\nalpha\n")}
	for p, txt := range want {
		if err := os.WriteFile(p, txt, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	so, se, code := orxRun(t, bin, "-dry", "-file", a+","+b, "-op", "replaceall", "-match", "alpha", "-text", "omega")
	if code != 0 {
		t.Errorf("exit %d, want 0 (stderr %q)", code, se)
	}
	for p, txt := range want {
		got, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, txt) {
			t.Errorf("-dry changed %s: %q -> %q", p, txt, got)
		}
		if !strings.Contains(so, "+++ "+p+"\n") {
			t.Errorf("stdout has no diff for %s: %q", p, so)
		}
	}
	if n := strings.Count(so, "\n-alpha\n"); n != 2 {
		t.Errorf("want 2 removed lines in the diffs, got %d in %q", n, so)
	}
	if n := strings.Count(so, "\n+omega\n"); n != 2 {
		t.Errorf("want 2 added lines in the diffs, got %d in %q", n, so)
	}
	if !strings.Contains(se, "dry-run: applied=false") {
		t.Errorf("stderr lacks the applied=false note: %q", se)
	}
	if n := strings.Count(se, "VERIFY: SKIP (dry-run)"); n != 2 {
		t.Errorf("want one skip note per file (2), got %d in %q", n, se)
	}
	if strings.Contains(se, "VERIFY: FAIL") {
		t.Errorf("stderr reports a verify failure under -dry: %q", se)
	}
}
