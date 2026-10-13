package main

import (
	"os"
	"strings"
	"testing"
)

func mdaTemp(t *testing.T, ext, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "mda_*"+ext)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

const mdaSrc = "package main\n\nfunc Up(s string) string {\n\treturn strings.ToUpper(s)\n}\n"

func mdaFileHas(t *testing.T, path, want string) bool {
	t.Helper()
	for _, l := range mraRead(t, path) {
		if strings.Contains(l, want) {
			return true
		}
	}
	return false
}

func TestMCPExec_CopyDest_GoImportHint(t *testing.T) {
	src := mdaTemp(t, ".go", mdaSrc)
	defer os.Remove(src)
	dst := mdaTemp(t, ".go", "package main\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_copy", map[string]any{
		"file": src, "block": "Up", "lang": "go", "dest": dst, "after": 1,
	})
	if r.IsError {
		t.Fatalf("copy dest error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "hint:") || !strings.Contains(mcpText(r), "strings") {
		t.Errorf("want a strings import hint, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not copied into the destination")
	}
}

func TestMCPExec_CopyDest_NoHintWhenImported(t *testing.T) {
	src := mdaTemp(t, ".go", mdaSrc)
	defer os.Remove(src)
	dst := mdaTemp(t, ".go", "package main\n\nimport (\n\t\"strings\"\n)\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_copy", map[string]any{
		"file": src, "block": "Up", "lang": "go", "dest": dst, "after": 1,
	})
	if r.IsError {
		t.Fatalf("copy dest error: %s", mcpText(r))
	}
	if strings.Contains(mcpText(r), "hint:") {
		t.Errorf("unexpected hint: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not copied into the destination")
	}
}

func TestMCPExec_CopyDest_NoHintForNonGoDest(t *testing.T) {
	src := mdaTemp(t, ".go", mdaSrc)
	defer os.Remove(src)
	dst := mdaTemp(t, ".txt", "notes\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_copy", map[string]any{
		"file": src, "block": "Up", "lang": "go", "dest": dst, "after": 1,
	})
	if r.IsError {
		t.Fatalf("copy dest error: %s", mcpText(r))
	}
	if strings.Contains(mcpText(r), "hint:") {
		t.Errorf("unexpected hint for a non-Go destination: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not copied into the destination")
	}
}

func TestMCPExec_MoveDest_GoImportHint(t *testing.T) {
	src := mdaTemp(t, ".go", mdaSrc)
	defer os.Remove(src)
	dst := mdaTemp(t, ".go", "package main\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_move", map[string]any{
		"file": src, "block": "Up", "lang": "go", "dest": dst, "after": 1,
	})
	if r.IsError {
		t.Fatalf("move dest error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "hint:") || !strings.Contains(mcpText(r), "strings") {
		t.Errorf("want a strings import hint, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") || mdaFileHas(t, src, "func Up") {
		t.Errorf("move did not complete: dest has it=%v, source still has it=%v", mdaFileHas(t, dst, "func Up"), mdaFileHas(t, src, "func Up"))
	}
}

func TestMCPExec_ReplaceDest_GoImportHint(t *testing.T) {
	src := mdaTemp(t, ".go", mdaSrc)
	defer os.Remove(src)
	dst := mdaTemp(t, ".go", "package main\n\nfunc Old() {\n}\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_replace", map[string]any{
		"file": src, "block": "Up", "lang": "go", "dest": dst, "destblock": "Old", "text": "x",
	})
	if r.IsError {
		t.Fatalf("replace dest error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "hint:") || !strings.Contains(mcpText(r), "strings") {
		t.Errorf("want a strings import hint, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not replaced in the destination")
	}
}
