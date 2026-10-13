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

// sourceBlockHint returns the Go import advisory for an insert whose content was
// pulled from another file with sourcefile and sourceblock. It returns an empty
// string when no source block was used, so callers can call it unconditionally.
// destLines must be the destination as it was before the edit. A hint is advice
// only and never changes the result of the insert.
func sourceBlockHint(sourceFile, sourceBlock, destFile string, inserted, destLines []string) string {
	if sourceFile == "" || sourceBlock == "" {
		return ""
	}
	return goImportHint(sourceFile, destFile, inserted, destLines)
}
