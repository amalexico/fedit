package main

import "testing"

// TestMultilineStringBlocksMore covers the PHP heredoc and # comment gaps and
// the Java text block escape rules (task 8 follow-ups).
func TestMultilineStringBlocksMore(t *testing.T) {
	cases := []struct {
		name string
		scan func([]string) []blockEntry
		src  []string
		want []mlWant
	}{
		{"php heredoc", getPHPBlocks, []string{
			"function alpha() {", "    $a = <<<EOT", "    } }", "    {{", "EOT;", "    $z = 1;", "}", "",
			"function beta() {", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 10}}},
		{"php nowdoc", getPHPBlocks, []string{
			"function alpha() {", "    $a = <<<'EOT'", "    } }", "    {{", "EOT;", "    $z = 1;", "}", "",
			"function beta() {", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 10}}},
		{"php quoted heredoc tag", getPHPBlocks, []string{
			"function alpha() {", `    $a = <<<"EOT"`, "        } }", "        {{", "        EOT;", "    $z = 1;", "}", "",
			"function beta() {", "}"},
			[]mlWant{{"alpha", 1, 7}, {"beta", 9, 10}}},
		{"php heredoc tag prefix in body", getPHPBlocks, []string{
			"function alpha() {", "    $a = <<<EOT", "EOTX {{", "EOT;", "}", "",
			"function beta() {", "}"},
			[]mlWant{{"alpha", 1, 5}, {"beta", 7, 8}}},
		{"php hash and slash comments", getPHPBlocks, []string{
			"function alpha() {", "    # } }", "    // {{", "    $a = 1;", "}"},
			[]mlWant{{"alpha", 1, 5}}},
		{"php attribute is not a comment", getPHPBlocks, []string{
			"function alpha() {", "    #[Attr] $f = function () {", "    };", "}"},
			[]mlWant{{"alpha", 1, 4}}},
		{"java text block escaped triple quote", getJavaBlocks, []string{
			"class Alpha {", `    String a = """`, `        say \""" } }`, "        { {", `        """;`, "    int z = 1;", "}", "",
			"class Beta {", "}"},
			[]mlWant{{"Alpha", 1, 7}, {"Beta", 9, 10}}},
		{"java text block escaped backslash then close", getJavaBlocks, []string{
			"class Alpha {", `    String a = """`, "        } {", `        path\\""";`, "    int z = 1;", "}"},
			[]mlWant{{"Alpha", 1, 6}}},
		{"csharp raw string backslash is literal", getCSharpBlocks, []string{
			"class Alpha {", `    string a = """`, `        path\""";`, "    int z = 1;", "}"},
			[]mlWant{{"Alpha", 1, 5}}},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			mlCheck(t, c.scan(c.src), c.want)
		})
	}
}
