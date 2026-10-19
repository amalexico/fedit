package main

import (
	"strings"
	"testing"
)

// Binary tests for the -stream replaceall paths: after the streaming rewrite
// the file is read again and the same VERIFY verdict as the in-memory runner
// is printed (PASS, occurrences, complete changed-line list, exit 0). The
// failure paths exit 1 and leave the file untouched.

func TestWRS_Binary(t *testing.T) {
	bin := clhBuild(t)
	const orig = "a x\nb\nx x\nc\nx\n"
	const origRe = "id=1 name=ann\nskip\nid=2 name=bob\n"

	t.Run("literal_stream_pass", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-stream", "-match", "x", "-text", "Y")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		for _, w := range []string{"VERIFY: PASS", "4 occurrence(s)", "changed lines: 1,3,5"} {
			if !strings.Contains(out, w) {
				t.Errorf("stderr lacks %q:\n%s", w, out)
			}
		}
		if got := wrrRead(t, p); got != "a Y\nb\nY Y\nc\nY\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("regex_stream_pass", func(t *testing.T) {
		p := wraFile(t, origRe)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-stream", "-match-regex", `id=(\d+) name=(\w+)`, "-text", "$2:$1")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		for _, w := range []string{"VERIFY: PASS", "changed lines: 1,3"} {
			if !strings.Contains(out, w) {
				t.Errorf("stderr lacks %q:\n%s", w, out)
			}
		}
		if got := wrrRead(t, p); got != "ann:1\nskip\nbob:2\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("literal_stream_no_match_exit1", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-stream", "-match", "zzz", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != orig {
			t.Errorf("file changed: %q", got)
		}
	})

	t.Run("regex_stream_invalid_exit1", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-stream", "-match-regex", "(", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != orig {
			t.Errorf("file changed: %q", got)
		}
	})
}
