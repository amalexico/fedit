package main

import (
	"os"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════
// v1.8.0 — negative line indices, -endmatch, -quiet (CLI only)
//
// Scope:
//   PART 1 — Negative line/end indices via MCP (line=-N, end=-N)
//             Colon syntax is CLI-only; MCP only accepts integers.
//   PART 2 — -endmatch content-anchored ranges (show/replace/delete)
//   PART 3 — -quiet: CLI-only flag; verified absent from MCP schemas
// ══════════════════════════════════════════════════════════════

// ── helpers ───────────────────────────────────────────────────

// tenLineFile writes a fresh 10-line file ("line 1".."line 10")
// and returns its path and the slice of lines.
func tenLineFile(t *testing.T) (string, []string) {
	t.Helper()
	lines := testLines(10)
	path := writeTestFile(t, lines)
	return path, lines
}

// ══════════════════════════════════════════════════════════════
// PART 1 — Negative line indices (MCP integer form)
//
// mcpDoShow/Delete/Replace resolve negative line args as:
//   resolved = len(lines) + 1 + lineFlag
// So line=-1 on a 10-line file → line 10 (last).
//    line=-3                   → line 8.
// ══════════════════════════════════════════════════════════════

// ── show with negative line ───────────────────────────────────

func TestMCPExec_Show_NegativeLine_LastLine(t *testing.T) {
	// line=-1 → show last line only
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file": path,
		"line": -1,
	})
	if r.IsError {
		t.Fatalf("show line=-1: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "line 10") {
		t.Errorf("show line=-1: expected 'line 10', got: %q", text)
	}
	if strings.Contains(text, "line 9 ") {
		t.Errorf("show line=-1: leaked 'line 9': %q", text)
	}
}

func TestMCPExec_Show_NegativeLine_ThirdFromEnd(t *testing.T) {
	// line=-3 on 10-line file → line 8
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file": path,
		"line": -3,
	})
	if r.IsError {
		t.Fatalf("show line=-3: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "line 8") {
		t.Errorf("show line=-3: expected 'line 8', got: %q", text)
	}
	// Should be a single line (no end specified → end=start)
	if strings.Contains(text, "line 9") || strings.Contains(text, "line 7") {
		t.Errorf("show line=-3: output wider than one line: %q", text)
	}
}

func TestMCPExec_Show_NegativeLine_NegativeEnd_Range(t *testing.T) {
	// line=-3, end=-1 → lines 8..10
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file": path,
		"line": -3,
		"end":  -1,
	})
	if r.IsError {
		t.Fatalf("show line=-3 end=-1: error: %s", mcpText(r))
	}
	text := mcpText(r)
	for _, want := range []string{"line 8", "line 9", "line 10"} {
		if !strings.Contains(text, want) {
			t.Errorf("show -3..-1: missing %q in output: %q", want, text)
		}
	}
	if strings.Contains(text, "line 7") {
		t.Errorf("show -3..-1: leaked 'line 7': %q", text)
	}
}

func TestMCPExec_Show_NegativeLine_ClampsBeyondStart(t *testing.T) {
	// line=-10 on 10-line file → line 1 (first line)
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file": path,
		"line": -10,
	})
	if r.IsError {
		t.Fatalf("show line=-10: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "line 1") {
		t.Errorf("show line=-10: expected 'line 1', got: %q", text)
	}
}

// ── delete with negative line ─────────────────────────────────

func TestMCPExec_Delete_NegativeLine_LastLine(t *testing.T) {
	// line=-1 → delete last line
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file": path,
		"line": -1,
	})
	if r.IsError {
		t.Fatalf("delete line=-1: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != 9 {
		t.Fatalf("delete line=-1: expected 9 lines, got %d", len(got))
	}
	if got[len(got)-1] != "line 9" {
		t.Errorf("delete line=-1: last line = %q, want 'line 9'", got[len(got)-1])
	}
}

func TestMCPExec_Delete_NegativeLine_Range(t *testing.T) {
	// line=-3, end=-1 → delete last 3 lines (8,9,10)
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file": path,
		"line": -3,
		"end":  -1,
	})
	if r.IsError {
		t.Fatalf("delete line=-3 end=-1: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != 7 {
		t.Fatalf("delete -3..-1: expected 7 lines, got %d", len(got))
	}
	if got[len(got)-1] != "line 7" {
		t.Errorf("delete -3..-1: last line = %q, want 'line 7'", got[len(got)-1])
	}
}

