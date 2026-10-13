package main

import (
	"fmt"
	"strings"
)

// goImportHint returns an advisory note when lines transferred between two Go
// files use standard library packages the destination does not import. It is
// a hint only and never affects the transfer: an empty string means no hint.
func goImportHint(srcFile, destFile string, transferred, destLines []string) string {
	if !isGoSource(srcFile) || !isGoSource(destFile) {
		return ""
	}
	missing := checkMissingGoImports(transferred, destLines)
	if len(missing) == 0 {
		return ""
	}
	return fmt.Sprintf("hint: %s may need imports: %s (advisory only; the transfer was done)", destFile, strings.Join(missing, ", "))
}

func isGoSource(path string) bool {
	return strings.HasSuffix(strings.ToLower(path), ".go")
}
