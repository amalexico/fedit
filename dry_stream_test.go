package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// drbBinDryStream: -stream under -dry rewrites nothing (no temp file, no
// rename) and gives counts only, no diff on stdout. Exit codes are those of a
// real run.
func drbBinDryStream(t *testing.T, bin string) {
	cases := []struct {
		name     string
		args     []string
		wantCode int
		wantSkip bool
	}{
		{"literal", []string{"-match", "alpha", "-text", "gamma"}, 0, true},
		{"regex", []string{"-match-regex", "alp(ha)", "-text", "[$1]"}, 0, true},
		{"nomatch", []string{"-match", "zzz", "-text", "gamma"}, 1, false},
	}
	for _, c := range cases {
		_, f := lwxSetup(t)
		before, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		args := append([]string{"-dry", "-stream", "-file", f, "-op", "replaceall"}, c.args...)
		so, se, code := orxRun(t, bin, args...)
		if code != c.wantCode {
			t.Errorf("%s: exit %d, want %d (stderr %q)", c.name, code, c.wantCode, se)
		}
		after, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(before, after) {
			t.Errorf("%s: -dry -stream changed the file: %q -> %q", c.name, before, after)
		}
		if _, err := os.Stat(f + ".fedit_stream_tmp"); err == nil {
			t.Errorf("%s: a stream temp file was left behind", c.name)
		}
		if so != "" {
			t.Errorf("%s: -stream under -dry prints counts only, no diff on stdout: %q", c.name, so)
		}
		if !strings.Contains(se, "dry-run: applied=false") {
			t.Errorf("%s: stderr lacks the applied=false note: %q", c.name, se)
		}
		if c.wantSkip {
			if n := strings.Count(se, "VERIFY: SKIP (dry-run)"); n != 1 {
				t.Errorf("%s: want one skip line, got %d in %q", c.name, n, se)
			}
			if !strings.Contains(se, "2 line(s)") {
				t.Errorf("%s: stderr lacks the count of changed lines: %q", c.name, se)
			}
		}
	}
}