// ── replace with negative line ────────────────────────────────

func TestMCPExec_Replace_NegativeLine_LastLine(t *testing.T) {
	// line=-1 → replace last line
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file": path,
		"line": -1,
		"end":  -1,
		"text": "LAST LINE REPLACED",
	})
	if r.IsError {
		t.Fatalf("replace line=-1: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != 10 {
		t.Fatalf("replace line=-1: expected 10 lines, got %d", len(got))
	}
	if got[9] != "LAST LINE REPLACED" {
		t.Errorf("replace line=-1: got[9]=%q, want 'LAST LINE REPLACED'", got[9])
	}
	if got[8] != "line 9" {
		t.Errorf("replace line=-1: got[8]=%q, want 'line 9' (untouched)", got[8])
	}
}

func TestMCPExec_Replace_NegativeLine_Range(t *testing.T) {
	// line=-3, end=-2 → replace lines 8-9 with single line
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file": path,
		"line": -3,
		"end":  -2,
		"text": "REPLACED 8-9",
	})
	if r.IsError {
		t.Fatalf("replace -3..-2: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	// 10 - 2 + 1 = 9 lines
	if len(got) != 9 {
		t.Fatalf("replace -3..-2: expected 9 lines, got %d", len(got))
	}
	if got[7] != "REPLACED 8-9" {
		t.Errorf("replace -3..-2: got[7]=%q, want 'REPLACED 8-9'", got[7])
	}
	if got[8] != "line 10" {
		t.Errorf("replace -3..-2: got[8]=%q, want 'line 10'", got[8])
	}
}

// ══════════════════════════════════════════════════════════════
// PART 2 — -endmatch content-anchored ranges
//
// resolveSourceLines(lines, 0, 0, match, endmatch, nth) returns
// (startLine, endLine, error).  All endmatch tests run via the
// MCP layer (mcpExecTool) which calls resolveSourceLines internally.
// ══════════════════════════════════════════════════════════════

// ── fixture ───────────────────────────────────────────────────

// endmatchSource is a small fixed file with clearly labelled sections.
const endmatchSource = `// section A start
alpha line 1
alpha line 2
// section A end
// section B start
beta line 1
beta line 2
// section B end
// section C start
gamma line 1
// section C end`

func endmatchFile(t *testing.T) string {
	t.Helper()
	lines := strings.Split(endmatchSource, "\n")
	path := writeTestFile(t, lines)
	t.Cleanup(func() { os.Remove(path) })
	return path
}

// ── show with match+endmatch ──────────────────────────────────

func TestMCPExec_Show_MatchEndmatch_Basic(t *testing.T) {
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "// section B start",
		"endmatch": "// section B end",
	})
	if r.IsError {
		t.Fatalf("show match+endmatch: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "beta line 1") {
		t.Errorf("show match+endmatch: missing 'beta line 1': %q", text)
	}
	if !strings.Contains(text, "beta line 2") {
		t.Errorf("show match+endmatch: missing 'beta line 2': %q", text)
	}
	if strings.Contains(text, "gamma") {
		t.Errorf("show match+endmatch: leaked into section C: %q", text)
	}
	if strings.Contains(text, "alpha") {
		t.Errorf("show match+endmatch: leaked into section A: %q", text)
	}
}

func TestMCPExec_Show_MatchEndmatch_IncludesAnchors(t *testing.T) {
	// Both anchor lines must appear in the output.
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "// section A start",
		"endmatch": "// section A end",
	})
	if r.IsError {
		t.Fatalf("show anchors: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "// section A start") {
		t.Errorf("show anchors: start anchor not in output: %q", text)
	}
	if !strings.Contains(text, "// section A end") {
		t.Errorf("show anchors: end anchor not in output: %q", text)
	}
}

func TestMCPExec_Show_MatchEndmatch_FirstSection(t *testing.T) {
	// Show section A: must contain alpha lines, not beta or gamma.
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "// section A start",
		"endmatch": "// section A end",
	})
	if r.IsError {
		t.Fatalf("show section A: error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "alpha line 1") || !strings.Contains(text, "alpha line 2") {
		t.Errorf("show section A: alpha lines missing: %q", text)
	}
	if strings.Contains(text, "beta") {
		t.Errorf("show section A: leaked beta: %q", text)
	}
}

