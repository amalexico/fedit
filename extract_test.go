package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// testFile used across all extract tests — mirrors the file created in the
// manual session so tests are grounded in real-world content.
// Line 1: First line involves some content
// Line 2:     Second line has tab and number 09876154.4567 that way we can test
// Line 3: Third line has some symbols $%^&*!@#""''
// Line 4: This is the last
const extractTestContent = "First line involves some content\n" +
	"    Second line has tab and number 09876154.4567 that way we can test\n" +
	"Third line has some symbols $%^&*!@#\"\"''\n" +
	"This is the last\n"

func writeExtractTestFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "fedit_extract_*.txt")
	if err != nil {
		t.Fatalf("writeExtractTestFile: %v", err)
	}
	f.Close()
	if err := writeLines(f.Name(), strings.Split(strings.TrimRight(extractTestContent, "\n"), "\n")); err != nil {
		t.Fatalf("writeExtractTestFile write: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// ── parseExtractSpec ──────────────────────────────────────────────────────────

func TestParseExtractSpec_WordOnly(t *testing.T) {
	cases := []struct {
		spec  string
		wordN int
	}{
		{"W1", 1},
		{"W3", 3},
		{"W10", 10},
		{"W99", 99},
	}
	for _, c := range cases {
		es, err := parseExtractSpec(c.spec)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.spec, err)
			continue
		}
		if es.wordN != c.wordN {
			t.Errorf("%s: wordN = %d, want %d", c.spec, es.wordN, c.wordN)
		}
		if es.charStart != 0 || es.charCount != 0 || es.subfieldDelim != "" {
			t.Errorf("%s: expected no char range or subfield, got %+v", c.spec, es)
		}
	}
}

func TestParseExtractSpec_CharRangeWithCount(t *testing.T) {
	cases := []struct {
		spec      string
		wordN     int
		charStart int
		charCount int
	}{
		{"W6[2:3]", 6, 2, 3},
		{"W5[1:3]", 5, 1, 3},
		{"W1[1:8]", 1, 1, 8},
		{"W3[4:1]", 3, 4, 1},
	}
	for _, c := range cases {
		es, err := parseExtractSpec(c.spec)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.spec, err)
			continue
		}
		if es.wordN != c.wordN || es.charStart != c.charStart || es.charCount != c.charCount {
			t.Errorf("%s: got {wordN:%d charStart:%d charCount:%d}, want {%d %d %d}",
				c.spec, es.wordN, es.charStart, es.charCount, c.wordN, c.charStart, c.charCount)
		}
	}
}

func TestParseExtractSpec_CharRangeToEnd(t *testing.T) {
	es, err := parseExtractSpec("W3[2:]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if es.wordN != 3 || es.charStart != 2 || es.charCount != 0 {
		t.Errorf("got %+v, want {wordN:3 charStart:2 charCount:0}", es)
	}
}

func TestParseExtractSpec_Subfield(t *testing.T) {
	cases := []struct {
		spec          string
		wordN         int
		subfieldDelim string
		subfieldN     int
	}{
		{"W1/./1", 1, ".", 1},
		{"W1/./2", 1, ".", 2},
		{"W7/./1", 7, ".", 1},
		{"W3/,/2", 3, ",", 2},
		{"W2/:/3", 2, ":", 3},
	}
	for _, c := range cases {
		es, err := parseExtractSpec(c.spec)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.spec, err)
			continue
		}
		if es.wordN != c.wordN || es.subfieldDelim != c.subfieldDelim || es.subfieldN != c.subfieldN {
			t.Errorf("%s: got %+v, want {wordN:%d subfieldDelim:%q subfieldN:%d}",
				c.spec, es, c.wordN, c.subfieldDelim, c.subfieldN)
		}
	}
}

