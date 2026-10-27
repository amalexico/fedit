package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drbBinDryMultiDestMove: a move into another file under -dry writes neither
// file, prints a diff for the destination and for the source, and its note
// says what would happen ("would remove"), not what happened.
func drbBinDryMultiDestMove(t *testing.T, bin string) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	want := map[string][]byte{a: []byte("one\ntwo\nthree\n"), b: []byte("x\ny\n")}
	for p, txt := range want {
		if err := os.WriteFile(p, txt, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	so, se, code := orxRun(t, bin, "-dry", "-op", "move", "-file", a, "-line", "1", "-end", "1", "-dest", b, "-aftermatch", "x")
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
	if !strings.Contains(so, "\n+one\n") || !strings.Contains(so, "\n-one\n") {
		t.Errorf("the diffs lack the moved line: %q", so)
	}
	if !strings.Contains(se, "would remove lines 1-1 from "+a) {
		t.Errorf("the note should say what would be removed: %q", se)
	}
	if strings.Contains(se, "removed lines") {
		t.Errorf("the note claims a removal under -dry: %q", se)
	}
	if !strings.Contains(se, "dry-run: applied=false") {
		t.Errorf("stderr lacks the applied=false note: %q", se)
	}
}
