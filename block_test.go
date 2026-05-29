package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// goSource is a small but complete Go source file that exercises block resolution.
// It mirrors the real use case: two functions, one of which will be replaced
// or have content inserted around it.
const goSource = `package demo

import "fmt"

func Hello(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

func Goodbye(name string) string {
	return fmt.Sprintf("Goodbye, %s!", name)
}

func helper() {
	// internal helper
}
`

// goSourceReplacement is the new function body used in replace-block tests.
const goSourceReplacement = `func Hello(name string) string {
	return fmt.Sprintf("Greetings, %s!", name)
}`

// goSourceNewFunc is inserted before/after existing functions.
const goSourceNewFunc = `func Farewell(name string) string {
	return fmt.Sprintf("Farewell, %s!", name)
}
`

func blockTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "fedit_block_*.go")
	if err != nil {
		t.Fatalf("blockTempFile: %v", err)
	}
	f.Close()
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	if err := writeLines(f.Name(), lines); err != nil {
		t.Fatalf("blockTempFile write: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// ── doShow -raw ───────────────────────────────────────────────────────────────

func TestDoShow_Raw_NoPrefix(t *testing.T) {
	path := blockTempFile(t, "line one\nline two\nline three\n")
	lines, _ := readLines(path)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doShow(lines, 1, 3, true)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	rows := strings.Split(out, "\n")

	if len(rows) != 3 {
		t.Fatalf("raw show: expected 3 lines, got %d: %v", len(rows), rows)
	}
	// Raw output must NOT contain "N |" prefix
	for _, row := range rows {
		if strings.Contains(row, " | ") {
			t.Errorf("raw show: line number prefix found in %q", row)
		}
	}
	if rows[0] != "line one" || rows[1] != "line two" || rows[2] != "line three" {
		t.Errorf("raw show: content wrong: %v", rows)
	}
}

func TestDoShow_Raw_SubRange(t *testing.T) {
	path := blockTempFile(t, "a\nb\nc\nd\ne\n")
	lines, _ := readLines(path)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doShow(lines, 2, 4, true)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	rows := strings.Split(out, "\n")

	if len(rows) != 3 || rows[0] != "b" || rows[1] != "c" || rows[2] != "d" {
		t.Errorf("raw show subrange: got %v", rows)
	}
}

func TestDoShow_NonRaw_HasPrefix(t *testing.T) {
	path := blockTempFile(t, "hello\n")
	lines, _ := readLines(path)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doShow(lines, 1, 1, false)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 256)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")

	if !strings.Contains(out, " | ") {
		t.Errorf("non-raw show must contain '| ' prefix, got: %q", out)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("non-raw show must contain content, got: %q", out)
	}
}

// ── show -block -lang go ──────────────────────────────────────────────────────

func TestDoShow_Block_Go_ResolvesRange(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	// Resolve Hello block — should be lines 5-7 in goSource
	start, end, err := resolveBlock(lines, "go", "Hello")
	if err != nil {
		t.Fatalf("resolveBlock Hello: %v", err)
	}
	if start < 1 || end < start {
		t.Fatalf("resolveBlock Hello: invalid range %d-%d", start, end)
	}

	// Show with raw — output must contain the function signature
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doShow(lines, start, end, true)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	if !strings.Contains(out, "func Hello") {
		t.Errorf("show block Hello: func Hello not in output: %q", out)
	}
	// Must NOT contain Goodbye
	if strings.Contains(out, "func Goodbye") {
		t.Errorf("show block Hello: leaked into Goodbye: %q", out)
	}
}

func TestDoShow_Block_Go_Raw_NoPrefix(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	start, end, _ := resolveBlock(lines, "go", "Hello")

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doShow(lines, start, end, true)
	w.Close()
	os.Stdout = old

	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	out := string(buf[:n])

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, " | ") {
			t.Errorf("raw block show: prefix leaked into output line: %q", line)
		}
	}
}

// ── replace -block -lang go ───────────────────────────────────────────────────

func TestReplace_Block_Go_ReplacesFunction(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	newLines := strings.Split(goSourceReplacement, "\n")
	// Remove empty last element if present
	if len(newLines) > 0 && newLines[len(newLines)-1] == "" {
		newLines = newLines[:len(newLines)-1]
	}

	// Resolve Hello block range
	start, end, err := resolveBlock(lines, "go", "Hello")
	if err != nil {
		t.Fatalf("resolveBlock: %v", err)
	}

	doReplace(lines, path, start, end, newLines)

	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	// New function body should be present
	if !strings.Contains(full, "Greetings") {
		t.Errorf("replace block: new content not found. file:\n%s", full)
	}
	// Old function body should be gone
	if strings.Contains(full, `"Hello, %s!"`) {
		t.Errorf("replace block: old content still present. file:\n%s", full)
	}
	// Other functions must survive
	if !strings.Contains(full, "func Goodbye") {
		t.Errorf("replace block: Goodbye function removed unexpectedly. file:\n%s", full)
	}
}

