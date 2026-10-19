package main

import (
	"strings"
	"testing"
)

// The MCP tool result for a multi -file replaceall: a file that was written
// but failed verification must make the whole result an error, not only the
// case where no file succeeded. mfInject and mfOpts come from
// multi_fail_test.go.

func mmcpText(t *testing.T, r mcpCallResult) string {
	t.Helper()
	if len(r.Content) == 0 {
		t.Fatal("result has no content")
	}
	return r.Content[0].Text
}

func TestMMCP_FailPath(t *testing.T) {
	t.Run("error_when_one_file_fails_verification", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\n")
		mfInject(t, a)
		res := mcpRunMultiMutate(mfOpts(a, b))
		text := mmcpText(t, res)
		if !res.IsError {
			t.Errorf("IsError = false, want true when a written file failed verification:\n%s", text)
		}
		if !strings.Contains(text, "VERIFY: FAIL") {
			t.Errorf("text lacks VERIFY: FAIL:\n%s", text)
		}
	})

	t.Run("ok_when_every_file_verifies", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\n")
		res := mcpRunMultiMutate(mfOpts(a, b))
		text := mmcpText(t, res)
		if res.IsError {
			t.Errorf("IsError = true, want false:\n%s", text)
		}
		if !strings.Contains(text, "VERIFY: PASS") {
			t.Errorf("text lacks VERIFY: PASS:\n%s", text)
		}
	})

	t.Run("error_when_every_file_fails", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\nc\n")
		mfInject(t, a, b)
		res := mcpRunMultiMutate(mfOpts(a, b))
		if text := mmcpText(t, res); !res.IsError {
			t.Errorf("IsError = false, want true:\n%s", text)
		}
	})
}
