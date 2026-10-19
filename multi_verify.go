package main

import (
	"fmt"
	"regexp"
	"strings"
)

// multiWriteLines is the writer the multi -file mutation uses for each file.
// Tests replace it to inject a write that stores the wrong content.
var multiWriteLines = writeLines

// expectedDiskLines splits every element that contains a newline, so the
// expected lines match the physical lines readLines returns for the file.
func expectedDiskLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, strings.Split(l, "\n")...)
	}
	return out
}

// multiReplaceAllCounter returns the function that counts matches of the old
// text (literal or regex) in one line.
func multiReplaceAllCounter(o multiMutOpts) func(string) int {
	if o.matchRegex != "" {
		re, err := regexp.Compile(o.matchRegex)
		if err != nil {
			return func(string) int { return 0 }
		}
		return func(l string) int { return len(re.FindAllStringIndex(l, -1)) }
	}
	return func(l string) int {
		if o.match == "" {
			return 0
		}
		return strings.Count(l, o.match)
	}
}

// verifyMultiReplaceAll re-reads path and compares it with the expected lines.
// It returns whether the file is exactly as expected and the text for the
// per-file report line.
func verifyMultiReplaceAll(path string, expected []string, count func(string) int) (bool, string) {
	v, err := verifyOnDiskWith(path, replaceAllResult{Lines: expectedDiskLines(expected)}, count)
	if err != nil {
		return false, fmt.Sprintf("VERIFY: could not re-read: %v", err)
	}
	if v.OK {
		return true, "VERIFY: PASS"
	}
	return false, fmt.Sprintf("VERIFY: FAIL (mismatched lines %s; old text remaining %d, expected %d; lines on disk %d, expected %d)",
		formatLineRanges(v.Bad), v.Remaining, v.WantRemaining, v.GotLines, v.WantLines)
}
