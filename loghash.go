package main

import "strings"

// Hashes of the one file a logged write operation changes. The before hash is
// taken right after the begin record, the after hash when the run finishes, so
// every single-file write operation (insert, delete, replace, move, copy,
// write and the like) reports sha256_before and sha256_after without a hook
// inside each operation. A run over several files (-file with a list or a
// glob, -files, -dest) is not hashed here: it reports per-file results instead.

var (
	logHashPath   string
	logHashBefore string
)

// logHashTarget returns the absolute path of the single file op changes, or ""
// when op reads only, is a guided parent, or the run covers several files.
func logHashTarget(op string, in logCLIInput) string {
	if !logOpIsWrite(op) || op == "guided" || in.Dest != "" || strings.TrimSpace(in.Files) != "" {
		return ""
	}
	spec := strings.TrimSpace(in.File)
	if spec == "" || strings.ContainsAny(spec, ",*?[") {
		return ""
	}
	return logAbsList(spec, in.Cwd)
}

// logHashStart remembers the target file and its hash. It does nothing when
// the run is not logged.
func logHashStart(op string, in logCLIInput) {
	logHashPath, logHashBefore = "", ""
	if logRun == nil {
		return
	}
	if p := logHashTarget(op, in); p != "" {
		logHashPath, logHashBefore = p, logFileSHA256(p)
	}
}

// logHashFinish notes the hashes before the end record is written. A hash an
// operation already noted (the verified replaceall runners do) is kept.
func logHashFinish() {
	p := logHashPath
	logHashPath = ""
	if p == "" || logRun == nil {
		return
	}
	if _, ok := logNoteValue("sha256_before"); !ok {
		logNoteHashes(logHashBefore, "")
	}
	if _, ok := logNoteValue("sha256_after"); !ok {
		logNoteHashes("", logFileSHA256(p))
	}
}
