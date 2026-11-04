package main

import "fmt"

// resolveMatchRange resolves the line range of a content-anchored replace or
// delete. -endmatch wins; otherwise a nonzero endBound ends the range at that
// line (a negative bound counts from the end of the file); with neither, the
// range is the single matched line. Errors name -match, not "source", because
// replace and delete have no source. move, copy and show keep their own
// resolver (resolveSourceLines), which still demands an end bound.
func resolveMatchRange(lines []string, match, endmatch string, endBound, nth int) (start, end int, err error) {
	start, err = resolveNth(findMatches(lines, match), nth)
	if err != nil {
		return 0, 0, fmt.Errorf("-match %q: %v", match, err)
	}
	switch {
	case endmatch != "":
		for _, h := range findMatches(lines, endmatch) {
			if h >= start {
				return start, h, nil
			}
		}
		return 0, 0, fmt.Errorf("-endmatch %q: no match at or after line %d", endmatch, start)
	case endBound != 0:
		end = endBound
		if end < 0 {
			end = len(lines) + 1 + end
		}
		if end < start {
			return 0, 0, fmt.Errorf("-end %d is before the -match line %d", end, start)
		}
		return start, end, nil
	}
	return start, start, nil
}

// cliEndBound returns the end line for resolveMatchRange from the CLI: endN is
// parseAndResolveLines' result, which is only an end bound when -end was given
// (without -end it is the -line value, which -match ignores).
func cliEndBound(endFlag string, endN int) int {
	if endFlag == "" {
		return 0
	}
	return endN
}

// matchEndBound returns the end line for resolveMatchRange from the MCP
// delete, whose dispatcher defaults end to line: an end equal to the start
// means no end was given.
func matchEndBound(startLine, endLine int) int {
	if endLine == startLine {
		return 0
	}
	return endLine
}
