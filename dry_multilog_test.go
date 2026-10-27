package main

import (
	"path/filepath"
	"testing"
)

// drbBinDryMultiLogged: a multi -file run under -dry is logged with dry=true,
// and its verify verdict is none: nothing was written, so nothing was verified.
func drbBinDryMultiLogged(t *testing.T, bin string) {
	logp, f := lhbSetup(t)
	g := orxWrite(t, filepath.Dir(f), "second.txt", lwxOrig)
	_, se, code := orxRun(t, bin, "-dry", "-file", f+","+g, "-op", "replaceall", "-match", "alpha", "-text", "omega")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	e := lhbEnd(t, logp)
	lhbField(t, e, "dry", "true")
	lhbField(t, e, "files_total", "2")
	lhbField(t, e, "files_ok", "2")
	lhbField(t, e, "verify", "none")
}
