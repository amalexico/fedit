package main

import (
	"strings"
	"testing"
)

// Multi-line string constructs with braces inside them must not move block
// boundaries (task 8). Each case gives the source lines and the expected
// start and end line of every block, found by name.

type mlWant struct {
	name       string
	start, end int
}

func mlCheck(t *testing.T, entries []blockEntry, want []mlWant) {
	t.Helper()
	if len(entries) != len(want) {
		t.Errorf("got %d blocks %v, want %d", len(entries), entries, len(want))
	}
	for _, w := range want {
		found := false
		for _, e := range entries {
			if strings.Contains(e.name, w.name) || e.key == w.name {
				found = true
				if e.start != w.start || e.end != w.end {
					t.Errorf("%s: got lines %d-%d, want %d-%d", w.name, e.start, e.end, w.start, w.end)
				}
			}
		}
		if !found {
			t.Errorf("block %q not found in %v", w.name, entries)
		}
	}
}

func TestMultilineStringBlocks(t *testing.T) {
	cases := []struct {
		name string
		scan func([]string) []blockEntry
		src  []string
		want []mlWant
	}{
		{"go raw string", getGoBlocks, []string{
			"func alpha() {", "\ts := `", "}", "{{", "`", "\t_ = s", "}", "", "func beta() {", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 10}}},
		{"csharp verbatim", getCSharpBlocks, []string{
			"class Alpha {", "    string a = @\"line one }", "    still string { {", "    end\";", "    int z = 1;", "}", "",
			"class Beta {", "    int y = 2;", "}"},
			[]mlWant{{"Alpha", 1, 6}, {"Beta", 8, 10}}},
		{"csharp interpolated verbatim", getCSharpBlocks, []string{
			"class Alpha {", "    string a = $@\"x {y} }", "    { {\";", "    int z = 1;", "}", "",
			"class Beta {", "    int y = 2;", "}"},
			[]mlWant{{"Alpha", 1, 5}, {"Beta", 7, 9}}},
		{"csharp verbatim doubled quote", getCSharpBlocks, []string{
			"class Alpha {", "    string a = @\"a \"\"}\"\" b\";", "    int z = 1;", "}", "",
			"class Beta {", "    int y = 2;", "}"},
			[]mlWant{{"Alpha", 1, 4}, {"Beta", 6, 8}}},
		{"csharp raw string", getCSharpBlocks, []string{
			"class Alpha {", "    string a = \"\"\"", "        } }", "        { {", "        \"\"\";", "    int z = 1;", "}", "",
			"class Beta {", "    int y = 2;", "}"},
			[]mlWant{{"Alpha", 1, 7}, {"Beta", 9, 11}}},
		{"java text block", getJavaBlocks, []string{
			"class Alpha {", "    String a = \"\"\"", "        } }", "        { {", "        \"\"\";", "    int z = 1;", "}", "",
			"class Beta {", "    int y = 2;", "}"},
			[]mlWant{{"Alpha", 1, 7}, {"Beta", 9, 11}}},
		{"rust raw with hashes", getRustBlocks, []string{
			"fn alpha() {", "    let a = r#\"", "        } }", "        \"{\" quote", "        \"#;", "    let z = 1;", "}", "",
			"fn beta() {", "    let y = 2;", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
		{"rust raw no hashes", getRustBlocks, []string{
			"fn alpha() {", "    let a = r\"", "}", "{{", "\";", "    let z = 1;", "}", "",
			"fn beta() {", "    let y = 2;", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
		{"rust raw one line", getRustBlocks, []string{
			"fn alpha() {", "    let a = r#\"}\"#;", "    let b = \"{{\";", "}", "",
			"fn beta() {", "    let y = 2;", "}"},
			[]mlWant{{"alpha", 1, 4}, {"beta", 6, 8}}},
		{"hcl heredoc", getHCLBlocks, []string{
			"resource \"test\" \"alpha\" {", "  value = <<EOT", "}", "{{", "EOT", "  other = \"x\"", "}", "",
			"resource \"test\" \"beta\" {", "  value = \"y\"", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
		{"hcl indented heredoc", getHCLBlocks, []string{
			"resource \"test\" \"alpha\" {", "  value = <<-EOT", "    }", "    {{", "    EOT", "  other = \"x\"", "}", "",
			"resource \"test\" \"beta\" {", "  value = \"y\"", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
		{"nix indented string", getNixBlocks, []string{
			"alpha = {", "  a = ''", "    } }", "    { {", "  '';", "  b = 1;", "};", "",
			"beta = {", "  c = 1;", "};"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
		{"nix one line indented string", getNixBlocks, []string{
			"alpha = {", "  a = ''}'';", "  b = \"{{\";", "};", "",
			"beta = {", "  c = 1;", "};"},
			[]mlWant{{"alpha", 1, 4}, {"beta", 6, 8}}},
		{"nix escapes", getNixBlocks, []string{
			"alpha = {", "  a = ''", "    it'''s a } and ''${x} {", "  '';", "};", "",
			"beta = {", "  c = 1;", "};"},
			[]mlWant{{"alpha", 1, 5}, {"beta", 7, 9}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mlCheck(t, c.scan(c.src), c.want)
		})
	}
}
