package main

import (
	"fmt"
	"os"
	"strings"
)

// multiDestOpts holds the flags for cross-file copy, move and replace (-dest fan-out).
type multiDestOpts struct {
	op          string
	srcFile     string
	dests       string
	lang        string
	block       string
	destBlock   string
	line        int
	end         int
	match       string
	endmatch    string
	nth         int
	times       int
	after       int
	before      int
	aftermatch  string
	beforematch string
	afterblock  string
	beforeblock string
}

// multiDestOne inserts block into one destination's lines. It returns the new
// lines and a short description of where the block landed. It never writes a
// file and never calls os.Exit.
func multiDestOne(lines []string, o multiDestOpts, block []string) ([]string, string, error) {
	aF, bF, aM, bM := o.after, o.before, o.aftermatch, o.beforematch
	if o.beforeblock != "" {
		bs, _, err := resolveBlock(lines, o.lang, o.beforeblock)
		if err != nil {
			return nil, "", err
		}
		bF, aF, aM, bM = bs, -1, "", ""
	}
	if o.afterblock != "" {
		_, be, err := resolveBlock(lines, o.lang, o.afterblock)
		if err != nil {
			return nil, "", err
		}
		aF, bF, aM, bM = be, -1, "", ""
	}
	destAfter, _, desc, err := resolveDestLine(lines, aF, bF, aM, bM)
	if err != nil {
		return nil, "", err
	}
	out := make([]string, 0, len(lines)+len(block)*o.times)
	out = append(out, lines[:destAfter]...)
	for i := 0; i < o.times; i++ {
		out = append(out, block...)
	}
	out = append(out, lines[destAfter:]...)
	return out, desc, nil
}

// parseDestList splits and dedupes a comma-separated -dest value, trimming
// whitespace and dropping empties and repeats. Shared by runMultiDest and
// runMultiDestReplace.
func parseDestList(dests string) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range strings.Split(dests, ",") {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		out = append(out, d)
	}
	return out
}

// runMultiDest copies or moves a block from srcFile into every file in the
// comma-separated dests list. Policy: report and continue, never abort. For
// move, destination writes happen first and the source text is removed only
// if at least one destination write succeeded, so text is never lost.
// Returns the process exit code: 0 if at least one destination succeeded.
func runMultiDest(o multiDestOpts) int {
	out, err := collectMultiDest(o)
	return printMultiDest(o.op, out, err)
}

// runMultiDestReplace overwrites -destblock in every file in the
// comma-separated dests list with the block resolved from srcFile (via
// -block, or -line/-match like the other ops). Unlike copy/move, a missing
// target block is always a SKIP with a report line -- there is no sensible
// "insert instead" fallback for "overwrite this thing that isn't there".
// Policy: report and continue, never abort. Returns the process exit code:
// 0 if at least one destination succeeded.
func runMultiDestReplace(o multiDestOpts) int {
	out, err := collectMultiDestReplace(o)
	return printMultiDest("replace", out, err)
}

// multiDestOutcome is what a -dest run produced. Nothing has been printed:
// the CLI prints the results and notes to stderr, the MCP server formats them
// into its tool result. failed is true when the run should exit 1.
type multiDestOutcome struct {
	results []batchResult
	okCount int
	notes   []string
	failed  bool
}

// printMultiDest prints an outcome the way the CLI always has and returns the
// process exit code. A non-nil err is a request-level problem (nothing ran).
func printMultiDest(label string, out multiDestOutcome, err error) int {
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	printBatchReport(label, out.results)
	for _, n := range out.notes {
		fmt.Fprintln(os.Stderr, n)
	}
	if out.failed {
		return 1
	}
	return 0
}

