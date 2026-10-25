package main

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// Wiring slice A: the logging session hooked into main() and exitNow. The
// unit tests cover the pure helpers; the TestLW_Bin* tests run the real
// binary (clhBuild, orxRun and the other helpers come from the wire tests)
// against a temp log. Every run points LOCALAPPDATA at a temp folder so no
// real log or spool is touched.

const lwxOrig = "alpha\nbeta\nalpha\n"

func lwxEnv(t *testing.T) string {
	t.Helper()
	la := filepath.Join(t.TempDir(), "la")
	t.Setenv("LOCALAPPDATA", la)
	for _, k := range []string{"FEDIT_LOG_MODE", "FEDIT_LOG_FILE", "FEDIT_LOG_MAX_MB", "FEDIT_LOG_CONTENT", "FEDIT_LOG_LEVEL", "FEDIT_LOG_SCOPE"} {
		t.Setenv(k, "")
	}
	return la
}

func lwxLog(t *testing.T, path string) []string {
	t.Helper()
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	s := strings.TrimRight(orxNorm(string(b)), "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func lwxKeys(kvs []logKV) []string {
	var k []string
	for _, kv := range kvs {
		k = append(k, kv.Key)
	}
	return k
}

func lwxVal(kvs []logKV, key string) (string, bool) {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.Value, true
		}
	}
	return "", false
}

// ---- unit tests ----

func TestLW_ExitConstants(t *testing.T) {
	if exitLogRefused != 4 || exitLogGap != 5 {
		t.Errorf("exit codes = %d and %d; want 4 (strict: begin record not written, nothing changed) and 5 (strict: done, end record not written)", exitLogRefused, exitLogGap)
	}
}

func TestLW_FinishCode(t *testing.T) {
	boom := errors.New("locked")
	cases := []struct {
		code   int
		err    error
		strict bool
		want   int
	}{
		{0, nil, true, 0},
		{0, boom, true, 5},
		{1, boom, true, 1},
		{3, boom, true, 3},
		{0, boom, false, 0},
		{2, nil, true, 2},
	}
	for _, c := range cases {
		if got := logFinishCode(c.code, c.err, c.strict); got != c.want {
			t.Errorf("logFinishCode(%d, %v, strict=%v) = %d; want %d", c.code, c.err, c.strict, got, c.want)
		}
	}
}

func TestLW_FieldsCore(t *testing.T) {
	cwd := t.TempDir()
	kvs := logCLIFields(logCLIInput{
		Source: "cli", User: `DOM\alex`, Host: "H1", Pid: 42, Cwd: cwd,
		File: "a.txt, sub/b.txt", Dest: "d.txt",
	})
	wantKeys := []string{"source", "user", "host", "pid", "cwd", "file", "dest"}
	if got := lwxKeys(kvs); strings.Join(got, ",") != strings.Join(wantKeys, ",") {
		t.Fatalf("keys = %v; want %v", got, wantKeys)
	}
	want := map[string]string{
		"source": "cli",
		"user":   `DOM\alex`,
		"host":   "H1",
		"pid":    "42",
		"cwd":    cwd,
		"file":   filepath.Join(cwd, "a.txt") + "," + filepath.Join(cwd, "sub", "b.txt"),
		"dest":   filepath.Join(cwd, "d.txt"),
	}
	for k, w := range want {
		if v, _ := lwxVal(kvs, k); v != w {
			t.Errorf("%s = %q; want %q", k, v, w)
		}
	}
}

func TestLW_FieldsAbsoluteKept(t *testing.T) {
	cwd := t.TempDir()
	abs := filepath.Join(t.TempDir(), "x.txt")
	kvs := logCLIFields(logCLIInput{Source: "guided", Cwd: cwd, File: abs, Files: "*.go"})
	if v, _ := lwxVal(kvs, "file"); v != abs {
		t.Errorf("an absolute path must stay as it is: %q; want %q", v, abs)
	}
	if v, _ := lwxVal(kvs, "files"); v != filepath.Join(cwd, "*.go") {
		t.Errorf("a glob keeps its pattern, made absolute: %q", v)
	}
	if v, _ := lwxVal(kvs, "source"); v != "guided" {
		t.Errorf("source = %q; want guided", v)
	}
}

