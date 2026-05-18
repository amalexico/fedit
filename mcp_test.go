package main

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// schemaBytes extracts the raw JSON bytes from a tool's InputSchema.
// InputSchema is typed as any but always holds a json.RawMessage.
func schemaBytes(def mcpToolDef) []byte {
	if raw, ok := def.InputSchema.(json.RawMessage); ok {
		return []byte(raw)
	}
	b, _ := json.Marshal(def.InputSchema)
	return b
}

// ── Schema validation ─────────────────────────────────────────────────────────

func TestMCPToolDefs_Count(t *testing.T) {
	defs := mcpToolDefs()
	if len(defs) != 13 {
		t.Errorf("expected 13 tools, got %d", len(defs))
	}
}

func TestMCPToolDefs_AllSchemasValidJSON(t *testing.T) {
	for _, def := range mcpToolDefs() {
		var schema any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			t.Errorf("tool %q: invalid JSON schema: %v", def.Name, err)
		}
	}
}

func TestMCPToolDefs_AllHaveNames(t *testing.T) {
	seen := map[string]bool{}
	for _, def := range mcpToolDefs() {
		if def.Name == "" {
			t.Error("tool with empty name found")
		}
		if seen[def.Name] {
			t.Errorf("duplicate tool name: %q", def.Name)
		}
		seen[def.Name] = true
	}
}

func TestMCPToolDefs_AllHaveDescriptions(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Description == "" {
			t.Errorf("tool %q has empty description", def.Name)
		}
	}
}

func TestMCPToolDefs_RequiredFieldsExistInProperties(t *testing.T) {
	for _, def := range mcpToolDefs() {
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			continue // caught by AllSchemasValidJSON
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Errorf("tool %q: missing properties object", def.Name)
			continue
		}
		req, _ := schema["required"].([]any)
		for _, r := range req {
			field, _ := r.(string)
			if field == "" {
				t.Errorf("tool %q: non-string entry in required[]", def.Name)
				continue
			}
			if _, exists := props[field]; !exists {
				t.Errorf("tool %q: required field %q not in properties", def.Name, field)
			}
		}
	}
}

// ── fedit_find schema regression ─────────────────────────────────────────────
// Regression for the v1.5.0 bug: stream was embedded inside required[]
// instead of properties{}, producing invalid JSON.

func TestMCPToolDefs_FindStreamInProperties(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_find" {
			continue
		}
		var schema map[string]any
		if err := json.Unmarshal(schemaBytes(def), &schema); err != nil {
			t.Fatalf("fedit_find: invalid JSON schema: %v", err)
		}
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatal("fedit_find: missing properties object")
		}
		if _, ok := props["stream"]; !ok {
			t.Error("fedit_find: stream not found in properties{}")
		}
		req, _ := schema["required"].([]any)
		for _, r := range req {
			if r == "stream" {
				t.Error("fedit_find: stream must not appear in required[]")
			}
		}
		return
	}
	t.Error("fedit_find not found in tool definitions")
}

func TestMCPToolDefs_FindRequiredFields(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_find" {
			continue
		}
		var schema map[string]any
		json.Unmarshal(schemaBytes(def), &schema)
		req, _ := schema["required"].([]any)
		reqStrs := make([]string, len(req))
		for i, r := range req {
			reqStrs[i], _ = r.(string)
		}
		for _, want := range []string{"file", "match"} {
			found := false
			for _, r := range reqStrs {
				if r == want {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("fedit_find: expected %q in required[], got %v", want, reqStrs)
			}
		}
		return
	}
	t.Error("fedit_find not found")
}

// ── fedit_map IaC language coverage ──────────────────────────────────────────

func TestMCPToolDefs_MapDescriptionMentionsIaC(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_map" {
			continue
		}
		for _, lang := range []string{"hcl", "terraform", "nix"} {
			if !strings.Contains(def.Description, lang) {
				t.Errorf("fedit_map description missing %q", lang)
			}
		}
		return
	}
	t.Error("fedit_map not found")
}

func TestMCPToolDefs_MapLangSchemaListsIaC(t *testing.T) {
	for _, def := range mcpToolDefs() {
		if def.Name != "fedit_map" {
			continue
		}
		var schema map[string]any
		json.Unmarshal(schemaBytes(def), &schema)
		props := schema["properties"].(map[string]any)
		langProp := props["lang"].(map[string]any)
		langDesc, _ := langProp["description"].(string)
		for _, lang := range []string{"hcl", "tf", "terraform", "nix"} {
			if !strings.Contains(langDesc, lang) {
				t.Errorf("fedit_map lang schema description missing %q", lang)
			}
		}
		return
	}
	t.Error("fedit_map not found")
}

// ── mcpExecTool integration helpers ──────────────────────────────────────────

func mcpTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "fedit_mcp_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func mcpText(r mcpCallResult) string {
	if len(r.Content) == 0 {
		return ""
	}
	return r.Content[0].Text
}

// ── mcpExecTool: error handling ───────────────────────────────────────────────

