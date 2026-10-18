package main

import "strings"

// flattenResult handles a multi-line replacement. A replacement that contains a
// newline leaves result elements such as "a 1\n2", but the written file then
// holds more lines than the slice. flattenResult splits those elements so Lines
// matches what the file will hold, and renumbers Changed to the new file's line
// numbers. Occurrences is unchanged. Results with no newline come back as is.
func flattenResult(res replaceAllResult) replaceAllResult {
	multi := false
	for _, l := range res.Lines {
		if strings.Contains(l, "\n") {
			multi = true
			break
		}
	}
	if !multi {
		return res
	}
	out := replaceAllResult{Lines: make([]string, 0, len(res.Lines)), Occurrences: res.Occurrences}
	ci := 0
	for i, l := range res.Lines {
		isChanged := ci < len(res.Changed) && res.Changed[ci] == i+1
		if isChanged {
			ci++
		}
		for _, piece := range strings.Split(l, "\n") {
			out.Lines = append(out.Lines, piece)
			if isChanged {
				out.Changed = append(out.Changed, len(out.Lines))
			}
		}
	}
	return out
}
