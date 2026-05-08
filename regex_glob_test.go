package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── execReplaceAllRegex ───────────────────────────────────────

func TestExecReplaceAllRegex_BasicLiteral(t *testing.T) {
	lines := []string{"hello world", "foo bar", "hello again"}
	result, count, err := execReplaceAllRegex(lines, "hello", "hi")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if result[0] != "hi world" {
		t.Errorf("result[0] = %q, want 'hi world'", result[0])
	}
	if result[2] != "hi again" {
		t.Errorf("result[2] = %q, want 'hi again'", result[2])
	}
	// Unchanged line preserved
	if result[1] != "foo bar" {
		t.Errorf("result[1] = %q, want 'foo bar'", result[1])
	}
}

func TestExecReplaceAllRegex_CaptureGroup(t *testing.T) {
	lines := []string{`version = "1.2.3"`, `other = "nope"`}
	result, count, err := execReplaceAllRegex(lines, `"(\d+\.\d+\.\d+)"`, `"[$1]"`)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if result[0] != `version = "[1.2.3]"` {
		t.Errorf("result[0] = %q, want 'version = \"[1.2.3]\"'", result[0])
	}
}

func TestExecReplaceAllRegex_MultipleCaptureGroups(t *testing.T) {
	lines := []string{"John Smith", "Jane Doe"}
	// Swap first and last name
	result, count, err := execReplaceAllRegex(lines, `^(\w+) (\w+)$`, "$2, $1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if result[0] != "Smith, John" {
		t.Errorf("result[0] = %q, want 'Smith, John'", result[0])
	}
	if result[1] != "Doe, Jane" {
		t.Errorf("result[1] = %q, want 'Doe, Jane'", result[1])
	}
}

func TestExecReplaceAllRegex_GlobalWithinLine(t *testing.T) {
	// All occurrences on the same line replaced
	lines := []string{"foo foo foo"}
	result, count, err := execReplaceAllRegex(lines, `foo`, "bar")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1 (1 line changed)", count)
	}
	if result[0] != "bar bar bar" {
		t.Errorf("result[0] = %q, want 'bar bar bar'", result[0])
	}
}

func TestExecReplaceAllRegex_NoMatch(t *testing.T) {
	lines := []string{"hello world"}
	_, count, err := execReplaceAllRegex(lines, `\d+`, "NUM")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestExecReplaceAllRegex_InvalidRegex(t *testing.T) {
	lines := []string{"hello"}
	_, _, err := execReplaceAllRegex(lines, `[invalid`, "x")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
	if !strings.Contains(err.Error(), "invalid regex") {
		t.Errorf("error = %q, want 'invalid regex'", err.Error())
	}
}

func TestExecReplaceAllRegex_EmptyInput(t *testing.T) {
	_, count, err := execReplaceAllRegex([]string{}, `\w+`, "x")
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 for empty input", count)
	}
}

func TestExecReplaceAllRegex_OriginalUnchanged(t *testing.T) {
	// Lines without match must be byte-identical in result
	lines := []string{"AAA", "has_number_123", "BBB"}
	result, _, err := execReplaceAllRegex(lines, `\d+`, "NUM")
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != "AAA" || result[2] != "BBB" {
		t.Errorf("unchanged lines modified: %q %q", result[0], result[2])
	}
	if result[1] != "has_number_NUM" {
		t.Errorf("result[1] = %q", result[1])
	}
}

func TestExecReplaceAllRegex_CountIsLinesNotOccurrences(t *testing.T) {
	// count = number of LINES changed, not total occurrences
	lines := []string{"a1b2c3", "d4e5f6", "nope"}
	_, count, err := execReplaceAllRegex(lines, `\d`, "N")
	if err != nil {
		t.Fatal(err)
	}
	// 2 lines changed (first two), even though 6 digits replaced
	if count != 2 {
		t.Errorf("count = %d, want 2 (lines changed)", count)
	}
}

func TestExecReplaceAllRegex_AnchoredPattern(t *testing.T) {
	lines := []string{"func alpha()", "  func nested()", "type Beta struct"}
	// Only top-level funcs (no leading space)
	result, count, err := execReplaceAllRegex(lines, `^func (\w+)`, "func NEW_$1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if result[0] != "func NEW_alpha()" {
		t.Errorf("result[0] = %q", result[0])
	}
	if result[1] != "  func nested()" {
		t.Errorf("nested line should be unchanged: %q", result[1])
	}
}

func TestExecReplaceAllRegex_LargeFile(t *testing.T) {
	lines := make([]string, 1000)
	for i := range lines {
		lines[i] = fmt.Sprintf("line_%04d_value", i)
	}
	result, count, err := execReplaceAllRegex(lines, `line_(\d{4})_value`, "item_$1")
	if err != nil {
		t.Fatal(err)
	}
	if count != 1000 {
		t.Errorf("count = %d, want 1000", count)
	}
	if result[0] != "item_0000" {
		t.Errorf("result[0] = %q, want 'item_0000'", result[0])
	}
	if result[999] != "item_0999" {
		t.Errorf("result[999] = %q, want 'item_0999'", result[999])
	}
}

// ── doReplaceAllRegex integration ────────────────────────────