func TestMCPExec_Show_MatchOnly_NoEndmatch_RequiresEndBound(t *testing.T) {
	// match without endmatch is rejected by resolveSourceLines:
	// "source -match requires an end bound: use -end N or -endmatch TEXT"
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":  path,
		"match": "// section B start",
	})
	if !r.IsError {
		t.Error("show match without endmatch: expected error (end bound required), got success")
	}
	if !strings.Contains(mcpText(r), "end bound") {
		t.Errorf("show match without endmatch: error message should mention end bound, got: %q", mcpText(r))
	}
}

func TestMCPExec_Show_EndmatchBeforeMatch_Error(t *testing.T) {
	// endmatch that appears before match in the file → error
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "// section B start",
		"endmatch": "// section A end", // A end is above B start
	})
	if !r.IsError {
		t.Error("show endmatch before match: expected error, got success")
	}
}

func TestMCPExec_Show_EndmatchNotFound_Error(t *testing.T) {
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "// section A start",
		"endmatch": "DOES NOT EXIST",
	})
	if !r.IsError {
		t.Error("show endmatch not found: expected error, got success")
	}
}

func TestMCPExec_Show_MatchNotFound_Error(t *testing.T) {
	path := endmatchFile(t)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":     path,
		"match":    "NO SUCH LINE",
		"endmatch": "// section A end",
	})
	if !r.IsError {
		t.Error("show match not found: expected error, got success")
	}
}

// ── replace with match+endmatch ───────────────────────────────

func TestMCPExec_Replace_MatchEndmatch_Basic(t *testing.T) {
	path := endmatchFile(t)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// section B start",
		"endmatch": "// section B end",
		"text":     "REPLACED SECTION B",
	})
	if r.IsError {
		t.Fatalf("replace match+endmatch: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	if !strings.Contains(full, "REPLACED SECTION B") {
		t.Errorf("replace match+endmatch: replacement missing: %s", full)
	}
	if strings.Contains(full, "beta line 1") || strings.Contains(full, "beta line 2") {
		t.Errorf("replace match+endmatch: original B content still present: %s", full)
	}
	// Sections A and C survive intact.
	if !strings.Contains(full, "alpha line 1") {
		t.Errorf("replace match+endmatch: section A lost: %s", full)
	}
	if !strings.Contains(full, "gamma line 1") {
		t.Errorf("replace match+endmatch: section C lost: %s", full)
	}
}

func TestMCPExec_Replace_MatchEndmatch_ShrinkRange(t *testing.T) {
	// Replace section A (4 lines) with 2 new lines → file shrinks by 2.
	path := endmatchFile(t)
	origLines, _ := readLines(path)
	origCount := len(origLines)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// section A start",
		"endmatch": "// section A end",
		"text":     "NEW LINE 1\nNEW LINE 2",
	})
	if r.IsError {
		t.Fatalf("replace shrink: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	// 4 lines removed, 2 added → net -2
	if len(got) != origCount-2 {
		t.Errorf("replace shrink: expected %d lines, got %d", origCount-2, len(got))
	}
	full := strings.Join(got, "\n")
	if !strings.Contains(full, "NEW LINE 1") || !strings.Contains(full, "NEW LINE 2") {
		t.Errorf("replace shrink: new lines missing: %s", full)
	}
}

func TestMCPExec_Replace_MatchEndmatch_GrowRange(t *testing.T) {
	// Replace section C (3 lines) with 5 new lines → file grows by 2.
	path := endmatchFile(t)
	origLines, _ := readLines(path)
	origCount := len(origLines)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// section C start",
		"endmatch": "// section C end",
		"text":     "C1\nC2\nC3\nC4\nC5",
	})
	if r.IsError {
		t.Fatalf("replace grow: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != origCount+2 {
		t.Errorf("replace grow: expected %d lines, got %d", origCount+2, len(got))
	}
}

func TestMCPExec_Replace_MatchEndmatch_NthOccurrence(t *testing.T) {
	// File with two identical comment pairs; replace only the 2nd.
	lines := []string{
		"// marker",
		"first body",
		"// end marker",
		"between",
		"// marker",
		"second body",
		"// end marker",
	}
	path := writeTestFile(t, lines)
	defer os.Remove(path)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// marker",
		"endmatch": "// end marker",
		"nth":      2,
		"text":     "SECOND REPLACED",
	})
	if r.IsError {
		t.Fatalf("replace nth=2: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	if !strings.Contains(full, "first body") {
		t.Errorf("replace nth=2: first occurrence was modified: %s", full)
	}
	if !strings.Contains(full, "SECOND REPLACED") {
		t.Errorf("replace nth=2: replacement not found: %s", full)
	}
	if strings.Contains(full, "second body") {
		t.Errorf("replace nth=2: second body still present: %s", full)
	}
}

