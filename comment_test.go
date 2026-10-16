package main

import "testing"

// Braces inside comments must not move block boundaries. Nix has no //
// comment (it is the update operator), so one case checks that it stays code.
func TestCommentBlocks(t *testing.T) {
	cases := []struct {
		name string
		scan func([]string) []blockEntry
		src  []string
		want []mlWant
	}{
		{"java line and block comments", getJavaBlocks, []string{
			"class Alpha {", "    int a = 1; // closing brace in comment }", "    /* block", "       } } {", "    */",
			"    int b = 2; /* inline } */ int c = 3;", "}", "",
			"class Beta {", "    int d = 4; // don't {", "}"},
			[]mlWant{{"Alpha", 1, 7}, {"Beta", 9, 11}}},
		{"java slashes inside a string", getJavaBlocks, []string{
			"class Alpha {", "    String u = \"http://x.com/}\";", "    int z = 1;", "}", "",
			"class Beta {", "}"},
			[]mlWant{{"Alpha", 1, 4}, {"Beta", 6, 7}}},
		{"go comments", getGoBlocks, []string{
			"func alpha() {", "\tx := 1 // }", "\t/* { { */", "\t_ = x", "}", "",
			"func beta() {", "}"},
			[]mlWant{{"alpha", 1, 5}, {"beta", 7, 8}}},
		{"rust nested block comment", getRustBlocks, []string{
			"fn alpha() {", "    /* outer /* inner } */ still } comment */", "    let a = 1;", "}", "",
			"fn beta() {", "}"},
			[]mlWant{{"alpha", 1, 4}, {"beta", 6, 7}}},
		{"hcl comments", getHCLBlocks, []string{
			"resource \"test\" \"alpha\" {", "  a = 1 # }", "  b = 2 // }", "  /* {", "  } } */", "  c = 3", "}", "",
			"resource \"test\" \"beta\" {", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 10}}},
		{"nix hash comment and update operator", getNixBlocks, []string{
			"alpha = {", "  a = 1; # }", "  b = x // {", "    c = 1;", "  };", "  /* } */", "};", "",
			"beta = {", "  d = 1;", "};"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 11}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mlCheck(t, c.scan(c.src), c.want)
		})
	}
}
