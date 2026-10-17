package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

var (
	g2Exe  string
	g2Once sync.Once
)

// g2Binary builds the binary once for the tests that need a real process.
func g2Binary(t *testing.T) string {
	t.Helper()
	g2Once.Do(func() {
		dir, err := os.MkdirTemp("", "fedit_g2bin")
		if err != nil {
			return
		}
		exe := filepath.Join(dir, "fedit_g2.exe")
		if out, err := exec.Command("go", "build", "-o", exe, ".").CombinedOutput(); err != nil {
			t.Logf("go build failed: %v\n%s", err, out)
			return
		}
		g2Exe = exe
	})
	if g2Exe == "" {
		t.Fatal("cannot build the fedit binary")
	}
	return g2Exe
}

// g2Dir makes a scratch dir: a.go has func one, b.go has func two, r.go has an old func one.
func g2Dir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"a.go": "package a\n\nfunc one() { println(\"new\") }\n",
		"b.go": "package b\n\nfunc two() {}\n",
		"r.go": "package r\n\nfunc one() {}\n",
	}
	for n, c := range files {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func g2Read(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// g2Run runs the built binary in dir with the given stdin and returns stdout, stderr and the exit code.
func g2Run(t *testing.T, dir, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(g2Binary(t), args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	var so, se bytes.Buffer
	cmd.Stdout, cmd.Stderr = &so, &se
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = ee.ExitCode()
	}
	return so.String(), se.String(), code
}

func TestGuidedReportOnlyPlan(t *testing.T) {
	guidedDir(t)
	argv, _, err := translateGuided([]string{"a.go", "one", "copy", "b.go", "after", "two", "report-only"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-afterblock", "two"}
	if strings.Join(argv, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %q, want %q", argv, want)
	}
}

func TestGuidedReportOnlyFanoutKeepsNotes(t *testing.T) {
	guidedDir(t)
	steps, notes, err := planGuided([]string{"a.go", "one", "copy", "b.go,c.go", "before", "two", "report-only"})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(steps))
	}
	for _, st := range steps {
		for _, a := range st.argv {
			if a == "-v" || a == "-quiet" {
				t.Errorf("report-only left %s in %q", a, st.argv)
			}
		}
	}
	if len(notes) == 0 {
		t.Error("report-only dropped the fallback note")
	}
}

func TestGuidedUsageMentionsReportOnly(t *testing.T) {
	if !strings.Contains(guidedUsage, "report-only") {
		t.Errorf("usage does not mention report-only:\n%s", guidedUsage)
	}
}

func TestGuidedReportOnlyRun(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "copy", "b.go", "after", "two", "report-only")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if strings.Contains(se, "guided: fedit") {
		t.Errorf("report-only echoed the command:\n%s", se)
	}
	if !strings.Contains(so+se, "OK") {
		t.Errorf("report-only lost the engine summary:\n%s", so)
	}
	if !strings.Contains(g2Read(t, dir, "b.go"), "func one()") {
		t.Errorf("copy did not land:\n%s", g2Read(t, dir, "b.go"))
	}
}

func TestGuidedBareCopy(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "copy\na.go\none\nb.go\nafter\ntwo\n\n", "guided")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	b := g2Read(t, dir, "b.go")
	if !strings.Contains(b, "func one()") || !strings.Contains(b, "func two()") {
		t.Errorf("b.go after copy:\n%s", b)
	}
	if !strings.Contains(se, "guided: fedit -op copy") {
		t.Errorf("the translated command was not echoed:\n%s", se)
	}
}

func TestGuidedBareNumbers(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "1\na.go\none\nb.go\n3\n\n", "guided")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(g2Read(t, dir, "b.go"), "func one()") {
		t.Errorf("numbered copy at-end did not land:\n%s", g2Read(t, dir, "b.go"))
	}
}

func TestGuidedBareReplace(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "replace\na.go\none\nr.go\n\n\n", "guided")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(g2Read(t, dir, "r.go"), `println("new")`) {
		t.Errorf("r.go after replace:\n%s", g2Read(t, dir, "r.go"))
	}
}

func TestGuidedBareInsertText(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "insertafter\ntext\nhello world\nb.go\ntwo\n\n", "guided")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(g2Read(t, dir, "b.go"), "hello world") {
		t.Errorf("b.go after insert:\n%s", g2Read(t, dir, "b.go"))
	}
}

func TestGuidedBareRetriesBadChoice(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "bogus\ncopy\na.go\none\nb.go\nafter\ntwo\n\n", "guided")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(g2Read(t, dir, "b.go"), "func one()") {
		t.Errorf("copy did not land after a bad first answer:\n%s", g2Read(t, dir, "b.go"))
	}
}

func TestGuidedBareEOFChangesNothing(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "copy\na.go\n", "guided")
	if code == 0 {
		t.Fatalf("exit 0 on truncated input\nstdout: %s\nstderr: %s", so, se)
	}
	if !strings.Contains(se, "nothing changed") {
		t.Errorf("stderr does not say nothing changed:\n%s", se)
	}
	if g2Read(t, dir, "b.go") != "package b\n\nfunc two() {}\n" {
		t.Errorf("b.go was modified:\n%s", g2Read(t, dir, "b.go"))
	}
}

func TestGuidedFastSyntaxNeverPrompts(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "copy\n", "guided", "a.go", "one")
	if code != 2 {
		t.Fatalf("exit %d, want 2\nstdout: %s\nstderr: %s", code, so, se)
	}
	if !strings.Contains(se, "usage:") {
		t.Errorf("stderr has no usage text:\n%s", se)
	}
	if g2Read(t, dir, "b.go") != "package b\n\nfunc two() {}\n" {
		t.Errorf("b.go was modified:\n%s", g2Read(t, dir, "b.go"))
	}
}