func TestLW_FieldsUnsetOmitted(t *testing.T) {
	kvs := logCLIFields(logCLIInput{Source: "cli", Cwd: t.TempDir()})
	for _, k := range []string{"file", "files", "dest", "text", "text_len", "text_sha256", "textfile"} {
		if _, ok := lwxVal(kvs, k); ok {
			t.Errorf("%s must be left out when it was not given", k)
		}
	}
}

func TestLW_FieldsText(t *testing.T) {
	cwd := t.TempDir()
	wantLen, wantSum := "", ""
	for _, kv := range logTextFields("text", []byte("gamma")) {
		if kv.Key == "text_len" {
			wantLen = kv.Value
		} else {
			wantSum = kv.Value
		}
	}

	kvs := logCLIFields(logCLIInput{Source: "cli", Cwd: cwd, Text: "gamma"})
	if v, _ := lwxVal(kvs, "text_len"); v != wantLen {
		t.Errorf("text_len = %q; want %q", v, wantLen)
	}
	if v, _ := lwxVal(kvs, "text_sha256"); v != wantSum {
		t.Errorf("text_sha256 = %q; want %q", v, wantSum)
	}
	if _, ok := lwxVal(kvs, "text"); ok {
		t.Errorf("the text itself must not be logged unless log_content is on")
	}

	kvs = logCLIFields(logCLIInput{Source: "cli", Cwd: cwd, Text: "gamma", Content: true})
	if v, _ := lwxVal(kvs, "text"); v != "gamma" {
		t.Errorf("with log_content on the text is logged as given: %q", v)
	}
	if _, ok := lwxVal(kvs, "text_sha256"); ok {
		t.Errorf("with log_content on there is no hash field")
	}

	kvs = logCLIFields(logCLIInput{Source: "cli", Cwd: cwd, TextHex: "67616d6d61"})
	if v, _ := lwxVal(kvs, "text_sha256"); v != wantSum {
		t.Errorf("-texthex is hashed after decoding: %q; want %q", v, wantSum)
	}

	kvs = logCLIFields(logCLIInput{Source: "cli", Cwd: cwd, TextHex: "zz"})
	if _, ok := lwxVal(kvs, "text_len"); !ok {
		t.Errorf("an invalid -texthex must not panic and still be recorded")
	}

	kvs = logCLIFields(logCLIInput{Source: "cli", Cwd: cwd, TextFile: "t.txt"})
	if v, _ := lwxVal(kvs, "textfile"); v != filepath.Join(cwd, "t.txt") {
		t.Errorf("textfile = %q; want the absolute path", v)
	}
}

// ---- binary-level tests ----

