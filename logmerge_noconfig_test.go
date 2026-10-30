package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLogMerge_NoConfig: with a broken settings file, fedit -log merge stops and its message names the file and the -noconfig way out; fedit -log merge -noconfig skips the file and merges; -noconfig before merge is still a usage error.
func TestLogMerge_NoConfig(t *testing.T) {
	tmp := t.TempDir()
	ini := filepath.Join(tmp, "fedit.ini")
	if err := os.WriteFile(ini, []byte("log_level = debug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"FEDIT_CONFIG": ini, "LOCALAPPDATA": filepath.Join(tmp, "la"), "FEDIT_LOG_FILE": filepath.Join(tmp, "main.log")}
	getenv := func(k string) string { return env[k] }
	ucd := func() (string, error) { return filepath.Join(tmp, "cfg"), nil }
	var out bytes.Buffer
	code := logCommand([]string{"merge"}, getenv, tmp, ucd, fileLogIO(), &out)
	if code != 1 || !strings.Contains(out.String(), ini) || !strings.Contains(out.String(), "-noconfig") {
		t.Errorf("a broken ini must stop the merge and name -noconfig: exit %d, output %q", code, out.String())
	}
	out.Reset()
	code = logCommand([]string{"merge", "-noconfig"}, getenv, tmp, ucd, fileLogIO(), &out)
	if code != 0 || !strings.Contains(out.String(), "No spooled records to merge") {
		t.Errorf("merge -noconfig must skip the broken ini: exit %d, output %q", code, out.String())
	}
	out.Reset()
	code = logCommand([]string{"-noconfig", "merge"}, getenv, tmp, ucd, fileLogIO(), &out)
	if code != 1 || !strings.Contains(out.String(), "usage: fedit -log merge") {
		t.Errorf("-noconfig before merge stays a usage error: exit %d, output %q", code, out.String())
	}
}
