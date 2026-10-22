package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// lmgxEnv returns a getenv with the main log at <tmp>/main.log and the
// per-user folder at <tmp>/la/fedit, so the spool folder is <tmp>/la/fedit/spool.
func lmgxEnv(tmp string, extra map[string]string) func(string) string {
	return func(k string) string {
		if v, ok := extra[k]; ok {
			return v
		}
		switch k {
		case "FEDIT_LOG_FILE":
			return filepath.Join(tmp, "main.log")
		case "LOCALAPPDATA":
			return filepath.Join(tmp, "la")
		}
		return ""
	}
}

func lmgxSpool(t *testing.T, tmp, id, line string) {
	t.Helper()
	failing := logIO{Append: func(string, string) error { return errors.New("log is down") }}
	sp := newSpoolIO(failing, filepath.Join(tmp, "la", "fedit", "spool"), id, nil)
	if err := sp.Append(filepath.Join(tmp, "main.log"), line); err != nil {
		t.Fatal(err)
	}
}

func lmgxRun(args []string, getenv func(string) string, lio logIO) (int, string) {
	var out bytes.Buffer
	code := logCommand(args, getenv, "", func() (string, error) { return "", errors.New("no config dir") }, lio, &out)
	return code, out.String()
}

func TestLMG_MergeMovesSpooledRecords(t *testing.T) {
	tmp := t.TempDir()
	lmgxSpool(t, tmp, "aaaaaaaaaaaaaaaa", "2026-10-22T10:00:00.000Z [INFO] id=aaaaaaaaaaaaaaaa phase=begin op=replace")
	code, out := lmgxRun([]string{"merge"}, lmgxEnv(tmp, nil), fileLogIO())
	if code != 0 || !strings.Contains(out, "Merged 1 spooled record(s)") {
		t.Fatalf("code = %d, output %q", code, out)
	}
	b, err := os.ReadFile(filepath.Join(tmp, "main.log"))
	if err != nil || !strings.Contains(string(b), "id=aaaaaaaaaaaaaaaa") || !strings.Contains(string(b), "late=true") {
		t.Errorf("the record is not in the main log: %q (%v)", b, err)
	}
	code, out = lmgxRun([]string{"merge"}, lmgxEnv(tmp, nil), fileLogIO())
	if code != 0 || !strings.Contains(out, "No spooled records") {
		t.Errorf("a second merge must find nothing: code %d, output %q", code, out)
	}
}

func TestLMG_NothingToMerge(t *testing.T) {
	code, out := lmgxRun([]string{"merge"}, lmgxEnv(t.TempDir(), nil), fileLogIO())
	if code != 0 || !strings.Contains(out, "No spooled records") {
		t.Errorf("code = %d, output %q", code, out)
	}
}

func TestLMG_UsageErrors(t *testing.T) {
	tmp := t.TempDir()
	for _, args := range [][]string{nil, {"bogus"}, {"merge", "extra"}} {
		code, out := lmgxRun(args, lmgxEnv(tmp, nil), fileLogIO())
		if code != 1 || !strings.Contains(out, "usage: fedit -log merge") {
			t.Errorf("args %v: code = %d, output %q", args, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(tmp, "main.log")); err == nil {
		t.Error("a usage error must not create the log")
	}
}

func TestLMG_InvalidSettingIsAnError(t *testing.T) {
	code, out := lmgxRun([]string{"merge"}, lmgxEnv(t.TempDir(), map[string]string{"FEDIT_LOG_LEVEL": "loud"}), fileLogIO())
	if code != 1 || !strings.Contains(out, "log_level") {
		t.Errorf("code = %d, output %q", code, out)
	}
}

func TestLMG_NoLocationIsAnError(t *testing.T) {
	code, out := lmgxRun([]string{"merge"}, func(string) string { return "" }, fileLogIO())
	if code != 1 || !strings.Contains(out, "cannot determine") {
		t.Errorf("code = %d, output %q", code, out)
	}
}

func TestLMG_MergeFailureIsReported(t *testing.T) {
	tmp := t.TempDir()
	lmgxSpool(t, tmp, "bbbbbbbbbbbbbbbb", "2026-10-22T10:00:00.000Z [INFO] id=bbbbbbbbbbbbbbbb phase=begin op=replace")
	failing := logIO{Append: func(string, string) error { return errors.New("log is down") }}
	code, out := lmgxRun([]string{"merge"}, lmgxEnv(tmp, nil), failing)
	if code != 1 || !strings.Contains(out, "the merge stopped") {
		t.Errorf("code = %d, output %q", code, out)
	}
}

// lwxBinLogMerge runs as a subtest of TestLW_Bin, which builds the binary once.
func lwxBinLogMerge(t *testing.T, bin string) {
	dir, _ := lwxSetup(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_FILE", logp)
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "la"))
	spool := filepath.Join(dir, "la", "fedit", "spool")
	if err := os.MkdirAll(spool, 0o755); err != nil {
		t.Fatal(err)
	}
	rec := "2026-10-22T10:00:00.000Z [INFO] id=cccccccccccccccc phase=begin op=replace logged=spool\n"
	if err := os.WriteFile(filepath.Join(spool, "cccccccccccccccc.log"), []byte(rec), 0o644); err != nil {
		t.Fatal(err)
	}
	_, se, code := orxRun(t, bin, "-log", "merge")
	if code != 0 || !strings.Contains(se, "Merged 1 spooled record(s)") {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if b, _ := os.ReadFile(logp); !strings.Contains(string(b), "id=cccccccccccccccc") || !strings.Contains(string(b), "late=true") {
		t.Errorf("the spooled record is not in the log: %q", b)
	}
	if _, se, code := orxRun(t, bin, "-log"); code != 1 || !strings.Contains(se, "usage: fedit -log merge") {
		t.Errorf("-log without a command: exit %d, stderr %q", code, se)
	}
}
