package main

import (
	"os"
	"strings"
	"testing"
)

// Binary tests for the regex replaceall dispatch in main(): a capture-group
// replacement must be verified on disk (VERIFY: PASS, exit 0), and the
// failure paths must exit 1 and leave the file untouched.

func wrrRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

func TestWRR_Binary(t *testing.T) {
	bin := clhBuild(t)
	const orig = "id=1 name=ann\nskip\nid=2 name=bob\n"

	t.Run("capture_group_pass", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match-regex", `id=(\d+) name=(\w+)`, "-text", "$2:$1", "-v")
		if code != 0 {
			t.Fatalf("exit %d, want 0:\n%s", code, out)
		}
		for _, w := range []string{"VERIFY: PASS", "Regex-replaced on 2 line(s) (3 total)"} {
			if !strings.Contains(out, w) {
				t.Errorf("stderr lacks %q:\n%s", w, out)
			}
		}
		if got := wrrRead(t, p); got != "ann:1\nskip\nbob:2\n" {
			t.Errorf("file = %q", got)
		}
	})

	t.Run("no_match_exit1", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match-regex", `zzz\d+`, "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != orig {
			t.Errorf("file changed: %q", got)
		}
	})

	t.Run("invalid_regex_exit1", func(t *testing.T) {
		p := wraFile(t, orig)
		out, code := wraRun(t, bin, "-file", p, "-op", "replaceall", "-match-regex", "(", "-text", "Y")
		if code != 1 {
			t.Fatalf("exit %d, want 1:\n%s", code, out)
		}
		if got := wrrRead(t, p); got != orig {
			t.Errorf("file changed: %q", got)
		}
	})
}
