package main

import (
	"strings"
	"testing"
)

// A JavaScript regex literal containing a slash-star used to make -block see one
// block for the whole file (map was not affected). replace by -block must find the
// right block from the CLI (the file shows which block was hit), and show must work
// over MCP too.
func TestJSR_Binary(t *testing.T) {
	bin := clhBuild(t)
	const orig = "function a() {\n  var r = /a\\/*b/;\n  return r;\n}\nfunction b() {\n}\n"

	t.Run("replace_block_b", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replace", "-block", "b", "-lang", "js", "-text", "function b() {}")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != "function a() {\n  var r = /a\\/*b/;\n  return r;\n}\nfunction b() {}\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("replace_block_a", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replace", "-block", "a", "-lang", "js", "-text", "function a() {}")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != "function a() {}\nfunction b() {\n}\n" {
			t.Errorf("file = %q", got)
		}
	})
}

func TestJSR_MCPShow(t *testing.T) {
	p := wraFile(t, "function a() {\n  var r = /a\\/*b/;\n  return r;\n}\nfunction b() {\n}\n")
	r := mcpExecTool("fedit_show", map[string]any{"file": p, "block": "b", "lang": "js"})
	if r.IsError {
		t.Fatalf("fedit_show block b: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "function b") {
		t.Errorf("fedit_show lacks function b: %q", mcpText(r))
	}
}