// collectMultiDest does the work of runMultiDest and prints nothing. The only
// error is a request-level problem; per-destination problems are SKIP results.
func collectMultiDest(o multiDestOpts) (multiDestOutcome, error) {
	var out multiDestOutcome
	if o.op != "copy" && o.op != "move" {
		return out, fmt.Errorf("-dest supports only copy and move, not %s", o.op)
	}
	if o.times < 1 {
		return out, fmt.Errorf("-times must be >= 1")
	}
	dests := parseDestList(o.dests)
	if len(dests) == 0 {
		return out, fmt.Errorf("-dest needs at least one file")
	}

	srcLines, err := readLines(o.srcFile)
	if err != nil {
		return out, err
	}
	var srcStart, srcEnd int
	if o.block != "" {
		srcStart, srcEnd, err = resolveBlock(srcLines, o.lang, o.block)
	} else {
		srcStart, srcEnd, err = resolveSourceLines(srcLines, o.line, o.end, o.match, o.endmatch, o.nth)
	}
	if err != nil {
		return out, err
	}
	if srcStart < 1 || srcEnd > len(srcLines) || srcStart > srcEnd {
		return out, fmt.Errorf("source range %d-%d invalid (file has %d lines)", srcStart, srcEnd, len(srcLines))
	}
	block := make([]string, srcEnd-srcStart+1)
	copy(block, srcLines[srcStart-1:srcEnd])

	srcInfo, srcStatErr := os.Stat(o.srcFile)

	for _, d := range dests {
		if srcStatErr == nil {
			if di, e := os.Stat(d); e == nil && os.SameFile(srcInfo, di) {
				out.results = append(out.results, batchResult{d, "SKIP", "destination is the source file (use plain copy/move)"})
				continue
			}
		}
		dl, e := readLines(d)
		if e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		hint := goImportHint(o.srcFile, d, block, dl)
		res, desc, e := multiDestOne(dl, o, block)
		if e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		if e := writeLines(d, res); e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		out.okCount++
		out.results = append(out.results, batchResult{d, "OK", fmt.Sprintf("%d line(s) x%d %s", len(block), o.times, desc)})
		if hint != "" {
			out.notes = append(out.notes, hint)
		}
	}

	if out.okCount == 0 {
		if o.op == "move" {
			out.notes = append(out.notes, fmt.Sprintf("move: every destination failed, %s left unmodified", o.srcFile))
		}
		out.failed = true
		return out, nil
	}
	if o.op == "move" {
		rest := make([]string, 0, len(srcLines)-len(block))
		rest = append(rest, srcLines[:srcStart-1]...)
		rest = append(rest, srcLines[srcEnd:]...)
		if err := writeLines(o.srcFile, rest); err != nil {
			out.notes = append(out.notes, fmt.Sprintf("Error: copies landed but source could not be trimmed: %v", err))
			out.failed = true
			return out, nil
		}
		out.notes = append(out.notes, fmt.Sprintf("move: removed lines %d-%d from %s", srcStart, srcEnd, o.srcFile))
	}
	return out, nil
}

// collectMultiDestReplace does the work of runMultiDestReplace and prints
// nothing. A destination without the target block is always a SKIP.
func collectMultiDestReplace(o multiDestOpts) (multiDestOutcome, error) {
	var out multiDestOutcome
	if o.destBlock == "" {
		return out, fmt.Errorf("replace -dest requires -destblock")
	}
	dests := parseDestList(o.dests)
	if len(dests) == 0 {
		return out, fmt.Errorf("-dest needs at least one file")
	}

	srcLines, err := readLines(o.srcFile)
	if err != nil {
		return out, err
	}
	var srcStart, srcEnd int
	if o.block != "" {
		srcStart, srcEnd, err = resolveBlock(srcLines, o.lang, o.block)
	} else {
		srcStart, srcEnd, err = resolveSourceLines(srcLines, o.line, o.end, o.match, o.endmatch, o.nth)
	}
	if err != nil {
		return out, err
	}
	if srcStart < 1 || srcEnd > len(srcLines) || srcStart > srcEnd {
		return out, fmt.Errorf("source range %d-%d invalid (file has %d lines)", srcStart, srcEnd, len(srcLines))
	}
	srcBlock := make([]string, srcEnd-srcStart+1)
	copy(srcBlock, srcLines[srcStart-1:srcEnd])

	srcInfo, srcStatErr := os.Stat(o.srcFile)

	for _, d := range dests {
		if srcStatErr == nil {
			if di, e := os.Stat(d); e == nil && os.SameFile(srcInfo, di) {
				out.results = append(out.results, batchResult{d, "SKIP", "destination is the source file (use plain replace)"})
				continue
			}
		}
		dl, e := readLines(d)
		if e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		hint := goImportHint(o.srcFile, d, srcBlock, dl)
		bs, be, e := resolveBlock(dl, o.lang, o.destBlock)
		if e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", fmt.Sprintf("target block %q not found: %v", o.destBlock, e)})
			continue
		}
		res, e := execReplace(dl, bs, be, srcBlock)
		if e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		if e := writeLines(d, res); e != nil {
			out.results = append(out.results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		out.okCount++
		out.results = append(out.results, batchResult{d, "OK", fmt.Sprintf("block %q replaced with %d line(s)", o.destBlock, len(srcBlock))})
		if hint != "" {
			out.notes = append(out.notes, hint)
		}
	}

	if out.okCount == 0 {
		out.failed = true
	}
	return out, nil
}