func TestMCPExec_UnknownTool(t *testing.T) {
	result := mcpExecTool("fedit_nonexistent", map[string]any{})
	if !result.IsError {
		t.Error("expected IsError=true for unknown tool")
	}
}

func TestMCPExec_MissingFile(t *testing.T) {
	result := mcpExecTool("fedit_show", map[string]any{
		"file": "/nonexistent/path/file_does_not_exist.txt",
	})
	if !result.IsError {
		t.Error("expected IsError=true for missing file")
	}
}

// ── mcpExecTool: core ops ─────────────────────────────────────────────────────

func TestMCPExec_Show(t *testing.T) {
	path := mcpTempFile(t, "alpha\nbeta\ngamma\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_show", map[string]any{"file": path})
	if r.IsError {
		t.Fatalf("show error: %s", mcpText(r))
	}
	for _, want := range []string{"alpha", "beta", "gamma"} {
		if !strings.Contains(mcpText(r), want) {
			t.Errorf("show missing %q", want)
		}
	}
}

func TestMCPExec_Insert(t *testing.T) {
	path := mcpTempFile(t, "line1\nline3\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_insert", map[string]any{
		"file": path,
		"line": float64(1),
		"text": "line2",
	})
	if r.IsError {
		t.Fatalf("insert error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if len(lines) != 3 || lines[1] != "line2" {
		t.Errorf("after insert: %v", lines)
	}
}

func TestMCPExec_Delete(t *testing.T) {
	path := mcpTempFile(t, "keep\ndelete_me\nkeep\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_delete", map[string]any{
		"file": path,
		"line": float64(2),
	})
	if r.IsError {
		t.Fatalf("delete error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if len(lines) != 2 {
		t.Errorf("expected 2 lines after delete, got %d: %v", len(lines), lines)
	}
}

func TestMCPExec_Replace(t *testing.T) {
	path := mcpTempFile(t, "old\nstuff\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replace", map[string]any{
		"file": path,
		"line": float64(1),
		"end":  float64(1),
		"text": "new",
	})
	if r.IsError {
		t.Fatalf("replace error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if lines[0] != "new" {
		t.Errorf("replace: line[0] = %q, want 'new'", lines[0])
	}
}

func TestMCPExec_Write(t *testing.T) {
	path := mcpTempFile(t, "old content\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_write", map[string]any{
		"file": path,
		"text": "brand new\nsecond line",
	})
	if r.IsError {
		t.Fatalf("write error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if lines[0] != "brand new" {
		t.Errorf("write: line[0] = %q", lines[0])
	}
}

// ── mcpExecTool: find + stream ────────────────────────────────────────────────

func TestMCPExec_Find(t *testing.T) {
	path := mcpTempFile(t, "hello world\nfoo bar\nhello again\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":  path,
		"match": "hello",
	})
	if r.IsError {
		t.Fatalf("find error: %s", mcpText(r))
	}
	if !strings.Contains(mcpText(r), "hello") {
		t.Errorf("find result missing match: %q", mcpText(r))
	}
}

func TestMCPExec_Find_Stream(t *testing.T) {
	path := mcpTempFile(t, "hello world\nfoo bar\nhello again\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":   path,
		"match":  "hello",
		"stream": true,
	})
	if r.IsError {
		t.Fatalf("find stream error: %s", mcpText(r))
	}
	text := mcpText(r)
	if !strings.Contains(text, "hello") {
		t.Errorf("find stream result missing match: %q", text)
	}
}

func TestMCPExec_Find_NoMatch(t *testing.T) {
	path := mcpTempFile(t, "alpha\nbeta\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_find", map[string]any{
		"file":  path,
		"match": "zzznothere",
	})
	// fedit_find returns IsError=true when no lines match — correct behavior
	if !r.IsError {
		t.Errorf("find with no match should not be an error: %s", mcpText(r))
	}
}

// ── mcpExecTool: replaceall + stream ─────────────────────────────────────────

func TestMCPExec_ReplaceAll(t *testing.T) {
	path := mcpTempFile(t, "foo bar\nfoo baz\nqux\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":  path,
		"match": "foo",
		"text":  "replaced",
	})
	if r.IsError {
		t.Fatalf("replaceall error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	for _, line := range lines {
		if strings.Contains(line, "foo") {
			t.Errorf("foo not replaced: %q", line)
		}
	}
}

func TestMCPExec_ReplaceAll_Stream(t *testing.T) {
	path := mcpTempFile(t, "hello world\nhello again\ngoodbye\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_replaceall", map[string]any{
		"file":   path,
		"match":  "hello",
		"text":   "hi",
		"stream": true,
	})
	if r.IsError {
		t.Fatalf("replaceall stream error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	for _, line := range lines {
		if strings.Contains(line, "hello") {
			t.Errorf("hello not replaced in stream mode: %q", line)
		}
	}
}

// ── mcpExecTool: fields ───────────────────────────────────────────────────────

func TestMCPExec_Fields_TSV(t *testing.T) {
	path := mcpTempFile(t, "a\tb\tc\n1\t2\t3\nx\ty\tz\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_fields", map[string]any{
		"file": path,
		"col":  float64(2),
	})
	if r.IsError {
		t.Fatalf("fields tsv error: %s", mcpText(r))
	}
	text := mcpText(r)
	for _, want := range []string{"b", "2", "y"} {
		if !strings.Contains(text, want) {
			t.Errorf("fields tsv missing %q: %s", want, text)
		}
	}
}

func TestMCPExec_Fields_CSV(t *testing.T) {
	path := mcpTempFile(t, "a,b,c\n1,2,3\nx,y,z\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_fields", map[string]any{
		"file":  path,
		"col":   float64(3),
		"delim": ",",
	})
	if r.IsError {
		t.Fatalf("fields csv error: %s", mcpText(r))
	}
	text := mcpText(r)
	for _, want := range []string{"c", "3", "z"} {
		if !strings.Contains(text, want) {
			t.Errorf("fields csv missing %q: %s", want, text)
		}
	}
}

func TestMCPExec_Fields_InvalidCol(t *testing.T) {
	path := mcpTempFile(t, "a,b,c\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_fields", map[string]any{
		"file": path,
		"col":  float64(0),
	})
	if !r.IsError {
		t.Error("expected error for col=0")
	}
}

// ── mcpExecTool: fedit_map with HCL ──────────────────────────────────────────

// ── mcpExecTool: HCL block move/copy ─────────────────────────────────────────

func TestMCPExec_Move_HCLBlock(t *testing.T) {
	src := "resource \"aws_instance\" \"web\" {\n  ami = \"ami-123\"\n}\n\nresource \"aws_s3_bucket\" \"data\" {\n  bucket = \"my-data\"\n}\n"
	path := mcpTempFile(t, src)
	defer os.Remove(path)
	r := mcpExecTool("fedit_move", map[string]any{
		"file":        path,
		"block":       "aws_s3_bucket",
		"beforeblock": "aws_instance",
		"lang":        "hcl",
	})
	if r.IsError {
		t.Fatalf("HCL block move error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if !strings.Contains(lines[0], "aws_s3_bucket") {
		t.Errorf("after move, first block should be aws_s3_bucket, got: %q", lines[0])
	}
}

func TestMCPExec_Copy_HCLBlock(t *testing.T) {
	src := "resource \"aws_instance\" \"web\" {\n  ami = \"ami-123\"\n}\n"
	path := mcpTempFile(t, src)
	defer os.Remove(path)
	r := mcpExecTool("fedit_copy", map[string]any{
		"file":       path,
		"block":      "aws_instance",
		"afterblock": "aws_instance",
		"lang":       "hcl",
	})
	if r.IsError {
		t.Fatalf("HCL block copy error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	count := 0
	for _, l := range lines {
		if strings.Contains(l, "aws_instance") {
			count++
		}
	}
	if count < 2 {
		t.Errorf("expected 2 occurrences of aws_instance after copy, got %d", count)
	}
}

// ── mcpExecTool: Nix block move ───────────────────────────────────────────────

func TestMCPExec_Move_NixBlock(t *testing.T) {
	src := "programs.git = {\n  enable = true;\n};\n\nprograms.ssh = {\n  enable = true;\n};\n"
	path := mcpTempFile(t, src)
	defer os.Remove(path)
	r := mcpExecTool("fedit_move", map[string]any{
		"file":        path,
		"block":       "programs.git",
		"afterblock":  "programs.ssh",
		"lang":        "nix",
	})
	if r.IsError {
		t.Fatalf("Nix block move error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	// The blank separator line may land at index 0 after the move;
	// find the first non-empty line to check block order.
	firstNonEmpty := ""
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			firstNonEmpty = l
			break
		}
	}
	if !strings.Contains(firstNonEmpty, "programs.ssh") {
		t.Errorf("after move, first non-empty line should be programs.ssh, got: %q", firstNonEmpty)
	}
}

// ── mcpExecTool: insertafter / insertbefore ───────────────────────────────────

func TestMCPExec_InsertAfter(t *testing.T) {
	path := mcpTempFile(t, "line1\nanchor\nline3\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_insertafter", map[string]any{
		"file":  path,
		"match": "anchor",
		"text":  "inserted",
	})
	if r.IsError {
		t.Fatalf("insertafter error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if len(lines) != 4 || lines[2] != "inserted" {
		t.Errorf("insertafter: %v", lines)
	}
}

func TestMCPExec_InsertBefore(t *testing.T) {
	path := mcpTempFile(t, "line1\nanchor\nline3\n")
	defer os.Remove(path)
	r := mcpExecTool("fedit_insertbefore", map[string]any{
		"file":  path,
		"match": "anchor",
		"text":  "before_anchor",
	})
	if r.IsError {
		t.Fatalf("insertbefore error: %s", mcpText(r))
	}
	lines, _ := readLines(path)
	if len(lines) != 4 || lines[1] != "before_anchor" {
		t.Errorf("insertbefore: %v", lines)
	}
}