// TestLW_Bin builds the binary once and runs every binary-level case as a
// subtest against it. clhBuild runs go build on each call, which takes many
// seconds on Windows, so calling it from every case made the suite much slower.
func TestLW_Bin(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name string
		run  func(*testing.T, string)
	}{
		{"OffByDefault", lwxBinOffByDefault},
		{"PermissiveEnv", lwxBinPermissiveEnv},
		{"ContentOn", lwxBinContentOn},
		{"FlagOverridesEnv", lwxBinFlagOverridesEnv},
		{"ReadOnlyOpsAreLogged", lwxBinReadOnlyOpsAreLogged},
		{"ExitCodeInEndRecord", lwxBinExitCodeInEndRecord},
		{"LevelWarning", lwxBinLevelWarning},
		{"StrictWritesBoth", lwxBinStrictWritesBoth},
		{"StrictRefuses", lwxBinStrictRefuses},
		{"PermissiveSpoolsThenMerges", lwxBinPermissiveSpoolsThenMerges},
		{"InvalidSettingIsError", lwxBinInvalidSettingIsError},
		{"OutputResultAndLog", lwxBinOutputResultAndLog},
		{"UsageIsNotAnOperation", lwxBinUsageIsNotAnOperation},
		{"WriteOpHashes", lwxBinWriteOpHashes},
		{"ReadOpNoHashes", lwxBinReadOpNoHashes},
		{"ReplaceAllResult", lwxBinReplaceAllResult},
		{"MultiFileResult", lwxBinMultiFileResult},
		{"LogMerge", lwxBinLogMerge},
		{"IniLayerAndPrecedence", cwxBinIniPrecedence},
		{"IniInvalidIsError", cwxBinIniInvalid},
		{"IniNoConfigFlag", cwxBinIniNoConfig},
		{"IniUnknownKeyWarns", cwxBinIniUnknownKey},
		{"IniUsage", cwxBinIniUsage},
		{"ConfigPathShowGet", cfcBinPathShowGet},
		{"ConfigSetUnset", cfcBinSetUnset},
		{"ConfigRejects", cfcBinRejects},
		{"ConfigBrokenIni", cfcBinBrokenIni},
		{"ConfigSetIsLogged", cfcBinSetIsLogged},
		{"DryReplaceAll", drbBinReplaceAll},
		{"DryInsertAfter", drbBinInsertAfter},
		{"DryWriteNewFile", drbBinWriteNewFile},
		{"DryNoMatchExit", drbBinNoMatchExit},
		{"DryOffNoNote", drbBinOffNoNote},
		{"DryLogged", drbBinDryLogged},
		{"DryConflict", drbBinDryConflict},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, bin) })
	}
}

func lwxSetup(t *testing.T) (dir, f string) {
	t.Helper()
	lwxEnv(t)
	dir = t.TempDir()
	f = orxWrite(t, dir, "a.txt", lwxOrig)
	return dir, f
}

func lwxBinOffByDefault(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_FILE", logp)
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if lines := lwxLog(t, logp); len(lines) != 0 {
		t.Errorf("with log_mode unset nothing is logged, got %q", lines)
	}
}

func lwxBinPermissiveEnv(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "audit", "fedit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "zzzword")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %q\nstderr: %q", code, so, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", lines)
	}
	b, e := lines[0], lines[1]
	id, _ := spoolLogField(b, "id")
	if len(id) != 16 {
		t.Fatalf("begin record %q has no 16-character id", b)
	}
	if !strings.Contains(b, "[INFO] id="+id+" phase=begin op=replaceall source=cli") {
		t.Errorf("begin record = %q", b)
	}
	for _, key := range []string{" user=", " host=", " pid=", " cwd="} {
		if !strings.Contains(b, key) {
			t.Errorf("begin record lacks %q: %q", key, b)
		}
	}
	if !strings.Contains(b, " file="+logQuote(f)) {
		t.Errorf("begin record lacks the absolute file path %q: %q", f, b)
	}
	if !strings.Contains(b, " text_len=7 text_sha256=") || strings.Contains(b, "zzzword") {
		t.Errorf("the text must be logged as length and SHA-256 only: %q", b)
	}
	if !strings.Contains(e, "[INFO] id="+id+" phase=end exit=0 ms=") {
		t.Errorf("end record = %q", e)
	}
}

func lwxBinContentOn(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "zzzword",
		"-log-mode", "permissive", "-log-file", logp, "-log-content", "on")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 || !strings.Contains(lines[0], " text=zzzword") || strings.Contains(lines[0], "text_sha256") {
		t.Errorf("with -log-content on the text is logged verbatim: %q", lines)
	}
}

func lwxBinFlagOverridesEnv(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	other := filepath.Join(dir, "other.log")
	good := filepath.Join(dir, "good.log")
	t.Setenv("FEDIT_LOG_MODE", "off")
	t.Setenv("FEDIT_LOG_FILE", other)
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma",
		"-log-mode", "permissive", "-log-file", good)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if n := len(lwxLog(t, good)); n != 2 {
		t.Errorf("the flags must win over the environment: %d records in the flag's log", n)
	}
	if n := len(lwxLog(t, other)); n != 0 {
		t.Errorf("the environment's log must stay untouched: %d records", n)
	}
}

