package main

import (
	"strings"
	"testing"
)

// -dry with -config or -log merge is a usage error, whatever the order.
func TestDRC_Conflict(t *testing.T) {
	cases := []struct {
		args []string
		want bool
	}{
		{[]string{"fedit", "-dry", "-config", "show"}, true},
		{[]string{"fedit", "-config", "show", "-dry"}, true},
		{[]string{"fedit", "-config", "-dry", "show"}, true},
		{[]string{"fedit", "-dry", "-log", "merge"}, true},
		{[]string{"fedit", "-log", "merge", "-dry"}, true},
		{[]string{"fedit", "--dry", "-config", "path"}, true},
		{[]string{"fedit", "-config", "-dry=true", "show"}, true},
		{[]string{"fedit", "-dry", "-file", "x", "-op", "show"}, false},
		{[]string{"fedit", "-config", "show"}, false},
		{[]string{"fedit", "-log", "merge"}, false},
		{[]string{"fedit", "-file", "-config", "-op", "find"}, false},
		{[]string{"fedit", "-dry"}, false},
		{[]string{"fedit", "mcp"}, false},
		{[]string{"fedit"}, false},
		{nil, false},
	}
	for _, c := range cases {
		if got := dryConflict(c.args); got != c.want {
			t.Errorf("dryConflict(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

// drbBinDryConflict: the binary refuses the combination with exit 1 and one
// clear message, in either order (before, a leading -dry gave the flag
// package's error and a usage dump with exit 2).
func drbBinDryConflict(t *testing.T, bin string) {
	for _, a := range [][]string{
		{"-dry", "-config", "show"},
		{"-config", "show", "-dry"},
		{"-dry", "-log", "merge"},
		{"-log", "merge", "-dry"},
	} {
		_, se, code := orxRun(t, bin, a...)
		if code != 1 {
			t.Errorf("%q: exit %d, want 1 (stderr %q)", a, code, se)
		}
		if !strings.Contains(se, "-dry cannot be combined with -config or -log merge") {
			t.Errorf("%q: stderr lacks the usage message: %q", a, se)
		}
	}
}
