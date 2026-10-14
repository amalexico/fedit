package main

import (
	"strings"
	"testing"
)

func TestNormMapLang_ShortAliases(t *testing.T) {
	for in, want := range map[string]string{
		"js": "javascript", "JS": "javascript", "ts": "typescript",
		"cs": "csharp", "c#": "csharp", "csharp": "csharp", "javascript": "javascript",
	} {
		if got := normMapLang(in); got != want {
			t.Errorf("normMapLang(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetTopLevelBlocks_ShortAliases(t *testing.T) {
	js := []string{"function foo() {", "  return 1;", "}"}
	cs := []string{"class Foo {", "  void Bar() {", "  }", "}"}
	tf := []string{`resource "a" "b" {`, "}"}
	cases := []struct {
		lang  string
		lines []string
	}{
		{"js", js}, {"javascript", js}, {"ts", js}, {"typescript", js},
		{"cs", cs}, {"c#", cs}, {"csharp", cs},
		{"tf", tf}, {"terraform", tf}, {"hcl", tf},
	}
	for _, c := range cases {
		got, err := getTopLevelBlocks(c.lines, c.lang)
		if err != nil || len(got) == 0 {
			t.Errorf("getTopLevelBlocks(%q) = %v, %v; want at least one block", c.lang, got, err)
		}
	}
	if _, err := getTopLevelBlocks(js, "cobol"); err == nil {
		t.Error("an unknown language should still be an error")
	}
}

func TestDoMap_ShortAliasMapsAsCanonicalLanguage(t *testing.T) {
	js := []string{"function foo() {", "  return 1;", "}"}
	out := mhCapture(t, func() { doMap(js, "x.txt", "js") })
	if !strings.Contains(out, "(javascript,") {
		t.Errorf("-lang js should map as javascript:\n%s", out)
	}
}
