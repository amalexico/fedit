package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

// execReplaceAllRegexDetail is the pure core of the verified regex replaceall.
// Capture groups work as in regexp ($1, ${name}). A line counts as changed only
// if its text differs afterwards, so a replacement identical to the match
// reports "No lines match regex", as the unverified path always did.
func execReplaceAllRegexDetail(lines []string, pattern, replacement string) (replaceAllResult, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return replaceAllResult{}, fmt.Errorf("invalid regex %q: %v", pattern, err)
	}
	res := replaceAllResult{Lines: make([]string, len(lines))}
	for i, line := range lines {
		newLine := re.ReplaceAllString(line, replacement)
		res.Lines[i] = newLine
		if newLine != line {
			res.Changed = append(res.Changed, i+1)
			res.Occurrences += len(re.FindAllStringIndex(line, -1))
		}
	}
	if len(res.Changed) == 0 {
		return replaceAllResult{}, fmt.Errorf("No lines match regex: %s", pattern)
	}
	return flattenResult(res), nil
}

// verifyOnDiskWith re-reads path and compares it line by line with res.Lines.
// count reports how many matches of the searched text a line holds, so the same
// check serves literal and regex replaceall.
func verifyOnDiskWith(path string, res replaceAllResult, count func(string) int) (replaceVerify, error) {
	got, err := readLines(path)
	if err != nil {
		return replaceVerify{}, err
	}
	v := replaceVerify{WantLines: len(res.Lines), GotLines: len(got)}
	n := len(res.Lines)
	if len(got) > n {
		n = len(got)
	}
	for i := 0; i < n; i++ {
		if i >= len(res.Lines) || i >= len(got) || res.Lines[i] != got[i] {
			v.Bad = append(v.Bad, i+1)
		}
	}
	for _, l := range got {
		v.Remaining += count(l)
	}
	for _, l := range res.Lines {
		v.WantRemaining += count(l)
	}
	v.OK = len(v.Bad) == 0 && v.WantLines == v.GotLines && v.Remaining == v.WantRemaining
	return v, nil
}

// verifyReplaceAllRegexOnDisk is the regex counterpart of verifyReplaceAllOnDisk.
// Remaining counts matches of the pattern still on disk; WantRemaining is what a
// correct result holds, which is non-zero when the replacement recreates the pattern.
func verifyReplaceAllRegexOnDisk(path, pattern string, res replaceAllResult) (replaceVerify, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return replaceVerify{}, err
	}
	return verifyOnDiskWith(path, res, func(s string) int {
		return len(re.FindAllStringIndex(s, -1))
	})
}

// finishVerified is the shared tail of the verified replaceall runners: write,
// announce, re-read and verify, print the verdict, and under verbose a window
// for every changed line. It returns the exit code: 0 verified, 1 write failed,
// 3 written but not verified.
func finishVerified(res replaceAllResult, path, doneMsg string, verbose bool, out io.Writer, write func(string, []string) error, verify func(string) (replaceVerify, error)) int {
	if err := write(path, res.Lines); err != nil {
		fmt.Fprintf(out, "Error writing file: %v\n", err)
		return 1
	}
	fmt.Fprint(out, doneMsg)
	v, err := verify(path)
	if err != nil {
		fmt.Fprintf(out, "VERIFY: could not re-read %s: %v\n", path, err)
		return 3
	}
	fmt.Fprint(out, formatReplaceVerify(res, v))
	if verbose {
		if disk, err := readLines(path); err == nil {
			fmt.Fprint(out, formatHitWindows(disk, res.Changed, 2, 2))
		}
	}
	if !v.OK {
		return 3
	}
	return 0
}

// runReplaceAllRegexVerified performs a regex replaceall and verifies the written
// file. Exit codes as in runReplaceAllVerified.
func runReplaceAllRegexVerified(lines []string, path, pattern, replacement string, verbose bool, out io.Writer, write func(string, []string) error) int {
	res, err := execReplaceAllRegexDetail(lines, pattern, replacement)
	if err != nil {
		if strings.HasPrefix(err.Error(), "invalid regex") {
			fmt.Fprintf(out, "Error: %v\n", err)
		} else {
			fmt.Fprintln(out, err)
		}
		return 1
	}
	msg := fmt.Sprintf("Regex-replaced on %d line(s) (%d total)\n", len(res.Changed), len(res.Lines))
	return finishVerified(res, path, msg, verbose, out, write, func(p string) (replaceVerify, error) {
		return verifyReplaceAllRegexOnDisk(p, pattern, res)
	})
}
