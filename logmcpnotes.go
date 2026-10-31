package main

import "strconv"

// Result fields the MCP tool handlers note for the end record.

// logNoteLines records how many lines an operation changed. A dry run changes nothing, so it notes nothing.
func logNoteLines(n int) {
	if !dryRun {
		logNote("lines_changed", strconv.Itoa(n))
	}
}

// logNoteVerdict records a verification result as PASS or FAIL.
func logNoteVerdict(ok bool) {
	if ok {
		logNoteVerify("PASS")
		return
	}
	logNoteVerify("FAIL")
}
