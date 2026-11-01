package main

import (
	"fmt"
	"os"
	"strings"
)

// dryText turns the past tense summary of a command line run into a preview
// when the run is a dry run: nothing was done, so it must not say it was. The
// MCP wording rules (mcpDryWould) cover the lead verbs they know; the regex and
// stream runners lead with Regex-replaced and Stream-replaced, which get a
// preview form of their own. Outside a dry run the text is returned unchanged.
func dryText(s string) string {
	if !dryRun {
		return s
	}
	for _, v := range [][2]string{{"Regex-replaced ", "Would regex-replace "}, {"Stream-replaced ", "Would stream-replace "}} {
		if strings.HasPrefix(s, v[0]) {
			return v[1] + strings.TrimPrefix(s, v[0])
		}
	}
	return mcpDryWould(s)
}

// summaryf prints a summary line to stderr, worded as a preview under -dry.
func summaryf(format string, a ...any) {
	fmt.Fprint(os.Stderr, dryText(fmt.Sprintf(format, a...)))
}
