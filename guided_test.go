package main

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

// guidedDir makes a scratch dir with b.go, c.go, d.go and chdirs into it.
func guidedDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		"b.go": "package b\n\nfunc two() {}\n",
		"c.go": "package c\n\nfunc three() {}\n",
		"d.go": "package d\n\nfunc two() {}\n",
	}
	for n, c := range files {
		if err := os.WriteFile(dir+"/"+n, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	old, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(old) })
}

func gs(argv ...string) guidedStep { return guidedStep{argv: argv} }

func TestGuidedPlan(t *testing.T) {
	guidedDir(t)
	cp := func(dest string, extra ...string) []string {
		return append([]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", dest}, extra...)
	}
	cases := []struct {
		name string
		in   []string
		want []guidedStep
	}{
		{"copy before", []string{"a.go", "one", "copy", "b.go", "before", "two"},
			[]guidedStep{gs(cp("b.go", "-beforeblock", "two", "-v")...)}},
		{"copy after, fanout both ok", []string{"a.go", "one", "copy", "(b.go,d.go)", "after", "two"},
			[]guidedStep{gs(cp("b.go,d.go", "-afterblock", "two", "-v")...)}},
		{"fanout anchor missing in c", []string{"a.go", "one", "copy", "b.go,c.go", "before", "two"},
			[]guidedStep{gs(cp("b.go", "-beforeblock", "two", "-v")...), gs(cp("c.go", "-after", "3", "-v")...)}},
		{"missing dest file is created", []string{"a.go", "one", "copy", "nope.go", "before", "two"},
			[]guidedStep{{argv: cp("nope.go", "-after", "0", "-v"), create: []string{"nope.go"}}}},
		{"at-end single", []string{"a.go", "one", "copy", "b.go", "at-end"},
			[]guidedStep{gs(cp("b.go", "-after", "3", "-v")...)}},
		{"no position", []string{"a.go", "one", "copy", "b.go"},
			[]guidedStep{gs(cp("b.go", "-after", "3", "-v")...)}},
		{"at-end fanout", []string{"a.go", "one", "copy", "b.go,c.go", "at-end"},
			[]guidedStep{gs(cp("b.go", "-after", "3", "-v")...), gs(cp("c.go", "-after", "3", "-v")...)}},
		{"copy line range", []string{"a.go", "line 12-18", "copy", "b.go", "before", "two"},
			[]guidedStep{gs("-op", "copy", "-file", "a.go", "-lang", "go", "-line", "12", "-end", "18", "-dest", "b.go", "-beforeblock", "two", "-v")}},
		{"move single", []string{"a.go", "one", "move", "b.go", "before", "two"},
			[]guidedStep{gs("-op", "move", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-beforeblock", "two", "-v")}},
		{"move fanout, move runs last", []string{"a.go", "one", "move", "b.go,c.go", "before", "two"},
			[]guidedStep{gs(cp("b.go", "-beforeblock", "two", "-v")...),
				gs("-op", "move", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "c.go", "-after", "3", "-v")}},
		{"replace default target", []string{"a.go", "one", "replace", "b.go"},
			[]guidedStep{gs("-op", "replace", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-destblock", "one", "-v")}},
		{"replace explicit target", []string{"a.go", "one", "replace", "b.go", "three"},
			[]guidedStep{gs("-op", "replace", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-destblock", "three", "-v")}},
		{"insert from file", []string{"a.go", "one", "insertbefore", "b.go", "two"},
			[]guidedStep{gs("-op", "insertbefore", "-file", "b.go", "-lang", "go", "-block", "two", "-sourcefile", "a.go", "-sourceblock", "one", "-v")}},
		{"insert text", []string{"some code", "insertafter", "b.go", "two"},
			[]guidedStep{gs("-op", "insertafter", "-file", "b.go", "-lang", "go", "-block", "two", "-text", "some code", "-v")}},
		{"silent", []string{"a.go", "one", "copy", "b.go,c.go", "before", "two", "silent"},
			[]guidedStep{gs(cp("b.go", "-beforeblock", "two", "-quiet")...), gs(cp("c.go", "-after", "3", "-quiet")...)}},
	}
	for _, c := range cases {
		got, _, err := planGuided(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got  %q\n want %q", c.name, got, c.want)
		}
	}
}

func TestGuidedNotes(t *testing.T) {
	guidedDir(t)
	cases := []struct {
		name, frag string
		in         []string
	}{
		{"no position", "no position given", []string{"a.go", "one", "copy", "b.go"}},
		{"missing file", "will be created", []string{"a.go", "one", "copy", "nope.go", "at-end"}},
		{"missing anchor", "not found, appending", []string{"a.go", "one", "copy", "c.go", "before", "two"}},
	}
	for _, c := range cases {
		_, notes, err := planGuided(c.in)
		if err != nil || !strings.Contains(strings.Join(notes, "|"), c.frag) {
			t.Errorf("%s: want note containing %q, got %v (err %v)", c.name, c.frag, notes, err)
		}
	}
}

func TestGuidedErrors(t *testing.T) {
	guidedDir(t)
	cases := []struct {
		name, frag string
		in         []string
	}{
		{"too few args", "usage", []string{"a.go", "one", "copy"}},
		{"unknown verb", "unknown verb", []string{"a.go", "one", "swap", "b.go"}},
		{"unknown position", "unknown position", []string{"a.go", "one", "copy", "b.go", "under", "two"}},
		{"before without target", "needs a target", []string{"a.go", "one", "copy", "b.go", "before"}},
		{"lang mismatch", "one language", []string{"a.go", "one", "copy", "b.py", "before", "two"}},
		{"unknown source ext", "cannot detect", []string{"a.xyz", "one", "copy", "b.go", "before", "two"}},
		{"replace line range no target", "explicit target", []string{"a.go", "line 1-3", "replace", "b.go"}},
		{"insert from line range", "named source block", []string{"a.go", "line 1-3", "insertbefore", "b.go", "two"}},
		{"insert file no target", "needs a target", []string{"a.go", "one", "insertafter", "b.go"}},
		{"insert two dests", "exactly one", []string{"a.go", "one", "insertafter", "b.go,c.go", "two"}},
	}
	for _, c := range cases {
		_, _, err := planGuided(c.in)
		if err == nil || !strings.Contains(err.Error(), c.frag) {
			t.Errorf("%s: want error containing %q, got %v", c.name, c.frag, err)
		}
	}
}

func TestIsGuidedCall(t *testing.T) {
	if !isGuidedCall([]string{"fedit", "guided"}) || isGuidedCall([]string{"fedit", "-op"}) || isGuidedCall([]string{"fedit"}) {
		t.Error("isGuidedCall misclassified")
	}
}
