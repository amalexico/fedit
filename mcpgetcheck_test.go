package main

import (
	"os"
	"strings"
	"testing"
)

func TestMGC_CheckGet_Unit(t *testing.T) {
	if got := mcpCheckGet(""); got != "" {
		t.Errorf("empty get: want no message, got %q", got)
	}
	if got := mcpCheckGet(`\d+`); got != "" {
		t.Errorf("valid get: want no message, got %q", got)
	}
	if got := mcpCheckGet("("); !strings.HasPrefix(got, "get: invalid regex:") {
		t.Errorf("invalid get: want a get: invalid regex message, got %q", got)
	}
}

func TestMGC_Find_InvalidGet_IsError(t *testing.T) {
	path := mcpTempFile(t, "alpha 123\nbeta 456\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{"file": path, "match": "a", "get": "("})
	if !r.IsError || !strings.Contains(mcpText(r), "invalid regex") {
		t.Errorf("want an invalid regex error, got error=%v text=%q", r.IsError, mcpText(r))
	}
}

func TestMGC_Find_InvalidGet_WithExtract_IsError(t *testing.T) {
	path := mcpTempFile(t, "alpha 123\nbeta 456\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{"file": path, "match": "a", "get": "(", "extract": "W1"})
	if !r.IsError || !strings.Contains(mcpText(r), "invalid regex") {
		t.Errorf("want an invalid regex error, got error=%v text=%q", r.IsError, mcpText(r))
	}
}

func TestMGC_Find_InvalidGet_MultiFile_IsError(t *testing.T) {
	a := mcpTempFile(t, "alpha 123\n")
	defer os.Remove(a)
	b := mcpTempFile(t, "beta 456\n")
	defer os.Remove(b)
	r := mcpExecTool("fedit_find", map[string]any{"file": a + "," + b, "match": "a", "get": "("})
	txt := mcpText(r)
	if !r.IsError || !strings.Contains(txt, "invalid regex") {
		t.Errorf("want an invalid regex error over a file list, got error=%v text=%q", r.IsError, txt)
	}
	if strings.Contains(txt, "==>") {
		t.Errorf("no per-file output expected when get is invalid, got %q", txt)
	}
}

func TestMGC_Find_ValidGet_StillWorks(t *testing.T) {
	path := mcpTempFile(t, "alpha 123\nbeta 456\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{"file": path, "match": "alpha", "get": `\d+`})
	if r.IsError || !strings.Contains(mcpText(r), "123") {
		t.Errorf("valid get: want 123, got error=%v text=%q", r.IsError, mcpText(r))
	}
}
