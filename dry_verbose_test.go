package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// drbBinDryVerbose: -v under -dry must not show a verify window or stats of
// the unchanged file. Every operation that would verify prints one
// "VERIFY: SKIP (dry-run)" line instead, and the file stays as it was.
func drbBinDryVerbose(t *testing.T, bin string) {
	cases := []struct {
		name string
		args []string
	}{
		{"insert", []string{"-op", "insert", "-line", "0", "-text", "fresh"}},
		{"insertafter", []string{"-op", "insertafter", "-match", "beta", "-text", "fresh"}},
		{"delete", []string{"-op", "delete", "-line", "2"}},
		{"replaceall", []string{"-op", "replaceall", "-match", "alpha", "-text", "gamma"}},
		{"copy", []string{"-op", "copy", "-line", "1", "-after", "3"}},
	}
	for _, c := range cases {
		_, f := lwxSetup(t)
		before, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"-dry", "-v", "-file", f}, c.args...)
		_, se, code := orxRun(t, bin, args...)
		if code != 0 {
			t.Errorf("%s: exit %d, stderr %q", c.name, code, se)
			continue
		}
		after, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("%s: -dry changed the file", c.name)
		}
		if n := strings.Count(se, "VERIFY: SKIP (dry-run)"); n != 1 {
			t.Errorf("%s: want one skip line, got %d in %q", c.name, n, se)
		}
		bads := []string{"=== VERIFY ==="}
		if c.name != "copy" { // copy prints its own stats, and they are right under -dry
			bads = append(bads, "=== STATS ===")
		}
		for _, bad := range bads {
			if strings.Contains(se, bad) {
				t.Errorf("%s: stderr must not contain %q under -dry: %q", c.name, bad, se)
			}
		}
	}
}