func lwxBinReadOnlyOpsAreLogged(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	_, se, code := orxRun(t, bin, "-file", f, "-op", "find", "-match", "alpha", "-log-mode", "permissive", "-log-file", logp)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 || !strings.Contains(lines[0], "phase=begin op=find source=cli") {
		t.Errorf("a read-only operation is an operation too: %q", lines)
	}

	scoped := filepath.Join(dir, "scoped.log")
	_, se, code = orxRun(t, bin, "-file", f, "-op", "find", "-match", "alpha", "-log-mode", "permissive", "-log-file", scoped, "-log-scope", "writes")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if n := len(lwxLog(t, scoped)); n != 0 {
		t.Errorf("with -log-scope writes a read-only operation is not logged, got %d records", n)
	}
}

func lwxBinExitCodeInEndRecord(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	_, _, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "nomatchword", "-text", "x",
		"-log-mode", "permissive", "-log-file", logp)
	if code == 0 {
		t.Fatalf("a replaceall without a match must not exit 0")
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 {
		t.Fatalf("want two records, got %q", lines)
	}
	level := "[ERROR]"
	if code == 1 {
		level = "[WARNING]"
	}
	if !strings.Contains(lines[1], level+" id=") || !strings.Contains(lines[1], "phase=end exit="+strconv.Itoa(code)+" ") {
		t.Errorf("end record = %q; want level %s and exit=%d", lines[1], level, code)
	}
}

func lwxBinLevelWarning(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	_, _, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma",
		"-log-mode", "permissive", "-log-file", logp, "-log-level", "warning")
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if n := len(lwxLog(t, logp)); n != 0 {
		t.Errorf("level warning must not write a successful run, got %d records", n)
	}
	_, _, code = orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "nomatchword", "-text", "x",
		"-log-mode", "permissive", "-log-file", logp, "-log-level", "warning")
	if code == 0 {
		t.Fatalf("the second run must fail")
	}
	lines := lwxLog(t, logp)
	if len(lines) != 1 || !strings.Contains(lines[0], "phase=end") || !strings.Contains(lines[0], " op=replaceall source=cli") {
		t.Errorf("a failure is written as one self-contained record: %q", lines)
	}
}

func lwxBinStrictWritesBoth(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma",
		"-log-mode", "strict", "-log-file", logp, "-log-level", "error")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if n := len(lwxLog(t, logp)); n != 2 {
		t.Errorf("strict writes every record whatever the level says, got %d", n)
	}
}

func lwxBinStrictRefuses(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	blocker := orxWrite(t, dir, "blocker", "x")
	logp := filepath.Join(blocker, "x.log")

	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma",
		"-log-mode", "strict", "-log-file", logp)
	if code != 4 {
		t.Errorf("exit %d; want 4 (begin record not written, nothing changed)\nstderr: %q", code, se)
	}
	if !strings.Contains(se, "strict") {
		t.Errorf("stderr should explain that strict logging refused the run: %q", se)
	}
	if got := orxNorm(orxRead(t, f)); got != lwxOrig {
		t.Errorf("the file must be untouched, got %q", got)
	}

	_, _, code = orxRun(t, bin, "-file", f, "-op", "show", "-log-mode", "strict", "-log-file", logp)
	if code != 4 {
		t.Errorf("strict also refuses a read-only operation it would have to log: exit %d", code)
	}

	_, se, code = orxRun(t, bin, "-file", f, "-op", "show", "-log-mode", "strict", "-log-file", logp, "-log-scope", "writes")
	if code != 0 {
		t.Errorf("with -log-scope writes a read-only operation is not logged, so strict cannot block it: exit %d, stderr %q", code, se)
	}
}

