package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// mcpDestTools maps the MCP tools that accept a dest list to the fedit op they
// run. A call without dest keeps the single-file path.
var mcpDestTools = map[string]string{
	"fedit_copy":    "copy",
	"fedit_move":    "move",
	"fedit_replace": "replace",
}

// mcpMultiDestOpts builds the options for a cross-file transfer from the MCP
// arguments. The second return value is an error message, empty when the
// request is valid. Same rules as the CLI: nothing is read or written before
// the request has been validated.
func mcpMultiDestOpts(name, file string, getStr func(string) string, getInt func(string, int) int) (multiDestOpts, string) {
	op := mcpDestTools[name]
	o := multiDestOpts{
		op:          op,
		srcFile:     file,
		lang:        getStr("lang"),
		block:       getStr("block"),
		destBlock:   getStr("destblock"),
		line:        getInt("line", 0),
		end:         getInt("end", 0),
		match:       getStr("match"),
		endmatch:    getStr("endmatch"),
		nth:         getInt("nth", 1),
		times:       getInt("times", 1),
		after:       getInt("after", -1),
		before:      getInt("before", -1),
		aftermatch:  getStr("aftermatch"),
		beforematch: getStr("beforematch"),
		afterblock:  getStr("afterblock"),
		beforeblock: getStr("beforeblock"),
	}
	if isMultiSpec(file) {
		return o, "dest does not combine with a multi file list: file must be the single source file"
	}
	var dests []string
	for _, d := range parseDestList(getStr("dest")) {
		if abs, err := filepath.Abs(d); err == nil {
			d = abs
		}
		dests = append(dests, d)
	}
	if len(dests) == 0 {
		return o, "dest needs at least one file"
	}
	o.dests = strings.Join(dests, ",")
	switch op {
	case "replace":
		if o.destBlock == "" {
			return o, "replace with dest requires destblock"
		}
	case "copy", "move":
		if o.destBlock != "" {
			return o, "destblock is only supported with fedit_replace"
		}
	}
	return o, ""
}

// mcpRunMultiDest runs a validated cross-file transfer and formats the
// per-destination report as the tool result. A problem in one destination is a
// SKIP line and that file is left unmodified; the result is an error only if
// the request failed or no destination succeeded.
func mcpRunMultiDest(o multiDestOpts) mcpCallResult {
	var out multiDestOutcome
	var err error
	label := o.op
	if o.op == "replace" {
		out, err = collectMultiDestReplace(o)
	} else {
		out, err = collectMultiDest(o)
	}
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("dest: %v", err))
	}
	var b strings.Builder
	for _, r := range out.results {
		fmt.Fprintf(&b, "  %-6s %s  %s\n", r.status, r.path, r.detail)
	}
	for _, n := range out.notes {
		fmt.Fprintf(&b, "%s\n", n)
	}
	fmt.Fprintf(&b, "%s: %d/%d file(s) succeeded", label, out.okCount, len(out.results))
	if out.failed {
		return mcpErrorResult(b.String())
	}
	return mcpOK(b.String())
}

// mcpSourceBlockContent resolves a named block from sourcefile for
// fedit_insertafter and fedit_insertbefore, the MCP twin of the CLI
// -sourcefile/-sourceblock flags. The second return value is an error message,
// empty on success.
func mcpSourceBlockContent(sourceFile, sourceBlock, lang string) ([]string, string) {
	if sourceFile == "" || sourceBlock == "" {
		return nil, "sourcefile and sourceblock must be used together"
	}
	if lang == "" {
		return nil, "sourcefile and sourceblock require lang"
	}
	if abs, err := filepath.Abs(sourceFile); err == nil {
		sourceFile = abs
	}
	lines, err := readLines(sourceFile)
	if err != nil {
		return nil, fmt.Sprintf("sourcefile: %v", err)
	}
	s, e, err := resolveBlock(lines, lang, sourceBlock)
	if err != nil {
		return nil, fmt.Sprintf("sourceblock: %v", err)
	}
	block := make([]string, e-s+1)
	copy(block, lines[s-1:e])
	return block, ""
}
