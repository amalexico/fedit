package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drbBinDryCleanFirst: -cleanfirst under -dry truncates nothing and creates
// nothing. The run goes on as if the file were empty, so the diff shows the
// old content removed and the new content added.
func drbBinDryCleanFirst(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	before, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	so, se, code := orxRun(t, bin, "-dry", "-cleanfirst", "-file", f, "-op", "insert", "-line", "0", "-text", "fresh")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	after, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("-dry -cleanfirst changed the file: %q -> %q", before, after)
	}
	if !strings.Contains(so, "+fresh\n") {
		t.Errorf("the diff lacks the added line: %q", so)
	}
	if !strings.Contains(so, "\n-") {
		t.Errorf("the diff lacks the removed old content: %q", so)
	}

	g := filepath.Join(filepath.Dir(f), "brand_new.txt")
	so, se, code = orxRun(t, bin, "-dry", "-cleanfirst", "-file", g, "-op", "insert", "-line", "0", "-text", "fresh")
	if code != 0 {
		t.Fatalf("new file: exit %d, stderr %q", code, se)
	}
	if _, err := os.Stat(g); err == nil {
		t.Error("-dry -cleanfirst created the file")
	}
	if !strings.Contains(so, "--- /dev/null") || !strings.Contains(so, "+fresh\n") {
		t.Errorf("new file: the diff should add the file from /dev/null: %q", so)
	}
}
