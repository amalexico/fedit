package main

import (
	"strings"
	"testing"
)

// The STATS line compares line counts only, so after a same-length edit it must
// say the line COUNT is unchanged, never that the content is.
func TestStatsLineCountWording(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name string
		args []string
	}{
		{"replace", []string{"-op", "replace", "-line", "2", "-text", "zz"}},
		{"replaceall", []string{"-op", "replaceall", "-match", "b", "-text", "B"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := wraFile(t, "a\nb\nc\n")
			args := append([]string{"-file", p}, c.args...)
			args = append(args, "-v")
			out, code := wraRun(t, bin, args...)
			if code != 0 {
				t.Fatalf("exit %d:\n%s", code, out)
			}
			if !strings.Contains(out, "lines:   0 (line count unchanged, 3 total)") {
				t.Errorf("STATS must say the line count is unchanged:\n%s", out)
			}
			if strings.Contains(out, "(unchanged,") {
				t.Errorf("old misleading wording still printed:\n%s", out)
			}
		})
	}
}
