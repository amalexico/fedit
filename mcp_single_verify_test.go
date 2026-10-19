package main

import (
	"strings"
	"testing"
)

// The single-file MCP fedit_replaceall verifies the written file the same way
// the multi -file path does: the result carries a VERIFY line, and a file that
// does not read back as expected makes the result an error. mfInject and
// mmcpText come from the multi -file tests.

func TestMSV_SingleFileVerify(t *testing.T) {
	t.Run("literal_pass", func(t *testing.T) {
		p := wraFile(t, "a x\nb\n")
		res := mcpExecTool("fedit_replaceall", map[string]any{"file": p, "match": "x", "text": "Y"})
		text := mmcpText(t, res)
		if res.IsError {
			t.Fatalf("IsError = true, want false:\n%s", text)
		}
		if !strings.Contains(text, "VERIFY: PASS") {
			t.Errorf("text lacks VERIFY: PASS:\n%s", text)
		}
		if got := wrrRead(t, p); got != "a Y\nb\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("regex_pass", func(t *testing.T) {
		p := wraFile(t, "id=1\nskip\n")
		res := mcpExecTool("fedit_replaceall", map[string]any{"file": p, "match_regex": `id=(\d+)`, "text": "N$1"})
		text := mmcpText(t, res)
		if res.IsError {
			t.Fatalf("IsError = true, want false:\n%s", text)
		}
		if !strings.Contains(text, "VERIFY: PASS") {
			t.Errorf("text lacks VERIFY: PASS:\n%s", text)
		}
		if got := wrrRead(t, p); got != "N1\nskip\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("injected_bad_write_is_error", func(t *testing.T) {
		p := wraFile(t, "a x\nb\n")
		mfInject(t, p)
		res := mcpExecTool("fedit_replaceall", map[string]any{"file": p, "match": "x", "text": "Y"})
		text := mmcpText(t, res)
		if !res.IsError {
			t.Errorf("IsError = false, want true when the file does not read back as written:\n%s", text)
		}
		if !strings.Contains(text, "VERIFY: FAIL") {
			t.Errorf("text lacks VERIFY: FAIL:\n%s", text)
		}
	})
}