func TestDoReplaceAllRegex_FileRoundTrip(t *testing.T) {
	lines := []string{`v1.0.0`, `still v1.0.0 here`, `unrelated`}
	path := writeTestFile(t, lines)
	fileLines, _ := readLines(path)
	result, _, err := execReplaceAllRegex(fileLines, `v(\d+\.\d+\.\d+)`, "v[$1]")
	if err != nil {
		t.Fatal(err)
	}
	if err := writeLines(path, result); err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	if got[0] != "v[1.0.0]" {
		t.Errorf("got[0] = %q, want 'v[1.0.0]'", got[0])
	}
	if got[2] != "unrelated" {
		t.Errorf("got[2] = %q, want 'unrelated'", got[2])
	}
}

// ── glob multi-file tests ─────────────────────────────────────

// tempDir creates a temp directory with named files and returns (dir, cleanup).
func makeTempDir(t *testing.T, files map[string][]string) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "fedit_glob_*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	for name, lines := range files {
		path := filepath.Join(dir, name)
		if err := writeLines(path, lines); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestDoReplaceAllGlob_MultipleFiles(t *testing.T) {
	dir := makeTempDir(t, map[string][]string{
		"a.go": {"package main", "OldName here"},
		"b.go": {"package foo", "OldName again"},
		"c.go": {"package bar", "nothing to replace"},
	})
	glob := filepath.Join(dir, "*.go")
	doReplaceAllGlob(glob, "OldName", "NewName")

	for _, tc := range []struct{ file, line0, line1 string }{
		{"a.go", "package main", "NewName here"},
		{"b.go", "package foo", "NewName again"},
		{"c.go", "package bar", "nothing to replace"},
	} {
		got, _ := readLines(filepath.Join(dir, tc.file))
		if got[0] != tc.line0 {
			t.Errorf("%s line 0 = %q, want %q", tc.file, got[0], tc.line0)
		}
		if got[1] != tc.line1 {
			t.Errorf("%s line 1 = %q, want %q", tc.file, got[1], tc.line1)
		}
	}
}

func TestDoReplaceAllGlob_OnlyMatchingExtension(t *testing.T) {
	dir := makeTempDir(t, map[string][]string{
		"main.go":  {"OldName in go"},
		"main.py":  {"OldName in py"},
		"notes.md": {"OldName in md"},
	})
	doReplaceAllGlob(filepath.Join(dir, "*.go"), "OldName", "NewName")

	goLines, _ := readLines(filepath.Join(dir, "main.go"))
	pyLines, _ := readLines(filepath.Join(dir, "main.py"))
	if goLines[0] != "NewName in go" {
		t.Errorf("go file not updated: %q", goLines[0])
	}
	if pyLines[0] != "OldName in py" {
		t.Errorf("py file should be unchanged: %q", pyLines[0])
	}
}

func TestDoReplaceAllRegexGlob_CaptureGroups(t *testing.T) {
	dir := makeTempDir(t, map[string][]string{
		"a.go": {`const Version = "1.0.0"`},
		"b.go": {`const Version = "2.0.0"`},
	})
	glob := filepath.Join(dir, "*.go")
	doReplaceAllRegexGlob(glob, `"(\d+\.\d+\.\d+)"`, `"v$1"`)

	for _, tc := range []struct{ file, want string }{
		{"a.go", `const Version = "v1.0.0"`},
		{"b.go", `const Version = "v2.0.0"`},
	} {
		got, _ := readLines(filepath.Join(dir, tc.file))
		if got[0] != tc.want {
			t.Errorf("%s: got %q, want %q", tc.file, got[0], tc.want)
		}
	}
}

func TestDoReplaceAllRegexGlob_InvalidRegex(t *testing.T) {
	// Invalid regex should call os.Exit — test via execReplaceAllRegex instead
	_, _, err := execReplaceAllRegex([]string{"test"}, `[bad`, "x")
	if err == nil {
		t.Error("expected error for invalid regex")
	}
}

func TestDoReplaceAllGlob_SingleFile(t *testing.T) {
	dir := makeTempDir(t, map[string][]string{
		"only.go": {"line one", "target line", "line three"},
	})
	doReplaceAllGlob(filepath.Join(dir, "*.go"), "target", "replaced")
	got, _ := readLines(filepath.Join(dir, "only.go"))
	if got[1] != "replaced line" {
		t.Errorf("got[1] = %q, want 'replaced line'", got[1])
	}
	if got[0] != "line one" || got[2] != "line three" {
		t.Error("unrelated lines should be unchanged")
	}
}

func TestExecReplaceAllRegex_VersionBumpScenario(t *testing.T) {
	// Real-world: bump version across CHANGELOG
	lines := []string{
		"## v1.2.0 Release",
		"See v1.2.0 for details",
		"Previous: v1.1.0",
	}
	result, count, err := execReplaceAllRegex(lines, `v1\.2\.0`, "v1.3.0")
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if result[0] != "## v1.3.0 Release" {
		t.Errorf("result[0] = %q", result[0])
	}
	if result[2] != "Previous: v1.1.0" {
		t.Errorf("previous version should be unchanged: %q", result[2])
	}
}

func TestExecReplaceAllRegex_FunctionRenameScenario(t *testing.T) {
	// Real-world: rename a Go function across a file
	lines := []string{
		"func OldHandler(w http.ResponseWriter, r *http.Request) {",
		"    OldHandler(w, r) // recursive call",
		"    return",
		"}",
		"// see OldHandler for details",
	}
	result, count, err := execReplaceAllRegex(lines, `OldHandler`, "NewHandler")
	if err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if !strings.Contains(result[0], "NewHandler") {
		t.Errorf("result[0] = %q, should contain NewHandler", result[0])
	}
}
