package main

import (
	"strings"
	"testing"
)

// Tests of the ini key registry and its pure text functions (config.go).

func TestCfg_Registry(t *testing.T) {
	want := map[string][2]string{
		"log_mode":    {"FEDIT_LOG_MODE", "-log-mode"},
		"log_file":    {"FEDIT_LOG_FILE", "-log-file"},
		"log_max_mb":  {"FEDIT_LOG_MAX_MB", "-log-max-mb"},
		"log_content": {"FEDIT_LOG_CONTENT", "-log-content"},
		"log_level":   {"FEDIT_LOG_LEVEL", "-log-level"},
		"log_scope":   {"FEDIT_LOG_SCOPE", "-log-scope"},
	}
	for name, ef := range want {
		k, ok := cfgFind(name)
		if !ok {
			t.Errorf("key %s is not in the registry", name)
			continue
		}
		if k.Env != ef[0] || k.Flag != ef[1] || !k.MCP {
			t.Errorf("%s: env %q flag %q mcp %v; want %q %q true", name, k.Env, k.Flag, k.MCP, ef[0], ef[1])
		}
	}
	if _, ok := cfgFind("  Log_Mode "); !ok {
		t.Errorf("key names are matched case-insensitively and trimmed")
	}
	if _, ok := cfgFind("no_such_key"); ok {
		t.Errorf("an unknown key must not be found")
	}
}

func TestCfg_Check(t *testing.T) {
	good := [][2]string{
		{"log_mode", "off"}, {"log_mode", "permissive"}, {"log_mode", "strict"},
		{"log_file", "~/x.log"}, {"log_max_mb", "0"}, {"log_max_mb", "25"},
		{"log_content", "on"}, {"log_level", "warning"}, {"log_scope", "writes"},
		{"LOG_MODE", " strict "},
	}
	for _, c := range good {
		if err := cfgCheck(c[0], c[1]); err != nil {
			t.Errorf("%s = %q should be valid: %v", c[0], c[1], err)
		}
	}
	bad := [][2]string{
		{"log_mode", "loud"}, {"log_mode", ""}, {"log_file", "  "}, {"log_max_mb", "-1"},
		{"log_max_mb", "ten"}, {"log_content", "yes"}, {"log_level", "debug"},
		{"log_scope", "reads"}, {"nope", "x"}, {"log_file", "a\nb"},
	}
	for _, c := range bad {
		if err := cfgCheck(c[0], c[1]); err == nil {
			t.Errorf("%s = %q should be rejected", c[0], c[1])
		}
	}
}

func TestCfg_Parse(t *testing.T) {
	text := "# fedit settings\r\n; another comment\r\n\r\nlog_mode = strict\r\nLog_File=~/fedit.log\r\nmystery = 1\r\nlog_max_mb = 5\r\nlog_max_mb = 7\r\n"
	vals, warns, err := cfgParse(text)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if vals["log_mode"] != "strict" || vals["log_file"] != "~/fedit.log" {
		t.Errorf("values: %v", vals)
	}
	if vals["log_max_mb"] != "7" {
		t.Errorf("the last of a repeated key wins: %v", vals)
	}
	if _, ok := vals["mystery"]; ok {
		t.Errorf("an unknown key must not be returned: %v", vals)
	}
	if len(warns) != 1 || !strings.Contains(warns[0], "mystery") || !strings.Contains(warns[0], "line 6") {
		t.Errorf("want one warning naming the unknown key and its line: %q", warns)
	}
}

func TestCfg_ParseErrors(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"log_mode = loud\n", "line 1"},
		{"# c\nlog_level = debug\n", "line 2"},
		{"just words\n", "line 1"},
		{"= value\n", "line 1"},
	} {
		if _, _, err := cfgParse(c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("cfgParse(%q): error %v; want one naming %s", c.text, err, c.want)
		}
	}
	if vals, warns, err := cfgParse(""); err != nil || len(vals) != 0 || len(warns) != 0 {
		t.Errorf("an empty text is valid and empty: %v %q %v", vals, warns, err)
	}
}

func TestCfg_SetReplacesInPlace(t *testing.T) {
	text := "# top\nlog_mode = off\n\n# keep me\nlog_file = a.log\n"
	got, err := cfgSet(text, "LOG_MODE", " strict ")
	if err != nil {
		t.Fatal(err)
	}
	if want := "# top\nlog_mode = strict\n\n# keep me\nlog_file = a.log\n"; got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestCfg_SetRepeatedKeyChangesTheLast(t *testing.T) {
	got, err := cfgSet("log_max_mb = 5\nlog_max_mb = 7\n", "log_max_mb", "9")
	if err != nil {
		t.Fatal(err)
	}
	if want := "log_max_mb = 5\nlog_max_mb = 9\n"; got != want {
		t.Errorf("got %q; want %q", got, want)
	}
}

func TestCfg_SetAppends(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"", "log_scope = writes\n"},
		{"# c\n", "# c\nlog_scope = writes\n"},
		{"# c", "# c\nlog_scope = writes\n"},
		{"# c\r\nlog_mode = off\r\n", "# c\r\nlog_mode = off\r\nlog_scope = writes\r\n"},
		{"# c\r\nlog_mode = off", "# c\r\nlog_mode = off\r\nlog_scope = writes\r\n"},
	} {
		got, err := cfgSet(c.in, "log_scope", "writes")
		if err != nil || got != c.want {
			t.Errorf("cfgSet(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
}

func TestCfg_SetKeepsCRLF(t *testing.T) {
	got, err := cfgSet("log_mode = off\r\nlog_file = a\r\n", "log_mode", "strict")
	if err != nil || got != "log_mode = strict\r\nlog_file = a\r\n" {
		t.Errorf("got %q, %v", got, err)
	}
}

func TestCfg_SetInvalidChangesNothing(t *testing.T) {
	text := "log_mode = off\n"
	for _, c := range [][2]string{{"log_mode", "loud"}, {"nope", "x"}, {"log_file", "a\nb"}} {
		got, err := cfgSet(text, c[0], c[1])
		if err == nil || got != text {
			t.Errorf("cfgSet(%s, %q) = %q, %v; want the text unchanged and an error", c[0], c[1], got, err)
		}
	}
}

func TestCfg_Unset(t *testing.T) {
	got, found := cfgUnset("# c\nlog_mode = off\nlog_file = a\nlog_mode = strict\n", "Log_Mode")
	if !found || got != "# c\nlog_file = a\n" {
		t.Errorf("got %q, %v; want every line of the key removed", got, found)
	}
	text := "# c\nlog_file = a\n"
	if got, found := cfgUnset(text, "log_mode"); found || got != text {
		t.Errorf("an absent key leaves the text alone: %q, %v", got, found)
	}
}

func TestCfg_LogInputs(t *testing.T) {
	in := cfgLogInputs(map[string]string{
		"log_mode": "strict", "log_file": "f.log", "log_max_mb": "3",
		"log_content": "on", "log_level": "error", "log_scope": "writes",
	})
	want := logInputs{Mode: "strict", File: "f.log", MaxMB: "3", Content: "on", Level: "error", Scope: "writes"}
	if in != want {
		t.Errorf("got %+v; want %+v", in, want)
	}
	if (cfgLogInputs(nil) != logInputs{}) {
		t.Errorf("no values give an empty layer")
	}
}