func TestReplace_Block_Go_ShorterReplacement(t *testing.T) {
	// Replace a 3-line function with a 1-line stub
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	start, end, _ := resolveBlock(lines, "go", "Hello")
	originalLen := len(lines)
	blockLen := end - start + 1

	stub := []string{`func Hello(name string) string { return "hi" }`}
	doReplace(lines, path, start, end, stub)

	got, _ := readLines(path)
	expectedLen := originalLen - blockLen + 1
	if len(got) != expectedLen {
		t.Errorf("replace block shorter: expected %d lines, got %d", expectedLen, len(got))
	}
}

func TestReplace_Block_Go_PreservesOtherFunctions(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	start, end, _ := resolveBlock(lines, "go", "Goodbye")
	newLines := []string{`func Goodbye(name string) string { return "bye" }`}
	doReplace(lines, path, start, end, newLines)

	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	if !strings.Contains(full, "func Hello") {
		t.Errorf("replace block: Hello was lost after replacing Goodbye")
	}
	if !strings.Contains(full, "func helper") {
		t.Errorf("replace block: helper was lost after replacing Goodbye")
	}
}

// ── insertbefore -block -lang go ─────────────────────────────────────────────

func TestInsertBefore_Block_Go_InsertsBeforeFunction(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	newLines := strings.Split(strings.TrimRight(goSourceNewFunc, "\n"), "\n")

	blockStart, _, err := resolveBlock(lines, "go", "Goodbye")
	if err != nil {
		t.Fatalf("resolveBlock Goodbye: %v", err)
	}

	// insertbefore: doInsert at blockStart-1
	doInsert(lines, path, blockStart-1, newLines)

	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	// Farewell must appear before Goodbye
	farewellIdx := strings.Index(full, "func Farewell")
	goodbyeIdx := strings.Index(full, "func Goodbye")
	if farewellIdx == -1 {
		t.Fatal("insertbefore block: Farewell not found")
	}
	if goodbyeIdx == -1 {
		t.Fatal("insertbefore block: Goodbye not found")
	}
	if farewellIdx >= goodbyeIdx {
		t.Errorf("insertbefore block: Farewell (%d) should appear before Goodbye (%d)", farewellIdx, goodbyeIdx)
	}
}

func TestInsertBefore_Block_Go_HelloSurvives(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	newLines := []string{"// new comment line"}
	blockStart, _, _ := resolveBlock(lines, "go", "Goodbye")
	doInsert(lines, path, blockStart-1, newLines)

	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	if !strings.Contains(full, "func Hello") {
		t.Error("insertbefore block: Hello function lost")
	}
	if !strings.Contains(full, "func helper") {
		t.Error("insertbefore block: helper function lost")
	}
}

// ── insertafter -block -lang go ──────────────────────────────────────────────

func TestInsertAfter_Block_Go_InsertsAfterFunction(t *testing.T) {
	path := blockTempFile(t, goSource)
	lines, _ := readLines(path)

	newLines := strings.Split(strings.TrimRight(goSourceNewFunc, "\n"), "\n")

	_, blockEnd, err := resolveBlock(lines, "go", "Hello")
	if err != nil {
		t.Fatalf("resolveBlock Hello: %v", err)
	}

	// insertafter: doInsert at blockEnd
	doInsert(lines, path, blockEnd, newLines)

	got, _ := readLines(path)
	full := strings.Join(got, "\n")

	// Farewell must appear after Hello and before Goodbye
	helloIdx := strings.Index(full, "func Hello")
	farewellIdx := strings.Index(full, "func Farewell")
	goodbyeIdx := strings.Index(full, "func Goodbye")

	if farewellIdx == -1 {
		t.Fatal("insertafter block: Farewell not found")
	}
	if farewellIdx <= helloIdx {
		t.Errorf("Farewell must be after Hello: hello=%d farewell=%d", helloIdx, farewellIdx)
	}
	if farewellIdx >= goodbyeIdx {
		t.Errorf("Farewell must be before Goodbye: farewell=%d goodbye=%d", farewellIdx, goodbyeIdx)
	}
}

// ── MCP block tests ───────────────────────────────────────────────────────────

func TestMCPExec_Show_Block(t *testing.T) {
	path := mcpTempFile(t, strings.Join(strings.Split(strings.TrimRight(goSource, "\n"), "\n"), "\n")+"\n")
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":  path,
		"block": "Hello",
		"lang":  "go",
	})
	if r.IsError {
		t.Fatalf("show block error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "func Hello") {
		t.Errorf("show block: func Hello not in output: %q", text)
	}
	if strings.Contains(text, "func Goodbye") {
		t.Errorf("show block: leaked into Goodbye: %q", text)
	}
}

