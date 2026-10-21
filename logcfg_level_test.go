package main

import (
	"strings"
	"testing"
)

// These tests extend the settings resolver with log_level (which severities
// are written) and log_scope (which operations are logged). They reuse the
// helpers of logcfg_test.go.

func TestLV_Defaults(t *testing.T) {
	s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig)
	if err != nil {
		t.Fatalf("defaults must not fail: %v", err)
	}
	if s.Level != logThresholdAll || s.Scope != logScopeAll {
		t.Errorf("defaults = level %q, scope %q; want all and all", s.Level, s.Scope)
	}
}

func TestLV_LevelValues(t *testing.T) {
	cases := map[string]logThreshold{
		"all":       logThresholdAll,
		"ALL":       logThresholdAll,
		" warning ": logThresholdWarning,
		"warnings":  logThresholdWarning,
		"Error":     logThresholdError,
		"errors":    logThresholdError,
	}
	for in, want := range cases {
		s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Level: in})
		if err != nil || s.Level != want {
			t.Errorf("log_level %q = %q, %v; want %q", in, s.Level, err, want)
		}
	}
}

func TestLV_ScopeValues(t *testing.T) {
	cases := map[string]logScope{
		"all":      logScopeAll,
		"writes":   logScopeWrites,
		"WRITES":   logScopeWrites,
		" writes ": logScopeWrites,
		"ALL":      logScopeAll,
	}
	for in, want := range cases {
		s, err := resolveLogSettings(lcxEnv(nil), "", lcxNoConfig, logInputs{Scope: in})
		if err != nil || s.Scope != want {
			t.Errorf("log_scope %q = %q, %v; want %q", in, s.Scope, err, want)
		}
	}
}

func TestLV_Invalid(t *testing.T) {
	_, env := lcxBase(t)
	cases := []struct {
		in   logInputs
		name string
	}{
		{logInputs{Level: "debug"}, "log_level"},
		{logInputs{Scope: "reads"}, "log_scope"},
		{logInputs{Mode: "strict", Level: "debug"}, "log_level"},
	}
	for _, c := range cases {
		_, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, c.in)
		if err == nil || !strings.Contains(err.Error(), c.name) {
			t.Errorf("%+v: error = %v; want one naming %s (an invalid value is an error even when strict would override it)", c.in, err, c.name)
		}
	}
}

func TestLV_EnvNames(t *testing.T) {
	env := map[string]string{"FEDIT_LOG_LEVEL": "warning", "FEDIT_LOG_SCOPE": "writes"}
	got := logInputsFromEnv(lcxEnv(env))
	if got.Level != "warning" || got.Scope != "writes" {
		t.Errorf("logInputsFromEnv = %+v; want level warning and scope writes", got)
	}
}

func TestLV_FlagBeatsEnv(t *testing.T) {
	_, env := lcxBase(t)
	env["FEDIT_LOG_LEVEL"] = "error"
	env["FEDIT_LOG_SCOPE"] = "writes"
	g := lcxEnv(env)
	s, err := resolveLogSettings(g, "", lcxNoConfig, logInputs{Mode: "permissive", Level: "warning"}, logInputsFromEnv(g))
	if err != nil || s.Level != logThresholdWarning || s.Scope != logScopeWrites {
		t.Errorf("flag over env = level %q, scope %q, %v; want warning (flag) and writes (env fills the gap)", s.Level, s.Scope, err)
	}
}

func TestLV_StrictForcesAll(t *testing.T) {
	_, env := lcxBase(t)
	s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, logInputs{Mode: "strict", Level: "error", Scope: "writes"})
	if err != nil {
		t.Fatalf("strict with a level must not be an error: %v", err)
	}
	if s.Level != logThresholdAll {
		t.Errorf("strict must force the level to all, got %q", s.Level)
	}
	if s.Scope != logScopeWrites {
		t.Errorf("strict must leave the scope alone, got %q", s.Scope)
	}
}

func TestLV_StrictFromFlagBeatsEnvLevel(t *testing.T) {
	_, env := lcxBase(t)
	env["FEDIT_LOG_LEVEL"] = "warning"
	g := lcxEnv(env)
	s, err := resolveLogSettings(g, "", lcxNoConfig, logInputs{Mode: "strict"}, logInputsFromEnv(g))
	if err != nil || s.Level != logThresholdAll {
		t.Errorf("strict from a flag with level warning in the environment = %q, %v; want all", s.Level, err)
	}
}

func TestLV_PermissiveAndOffKeepLevel(t *testing.T) {
	_, env := lcxBase(t)
	for _, mode := range []string{"permissive", "off"} {
		s, err := resolveLogSettings(lcxEnv(env), "", lcxNoConfig, logInputs{Mode: mode, Level: "warning"})
		if err != nil || s.Level != logThresholdWarning {
			t.Errorf("mode %s keeps the level: got %q, %v; want warning", mode, s.Level, err)
		}
	}
}
