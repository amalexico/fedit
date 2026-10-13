package main

import (
	"strings"
	"testing"
)

func sbhMCPFixture(t *testing.T) (src, a, b, c string) {
	t.Helper()
	dir := t.TempDir()
	src = mmFile(t, dir, "src.go", mdaSrc)
	a = mmFile(t, dir, "a.go", `package main
`)
	b = mmFile(t, dir, "b.go", `package main

import "strings"
`)
	c = mmFile(t, dir, "c.txt", `package main
`)
	return
}

func TestSbhMCPMulti_InsertAfterWithHint(t *testing.T) {
	src, a, b, c := sbhMCPFixture(t)
	r := mcpExecTool("fedit_insertafter", map[string]any{
		"file": mmSpec(a, b, c), "match": "package main",
		"sourcefile": src, "sourceblock": "Up", "lang": "go",
	})
	if r.IsError {
		t.Fatalf("multi insert error: %s", mcpText(r))
	}
	out := mcpText(r)
	if !strings.Contains(out, "3/3") {
		t.Errorf("want 3/3 files, got: %s", out)
	}
	if strings.Count(out, "hint:") != 1 || !strings.Contains(out, "strings") {
		t.Errorf("want exactly one strings hint, got: %s", out)
	}
	for _, p := range []string{a, b, c} {
		if !mdaFileHas(t, p, "func Up") {
			t.Errorf("%s: block was not inserted", p)
		}
	}
}

func TestSbhMCPMulti_RejectsTextWithSource(t *testing.T) {
	src, a, b, _ := sbhMCPFixture(t)
	r := mcpExecTool("fedit_insertbefore", map[string]any{
		"file": mmSpec(a, b), "match": "package main", "text": "extra",
		"sourcefile": src, "sourceblock": "Up", "lang": "go",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "cannot be combined with text") {
		t.Errorf("want a text conflict error, got error=%v: %s", r.IsError, mcpText(r))
	}
	if n := len(mraRead(t, a)); n != 1 {
		t.Errorf("a.go must be unmodified, has %d lines", n)
	}
}

func TestSbhMCPMulti_RequiresLang(t *testing.T) {
	src, a, b, _ := sbhMCPFixture(t)
	r := mcpExecTool("fedit_insertafter", map[string]any{
		"file": mmSpec(a, b), "match": "package main",
		"sourcefile": src, "sourceblock": "Up",
	})
	if !r.IsError || !strings.Contains(mcpText(r), "require lang") {
		t.Errorf("want a lang error, got error=%v: %s", r.IsError, mcpText(r))
	}
	if n := len(mraRead(t, a)); n != 1 {
		t.Errorf("a.go must be unmodified, has %d lines", n)
	}
}
