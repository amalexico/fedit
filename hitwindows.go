package main

import (
	"fmt"
	"strings"
)

// formatHitWindows renders a context window for every changed line, in the
// same layout as showVerify. The number of hits is never capped: windows that
// overlap or touch are merged, distant windows are separated by a "  ..." line,
// and every changed line is marked with "> ". changed must be ascending
// 1-based line numbers; entries outside the file are ignored. It works from the
// lines already in memory and makes one pass over the hit list.
func formatHitWindows(lines []string, changed []int, before, after int) string {
	type window struct{ start, end int }
	var ws []window
	for _, c := range changed {
		if c < 1 || c > len(lines) {
			continue
		}
		s, e := c-before, c+after
		if s < 1 {
			s = 1
		}
		if e > len(lines) {
			e = len(lines)
		}
		if n := len(ws); n > 0 && s <= ws[n-1].end+1 {
			if e > ws[n-1].end {
				ws[n-1].end = e
			}
		} else {
			ws = append(ws, window{s, e})
		}
	}
	var b strings.Builder
	b.WriteString("\n=== VERIFY ===\n")
	hi := 0
	for wi, w := range ws {
		if wi > 0 {
			b.WriteString("  ...\n")
		}
		for ln := w.start; ln <= w.end; ln++ {
			for hi < len(changed) && changed[hi] < ln {
				hi++
			}
			marker := "  "
			if hi < len(changed) && changed[hi] == ln {
				marker = "> "
			}
			fmt.Fprintf(&b, "%s%4d | %s\n", marker, ln, lines[ln-1])
		}
	}
	fmt.Fprintf(&b, "--- %d total lines ---\n", len(lines))
	return b.String()
}