func TestMCPExec_Replace_MatchEndmatch_LastOccurrence(t *testing.T) {
	// nth=-1 targets last occurrence.
	lines := []string{
		"// mark", "body 1", "// end mark",
		"middle",
		"// mark", "body 2", "// end mark",
		"// mark", "body 3", "// end mark",
	}
	path := writeTestFile(t, lines)
	defer os.Remove(path)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// mark",
		"endmatch": "// end mark",
		"nth":      -1,
		"text":     "LAST REPLACED",
	})
	if r.IsError {
		t.Fatalf("replace nth=-1: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	if strings.Contains(full, "body 3") {
		t.Errorf("replace nth=-1: last occurrence not replaced: %s", full)
	}
	if !strings.Contains(full, "LAST REPLACED") {
		t.Errorf("replace nth=-1: replacement not found: %s", full)
	}
	// First two occurrences untouched.
	if !strings.Contains(full, "body 1") || !strings.Contains(full, "body 2") {
		t.Errorf("replace nth=-1: earlier occurrences damaged: %s", full)
	}
}

// ── delete with match+endmatch ────────────────────────────────

func TestMCPExec_Delete_MatchEndmatch_Basic(t *testing.T) {
	path := endmatchFile(t)
	origLines, _ := readLines(path)
	// section B: start + body1 + body2 + end = 4 lines
	const sectionBLines = 4

	r := mcpExecTool("fedit_delete", map[string]any{
		"file":     path,
		"match":    "// section B start",
		"endmatch": "// section B end",
	})
	if r.IsError {
		t.Fatalf("delete match+endmatch: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != len(origLines)-sectionBLines {
		t.Errorf("delete match+endmatch: expected %d lines, got %d",
			len(origLines)-sectionBLines, len(got))
	}
	full := strings.Join(got, "\n")
	if strings.Contains(full, "beta") {
		t.Errorf("delete match+endmatch: beta lines still present: %s", full)
	}
	if !strings.Contains(full, "alpha line 1") {
		t.Errorf("delete match+endmatch: section A damaged: %s", full)
	}
	if !strings.Contains(full, "gamma line 1") {
		t.Errorf("delete match+endmatch: section C damaged: %s", full)
	}
}

func TestMCPExec_Delete_MatchEndmatch_EntireFile(t *testing.T) {
	// Delete from first match to last match → file empty or single blank line.
	lines := []string{"START", "middle 1", "middle 2", "END"}
	path := writeTestFile(t, lines)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file":     path,
		"match":    "START",
		"endmatch": "END",
	})
	if r.IsError {
		t.Fatalf("delete entire file: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	// writeTestFile may produce a trailing newline yielding one empty string;
	// treat both 0 and [""] as "effectively empty".
	nonEmpty := 0
	for _, l := range got {
		if l != "" {
			nonEmpty++
		}
	}
	if nonEmpty != 0 {
		t.Errorf("delete entire: expected empty file, got %d non-empty lines: %v", nonEmpty, got)
	}
}

func TestMCPExec_Delete_MatchEndmatch_SameAnchor_SingleLine(t *testing.T) {
	// match == endmatch → deletes exactly one line (the first occurrence).
	lines := []string{"AAA", "BBB", "CCC", "BBB", "DDD"}
	path := writeTestFile(t, lines)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file":     path,
		"match":    "BBB",
		"endmatch": "BBB",
	})
	if r.IsError {
		t.Fatalf("delete same anchor: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	if len(got) != 4 {
		t.Fatalf("delete same anchor: expected 4 lines, got %d: %v", len(got), got)
	}
	// First BBB gone; second BBB survives at index 2.
	if got[0] != "AAA" || got[1] != "CCC" || got[2] != "BBB" || got[3] != "DDD" {
		t.Errorf("delete same anchor: wrong result: %v", got)
	}
}

func TestMCPExec_Delete_MatchEndmatch_NthOccurrence(t *testing.T) {
	lines := []string{
		"// block", "body 1", "// end block",
		"gap",
		"// block", "body 2", "// end block",
	}
	path := writeTestFile(t, lines)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file":     path,
		"match":    "// block",
		"endmatch": "// end block",
		"nth":      2,
	})
	if r.IsError {
		t.Fatalf("delete nth=2: error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")
	if !strings.Contains(full, "body 1") {
		t.Errorf("delete nth=2: first block damaged: %s", full)
	}
	if strings.Contains(full, "body 2") {
		t.Errorf("delete nth=2: second block still present: %s", full)
	}
}

