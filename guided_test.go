package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestGuidedTranslate(t *testing.T) {
	dir := t.TempDir()
	b := filepath.Join(dir, "b.go")
	if err := os.WriteFile(b, []byte("package b\n\nfunc two() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"copy before", []string{"a.go", "one", "copy", "b.go", "before", "two"},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-beforeblock", "two", "-v"}},
		{"copy after fanout", []string{"a.go", "one", "copy", "(b.go,c.go)", "after", "two"},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go,c.go", "-afterblock", "two", "-v"}},
		{"copy at-end", []string{"a.go", "one", "copy", b, "at-end"},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", b, "-after", "3", "-v"}},
		{"copy no position appends", []string{"a.go", "one", "copy", b},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", b, "-after", "3", "-v"}},
		{"copy missing dest file", []string{"a.go", "one", "copy", filepath.Join(dir, "new.go")},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", filepath.Join(dir, "new.go"), "-after", "0", "-v"}},
		{"copy line range", []string{"a.go", "line 12-18", "copy", "b.go", "before", "two"},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-line", "12", "-end", "18", "-dest", "b.go", "-beforeblock", "two", "-v"}},
		{"move", []string{"a.go", "one", "move", "b.go", "before", "two"},
			[]string{"-op", "move", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-beforeblock", "two", "-v"}},
		{"replace default target", []string{"a.go", "one", "replace", "b.go"},
			[]string{"-op", "replace", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-destblock", "one", "-v"}},
		{"replace explicit target", []string{"a.go", "one", "replace", "b.go", "three"},
			[]string{"-op", "replace", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-destblock", "three", "-v"}},
		{"insert from file", []string{"a.go", "one", "insertbefore", "b.go", "two"},
			[]string{"-op", "insertbefore", "-file", "b.go", "-lang", "go", "-block", "two", "-sourcefile", "a.go", "-sourceblock", "one", "-v"}},
		{"insert text", []string{"some code", "insertafter", "b.go", "two"},
			[]string{"-op", "insertafter", "-file", "b.go", "-lang", "go", "-block", "two", "-text", "some code", "-v"}},
		{"silent", []string{"a.go", "one", "copy", "b.go", "before", "two", "silent"},
			[]string{"-op", "copy", "-file", "a.go", "-lang", "go", "-block", "one", "-dest", "b.go", "-beforeblock", "two", "-quiet"}},
	}
	for _, c := range cases {
		got, _, err := translateGuided(c.in)
		if err != nil {
			t.Errorf("%s: unexpected error: %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got  %q\n want %q", c.name, got, c.want)
		}
	}
}

func TestGuidedErrors(t *testing.T) {
	cases := []struct {
		name, frag string
		in         []string
	}{
		{"too few args", "usage", []string{"a.go", "one", "copy"}},
		{"unknown verb", "unknown verb", []string{"a.go", "one", "swap", "b.go"}},
		{"unknown position", "unknown position", []string{"a.go", "one", "copy", "b.go", "under", "two"}},
		{"before without target", "needs a target", []string{"a.go", "one", "copy", "b.go", "before"}},
		{"fanout at-end", "one destination", []string{"a.go", "one", "copy", "b.go,c.go"}},
		{"lang mismatch", "one language", []string{"a.go", "one", "copy", "b.py", "before", "two"}},
		{"unknown source ext", "cannot detect", []string{"a.xyz", "one", "copy", "b.go", "before", "two"}},
		{"replace line range no target", "explicit target", []string{"a.go", "line 1-3", "replace", "b.go"}},
		{"insert from line range", "named source block", []string{"a.go", "line 1-3", "insertbefore", "b.go", "two"}},
		{"insert file no target", "needs a target", []string{"a.go", "one", "insertafter", "b.go"}},
		{"insert two dests", "exactly one", []string{"a.go", "one", "insertafter", "b.go,c.go", "two"}},
	}
	for _, c := range cases {
		_, _, err := translateGuided(c.in)
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
