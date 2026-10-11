package main

import (
	"os"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════
// map -lang json / map -lang csv (v1.9.0)
// ══════════════════════════════════════════════════════════════

func captureMapOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old

	buf := make([]byte, 8192)
	n, _ := r.Read(buf)
	return string(buf[:n])
}

// ── JSON ─────────────────────────────────────────────────────

func TestDoMapJSON_Clean_NoDuplicates(t *testing.T) {
	lines := []string{
		`{`,
		`  "name": "Alex",`,
		`  "role": "founder",`,
		`  "projects": ["fedit", "fwrite"],`,
		`  "meta": {"org": "Amalex Corp", "since": 2023}`,
		`}`,
	}
	out := captureMapOutput(t, func() { doMapJSON(lines) })

	if !strings.Contains(out, "✓ Valid JSON syntax") {
		t.Errorf("expected valid JSON syntax, got: %s", out)
	}
	if !strings.Contains(out, "✓ No duplicate keys within any object") {
		t.Errorf("expected no-duplicates line, got: %s", out)
	}
	if !strings.Contains(out, "Root type: object") {
		t.Errorf("expected root type object, got: %s", out)
	}
}

func TestDoMapJSON_DuplicateKeys_SameObject(t *testing.T) {
	lines := []string{
		`{`,
		`  "name": "Alex",`,
		`  "name": "Duplicate",`,
		`  "meta": {"org": "Amalex", "org": "Second"}`,
		`}`,
	}
	out := captureMapOutput(t, func() { doMapJSON(lines) })

	if !strings.Contains(out, "⚠ DUPLICATE KEYS") {
		t.Errorf("expected duplicate key warning, got: %s", out)
	}
	if !strings.Contains(out, `"name" appears 2 times, first seen at line 2`) {
		t.Errorf("expected 'name' duplicate at line 2, got: %s", out)
	}
	if !strings.Contains(out, `"org" appears 2 times`) {
		t.Errorf("expected 'org' duplicate, got: %s", out)
	}
}

func TestDoMapJSON_DuplicateKeys_DifferentObjects_NotFlagged(t *testing.T) {
	// Same key name in two DIFFERENT objects is not a duplicate.
	lines := []string{
		`{`,
		`  "a": {"id": 1},`,
		`  "b": {"id": 2}`,
		`}`,
	}
	out := captureMapOutput(t, func() { doMapJSON(lines) })

	if !strings.Contains(out, "✓ No duplicate keys within any object") {
		t.Errorf("cross-object same key name should not be flagged, got: %s", out)
	}
}

func TestDoMapJSON_Malformed_SkipsStructureAndDuplicates(t *testing.T) {
	lines := []string{
		`{`,
		`  "name": "Alex",`,
		`  "role": "founder",`,
	}
	out := captureMapOutput(t, func() { doMapJSON(lines) })

	if !strings.Contains(out, "✗ Malformed JSON") {
		t.Errorf("expected malformed JSON marker, got: %s", out)
	}
	if strings.Count(out, "skipped") != 2 {
		t.Errorf("expected structure AND duplicate sections both skipped, got: %s", out)
	}
}

func TestDoMapJSON_TrailingGarbage_Invalid(t *testing.T) {
	lines := []string{`{"a": 1}`, `{"b": 2}`}
	out := captureMapOutput(t, func() { doMapJSON(lines) })

	if !strings.Contains(out, "✗ Malformed JSON") {
		t.Errorf("two root values should be malformed, got: %s", out)
	}
}

// ── CSV ──────────────────────────────────────────────────────

func TestDoMapCSV_Clean_NoIssues(t *testing.T) {
	lines := []string{
		"name,role,email",
		"Alex,founder,alex@amalexhandler.com",
		"Sam,engineer,sam@amalexhandler.com",
	}
	out := captureMapOutput(t, func() { doMapCSV(lines) })

	if !strings.Contains(out, "✓ Valid CSV syntax") {
		t.Errorf("expected valid CSV, got: %s", out)
	}
	if !strings.Contains(out, "✓ No ragged rows") {
		t.Errorf("expected no ragged rows, got: %s", out)
	}
	if !strings.Contains(out, "✓ No duplicate header columns") {
		t.Errorf("expected no duplicate headers, got: %s", out)
	}
	if !strings.Contains(out, "✓ No empty fields") {
		t.Errorf("expected no empty fields, got: %s", out)
	}
}

func TestDoMapCSV_RaggedRows_Flagged(t *testing.T) {
	lines := []string{
		"name,role,email",
		"Alex,founder,alex@amalexhandler.com",
		"Sam,engineer",
		"Jo,designer,jo@amalexhandler.com,extra",
	}
	out := captureMapOutput(t, func() { doMapCSV(lines) })

	if !strings.Contains(out, "⚠ RAGGED ROWS") {
		t.Errorf("expected ragged row warning, got: %s", out)
	}
	if !strings.Contains(out, "expected 3 fields, got 2") {
		t.Errorf("expected short-row detail, got: %s", out)
	}
	if !strings.Contains(out, "expected 3 fields, got 4") {
		t.Errorf("expected long-row detail, got: %s", out)
	}
}

func TestDoMapCSV_DuplicateHeaders_Flagged(t *testing.T) {
	lines := []string{
		"name,role,email,name",
		"Alex,founder,alex@amalexhandler.com,Alex2",
	}
	out := captureMapOutput(t, func() { doMapCSV(lines) })

	if !strings.Contains(out, "⚠ DUPLICATE HEADER COLUMNS") {
		t.Errorf("expected duplicate header warning, got: %s", out)
	}
	if !strings.Contains(out, `"name" at columns: 1, 4`) {
		t.Errorf("expected duplicate header columns 1 and 4, got: %s", out)
	}
}

func TestDoMapCSV_EmptyFields_Counted(t *testing.T) {
	lines := []string{
		"name,role,email",
		"Alex,,alex@amalexhandler.com",
		",engineer,",
	}
	out := captureMapOutput(t, func() { doMapCSV(lines) })

	if !strings.Contains(out, "⚠ 3 empty field(s) found") {
		t.Errorf("expected 3 empty fields, got: %s", out)
	}
}

func TestDoMapCSV_MalformedQuoting_StillReportsHeader(t *testing.T) {
	lines := []string{
		"name,role",
		`"unterminated,founder`,
	}
	out := captureMapOutput(t, func() { doMapCSV(lines) })

	if !strings.Contains(out, "✗ Malformed CSV") {
		t.Errorf("expected malformed CSV marker, got: %s", out)
	}
	if !strings.Contains(out, "Header: name, role") {
		t.Errorf("header should still be reported despite later parse error, got: %s", out)
	}
}

func TestDoMapCSV_EmptyFile(t *testing.T) {
	out := captureMapOutput(t, func() { doMapCSV([]string{}) })

	if !strings.Contains(out, "(empty file — no rows)") {
		t.Errorf("expected empty-file message, got: %s", out)
	}
	if !strings.Contains(out, "(skipped — no header row available)") {
		t.Errorf("expected duplicate-header section to skip cleanly, got: %s", out)
	}
}
