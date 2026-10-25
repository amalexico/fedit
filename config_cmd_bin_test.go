package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Binary-level cases for -config. They are subtests of TestLW_Bin (one build)
// and use the helpers of the logging tests.

func cfcBinRun(t *testing.T, bin string, args ...string) (string, string, int) {
	t.Helper()
	so, se, code := orxRun(t, bin, append([]string{"-config"}, args...)...)
	return orxNorm(so), orxNorm(se), code
}

func cfcBinFile(t *testing.T, path string) (string, bool) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return orxNorm(string(b)), true
}

func cfcBinPathShowGet(t *testing.T, bin string) {
	lwxEnv(t)
	dir := t.TempDir()
	ini := filepath.Join(dir, "my.ini")
	t.Setenv("FEDIT_CONFIG", ini)

	so, se, code := cfcBinRun(t, bin, "path")
	if code != 0 || filepath.Clean(strings.TrimSpace(so)) != filepath.Clean(ini) {
		t.Fatalf("path: exit %d, stdout %q, stderr %q, want %q", code, so, se, ini)
	}
	if so, _, code = cfcBinRun(t, bin, "get", "log_mode"); code != 0 || strings.TrimSpace(so) != "off" {
		t.Errorf("get log_mode with no file: exit %d, stdout %q, want off", code, so)
	}

	orxWrite(t, dir, "my.ini", "log_mode = permissive\n")
	if so, _, code = cfcBinRun(t, bin, "get", "log_mode"); code != 0 || strings.TrimSpace(so) != "permissive" {
		t.Errorf("get log_mode from the ini: exit %d, stdout %q, want permissive", code, so)
	}

	t.Setenv("FEDIT_LOG_LEVEL", "warning")
	if so, _, code = cfcBinRun(t, bin, "get", "log_level"); code != 0 || strings.TrimSpace(so) != "warning" {
		t.Errorf("get log_level from the environment: exit %d, stdout %q, want warning", code, so)
	}
	so, se, code = cfcBinRun(t, bin, "show")
	if code != 0 {
		t.Fatalf("show: exit %d, stderr %q", code, se)
	}
	for _, want := range []string{"log_mode = permissive (ini)", "log_level = warning (environment)", "log_scope = all (default)"} {
		if !strings.Contains(so, want) {
			t.Errorf("show lacks %q:\n%s", want, so)
		}
	}
}

func cfcBinSetUnset(t *testing.T, bin string) {
	lwxEnv(t)
	dir := t.TempDir()
	ini := filepath.Join(dir, "sub", "fedit.ini")
	t.Setenv("FEDIT_CONFIG", ini)

	steps := []struct {
		args []string
		want string
	}{
		{[]string{"set", "log_scope", "writes"}, "log_scope = writes\n"},
		{[]string{"set", "log_max_mb", "5"}, "log_scope = writes\nlog_max_mb = 5\n"},
		{[]string{"unset", "log_scope"}, "log_max_mb = 5\n"},
	}
	for _, s := range steps {
		if _, se, code := cfcBinRun(t, bin, s.args...); code != 0 {
			t.Fatalf("%q: exit %d, stderr %q", s.args, code, se)
		}
		if got, _ := cfcBinFile(t, ini); got != s.want {
			t.Fatalf("after %q the file is %q, want %q", s.args, got, s.want)
		}
	}
	if so, _, code := cfcBinRun(t, bin, "get", "log_max_mb"); code != 0 || strings.TrimSpace(so) != "5" {
		t.Errorf("get log_max_mb: exit %d, stdout %q, want 5", code, so)
	}
	so, _, code := cfcBinRun(t, bin, "unset", "log_scope")
	if code != 0 || !strings.Contains(so, "not set") {
		t.Errorf("unset of an absent key: exit %d, stdout %q", code, so)
	}
	if got, _ := cfcBinFile(t, ini); got != "log_max_mb = 5\n" {
		t.Errorf("unset of an absent key changed the file to %q", got)
	}
}

