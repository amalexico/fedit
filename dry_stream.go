package main

import (
	"bufio"
	"fmt"
	"os"
)

// dryStreamPass is execStreamOp under -dry. It reads the file and runs fn for
// every line, so the callback sees (and counts) exactly what a real run would,
// and it writes nothing: no temp file and no rename. It returns (linesRead,
// linesWritten, error) like execStreamOp.
func dryStreamPass(path string, fn func(lineNum int, line string) []string) (int, int, error) {
	src, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open: %v", err)
	}
	defer src.Close()
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	linesRead, linesWritten := 0, 0
	for scanner.Scan() {
		linesRead++
		linesWritten += len(fn(linesRead, scanner.Text()))
	}
	if err := scanner.Err(); err != nil {
		return linesRead, 0, fmt.Errorf("scan: %v", err)
	}
	return linesRead, linesWritten, nil
}
