package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const mcpSBSrc = "package main\n\nfunc Helper() int {\n\treturn 42\n}\n"
const mcpSBTarget = "package main\n\n// helpers\n\nfunc Other() {}\n"

func mcpSBFixture(t *testing.T) (target, source string) {
	t.Helper()
	dir := t.TempDir()
	target = filepath.Join(dir, "target.go")
	source = filepath.Join(dir, "source.go")
	if err := os.WriteFile(target, []byte(mcpSBTarget), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte(mcpSBSrc), 0o644); err != nil {
		t.Fatal(err)
	}
	return target, source
}

func mcpSBRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func TestMCPSourceBlock_InsertAfter(t *testing.T) {
	target, source := mcpSBFixture(t)
	res := mcpExecTool("fedit_insertafter", map[string]any{
		"file": target, "match": "// helpers",
		"sourcefile": source, "sourceblock": "Helper", "lang": "go",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", mcpMultiText(res))
	}
	got := mcpSBRead(t, target)
	if !strings.Contains(got, "// helpers\nfunc Helper() int {\n\treturn 42\n}\n") {
		t.Errorf("block not inserted right after the anchor:\n%s", got)
	}
	if mcpSBRead(t, source) != mcpSBSrc {
		t.Error("the source file must stay untouched")
	}
}

func TestMCPSourceBlock_InsertBefore(t *testing.T) {
	target, source := mcpSBFixture(t)
	res := mcpExecTool("fedit_insertbefore", map[string]any{
		"file": target, "match": "func Other",
		"sourcefile": source, "sourceblock": "Helper", "lang": "go",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", mcpMultiText(res))
	}
	got := mcpSBRead(t, target)
	if !strings.Contains(got, "return 42\n}\nfunc Other() {}") {
		t.Errorf("block not inserted right before the anchor:\n%s", got)
	}
}

func TestMCPSourceBlock_RejectedRequestsModifyNothing(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"text-with-sourcefile", map[string]any{"text": "x", "sourcefile": "SRC", "sourceblock": "Helper", "lang": "go"}, "cannot be combined with text"},
		{"missing-lang", map[string]any{"sourcefile": "SRC", "sourceblock": "Helper"}, "require lang"},
		{"missing-sourceblock", map[string]any{"sourcefile": "SRC", "lang": "go"}, "used together"},
		{"missing-sourcefile", map[string]any{"sourceblock": "Helper", "lang": "go"}, "used together"},
		{"unknown-block", map[string]any{"sourcefile": "SRC", "sourceblock": "NoSuchBlock", "lang": "go"}, "sourceblock"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			target, source := mcpSBFixture(t)
			if v, ok := c.args["sourcefile"]; ok && v == "SRC" {
				c.args["sourcefile"] = source
			}
			c.args["file"] = target
			c.args["match"] = "// helpers"
			res := mcpExecTool("fedit_insertafter", c.args)
			text := mcpMultiText(res)
			if !res.IsError || !strings.Contains(text, c.want) {
				t.Errorf("want error containing %q, got:\n%s", c.want, text)
			}
			if mcpSBRead(t, target) != mcpSBTarget {
				t.Error("a rejected request must not modify the target")
			}
		})
	}
}

func TestMCPSourceBlock_PlainInsertUnchanged(t *testing.T) {
	target, _ := mcpSBFixture(t)
	res := mcpExecTool("fedit_insertafter", map[string]any{
		"file": target, "match": "// helpers", "text": "// added",
	})
	if res.IsError {
		t.Fatalf("unexpected error: %s", mcpMultiText(res))
	}
	if !strings.Contains(mcpSBRead(t, target), "// helpers\n// added\n") {
		t.Errorf("plain text insert changed:\n%s", mcpSBRead(t, target))
	}
}
