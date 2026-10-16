package main

import "testing"

// Rust lifetimes, loop labels and char literals must not be read as strings
// (a lifetime has no closing quote and used to swallow the rest of its line).
func TestRustQuoteBlocks(t *testing.T) {
	cases := []struct {
		name string
		src  []string
		want []mlWant
	}{
		{"lifetimes and a brace char", []string{
			"fn alpha<'a>(x: &'a str) -> &'a str {", "    let c = '{';", "    x", "}", "",
			"fn beta() {", "    let d = 1;", "}"},
			[]mlWant{{"alpha", 1, 4}, {"beta", 6, 8}}},
		{"escaped chars", []string{
			"fn alpha() {", "    let a = '\\'';", "    let b = '}';", "    let c = '\\u{7d}';", "    let d = '\\\\';", "}", "",
			"fn beta() {", "}"},
			[]mlWant{{"alpha", 1, 6}, {"beta", 8, 9}}},
		{"static and loop labels", []string{
			"fn alpha() -> &'static str {", "    'outer: loop {", "        break 'outer;", "    }", "    \"x}\"", "}", "",
			"fn beta() {", "}"},
			[]mlWant{{"alpha", 1, 6}, {"beta", 8, 9}}},
		{"multibyte char", []string{
			"fn alpha() {", "    let a = '\u2192';", "    let b = '}';", "}", "",
			"fn beta() {", "}"},
			[]mlWant{{"alpha", 1, 4}, {"beta", 6, 7}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mlCheck(t, getRustBlocks(c.src), c.want)
		})
	}
}
