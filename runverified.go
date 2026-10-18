package main

import (
	"fmt"
	"io"
)

// runReplaceAllVerified performs a literal replaceall and verifies the written
// file. It returns the process exit code: 0 = replaced and verified, 1 = the
// edit did not happen (no match, or the write failed), 3 = the file was
// written but verification failed or could not be done. The writer and the
// output stream are parameters so every outcome can be tested.
func runReplaceAllVerified(lines []string, path, search, replacement string, verbose bool, out io.Writer, write func(string, []string) error) int {
	res, err := execReplaceAllDetail(lines, search, replacement)
	if err != nil {
		fmt.Fprintln(out, err)
		return 1
	}
	if err := write(path, res.Lines); err != nil {
		fmt.Fprintf(out, "Error writing file: %v\n", err)
		return 1
	}
	fmt.Fprintf(out, "Replaced '%s' on %d line(s) (%d total)\n", search, len(res.Changed), len(res.Lines))
	v, err := verifyReplaceAllOnDisk(path, search, res)
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
