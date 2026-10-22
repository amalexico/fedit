package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lmcxEnv points the audit log at logp in the given mode through the
// environment, which is all the mcp command reads.
func lmcxEnv(t *testing.T, mode, logp string) {
	t.Helper()
	for _, k := range []string{"FEDIT_LOG_MODE", "FEDIT_LOG_FILE", "FEDIT_LOG_MAX_MB", "FEDIT_LOG_CONTENT", "FEDIT_LOG_LEVEL", "FEDIT_LOG_SCOPE", "FEDIT_LOG_PARENT"} {
		t.Setenv(k, "")
	}
	t.Setenv("FEDIT_LOG_MODE", mode)
	t.Setenv("FEDIT_LOG_FILE", logp)
	mcpClientName, mcpClientVersion = "", ""
	t.Cleanup(func() { mcpClientName, mcpClientVersion = "", "" })
}

func lmcxLines(t *testing.T, logp string) []string {
	t.Helper()
	b, err := os.ReadFile(logp)
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimRight(strings.ReplaceAll(string(b), "\r\n", "\n"), "\n"), "\n")
}

func lmcxArgs(f string) map[string]any {
	return map[string]any{"file": f, "match": "alpha", "text": "beta"}
}

func lmcxHas(t *testing.T, rec string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(rec, w) {
			t.Errorf("record lacks %q: %s", w, rec)
		}
	}
}

func TestLMC_NoteClient(t *testing.T) {
	mcpNoteClient([]byte(`{"protocolVersion":"2024-11-05","clientInfo":{"name":"claude-desktop","version":"1.2.3"}}`))
	if mcpClientName != "claude-desktop" || mcpClientVersion != "1.2.3" {
		t.Errorf("client = %q %q", mcpClientName, mcpClientVersion)
	}
	mcpNoteClient([]byte(`not json`))
	if mcpClientName != "" || mcpClientVersion != "" {
		t.Errorf("invalid params must leave the client unknown: %q %q", mcpClientName, mcpClientVersion)
	}
	mcpNoteClient(nil)
	if mcpClientName != "" {
		t.Errorf("empty params must leave the client unknown: %q", mcpClientName)
	}
	mcpNoteClient([]byte(`{"clientInfo":{"name":"a\nb\u0007c","version":"` + strings.Repeat("9", 300) + `"}}`))
	if mcpClientName != "abc" {
		t.Errorf("control characters must be dropped: %q", mcpClientName)
	}
	if len(mcpClientVersion) != 100 {
		t.Errorf("version must be capped at 100 bytes, got %d", len(mcpClientVersion))
	}
	mcpNoteClient(nil)
}

func TestLMC_LoggingOffWritesNothing(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit.log")
	lmcxEnv(t, "", logp)
	f := lhxFile(t, "alpha\n")
	if res := mcpLoggedExec("fedit_replaceall", lmcxArgs(f)); res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	if _, err := os.Stat(logp); err == nil {
		t.Error("a log file was created while log_mode is unset")
	}
	if logRun != nil {
		t.Error("the session was not reset")
	}
}

