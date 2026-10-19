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
	msg := fmt.Sprintf("Replaced '%s' on %d line(s) (%d total)\n", search, len(res.Changed), len(res.Lines))
	return finishVerified(res, path, msg, verbose, out, write, func(p string) (replaceVerify, error) {
		return verifyReplaceAllOnDisk(p, search, res)
	})
}
