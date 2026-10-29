package main

import "strings"

// mcpDryVerbs maps the past tense lead word of a tool summary to its preview form.
var mcpDryVerbs = [][2]string{{"Inserted ", "Would insert "}, {"Deleted ", "Would delete "}, {"Replaced ", "Would replace "}, {"Moved ", "Would move "}, {"Copied ", "Would copy "}, {"Wrote ", "Would write "}}

// mcpDryWould turns the summary of a tool that did not run for real into a
// preview: the lead verb goes to Would ... and the first line no longer claims
// a total now. Text that starts with another word is left alone.
func mcpDryWould(text string) string {
	for _, v := range mcpDryVerbs {
		if strings.HasPrefix(text, v[0]) {
			first, rest := text, ""
			if i := strings.Index(text, "\n"); i >= 0 {
				first, rest = text[:i], text[i:]
			}
			first = v[1] + strings.TrimPrefix(first, v[0])
			return strings.Replace(first, " total now)", " total after)", 1) + rest
		}
	}
	return text
}
