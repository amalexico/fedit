package main

import (
	"bytes"
	"os"
	"testing"
)

// drbBinDryLogged: a dry run is logged like any operation, with dry=true in
// the end record; the file is unchanged, so both hashes are the hash of the
// original bytes.
func drbBinDryLogged(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	before, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	_, se, code := orxRun(t, bin, "-dry", "-file", f, "-op", "insertafter", "-match", "alpha", "-text", "zzzline")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	after, err := os.ReadFile(f)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("-dry changed the file")
	}
	e := lhbEnd(t, logp)
	lhbField(t, e, "dry", "true")
	lhbField(t, e, "sha256_before", lhxSum(before))
	lhbField(t, e, "sha256_after", lhxSum(before))
}
