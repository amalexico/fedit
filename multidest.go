package main

import (
	"fmt"
	"os"
	"strings"
)

// multiDestOpts holds the flags for cross-file copy and move (-dest fan-out).
type multiDestOpts struct {
	op          string
	srcFile     string
	dests       string
	lang        string
	block       string
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

// runMultiDest copies or moves a block from srcFile into every file in the
// comma-separated dests list. Policy: report and continue, never abort. For
// move, destination writes happen first and the source text is removed only
// if at least one destination write succeeded, so text is never lost.
// Returns the process exit code: 0 if at least one destination succeeded.
func runMultiDest(o multiDestOpts) int {
	if o.op != "copy" && o.op != "move" {
		fmt.Fprintf(os.Stderr, "Error: -dest supports only copy and move, not %s\n", o.op)
		return 1
	}
	if o.times < 1 {
		fmt.Fprintln(os.Stderr, "Error: -times must be >= 1")
		return 1
	}
	var dests []string
	seen := map[string]bool{}
	for _, d := range strings.Split(o.dests, ",") {
		d = strings.TrimSpace(d)
		if d == "" || seen[d] {
			continue
		}
		seen[d] = true
		dests = append(dests, d)
	}
	if len(dests) == 0 {
		fmt.Fprintln(os.Stderr, "Error: -dest needs at least one file")
		return 1
	}

	srcLines, err := readLines(o.srcFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	var srcStart, srcEnd int
	if o.block != "" {
		srcStart, srcEnd, err = resolveBlock(srcLines, o.lang, o.block)
	} else {
		srcStart, srcEnd, err = resolveSourceLines(srcLines, o.line, o.end, o.match, o.endmatch, o.nth)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	if srcStart < 1 || srcEnd > len(srcLines) || srcStart > srcEnd {
		fmt.Fprintf(os.Stderr, "Error: source range %d-%d invalid (file has %d lines)\n", srcStart, srcEnd, len(srcLines))
		return 1
	}
	block := make([]string, srcEnd-srcStart+1)
	copy(block, srcLines[srcStart-1:srcEnd])

	srcInfo, srcStatErr := os.Stat(o.srcFile)

	var results []batchResult
	okCount := 0
	for _, d := range dests {
		if srcStatErr == nil {
			if di, e := os.Stat(d); e == nil && os.SameFile(srcInfo, di) {
				results = append(results, batchResult{d, "SKIP", "destination is the source file (use plain copy/move)"})
				continue
			}
		}
		dl, e := readLines(d)
		if e != nil {
			results = append(results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		out, desc, e := multiDestOne(dl, o, block)
		if e != nil {
			results = append(results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		if e := writeLines(d, out); e != nil {
			results = append(results, batchResult{d, "SKIP", e.Error()})
			continue
		}
		okCount++
		results = append(results, batchResult{d, "OK", fmt.Sprintf("%d line(s) x%d %s", len(block), o.times, desc)})
	}
	printBatchReport(o.op, results)

	if okCount == 0 {
		if o.op == "move" {
			fmt.Fprintf(os.Stderr, "move: every destination failed, %s left unmodified\n", o.srcFile)
		}
		return 1
	}
	if o.op == "move" {
		rest := make([]string, 0, len(srcLines)-len(block))
		rest = append(rest, srcLines[:srcStart-1]...)
		rest = append(rest, srcLines[srcEnd:]...)
		if err := writeLines(o.srcFile, rest); err != nil {
			fmt.Fprintf(os.Stderr, "Error: copies landed but source could not be trimmed: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "move: removed lines %d-%d from %s\n", srcStart, srcEnd, o.srcFile)
	}
	return 0
}
