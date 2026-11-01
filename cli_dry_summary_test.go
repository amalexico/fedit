package main

import (
	"strings"
	"testing"
)

// A single-file dry run changes nothing, so its summary line on the command
// line must read as a preview with the same wording the MCP server uses:
// Would insert ..., and "total after" instead of "total now".
func TestCDS_DrySummaryWording(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name   string
		args   []string
		want   string
		forbid string
	}{
		{"insert", []string{"-op", "insert", "-line", "1", "-text", "new"}, "Would insert 1 line(s) after line 1 (4 total after)", "Inserted "},
		{"insertafter", []string{"-op", "insertafter", "-match", "alpha", "-text", "new"}, "Would insert 1 line(s) after line 1 (4 total after)", "Inserted "},
		{"insertbefore", []string{"-op", "insertbefore", "-match", "beta", "-text", "new"}, "Would insert 1 line(s) before line 2 (4 total after)", "Inserted "},
		{"delete", []string{"-op", "delete", "-line", "2", "-end", "2"}, "Would delete lines 2-2 (2 total after)", "Deleted "},
		{"replace", []string{"-op", "replace", "-line", "2", "-end", "2", "-text", "new"}, "Would replace lines 2-2 with 1 line(s) (3 total after)", "Replaced lines"},
		{"replaceall", []string{"-op", "replaceall", "-match", "alpha", "-text", "zzz"}, "Would replace \x27alpha\x27 on 1 line(s) (3 total)", "Replaced "},
		{"regex", []string{"-op", "replaceall", "-match-regex", "alph[a]", "-text", "zzz"}, "Would regex-replace on 1 line(s) (3 total)", "Regex-replaced"},
		{"stream", []string{"-op", "replaceall", "-match", "alpha", "-text", "zzz", "-stream"}, "Would stream-replace on 1 line(s) of 3 total", "Stream-replaced"},
		{"write", []string{"-op", "write", "-text", "hello"}, "Would write 1 line(s) to", "Wrote "},
		{"writeraw", []string{"-op", "writeraw", "-text", "hello"}, "Would write 1 line(s) to", "Wrote "},
		{"move", []string{"-op", "move", "-line", "1", "-end", "1", "-after", "3"}, "Would move lines 1-1 after line 3 (3 total after)", "Moved "},
		{"copy", []string{"-op", "copy", "-line", "1", "-end", "1", "-after", "3"}, "Would copy lines 1-1 after line 3 (4 total after)", "Copied "},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := lhxFile(t, "alpha\nbeta\ngamma\n")
			args := append([]string{"-file", f}, c.args...)
			args = append(args, "-dry")
			stdout, stderr, code := orxRun(t, bin, args...)
			if code != 0 {
				t.Fatalf("exit %d, want 0: %s%s", code, stdout, stderr)
			}
			out := stdout + stderr
			if c.want != "" && !strings.Contains(out, c.want) {
				t.Errorf("want %q in the output:\n%s", c.want, out)
			}
			if strings.Contains(out, c.forbid) {
				t.Errorf("a dry run must not say %q:\n%s", c.forbid, out)
			}
		})
	}
}
