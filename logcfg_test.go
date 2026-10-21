package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func lcxEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func lcxNoConfig() (string, error) { return "", errors.New("no config dir") }

// lcxBase returns a fake %LOCALAPPDATA% and an environment that holds it.
func lcxBase(t *testing.T) (string, map[string]string) {
	t.Helper()
	la := filepath.Join(t.TempDir(), "la")
	return la, map[string]string{"LOCALAPPDATA": la}
}

func TestLC_Defaults(t *testing.T) {
	s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig)
	if err != nil {
		t.Fatalf("defaults must not fail: %v", err)
	}
	if s.Mode != logOff || s.MaxMB != 10 || s.Content || s.File != "" || s.SpoolDir != "" {
		t.Errorf("defaults = %+v; want mode off, 10 MB, content off, no file, no spool", s)
	}
}

func TestLC_EnvNames(t *testing.T) {
	env := map[string]string{
		"FEDIT_LOG_MODE":    "strict",
		"FEDIT_LOG_FILE":    "x.log",
		"FEDIT_LOG_MAX_MB":  "7",
		"FEDIT_LOG_CONTENT": "on",
	}
	got := logInputsFromEnv(lcxEnv(env))
	want := logInputs{Mode: "strict", File: "x.log", MaxMB: "7", Content: "on"}
	if got != want {
		t.Errorf("logInputsFromEnv = %+v; want %+v", got, want)
	}
}

func TestLC_FlagBeatsEnvBeatsDefault(t *testing.T) {
	_, env := lcxBase(t)
	env["FEDIT_LOG_MODE"] = "strict"
	env["FEDIT_LOG_MAX_MB"] = "5"
	g := lcxEnv(env)

	s, err := resolveLogSettings(g, "", lcxNoConfig, logInputsFromEnv(g))
	if err != nil || s.Mode != logStrict || s.MaxMB != 5 {
		t.Fatalf("env only = %+v, %v; want strict and 5 MB", s, err)
	}
	s, err = resolveLogSettings(g, "", lcxNoConfig, logInputs{Mode: "permissive"}, logInputsFromEnv(g))
	if err != nil || s.Mode != logPermissive || s.MaxMB != 5 {
		t.Fatalf("flag over env = %+v, %v; want permissive (flag) and 5 MB (env fills the gap)", s, err)
	}
}

func TestLC_WhitespaceIsUnset(t *testing.T) {
	_, env := lcxBase(t)
	s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, logInputs{Mode: "   "}, logInputs{Mode: "strict"})
	if err != nil || s.Mode != logStrict {
		t.Errorf("a blank value must fall through to the next layer: %+v, %v", s, err)
	}
}

func TestLC_Invalid(t *testing.T) {
	cases := []struct {
		in   logInputs
		name string
	}{
		{logInputs{Mode: "loud"}, "log_mode"},
		{logInputs{MaxMB: "abc"}, "log_max_mb"},
		{logInputs{MaxMB: "-1"}, "log_max_mb"},
		{logInputs{MaxMB: "1.5"}, "log_max_mb"},
		{logInputs{Content: "maybe"}, "log_content"},
	}
	for _, c := range cases {
		_, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, c.in)
		if err == nil || !strings.Contains(err.Error(), c.name) {
			t.Errorf("%+v: error = %v; want one naming %s", c.in, err, c.name)
		}
	}
}

func TestLC_MaxMBZero(t *testing.T) {
	s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{MaxMB: "0"})
	if err != nil || s.MaxMB != 0 {
		t.Errorf("0 means never rotate: %+v, %v", s, err)
	}
}

func TestLC_ContentOnOff(t *testing.T) {
	for in, want := range map[string]bool{"on": true, "ON": true, " off ": false, "off": false} {
		s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Content: in})
		if err != nil || s.Content != want {
			t.Errorf("log_content %q = %v, %v; want %v", in, s.Content, err, want)
		}
	}
}

func TestLC_DefaultFileAndSpool(t *testing.T) {
	la, env := lcxBase(t)
	s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, logInputs{Mode: "permissive"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if want := filepath.Join(la, "fedit", "fedit.log"); s.File != want {
		t.Errorf("File = %q; want %q", s.File, want)
	}
	if want := filepath.Join(la, "fedit", "spool"); s.SpoolDir != want {
		t.Errorf("SpoolDir = %q; want %q", s.SpoolDir, want)
	}
}

func TestLC_OffStillResolvesWhenKnown(t *testing.T) {
	la, env := lcxBase(t)
	s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig)
	if err != nil || s.Mode != logOff {
		t.Fatalf("resolve: %+v, %v", s, err)
	}
	if want := filepath.Join(la, "fedit", "fedit.log"); s.File != want {
		t.Errorf("with mode off the file is still resolved (for -log merge): %q; want %q", s.File, want)
	}
}

func TestLC_FileExpansion(t *testing.T) {
	la, env := lcxBase(t)
	home := t.TempDir()
	s, err := resolveLogSettings(lcxEnv(env), home, lcxNoConfig, logInputs{Mode: "permissive", File: "~/x/a.log"})
	if err != nil || s.File != filepath.Join(home, "x", "a.log") {
		t.Errorf("~ expansion: %q, %v", s.File, err)
	}
	s, err = resolveLogSettings(lcxEnv(env), home, lcxNoConfig, logInputs{Mode: "permissive", File: "%LOCALAPPDATA%/logs/a.log"})
	if err != nil || s.File != filepath.Join(la, "logs", "a.log") {
		t.Errorf("%%VAR%% expansion: %q, %v", s.File, err)
	}
}

func TestLC_RelativeBecomesAbsolute(t *testing.T) {
	s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Mode: "permissive", File: "rel.log"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !filepath.IsAbs(s.File) || filepath.Base(s.File) != "rel.log" {
		t.Errorf("a relative log_file must become absolute so it does not depend on later directory changes: %q", s.File)
	}
}

func TestLC_CustomFileSpoolInBase(t *testing.T) {
	la, env := lcxBase(t)
	custom := filepath.Join(t.TempDir(), "custom.log")
	s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, logInputs{Mode: "permissive", File: custom})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if s.File != custom {
		t.Errorf("File = %q; want %q", s.File, custom)
	}
	if want := filepath.Join(la, "fedit", "spool"); s.SpoolDir != want {
		t.Errorf("the spool lives in the per-user folder even with a custom log_file: %q; want %q", s.SpoolDir, want)
	}
}

func TestLC_CustomFileNoBase(t *testing.T) {
	custom := filepath.Join(t.TempDir(), "custom.log")
	s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Mode: "strict", File: custom})
	if err != nil {
		t.Fatalf("a custom log_file needs no per-user folder: %v", err)
	}
	if want := filepath.Join(filepath.Dir(custom), "spool"); s.SpoolDir != want {
		t.Errorf("SpoolDir = %q; want %q (next to the log file)", s.SpoolDir, want)
	}
}

func TestLC_NoLocationIsError(t *testing.T) {
	for _, mode := range []string{"permissive", "strict"} {
		_, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Mode: mode})
		if err == nil || !strings.Contains(err.Error(), "log_file") {
			t.Errorf("mode %s with no usable location: error = %v; want one telling to set log_file", mode, err)
		}
	}
}
