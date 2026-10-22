package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"strings"
)

// Result fields for the end record of a logged run. The operations report what
// they did through logNote and its helpers; logFinish hands the collected
// fields to the end record. Nothing is kept, and nothing is hashed, when the
// run is not logged, so a run with logging off pays nothing.

type logNoteItem struct{ key, val string }

// logNotes holds the result fields of the running operation, in the order
// they were first noted.
var logNotes []logNoteItem

// logNote records one result field for the end record. A later value for the
// same key replaces the earlier one. It does nothing when the run is not logged.
func logNote(key, val string) {
	if logRun == nil {
		return
	}
	for i := range logNotes {
		if logNotes[i].key == key {
			logNotes[i].val = val
			return
		}
	}
	logNotes = append(logNotes, logNoteItem{key, val})
}

// logNoteValue returns a noted value; it exists for tests.
func logNoteValue(key string) (string, bool) {
	for _, n := range logNotes {
		if n.key == key {
			return n.val, true
		}
	}
	return "", false
}

// logTakeNotes returns the noted fields for the end record and clears them.
func logTakeNotes() []logKV {
	if len(logNotes) == 0 {
		return nil
	}
	out := make([]logKV, 0, len(logNotes))
	for _, n := range logNotes {
		out = append(out, logKV{n.key, n.val})
	}
	logNotes = nil
	return out
}

// logFileSHA256 returns the SHA-256 of a file as lower-case hex, or "" when the
// run is not logged or the file cannot be read (a file that does not exist yet
// has no hash before the edit).
func logFileSHA256(path string) string {
	if logRun == nil {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// logNoteHashes records the hashes of a file before and after the edit; an
// empty hash is not recorded.
func logNoteHashes(before, after string) {
	if before != "" {
		logNote("sha256_before", before)
	}
	if after != "" {
		logNote("sha256_after", after)
	}
}

// logNoteReplace records how many occurrences were replaced and on how many lines.
func logNoteReplace(occurrences, changedLines int) {
	logNote("replacements", strconv.Itoa(occurrences))
	logNote("lines_changed", strconv.Itoa(changedLines))
}

// logNoteVerify records the verification verdict: PASS, FAIL or none.
func logNoteVerify(v string) { logNote("verify", v) }

// logNoteBatch records the per-file results of a multi-file run: the totals,
// the verify verdict and one "STATUS path" entry per file. A file that is FAIL
// or ERROR counts as failed; verify is FAIL when any file failed verification,
// PASS when the run verified its files and at least one succeeded, else none.
func logNoteBatch(results []batchResult, verified bool) {
	if logRun == nil {
		return
	}
	ok, skip, fail, vfail := 0, 0, 0, 0
	parts := make([]string, 0, len(results))
	for _, r := range results {
		switch r.status {
		case "OK":
			ok++
		case "SKIP":
			skip++
		case "FAIL":
			fail++
			vfail++
		default:
			fail++
		}
		parts = append(parts, r.status+" "+r.path)
	}
	verdict := "none"
	switch {
	case !verified:
	case vfail > 0:
		verdict = "FAIL"
	case ok > 0:
		verdict = "PASS"
	}
	logNote("files_total", strconv.Itoa(len(results)))
	logNote("files_ok", strconv.Itoa(ok))
	logNote("files_skip", strconv.Itoa(skip))
	logNote("files_fail", strconv.Itoa(fail))
	logNote("verify", verdict)
	logNote("results", strings.Join(parts, "; "))
}