func TestLMC_PermissiveRecords(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit", "fedit.log")
	lmcxEnv(t, "permissive", logp)
	mcpNoteClient([]byte(`{"clientInfo":{"name":"testclient","version":"9.9"}}`))
	f := lhxFile(t, "alpha one\n")
	before, _ := os.ReadFile(f)
	if res := mcpLoggedExec("fedit_replaceall", lmcxArgs(f)); res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	after, _ := os.ReadFile(f)
	lines := lmcxLines(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	lmcxHas(t, lines[0], "phase=begin", "op=replaceall", "source=mcp", "client=testclient", "client_version=9.9")
	lmcxHas(t, lines[1], "phase=end", "exit=0", "sha256_before="+lhxSum(before), "sha256_after="+lhxSum(after))
	if strings.Contains(lines[0], "=beta") {
		t.Errorf("the text must not be logged verbatim: %s", lines[0])
	}
	if logSourceName != "cli" {
		t.Errorf("logSourceName = %q, want it restored to cli", logSourceName)
	}
}

func TestLMC_MultiFileRecordsResults(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit.log")
	lmcxEnv(t, "permissive", logp)
	a, b := lhxFile(t, "alpha\n"), lhxFile(t, "alpha two\n")
	args := map[string]any{"file": a + "," + b, "match": "alpha", "text": "beta"}
	if res := mcpLoggedExec("fedit_replaceall", args); res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	lines := lmcxLines(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	lmcxHas(t, lines[1], "files_total=2", "files_ok=2", "verify=PASS")
	if strings.Contains(lines[1], "sha256_") {
		t.Errorf("a multi-file call reports per-file results, not hashes: %s", lines[1])
	}
}

func TestLMC_ReadCallIsLoggedWithoutHashes(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit.log")
	lmcxEnv(t, "permissive", logp)
	f := lhxFile(t, "alpha\n")
	if res := mcpLoggedExec("fedit_find", map[string]any{"file": f, "match": "alpha"}); res.IsError {
		t.Fatalf("unexpected error result: %+v", res)
	}
	lines := lmcxLines(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	lmcxHas(t, lines[0], "op=find", "source=mcp")
	if strings.Contains(lines[1], "sha256_") {
		t.Errorf("a read must not log hashes: %s", lines[1])
	}
}

func TestLMC_ErrorResultIsExit1(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit.log")
	lmcxEnv(t, "permissive", logp)
	if res := mcpLoggedExec("fedit_replaceall", map[string]any{"match": "x"}); !res.IsError {
		t.Fatalf("a call without file must be an error: %+v", res)
	}
	lines := lmcxLines(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	lmcxHas(t, lines[1], "[WARNING]", "phase=end", "exit=1")
}

func TestLMC_EachCallGetsItsOwnRecords(t *testing.T) {
	logp := filepath.Join(t.TempDir(), "audit.log")
	lmcxEnv(t, "permissive", logp)
	f := lhxFile(t, "alpha alpha\n")
	mcpLoggedExec("fedit_replaceall", lmcxArgs(f))
	mcpLoggedExec("fedit_find", map[string]any{"file": f, "match": "beta"})
	lines := lmcxLines(t, logp)
	if len(lines) != 4 {
		t.Fatalf("want 4 records, got %q", lines)
	}
	id1, _ := spoolLogField(lines[0], "id")
	id2, _ := spoolLogField(lines[2], "id")
	if id1 == "" || id1 == id2 {
		t.Errorf("ids = %q and %q, want two different ids", id1, id2)
	}
	if end1, _ := spoolLogField(lines[1], "id"); end1 != id1 {
		t.Errorf("the end record id %q does not match the begin id %q", end1, id1)
	}
	if strings.Contains(lines[3], "sha256_") || strings.Contains(lines[3], "replacements=") {
		t.Errorf("fields of the first call leaked into the second: %s", lines[3])
	}
	if len(logNotes) != 0 || logRun != nil {
		t.Errorf("state not reset: notes %v, session %v", logNotes, logRun)
	}
}

func TestLMC_StrictRefusesWhenTheLogCannotBeWritten(t *testing.T) {
	blocker := lhxFile(t, "not a folder")
	lmcxEnv(t, "strict", filepath.Join(blocker, "audit.log"))
	f := lhxFile(t, "alpha\n")
	res := mcpLoggedExec("fedit_replaceall", lmcxArgs(f))
	if !res.IsError || len(res.Content) == 0 || !strings.Contains(res.Content[0].Text, "nothing was changed") {
		t.Fatalf("strict must refuse the call: %+v", res)
	}
	if got, _ := os.ReadFile(f); string(got) != "alpha\n" {
		t.Errorf("the file was changed: %q", got)
	}
}

func TestLMC_InvalidSettingIsAnErrorResult(t *testing.T) {
	lmcxEnv(t, "loud", filepath.Join(t.TempDir(), "audit.log"))
	f := lhxFile(t, "alpha\n")
	res := mcpLoggedExec("fedit_replaceall", lmcxArgs(f))
	if !res.IsError {
		t.Fatalf("an invalid log_mode must be an error result: %+v", res)
	}
	if got, _ := os.ReadFile(f); string(got) != "alpha\n" {
		t.Errorf("the file was changed: %q", got)
	}
}
