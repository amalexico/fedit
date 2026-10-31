package main

import "testing"

// A dry run changes nothing, so its end record notes no replacements and no
// lines_changed, the same rule as over MCP. Every replaceall path of the
// command line is covered.
func TestCDL_DryReplaceAllNotesNoCounts(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name string
		args []string
	}{
		{"literal", []string{"-op", "replaceall", "-match", "alpha", "-text", "beta", "-dry"}},
		{"regex", []string{"-op", "replaceall", "-match-regex", "alph[a]", "-text", "beta", "-dry"}},
		{"stream", []string{"-op", "replaceall", "-match", "alpha", "-text", "beta", "-stream", "-dry"}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			logp, f := lhbSetup(t)
			args := append([]string{"-file", f}, c.args...)
			if _, _, code := orxRun(t, bin, args...); code != 0 {
				t.Fatalf("exit %d, want 0", code)
			}
			e := lhbEnd(t, logp)
			lhbField(t, e, "dry", "true")
			llnAbsent(t, e, "replacements")
			llnAbsent(t, e, "lines_changed")
		})
	}
}
