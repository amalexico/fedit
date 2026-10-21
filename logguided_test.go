package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Wiring slice A2: guided runs. A one-step plain command continues in the same
// process (source=guided). A run with several steps, or with a file to create,
// is a parent process that creates the files and starts one child fedit per
// step. The parent logs its own record pair (op=guided, steps, create) and
// hands its id to the children in FEDIT_LOG_PARENT, so each child logs
// source=guided and parent=<id>. These tests reuse the helpers of
// logwire_test.go and build the binary once.

const lgdA = "package p\n\nfunc one() {\n\tprintln(\"one\")\n}\n"
const lgdB = "package p\n\nfunc two() {\n}\n"

func lgdFixture(t *testing.T) string {
	t.Helper()
	lwxEnv(t)
	dir := t.TempDir()
	orxWrite(t, dir, "a.go", lgdA)
	orxWrite(t, dir, "b.go", lgdB)
	orxWrite(t, dir, "c.go", lgdB)
	return dir
}

func lgdRun(t *testing.T, bin, dir string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	code := 0
	if err := cmd.Run(); err != nil {
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			t.Fatalf("run %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

func lgdID(t *testing.T, line string) string {
	t.Helper()
	id, _ := spoolLogField(line, "id")
	if len(id) != 16 {
		t.Fatalf("record %q has no 16-character id", line)
	}
	return id
}

// ---- unit tests ----

func TestLGD_OpTable(t *testing.T) {
	if !logOpKnown("guided") || !logOpIsWrite("guided") {
		t.Errorf("guided must be a known writing operation: known = %v, write = %v", logOpKnown("guided"), logOpIsWrite("guided"))
	}
}

func TestLGD_ParentEnvName(t *testing.T) {
	if logParentEnv != "FEDIT_LOG_PARENT" {
		t.Errorf("logParentEnv = %q; want FEDIT_LOG_PARENT", logParentEnv)
	}
}

func TestLGD_FlagString(t *testing.T) {
	if got := logFlagString(nil); got != "" {
		t.Errorf("logFlagString(nil) = %q; guided runs start logging before any flag is registered", got)
	}
	s := "strict"
	if got := logFlagString(&s); got != "strict" {
		t.Errorf("logFlagString = %q; want strict", got)
	}
}

func TestLGD_FieldsParent(t *testing.T) {
	cwd := t.TempDir()
	kvs := logCLIFields(logCLIInput{Source: "guided", Cwd: cwd, Parent: "abc123", File: "a.go"})
	want := []string{"source", "user", "host", "pid", "cwd", "parent", "file"}
	if got := lwxKeys(kvs); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("keys = %v; want %v", got, want)
	}
	if v, _ := lwxVal(kvs, "parent"); v != "abc123" {
		t.Errorf("parent = %q; want abc123", v)
	}
}

func TestLGD_FieldsStepsAndCreate(t *testing.T) {
	cwd := t.TempDir()
	kvs := logCLIFields(logCLIInput{Source: "guided", Cwd: cwd, Steps: 2, Create: "nope.go, sub/x.go"})
	if v, _ := lwxVal(kvs, "steps"); v != "2" {
		t.Errorf("steps = %q; want 2", v)
	}
	want := filepath.Join(cwd, "nope.go") + "," + filepath.Join(cwd, "sub", "x.go")
	if v, _ := lwxVal(kvs, "create"); v != want {
		t.Errorf("create = %q; want %q", v, want)
	}

	kvs = logCLIFields(logCLIInput{Source: "cli", Cwd: cwd})
	for _, k := range []string{"parent", "steps", "create"} {
		if _, ok := lwxVal(kvs, k); ok {
			t.Errorf("%s must be left out when it was not given", k)
		}
	}
}

// ---- binary-level tests ----

// TestLGD_Guided builds the binary once and runs every guided case as a
// subtest against it.
func TestLGD_Guided(t *testing.T) {
	bin := clhBuild(t)
	cases := []struct {
		name string
		run  func(*testing.T, string)
	}{
		{"MultiStepParentAndChildren", lgdMultiStep},
		{"CreateIsLogged", lgdCreate},
		{"StrictRefusesBeforeCreating", lgdStrictRefuses},
		{"OffLogsNothing", lgdOff},
		{"SingleStepStaysInProcess", lgdSingleStep},
		{"ParentEnvMarksTheRun", lgdParentEnv},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { c.run(t, bin) })
	}
}

func lgdMultiStep(t *testing.T, bin string) {
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)

	so, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "b.go,c.go", "at-end")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	for _, name := range []string{"b.go", "c.go"} {
		if !strings.Contains(orxRead(t, filepath.Join(dir, name)), "func one()") {
			t.Errorf("the copy did not land in %s", name)
		}
	}
	lines := lwxLog(t, logp)
	if len(lines) != 6 {
		t.Fatalf("want the parent begin, two steps (begin and end each) and the parent end: 6 records, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	parent := lgdID(t, lines[0])
	if !strings.Contains(lines[0], "phase=begin op=guided source=guided") || !strings.Contains(lines[0], " steps=2") || strings.Contains(lines[0], " create=") {
		t.Errorf("parent begin record = %q", lines[0])
	}
	c1, c2 := lgdID(t, lines[1]), lgdID(t, lines[3])
	if c1 == parent || c2 == parent || c1 == c2 {
		t.Errorf("every process has its own id: parent %s, children %s and %s", parent, c1, c2)
	}
	for _, i := range []int{1, 3} {
		if !strings.Contains(lines[i], "phase=begin op=copy source=guided") || !strings.Contains(lines[i], " parent="+parent) {
			t.Errorf("child begin record = %q; want source=guided and parent=%s", lines[i], parent)
		}
	}
	if lgdID(t, lines[2]) != c1 || lgdID(t, lines[4]) != c2 || !strings.Contains(lines[2], "phase=end exit=0") || !strings.Contains(lines[4], "phase=end exit=0") {
		t.Errorf("each child end record must share its begin record's id:\n%s\n%s", lines[2], lines[4])
	}
	if !strings.Contains(lines[5], "id="+parent+" phase=end exit=0") {
		t.Errorf("parent end record = %q", lines[5])
	}
}

func lgdCreate(t *testing.T, bin string) {
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)

	so, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "nope.go", "at-end")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(orxRead(t, filepath.Join(dir, "nope.go")), "func one()") {
		t.Errorf("the new file must hold the copied block")
	}
	lines := lwxLog(t, logp)
	if len(lines) != 4 {
		t.Fatalf("want 4 records, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], " steps=1") || !strings.Contains(lines[0], " create="+logQuote(filepath.Join(dir, "nope.go"))) {
		t.Errorf("the parent must record the file it creates: %q", lines[0])
	}
}

