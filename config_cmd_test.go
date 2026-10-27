package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Tests of the -config command (configCommand and cfgWriteFile). Helpers use
// the prefix cfc; the binary-level cases are in config_cmd_bin_test.go.

type cfcFix struct {
	t   *testing.T
	dir string
	ini string
	env map[string]string
}

// cfcSetup makes a fixture whose settings file is <dir>/cfg/fedit.ini (the
// folder does not exist yet). A non-empty iniText is written to it first.
func cfcSetup(t *testing.T, iniText string) *cfcFix {
	t.Helper()
	dir := t.TempDir()
	f := &cfcFix{t: t, dir: dir, ini: filepath.Join(dir, "cfg", "fedit.ini"), env: map[string]string{}}
	f.env["FEDIT_CONFIG"] = f.ini
	f.env["LOCALAPPDATA"] = filepath.Join(dir, "la")
	if iniText != "" {
		if err := os.MkdirAll(filepath.Dir(f.ini), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f.ini, []byte(iniText), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

func (f *cfcFix) run(args ...string) (stdout, stderr string, code int) {
	f.t.Helper()
	var so, se strings.Builder
	ucd := func() (string, error) { return filepath.Join(f.dir, "ucd"), nil }
	code = configCommand(args, func(k string) string { return f.env[k] }, "", ucd, &so, &se)
	return so.String(), se.String(), code
}

// read returns the settings file and whether it exists.
func (f *cfcFix) read() (string, bool) {
	f.t.Helper()
	b, err := os.ReadFile(f.ini)
	if errors.Is(err, os.ErrNotExist) {
		return "", false
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return string(b), true
}

func cfcLines(s string) []string {
	return strings.Split(strings.TrimRight(s, "\n"), "\n")
}

func TestCFC_Usage(t *testing.T) {
	for _, args := range [][]string{
		nil, {"bogus"}, {"path", "x"}, {"show", "x"}, {"get"}, {"get", "a", "b"},
		{"set", "log_mode"}, {"set", "a", "b", "c"}, {"unset"}, {"unset", "a", "b"},
	} {
		f := cfcSetup(t, "")
		so, se, code := f.run(args...)
		if code != 1 || so != "" || !strings.Contains(se, "usage: fedit -config") {
			t.Errorf("args %q: code %d, stdout %q, stderr %q; want exit 1, usage on stderr", args, code, so, se)
		}
		if _, ok := f.read(); ok {
			t.Errorf("args %q: a usage error must not create the settings file", args)
		}
	}
}

func TestCFC_Path(t *testing.T) {
	f := cfcSetup(t, "log_mode = loud\n") // path works even when the file is broken
	so, _, code := f.run("path")
	if code != 0 || so != f.ini+"\n" {
		t.Errorf("FEDIT_CONFIG: code %d, stdout %q, want %q", code, so, f.ini+"\n")
	}
	delete(f.env, "FEDIT_CONFIG")
	so, _, code = f.run("path")
	want := filepath.Join(f.dir, "ucd", "fedit", "fedit.ini") + "\n"
	if code != 0 || so != want {
		t.Errorf("default location: code %d, stdout %q, want %q", code, so, want)
	}
	var o, e strings.Builder
	none := func() (string, error) { return "", errors.New("no config dir") }
	code = configCommand([]string{"path"}, func(string) string { return "" }, "", none, &o, &e)
	if code != 1 || o.String() != "" || !strings.Contains(e.String(), "settings file") {
		t.Errorf("no location: code %d, stdout %q, stderr %q; want exit 1 and a message about the settings file", code, o.String(), e.String())
	}
}

func TestCFC_ShowDefaults(t *testing.T) {
	f := cfcSetup(t, "")
	so, se, code := f.run("show")
	if code != 0 || se != "" {
		t.Fatalf("code %d, stderr %q", code, se)
	}
	want := []string{
		"log_mode = off (default)",
		"log_file = " + filepath.Join(f.dir, "la", "fedit", "fedit.log") + " (default)",
		"log_max_mb = 10 (default)",
		"log_content = off (default)",
		"log_level = all (default)",
		"log_scope = all (default)",
	}
	got := cfcLines(so)
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want[i])
		}
	}
}

func TestCFC_ShowSources(t *testing.T) {
	f := cfcSetup(t, "# top\nlog_mode = permissive\nlog_level = warning\nLOG_FILE = x.log\nbogus = 1\n")
	f.env["FEDIT_LOG_LEVEL"] = "error"
	f.env["FEDIT_LOG_SCOPE"] = " writes "
	so, se, code := f.run("show")
	if code != 0 {
		t.Fatalf("code %d, stderr %q", code, se)
	}
	if !strings.Contains(se, "unknown key") || !strings.Contains(se, f.ini) {
		t.Errorf("an unknown key is a warning that names the file; stderr %q", se)
	}
	want := []string{
		"log_mode = permissive (ini)",
		"log_file = x.log (ini)",
		"log_max_mb = 10 (default)",
		"log_content = off (default)",
		"log_level = error (environment)",
		"log_scope = writes (environment)",
	}
	got := cfcLines(so)
	if len(got) != len(want) {
		t.Fatalf("got %d lines %q, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want[i])
		}
	}
}

func TestCFC_ShowErrors(t *testing.T) {
	f := cfcSetup(t, "")
	f.env["FEDIT_LOG_MODE"] = "loud"
	so, se, code := f.run("show")
	if code != 1 || so != "" || !strings.Contains(se, "log_mode") || !strings.Contains(se, "loud") {
		t.Errorf("invalid environment value: code %d, stdout %q, stderr %q", code, so, se)
	}
	f = cfcSetup(t, "# c\nlog_mode = loud\n")
	so, se, code = f.run("show")
	if code != 1 || so != "" || !strings.Contains(se, f.ini) || !strings.Contains(se, "line 2") {
		t.Errorf("broken settings file: code %d, stdout %q, stderr %q; want exit 1 naming the file and line 2", code, so, se)
	}
}

func TestCFC_EnvEnumCase(t *testing.T) {
	for _, c := range []struct{ env, want string }{
		{"Strict", "strict"},
		{" PERMISSIVE ", "permissive"},
	} {
		f := cfcSetup(t, "")
		f.env["FEDIT_LOG_MODE"] = c.env
		so, se, code := f.run("get", "log_mode")
		if code != 0 || so != c.want+"\n" {
			t.Errorf("get with FEDIT_LOG_MODE=%q: code %d, stdout %q, stderr %q; want %q", c.env, code, so, se, c.want+"\n")
		}
		so, se, code = f.run("show")
		if code != 0 {
			t.Fatalf("show with FEDIT_LOG_MODE=%q: code %d, stderr %q", c.env, code, se)
		}
		line := "log_mode = " + c.want + " (environment)"
		found := false
		for _, l := range cfcLines(so) {
			if l == line {
				found = true
			}
		}
		if !found {
			t.Errorf("show with FEDIT_LOG_MODE=%q: no line %q in %q", c.env, line, so)
		}
	}
}

func TestCFC_Get(t *testing.T) {
	cases := []struct {
		name string
		ini  string
		env  map[string]string
		args []string
		want string
	}{
		{"default", "", nil, []string{"get", "log_mode"}, "off\n"},
		{"case-insensitive key", "", nil, []string{"get", "LOG_LEVEL"}, "all\n"},
		{"default number", "", nil, []string{"get", "log_max_mb"}, "10\n"},
		{"ini value", "log_mode = strict\n", nil, []string{"get", "log_mode"}, "strict\n"},
		{"environment beats ini", "log_mode = strict\n", map[string]string{"FEDIT_LOG_MODE": "permissive"}, []string{"get", "log_mode"}, "permissive\n"},
	}
	for _, c := range cases {
		f := cfcSetup(t, c.ini)
		for k, v := range c.env {
			f.env[k] = v
		}
		so, se, code := f.run(c.args...)
		if code != 0 || so != c.want {
			t.Errorf("%s: code %d, stdout %q, stderr %q; want %q", c.name, code, so, se, c.want)
		}
	}
	f := cfcSetup(t, "")
	so, se, code := f.run("get", "nope")
	if code != 1 || so != "" || !strings.Contains(se, "unknown key") || !strings.Contains(se, "nope") {
		t.Errorf("unknown key: code %d, stdout %q, stderr %q", code, so, se)
	}
	f = cfcSetup(t, "log_mode = loud\n")
	if _, _, code = f.run("get", "log_mode"); code != 1 {
		t.Errorf("a broken settings file is an error for get, got exit %d", code)
	}
}

func TestCFC_SetCreates(t *testing.T) {
	f := cfcSetup(t, "")
	so, se, code := f.run("set", "log_mode", "permissive")
	if code != 0 || se != "" {
		t.Fatalf("code %d, stderr %q", code, se)
	}
	if !strings.Contains(so, f.ini) || !strings.Contains(so, "log_mode = permissive") {
		t.Errorf("stdout %q should name the file and the setting", so)
	}
	got, ok := f.read()
	if !ok || got != "log_mode = permissive\n" {
		t.Errorf("settings file = %q (exists %v), want %q", got, ok, "log_mode = permissive\n")
	}
}

func TestCFC_SetKeepsText(t *testing.T) {
	orig := "\ufeff# my settings\r\nlog_mode = off\r\n; note\r\nlog_level = all\r\n"
	f := cfcSetup(t, orig)
	if _, se, code := f.run("set", "log_mode", "strict"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	want := "\ufeff# my settings\r\nlog_mode = strict\r\n; note\r\nlog_level = all\r\n"
	if got, _ := f.read(); got != want {
		t.Errorf("after replacing a value: %q, want %q", got, want)
	}
	if _, se, code := f.run("set", "LOG_SCOPE", "writes"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	want += "log_scope = writes\r\n"
	if got, _ := f.read(); got != want {
		t.Errorf("after adding a key: %q, want %q", got, want)
	}
}

func TestCFC_SetInvalid(t *testing.T) {
	bad := [][]string{
		{"log_mode", "loud"}, {"log_max_mb", "-1"}, {"log_max_mb", "abc"},
		{"log_content", "maybe"}, {"log_file", ""}, {"nope", "x"},
	}
	for _, kv := range bad {
		f := cfcSetup(t, "")
		so, se, code := f.run("set", kv[0], kv[1])
		if code != 1 || so != "" || se == "" {
			t.Errorf("set %q: code %d, stdout %q, stderr %q; want exit 1 with a message", kv, code, so, se)
		}
		if _, ok := f.read(); ok {
			t.Errorf("set %q: a rejected value must not create the file", kv)
		}
		f = cfcSetup(t, "log_mode = off\n")
		if _, _, code = f.run("set", kv[0], kv[1]); code != 1 {
			t.Errorf("set %q on an existing file: exit %d, want 1", kv, code)
		}
		if got, _ := f.read(); got != "log_mode = off\n" {
			t.Errorf("set %q: the file changed to %q", kv, got)
		}
	}
}

func TestCFC_BrokenIniBlocksEdits(t *testing.T) {
	const orig = "log_mode = loud\n"
	for _, args := range [][]string{{"set", "log_level", "warning"}, {"unset", "log_mode"}} {
		f := cfcSetup(t, orig)
		so, se, code := f.run(args...)
		if code != 1 || so != "" || !strings.Contains(se, "line 1") {
			t.Errorf("%q: code %d, stdout %q, stderr %q; want exit 1 naming line 1", args, code, so, se)
		}
		if got, _ := f.read(); got != orig {
			t.Errorf("%q: the broken file was changed to %q", args, got)
		}
	}
}

func TestCFC_UnsetRemoves(t *testing.T) {
	f := cfcSetup(t, "# c\nlog_mode = strict\nlog_level = all\nlog_mode = off\n")
	so, se, code := f.run("unset", "log_mode")
	if code != 0 || se != "" || !strings.Contains(so, "log_mode") || !strings.Contains(so, f.ini) {
		t.Fatalf("code %d, stdout %q, stderr %q", code, so, se)
	}
	if got, _ := f.read(); got != "# c\nlog_level = all\n" {
		t.Errorf("settings file = %q", got)
	}
}

func TestCFC_UnsetAbsent(t *testing.T) {
	f := cfcSetup(t, "log_level = all\n")
	so, _, code := f.run("unset", "log_mode")
	if code != 0 || !strings.Contains(so, "not set") {
		t.Errorf("code %d, stdout %q; want exit 0 and a note that the key is not set", code, so)
	}
	if got, _ := f.read(); got != "log_level = all\n" {
		t.Errorf("the file changed to %q", got)
	}
	f = cfcSetup(t, "")
	if _, _, code = f.run("unset", "log_mode"); code != 0 {
		t.Errorf("no file: exit %d, want 0", code)
	}
	if _, ok := f.read(); ok {
		t.Errorf("unset must not create the settings file")
	}
}

func TestCFC_UnsetUnknown(t *testing.T) {
	f := cfcSetup(t, "log_mode = off\n")
	so, se, code := f.run("unset", "nope")
	if code != 1 || so != "" || !strings.Contains(se, "unknown key") {
		t.Errorf("code %d, stdout %q, stderr %q", code, so, se)
	}
}

func TestCFC_WriteFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a", "b", "x.ini")
	if err := cfcWriteTwice(p); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "two\n" {
		t.Errorf("content %q, err %v; want the second write", b, err)
	}
	entries, err := os.ReadDir(filepath.Dir(p))
	if err != nil || len(entries) != 1 {
		t.Errorf("the folder should hold only the settings file, got %v (err %v)", entries, err)
	}
}

func cfcWriteTwice(p string) error {
	if err := cfgWriteFile(p, "one\r\n"); err != nil {
		return err
	}
	if b, err := os.ReadFile(p); err != nil || string(b) != "one\r\n" {
		return errors.New("first write not exact")
	}
	return cfgWriteFile(p, "two\n")
}

func TestCFC_WriteFileError(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "x.ini")
	if err := os.Mkdir(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cfgWriteFile(p, "a\n"); err == nil {
		t.Fatal("writing onto a folder must fail")
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Errorf("a failed write must leave no temp file, got %v (err %v)", entries, err)
	}
}

func TestCFC_Ops(t *testing.T) {
	for _, op := range []string{"config-set", "config-unset"} {
		if !logOpKnown(op) || !logOpIsWrite(op) {
			t.Errorf("%s must be a known write operation", op)
		}
	}
	for _, op := range []string{"config-path", "config-show", "config-get", "config"} {
		if logOpKnown(op) {
			t.Errorf("%s is not logged and must not be a known operation", op)
		}
	}
}
