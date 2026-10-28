package main

import (
	"strings"
	"testing"
)

// TestUsageMentionsDry: the usage text of a bare fedit run lists -dry, says
// what it does and says that it cannot be combined with -config or -log merge.
func TestUsageMentionsDry(t *testing.T) {
	bin := g2Binary(t)
	_, se, code := orxRun(t, bin)
	if code != 1 {
		t.Fatalf("a bare run should exit 1, got %d", code)
	}
	line := ""
	for _, l := range strings.Split(se, "\n") {
		if strings.HasPrefix(strings.TrimSpace(l), "-dry") {
			line = l
		}
	}
	if line == "" {
		t.Fatalf("the usage text has no -dry line:\n%s", se)
	}
	for _, w := range []string{"unified diff", "applied=false", "-config", "-log merge"} {
		if !strings.Contains(line, w) {
			t.Errorf("the -dry line lacks %q: %q", w, line)
		}
	}
}
