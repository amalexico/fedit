package main

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// streamScan is what the verification pass learns about the rewritten file
// without holding it in memory: the SHA-256 of its bytes, its line count and
// how many matches of the old text it still contains.
type streamScan struct {
	sum       [32]byte
	lines     int
	remaining int
}

// scanStreamFile reads path once with the same scanner settings the stream
// engine uses. Every byte read goes through the hasher, so the sum covers the
// whole file whatever the line endings are.
func scanStreamFile(path string, remaining func(string) int) (streamScan, error) {
	f, err := os.Open(path)
	if err != nil {
		return streamScan{}, err
	}
	defer f.Close()
	h := sha256.New()
	sc := bufio.NewScanner(io.TeeReader(f, h))
	sc.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	var s streamScan
	for sc.Scan() {
		s.lines++
		s.remaining += remaining(sc.Text())
	}
	if err := sc.Err(); err != nil {
		return streamScan{}, err
	}
	copy(s.sum[:], h.Sum(nil))
	return s, nil
}

// runStreamVerified is the shared core of the verified -stream replaceall.
// xform returns the new line and the number of occurrences replaced (0 = the
// line is unchanged); remaining counts matches of the old text in one line.
// While execStreamOp rewrites the file, the callback hashes exactly the bytes
// being written and records the output line number of every changed line. A
// second streaming pass then reads the file back and compares hash, line
// count and remaining matches. Exit 0 = replaced and verified, 1 = the edit
// did not happen, 3 = written but verification failed or could not be done.
func runStreamVerified(path string, xform func(string) (string, int), remaining func(string) int, noMatchMsg, doneFmt string, out io.Writer) int {
	hasher := sha256.New()
	res := replaceAllResult{}
	changedIn, wantLines, wantRemaining := 0, 0, 0
	linesRead, _, err := execStreamOp(path, func(_ int, line string) []string {
		newLine, n := xform(line)
		pieces := strings.Split(newLine, "\n")
		start := wantLines + 1
		wantLines += len(pieces)
		if n > 0 {
			changedIn++
			res.Occurrences += n
			for i := range pieces {
				res.Changed = append(res.Changed, start+i)
			}
		}
		for _, p := range pieces {
			wantRemaining += remaining(p)
		}
		hasher.Write([]byte(newLine))
		hasher.Write([]byte{'\n'})
		return []string{newLine}
	})
	if err != nil {
		fmt.Fprintf(out, "Error: %v\n", err)
		return 1
	}
	if changedIn == 0 {
		fmt.Fprintln(out, noMatchMsg)
		return 1
	}
	fmt.Fprintf(out, doneFmt, changedIn, linesRead)
	got, err := scanStreamFile(path, remaining)
	if err != nil {
		fmt.Fprintf(out, "VERIFY: could not re-read %s: %v\n", path, err)
		return 3
	}
	var want [32]byte
	copy(want[:], hasher.Sum(nil))
	hashOK := got.sum == want
	v := replaceVerify{WantLines: wantLines, GotLines: got.lines, Remaining: got.remaining, WantRemaining: wantRemaining}
	v.OK = hashOK && v.WantLines == v.GotLines && v.Remaining == v.WantRemaining
	if v.OK {
		fmt.Fprint(out, formatReplaceVerify(res, v))
		return 0
	}
	fmt.Fprintf(out, "VERIFY: FAIL (content hash match %v; old text remaining %d, expected %d; lines on disk %d, expected %d)\n",
		hashOK, v.Remaining, v.WantRemaining, v.GotLines, v.WantLines)
	fmt.Fprintf(out, "  changed lines: %s\n", formatLineRanges(res.Changed))
	return 3
}

// runStreamLiteralVerified is the verified streaming literal replaceall.
func runStreamLiteralVerified(path, search, replacement string, out io.Writer) int {
	xform := func(line string) (string, int) {
		c := strings.Count(line, search)
		if c == 0 {
			return line, 0
		}
		return strings.ReplaceAll(line, search, replacement), c
	}
	remaining := func(l string) int {
		if search == "" {
			return 0
		}
		return strings.Count(l, search)
	}
	return runStreamVerified(path, xform, remaining, "No lines match: "+search, "Stream-replaced on %d line(s) of %d total\n", out)
}

// runStreamRegexVerified is the verified streaming regex replaceall.
func runStreamRegexVerified(path, pattern, replacement string, out io.Writer) int {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(out, "Invalid regex %q: %v\n", pattern, err)
		return 1
	}
	xform := func(line string) (string, int) {
		nl := re.ReplaceAllString(line, replacement)
		if nl == line {
			return line, 0
		}
		return nl, len(re.FindAllStringIndex(line, -1))
	}
	remaining := func(l string) int {
		return len(re.FindAllStringIndex(l, -1))
	}
	return runStreamVerified(path, xform, remaining, "No lines match regex: "+pattern, "Stream regex-replaced on %d line(s) of %d total\n", out)
}
