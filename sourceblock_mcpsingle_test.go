package main

import (
	"os"
	"strings"
	"testing"
)

func sbhMCPSingleRun(t *testing.T, ext, destContent string, useSource bool) (mcpCallResult, string) {
	t.Helper()
	dst := mdaTemp(t, ext, destContent)
	t.Cleanup(func() { os.Remove(dst) })
	args := map[string]any{"file": dst, "match": "package main", "lang": "go"}
	if useSource {
		src := mdaTemp(t, ".go", mdaSrc)
		t.Cleanup(func() { os.Remove(src) })
		args["sourcefile"] = src
		args["sourceblock"] = "Up"
	} else {
		args["text"] = `x := strings.ToUpper("a")`
	}
	return mcpExecTool("fedit_insertafter", args), dst
}

func TestSbhMCPSingle_HintOnMissingImport(t *testing.T) {
	r, dst := sbhMCPSingleRun(t, ".go", "package main\n", true)
	if r.IsError {
		t.Fatalf("insert error: %s", mcpText(r))
	}
	out := mcpText(r)
	if !strings.Contains(out, "hint:") || !strings.Contains(out, "strings") {
		t.Errorf("want a strings import hint, got: %s", out)
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not inserted")
	}
}

func TestSbhMCPSingle_NoHintWhenImported(t *testing.T) {
	r, dst := sbhMCPSingleRun(t, ".go", `package main

import "strings"
`, true)
	if r.IsError {
		t.Fatalf("insert error: %s", mcpText(r))
	}
	if strings.Contains(mcpText(r), "hint:") {
		t.Errorf("want no hint when strings is imported, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not inserted")
	}
}

func TestSbhMCPSingle_PlainTextNeverHints(t *testing.T) {
	r, dst := sbhMCPSingleRun(t, ".go", "package main\n", false)
	if r.IsError {
		t.Fatalf("insert error: %s", mcpText(r))
	}
	if strings.Contains(mcpText(r), "hint:") {
		t.Errorf("a plain text insert must not hint, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "strings.ToUpper") {
		t.Errorf("text was not inserted")
	}
}

func TestSbhMCPSingle_NonGoDestination_NoHint(t *testing.T) {
	r, dst := sbhMCPSingleRun(t, ".txt", "package main\n", true)
	if r.IsError {
		t.Fatalf("insert error: %s", mcpText(r))
	}
	if strings.Contains(mcpText(r), "hint:") {
		t.Errorf("a non-Go destination must not hint, got: %s", mcpText(r))
	}
	if !mdaFileHas(t, dst, "func Up") {
		t.Errorf("block was not inserted")
	}
}