func lgdStrictRefuses(t *testing.T, bin string) {
	dir := lgdFixture(t)
	blocker := orxWrite(t, dir, "blocker", "x")
	t.Setenv("FEDIT_LOG_MODE", "strict")
	t.Setenv("FEDIT_LOG_FILE", filepath.Join(blocker, "x.log"))

	_, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "nope.go", "at-end")
	if code != 4 {
		t.Errorf("exit %d; want 4 (begin record not written, nothing changed)\nstderr: %s", code, se)
	}
	if !strings.Contains(se, "strict") {
		t.Errorf("stderr should explain that strict logging refused the run: %q", se)
	}
	if _, err := os.Stat(filepath.Join(dir, "nope.go")); err == nil {
		t.Errorf("strict must refuse before the parent creates any file")
	}
}

func lgdOff(t *testing.T, bin string) {
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_FILE", logp)

	_, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "b.go,c.go", "at-end")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if n := len(lwxLog(t, logp)); n != 0 {
		t.Errorf("with log_mode unset nothing is logged, got %d records", n)
	}
}

func lgdSingleStep(t *testing.T, bin string) {
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)

	so, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "b.go", "at-end")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 || !strings.Contains(lines[0], "phase=begin op=copy source=guided") || strings.Contains(lines[0], " parent=") {
		t.Errorf("a one-step plain command is one logged run with source=guided and no parent: %q", lines)
	}
}

func lgdParentEnv(t *testing.T, bin string) {
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	t.Setenv("FEDIT_LOG_PARENT", "abc123")

	_, se, code := lgdRun(t, bin, dir, "-file", filepath.Join(dir, "b.go"), "-op", "find", "-match", "two")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	lines := lwxLog(t, logp)
	if len(lines) != 2 || !strings.Contains(lines[0], "phase=begin op=find source=guided") || !strings.Contains(lines[0], " parent=abc123") {
		t.Errorf("a run that sees FEDIT_LOG_PARENT logs source=guided and its parent: %q", lines)
	}
}