func TestMCPExec_Show_Block_Raw(t *testing.T) {
	path := mcpTempFile(t, strings.Join(strings.Split(strings.TrimRight(goSource, "\n"), "\n"), "\n")+"\n")
	defer os.Remove(path)

	r := mcpExecTool("fedit_show", map[string]any{
		"file":  path,
		"block": "Hello",
		"lang":  "go",
		"raw":   true,
	})
	if r.IsError {
		t.Fatalf("show block raw error: %s", mcpText(r))
	}
	text := mcpText(r)
	// Raw: no "N |" prefix
	for _, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		if strings.Contains(line, " | ") {
			t.Errorf("show block raw: prefix found in line %q", line)
		}
	}
}

func TestMCPExec_Replace_Block(t *testing.T) {
	path := mcpTempFile(t, strings.Join(strings.Split(strings.TrimRight(goSource, "\n"), "\n"), "\n")+"\n")
	defer os.Remove(path)

	r := mcpExecTool("fedit_replace", map[string]any{
		"file":  path,
		"block": "Hello",
		"lang":  "go",
		"text":  `func Hello(name string) string { return "Greetings, " + name }`,
	})
	if r.IsError {
		t.Fatalf("replace block error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")
	if !strings.Contains(full, "Greetings") {
		t.Errorf("replace block MCP: new content not found:\n%s", full)
	}
	if strings.Contains(full, `"Hello, %s!"`) {
		t.Errorf("replace block MCP: old content still present:\n%s", full)
	}
	if !strings.Contains(full, "func Goodbye") {
		t.Errorf("replace block MCP: Goodbye removed unexpectedly")
	}
}

func TestMCPExec_InsertBefore_Block(t *testing.T) {
	path := mcpTempFile(t, strings.Join(strings.Split(strings.TrimRight(goSource, "\n"), "\n"), "\n")+"\n")
	defer os.Remove(path)

	r := mcpExecTool("fedit_insertbefore", map[string]any{
		"file":  path,
		"block": "Goodbye",
		"lang":  "go",
		"text":  "func Farewell(name string) string { return name }",
	})
	if r.IsError {
		t.Fatalf("insertbefore block error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")
	farewellIdx := strings.Index(full, "func Farewell")
	goodbyeIdx := strings.Index(full, "func Goodbye")
	if farewellIdx == -1 {
		t.Fatal("insertbefore block MCP: Farewell not found")
	}
	if farewellIdx >= goodbyeIdx {
		t.Errorf("insertbefore block MCP: Farewell must be before Goodbye")
	}
}

func TestMCPExec_InsertAfter_Block(t *testing.T) {
	path := mcpTempFile(t, strings.Join(strings.Split(strings.TrimRight(goSource, "\n"), "\n"), "\n")+"\n")
	defer os.Remove(path)

	r := mcpExecTool("fedit_insertafter", map[string]any{
		"file":  path,
		"block": "Hello",
		"lang":  "go",
		"text":  "func Farewell(name string) string { return name }",
	})
	if r.IsError {
		t.Fatalf("insertafter block error: %s", mcpText(r))
	}
	got, _ := readLines(path)
	full := strings.Join(got, "\n")
	helloIdx := strings.Index(full, "func Hello")
	farewellIdx := strings.Index(full, "func Farewell")
	goodbyeIdx := strings.Index(full, "func Goodbye")
	if farewellIdx == -1 {
		t.Fatal("insertafter block MCP: Farewell not found")
	}
	if farewellIdx <= helloIdx {
		t.Errorf("Farewell must be after Hello")
	}
	if farewellIdx >= goodbyeIdx {
		t.Errorf("Farewell must be before Goodbye")
	}
}

func TestMCPToolDefs_ShowHasBlockRawParams(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_show" {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			t.Fatalf("fedit_show: invalid schema: %v", err)
		}
		props, _ := schema["properties"].(map[string]any)
		for _, p := range []string{"block", "lang", "raw"} {
			if _, ok := props[p]; !ok {
				t.Errorf("fedit_show: missing property %q", p)
			}
		}
		return
	}
	t.Error("fedit_show not found")
}

func TestMCPToolDefs_ReplaceHasBlockParam(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_replace" {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			t.Fatalf("fedit_replace: invalid schema: %v", err)
		}
		props, _ := schema["properties"].(map[string]any)
		for _, p := range []string{"block", "lang"} {
			if _, ok := props[p]; !ok {
				t.Errorf("fedit_replace: missing property %q", p)
			}
		}
		return
	}
	t.Error("fedit_replace not found")
}

func TestMCPToolDefs_InsertBeforeAfterHaveBlockParam(t *testing.T) {
	for _, name := range []string{"fedit_insertbefore", "fedit_insertafter"} {
		found := false
		for _, def := range mcpToolDefs() {
			if def.Name != name {
				continue
			}
			found = true
			var schema map[string]any
			if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
				t.Fatalf("%s: invalid schema: %v", name, err)
			}
			props, _ := schema["properties"].(map[string]any)
			for _, p := range []string{"block", "lang"} {
				if _, ok := props[p]; !ok {
					t.Errorf("%s: missing property %q", name, p)
				}
			}
		}
		if !found {
			t.Errorf("%s not found in tool defs", name)
		}
	}
}
