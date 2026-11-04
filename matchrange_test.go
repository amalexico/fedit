package main

import (
	"strings"
	"testing"
)

// Tests for content-anchored replace and delete without an end bound:
//   replace/delete -match X            -> the single matched line
//   replace/delete -match X -end N     -> from the match line through line N
//   replace/delete -match X -endmatch  -> unchanged (range)
// show, move and copy keep requiring an end bound. The multi -file path rejects -end.

// mrgSrc is the fixture: seven lines, "beta" twice.
const mrgSrc = "// head\nalpha\nbeta one\ngamma\nbeta two\ndelta\n// tail\n"

var mrgOrig = []string{"// head", "alpha", "beta one", "gamma", "beta two", "delta", "// tail"}

func mrgNorm(s string) string {
	return strings.TrimRight(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

// mrgWant fails the test unless the file at path holds exactly the given lines.
func mrgWant(t *testing.T, path string, want []string) {
	t.Helper()
	got := mrgNorm(wrrRead(t, path))
	if exp := strings.Join(want, "\n"); got != exp {
		t.Errorf("file content:\n got: %q\nwant: %q", got, exp)
	}
}

// mrgCase is one scenario. want == nil means the file must stay unchanged.
type mrgCase struct {
	name   string
	cli    []string       // CLI args after -file FILE
	mcp    map[string]any // MCP tool args besides file
	tool   string         // MCP tool name
	fail   bool           // expect an error (exit 1 / IsError)
	want   []string
	errHas string
	errNot string
}

func mrgCases() []mrgCase {
	del := func(n ...int) []string { // original lines without the 1-based line numbers in n
		skip := map[int]bool{}
		for _, i := range n {
			skip[i] = true
		}
		var out []string
		for i, l := range mrgOrig {
			if !skip[i+1] {
				out = append(out, l)
			}
		}
		return out
	}
	rep := func(from, to int, text string) []string { // replace lines from..to with one line
		var out []string
		for i, l := range mrgOrig {
			n := i + 1
			switch {
			case n == from:
				out = append(out, text)
			case n > from && n <= to:
			default:
				out = append(out, l)
			}
		}
		return out
	}
	return []mrgCase{
		{name: "delete single line", cli: []string{"-op", "delete", "-match", "gamma"},
			tool: "fedit_delete", mcp: map[string]any{"match": "gamma"}, want: del(4)},
		{name: "delete nth", cli: []string{"-op", "delete", "-match", "beta", "-nth", "2"},
			tool: "fedit_delete", mcp: map[string]any{"match": "beta", "nth": float64(2)}, want: del(5)},
		{name: "delete match to end N", cli: []string{"-op", "delete", "-match", "alpha", "-end", "4"},
			tool: "fedit_delete", mcp: map[string]any{"match": "alpha", "end": float64(4)}, want: del(2, 3, 4)},
		{name: "delete match to negative end", cli: []string{"-op", "delete", "-match", "alpha", "-end", "-2"},
			tool: "fedit_delete", mcp: map[string]any{"match": "alpha", "end": float64(-2)}, want: del(2, 3, 4, 5, 6)},
		{name: "delete endmatch unchanged", cli: []string{"-op", "delete", "-match", "alpha", "-endmatch", "gamma"},
			tool: "fedit_delete", mcp: map[string]any{"match": "alpha", "endmatch": "gamma"}, want: del(2, 3, 4)},
		{name: "delete end before match", cli: []string{"-op", "delete", "-match", "gamma", "-end", "2"},
			tool: "fedit_delete", mcp: map[string]any{"match": "gamma", "end": float64(2)}, fail: true, errHas: "before"},
		{name: "delete end past EOF", cli: []string{"-op", "delete", "-match", "alpha", "-end", "99"},
			tool: "fedit_delete", mcp: map[string]any{"match": "alpha", "end": float64(99)}, fail: true, errHas: "Invalid range"},
		{name: "delete match not found", cli: []string{"-op", "delete", "-match", "zzz"},
			tool: "fedit_delete", mcp: map[string]any{"match": "zzz"}, fail: true, errHas: "zzz", errNot: "source"},
		{name: "delete by line unchanged", cli: []string{"-op", "delete", "-line", "3"},
			tool: "fedit_delete", mcp: map[string]any{"line": float64(3)}, want: del(3)},
		{name: "delete line range unchanged", cli: []string{"-op", "delete", "-line", "2", "-end", "3"},
			tool: "fedit_delete", mcp: map[string]any{"line": float64(2), "end": float64(3)}, want: del(2, 3)},

		{name: "replace single line", cli: []string{"-op", "replace", "-match", "gamma", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "gamma", "text": "NEW"}, want: rep(4, 4, "NEW")},
		{name: "replace nth", cli: []string{"-op", "replace", "-match", "beta", "-nth", "2", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "beta", "nth": float64(2), "text": "NEW"}, want: rep(5, 5, "NEW")},
		{name: "replace match to end N", cli: []string{"-op", "replace", "-match", "alpha", "-end", "3", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "alpha", "end": float64(3), "text": "NEW"}, want: rep(2, 3, "NEW")},
		{name: "replace endmatch unchanged", cli: []string{"-op", "replace", "-match", "alpha", "-endmatch", "gamma", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "alpha", "endmatch": "gamma", "text": "NEW"}, want: rep(2, 4, "NEW")},
		{name: "replace end before match", cli: []string{"-op", "replace", "-match", "gamma", "-end", "2", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "gamma", "end": float64(2), "text": "NEW"}, fail: true, errHas: "before"},
		{name: "replace match not found", cli: []string{"-op", "replace", "-match", "zzz", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"match": "zzz", "text": "NEW"}, fail: true, errHas: "zzz", errNot: "source"},
		{name: "replace line range unchanged", cli: []string{"-op", "replace", "-line", "2", "-end", "3", "-text", "NEW"},
			tool: "fedit_replace", mcp: map[string]any{"line": float64(2), "end": float64(3), "text": "NEW"}, want: rep(2, 3, "NEW")},

		{name: "show match still needs end bound", cli: []string{"-op", "show", "-match", "gamma"},
			tool: "fedit_show", mcp: map[string]any{"match": "gamma"}, fail: true, errHas: "end bound"},
	}
}

func TestMrgCLI(t *testing.T) {
	bin := clhBuild(t)
	for _, c := range mrgCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := wraFile(t, mrgSrc)
			stderr, code := wraRun(t, bin, append([]string{"-file", f}, c.cli...)...)
			want := c.want
			if c.fail {
				want = mrgOrig
				if code != 1 {
					t.Errorf("exit code %d, want 1; stderr: %s", code, stderr)
				}
			} else if code != 0 {
				t.Errorf("exit code %d, want 0; stderr: %s", code, stderr)
			}
			if c.errHas != "" && !strings.Contains(stderr, c.errHas) {
				t.Errorf("stderr should contain %q, got: %s", c.errHas, stderr)
			}
			if c.errNot != "" && strings.Contains(stderr, c.errNot) {
				t.Errorf("stderr should not contain %q, got: %s", c.errNot, stderr)
			}
			mrgWant(t, f, want)
		})
	}
	for _, op := range []string{"delete", "replace"} {
		op := op
		t.Run("dry run changes nothing "+op, func(t *testing.T) {
			f := wraFile(t, mrgSrc)
			stderr, code := wraRun(t, bin, "-dry", "-file", f, "-op", op, "-match", "gamma", "-text", "NEW")
			if code != 0 {
				t.Errorf("exit code %d, want 0; stderr: %s", code, stderr)
			}
			mrgWant(t, f, mrgOrig)
		})
		t.Run("multi file rejects end "+op, func(t *testing.T) {
			a := wraFile(t, mrgSrc)
			b := wraFile(t, mrgSrc)
			stderr, code := wraRun(t, bin, "-file", a+","+b, "-op", op, "-match", "gamma", "-end", "4", "-text", "NEW")
			if code != 1 {
				t.Errorf("exit code %d, want 1; stderr: %s", code, stderr)
			}
			if !strings.Contains(stderr, "-end") {
				t.Errorf("stderr should name -end, got: %s", stderr)
			}
			mrgWant(t, a, mrgOrig)
			mrgWant(t, b, mrgOrig)
		})
	}
}

