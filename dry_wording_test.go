package main

import (
	"strings"
	"testing"
)

// Under a dry run nothing was done, so the per-file lines of a multi-file or
// -dest report must be worded as a preview (would ...), not in the past tense.
// The wording is shared by the command line and the MCP server.
func TestPDW_DryReportWording(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name   string
		args   []string
		want   string
		forbid string
	}{
		{"replaceall", []string{"-op", "replaceall", "-match", "alpha", "-text", "beta"}, "would replace on 1 line(s)", "replaced on"},
		{"regex", []string{"-op", "replaceall", "-match-regex", "alph[a]", "-text", "beta"}, "would replace on 1 line(s) (regex)", "replaced on"},
		{"insertafter", []string{"-op", "insertafter", "-match", "alpha", "-text", "new"}, "would insert 1 line(s) at match on line 1", "inserted "},
		{"insertbefore", []string{"-op", "insertbefore", "-match", "alpha", "-text", "new"}, "would insert 1 line(s) at match on line 1", "inserted "},
		{"delete", []string{"-op", "delete", "-match", "alpha"}, "would delete lines 1-1", "deleted lines"},
		{"replace", []string{"-op", "replace", "-match", "alpha", "-text", "new"}, "would replace lines 1-1", "replaced lines"},
		{"write", []string{"-op", "write", "-text", "hello"}, "would write 1 line(s)", "wrote "},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			a := lhxFile(t, "alpha one\nx\n")
			b := lhxFile(t, "alpha two\ny\n")
			args := append([]string{"-file", a + "," + b}, c.args...)
			args = append(args, "-dry")
			stdout, stderr, code := orxRun(t, bin, args...)
			if code != 0 {
				t.Fatalf("exit %d, want 0: %s%s", code, stdout, stderr)
			}
			out := stdout + stderr
			if !strings.Contains(out, c.want) {
				t.Errorf("want %q in the report:\n%s", c.want, out)
			}
			if strings.Contains(out, c.forbid) {
				t.Errorf("a dry report must not say %q:\n%s", c.forbid, out)
			}
		})
	}
	t.Run("dest_replace", func(t *testing.T) {
		src := lhxFile(t, "package p\n\nfunc New() int {\n\treturn 2\n}\n")
		dst := lhxFile(t, "package p\n\nfunc Old() int {\n\treturn 1\n}\n")
		stdout, stderr, code := orxRun(t, bin, "-file", src, "-op", "replace", "-block", "New", "-lang", "go", "-dest", dst, "-destblock", "Old", "-dry")
		if code != 0 {
			t.Fatalf("exit %d, want 0: %s%s", code, stdout, stderr)
		}
		out := stdout + stderr
		if !strings.Contains(out, "would be replaced with 3 line(s)") {
			t.Errorf("want the preview wording in the report:\n%s", out)
		}
		if strings.Contains(out, "\" replaced with") {
			t.Errorf("a dry report must not use the past tense:\n%s", out)
		}
	})
	t.Run("dest_copy_report", func(t *testing.T) {
		src := lhxFile(t, "package p\n\nfunc New() int {\n\treturn 2\n}\n")
		dst := lhxFile(t, "package p\n\nfunc Old() int {\n\treturn 1\n}\n")
		stdout, stderr, code := orxRun(t, bin, "-file", src, "-op", "copy", "-block", "New", "-lang", "go", "-dest", dst, "-afterblock", "Old", "-dry")
		t.Logf("exit %d\n%s%s", code, stdout, stderr)
	})
	t.Run("mcp_replaceall", func(t *testing.T) {
		a := lhxFile(t, "alpha one\nx\n")
		b := lhxFile(t, "alpha two\ny\n")
		res := mcpLoggedExec("fedit_replaceall", map[string]any{"file": a + "," + b, "match": "alpha", "text": "beta", "dry_run": true})
		if res.IsError {
			t.Fatalf("error result: %s", mcpText(res))
		}
		out := mcpText(res)
		if !strings.Contains(out, "would replace on 1 line(s)") {
			t.Errorf("want the preview wording in the result:\n%s", out)
		}
		if strings.Contains(out, "replaced on") {
			t.Errorf("a dry result must not use the past tense:\n%s", out)
		}
	})
}
