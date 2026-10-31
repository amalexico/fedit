package main

import (
	"testing"
)

// Gap a: the end record of every plain write operation carries lines_changed,
// from the command line and from MCP alike. The count means lines inserted
// (insert, insertafter, insertbefore), lines deleted, lines in the replaced
// range, lines written (write, writeraw), or lines in the block times the
// number of copies (move, copy). A dry run, a failed call or a refused call
// carries no count.

type llnCase struct {
	name string
	op   string
	cli  []string
	tool string
	mcp  map[string]any
	want string
}

func llnCases() []llnCase {
	return []llnCase{
		{"insert", "insert", []string{"-line", "1", "-text", "x\\ny"}, "fedit_insert", map[string]any{"line": 1, "text": "x\\ny"}, "2"},
		{"insertafter", "insertafter", []string{"-match", "beta", "-text", "x\\ny"}, "fedit_insertafter", map[string]any{"match": "beta", "text": "x\\ny"}, "2"},
		{"insertbefore", "insertbefore", []string{"-match", "beta", "-text", "x\\ny"}, "fedit_insertbefore", map[string]any{"match": "beta", "text": "x\\ny"}, "2"},
		{"delete", "delete", []string{"-line", "1", "-end", "2"}, "fedit_delete", map[string]any{"line": 1, "end": 2}, "2"},
		{"replace", "replace", []string{"-line", "1", "-end", "2", "-text", "x\\ny\\nz"}, "fedit_replace", map[string]any{"line": 1, "end": 2, "text": "x\\ny\\nz"}, "2"},
		{"write", "write", []string{"-text", "p\\nq\\nr"}, "fedit_write", map[string]any{"text": "p\\nq\\nr"}, "3"},
		{"writeraw", "writeraw", []string{"-text", "p\nq"}, "fedit_writeraw", map[string]any{"text": "p\nq"}, "2"},
		{"move", "move", []string{"-line", "1", "-end", "1", "-after", "3"}, "fedit_move", map[string]any{"line": 1, "end": 1, "after": 3}, "1"},
		{"copy", "copy", []string{"-line", "1", "-end", "2", "-after", "3", "-times", "2"}, "fedit_copy", map[string]any{"line": 1, "end": 2, "after": 3, "times": 2}, "4"},
	}
}

func llnArgs(f string, extra map[string]any) map[string]any {
	m := map[string]any{"file": f}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func llnAbsent(t *testing.T, rec, key string) {
	t.Helper()
	if v, ok := spoolLogField(rec, key); ok {
		t.Errorf("the end record must not carry %s, got %q in %q", key, v, rec)
	}
}

func TestLLN_Bin(t *testing.T) {
	bin := clhBuild(t)
	for _, c := range llnCases() {
		c := c
		t.Run("cli_"+c.name, func(t *testing.T) {
			logp, f := lhbSetup(t)
			args := append([]string{"-file", f, "-op", c.op}, c.cli...)
			if _, se, code := orxRun(t, bin, args...); code != 0 {
				t.Fatalf("exit %d, stderr %q", code, se)
			}
			lhbField(t, lhbEnd(t, logp), "lines_changed", c.want)
		})
		t.Run("mcp_"+c.name, func(t *testing.T) {
			logp, f := lhbSetup(t)
			res := mcpLoggedExec(c.tool, llnArgs(f, c.mcp))
			if res.IsError {
				t.Fatalf("error result: %s", mcpText(res))
			}
			lhbField(t, lhbEnd(t, logp), "lines_changed", c.want)
		})
	}
	t.Run("cli_dry", func(t *testing.T) {
		logp, f := lhbSetup(t)
		if _, se, code := orxRun(t, bin, "-file", f, "-op", "insert", "-line", "1", "-text", "x", "-dry"); code != 0 {
			t.Fatalf("exit %d, stderr %q", code, se)
		}
		e := lhbEnd(t, logp)
		lhbField(t, e, "dry", "true")
		llnAbsent(t, e, "lines_changed")
	})
	t.Run("mcp_dry", func(t *testing.T) {
		for _, c := range llnCases() {
			logp, f := lhbSetup(t)
			a := llnArgs(f, c.mcp)
			a["dry_run"] = true
			if res := mcpLoggedExec(c.tool, a); res.IsError {
				t.Fatalf("%s: error result: %s", c.name, mcpText(res))
			}
			e := lhbEnd(t, logp)
			lhbField(t, e, "dry", "true")
			llnAbsent(t, e, "lines_changed")
		}
	})
	t.Run("mcp_dry_replaceall", func(t *testing.T) {
		logp, f := lhbSetup(t)
		a := llnArgs(f, map[string]any{"match": "alpha", "text": "beta", "dry_run": true})
		if res := mcpLoggedExec("fedit_replaceall", a); res.IsError {
			t.Fatalf("error result: %s", mcpText(res))
		}
		e := lhbEnd(t, logp)
		lhbField(t, e, "dry", "true")
		llnAbsent(t, e, "lines_changed")
	})
	t.Run("cli_failed", func(t *testing.T) {
		logp, f := lhbSetup(t)
		if _, _, code := orxRun(t, bin, "-file", f, "-op", "delete", "-line", "9", "-end", "9"); code != 1 {
			t.Fatalf("exit %d, want 1", code)
		}
		llnAbsent(t, lhbEnd(t, logp), "lines_changed")
	})
	t.Run("mcp_failed", func(t *testing.T) {
		logp, f := lhbSetup(t)
		if res := mcpLoggedExec("fedit_delete", llnArgs(f, map[string]any{"line": 9, "end": 9})); !res.IsError {
			t.Fatalf("want an error result: %s", mcpText(res))
		}
		llnAbsent(t, lhbEnd(t, logp), "lines_changed")
	})
}