func cfcBinRejects(t *testing.T, bin string) {
	lwxEnv(t)
	dir := t.TempDir()
	ini := filepath.Join(dir, "fedit.ini")
	t.Setenv("FEDIT_CONFIG", ini)

	if _, se, code := cfcBinRun(t, bin, "set", "log_mode", "loud"); code != 1 || !strings.Contains(se, "invalid value") {
		t.Errorf("invalid value: exit %d, stderr %q", code, se)
	}
	if _, se, code := cfcBinRun(t, bin, "get", "nope"); code != 1 || !strings.Contains(se, "nope") {
		t.Errorf("unknown key: exit %d, stderr %q", code, se)
	}
	if _, se, code := cfcBinRun(t, bin, "set", "log_mode"); code != 1 || !strings.Contains(se, "usage: fedit -config") {
		t.Errorf("missing value: exit %d, stderr %q", code, se)
	}
	if _, se, code := cfcBinRun(t, bin); code != 1 || !strings.Contains(se, "usage: fedit -config") {
		t.Errorf("no subcommand: exit %d, stderr %q", code, se)
	}
	if _, ok := cfcBinFile(t, ini); ok {
		t.Errorf("rejected commands must not create the settings file")
	}
}

func cfcBinBrokenIni(t *testing.T, bin string) {
	lwxEnv(t)
	dir := t.TempDir()
	const orig = "log_mode = loud\n"
	ini := orxWrite(t, dir, "fedit.ini", orig)
	t.Setenv("FEDIT_CONFIG", ini)

	if so, _, code := cfcBinRun(t, bin, "path"); code != 0 || filepath.Clean(strings.TrimSpace(so)) != filepath.Clean(ini) {
		t.Errorf("path must work with a broken file: exit %d, stdout %q", code, so)
	}
	for _, args := range [][]string{{"show"}, {"get", "log_mode"}, {"set", "log_level", "warning"}, {"unset", "log_mode"}} {
		_, se, code := cfcBinRun(t, bin, args...)
		if code != 1 || !strings.Contains(se, "line 1") {
			t.Errorf("%q: exit %d, stderr %q; want exit 1 naming line 1", args, code, se)
		}
	}
	if got, _ := cfcBinFile(t, ini); got != orig {
		t.Errorf("the broken file was changed to %q", got)
	}
}

func cfcBinSetIsLogged(t *testing.T, bin string) {
	lwxEnv(t)
	dir := t.TempDir()
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_CONFIG", filepath.Join(dir, "fedit.ini"))
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)

	if _, se, code := cfcBinRun(t, bin, "set", "log_max_mb", "5"); code != 0 {
		t.Fatalf("set: exit %d, stderr %q", code, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 {
		t.Fatalf("set must write a begin and an end record, got %q", lines)
	}
	for _, want := range []string{"phase=begin op=config-set source=cli", " config_key=log_max_mb", " config_value=5"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("begin record lacks %q: %q", want, lines[0])
		}
	}
	if !strings.Contains(lines[1], "phase=end exit=0") {
		t.Errorf("end record = %q", lines[1])
	}

	for _, args := range [][]string{{"path"}, {"show"}, {"get", "log_max_mb"}} {
		cfcBinRun(t, bin, args...)
	}
	if n := len(lwxLog(t, logp)); n != 2 {
		t.Errorf("path, show and get are not logged: %d records", n)
	}

	if _, _, code := cfcBinRun(t, bin, "set", "log_max_mb", "abc"); code != 1 {
		t.Errorf("a rejected value must exit 1, got %d", code)
	}
	lines = lwxLog(t, logp)
	if len(lines) != 4 || !strings.Contains(lines[2], " config_value=abc") || !strings.Contains(lines[3], "phase=end exit=1") {
		t.Errorf("a rejected set is logged with exit 1, got %q", lines)
	}

	cfcBinRun(t, bin, "set", "log_max_mb")
	if n := len(lwxLog(t, logp)); n != 4 {
		t.Errorf("a usage error is not an operation and is not logged: %d records", n)
	}

	if _, _, code := cfcBinRun(t, bin, "unset", "log_max_mb"); code != 0 {
		t.Fatalf("unset: exit %d", code)
	}
	lines = lwxLog(t, logp)
	if len(lines) != 6 || !strings.Contains(lines[4], "op=config-unset") || !strings.Contains(lines[4], " config_key=log_max_mb") || strings.Contains(lines[4], "config_value") {
		t.Errorf("unset is logged with the key only, got %q", lines)
	}
}
