package main

import "strconv"

// Result fields the MCP tool handlers note for the end record.

// logNoteLines records how many lines an operation changed. A dry run changes nothing, so it notes nothing.
func logNoteLines(n int) {
	if !dryRun {
		logNote("lines_changed", strconv.Itoa(n))
	}
}

// logNoteReplacements records the occurrences replaced and the lines changed by
// replaceall over MCP, the same two fields the command line notes. A dry run
// changes nothing, so it notes neither.
func logNoteReplacements(occurrences, lines int) {
	if !dryRun {
		logNoteReplace(occurrences, lines)
	}
}

// logCountOccurrences sums a per-line match counter over lines.
func logCountOccurrences(lines []string, count func(string) int) int {
	n := 0
	for _, l := range lines {
		n += count(l)
	}
	return n
}

// logNoteVerdict records a verification result as PASS or FAIL.
func logNoteVerdict(ok bool) {
	if ok {
		logNoteVerify("PASS")
		return
	}
	logNoteVerify("FAIL")
}
