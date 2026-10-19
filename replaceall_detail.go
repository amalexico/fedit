package main

import (
	"fmt"
	"strconv"
	"strings"
)

// replaceAllResult is the full outcome of a literal replaceall on in-memory
// lines: the new lines, the 1-based numbers of the lines that changed, and the
// total number of occurrences replaced.
type replaceAllResult struct {
	Lines       []string
	Changed     []int
	Occurrences int
}

// execReplaceAllDetail is the pure core behind the verified replaceall. It never
// mutates its input, never touches a file and never calls os.Exit.
func execReplaceAllDetail(lines []string, search, replacement string) (replaceAllResult, error) {
	if search == "" {
		return replaceAllResult{}, fmt.Errorf("replaceall requires -match (text to find)")
	}
	res := replaceAllResult{Lines: make([]string, len(lines))}
	for i, line := range lines {
		n := strings.Count(line, search)
		if n > 0 {
			res.Lines[i] = strings.ReplaceAll(line, search, replacement)
			res.Changed = append(res.Changed, i+1)
			res.Occurrences += n
		} else {
			res.Lines[i] = line
		}
	}
	if len(res.Changed) == 0 {
		return replaceAllResult{}, fmt.Errorf("No lines contain: %s", search)
	}
	return flattenResult(res), nil
}

// replaceVerify is the verdict of re-reading the written file from disk and
// comparing it with the expected result of the replaceall.
type replaceVerify struct {
	OK            bool
	WantLines     int
	GotLines      int
	Bad           []int
	Remaining     int
	WantRemaining int
}

// verifyReplaceAllOnDisk re-reads path and compares it line by line with
// res.Lines. Bad lists every 1-based line that differs, including lines missing
// from or extra on disk. Remaining counts the old text still on disk;
// WantRemaining is what a correct result contains (non-zero when the
// replacement itself contains the match).
func verifyReplaceAllOnDisk(path, search string, res replaceAllResult) (replaceVerify, error) {
	return verifyOnDiskWith(path, res, func(l string) int {
		if search == "" {
			return 0
		}
		return strings.Count(l, search)
	})
}

// formatLineRanges collapses ascending line numbers into ranges: 1,2,3,5 -> "1-3,5".
func formatLineRanges(nums []int) string {
	var parts []string
	for i := 0; i < len(nums); {
		j := i
		for j+1 < len(nums) && nums[j+1] == nums[j]+1 {
			j++
		}
		if j > i {
			parts = append(parts, strconv.Itoa(nums[i])+"-"+strconv.Itoa(nums[j]))
		} else {
			parts = append(parts, strconv.Itoa(nums[i]))
		}
		i = j + 1
	}
	return strings.Join(parts, ",")
}

// formatReplaceVerify renders the PASS/FAIL verdict and the complete list of
// changed lines. The list is never capped.
func formatReplaceVerify(res replaceAllResult, v replaceVerify) string {
	var b strings.Builder
	if v.OK {
		fmt.Fprintf(&b, "VERIFY: PASS (%d occurrence(s) on %d line(s); old text remaining %d, expected %d)\n",
			res.Occurrences, len(res.Changed), v.Remaining, v.WantRemaining)
	} else {
		fmt.Fprintf(&b, "VERIFY: FAIL (old text remaining %d, expected %d; lines on disk %d, expected %d)\n",
			v.Remaining, v.WantRemaining, v.GotLines, v.WantLines)
		fmt.Fprintf(&b, "  mismatched lines: %s\n", formatLineRanges(v.Bad))
	}
	fmt.Fprintf(&b, "  changed lines: %s\n", formatLineRanges(res.Changed))
	return b.String()
}
