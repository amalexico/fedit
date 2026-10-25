package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// dryRun is set by -dry: writeLines then prints a unified diff and writes nothing.
var dryRun bool

// dryOut receives the diff; nil means os.Stdout at call time (so an
// -outputresult capture that swaps os.Stdout still sees the diff).
var dryOut io.Writer

const (
	dryContext = 3
	dryLCSCap  = 4000000
)

type dryEdit struct {
	kind byte // ' ' same, '-' removed, '+' added
	text string
}

func drySliceEq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// dryEdits returns the edit script from a to b. Common head and tail are
// trimmed first; a middle too large for the LCS table becomes one replace.
func dryEdits(a, b []string) []dryEdit {
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	ma, mb := a[p:len(a)-s], b[p:len(b)-s]
	var out []dryEdit
	for _, l := range a[:p] {
		out = append(out, dryEdit{' ', l})
	}
	if len(ma)*len(mb) > dryLCSCap {
		for _, l := range ma {
			out = append(out, dryEdit{'-', l})
		}
		for _, l := range mb {
			out = append(out, dryEdit{'+', l})
		}
	} else {
		n, m := len(ma), len(mb)
		t := make([][]uint32, n+1)
		for i := range t {
			t[i] = make([]uint32, m+1)
		}
		for i := n - 1; i >= 0; i-- {
			for j := m - 1; j >= 0; j-- {
				if ma[i] == mb[j] {
					t[i][j] = t[i+1][j+1] + 1
				} else if t[i+1][j] >= t[i][j+1] {
					t[i][j] = t[i+1][j]
				} else {
					t[i][j] = t[i][j+1]
				}
			}
		}
		i, j := 0, 0
		for i < n || j < m {
			switch {
			case i < n && j < m && ma[i] == mb[j]:
				out = append(out, dryEdit{' ', ma[i]})
				i++
				j++
			case i < n && (j == m || t[i+1][j] >= t[i][j+1]):
				out = append(out, dryEdit{'-', ma[i]})
				i++
			default:
				out = append(out, dryEdit{'+', mb[j]})
				j++
			}
		}
	}
	for _, l := range a[len(a)-s:] {
		out = append(out, dryEdit{' ', l})
	}
	return out
}

func dryRange(start, count int) string {
	if count == 1 {
		return fmt.Sprintf("%d", start)
	}
	return fmt.Sprintf("%d,%d", start, count)
}

// dryDiff returns a unified diff of a to b (3 lines of context), or "" when
// the names are equal and the lines are equal.
func dryDiff(oldName, newName string, a, b []string) string {
	if oldName == newName && drySliceEq(a, b) {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- %s\n+++ %s\n", oldName, newName)
	ops := dryEdits(a, b)
	oldNo := make([]int, len(ops)+1)
	newNo := make([]int, len(ops)+1)
	for i, e := range ops {
		oldNo[i+1], newNo[i+1] = oldNo[i], newNo[i]
		if e.kind != '+' {
			oldNo[i+1]++
		}
		if e.kind != '-' {
			newNo[i+1]++
		}
	}
	type span struct{ s, e int }
	var spans []span
	for i, e := range ops {
		if e.kind == ' ' {
			continue
		}
		s, en := i-dryContext, i+dryContext
		if s < 0 {
			s = 0
		}
		if en > len(ops)-1 {
			en = len(ops) - 1
		}
		if n := len(spans); n > 0 && s <= spans[n-1].e+1 {
			spans[n-1].e = en
		} else {
			spans = append(spans, span{s, en})
		}
	}
	for _, sp := range spans {
		oc := oldNo[sp.e+1] - oldNo[sp.s]
		nc := newNo[sp.e+1] - newNo[sp.s]
		oldStart, newStart := oldNo[sp.s]+1, newNo[sp.s]+1
		if oc == 0 {
			oldStart--
		}
		if nc == 0 {
			newStart--
		}
		fmt.Fprintf(&sb, "@@ -%s +%s @@\n", dryRange(oldStart, oc), dryRange(newStart, nc))
		for _, e := range ops[sp.s : sp.e+1] {
			sb.WriteByte(e.kind)
			sb.WriteString(e.text)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// dryWrite is writeLines under -dry: it compares lines with the file on disk,
// prints the diff and writes nothing. A missing file is shown as /dev/null.
func dryWrite(path string, lines []string) error {
	oldName := path
	var old []string
	if _, serr := os.Stat(path); errors.Is(serr, os.ErrNotExist) {
		oldName = "/dev/null"
	} else {
		var err error
		if old, err = readLines(path); err != nil {
			return err
		}
	}
	d := dryDiff(oldName, path, old, lines)
	if d == "" {
		return nil
	}
	w := dryOut
	if w == nil {
		w = os.Stdout
	}
	_, err := io.WriteString(w, d)
	return err
}