func TestMrgMCP(t *testing.T) {
	for _, c := range mrgCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f := wraFile(t, mrgSrc)
			args := map[string]any{"file": f}
			for k, v := range c.mcp {
				args[k] = v
			}
			r := mcpExecTool(c.tool, args)
			text := mcpText(r)
			want := c.want
			if c.fail {
				want = mrgOrig
				if !r.IsError {
					t.Errorf("expected an error, got success: %s", text)
				}
			} else if r.IsError {
				t.Errorf("unexpected error: %s", text)
			}
			if c.errHas != "" && !strings.Contains(text, c.errHas) {
				t.Errorf("message should contain %q, got: %s", c.errHas, text)
			}
			if c.errNot != "" && strings.Contains(text, c.errNot) {
				t.Errorf("message should not contain %q, got: %s", c.errNot, text)
			}
			mrgWant(t, f, want)
		})
	}
	for _, tool := range []string{"fedit_delete", "fedit_replace"} {
		tool := tool
		t.Run("multi file rejects end "+tool, func(t *testing.T) {
			a := wraFile(t, mrgSrc)
			b := wraFile(t, mrgSrc)
			r := mcpExecTool(tool, map[string]any{"file": a + "," + b, "match": "gamma", "end": float64(4), "text": "NEW"})
			if !r.IsError {
				t.Errorf("expected an error, got success: %s", mcpText(r))
			}
			if !strings.Contains(mcpText(r), "end") {
				t.Errorf("message should name end, got: %s", mcpText(r))
			}
			mrgWant(t, a, mrgOrig)
			mrgWant(t, b, mrgOrig)
		})
	}
}
