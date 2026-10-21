package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// State of -outputresult. While the capture is active, os.Stdout and os.Stderr
// both point at one temp file, so everything the operation prints is kept in
// order. exitNow replays or saves it when the process ends.
var (
	outResPath   string
	outResAppend bool
	outResTmp    *os.File
	outResOut    *os.File
	outResErr    *os.File
	outResDone   bool
)

func outResSame(a, b string) bool {
	aa, err1 := filepath.Abs(a)
	bb, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(aa, bb)
	}
	return aa == bb
}

// outputResultRefuse returns an error message when path is one of the files
// the run would edit (-file, -files or -dest, including glob matches).
func outputResultRefuse(path, files, filesGlob, dest string) string {
	var cand []string
	for _, spec := range []string{files, filesGlob, dest} {
		for _, p := range strings.Split(spec, ",") {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			cand = append(cand, p)
			if m, err := filepath.Glob(p); err == nil {
				cand = append(cand, m...)
			}
		}
	}
	for _, c := range cand {
		if outResSame(path, c) {
			return fmt.Sprintf("-outputresult: %s is also an edited file (%s); choose a different result file", path, c)
		}
	}
	return ""
}

// startOutputResult checks that the result file can be created, then starts
// capturing stdout and stderr. It fails before anything is edited.
func startOutputResult(path string, appendMode bool) error {
	dir := filepath.Dir(path)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return fmt.Errorf("directory %s does not exist", dir)
	}
	tmp, err := os.CreateTemp("", "fedit_out_*.tmp")
	if err != nil {
		return err
	}
	outResPath, outResAppend, outResTmp = path, appendMode, tmp
	outResOut, outResErr = os.Stdout, os.Stderr
	os.Stdout, os.Stderr = tmp, tmp
	return nil
}

// finishOutputResult ends the capture and returns the final exit code. On
// success the captured report goes to the result file and one summary line to
// the terminal; on a non-zero code everything captured goes to stderr and the
// result file is left alone. It does nothing when no capture is active.
func finishOutputResult(code int) int {
	if outResTmp == nil || outResDone {
		return code
	}
	outResDone = true
	os.Stdout, os.Stderr = outResOut, outResErr
	name := outResTmp.Name()
	outResTmp.Close()
	data, rerr := os.ReadFile(name)
	os.Remove(name)
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "-outputresult: cannot read the captured output: %v\n", rerr)
		if code == 0 {
			return 3
		}
		return code
	}
	if code != 0 {
		os.Stderr.Write(data)
		return code
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	flags := os.O_CREATE | os.O_WRONLY
	if outResAppend {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(outResPath, flags, 0o644)
	if err == nil {
		_, err = f.Write(data)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "-outputresult: the operation succeeded but the result could not be written to %s: %v\n", outResPath, err)
		os.Stderr.Write(data)
		return 3
	}
	verb := "written to"
	if outResAppend {
		verb = "appended to"
	}
	fmt.Fprintf(os.Stdout, "result %s %s (%d lines)\n", verb, outResPath, bytes.Count(data, []byte("\n")))
	return 0
}

// exitNow is the single exit point: it finishes any -outputresult capture and
// then exits with the resulting code.
func exitNow(code int) {
	os.Exit(logFinish(finishOutputResult(code)))
}
