package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests of locating and reading the settings file (config_load.go).

func cflEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func cflCfgDir(dir string) func() (string, error) {
	return func() (string, error) { return dir, nil }
}

func cflWrite(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCfgLoad_DefaultPath(t *testing.T) {
	dir := t.TempDir()
	want := filepath.Join(dir, "fedit", "fedit.ini")
	if got := cfgPath(cflEnv(nil), "", cflCfgDir(dir)); got != want {
		t.Errorf("default path %q; want %q", got, want)
	}
	cflWrite(t, want, "log_mode = permissive\nlog_max_mb = 3\n")
	r, err := cfgLoad(cflEnv(nil), "", cflCfgDir(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !r.Exists || r.Path != want || r.Inputs.Mode != "permissive" || r.Inputs.MaxMB != "3" {
		t.Errorf("loaded %+v", r)
	}
}

func TestCfgLoad_EnvOverridesTheDefault(t *testing.T) {
	dir := t.TempDir()
	cflWrite(t, filepath.Join(dir, "fedit", "fedit.ini"), "log_mode = off\n")
	mine := filepath.Join(t.TempDir(), "mine.ini")
	cflWrite(t, mine, "log_mode = strict\n")
	r, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": mine}), "", cflCfgDir(dir))
	if err != nil || r.Path != mine || r.Inputs.Mode != "strict" {
		t.Errorf("got %+v, %v; want the FEDIT_CONFIG file with mode strict", r, err)
	}
}

func TestCfgLoad_TildeInEnvPath(t *testing.T) {
	home := t.TempDir()
	cflWrite(t, filepath.Join(home, "my.ini"), "log_level = error\n")
	r, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": "~/my.ini"}), home, cflCfgDir(t.TempDir()))
	if err != nil || filepath.Clean(r.Path) != filepath.Join(home, "my.ini") || r.Inputs.Level != "error" {
		t.Errorf("got %+v, %v", r, err)
	}
}

func TestCfgLoad_MissingFileIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	r, err := cfgLoad(cflEnv(nil), "", cflCfgDir(dir))
	if err != nil {
		t.Fatalf("a missing file must not be an error: %v", err)
	}
	if r.Exists || r.Path != filepath.Join(dir, "fedit", "fedit.ini") || (r.Inputs != logInputs{}) {
		t.Errorf("got %+v", r)
	}
}

func TestCfgLoad_NoLocationIsNotAnError(t *testing.T) {
	failing := func() (string, error) { return "", errors.New("no config dir") }
	r, err := cfgLoad(cflEnv(nil), "", failing)
	if err != nil || r.Path != "" || r.Exists {
		t.Errorf("got %+v, %v; want nothing loaded and no error", r, err)
	}
}

func TestCfgLoad_InvalidFileIsAnError(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bad.ini")
	cflWrite(t, p, "log_mode = strict\nlog_level = debug\n")
	_, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": p}), "", cflCfgDir(t.TempDir()))
	if err == nil || !strings.Contains(err.Error(), p) || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("want an error naming the file and line 2: %v", err)
	}
}

func TestCfgLoad_UnknownKeyIsAWarning(t *testing.T) {
	p := filepath.Join(t.TempDir(), "w.ini")
	cflWrite(t, p, "mystery = 1\nlog_scope = writes\n")
	r, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": p}), "", cflCfgDir(t.TempDir()))
	if err != nil || r.Inputs.Scope != "writes" {
		t.Fatalf("got %+v, %v", r, err)
	}
	if len(r.Warnings) != 1 || !strings.Contains(r.Warnings[0], "mystery") {
		t.Errorf("want one warning naming the key: %q", r.Warnings)
	}
}

func TestCfgLoad_ByteOrderMarkIsIgnored(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bom.ini")
	cflWrite(t, p, "\ufefflog_mode = strict\r\n")
	r, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": p}), "", cflCfgDir(t.TempDir()))
	if err != nil || r.Inputs.Mode != "strict" || len(r.Warnings) != 0 {
		t.Errorf("got %+v, %v", r, err)
	}
}

func TestCfgLoad_DirectoryAtThePathIsAnError(t *testing.T) {
	d := t.TempDir()
	_, err := cfgLoad(cflEnv(map[string]string{"FEDIT_CONFIG": d}), "", cflCfgDir(t.TempDir()))
	if err == nil || !strings.Contains(err.Error(), d) {
		t.Errorf("want an error naming the path: %v", err)
	}
}
