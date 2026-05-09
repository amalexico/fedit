package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── execStreamOp ─────────────────────────────────────────────

func TestExecStreamOp_PassThrough(t *testing.T) {
	lines := testLines(10)
	path := writeTestFile(t, lines)
	linesRead, linesWritten, err := execStreamOp(path, func(_ int, line string) []string {
		return []string{line}
	})
	if err != nil {
		t.Fatal(err)
	}
	if linesRead != 10 || linesWritten != 10 {
		t.Errorf("read=%d written=%d, want 10/10", linesRead, linesWritten)
	}
	got := readTestFile(t, path)
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want)
		}
	}
}

func TestExecStreamOp_Replace(t *testing.T) {
	lines := []string{"hello world", "foo bar", "hello again"}
	path := writeTestFile(t, lines)
	_, _, err := execStreamOp(path, func(_ int, line string) []string {
		return []string{strings.ReplaceAll(line, "hello", "hi")}
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	if got[0] != "hi world" || got[2] != "hi again" {
		t.Errorf("replace failed: %v", got)
	}
	if got[1] != "foo bar" {
		t.Errorf("unchanged line modified: %q", got[1])
	}
}

func TestExecStreamOp_Delete(t *testing.T) {
	lines := []string{"keep", "delete me", "keep too"}
	path := writeTestFile(t, lines)
	_, linesWritten, err := execStreamOp(path, func(_ int, line string) []string {
		if strings.Contains(line, "delete") {
			return []string{} // delete
		}
		return []string{line}
	})
	if err != nil {
		t.Fatal(err)
	}
	if linesWritten != 2 {
		t.Errorf("linesWritten=%d, want 2", linesWritten)
	}
	got := readTestFile(t, path)
	if len(got) != 2 || got[0] != "keep" || got[1] != "keep too" {
		t.Errorf("delete failed: %v", got)
	}
}

func TestExecStreamOp_Insert(t *testing.T) {
	lines := []string{"A", "B", "C"}
	path := writeTestFile(t, lines)
	_, _, err := execStreamOp(path, func(_ int, line string) []string {
		if line == "B" {
			return []string{"BEFORE", line, "AFTER"}
		}
		return []string{line}
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	// A, BEFORE, B, AFTER, C
	if len(got) != 5 {
		t.Fatalf("expected 5 lines, got %d: %v", len(got), got)
	}
	if got[1] != "BEFORE" || got[2] != "B" || got[3] != "AFTER" {
		t.Errorf("insert failed: %v", got)
	}
}

func TestExecStreamOp_AtomicOnError(t *testing.T) {
	// If the source can't be opened, original file is untouched.
	_, _, err := execStreamOp("/nonexistent/path/file.txt", func(_ int, line string) []string {
		return []string{line}
	})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestExecStreamOp_LineNumbers(t *testing.T) {
	lines := testLines(5)
	path := writeTestFile(t, lines)
	var seen []int
	execStreamOp(path, func(lineNum int, line string) []string {
		seen = append(seen, lineNum)
		return []string{line}
	})
	for i, n := range seen {
		if n != i+1 {
			t.Errorf("lineNum[%d] = %d, want %d", i, n, i+1)
		}
	}
}

func TestExecStreamOp_EmptyFile(t *testing.T) {
	f, _ := os.CreateTemp("", "fedit_empty_*.txt")
	f.Close()
	path := f.Name()
	t.Cleanup(func() { os.Remove(path) })
	linesRead, linesWritten, err := execStreamOp(path, func(_ int, line string) []string {
		return []string{line}
	})
	if err != nil {
		t.Fatal(err)
	}
	if linesRead != 0 || linesWritten != 0 {
		t.Errorf("truly empty file: read=%d written=%d, want 0/0", linesRead, linesWritten)
	}
}

func TestExecStreamOp_LargeFile(t *testing.T) {
	// 100k lines -- verifies streaming doesn't OOM
	n := 100000
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("row_%07d_data", i)
	}
	path := writeTestFile(t, lines)
	changed := 0
	target := "row_0000000_data"
	linesRead, _, err := execStreamOp(path, func(_ int, line string) []string {
		if line == target {
			changed++
			return []string{"ROW_0000000_data"}
		}
		return []string{line}
	})
	if err != nil {
		t.Fatal(err)
	}
	if linesRead != n {
		t.Errorf("linesRead=%d, want %d", linesRead, n)
	}
	if changed != 1 {
		t.Errorf("changed=%d, want 1", changed)
	}
	got := readTestFile(t, path)
	if got[0] != "ROW_0000000_data" {
		t.Errorf("got[0] = %q", got[0])
	}
	if got[1] != "row_0000001_data" {
		t.Errorf("got[1] = %q", got[1])
	}
}

// ── doStreamReplaceAll ────────────────────────────────────────

func TestDoStreamReplaceAll_Basic(t *testing.T) {
	lines := []string{"hello world", "foo bar", "hello again"}
	path := writeTestFile(t, lines)
	doStreamReplaceAll(path, "hello", "hi")
	got := readTestFile(t, path)
	if got[0] != "hi world" || got[2] != "hi again" {
		t.Errorf("stream replace failed: %v", got)
	}
	if got[1] != "foo bar" {
		t.Errorf("unchanged line modified: %q", got[1])
	}
}

func TestDoStreamReplaceAll_MultipleOccurrencesPerLine(t *testing.T) {
	lines := []string{"foo foo foo"}
	path := writeTestFile(t, lines)
	doStreamReplaceAll(path, "foo", "bar")
	got := readTestFile(t, path)
	if got[0] != "bar bar bar" {
		t.Errorf("got %q, want 'bar bar bar'", got[0])
	}
}

func TestDoStreamReplaceAll_FilePreservedOnNoMatch(t *testing.T) {
	// When no match, doStreamReplaceAll calls os.Exit(1).
	// Test via execStreamOp directly.
	lines := testLines(5)
	path := writeTestFile(t, lines)
	_, _, err := execStreamOp(path, func(_ int, line string) []string {
		return []string{strings.ReplaceAll(line, "NOMATCH_XYZ", "new")}
	})
	if err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	// File unchanged (no match = pass-through)
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("line %d modified when no match: %q", i+1, got[i])
		}
	}
}

// ── doStreamReplaceAllRegex ───────────────────────────────────

func TestDoStreamReplaceAllRegex_CaptureGroup(t *testing.T) {
	lines := []string{`version = "1.2.3"`, `other = "nope"`}
	path := writeTestFile(t, lines)
	doStreamReplaceAllRegex(path, `"(\d+\.\d+\.\d+)"`, `"[$1]"`)
	got := readTestFile(t, path)
	if got[0] != `version = "[1.2.3]"` {
		t.Errorf("got %q", got[0])
	}
	if got[1] != `other = "nope"` {
		t.Errorf("unchanged line modified: %q", got[1])
	}
}

func TestDoStreamReplaceAllRegex_AnchoredPattern(t *testing.T) {
	lines := []string{"func alpha()", "  func nested()", "func beta()"}
	path := writeTestFile(t, lines)
	doStreamReplaceAllRegex(path, `^func (\w+)`, "func NEW_$1")
	got := readTestFile(t, path)
	if got[0] != "func NEW_alpha()" {
		t.Errorf("got[0] = %q", got[0])
	}
	if got[1] != "  func nested()" {
		t.Errorf("nested should be unchanged: %q", got[1])
	}
	if got[2] != "func NEW_beta()" {
		t.Errorf("got[2] = %q", got[2])
	}
}

// ── doStreamFind ─────────────────────────────────────────────

func TestDoStreamFind_DoesNotModifyFile(t *testing.T) {
	lines := testLines(10)
	path := writeTestFile(t, lines)
	// Redirect stdout to /dev/null for test cleanliness
	old := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	doStreamFind(path, "line 3")
	os.Stdout = old
	// File must be unchanged
	got := readTestFile(t, path)
	for i, want := range lines {
		if got[i] != want {
			t.Errorf("doStreamFind modified file at line %d", i+1)
		}
	}
}

// ── doFields ─────────────────────────────────────────────────

func TestDoFields_TSV(t *testing.T) {
	lines := []string{"alpha\tbeta\tgamma", "one\ttwo\tthree"}
	path := writeTestFile(t, lines)
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 2, "\t")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	out := string(buf[:n])
	lines2 := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if lines2[0] != "beta" {
		t.Errorf("col2 row1 = %q, want 'beta'", lines2[0])
	}
	if lines2[1] != "two" {
		t.Errorf("col2 row2 = %q, want 'two'", lines2[1])
	}
}

func TestDoFields_CSV(t *testing.T) {
	lines := []string{"a,b,c", "1,2,3"}
	path := writeTestFile(t, lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 1, ",")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	rows := strings.Split(out, "\n")
	if rows[0] != "a" || rows[1] != "1" {
		t.Errorf("CSV col1 = %v, want [a 1]", rows)
	}
}

func TestDoFields_LastColumn(t *testing.T) {
	lines := []string{"x\ty\tz"}
	path := writeTestFile(t, lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 3, "\t")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	if out != "z" {
		t.Errorf("last col = %q, want 'z'", out)
	}
}

func TestDoFields_ColBeyondWidth_Skipped(t *testing.T) {
	// Lines shorter than col are skipped silently
	lines := []string{"a\tb", "x\ty\tz"}
	path := writeTestFile(t, lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 3, "\t") // first line has only 2 cols
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	rows := strings.Split(out, "\n")
	// Only second row extracted
	if len(rows) != 1 || rows[0] != "z" {
		t.Errorf("expected only row 2 col 3: %v", rows)
	}
}

func TestDoFields_EmptyDelimiter(t *testing.T) {
	// Single-space delimiter
	lines := []string{"hello world foo"}
	path := writeTestFile(t, lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 2, " ")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	out := strings.TrimRight(string(buf[:n]), "\n")
	if out != "world" {
		t.Errorf("space-delim col2 = %q, want 'world'", out)
	}
}

func TestDoFields_LargeFile(t *testing.T) {
	// 10k lines, verify streaming doesn't break
	dir, _ := os.MkdirTemp("", "fedit_fields_*")
	defer os.RemoveAll(dir)
	path := filepath.Join(dir, "data.tsv")
	f, _ := os.Create(path)
	for i := 0; i < 10000; i++ {
		fmt.Fprintf(f, "col1_%d\tcol2_%d\tcol3_%d\n", i, i, i)
	}
	f.Close()

	old := os.Stdout
	dev, _ := os.Open(os.DevNull)
	os.Stdout = dev
	doFields(path, 2, "\t")
	dev.Close()
	os.Stdout = old
	// No panic = pass
}

func TestDoFields_FirstColumn(t *testing.T) {
	lines := []string{"AAA\tBBB\tCCC"}
	path := writeTestFile(t, lines)
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	doFields(path, 1, "\t")
	w.Close()
	os.Stdout = old
	buf := make([]byte, 512)
	n, _ := r.Read(buf)
	if strings.TrimRight(string(buf[:n]), "\n") != "AAA" {
		t.Errorf("col1 = %q, want 'AAA'", string(buf[:n]))
	}
}

// ── integration: stream replaceall on glob ────────────────────

func TestStreamReplaceAll_GlobAndStream_Independent(t *testing.T) {
	// Verify glob (doReplaceAllGlob) and stream (doStreamReplaceAll)
	// produce the same result on single file.
	dir, _ := os.MkdirTemp("", "fedit_stream_glob_*")
	defer os.RemoveAll(dir)

	content := []string{"OldName alpha", "OldName beta", "unchanged"}
	p1 := filepath.Join(dir, "f1.go")
	p2 := filepath.Join(dir, "f2.go")
	writeLines(p1, content)
	writeLines(p2, content)

	// Glob path
	doReplaceAllGlob(filepath.Join(dir, "f1.go"), "OldName", "NewName")
	// Stream path
	doStreamReplaceAll(p2, "OldName", "NewName")

	g1, _ := readLines(p1)
	g2, _ := readLines(p2)
	for i := range g1 {
		if g1[i] != g2[i] {
			t.Errorf("line %d: glob=%q stream=%q (should match)", i+1, g1[i], g2[i])
		}
	}
}
