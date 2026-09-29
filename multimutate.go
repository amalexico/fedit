package main

import (
	"fmt"
	"os"
	"strings"
)

// multiMutOpts holds the options for a self-contained mutating op run over
// several files (multi -file). Every file is its own source and target.
type multiMutOpts struct {
	spec        string
	op          string
	match       string
	endmatch    string
	nth         int
	newLines    []string
	replacement string
	matchRegex  string
}

// multiMutateOne applies the op to one file's lines using the pure exec*
// functions. It returns the new lines and a short detail for the report.
func multiMutateOne(o multiMutOpts, lines []string) ([]string, string, error) {
	switch o.op {
	case "replaceall":
		if o.matchRegex != "" {
			result, count, err := execReplaceAllRegex(lines, o.matchRegex, o.replacement)
			if err != nil {
				return nil, "", err
			}
			return result, fmt.Sprintf("replaced on %d line(s) (regex)", count), nil
		}
		result, count, err := execReplaceAll(lines, o.match, o.replacement)
		if err != nil {
			return nil, "", err
		}
		return result, fmt.Sprintf("replaced on %d line(s)", count), nil
	case "insertafter", "insertbefore":
		result, target, _, err := execInsertMatch(lines, o.match, o.nth, o.newLines, o.op == "insertbefore")
		if err != nil {
			return nil, "", err
		}
		return result, fmt.Sprintf("inserted %d line(s) at match on line %d", len(o.newLines), target), nil
	case "delete":
		var start, end int
		if o.endmatch == "" {
			hits := findMatches(lines, o.match)
			target, err := resolveNth(hits, o.nth)
			if err != nil {
				return nil, "", err
			}
			start, end = target, target
		} else {
			s, e, err := resolveSourceLines(lines, 0, 0, o.match, o.endmatch, o.nth)
			if err != nil {
				return nil, "", err
			}
			start, end = s, e
		}
		result, err := execDelete(lines, start, end)
		if err != nil {
			return nil, "", err
		}
		return result, fmt.Sprintf("deleted lines %d-%d", start, end), nil
	case "replace":
		var start, end int
		if o.endmatch == "" {
			hits := findMatches(lines, o.match)
			target, err := resolveNth(hits, o.nth)
			if err != nil {
				return nil, "", err
			}
			start, end = target, target
		} else {
			s, e, err := resolveSourceLines(lines, 0, 0, o.match, o.endmatch, o.nth)
			if err != nil {
				return nil, "", err
			}
			start, end = s, e
		}
		result, err := execReplace(lines, start, end, o.newLines)
		if err != nil {
			return nil, "", err
		}
		return result, fmt.Sprintf("replaced lines %d-%d", start, end), nil
	case "write":
		return o.newLines, fmt.Sprintf("wrote %d line(s)", len(o.newLines)), nil
	}
	return nil, "", fmt.Errorf("unsupported op %s", o.op)
}

// runMultiMutate runs a self-contained mutating op (replaceall, insertafter,
// insertbefore, delete) over every file in o.spec. A problem in one file is
// reported as SKIP and never aborts the batch. Returns the process exit code:
// 1 if the request is invalid or no file succeeded.
func runMultiMutate(o multiMutOpts) int {
	switch o.op {
	case "replaceall", "insertafter", "insertbefore", "delete", "replace", "write":
	default:
		fmt.Fprintf(os.Stderr, "Error: multi -file mutation supports replaceall, insertafter, insertbefore, delete, replace and write (got -op %s)\n", o.op)
		return 1
	}
	if o.op != "write" && o.match == "" && o.matchRegex == "" {
		fmt.Fprintf(os.Stderr, "Error: -match or -match-regex is required for %s\n", o.op)
		return 1
	}
	if o.matchRegex != "" && o.op != "replaceall" {
		fmt.Fprintln(os.Stderr, "Error: -match-regex is only supported with replaceall")
		return 1
	}
	if (o.op == "insertafter" || o.op == "insertbefore" || o.op == "replace" || o.op == "write") && len(o.newLines) == 0 {
		fmt.Fprintln(os.Stderr, "Nothing to insert (-text or -textfile is empty)")
		return 1
	}
	paths, err := expandMultiSpec(o.spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	results := make([]batchResult, 0, len(paths))
	okCount := 0
	for _, p := range paths {
		lines, rerr := readLines(p)
		if rerr != nil {
			results = append(results, batchResult{path: p, status: "SKIP", detail: rerr.Error()})
			continue
		}
		result, detail, merr := multiMutateOne(o, lines)
		if merr != nil {
			flat := strings.ReplaceAll(merr.Error(), "\n", "; ")
			results = append(results, batchResult{path: p, status: "SKIP", detail: flat})
			continue
		}
		if werr := writeLines(p, result); werr != nil {
			results = append(results, batchResult{path: p, status: "SKIP", detail: "write failed: " + werr.Error()})
			continue
		}
		okCount++
		results = append(results, batchResult{path: p, status: "OK", detail: detail})
	}
	printBatchReport("multi -file "+o.op, results)
	if okCount == 0 {
		return 1
	}
	return 0
}
