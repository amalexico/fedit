package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Binary-level tests of the settings file (fedit.ini): it is the layer between
// the environment and the defaults, a broken one is an error, and -noconfig
// skips it. The subtests are listed in TestLW_Bin, which builds the binary once.

// TestMain points FEDIT_CONFIG at a file that does not exist, for the whole
// test process and the binaries it starts, so no test ever reads the real
// settings file of the person running the suite. A test that wants an ini
// sets FEDIT_CONFIG itself.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "fedit-cfg-")
	if err == nil {
		os.Setenv("FEDIT_CONFIG", filepath.Join(dir, "none.ini"))
	}
	code := m.Run()
	if err == nil {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

// cwxIni writes an ini in dir and points FEDIT_CONFIG at it for this test.
func cwxIni(t *testing.T, dir, text string) string {
	t.Helper()
	p := orxWrite(t, dir, "fedit.ini", text)
	t.Setenv("FEDIT_CONFIG", p)
	return p
}

// cwxShow runs a read-only operation, which is logged like any other.
func cwxShow(t *testing.T, bin, f string, extra ...string) (string, int) {
	t.Helper()
	args := append([]string{"-file", f, "-op", "show"}, extra...)
	_, se, code := orxRun(t, bin, args...)
	return se, code
}

func cwxCount(t *testing.T, path string) int {
	t.Helper()
	return len(lwxLog(t, path))
}

func cwxBinIniPrecedence(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	a, b, c := filepath.Join(dir, "a.log"), filepath.Join(dir, "b.log"), filepath.Join(dir, "c.log")
	cwxIni(t, dir, "log_mode = permissive\nlog_file = "+a+"\n")

	if se, code := cwxShow(t, bin, f); code != 0 {
		t.Fatalf("exit %d: %s", code, se)
	}
	if n := cwxCount(t, a); n != 2 {
		t.Errorf("the ini alone turns logging on: %d lines in %s, want 2", n, a)
	}

	t.Setenv("FEDIT_LOG_FILE", b)
	cwxShow(t, bin, f)
	if cwxCount(t, b) != 2 || cwxCount(t, a) != 2 {
		t.Errorf("the environment beats the ini: a=%d b=%d, want 2 and 2", cwxCount(t, a), cwxCount(t, b))
	}

	cwxShow(t, bin, f, "-log-file", c)
	if cwxCount(t, c) != 2 || cwxCount(t, b) != 2 || cwxCount(t, a) != 2 {
		t.Errorf("a flag beats the environment: a=%d b=%d c=%d, want 2 each", cwxCount(t, a), cwxCount(t, b), cwxCount(t, c))
	}

	t.Setenv("FEDIT_LOG_MODE", "off")
	cwxShow(t, bin, f)
	if cwxCount(t, a) != 2 || cwxCount(t, b) != 2 {
		t.Errorf("FEDIT_LOG_MODE=off beats log_mode in the ini: a=%d b=%d, want 2 each", cwxCount(t, a), cwxCount(t, b))
	}
	cwxShow(t, bin, f, "-log-mode", "permissive", "-log-file", c)
	if n := cwxCount(t, c); n != 4 {
		t.Errorf("-log-mode beats the environment: %d lines in %s, want 4", n, c)
	}
}

func cwxBinIniInvalid(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	p := cwxIni(t, dir, "log_mode = permissive\nlog_level = debug\n")
	se, code := cwxShow(t, bin, f)
	if code != 1 || !strings.Contains(se, p) || !strings.Contains(se, "line 2") {
		t.Errorf("a broken ini exits 1 naming the file and line: exit %d, stderr %q", code, se)
	}
	_, se, code = orxRun(t, bin, "-log", "merge")
	if code != 1 || !strings.Contains(se, p) {
		t.Errorf("-log merge reads the ini too: exit %d, stderr %q", code, se)
	}
}

func cwxBinIniNoConfig(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	cwxIni(t, dir, "log_level = debug\n")
	if se, code := cwxShow(t, bin, f, "-noconfig"); code != 0 {
		t.Errorf("-noconfig must skip a broken ini: exit %d, stderr %q", code, se)
	}
	a := filepath.Join(dir, "a.log")
	cwxIni(t, dir, "log_mode = permissive\nlog_file = "+a+"\n")
	if se, code := cwxShow(t, bin, f, "-noconfig"); code != 0 || cwxCount(t, a) != 0 {
		t.Errorf("-noconfig must skip a valid ini: exit %d, %d lines logged, stderr %q", code, cwxCount(t, a), se)
	}
}

func cwxBinIniUnknownKey(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	a := filepath.Join(dir, "a.log")
	cwxIni(t, dir, "mystery = 1\nlog_mode = permissive\nlog_file = "+a+"\n")
	se, code := cwxShow(t, bin, f)
	if code != 0 || !strings.Contains(se, "mystery") {
		t.Errorf("an unknown key is a warning, not an error: exit %d, stderr %q", code, se)
	}
	if n := cwxCount(t, a); n != 2 {
		t.Errorf("the other keys still apply: %d lines in %s, want 2", n, a)
	}
}

func cwxBinIniUsage(t *testing.T, bin string) {
	lwxSetup(t)
	_, se, code := orxRun(t, bin)
	if code != 1 || !strings.Contains(se, "-noconfig") || !strings.Contains(se, "FEDIT_CONFIG") {
		t.Errorf("the usage text must mention -noconfig and FEDIT_CONFIG: exit %d", code)
	}
}