func TestParseExtractSpec_SlashDelimiter(t *testing.T) {
	// Delimiter that is itself "/" — W1///2 means word 1, split by "/", field 2
	es, err := parseExtractSpec("W1///2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if es.subfieldDelim != "/" || es.subfieldN != 2 {
		t.Errorf("got subfieldDelim=%q subfieldN=%d, want /  2", es.subfieldDelim, es.subfieldN)
	}
}

func TestParseExtractSpec_Errors(t *testing.T) {
	bad := []string{
		"",        // empty
		"w3",      // lowercase W
		"3",       // no W prefix
		"W0",      // word 0 invalid
		"Wabc",    // non-numeric word number
		"W3[0:2]", // charStart 0
		"W3[2:0]", // charCount 0
		"W3[abc:2]",  // non-numeric start
		"W3[2:abc]",  // non-numeric count
		"W3[2",    // missing ]
		"W1//1",   // empty delimiter (slashIdx < 1)
		"W1/./0",  // subfield 0
		"W1/./abc", // non-numeric subfield
	}
	for _, spec := range bad {
		_, err := parseExtractSpec(spec)
		if err == nil {
			t.Errorf("expected error for %q, got nil", spec)
		}
	}
}

// ── applyExtract ─────────────────────────────────────────────────────────────

// Line 2 (trimmed): "Second line has tab and number 09876154.4567 that way we can test"
// Words (normalized): Second(1) line(2) has(3) tab(4) and(5) number(6)
//                     09876154.4567(7) that(8) way(9) we(10) can(11) test(12)
const line2 = "    Second line has tab and number 09876154.4567 that way we can test"

// Line 3: "Third line has some symbols $%^&*!@#\"\"''"
// Words: Third(1) line(2) has(3) some(4) symbols(5) $%^&*!@#""''(6)
const line3 = `Third line has some symbols $%^&*!@#""''`

func TestApplyExtract_WordOnly(t *testing.T) {
	cases := []struct{ spec, line, want string }{
		{"W1", line2, "Second"},
		{"W3", line2, "has"},
		{"W6", line2, "number"},
		{"W7", line2, "09876154.4567"},
		{"W5", line3, "symbols"},
		{"W1", line3, "Third"},
	}
	for _, c := range cases {
		es, _ := parseExtractSpec(c.spec)
		got, ok := applyExtract(c.line, es, "")
		if !ok {
			t.Errorf("%s on %q: ok=false, want %q", c.spec, c.line[:20], c.want)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.spec, got, c.want)
		}
	}
}

func TestApplyExtract_CharRange(t *testing.T) {
	cases := []struct {
		spec, line, want string
	}{
		// W6 = "number"; [2:3] = chars 2,3,4 = "umb"
		{"W6[2:3]", line2, "umb"},
		// W5 = "symbols"; [1:3] = "sym"
		{"W5[1:3]", line3, "sym"},
		// W7 = "09876154.4567"; [1:8] = "09876154"
		{"W7[1:8]", line2, "09876154"},
		// W6 = "number"; [2:] = chars 2 to end = "umber"
		{"W6[2:]", line2, "umber"},
		// W1 = "Second"; [1:3] = "Sec"
		{"W1[1:3]", line2, "Sec"},
	}
	for _, c := range cases {
		es, err := parseExtractSpec(c.spec)
		if err != nil {
			t.Fatalf("%s: parse error: %v", c.spec, err)
		}
		got, ok := applyExtract(c.line, es, "")
		if !ok {
			t.Errorf("%s: ok=false, want %q", c.spec, c.want)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.spec, got, c.want)
		}
	}
}

