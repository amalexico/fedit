package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tfhWrite writes the text file of these tests and returns its path, its length and its SHA-256.
func tfhWrite(t *testing.T, dir string) (string, string, string) {
	t.Helper()
	raw := "alpha\r\nbeta\n"
	p := filepath.Join(dir, "body.txt")
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(raw))
	return p, "12", hex.EncodeToString(sum[:])
}

// TestLogCLIFields_TextFileHash: the begin record of a -textfile run carries the length and SHA-256 of the file, never its content, with or without -log-content; a file that cannot be read logs the path only.
func TestLogCLIFields_TextFileHash(t *testing.T) {
	dir := t.TempDir()
	p, wantLen, wantSum := tfhWrite(t, dir)
	for _, content := range []bool{false, true} {
		kvs := logCLIFields(logCLIInput{Cwd: dir, TextFile: "body.txt", Content: content})
		if v, _ := lwxVal(kvs, "textfile"); v != p {
			t.Errorf("content=%v: textfile = %q; want %q", content, v, p)
		}
		if v, _ := lwxVal(kvs, "textfile_len"); v != wantLen {
			t.Errorf("content=%v: textfile_len = %q; want %q", content, v, wantLen)
		}
		if v, _ := lwxVal(kvs, "textfile_sha256"); v != wantSum {
			t.Errorf("content=%v: textfile_sha256 = %q; want %q", content, v, wantSum)
		}
		if _, ok := lwxVal(kvs, "text"); ok {
			t.Errorf("content=%v: the file content must never be logged verbatim", content)
		}
	}
	kvs := logCLIFields(logCLIInput{Cwd: dir, TextFile: "missing.txt"})
	if _, ok := lwxVal(kvs, "textfile"); !ok {
		t.Error("a missing file must still log the path")
	}
	for _, k := range []string{"textfile_len", "textfile_sha256"} {
		if _, ok := lwxVal(kvs, k); ok {
			t.Errorf("a missing file must not log %s", k)
		}
	}
}

// TestMCPTextFileLogged: the begin record of a logged MCP call with a textfile argument carries the length and SHA-256 of that file.
func TestMCPTextFileLogged(t *testing.T) {
	dir := mcpMMFixture(t)
	a := filepath.Join(dir, "a.txt")
	tf, wantLen, wantSum := tfhWrite(t, t.TempDir())
	logp := filepath.Join(t.TempDir(), "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	mcpLoggedExec("fedit_insert", map[string]any{"file": a, "line": 1, "textfile": tf})
	begins := mogBegins(t, logp)
	if len(begins) != 1 {
		t.Fatalf("want 1 begin record, got %d: %q", len(begins), begins)
	}
	if want := " textfile_len=" + wantLen + " textfile_sha256=" + wantSum; !strings.Contains(begins[0], want) {
		t.Errorf("the begin record lacks %q: %q", want, begins[0])
	}
}
