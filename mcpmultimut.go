package main

import (
	"fmt"
	"strings"
)

// mcpMultiMutTools maps the MCP tools that accept a multi file list to the
// fedit op they run. fedit_write is deliberately absent: a write to a file whose
// name contains a comma must keep creating that file.
var mcpMultiMutTools = map[string]string{
	"fedit_replaceall":   "replaceall",
	"fedit_insertafter":  "insertafter",
	"fedit_insertbefore": "insertbefore",
	"fedit_delete":       "delete",
	"fedit_replace":      "replace",
}

// mcpMultiMutOpts builds the options for a multi file mutation from the MCP
// arguments. The second return value is an error message, empty when the request
// is valid. Same rules as the CLI: match based forms only, never a line number or
// block, and nothing is read or written before the request has been validated.
func mcpMultiMutOpts(name, spec string, getStr func(string) string, getInt func(string, int) int, getBool func(string) bool) (multiMutOpts, string) {
	op := mcpMultiMutTools[name]
	o := multiMutOpts{
		spec:     spec,
		op:       op,
		match:    getStr("match"),
		endmatch: getStr("endmatch"),
		nth:      getInt("nth", 1),
	}
	if getStr("block") != "" {
		return o, "block is not supported with a multi file list; use match"
	}
	if getInt("line", 0) != 0 || getInt("end", 0) != 0 {
		return o, "line numbers are not supported with a multi file list; use match"
	}
	text := getStr("text")
	switch op {
	case "replaceall":
		if getBool("stream") || getStr("files") != "" {
			return o, "stream and files are not supported with a multi file list"
		}
		o.matchRegex = getStr("match_regex")
		o.replacement = text
	case "insertafter", "insertbefore", "replace":
		o.newLines = expandText(text)
	}
	if o.match == "" && o.matchRegex == "" {
		return o, op + " requires match"
	}
	if (op == "insertafter" || op == "insertbefore" || op == "replace") && len(o.newLines) == 0 {
		return o, "Nothing to insert"
	}
	return o, ""
}

// mcpRunMultiMutate applies a validated multi file mutation and formats the
// per-file report as the tool result. A problem in one file is a SKIP line and
// that file is left unmodified; the result is an error only if no file succeeded.
func mcpRunMultiMutate(o multiMutOpts) mcpCallResult {
	results, okCount, err := collectMultiMutate(o)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("multi file: %v", err))
	}
	var b strings.Builder
	for _, r := range results {
		fmt.Fprintf(&b, "  %-6s %s  %s\n", r.status, r.path, r.detail)
	}
	fmt.Fprintf(&b, "multi file %s: %d/%d file(s) succeeded", o.op, okCount, len(results))
	if okCount == 0 {
		return mcpErrorResult(b.String())
	}
	return mcpOK(b.String())
}
