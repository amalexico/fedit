package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// gdrWant is the plan for copying one from a.go into b.go after two, as the
// report-only plan test spells it.
var gdrWant = []string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-afterblock", "two"}

func gdrCount(argv []string, w string) int {
	n := 0
	for _, a := range argv {
		if a == w {
			n++
		}
	}
	return n
}

func gdrMissing(t *testing.T, dir, name string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
		t.Errorf("-dry created %s", name)
	}
}

func gdrSame(t *testing.T, dir, name, want string) {
	t.Helper()
	if got := g2Read(t, dir, name); got != want {
		t.Errorf("-dry changed %s: %q -> %q", name, want, got)
	}
}

const (
	gdrA = "package a\n\nfunc one() { println(\"new\") }\n"
	gdrB = "package b\n\nfunc two() {}\n"
)

func TestGuidedDryPlanReportOnly(t *testing.T) {
	guidedDir(t)
	argv, _, err := translateGuided([]string{"a.go", "one", "copy", "b.go", "after", "two", "report-only", "dry"})
	if err != nil {
		t.Fatal(err)
	}
	want := append(append([]string{}, gdrWant...), "-dry")
	if strings.Join(argv, " ") != strings.Join(want, " ") {
		t.Errorf("argv = %q, want %q", argv, want)
	}
}

func TestGuidedDryPlanKeepsVerbose(t *testing.T) {
	guidedDir(t)
	argv, _, err := translateGuided([]string{"a.go", "one", "copy", "b.go", "after", "two", "dry"})
	if err != nil {
		t.Fatal(err)
	}
	if gdrCount(argv, "-dry") != 1 || gdrCount(argv, "-v") != 1 {
		t.Errorf("want one -dry and one -v in %q", argv)
	}
}

func TestGuidedDryPlanSilent(t *testing.T) {
	guidedDir(t)
	argv, _, err := translateGuided([]string{"a.go", "one", "copy", "b.go", "after", "two", "silent", "dry"})
	if err != nil {
		t.Fatal(err)
	}
	if gdrCount(argv, "-dry") != 1 || gdrCount(argv, "-quiet") != 1 || gdrCount(argv, "-v") != 0 {
		t.Errorf("want one -dry, one -quiet and no -v in %q", argv)
	}
}

func TestGuidedDryPlanFanout(t *testing.T) {
	guidedDir(t)
	steps, _, err := planGuided([]string{"a.go", "one", "copy", "b.go,c.go", "before", "two", "dry"})
	if err != nil {
		t.Fatal(err)
	}
	if len(steps) != 2 {
		t.Fatalf("got %d steps, want 2", len(steps))
	}
	for _, st := range steps {
		if gdrCount(st.argv, "-dry") != 1 {
			t.Errorf("every step needs one -dry, got %q", st.argv)
		}
	}
}

func TestGuidedDryNeedsFourWords(t *testing.T) {
	guidedDir(t)
	if _, _, err := planGuided([]string{"a.go", "one", "copy", "dry"}); err == nil {
		t.Error("a trailing dry must not be taken as the destination file")
	}
}

func TestGuidedUsageMentionsDry(t *testing.T) {
	if !strings.Contains(guidedUsage, "dry") {
		t.Errorf("usage does not mention dry:\n%s", guidedUsage)
	}
}

func TestGuidedBareDryModifier(t *testing.T) {
	for _, c := range []struct{ ans, tail string }{
		{"dry", "dry"},
		{"silent dry", "silent dry"},
		{"Report-Only DRY", "report-only dry"},
	} {
		in := "copy\na.go\none\nb.go\nafter\ntwo\n" + c.ans + "\n"
		var out bytes.Buffer
		args, err := guidedInteractive(strings.NewReader(in), &out)
		if err != nil {
			t.Errorf("answer %q: %v", c.ans, err)
			continue
		}
		want := "a.go one copy b.go after two " + c.tail
		if got := strings.Join(args, " "); got != want {
			t.Errorf("answer %q: args = %q, want %q", c.ans, got, want)
		}
	}
}

func TestGuidedDryRunSingle(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "copy", "b.go", "after", "two", "dry")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	gdrSame(t, dir, "b.go", gdrB)
	if !strings.Contains(so, "+func one()") {
		t.Errorf("stdout lacks the diff: %q", so)
	}
	if !strings.Contains(se, "dry-run: applied=false") {
		t.Errorf("stderr lacks the applied=false note: %q", se)
	}
}

func TestGuidedDryMissingDest(t *testing.T) {
	dir := g2Dir(t)
	_, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "copy", "new.go", "at-end", "dry")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	gdrMissing(t, dir, "new.go")
	if !strings.Contains(se, "would create") {
		t.Errorf("stderr should say what would be created: %q", se)
	}
	if !strings.Contains(se, "dry-run: applied=false") {
		t.Errorf("stderr lacks the applied=false note: %q", se)
	}
}

func TestGuidedDryFanout(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "copy", "b.go,new.go", "at-end", "dry")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	gdrSame(t, dir, "a.go", gdrA)
	gdrSame(t, dir, "b.go", gdrB)
	gdrMissing(t, dir, "new.go")
	if !strings.Contains(so, "+++ b.go\n") {
		t.Errorf("stdout lacks the diff for b.go: %q", so)
	}
	if !strings.Contains(se, "would create") {
		t.Errorf("stderr should say what would be created: %q", se)
	}
}

func TestGuidedDryMoveKeepsSource(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "move", "b.go,new.go", "at-end", "dry")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	gdrSame(t, dir, "a.go", gdrA)
	gdrSame(t, dir, "b.go", gdrB)
	gdrMissing(t, dir, "new.go")
	if !strings.Contains(se, "would create") {
		t.Errorf("stderr should say what would be created: %q", se)
	}
}

func TestGuidedDryMoveSingle(t *testing.T) {
	dir := g2Dir(t)
	so, se, code := g2Run(t, dir, "", "guided", "a.go", "one", "move", "b.go", "at-end", "dry")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	gdrSame(t, dir, "a.go", gdrA)
	gdrSame(t, dir, "b.go", gdrB)
	if !strings.Contains(se, "would remove lines") {
		t.Errorf("the move note should say what would happen: %q", se)
	}
}

func TestGuidedDryLogged(t *testing.T) {
	bin := g2Binary(t)
	dir := lgdFixture(t)
	logp := filepath.Join(dir, "audit.log")
	t.Setenv("FEDIT_LOG_MODE", "permissive")
	t.Setenv("FEDIT_LOG_FILE", logp)
	so, se, code := lgdRun(t, bin, dir, "guided", "a.go", "one", "copy", "b.go,c.go", "at-end", "dry")
	if code != 0 {
		t.Fatalf("exit %d\nstdout: %s\nstderr: %s", code, so, se)
	}
	for _, name := range []string{"b.go", "c.go"} {
		if strings.Contains(orxRead(t, filepath.Join(dir, name)), "func one()") {
			t.Errorf("-dry changed %s", name)
		}
	}
	lines := lwxLog(t, logp)
	if len(lines) == 0 {
		t.Fatal("no log records")
	}
	last := lines[len(lines)-1]
	if !strings.Contains(last, "phase=end exit=0") || !strings.Contains(last, " dry=true") {
		t.Errorf("the parent end record should carry dry=true: %q", last)
	}
}
