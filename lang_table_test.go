package main

import (
	"strings"
	"testing"
)

// langTableCases is the one extension table every language-detection path must
// agree on: detectLang, inferLang (MCP map) and doMap auto-detect (CLI map).
var langTableCases = []struct{ file, lang string }{
	{"x.go", "go"}, {"x.html", "html"}, {"x.htm", "html"}, {"x.sql", "sql"},
	{"x.py", "python"}, {"x.js", "javascript"}, {"x.jsx", "javascript"},
	{"x.ts", "typescript"}, {"x.tsx", "typescript"}, {"x.css", "css"},
	{"x.rs", "rust"}, {"x.java", "java"}, {"x.cs", "csharp"},
	{"x.yaml", "yaml"}, {"x.yml", "yaml"}, {"x.toml", "toml"},
	{"x.md", "markdown"}, {"x.markdown", "markdown"}, {"x.rb", "ruby"},
	{"x.php", "php"}, {"Dockerfile", "dockerfile"}, {"Makefile", "makefile"},
	{"x.mk", "makefile"}, {"x.json", "json"}, {"x.csv", "csv"},
	{"x.tf", "hcl"}, {"x.hcl", "hcl"}, {"x.nix", "nix"}, {"UPPER.PY", "python"},
}

func TestLangTable_InferMatchesDetect(t *testing.T) {
	for _, c := range langTableCases {
		if got, ok := detectLang(c.file); !ok || got != c.lang {
			t.Errorf("detectLang(%q) = %q, %v; want %q", c.file, got, ok, c.lang)
		}
		if got := inferLang(c.file); got != c.lang {
			t.Errorf("inferLang(%q) = %q; want %q", c.file, got, c.lang)
		}
	}
	if got, ok := detectLang("notes.txt"); ok || got != "" {
		t.Errorf("detectLang(notes.txt) = %q, %v; want no language", got, ok)
	}
	if got := inferLang("notes.txt"); got != "" {
		t.Errorf("inferLang(notes.txt) = %q; want empty", got)
	}
}

func TestCLIMapAutoDetect_ExtensionTable(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	for _, c := range langTableCases {
		clhWrite(t, dir, c.file, "x\n")
		so, se, _ := clhRun(t, bin, dir, "-file", c.file, "-op", "map")
		if !strings.Contains(so, "("+c.lang+",") {
			t.Errorf("%s: want a map header naming %q, got stdout %q stderr %q", c.file, c.lang, so, se)
		}
	}
	clhWrite(t, dir, "notes.txt", "x\n")
	_, se, code := clhRun(t, bin, dir, "-file", "notes.txt", "-op", "map")
	if code == 0 || !strings.Contains(se, "Cannot auto-detect language") {
		t.Errorf("notes.txt: want exit 1 with the auto-detect error, got code %d stderr %q", code, se)
	}
}
