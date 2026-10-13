package main

import (
	"os"
	"strings"
	"testing"
)

// A comma list in file must not combine with dest on copy or move. Nothing may
// be modified and the message must say why.
func TestMCPExec_Dest_RejectsMultiSource(t *testing.T) {
	for _, tool := range []string{"fedit_copy", "fedit_move"} {
		p1 := mcpTempFile(t, "one\ntwo\n")
		defer os.Remove(p1)
		p2 := mcpTempFile(t, "three\n")
		defer os.Remove(p2)
		dst := mcpTempFile(t, "dest\n")
		defer os.Remove(dst)
		r := mcpExecTool(tool, map[string]any{
			"file":  p1 + "," + p2,
			"line":  1,
			"after": 0,
			"dest":  dst,
		})
		t.Logf("%s: error=%v text=%s", tool, r.IsError, mcpText(r))
		if !r.IsError || !strings.Contains(mcpText(r), "multi file list") {
			t.Errorf("%s: want multi file list error, got error=%v text=%s", tool, r.IsError, mcpText(r))
		}
		mraWant(t, p1, "one", "two")
		mraWant(t, p2, "three")
		mraWant(t, dst, "dest")
	}
}

// fedit_replace with dest and a comma list in file must not fall into the
// multi file replace path and overwrite the sources with the placeholder text.
func TestMCPExec_ReplaceDest_MultiSourceNotMutated(t *testing.T) {
	p1 := mcpTempFile(t, "func A() {\n}\n")
	defer os.Remove(p1)
	p2 := mcpTempFile(t, "func A() {\n}\n")
	defer os.Remove(p2)
	dst := mcpTempFile(t, "func B() {\n}\n")
	defer os.Remove(dst)
	r := mcpExecTool("fedit_replace", map[string]any{
		"file":      p1 + "," + p2,
		"match":     "func A",
		"dest":      dst,
		"destblock": "B",
		"lang":      "go",
		"text":      "x",
	})
	t.Logf("fedit_replace: error=%v text=%s", r.IsError, mcpText(r))
	if !r.IsError {
		t.Errorf("want an error for dest with a multi file list, got: %s", mcpText(r))
	}
	mraWant(t, p1, "func A() {", "}")
	mraWant(t, p2, "func A() {", "}")
	mraWant(t, dst, "func B() {", "}")
}