func TestApplyExtract_CharRangeToEnd(t *testing.T) {
	// W6 = "number"; [3:] = "mber"
	es, _ := parseExtractSpec("W6[3:]")
	got, ok := applyExtract(line2, es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "mber" {
		t.Errorf("got %q, want %q", got, "mber")
	}
}

func TestApplyExtract_SubfieldDot_IntegerPart(t *testing.T) {
	// W7 = "09876154.4567"; /./1 → "09876154"
	es, _ := parseExtractSpec("W7/./1")
	got, ok := applyExtract(line2, es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "09876154" {
		t.Errorf("got %q, want %q", got, "09876154")
	}
}

func TestApplyExtract_SubfieldDot_DecimalPart(t *testing.T) {
	// W7 = "09876154.4567"; /./2 → "4567"
	es, _ := parseExtractSpec("W7/./2")
	got, ok := applyExtract(line2, es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "4567" {
		t.Errorf("got %q, want %q", got, "4567")
	}
}

func TestApplyExtract_SilentSkip_WordOutOfRange(t *testing.T) {
	// "This is the last" has 4 words; W5 should silently skip
	es, _ := parseExtractSpec("W5")
	_, ok := applyExtract("This is the last", es, "")
	if ok {
		t.Error("expected ok=false for word out of range, got true")
	}
}

func TestApplyExtract_SilentSkip_CharStartBeyondWord(t *testing.T) {
	// W1 = "Hi" (2 chars); [5:2] — start beyond word length
	es, _ := parseExtractSpec("W1[5:2]")
	_, ok := applyExtract("Hi there", es, "")
	if ok {
		t.Error("expected ok=false for charStart beyond word, got true")
	}
}

func TestApplyExtract_SilentSkip_SubfieldOutOfRange(t *testing.T) {
	// "hello.world" has 2 subfields on '.'; field 3 → skip
	es, _ := parseExtractSpec("W1/./3")
	_, ok := applyExtract("hello.world", es, "")
	if ok {
		t.Error("expected ok=false for subfield out of range, got true")
	}
}

func TestApplyExtract_CharRangeClampsToWordEnd(t *testing.T) {
	// W1 = "Hi" (2 chars); [1:10] — count exceeds length, clamp to end
	es, _ := parseExtractSpec("W1[1:10]")
	got, ok := applyExtract("Hi there", es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "Hi" {
		t.Errorf("got %q, want %q", got, "Hi")
	}
}

func TestApplyExtract_CustomWdelim(t *testing.T) {
	// Using comma as word delimiter
	es, _ := parseExtractSpec("W2")
	got, ok := applyExtract("alpha,beta,gamma", es, ",")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "beta" {
		t.Errorf("got %q, want %q", got, "beta")
	}
}

func TestApplyExtract_NormalizedWhitespace(t *testing.T) {
	// Multiple spaces/tabs between words — normalized to single split
	got_es, _ := parseExtractSpec("W2")
	got, ok := applyExtract("word1   \t  word2   word3", got_es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "word2" {
		t.Errorf("got %q, want %q", got, "word2")
	}
}

func TestApplyExtract_LeadingWhitespaceIgnored(t *testing.T) {
	// Leading spaces must NOT create empty word slots (awk-style normalization)
	es, _ := parseExtractSpec("W1")
	got, ok := applyExtract("    First word", es, "")
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "First" {
		t.Errorf("leading space counted as word: got %q, want %q", got, "First")
	}
}

// ── applyGet ─────────────────────────────────────────────────────────────────

func TestApplyGet_BasicNumber(t *testing.T) {
	got, ok := applyGet(line2, `\d+\.\d+`)
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "09876154.4567" {
		t.Errorf("got %q, want %q", got, "09876154.4567")
	}
}

func TestApplyGet_IntegerOnly(t *testing.T) {
	// Matches first integer in the line
	got, ok := applyGet(line2, `\d+`)
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "09876154" {
		t.Errorf("got %q, want %q", got, "09876154")
	}
}

func TestApplyGet_NoMatch_SilentSkip(t *testing.T) {
	_, ok := applyGet("no numbers here", `\d+`)
	if ok {
		t.Error("expected ok=false for no match")
	}
}

func TestApplyGet_InvalidRegex_SilentSkip(t *testing.T) {
	_, ok := applyGet("anything", `[invalid`)
	if ok {
		t.Error("expected ok=false for invalid regex")
	}
}

func TestApplyGet_CaptureGroup(t *testing.T) {
	// FindString returns the full match, not just the capture group
	got, ok := applyGet("version=1.2.3 other", `version=(\S+)`)
	if !ok {
		t.Fatal("ok=false")
	}
	if got != "version=1.2.3" {
		t.Errorf("got %q", got)
	}
}

// ── doFind with extract ───────────────────────────────────────────────────────

func TestDoFind_Extract_WordOnly(t *testing.T) {
	// Task 2: third word of line 2 (line containing "09876154")
	lines, _ := readLines(writeExtractTestFile(t))
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, "", "W3", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "has" {
		t.Errorf("W3 of line with 09876154: got %q, want %q", got, "has")
	}
}

func TestDoFind_Extract_CharRange(t *testing.T) {
	// W6[2:3] on line containing "098" → "umb" (from "number")
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "098", 0, false, "", "W6[2:3]", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "umb" {
		t.Errorf("W6[2:3]: got %q, want %q", got, "umb")
	}
}

func TestDoFind_Extract_IntegerPartViaSubfield(t *testing.T) {
	// Task 3: integer part of number on line 2
	// W7/./1 → "09876154"
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, "", "W7/./1", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "09876154" {
		t.Errorf("W7/./1: got %q, want %q", got, "09876154")
	}
}

func TestDoFind_Extract_DecimalPartViaSubfield(t *testing.T) {
	// Task 4: decimal part of number on line 2
	// W7/./2 → "4567"
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, "", "W7/./2", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "4567" {
		t.Errorf("W7/./2: got %q, want %q", got, "4567")
	}
}

func TestDoFind_Extract_Word5CharRange_Line3(t *testing.T) {
	// Task 5: first 3 chars of word 5 of line 3
	// W5[1:3] on line containing "symbols" → "sym"
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "symbols", 0, false, "", "W5[1:3]", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "sym" {
		t.Errorf("W5[1:3]: got %q, want %q", got, "sym")
	}
}

func TestDoFind_Get_Then_Extract(t *testing.T) {
	// -get extracts "09876154.4567", then -extract W1[1:3] → "098"
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, `\d+\.\d+`, "W1[1:3]", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "098" {
		t.Errorf("get+extract: got %q, want %q", got, "098")
	}
}

func TestDoFind_Get_Only_NoExtract(t *testing.T) {
	// -get without -extract: outputs the regex match
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, `\d+\.\d+`, "", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "09876154.4567" {
		t.Errorf("get only: got %q, want %q", got, "09876154.4567")
	}
}

func TestDoFind_Get_IntegerPartViaSubfield(t *testing.T) {
	// -get \d+\.\d+ → "09876154.4567", -extract W1/./1 → "09876154"
	// Alternative to W7/./1 — uses get to isolate the number first
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "09876154", 0, false, `\d+\.\d+`, "W1/./1", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	got := strings.TrimSpace(string(buf[:n]))
	if got != "09876154" {
		t.Errorf("get+W1/./1: got %q, want %q", got, "09876154")
	}
}

func TestDoFind_Extract_MultipleMatchLines(t *testing.T) {
	// -extract on a match that hits multiple lines returns one result per line
	lines, _ := readLines(writeExtractTestFile(t))
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	// "line" appears in lines 1,2,3: "First LINE", "Second LINE", "Third LINE"
	doFind(lines, "line", 0, false, "", "W1", "")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	results := strings.Split(output, "\n")
	// Lines 1,2,3 all contain "line" (case-sensitive: "line", "line", "line")
	// W1 of each: "First", "Second", "Third" (leading spaces normalized)
	if len(results) < 3 {
		t.Fatalf("expected >= 3 results, got %d: %v", len(results), results)
	}
	if results[0] != "First" {
		t.Errorf("result[0] = %q, want %q", results[0], "First")
	}
	if results[1] != "Second" {
		t.Errorf("result[1] = %q, want %q", results[1], "Second")
	}
	if results[2] != "Third" {
		t.Errorf("result[2] = %q, want %q", results[2], "Third")
	}
}

func TestDoFind_Extract_SilentSkip_NoWordInSomeLine(t *testing.T) {
	// W5 on lines: "A B C D" has only 4 words → skip that line
	path, _ := os.CreateTemp("", "fedit_skip_*.txt")
	path.Close()
	defer os.Remove(path.Name())
	writeLines(path.Name(), []string{"one two three four five", "short line", "alpha beta gamma delta epsilon"})

	lines, _ := readLines(path.Name())
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFind(lines, "o", 0, false, "", "W5", "") // all 3 lines match "o"
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	output := strings.TrimSpace(string(buf[:n]))
	results := strings.Split(output, "\n")
	// "short line" (2 words) and "alpha beta gamma delta epsilon" (5 words) match "o"
	// W5 of "one two three four five" = "five" ✓
	// W5 of "short line" → skip (only 2 words)
	// W5 of "alpha beta gamma delta epsilon" = "epsilon" ✓
	if len(results) != 2 {
		t.Errorf("expected 2 results (1 skipped), got %d: %v", len(results), results)
	}
}

// ── MCP fedit_find with extract ───────────────────────────────────────────────

func TestMCPExec_Find_Extract_WordOnly(t *testing.T) {
	path := mcpTempFile(t, "    Second line has tab and number 09876154.4567 that way we can test\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":    path,
		"match":   "09876154",
		"extract": "W3",
	})
	if r.IsError {
		t.Fatalf("extract W3 error: %s", mcpText(r))
	}
	text := strings.TrimSpace(mcpText(r))
	if !strings.Contains(text, "has") {
		t.Errorf("W3 should contain 'has', got: %q", text)
	}
}

func TestMCPExec_Find_Extract_SubfieldIntegerPart(t *testing.T) {
	path := mcpTempFile(t, "    Second line has tab and number 09876154.4567 that way we can test\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":    path,
		"match":   "09876154",
		"extract": "W7/./1",
	})
	if r.IsError {
		t.Fatalf("extract W7/./1 error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "09876154") {
		t.Errorf("W7/./1 should return integer part, got: %q", mcpText(r))
	}
}

func TestMCPExec_Find_Extract_DecimalPart(t *testing.T) {
	path := mcpTempFile(t, "    Second line has tab and number 09876154.4567 that way we can test\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":    path,
		"match":   "09876154",
		"extract": "W7/./2",
	})
	if r.IsError {
		t.Fatalf("extract W7/./2 error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "4567") {
		t.Errorf("W7/./2 should return decimal part, got: %q", mcpText(r))
	}
}

func TestMCPExec_Find_Get_Then_Extract(t *testing.T) {
	path := mcpTempFile(t, "    Second line has tab and number 09876154.4567 that way we can test\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":    path,
		"match":   "09876154",
		"get":     `\d+\.\d+`,
		"extract": "W1[1:3]",
	})
	if r.IsError {
		t.Fatalf("get+extract error: %s", mcpText(r))
	}
	text := strings.TrimSpace(strings.Split(mcpText(r), "\n")[0])
	if text != "098" {
		t.Errorf("get+W1[1:3]: got %q, want %q", text, "098")
	}
}

func TestMCPToolDefs_FindHasExtractParam(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_find" {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			t.Fatalf("fedit_find: invalid JSON schema: %v", err)
		}
		props, _ := schema["properties"].(map[string]any)
		for _, param := range []string{"extract", "get", "wdelim"} {
			if _, ok := props[param]; !ok {
				t.Errorf("fedit_find: missing property %q in schema", param)
			}
		}
		return
	}
	t.Error("fedit_find not found in tool definitions")
}
