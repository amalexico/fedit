package main

import "testing"

// A JavaScript regex literal must not be read as a comment opener or as code:
// a "/*" or a brace inside a regex used to swallow or unbalance the rest of the
// file, so -block found only one top-level block (map was not affected).
func TestJSR_Blocks(t *testing.T) {
	cases := []struct {
		name string
		src  []string
		want []mlWant
	}{
		{"escaped slash then star", []string{
			"function a() {", "  var r = /a\\/*b/;", "  return r;", "}", "",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 6, 7}}},
		{"star inside a char class", []string{
			"function a() {", "  return /[/*]/.test(1);", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 3}, {"b", 4, 5}}},
		{"braces inside a regex", []string{
			"function a() {", "  var r = /\\{/;", "  var q = /a{2}/;", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 5, 6}}},
		{"closing brace inside a regex after return", []string{
			"function a() {", "  return /}/.test(s);", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 3}, {"b", 4, 5}}},
		{"regex after a comma and a paren", []string{
			"function a() {", "  f(x, /[{]/g, 1);", "  g(/}/);", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 5, 6}}},
		{"division stays division", []string{
			"function a() {", "  var x = n / 2; // half", "  var y = m / 3 / 4;", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 5, 6}}},
		{"a real block comment is still a comment", []string{
			"function a() {", "  /* { */", "  var x = 1;", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 5, 6}}},
		{"block comment after a division", []string{
			"function a() {", "  var x = n /* { */ / 2;", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 3}, {"b", 4, 5}}},
		{"slash without a closing slash is division", []string{
			"function a() {", "  var x = (n + 1) / 2;", "  var y = z[0] / 3;", "}",
			"function b() {", "}"},
			[]mlWant{{"a", 1, 4}, {"b", 5, 6}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mlCheck(t, getJSBlocks(c.src), c.want)
		})
	}
}

// Other brace-tracked languages keep their behaviour: a slash there is never a
// regex literal, so a Java line with a division and a comment is unchanged.
func TestJSR_OtherLangsUnchanged(t *testing.T) {
	var s strState
	if got := s.strip("int x = a / b; /* { */ int y = 1;", "java"); got != "int x = a / b;  int y = 1;" {
		t.Errorf("java strip = %q", got)
	}
	var r strState
	if got := r.strip("let x = /{/;", "rust"); got != "let x = /{/;" {
		t.Errorf("rust strip = %q", got)
	}
}
