package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func cssCapture(t *testing.T, lines []string) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()
	done := make(chan string)
	go func() {
		var b bytes.Buffer
		io.Copy(&b, r)
		done <- b.String()
	}()
	doMapCSSSelectors(lines)
	w.Close()
	return <-done
}

func TestCSSBraces_SkipsStringsAndComments(t *testing.T) {
	ev, in := cssBraces(`a::after { content: "}"; } /* { */`, false)
	if string(ev) != "{}" || in {
		t.Errorf("got %q inComment=%v, want {} and false", ev, in)
	}
	ev, in = cssBraces("/* start {", false)
	if len(ev) != 0 || !in {
		t.Errorf("open comment: got %q inComment=%v", ev, in)
	}
	ev, in = cssBraces("end } */ .a {", true)
	if string(ev) != "{" || in {
		t.Errorf("close comment: got %q inComment=%v, want { and false", ev, in)
	}
}

func TestCSSDups_MediaScoped(t *testing.T) {
	out := cssCapture(t, []string{
		".btn { color: red; }",
		"@media (max-width: 600px) {",
		"  .btn { color: blue; }",
		"}",
		"@media (min-width: 900px) {",
		"  .btn { color: green; }",
		"}",
		".card { margin: 0; }",
		".card { padding: 0; }",
	})
	if !strings.Contains(out, ".card (2 times)") {
		t.Errorf("a real top-level duplicate must still be reported, got %q", out)
	}
	if strings.Contains(out, ".btn (") {
		t.Errorf(".btn in separate @media blocks is not a duplicate, got %q", out)
	}
}

func TestCSSDups_SameMediaBlockReported(t *testing.T) {
	out := cssCapture(t, []string{
		"@media (max-width: 600px) {",
		"  .a { color: red; }",
		"  .a { color: blue; }",
		"}",
	})
	if !strings.Contains(out, ".a (2 times, inside @media (max-width: 600px))") {
		t.Errorf("want a duplicate inside the media block, got %q", out)
	}
}

func TestCSSDups_KeyframesFromToNotDuplicates(t *testing.T) {
	out := cssCapture(t, []string{
		"@keyframes a {",
		"  from { opacity: 0; }",
		"  to { opacity: 1; }",
		"}",
		"@keyframes b {",
		"  from { opacity: 1; }",
		"  to { opacity: 0; }",
		"}",
	})
	if !strings.Contains(out, "No duplicate selectors") {
		t.Errorf("from/to in different keyframes are not duplicates, got %q", out)
	}
}

func TestCSSDups_SameKeyframesNameTwiceReported(t *testing.T) {
	out := cssCapture(t, []string{
		"@keyframes spin {",
		"  to { opacity: 1; }",
		"}",
		"@keyframes spin {",
		"  to { opacity: 0; }",
		"}",
	})
	if !strings.Contains(out, "@keyframes spin (2 times)") {
		t.Errorf("a repeated @keyframes name must be reported, got %q", out)
	}
}

func TestCSSDups_BraceOnNextLineMedia(t *testing.T) {
	out := cssCapture(t, []string{
		"@media (max-width: 600px)",
		"{",
		"  .a { color: red; }",
		"}",
		".a { color: blue; }",
	})
	if !strings.Contains(out, "No duplicate selectors") {
		t.Errorf(".a inside and outside @media is not a duplicate, got %q", out)
	}
}

func TestCSSDups_CommentedSelectorIgnored(t *testing.T) {
	out := cssCapture(t, []string{
		"/*",
		".a { color: red; }",
		"*/",
		".a { color: blue; }",
	})
	if !strings.Contains(out, "No duplicate selectors") {
		t.Errorf("a selector inside a comment must not count, got %q", out)
	}
}

func TestCLICSSMap_MediaScopedDuplicates(t *testing.T) {
	bin := clhBuild(t)
	dir := t.TempDir()
	clhWrite(t, dir, "a.css", ".btn { color: red; }\n@media (max-width: 600px) {\n  .btn { color: blue; }\n}\n@media (min-width: 900px) {\n  .btn { color: green; }\n}\n.card { margin: 0; }\n.card { padding: 0; }\n")
	so, _, code := clhRun(t, bin, dir, "-file", "a.css", "-op", "map")
	if code != 0 {
		t.Fatalf("map exit code %d", code)
	}
	if strings.Contains(so, ".btn (") {
		t.Errorf("map must not report .btn across @media blocks, got %q", so)
	}
	if !strings.Contains(so, ".card (2 times)") {
		t.Errorf("map must still report the real .card duplicate, got %q", so)
	}
}
