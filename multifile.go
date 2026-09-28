package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// multiOpts carries the flag values the multi -file read-only ops need.
type multiOpts struct {
	spec     string
	op       string
	match    string
	endmatch string
	nth      int
	x        bool
	get      string
	extract  string
	wdelim   string
	block    string
	lang     string
	line     string
	endLine  string
	raw      bool
	stream   bool
}

const (
	multiDone = iota
	multiSkip
	multiNone
)

// mapLangs mirrors the case list in doMap's language switch.
var mapLangs = map[string]bool{
	"go": true, "html": true, "sql": true, "python": true, "javascript": true,
	"typescript": true, "css": true, "rust": true, "java": true, "csharp": true,
	"yaml": true, "toml": true, "markdown": true, "ruby": true, "php": true,
	"dockerfile": true, "makefile": true, "json": true, "csv": true,
}

// isMultiSpec reports whether a -file value is a comma list or glob.
// A real file with that exact name always wins.
func isMultiSpec(spec string) bool {
	if !strings.ContainsAny(spec, ",*?[") {
		return false
	}
	if _, err := os.Stat(spec); err == nil {
		return false
	}
	return true
}

// expandMultiSpec splits on commas, globs each element, drops duplicates, keeps order.
func expandMultiSpec(spec string) ([]string, error) {
	seen := map[string]bool{}
	var out []string
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		matches := []string{part}
		if strings.ContainsAny(part, "*?[") {
			m, err := filepath.Glob(part)
			if err != nil {
				return nil, fmt.Errorf("bad glob %q: %v", part, err)
			}
			if len(m) == 0 {
				fmt.Fprintf(os.Stderr, "  SKIP %-40s no files matched\n", part)
			}
			matches = m
		}
		for _, p := range matches {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no files matched: %s", spec)
	}
	return out, nil
}

// detectLang mirrors the extension switch at the top of doMap.
func detectLang(filename string) (string, bool) {
	lower := strings.ToLower(filename)
	switch {
	case strings.HasSuffix(lower, ".go"):
		return "go", true
	case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
		return "html", true
	case strings.HasSuffix(lower, ".sql"):
		return "sql", true
	case strings.HasSuffix(lower, ".py"):
		return "python", true
	case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx"):
		return "javascript", true
	case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
		return "typescript", true
	case strings.HasSuffix(lower, ".css"):
		return "css", true
	case strings.HasSuffix(lower, ".rs"):
		return "rust", true
	case strings.HasSuffix(lower, ".java"):
		return "java", true
	case strings.HasSuffix(lower, ".cs"):
		return "csharp", true
	case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml"):
		return "yaml", true
	case strings.HasSuffix(lower, ".toml"):
		return "toml", true
	case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"):
		return "markdown", true
	case strings.HasSuffix(lower, ".rb"):
		return "ruby", true
	case strings.HasSuffix(lower, ".php"):
		return "php", true
	case strings.HasSuffix(lower, "dockerfile"):
		return "dockerfile", true
	case strings.HasSuffix(lower, "makefile") || strings.HasSuffix(lower, ".mk"):
		return "makefile", true
	case strings.HasSuffix(lower, ".json"):
		return "json", true
	case strings.HasSuffix(lower, ".csv"):
		return "csv", true
	}
	return "", false
}

func multiFail(path string, err error) int {
	fmt.Fprintf(os.Stderr, "  SKIP %-40s %v\n", path, err)
	return multiSkip
}

// runMultiFile runs a read-only op (show, find, map) over every file in o.spec.
// A problem in one file is reported and skipped; it never aborts the batch.
// Returns the process exit code: 1 if the spec is invalid or nothing produced results.
func runMultiFile(o multiOpts) int {
	switch o.op {
	case "show", "find", "map":
	default:
		fmt.Fprintf(os.Stderr, "Error: multi -file supports show, find and map only (got -op %s)\n", o.op)
		return 1
	}
	if o.stream {
		fmt.Fprintln(os.Stderr, "Error: -stream is not supported with multi -file")
		return 1
	}
	if o.op == "find" && o.match == "" {
		fmt.Fprintln(os.Stderr, "Error: -match is required for find")
		return 1
	}
	if o.op == "map" && o.lang != "" && !mapLangs[o.lang] {
		fmt.Fprintf(os.Stderr, "Error: unknown -lang for map: %s\n", o.lang)
		return 1
	}
	paths, err := expandMultiSpec(o.spec)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	done, none, skipped := 0, 0, 0
	for _, p := range paths {
		lines, rerr := readLines(p)
		if rerr != nil {
			multiFail(p, rerr)
			skipped++
			continue
		}
		switch multiOne(o, p, lines) {
		case multiDone:
			done++
		case multiNone:
			none++
		default:
			skipped++
		}
	}
	fmt.Fprintf(os.Stderr, "Multi -file %s: %d with results, %d without, %d skipped\n", o.spec, done, none, skipped)
	if done == 0 {
		return 1
	}
	return 0
}

func multiOne(o multiOpts, path string, lines []string) int {
	switch o.op {
	case "show":
		start, end, err := parseAndResolveLines(o.line, o.endLine, len(lines))
		if err != nil {
			return multiFail(path, err)
		}
		if o.block != "" {
			lang := o.lang
			if lang == "" {
				lang, _ = detectLang(path)
			}
			start, end, err = resolveBlock(lines, lang, o.block)
			if err != nil {
				return multiFail(path, err)
			}
		} else if o.match != "" {
			start, end, err = resolveSourceLines(lines, 0, 0, o.match, o.endmatch, o.nth)
			if err != nil {
				return multiFail(path, err)
			}
		}
		fmt.Printf("==> %s <==\n", path)
		doShow(lines, start, end, o.raw)
		return multiDone
	case "find":
		hits := findMatches(lines, o.match)
		if len(hits) == 0 {
			return multiNone
		}
		if o.nth != 0 {
			if _, err := resolveNth(hits, o.nth); err != nil {
				return multiFail(path, err)
			}
		}
		fmt.Printf("==> %s <==\n", path)
		doFind(lines, o.match, o.nth, o.x, o.get, o.extract, o.wdelim)
		return multiDone
	case "map":
		lang := o.lang
		if lang == "" {
			l, ok := detectLang(path)
			if !ok {
				return multiFail(path, fmt.Errorf("cannot auto-detect language (use -lang)"))
			}
			lang = l
		}
		doMap(lines, path, lang)
		fmt.Println()
		return multiDone
	}
	return multiSkip
}
