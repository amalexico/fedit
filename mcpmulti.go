package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

// mcpRunMulti runs a read-only MCP tool (fedit_show, fedit_find, fedit_map) over every
// file in spec (comma list and/or globs). run is called once per file with an absolute
// path. Same policy as the CLI: report and continue, never abort. A find with no hits in
// a file is not an error and is skipped silently. The result is an error only if no file
// produced results.
func mcpRunMulti(name, spec string, run func(path string) mcpCallResult) mcpCallResult {
	paths, err := expandMultiSpec(spec)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("multi file: %v", err))
	}
	var b strings.Builder
	done, none, skipped := 0, 0, 0
	for _, p := range paths {
		if abs, aerr := filepath.Abs(p); aerr == nil {
			p = abs
		}
		res := run(p)
		text := ""
		if len(res.Content) > 0 {
			text = res.Content[0].Text
		}
		if res.IsError {
			if name == "fedit_find" && strings.HasPrefix(text, "No matches found") {
				none++
				continue
			}
			skipped++
			fmt.Fprintf(&b, "  SKIP %s: %s\n", p, text)
			continue
		}
		done++
		fmt.Fprintf(&b, "==> %s <==\n%s\n\n", p, text)
	}
	fmt.Fprintf(&b, "Multi file %s: %d with results, %d without, %d skipped", spec, done, none, skipped)
	if done == 0 {
		return mcpErrorResult(b.String())
	}
	return mcpOK(b.String())
}