// ── endmatch: MCP schema contains the property ───────────────

func TestMCPSchema_ShowHasEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_show" {
			continue
		}
		if !strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_show schema: missing 'endmatch' property")
		}
		return
	}
	t.Error("fedit_show not found in tool defs")
}

func TestMCPSchema_ReplaceHasEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_replace" {
			continue
		}
		if !strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_replace schema: missing 'endmatch' property")
		}
		return
	}
	t.Error("fedit_replace not found in tool defs")
}

func TestMCPSchema_DeleteHasEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_delete" {
			continue
		}
		if !strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_delete schema: missing 'endmatch' property")
		}
		return
	}
	t.Error("fedit_delete not found in tool defs")
}

func TestMCPSchema_MoveHasEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_move" {
			continue
		}
		if !strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_move schema: missing 'endmatch' property")
		}
		return
	}
	t.Error("fedit_move not found in tool defs")
}

func TestMCPSchema_CopyHasEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_copy" {
			continue
		}
		if !strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_copy schema: missing 'endmatch' property")
		}
		return
	}
	t.Error("fedit_copy not found in tool defs")
}

// ── endmatch not present on read-only / non-range ops ─────────

func TestMCPSchema_FindHasNoEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_find" {
			continue
		}
		if strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_find schema: 'endmatch' should not be present (find uses match only)")
		}
		return
	}
	t.Error("fedit_find not found in tool defs")
}

func TestMCPSchema_ReplaceallHasNoEndmatch(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_replaceall" {
			continue
		}
		if strings.Contains(string(schemaBytes(def)), `"endmatch"`) {
			t.Errorf("fedit_replaceall schema: 'endmatch' should not be present")
		}
		return
	}
	t.Error("fedit_replaceall not found in tool defs")
}

// ══════════════════════════════════════════════════════════════
// PART 3 — -quiet flag (CLI only)
//
// quiet is not wired into mcp.go at all: mcpExecTool does not
// read a "quiet" argument, and no mcpDo* function takes one.
// The MCP result is always mcpOK(msg) with === STATS ===.
//
// These tests confirm:
//   a) quiet is absent from every MCP tool schema
//   b) the MCP layer always emits STATS (never suppressed)
// ══════════════════════════════════════════════════════════════

func TestMCPSchema_QuietAbsentFromAllTools(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if strings.Contains(string(schemaBytes(def)), `"quiet"`) {
			t.Errorf("tool %q schema: 'quiet' must not appear (CLI-only flag)", def.Name)
		}
	}
}

func TestMCPExec_Replace_AlwaysEmitsStats(t *testing.T) {
	// STATS block is always present in the MCP result (quiet does not suppress it).
	// Use match+endmatch form since fedit_replace requires explicit end when using line.
	path := endmatchFile(t)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":     path,
		"match":    "// section C start",
		"endmatch": "// section C end",
		"text":     "REPLACED C",
	})
	if r.IsError {
		t.Fatalf("replace stats check: error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "STATS") {
		t.Errorf("replace: expected STATS block in MCP result, got: %q", mcpText(r))
	}
}

func TestMCPExec_Delete_AlwaysEmitsStats(t *testing.T) {
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_delete", map[string]any{
		"file": path,
		"line": 3,
	})
	if r.IsError {
		t.Fatalf("delete stats check: error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "STATS") {
		t.Errorf("delete: expected STATS block in MCP result, got: %q", mcpText(r))
	}
}

func TestMCPExec_Insert_AlwaysEmitsStats(t *testing.T) {
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_insert", map[string]any{
		"file": path,
		"line": 2,
		"text": "INSERTED",
	})
	if r.IsError {
		t.Fatalf("insert stats check: error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "STATS") {
		t.Errorf("insert: expected STATS block in MCP result, got: %q", mcpText(r))
	}
}

func TestMCPExec_Replaceall_AlwaysEmitsStats(t *testing.T) {
	path, _ := tenLineFile(t)
	defer os.Remove(path)

	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":  path,
		"match": "line 1",
		"text":  "LINE ONE",
	})
	if r.IsError {
		t.Fatalf("replaceall stats check: error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "STATS") {
		t.Errorf("replaceall: expected STATS block in MCP result, got: %q", mcpText(r))
	}
}
