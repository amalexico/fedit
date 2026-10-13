package main

import (
	"strings"
	"testing"
)

var sbhInserted = []string{
	`func Up(s string) string {`,
	`	return strings.ToUpper(s)`,
	`}`,
}

var sbhDestNoImport = []string{`package main`, ``}

var sbhDestWithImport = []string{`package main`, ``, `import "strings"`}

func TestSourceBlockHint_MissingImport(t *testing.T) {
	got := sourceBlockHint("a.go", "Up", "b.go", sbhInserted, sbhDestNoImport)
	if !strings.HasPrefix(got, "hint:") || !strings.Contains(got, "strings") || !strings.Contains(got, "b.go") {
		t.Errorf("want a strings import hint naming b.go, got %q", got)
	}
}

func TestSourceBlockHint_AlreadyImported_NoHint(t *testing.T) {
	got := sourceBlockHint("a.go", "Up", "b.go", sbhInserted, sbhDestWithImport)
	if got != "" {
		t.Errorf("want no hint when strings is already imported, got %q", got)
	}
}

func TestSourceBlockHint_NeedsBothSourceFlags(t *testing.T) {
	cases := []struct {
		name, srcFile, srcBlock string
	}{
		{"neither", "", ""},
		{"file only", "a.go", ""},
		{"block only", "", "Up"},
	}
	for _, c := range cases {
		got := sourceBlockHint(c.srcFile, c.srcBlock, "b.go", sbhInserted, sbhDestNoImport)
		if got != "" {
			t.Errorf("%s: want no hint, got %q", c.name, got)
		}
	}
}

func TestSourceBlockHint_NonGoFiles_NoHint(t *testing.T) {
	if got := sourceBlockHint("a.go", "Up", "b.txt", sbhInserted, sbhDestNoImport); got != "" {
		t.Errorf("non-Go destination: want no hint, got %q", got)
	}
	if got := sourceBlockHint("a.txt", "Up", "b.go", sbhInserted, sbhDestNoImport); got != "" {
		t.Errorf("non-Go source: want no hint, got %q", got)
	}
}