func lwxBinPermissiveSpoolsThenMerges(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	la := os.Getenv("LOCALAPPDATA")
	blocker := orxWrite(t, dir, "blocker", "x")
	bad := filepath.Join(blocker, "x.log")

	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "zzzword",
		"-log-mode", "permissive", "-log-file", bad)
	if code != 0 {
		t.Fatalf("permissive must not fail the operation: exit %d, stderr %q", code, se)
	}
	if !strings.Contains(se, "The action was not logged") {
		t.Errorf("stderr must carry the notice, got %q", se)
	}
	if got := orxNorm(orxRead(t, f)); !strings.Contains(got, "zzzword") {
		t.Errorf("the edit must have been applied, got %q", got)
	}
	spoolDir := filepath.Join(la, "fedit", "spool")
	spooled := spGlob(t, filepath.Join(spoolDir, "*.log"))
	if len(spooled) != 1 {
		t.Fatalf("want one spool file, found %v", spooled)
	}
	for _, l := range strings.Split(strings.TrimRight(orxNorm(spRead(t, spooled[0])), "\n"), "\n") {
		if !strings.HasSuffix(l, " logged=spool") {
			t.Errorf("spooled record %q must end with logged=spool", l)
		}
	}

	good := filepath.Join(dir, "good.log")
	_, se, code = orxRun(t, bin, "-file", f, "-op", "find", "-match", "zzzword", "-log-mode", "permissive", "-log-file", good)
	if code != 0 {
		t.Fatalf("second run: exit %d, stderr %q", code, se)
	}
	lines := lwxLog(t, good)
	if len(lines) != 4 || !strings.HasSuffix(lines[0], " late=true") || !strings.HasSuffix(lines[1], " late=true") || strings.HasSuffix(lines[2], " late=true") {
		t.Errorf("the next logged run must merge the spooled records first, got %q", lines)
	}
	if m := spGlob(t, filepath.Join(spoolDir, "*")); len(m) != 0 {
		t.Errorf("the spool folder should be empty after the merge, found %v", m)
	}
}

func lwxBinInvalidSettingIsError(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	t.Setenv("FEDIT_LOG_MODE", "loud")
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma")
	if code != 1 || !strings.Contains(se, "log_mode") {
		t.Errorf("exit %d, stderr %q; want exit 1 naming log_mode", code, se)
	}
	if got := orxNorm(orxRead(t, f)); got != lwxOrig {
		t.Errorf("nothing may be edited when a logging setting is invalid, got %q", got)
	}
}

func lwxBinOutputResultAndLog(t *testing.T, bin string) {
	dir, f := lwxSetup(t)
	logp := filepath.Join(dir, "fedit.log")
	res := filepath.Join(dir, "result.txt")
	so, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma",
		"-log-mode", "permissive", "-log-file", logp, "-outputresult", res)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if lines := strings.Split(strings.TrimRight(so, "\r\n"), "\n"); len(lines) != 1 || !strings.Contains(lines[0], res) {
		t.Errorf("the terminal still gets one summary line: %q", so)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 || !strings.Contains(lines[1], "phase=end exit=0 ") {
		t.Errorf("the log must have both records with exit 0: %q", lines)
	}
	if strings.Contains(orxRead(t, res), "phase=begin") {
		t.Errorf("log records must not leak into the result file")
	}
}

func lwxBinUsageIsNotAnOperation(t *testing.T, bin string) {
	dir, _ := lwxSetup(t)
	blocker := orxWrite(t, dir, "blocker", "x")
	t.Setenv("FEDIT_LOG_MODE", "strict")
	t.Setenv("FEDIT_LOG_FILE", filepath.Join(blocker, "x.log"))
	_, se, code := orxRun(t, bin)
	if code != 1 {
		t.Errorf("a usage error exits 1 even in strict mode (it is not an operation): exit %d", code)
	}
	for _, flag := range []string{"-log-mode", "-log-file", "-log-max-mb", "-log-content", "-log-level", "-log-scope", "-log merge"} {
		if !strings.Contains(se, flag) {
			t.Errorf("the usage text must mention %s", flag)
		}
	}
}
