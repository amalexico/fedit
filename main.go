package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func main() {
	if len(os.Args) >= 2 && os.Args[1] == "mcp" {
		runMCP()
		return
	}

	file := flag.String("file", "", "Path to the file")
	op := flag.String("op", "", "Operation: insert, delete, replace, replaceall, show, write, map, find, insertafter, insertbefore, move, copy (move/copy support -block/-beforeblock/-afterblock with -lang)")
	line := flag.Int("line", 0, "Line number (1-based)")
	endLine := flag.Int("end", 0, "End line for delete/replace range (inclusive)")
	text := flag.String("text", "", "Text to insert/replace (use \\n for newlines, \\t for tabs)")
	textFile := flag.String("textfile", "", "Read insert/replace text from this file instead of -text")
	lang := flag.String("lang", "", "Language for map: go, html, sql")
	match := flag.String("match", "", "Text to search for (find/insertafter/insertbefore)")
	nth := flag.Int("nth", 1, "Which occurrence to match (default 1, use -1 for last)")
	v := flag.Bool("v", false, "Verify: show affected lines after mutation")
	endmatch    := flag.String("endmatch", "", "End line bound for move/copy source (content-based)")
	after       := flag.Int("after", -1, "Destination: insert after line N (0 = beginning of file)")
	before      := flag.Int("before", -1, "Destination: insert before line N")
	aftermatch  := flag.String("aftermatch", "", "Destination: insert after first line matching TEXT")
	beforematch := flag.String("beforematch", "", "Destination: insert before first line matching TEXT")
	times       := flag.Int("times", 1, "Repeat N times (default 1, min 1; for move: cut once, paste N times)")
	block       := flag.String("block", "", "Source: auto-derive range from named block (requires -lang)")
	beforeblock := flag.String("beforeblock", "", "Destination: insert before named block (requires -lang)")
	afterblock  := flag.String("afterblock", "", "Destination: insert after named block (requires -lang)")
	matchRegex  := flag.String("match-regex", "", "Regex pattern for replaceall with capture groups (e.g. -match-regex \"(\\w+)\" -text \"[$1]\")")
	files       := flag.String("files", "", "Apply replaceall to all files matching a glob (e.g. -files \"*.go\")")
		flag.Parse()

	if (*file == "" && *files == "") || *op == "" {
		fmt.Fprintln(os.Stderr, "Usage: fedit -file PATH -op OPERATION [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Operations:")
		fmt.Fprintln(os.Stderr, "  show          Show file with line numbers (-line/-end to limit range)")
		fmt.Fprintln(os.Stderr, "  insert        Insert text AFTER line N (use -line 0 for top)")
		fmt.Fprintln(os.Stderr, "  delete        Delete line N (or range N to -end)")
		fmt.Fprintln(os.Stderr, "  replace       Replace line N (or range N to -end) with text")
		fmt.Fprintln(os.Stderr, "  write         Write text to -file (creates/overwrites)")
		fmt.Fprintln(os.Stderr, "  map           Show structure map (-lang go|html|sql)")
		fmt.Fprintln(os.Stderr, "  find          Find lines containing -match text, print line numbers")
		fmt.Fprintln(os.Stderr, "  insertafter   Find -match text, insert -textfile content AFTER matched line")
		fmt.Fprintln(os.Stderr, "  insertbefore  Find -match text, insert -textfile content BEFORE matched line")
		fmt.Fprintln(os.Stderr, "  replaceall    Replace ALL occurrences of -match text with -text")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Match flags (for find/insertafter/insertbefore):")
		fmt.Fprintln(os.Stderr, "  -match TEXT   Substring to search for (required)")
		fmt.Fprintln(os.Stderr, "  -nth N        Which occurrence (default 1, -1 for last)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Escapes in -text: \\n = newline, \\t = tab, \\\\ = literal backslash")
		os.Exit(1)
	}

	if *op == "write" {
		content := expandText(*text)
		if *textFile != "" {
			var err error
			content, err = readLines(*textFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
				os.Exit(1)
			}
		}
		if len(content) == 0 {
			fmt.Fprintln(os.Stderr, "Nothing to write")
			os.Exit(1)
		}
		if err := writeLines(*file, content); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Wrote %d line(s) to %s\n", len(content), *file)
		return
	}

	var lines []string
	if *file != "" {
		var rdErr error
		lines, rdErr = readLines(*file)
		if rdErr != nil {
			fmt.Fprintf(os.Stderr, "Error reading file: %v\n", rdErr)
			os.Exit(1)
		}
	}

	if *endLine == 0 {
		*endLine = *line
	}

	linesBefore := len(lines)
	startTime := time.Now()

	switch *op {
	case "show":
		doShow(lines, *line, *endLine)
	case "insert":
		newText := resolveText(*text, *textFile)
		doInsert(lines, *file, *line, newText)
	case "delete":
		doDelete(lines, *file, *line, *endLine)
	case "replace":
		newText := resolveText(*text, *textFile)
		doReplace(lines, *file, *line, *endLine, newText)
	case "map":
		doMap(lines, *file, *lang)
	case "find":
		doFind(lines, *match, *nth)
	case "insertafter":
		newText := resolveText(*text, *textFile)
		doInsertMatch(lines, *file, *match, *nth, newText, false)
	case "insertbefore":
		newText := resolveText(*text, *textFile)
		doInsertMatch(lines, *file, *match, *nth, newText, true)
	case "replaceall":
		isRegex := *matchRegex != ""
		if *match == "" && !isRegex {
			fmt.Fprintln(os.Stderr, "replaceall requires -match TEXT or -match-regex PATTERN")
			os.Exit(1)
		}
		searchStr := *match
		if isRegex {
			searchStr = *matchRegex
		}
		replacement := *text
		if *textFile != "" {
			rLines := resolveText("", *textFile)
			replacement = strings.Join(rLines, "\n")
		}
		if *files != "" {
			if isRegex {
				doReplaceAllRegexGlob(*files, searchStr, replacement)
			} else {
				doReplaceAllGlob(*files, searchStr, replacement)
			}
		} else if isRegex {
			doReplaceAllRegex(lines, *file, searchStr, replacement)
		} else {
			doReplaceAll(lines, *file, searchStr, replacement)
		}
	case "move":
		srcL, srcE := *line, *endLine
		srcM, srcEM := *match, *endmatch
		dstA, dstB := *after, *before
		dstAM, dstBM := *aftermatch, *beforematch
		if *block != "" {
			s, e, bErr := resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			srcL, srcE, srcM, srcEM = s, e, "", ""
		}
		if *beforeblock != "" {
			bs, _, bErr := resolveBlock(lines, *lang, *beforeblock)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			dstB, dstA, dstAM, dstBM = bs, -1, "", ""
		}
		if *afterblock != "" {
			_, be, bErr := resolveBlock(lines, *lang, *afterblock)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			dstA, dstB, dstAM, dstBM = be, -1, "", ""
		}
		doMoveOp(lines, *file, srcL, srcE, srcM, srcEM, dstA, dstB, dstAM, dstBM, *nth, *times, linesBefore, startTime, *v)
	case "copy":
		srcL, srcE := *line, *endLine
		srcM, srcEM := *match, *endmatch
		dstA, dstB := *after, *before
		dstAM, dstBM := *aftermatch, *beforematch
		if *block != "" {
			s, e, bErr := resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			srcL, srcE, srcM, srcEM = s, e, "", ""
		}
		if *beforeblock != "" {
			bs, _, bErr := resolveBlock(lines, *lang, *beforeblock)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			dstB, dstA, dstAM, dstBM = bs, -1, "", ""
		}
		if *afterblock != "" {
			_, be, bErr := resolveBlock(lines, *lang, *afterblock)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			dstA, dstB, dstAM, dstBM = be, -1, "", ""
		}
		doCopyOp(lines, *file, srcL, srcE, srcM, srcEM, dstA, dstB, dstAM, dstBM, *nth, *times, linesBefore, startTime, *v)
		default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", *op)
		os.Exit(1)
	}
	if *v {
		switch *op {
		case "show", "map", "find", "write", "move", "copy":
		// read-only ops, no verify
		default:
			center := *line
			if center == 0 {
				searchFor := *match
				if *op == "replaceall" && *text != "" {
					searchFor = *text
				}
				if searchFor != "" {
					updated, err := readLines(*file)
					if err == nil {
						for i, l := range updated {
							if strings.Contains(l, searchFor) {
								center = i + 1
								break
							}
						}
					}
				}
			}
			if center == 0 {
				center = 1
			}
			showVerify(*file, center)

			// Operation stats
			elapsed := time.Since(startTime)
			var elapsedStr string
			if elapsed < time.Millisecond {
				elapsedStr = "<1ms"
			} else {
				elapsedStr = elapsed.Round(time.Millisecond).String()
			}
			linesAfter := 0
			if afterLines, err := readLines(*file); err == nil {
				linesAfter = len(afterLines)
			}
			delta := linesAfter - linesBefore
			fmt.Fprintf(os.Stderr, "\n=== STATS ===\n")
			fmt.Fprintf(os.Stderr, "  op:      %s\n", *op)
			fmt.Fprintf(os.Stderr, "  file:    %s\n", *file)
			if *match != "" {
				fmt.Fprintf(os.Stderr, "  match:   %q\n", *match)
			}
			if delta > 0 {
				fmt.Fprintf(os.Stderr, "  lines:   +%d (%d -> %d)\n", delta, linesBefore, linesAfter)
			} else if delta < 0 {
				fmt.Fprintf(os.Stderr, "  lines:   %d (%d -> %d)\n", delta, linesBefore, linesAfter)
			} else {
				fmt.Fprintf(os.Stderr, "  lines:   0 (unchanged, %d total)\n", linesBefore)
			}
			fmt.Fprintf(os.Stderr, "  elapsed: %s\n", elapsedStr)
		}
	}
}

func resolveText(text, textFile string) []string {
	if textFile != "" {
		lines, err := readLines(textFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
			os.Exit(1)
		}
		return lines
	}
	return expandText(text)
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func writeLines(path string, lines []string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for i, line := range lines {
		w.WriteString(line)
		if i < len(lines)-1 {
			w.WriteByte('\n')
		}
	}
	w.WriteByte('\n')
	return w.Flush()
}

func expandText(text string) []string {
	if text == "" {
		return nil
	}
	placeholder := "\x00ESCAPED_BS\x00"
	text = strings.ReplaceAll(text, "\\\\", placeholder)
	text = strings.ReplaceAll(text, "\\n", "\n")
	text = strings.ReplaceAll(text, "\\t", "\t")
	text = strings.ReplaceAll(text, "\\q", "\"")
	text = strings.ReplaceAll(text, placeholder, "\\")
	return strings.Split(text, "\n")
}

func showVerify(path string, centerLine int) {
	lines, err := readLines(path)
	if err != nil {
		return
	}
	start := centerLine - 3
	end := centerLine + 7
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	fmt.Fprintf(os.Stderr, "\n=== VERIFY ===\n")
	for i := start - 1; i < end; i++ {
		marker := "  "
		if i+1 == centerLine {
			marker = "> "
		}
		fmt.Fprintf(os.Stderr, "%s%4d | %s\n", marker, i+1, lines[i])
	}
	fmt.Fprintf(os.Stderr, "--- %d total lines ---\n", len(lines))
}

// ════════════════════════════════════════════════════════════
// FIND — search for text, print matching lines
// ════════════════════════════════════════════════════════════

func findMatches(lines []string, match string) []int {
	var hits []int
	for i, line := range lines {
		if strings.Contains(line, match) {
			hits = append(hits, i+1) // 1-based
		}
	}
	return hits
}

func resolveNth(hits []int, nth int) (int, error) {
	if len(hits) == 0 {
		return 0, fmt.Errorf("no matches found")
	}
	if nth == -1 {
		return hits[len(hits)-1], nil
	}
	if nth < 1 || nth > len(hits) {
		return 0, fmt.Errorf("requested occurrence %d but only %d match(es) found", nth, len(hits))
	}
	return hits[nth-1], nil
}

func doFind(lines []string, match string, nth int) {
	if match == "" {
		fmt.Fprintln(os.Stderr, "Error: -match is required for find")
		os.Exit(1)
	}

	hits := findMatches(lines, match)
	if len(hits) == 0 {
		fmt.Fprintf(os.Stderr, "No matches for: %s\n", match)
		os.Exit(1)
	}

	width := len(strconv.Itoa(len(lines)))
	fmt.Fprintf(os.Stderr, "Found %d match(es) for: %s\n", len(hits), match)

	// Show context: matched line + 1 line before and after
	for _, ln := range hits {
		fmt.Println()
		start := ln - 2
		if start < 0 {
			start = 0
		}
		end := ln + 1
		if end > len(lines) {
			end = len(lines)
		}
		for i := start; i < end; i++ {
			marker := " "
			if i+1 == ln {
				marker = ">"
			}
			fmt.Printf(" %s %*d | %s\n", marker, width, i+1, lines[i])
		}
	}

	// If nth specified, highlight which one
	if nth != 0 {
		resolved, err := resolveNth(hits, nth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n%v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "\nOccurrence %d is at line %d\n", nth, resolved)
	}
}

// ════════════════════════════════════════════════════════════
// INSERT AFTER/BEFORE — content-based insertion
// ════════════════════════════════════════════════════════════

func doInsertMatch(lines []string, path, match string, nth int, newLines []string, before bool) {
	if match == "" {
		fmt.Fprintln(os.Stderr, "Error: -match is required for insertafter/insertbefore")
		os.Exit(1)
	}
	if len(newLines) == 0 {
		fmt.Fprintln(os.Stderr, "Nothing to insert (-text or -textfile is empty)")
		os.Exit(1)
	}

	hits := findMatches(lines, match)
	targetLine, err := resolveNth(hits, nth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Match error: %v\nSearched for: %s\n", err, match)
		os.Exit(1)
	}

	// Show what we matched for confirmation
	fmt.Fprintf(os.Stderr, "Matched line %d: %s\n", targetLine, strings.TrimSpace(lines[targetLine-1]))

	var insertAfter int
	direction := "after"
	if before {
		insertAfter = targetLine - 1 // insert after the line BEFORE the match
		direction = "before"
	} else {
		insertAfter = targetLine // insert after the matched line
	}

	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertAfter]...)
	result = append(result, newLines...)
	result = append(result, lines[insertAfter:]...)

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Inserted %d line(s) %s line %d (%d total now)\n",
		len(newLines), direction, targetLine, len(result))
}

// ════════════════════════════════════════════════════════════
// SHOW / INSERT / DELETE / REPLACE
// ════════════════════════════════════════════════════════════

func doShow(lines []string, start, end int) {
	if start == 0 && end == 0 {
		start = 1
		end = len(lines)
	}
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	width := len(strconv.Itoa(end))
	for i := start; i <= end; i++ {
		fmt.Printf("%*d | %s\n", width, i, lines[i-1])
	}
	fmt.Fprintf(os.Stderr, "--- %d total lines ---\n", len(lines))
}

func doInsert(lines []string, path string, afterLine int, newLines []string) {
	if len(newLines) == 0 {
		fmt.Fprintln(os.Stderr, "Nothing to insert (-text or -textfile is empty)")
		os.Exit(1)
	}
	if afterLine < 0 || afterLine > len(lines) {
		fmt.Fprintf(os.Stderr, "Line %d out of range (file has %d lines)\n", afterLine, len(lines))
		os.Exit(1)
	}

	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:afterLine]...)
	result = append(result, newLines...)
	result = append(result, lines[afterLine:]...)

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Inserted %d line(s) after line %d (%d total now)\n",
		len(newLines), afterLine, len(result))
}

func doDelete(lines []string, path string, start, end int) {
	if start < 1 || end > len(lines) || start > end {
		fmt.Fprintf(os.Stderr, "Invalid range %d-%d (file has %d lines)\n", start, end, len(lines))
		os.Exit(1)
	}
	result := make([]string, 0, len(lines)-(end-start+1))
	result = append(result, lines[:start-1]...)
	result = append(result, lines[end:]...)

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Deleted lines %d-%d (%d total now)\n", start, end, len(result))
}

func doReplace(lines []string, path string, start, end int, newLines []string) {
	if start < 1 || end > len(lines) || start > end {
		fmt.Fprintf(os.Stderr, "Invalid range %d-%d (file has %d lines)\n", start, end, len(lines))
		os.Exit(1)
	}
	result := make([]string, 0, len(lines)-(end-start+1)+len(newLines))
	result = append(result, lines[:start-1]...)
	result = append(result, newLines...)
	result = append(result, lines[end:]...)

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Replaced lines %d-%d with %d line(s) (%d total now)\n",
		start, end, len(newLines), len(result))
}
func doReplaceAll(lines []string, path, search, replacement string) {
	if search == "" {
		fmt.Fprintln(os.Stderr, "replaceall requires -match (text to find)")
		os.Exit(1)
	}
	count := 0
	for i, line := range lines {
		if strings.Contains(line, search) {
			lines[i] = strings.ReplaceAll(line, search, replacement)
			count++
		}
	}
	if count == 0 {
		fmt.Fprintf(os.Stderr, "No lines contain: %s\n", search)
		os.Exit(1)
	}
	if err := writeLines(path, lines); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Replaced '%s' on %d line(s) (%d total)\n", search, count, len(lines))
}

// ════════════════════════════════════════════════════════════
// MAP — structural overview of source files
// ════════════════════════════════════════════════════════════

func doMap(lines []string, filename, lang string) {
	if lang == "" {
		// Auto-detect from extension
		lower := strings.ToLower(filename)
		switch {
		case strings.HasSuffix(lower, ".go"):
			lang = "go"
		case strings.HasSuffix(lower, ".html") || strings.HasSuffix(lower, ".htm"):
			lang = "html"
		case strings.HasSuffix(lower, ".sql"):
			lang = "sql"
		case strings.HasSuffix(lower, ".py"):
			lang = "python"
		case strings.HasSuffix(lower, ".js") || strings.HasSuffix(lower, ".jsx"):
			lang = "javascript"
		case strings.HasSuffix(lower, ".ts") || strings.HasSuffix(lower, ".tsx"):
			lang = "typescript"
		case strings.HasSuffix(lower, ".css"):
			lang = "css"
		case strings.HasSuffix(lower, ".rs"):
			lang = "rust"
		case strings.HasSuffix(lower, ".java"):
			lang = "java"
		case strings.HasSuffix(lower, ".cs"):
			lang = "csharp"
		case strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml"):
			lang = "yaml"
		case strings.HasSuffix(lower, ".toml"):
			lang = "toml"
		case strings.HasSuffix(lower, ".md") || strings.HasSuffix(lower, ".markdown"):
			lang = "markdown"
		case strings.HasSuffix(lower, ".rb"):
			lang = "ruby"
		case strings.HasSuffix(lower, ".php"):
			lang = "php"
		case strings.HasSuffix(lower, "dockerfile"):
			lang = "dockerfile"
		case strings.HasSuffix(lower, "makefile") || strings.HasSuffix(lower, ".mk"):
			lang = "makefile"
		default:
			fmt.Fprintln(os.Stderr, "Cannot auto-detect language. Use -lang flag.")
			os.Exit(1)
		}
	}

	fmt.Printf("=== MAP: %s (%s, %d lines) ===\n\n", filename, lang, len(lines))

	switch lang {
	case "go":
		doMapGo(lines)
	case "html":
		doMapHTML(lines)
	case "sql":
		doMapSQL(lines)
	case "python":
		doMapPython(lines)
	case "javascript", "typescript":
		doMapJavaScript(lines)
	case "css":
		doMapCSS(lines)
	case "rust":
		doMapRust(lines)
	case "java":
		doMapJava(lines)
	case "csharp":
		doMapCSharp(lines)
	case "yaml":
		doMapYAML(lines)
	case "toml":
		doMapTOML(lines)
	case "markdown":
		doMapMarkdown(lines)
	case "ruby":
		doMapRuby(lines)
	case "php":
		doMapPHP(lines)
	case "dockerfile":
		doMapDockerfile(lines)
	case "makefile":
		doMapMakefile(lines)
	default:
		fmt.Fprintf(os.Stderr, "Unknown language: %s\n", lang)
		os.Exit(1)
	}
}

func doMapGo(lines []string) {
	type funcInfo struct {
		line int
		name string
	}
	var funcs []funcInfo
	funcNames := make(map[string][]int)
	width := len(strconv.Itoa(len(lines)))

	reFuncDecl := regexp.MustCompile(`^func\s+(\(.*?\)\s*)?(\w+)\s*\(`)
	reType := regexp.MustCompile(`^type\s+(\w+)\s+`)
	reVar := regexp.MustCompile(`^var\s+`)
	reConst := regexp.MustCompile(`^const\s+`)

	// Track brace depth to find function end lines
	type funcRange struct {
		name  string
		start int
		end   int
	}
	var ranges []funcRange
	inFunc := false
	braceDepth := 0
	currentFunc := ""
	currentStart := 0

	for i, line := range lines {
		ln := i + 1
		trimmed := strings.TrimSpace(line)

		// Package
		if strings.HasPrefix(trimmed, "package ") {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
		}

		// Import
		if trimmed == "import (" || strings.HasPrefix(trimmed, "import \"") {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
		}

		// Type
		if m := reType.FindStringSubmatch(trimmed); m != nil {
			fmt.Printf("%*d │ type %s\n", width, ln, m[1])
		}

		// Var block
		if reVar.MatchString(trimmed) {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
		}

		// Const block
		if reConst.MatchString(trimmed) {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
		}

		// Function declaration
		if m := reFuncDecl.FindStringSubmatch(trimmed); m != nil {
			receiver := ""
			if m[1] != "" {
				receiver = strings.TrimSpace(m[1]) + " "
			}
			fname := receiver + m[2]
			funcs = append(funcs, funcInfo{ln, fname})
			funcNames[m[2]] = append(funcNames[m[2]], ln)

			if inFunc {
				// Previous function never closed properly
				ranges = append(ranges, funcRange{currentFunc, currentStart, ln - 1})
			}
			inFunc = true
			braceDepth = 0
			currentFunc = fname
			currentStart = ln
		}

		// Track braces for function boundaries
		if inFunc {
			braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if braceDepth <= 0 && ln > currentStart {
				ranges = append(ranges, funcRange{currentFunc, currentStart, ln})
				inFunc = false
			}
		}
	}

	// Close last function if still open
	if inFunc {
		ranges = append(ranges, funcRange{currentFunc, currentStart, len(lines)})
	}

	// Print functions with ranges
	fmt.Println()
	fmt.Printf("── Functions (%d) ──\n", len(ranges))
	for _, r := range ranges {
		fmt.Printf("%*d–%-*d │ func %s\n", width, r.start, width, r.end, r.name)
	}

	// Check for duplicates
	fmt.Println()
	hasDupes := false
	for name, locs := range funcNames {
		if len(locs) > 1 {
			if !hasDupes {
				fmt.Println("⚠ DUPLICATE FUNCTIONS:")
				hasDupes = true
			}
			strs := make([]string, len(locs))
			for i, l := range locs {
				strs[i] = strconv.Itoa(l)
			}
			fmt.Printf("  %s defined at lines: %s\n", name, strings.Join(strs, ", "))
		}
	}
	if !hasDupes {
		fmt.Println("✓ No duplicate functions")
	}

	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapHTML(lines []string) {
	width := len(strconv.Itoa(len(lines)))

	reFuncDecl := regexp.MustCompile(`function\s+(\w+)\s*\(`)
	reLetConst := regexp.MustCompile(`^\s*(let|const|var)\s+(\w+)`)

	type funcInfo struct {
		name  string
		start int
		end   int
	}
	var jsFuncs []funcInfo
	funcNames := make(map[string][]int)
	inScript := false
	inStyle := false
	inJSFunc := false
	braceDepth := 0
	currentFunc := ""
	currentStart := 0

	for i, line := range lines {
		ln := i + 1
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Style tags
		if strings.Contains(lower, "<style") {
			fmt.Printf("%*d │ <style>\n", width, ln)
			inStyle = true
		}
		if strings.Contains(lower, "</style>") {
			fmt.Printf("%*d │ </style>\n", width, ln)
			inStyle = false
		}

		// Script tags
		if strings.Contains(lower, "<script") {
			fmt.Printf("%*d │ <script>\n", width, ln)
			inScript = true
		}
		if strings.Contains(lower, "</script>") {
			if inJSFunc {
				jsFuncs = append(jsFuncs, funcInfo{currentFunc, currentStart, ln - 1})
				inJSFunc = false
			}
			fmt.Printf("%*d │ </script>\n", width, ln)
			inScript = false
		}

		// Template markers
		if strings.Contains(trimmed, "{{define") || strings.Contains(trimmed, "{{template") ||
			strings.Contains(trimmed, "{{block") {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
		}
		if strings.Contains(trimmed, "{{end}}") {
			fmt.Printf("%*d │ {{end}}\n", width, ln)
		}

		// Inside script block
		if inScript && !inStyle {
			// Function declarations
			if m := reFuncDecl.FindStringSubmatch(trimmed); m != nil {
				if inJSFunc {
					// Close previous
					jsFuncs = append(jsFuncs, funcInfo{currentFunc, currentStart, ln - 1})
				}
				inJSFunc = true
				braceDepth = 0
				currentFunc = m[1]
				currentStart = ln
				funcNames[m[1]] = append(funcNames[m[1]], ln)
			}

			// Top-level let/const/var
			if m := reLetConst.FindStringSubmatch(trimmed); m != nil && !inJSFunc {
				fmt.Printf("%*d │ %s %s\n", width, ln, m[1], m[2])
			}

			// Track braces for JS function boundaries
			if inJSFunc {
				braceDepth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
				if braceDepth <= 0 && ln > currentStart {
					jsFuncs = append(jsFuncs, funcInfo{currentFunc, currentStart, ln})
					inJSFunc = false
				}
			}
		}
	}

	// Close last function if still open
	if inJSFunc {
		jsFuncs = append(jsFuncs, funcInfo{currentFunc, currentStart, len(lines)})
	}

	// Print JS functions with ranges
	if len(jsFuncs) > 0 {
		fmt.Println()
		fmt.Printf("── JS Functions (%d) ──\n", len(jsFuncs))
		for _, f := range jsFuncs {
			fmt.Printf("%*d–%-*d │ function %s()\n", width, f.start, width, f.end, f.name)
		}
	}

	// Check for duplicates
	fmt.Println()
	hasDupes := false
	for name, locs := range funcNames {
		if len(locs) > 1 {
			if !hasDupes {
				fmt.Println("⚠ DUPLICATE JS FUNCTIONS:")
				hasDupes = true
			}
			strs := make([]string, len(locs))
			for i, l := range locs {
				strs[i] = strconv.Itoa(l)
			}
			fmt.Printf("  %s() defined at lines: %s\n", name, strings.Join(strs, ", "))
		}
	}
	if !hasDupes {
		fmt.Println("✓ No duplicate JS functions")
	}

	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapSQL(lines []string) {
	width := len(strconv.Itoa(len(lines)))

	type block struct {
		name  string
		start int
		end   int
	}
	var blocks []block
	nameRe := regexp.MustCompile(`^--\s*name:\s*(.+)$`)
	headerRe := regexp.MustCompile(`^--\s*=+`)
	commentRe := regexp.MustCompile(`^--\s+(.+)$`)

	currentName := ""
	currentStart := 0

	for i, line := range lines {
		ln := i + 1
		trimmed := strings.TrimSpace(line)

		// Section headers (== lines)
		if headerRe.MatchString(trimmed) {
			fmt.Printf("%*d │ %s\n", width, ln, trimmed)
			continue
		}

		// Named query
		if m := nameRe.FindStringSubmatch(trimmed); m != nil {
			if currentName != "" {
				blocks = append(blocks, block{currentName, currentStart, ln - 1})
			}
			currentName = m[1]
			currentStart = ln
			continue
		}

		// Section comment (like "-- JOBS")
		if m := commentRe.FindStringSubmatch(trimmed); m != nil && currentName == "" {
			if !strings.HasPrefix(m[1], "=") {
				fmt.Printf("%*d │ -- %s\n", width, ln, m[1])
			}
		}
	}

	// Close last block
	if currentName != "" {
		blocks = append(blocks, block{currentName, currentStart, len(lines)})
	}

	// Print query blocks
	if len(blocks) > 0 {
		fmt.Println()
		fmt.Printf("── SQL Queries (%d) ──\n", len(blocks))
		for _, b := range blocks {
			fmt.Printf("%*d–%-*d │ %s\n", width, b.start, width, b.end, b.name)
		}
	}

	// Check for duplicate names
	nameCount := make(map[string][]int)
	for _, b := range blocks {
		parts := strings.Fields(b.name)
		qname := parts[0]
		nameCount[qname] = append(nameCount[qname], b.start)
	}
	fmt.Println()
	hasDupes := false
	for name, locs := range nameCount {
		if len(locs) > 1 {
			if !hasDupes {
				fmt.Println("⚠ DUPLICATE QUERIES:")
				hasDupes = true
			}
			strs := make([]string, len(locs))
			for i, l := range locs {
				strs[i] = strconv.Itoa(l)
			}
			fmt.Printf("  %s at lines: %s\n", name, strings.Join(strs, ", "))
		}
	}
	if !hasDupes {
		fmt.Println("✓ No duplicate queries")
	}

	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapPython(lines []string) {
	reClass := regexp.MustCompile(`^(\s*)class\s+(\w+)`)
	reDef := regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(`)
	reImport := regexp.MustCompile(`^(import |from \S+ import )`)

	fmt.Println("── Imports ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reImport.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Classes & Functions ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reClass.FindStringSubmatch(line); m != nil {
			indent := ""
			if len(m[1]) > 0 {
				indent = "(nested) "
			}
			fmt.Printf("%4d │ %sclass %s\n", i+1, indent, m[2])
			names["class:"+m[2]]++
		}
		if m := reDef.FindStringSubmatch(line); m != nil {
			indent := "  "
			if len(m[1]) == 0 {
				indent = ""
			}
			dec := ""
			if i > 0 {
				prev := strings.TrimSpace(lines[i-1])
				if strings.HasPrefix(prev, "@") {
					dec = " " + prev
				}
			}
			fmt.Printf("%4d │ %sdef %s%s\n", i+1, indent, m[2], dec)
			names["def:"+m[2]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapJavaScript(lines []string) {
	reFunc := regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)`)
	reArrow := regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(`)
	reClass := regexp.MustCompile(`^(?:export\s+)?class\s+(\w+)`)
	reMethod := regexp.MustCompile(`^\s+(?:async\s+)?(\w+)\s*\(`)
	reImport := regexp.MustCompile(`^import\s+`)
	reExport := regexp.MustCompile(`^export\s+(?:default\s+)?{`)

	fmt.Println("── Imports & Exports ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reImport.MatchString(t) || reExport.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Classes, Functions & Arrow Functions ──")
	names := map[string]int{}
	inClass := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if m := reClass.FindStringSubmatch(t); m != nil {
			fmt.Printf("%4d │ class %s\n", i+1, m[1])
			names[m[1]]++
			inClass = true
			continue
		}
		if inClass && t == "}" {
			inClass = false
			continue
		}
		if inClass {
			if m := reMethod.FindStringSubmatch(line); m != nil {
				if m[1] != "if" && m[1] != "for" && m[1] != "while" && m[1] != "switch" && m[1] != "catch" {
					fmt.Printf("%4d │   method %s\n", i+1, m[1])
				}
			}
			continue
		}
		if m := reFunc.FindStringSubmatch(t); m != nil {
			fmt.Printf("%4d │ function %s\n", i+1, m[1])
			names[m[1]]++
		} else if m := reArrow.FindStringSubmatch(t); m != nil {
			fmt.Printf("%4d │ const %s (arrow)\n", i+1, m[1])
			names[m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapCSS(lines []string) {
	reSelector := regexp.MustCompile(`^([a-zA-Z.#@\[\*:][^{]*)\{`)
	reMedia := regexp.MustCompile(`^@media\s+`)
	reKeyframes := regexp.MustCompile(`^@keyframes\s+(\S+)`)
	reImport := regexp.MustCompile(`^@import\s+`)
	reVar := regexp.MustCompile(`^\s*(--[\w-]+)\s*:`)

	fmt.Println("── Imports ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reImport.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Custom Properties ──")
	for i, line := range lines {
		if m := reVar.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, m[1])
		}
	}

	fmt.Println("\n── Selectors & At-Rules ──")
	names := map[string]int{}
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if m := reKeyframes.FindStringSubmatch(t); m != nil {
			fmt.Printf("%4d │ @keyframes %s\n", i+1, m[1])
			names["@keyframes "+m[1]]++
			continue
		}
		if reMedia.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
			continue
		}
		if m := reSelector.FindStringSubmatch(t); m != nil {
			sel := strings.TrimSpace(m[1])
			fmt.Printf("%4d │ %s\n", i+1, sel)
			names[sel]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate selectors")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapRust(lines []string) {
	reFn := regexp.MustCompile(`^\s*(?:pub\s+)?(?:async\s+)?fn\s+(\w+)`)
	reStruct := regexp.MustCompile(`^\s*(?:pub\s+)?struct\s+(\w+)`)
	reEnum := regexp.MustCompile(`^\s*(?:pub\s+)?enum\s+(\w+)`)
	reTrait := regexp.MustCompile(`^\s*(?:pub\s+)?trait\s+(\w+)`)
	reImpl := regexp.MustCompile(`^\s*impl(?:<[^>]*>)?\s+(\w+)`)
	reMod := regexp.MustCompile(`^\s*(?:pub\s+)?mod\s+(\w+)`)
	reUse := regexp.MustCompile(`^\s*use\s+`)
	reMacro := regexp.MustCompile(`^\s*macro_rules!\s+(\w+)`)

	fmt.Println("── Use Statements ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reUse.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Modules ──")
	for i, line := range lines {
		if m := reMod.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ mod %s\n", i+1, m[1])
		}
	}

	fmt.Println("\n── Types ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reStruct.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ struct %s\n", i+1, m[1])
			names["struct:"+m[1]]++
		}
		if m := reEnum.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ enum %s\n", i+1, m[1])
			names["enum:"+m[1]]++
		}
		if m := reTrait.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ trait %s\n", i+1, m[1])
			names["trait:"+m[1]]++
		}
	}

	fmt.Println("\n── Impls ──")
	for i, line := range lines {
		if m := reImpl.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ impl %s\n", i+1, m[1])
		}
	}

	fmt.Println("\n── Functions & Macros ──")
	for i, line := range lines {
		if m := reFn.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ fn %s\n", i+1, m[1])
			names["fn:"+m[1]]++
		}
		if m := reMacro.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ macro %s\n", i+1, m[1])
			names["macro:"+m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapJava(lines []string) {
	reClass := regexp.MustCompile(`^\s*(?:public|private|protected)?\s*(?:abstract|final)?\s*(?:class|interface|enum)\s+(\w+)`)
	reMethod := regexp.MustCompile(`^\s*(?:public|private|protected)?\s*(?:static\s+)?(?:abstract\s+)?(?:synchronized\s+)?(?:final\s+)?[\w<>$$$$,\s]+\s+(\w+)\s*\(`)
	reImport := regexp.MustCompile(`^\s*import\s+`)
	rePackage := regexp.MustCompile(`^\s*package\s+(\S+)`)
	reAnnotation := regexp.MustCompile(`^\s*@(\w+)`)

	fmt.Println("── Package & Imports ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if rePackage.MatchString(t) || reImport.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Classes, Interfaces & Enums ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reClass.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, strings.TrimSpace(line))
			names["type:"+m[1]]++
		}
	}

	fmt.Println("\n── Methods ──")
	for i, line := range lines {
		if reClass.MatchString(line) {
			continue
		}
		if m := reMethod.FindStringSubmatch(line); m != nil {
			if m[1] == "if" || m[1] == "for" || m[1] == "while" || m[1] == "switch" || m[1] == "catch" || m[1] == "return" {
				continue
			}
			ann := ""
			if i > 0 {
				if a := reAnnotation.FindStringSubmatch(lines[i-1]); a != nil {
					ann = " @" + a[1]
				}
			}
			fmt.Printf("%4d │ %s%s\n", i+1, m[1], ann)
			names["method:"+m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapCSharp(lines []string) {
	reClass := regexp.MustCompile(`^\s*(?:public|private|protected|internal)?\s*(?:static\s+)?(?:abstract\s+)?(?:sealed\s+)?(?:partial\s+)?(?:class|interface|struct|enum|record)\s+(\w+)`)
	reMethod := regexp.MustCompile(`^\s*(?:public|private|protected|internal)?\s*(?:static\s+)?(?:async\s+)?(?:virtual\s+)?(?:override\s+)?(?:abstract\s+)?[\w<>$$$$?,\s]+\s+(\w+)\s*\(`)
	reNamespace := regexp.MustCompile(`^\s*namespace\s+(\S+)`)
	reUsing := regexp.MustCompile(`^\s*using\s+`)
	reProp := regexp.MustCompile(`^\s*(?:public|private|protected|internal)?\s*(?:static\s+)?[\w<>$$$$?,]+\s+(\w+)\s*\{\s*get`)

	fmt.Println("── Usings & Namespace ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reUsing.MatchString(t) || reNamespace.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Types ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reClass.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, strings.TrimSpace(line))
			names["type:"+m[1]]++
		}
	}

	fmt.Println("\n── Methods & Properties ──")
	for i, line := range lines {
		if reClass.MatchString(line) {
			continue
		}
		if m := reProp.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ property %s\n", i+1, m[1])
			names["prop:"+m[1]]++
			continue
		}
		if m := reMethod.FindStringSubmatch(line); m != nil {
			if m[1] == "if" || m[1] == "for" || m[1] == "while" || m[1] == "switch" || m[1] == "catch" || m[1] == "return" || m[1] == "new" {
				continue
			}
			fmt.Printf("%4d │ method %s\n", i+1, m[1])
			names["method:"+m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapYAML(lines []string) {
	reTopKey := regexp.MustCompile(`^([a-zA-Z_][\w.-]*)\s*:`)
	reComment := regexp.MustCompile(`^\s*#`)
	reDoc := regexp.MustCompile(`^---\s*$`)

	fmt.Println("── Document Separators ──")
	for i, line := range lines {
		if reDoc.MatchString(line) {
			fmt.Printf("%4d │ ---\n", i+1)
		}
	}

	fmt.Println("\n── Top-Level Keys ──")
	names := map[string]int{}
	for i, line := range lines {
		if reComment.MatchString(line) {
			continue
		}
		if m := reTopKey.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, m[1])
			names[m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate top-level keys")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapTOML(lines []string) {
	reTable := regexp.MustCompile(`^\s*\[([^\]]+)\]`)
	reArrayTable := regexp.MustCompile(`^\s*\[\[([^\]]+)\]\]`)
	reKey := regexp.MustCompile(`^([a-zA-Z_][\w.-]*)\s*=`)
	reComment := regexp.MustCompile(`^\s*#`)

	fmt.Println("── Tables ──")
	names := map[string]int{}
	for i, line := range lines {
		if reComment.MatchString(line) {
			continue
		}
		if m := reArrayTable.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ [[%s]]\n", i+1, strings.TrimSpace(m[1]))
			names["[["+strings.TrimSpace(m[1])+"]]"]++
			continue
		}
		if m := reTable.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ [%s]\n", i+1, strings.TrimSpace(m[1]))
			names["["+strings.TrimSpace(m[1])+"]"]++
		}
	}

	fmt.Println("\n── Top-Level Keys ──")
	inTable := false
	for i, line := range lines {
		if reComment.MatchString(line) {
			continue
		}
		if reTable.MatchString(line) || reArrayTable.MatchString(line) {
			inTable = true
			continue
		}
		if !inTable {
			if m := reKey.FindStringSubmatch(line); m != nil {
				fmt.Printf("%4d │ %s\n", i+1, m[1])
			}
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 && !strings.HasPrefix(name, "[[") {
			if !dups {
				fmt.Println("⚠ Duplicate tables:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate tables")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapMarkdown(lines []string) {
	reHeading := regexp.MustCompile(`^(#{1,6})\s+(.+)`)
	reLink := regexp.MustCompile(`$$([^$$]+)\]\(([^)]+)\)`)
	reCodeBlock := regexp.MustCompile("^```(\\w*)")

	fmt.Println("── Headings ──")
	for i, line := range lines {
		if m := reHeading.FindStringSubmatch(line); m != nil {
			indent := strings.Repeat("  ", len(m[1])-1)
			fmt.Printf("%4d │ %s%s %s\n", i+1, indent, m[1], m[2])
		}
	}

	fmt.Println("\n── Code Blocks ──")
	inCode := false
	for i, line := range lines {
		if m := reCodeBlock.FindStringSubmatch(line); m != nil {
			if !inCode {
				lang := m[1]
				if lang == "" {
					lang = "(no lang)"
				}
				fmt.Printf("%4d │ ```%s\n", i+1, lang)
				inCode = true
			} else {
				inCode = false
			}
		}
	}

	fmt.Println("\n── Links ──")
	for i, line := range lines {
		matches := reLink.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			fmt.Printf("%4d │ [%s] → %s\n", i+1, m[1], m[2])
		}
	}

	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapRuby(lines []string) {
	reClass := regexp.MustCompile(`^\s*class\s+(\S+)`)
	reModule := regexp.MustCompile(`^\s*module\s+(\S+)`)
	reDef := regexp.MustCompile(`^\s*def\s+(\S+)`)
	reRequire := regexp.MustCompile(`^\s*require\s+`)
	reAttr := regexp.MustCompile(`^\s*attr_(accessor|reader|writer)\s+(.+)`)

	fmt.Println("── Requires ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reRequire.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Modules & Classes ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reModule.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ module %s\n", i+1, m[1])
			names["module:"+m[1]]++
		}
		if m := reClass.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ class %s\n", i+1, m[1])
			names["class:"+m[1]]++
		}
	}

	fmt.Println("\n── Methods ──")
	for i, line := range lines {
		if m := reDef.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ def %s\n", i+1, m[1])
			names["def:"+m[1]]++
		}
	}

	fmt.Println("\n── Attributes ──")
	for i, line := range lines {
		if m := reAttr.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ attr_%s %s\n", i+1, m[1], m[2])
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapPHP(lines []string) {
	reClass := regexp.MustCompile(`^\s*(?:abstract\s+)?(?:final\s+)?class\s+(\w+)`)
	reInterface := regexp.MustCompile(`^\s*interface\s+(\w+)`)
	reTrait := regexp.MustCompile(`^\s*trait\s+(\w+)`)
	reFunc := regexp.MustCompile(`^\s*(?:public|private|protected)?\s*(?:static\s+)?function\s+(\w+)`)
	reNamespace := regexp.MustCompile(`^\s*namespace\s+(\S+)`)
	reUse := regexp.MustCompile(`^\s*use\s+(\S+)`)

	fmt.Println("── Namespace & Use ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if reNamespace.MatchString(t) || reUse.MatchString(t) {
			fmt.Printf("%4d │ %s\n", i+1, t)
		}
	}

	fmt.Println("\n── Classes, Interfaces & Traits ──")
	names := map[string]int{}
	for i, line := range lines {
		if m := reClass.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ class %s\n", i+1, m[1])
			names["class:"+m[1]]++
		}
		if m := reInterface.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ interface %s\n", i+1, m[1])
			names["interface:"+m[1]]++
		}
		if m := reTrait.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ trait %s\n", i+1, m[1])
			names["trait:"+m[1]]++
		}
	}

	fmt.Println("\n── Functions ──")
	for i, line := range lines {
		if m := reFunc.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ function %s\n", i+1, m[1])
			names["func:"+m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate definitions")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapDockerfile(lines []string) {
	reFrom := regexp.MustCompile(`(?i)^\s*FROM\s+(\S+)`)
	reAs := regexp.MustCompile(`(?i)\s+AS\s+(\S+)`)
	reInstruction := regexp.MustCompile(`(?i)^\s*(RUN|COPY|ADD|ENV|ARG|EXPOSE|VOLUME|WORKDIR|ENTRYPOINT|CMD|LABEL|HEALTHCHECK|USER|SHELL)\s`)

	fmt.Println("── Stages ──")
	for i, line := range lines {
		if m := reFrom.FindStringSubmatch(line); m != nil {
			stage := ""
			if a := reAs.FindStringSubmatch(line); a != nil {
				stage = " AS " + a[1]
			}
			fmt.Printf("%4d │ FROM %s%s\n", i+1, m[1], stage)
		}
	}

	fmt.Println("\n── Instructions ──")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "#") || t == "" {
			continue
		}
		if m := reInstruction.FindStringSubmatch(line); m != nil {
			summary := t
			if len(summary) > 80 {
				summary = summary[:77] + "..."
			}
			fmt.Printf("%4d │ %s\n", i+1, summary)
		}
	}

	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

func doMapMakefile(lines []string) {
	reTarget := regexp.MustCompile(`^([a-zA-Z_][\w.-]*)\s*:`)
	reVar := regexp.MustCompile(`^([A-Z_][\w]*)\s*[:?]?=`)
	reInclude := regexp.MustCompile(`^-?include\s+(.+)`)
	rePhony := regexp.MustCompile(`^\.PHONY\s*:\s*(.+)`)

	fmt.Println("── Includes ──")
	for i, line := range lines {
		if m := reInclude.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ include %s\n", i+1, m[1])
		}
	}

	fmt.Println("\n── Variables ──")
	for i, line := range lines {
		if m := reVar.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, m[1])
		}
	}

	fmt.Println("\n── .PHONY ──")
	for i, line := range lines {
		if m := rePhony.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ .PHONY: %s\n", i+1, strings.TrimSpace(m[1]))
		}
	}

	fmt.Println("\n── Targets ──")
	names := map[string]int{}
	for i, line := range lines {
		if reVar.MatchString(line) || rePhony.MatchString(line) {
			continue
		}
		if m := reTarget.FindStringSubmatch(line); m != nil {
			fmt.Printf("%4d │ %s\n", i+1, m[1])
			names[m[1]]++
		}
	}

	fmt.Println()
	dups := false
	for name, count := range names {
		if count > 1 {
			if !dups {
				fmt.Println("⚠ Duplicates:")
				dups = true
			}
			fmt.Printf("  %s (%d times)\n", name, count)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate targets")
	}
	fmt.Fprintf(os.Stderr, "\n--- %d total lines ---\n", len(lines))
}

// ════════════════════════════════════════════════════════════
// MOVE / COPY — block relocation operations (v1.2.0)
// ════════════════════════════════════════════════════════════

// resolveSourceLines resolves the source range for move/copy.
// Uses lineFlag/endFlag for explicit ranges, or matchFlag/endmatchFlag for content-based bounds.
func resolveSourceLines(lines []string, lineFlag, endFlag int, matchFlag, endmatchFlag string, nth int) (srcStart, srcEnd int, err error) {
	if matchFlag != "" {
		hits := findMatches(lines, matchFlag)
		srcStart, err = resolveNth(hits, nth)
		if err != nil {
			return 0, 0, fmt.Errorf("source -match %q: %v", matchFlag, err)
		}
		if endmatchFlag != "" {
			ehits := findMatches(lines, endmatchFlag)
			for _, h := range ehits {
				if h >= srcStart {
					srcEnd = h
					break
				}
			}
			if srcEnd == 0 {
				return 0, 0, fmt.Errorf("source -endmatch %q: no match at or after line %d", endmatchFlag, srcStart)
			}
		} else if endFlag != 0 {
			srcEnd = endFlag
		} else {
			return 0, 0, fmt.Errorf("source -match requires an end bound: use -end N or -endmatch TEXT")
		}
		return srcStart, srcEnd, nil
	}
	if lineFlag != 0 {
		srcStart = lineFlag
		if endFlag != 0 {
			srcEnd = endFlag
		} else {
			srcEnd = lineFlag
		}
		return srcStart, srcEnd, nil
	}
	return 0, 0, fmt.Errorf("move/copy requires a source: use -line/-end or -match/-endmatch")
}

// resolveDestLine resolves the insertion point for move/copy.
// Exactly one of the four destination flags must be set (non-sentinel).
// afterFlag/beforeFlag use -1 as "not set" sentinel; afterMatchFlag/beforeMatchFlag use "".
// Returns:
//   destAfter  — insert after this line index (0 = top of file)
//   destLine   — human-facing line number used for overlap checks
//   destDesc   — human-readable description for messages
func resolveDestLine(lines []string, afterFlag, beforeFlag int, afterMatchFlag, beforeMatchFlag string) (destAfter, destLine int, destDesc string, err error) {
	specified := 0
	if afterFlag != -1 {
		specified++
	}
	if beforeFlag != -1 {
		specified++
	}
	if afterMatchFlag != "" {
		specified++
	}
	if beforeMatchFlag != "" {
		specified++
	}
	if specified == 0 {
		return 0, 0, "", fmt.Errorf("destination required: use -after, -before, -aftermatch, or -beforematch")
	}
	if specified > 1 {
		return 0, 0, "", fmt.Errorf("only one destination allowed: use exactly one of -after, -before, -aftermatch, -beforematch")
	}

	switch {
	case afterFlag != -1:
		if afterFlag < 0 || afterFlag > len(lines) {
			return 0, 0, "", fmt.Errorf("-after %d out of range (file has %d lines; use 0 for beginning)", afterFlag, len(lines))
		}
		return afterFlag, afterFlag, fmt.Sprintf("after line %d", afterFlag), nil

	case beforeFlag != -1:
		if beforeFlag < 1 || beforeFlag > len(lines) {
			return 0, 0, "", fmt.Errorf("-before %d out of range (file has %d lines)", beforeFlag, len(lines))
		}
		return beforeFlag - 1, beforeFlag, fmt.Sprintf("before line %d", beforeFlag), nil

	case afterMatchFlag != "":
		hits := findMatches(lines, afterMatchFlag)
		if len(hits) == 0 {
			return 0, 0, "", fmt.Errorf("-aftermatch %q: no match found", afterMatchFlag)
		}
		ln := hits[0]
		return ln, ln, fmt.Sprintf("after line %d (%q)", ln, afterMatchFlag), nil

	default: // beforeMatchFlag != ""
		hits := findMatches(lines, beforeMatchFlag)
		if len(hits) == 0 {
			return 0, 0, "", fmt.Errorf("-beforematch %q: no match found", beforeMatchFlag)
		}
		ln := hits[0]
		return ln - 1, ln, fmt.Sprintf("before line %d (%q)", ln, beforeMatchFlag), nil
	}
}

// execMove performs the core move logic atomically (pure function, no I/O).
// Returns the result slice and the 1-based line number of the first pasted line (for verify).
//
// Overlap rule: if destLine is inside [srcStart, srcEnd] (inclusive) the operation
// is rejected with a descriptive error. Boundaries count as overlap.
//
// times > 1: source is removed once; N copies are pasted at destination ("cut once, paste N times").
// Net line delta = blockSize * (times - 1).
func execMove(lines []string, srcStart, srcEnd, destAfter, destLine, times int) ([]string, int, error) {
	if times < 1 {
		return nil, 0, fmt.Errorf("-times must be >= 1 (use delete to remove a block)")
	}
	if srcStart < 1 || srcEnd > len(lines) || srcStart > srcEnd {
		return nil, 0, fmt.Errorf("source range %d-%d invalid (file has %d lines)", srcStart, srcEnd, len(lines))
	}
	if destLine >= srcStart && destLine <= srcEnd {
		return nil, 0, fmt.Errorf("destination line %d is inside source range %d-%d", destLine, srcStart, srcEnd)
	}

	blockSize := srcEnd - srcStart + 1
	block := make([]string, blockSize)
	copy(block, lines[srcStart-1:srcEnd])

	// Build file with source block removed.
	reduced := make([]string, 0, len(lines)-blockSize)
	reduced = append(reduced, lines[:srcStart-1]...)
	reduced = append(reduced, lines[srcEnd:]...)

	// Adjust destination for the removed block.
	adjustedDest := destAfter
	if destAfter >= srcStart {
		// destAfter > srcEnd is guaranteed by the overlap check above.
		adjustedDest = destAfter - blockSize
	}

	// Paste N copies.
	result := make([]string, 0, len(reduced)+blockSize*times)
	result = append(result, reduced[:adjustedDest]...)
	for i := 0; i < times; i++ {
		result = append(result, block...)
	}
	result = append(result, reduced[adjustedDest:]...)

	firstDestLine := adjustedDest + 1 // 1-based line where first copy lands
	return result, firstDestLine, nil
}

// execCopy performs the core copy logic atomically (pure function, no I/O).
// Returns the result slice and the 1-based line number of the first copy (for verify).
//
// Snapshot semantics: the source block is read ONCE before any writes.
// All N copies are identical clones of the original block even when the
// destination overlaps the source range.
//
// Net line delta = blockSize * times (original lines are preserved).
func execCopy(lines []string, srcStart, srcEnd, destAfter, times int) ([]string, int, error) {
	if times < 1 {
		return nil, 0, fmt.Errorf("-times must be >= 1")
	}
	if srcStart < 1 || srcEnd > len(lines) || srcStart > srcEnd {
		return nil, 0, fmt.Errorf("source range %d-%d invalid (file has %d lines)", srcStart, srcEnd, len(lines))
	}

	blockSize := srcEnd - srcStart + 1
	// Snapshot: read before any modifications.
	block := make([]string, blockSize)
	copy(block, lines[srcStart-1:srcEnd])

	// Original lines stay; insert N copies at destAfter.
	result := make([]string, 0, len(lines)+blockSize*times)
	result = append(result, lines[:destAfter]...)
	for i := 0; i < times; i++ {
		result = append(result, block...)
	}
	result = append(result, lines[destAfter:]...)

	firstDestLine := destAfter + 1
	return result, firstDestLine, nil
}

func printMoveCopyStats(op, path, destDesc string, srcStart, srcEnd, times, linesBefore, linesAfter int, startTime time.Time) {
	elapsed := time.Since(startTime)
	var elapsedStr string
	if elapsed < time.Millisecond {
		elapsedStr = "<1ms"
	} else {
		elapsedStr = elapsed.Round(time.Millisecond).String()
	}
	blockSize := srcEnd - srcStart + 1
	delta := linesAfter - linesBefore
	fmt.Fprintf(os.Stderr, "\n=== STATS ===\n")
	fmt.Fprintf(os.Stderr, "  op:      %s\n", op)
	fmt.Fprintf(os.Stderr, "  file:    %s\n", path)
	fmt.Fprintf(os.Stderr, "  src:     lines %d-%d (%d lines)\n", srcStart, srcEnd, blockSize)
	fmt.Fprintf(os.Stderr, "  dest:    %s\n", destDesc)
	if times > 1 {
		fmt.Fprintf(os.Stderr, "  times:   %d\n", times)
	}
	if delta > 0 {
		fmt.Fprintf(os.Stderr, "  lines:   +%d (%d -> %d)\n", delta, linesBefore, linesAfter)
	} else if delta < 0 {
		fmt.Fprintf(os.Stderr, "  lines:   %d (%d -> %d)\n", delta, linesBefore, linesAfter)
	} else {
		fmt.Fprintf(os.Stderr, "  lines:   0 (unchanged, %d total)\n", linesBefore)
	}
	fmt.Fprintf(os.Stderr, "  elapsed: %s\n", elapsedStr)
}

func doMoveOp(lines []string, path string, lineFlag, endFlag int, matchFlag, endmatchFlag string,
	afterFlag, beforeFlag int, afterMatchFlag, beforeMatchFlag string,
	nth, times, linesBefore int, startTime time.Time, verify bool) {

	srcStart, srcEnd, err := resolveSourceLines(lines, lineFlag, endFlag, matchFlag, endmatchFlag, nth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	destAfter, destLine, destDesc, err := resolveDestLine(lines, afterFlag, beforeFlag, afterMatchFlag, beforeMatchFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result, firstDest, err := execMove(lines, srcStart, srcEnd, destAfter, destLine, times)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	if times == 1 {
		fmt.Fprintf(os.Stderr, "Moved lines %d-%d %s (%d total now)\n", srcStart, srcEnd, destDesc, len(result))
	} else {
		fmt.Fprintf(os.Stderr, "Moved lines %d-%d %s x%d (%d total now)\n", srcStart, srcEnd, destDesc, times, len(result))
	}

	if verify {
		showVerify(path, firstDest)
		printMoveCopyStats("move", path, destDesc, srcStart, srcEnd, times, linesBefore, len(result), startTime)
	}
}

func doCopyOp(lines []string, path string, lineFlag, endFlag int, matchFlag, endmatchFlag string,
	afterFlag, beforeFlag int, afterMatchFlag, beforeMatchFlag string,
	nth, times, linesBefore int, startTime time.Time, verify bool) {

	srcStart, srcEnd, err := resolveSourceLines(lines, lineFlag, endFlag, matchFlag, endmatchFlag, nth)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	destAfter, _, destDesc, err := resolveDestLine(lines, afterFlag, beforeFlag, afterMatchFlag, beforeMatchFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	result, firstDest, err := execCopy(lines, srcStart, srcEnd, destAfter, times)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	if times == 1 {
		fmt.Fprintf(os.Stderr, "Copied lines %d-%d %s (%d total now)\n", srcStart, srcEnd, destDesc, len(result))
	} else {
		fmt.Fprintf(os.Stderr, "Copied lines %d-%d %s x%d (%d total now)\n", srcStart, srcEnd, destDesc, times, len(result))
	}

	if verify {
		showVerify(path, firstDest)
		printMoveCopyStats("copy", path, destDesc, srcStart, srcEnd, times, linesBefore, len(result), startTime)
	}
}

// ════════════════════════════════════════════════════════════
// BLOCK RESOLUTION — mapper-aware source/dest for -block (v1.2.1)
// ════════════════════════════════════════════════════════════

// blockEntry represents one top-level structural element found by a language scanner.
type blockEntry struct {
	name  string // full display name, e.g. "class Foo" or "func Bar"
	key   string // identifier only, e.g. "Foo" or "Bar"
	start int    // 1-based start line
	end   int    // 1-based end line (inclusive)
}

// resolveBlock finds a named top-level block and returns its (start, end) line range.
// Error when: lang unsupported, no match, or multiple matches (ambiguous).
func resolveBlock(lines []string, lang, name string) (start, end int, err error) {
	if lang == "" {
		return 0, 0, fmt.Errorf("-block requires -lang")
	}
	entries, err := getTopLevelBlocks(lines, lang)
	if err != nil {
		return 0, 0, err
	}
	var matches []blockEntry
	for _, e := range entries {
		if strings.Contains(e.name, name) || strings.Contains(e.key, name) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 0 {
		return 0, 0, fmt.Errorf("-block %q: no match found in %s structure (%d top-level blocks scanned)",
			name, lang, len(entries))
	}
	if len(matches) > 1 {
		lines2 := make([]string, len(matches))
		for i, m := range matches {
			lines2[i] = fmt.Sprintf("  line %d: %s", m.start, m.name)
		}
		return 0, 0, fmt.Errorf("-block %q matched %d blocks — use a more specific name:\n%s",
			name, len(matches), strings.Join(lines2, "\n"))
	}
	return matches[0].start, matches[0].end, nil
}

// getTopLevelBlocks routes to the correct language scanner.
func getTopLevelBlocks(lines []string, lang string) ([]blockEntry, error) {
	switch strings.ToLower(lang) {
	case "go":
		return getGoBlocks(lines), nil
	case "python":
		return getPythonBlocks(lines), nil
	case "javascript", "typescript":
		return getJSBlocks(lines), nil
	case "rust":
		return getRustBlocks(lines), nil
	case "java":
		return getJavaBlocks(lines), nil
	case "c#", "csharp":
		return getCSharpBlocks(lines), nil
	case "ruby":
		return getRubyBlocks(lines), nil
	case "php":
		return getPHPBlocks(lines), nil
	default:
		return nil, fmt.Errorf("language %q does not yet support -block (use -match/-endmatch instead; block support for this language is planned for v1.3.0)", lang)
	}
}

// ── Go ────────────────────────────────────────────────────────────────────────

func getGoBlocks(lines []string) []blockEntry {
	reFn := regexp.MustCompile(`^func\s+(\(.*?\)\s*)?(\w+)\s*[\(\[]`)
	reType := regexp.MustCompile(`^type\s+(\w+)\s+`)
	reVar := regexp.MustCompile(`^(?:var|const)\s+(\w+)\b`)

	var entries []blockEntry
	inBlock := false
	depth := 0
	curName, curKey := "", ""
	curStart := 0

	for i, line := range lines {
		ln := i + 1
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "//") {
			continue
		}

		if !inBlock {
			if m := reFn.FindStringSubmatch(line); m != nil {
				recv := strings.TrimSpace(m[1])
				name := m[2]
				if recv != "" {
					curName = "func (" + recv + ") " + name
				} else {
					curName = "func " + name
				}
				curKey = name
				curStart = ln
				inBlock = true
				depth = 0
			} else if m := reType.FindStringSubmatch(line); m != nil {
				curKey = m[1]
				curName = "type " + m[1]
				curStart = ln
				inBlock = true
				depth = 0
			} else if m := reVar.FindStringSubmatch(line); m != nil {
				curKey = m[1]
				curName = "var " + m[1]
				curStart = ln
				inBlock = true
				depth = 0
			}
		}

		if inBlock {
			depth += strings.Count(t, "{") - strings.Count(t, "}")
			if depth <= 0 && ln > curStart {
				entries = append(entries, blockEntry{curName, curKey, curStart, ln})
				inBlock = false
				curName, curKey = "", ""
			}
		}
	}
	if inBlock && curStart > 0 {
		entries = append(entries, blockEntry{curName, curKey, curStart, len(lines)})
	}
	return entries
}

// ── Python ────────────────────────────────────────────────────────────────────

func getPythonBlocks(lines []string) []blockEntry {
	reClass := regexp.MustCompile(`^class\s+(\w+)`)
	reDef := regexp.MustCompile(`^def\s+(\w+)\s*\(`)

	type start struct {
		name, key string
		line      int
	}
	var starts []start

	for i, line := range lines {
		// Top-level = first char is not whitespace (and not blank/comment/decorator)
		if len(line) == 0 || line[0] == ' ' || line[0] == '\t' || line[0] == '#' || line[0] == '@' {
			continue
		}
		t := strings.TrimSpace(line)
		if m := reClass.FindStringSubmatch(t); m != nil {
			starts = append(starts, start{"class " + m[1], m[1], i + 1})
		} else if m := reDef.FindStringSubmatch(t); m != nil {
			starts = append(starts, start{"def " + m[1], m[1], i + 1})
		}
	}

	entries := make([]blockEntry, len(starts))
	for i, s := range starts {
		endLine := len(lines)
		if i+1 < len(starts) {
			// Find the last non-blank line before the next block's decorator or definition
			endLine = starts[i+1].line - 1
			for endLine > s.line && strings.TrimSpace(lines[endLine-1]) == "" {
				endLine--
			}
		}
		entries[i] = blockEntry{s.name, s.key, s.line, endLine}
	}
	return entries
}

// ── JavaScript / TypeScript ───────────────────────────────────────────────────

func getJSBlocks(lines []string) []blockEntry {
	reClass := regexp.MustCompile(`^(?:export\s+(?:default\s+)?)?class\s+(\w+)`)
	reFunc := regexp.MustCompile(`^(?:export\s+)?(?:async\s+)?function\s+(\w+)`)
	reArrow := regexp.MustCompile(`^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?\(`)

	return braceTrackedBlocks(lines, func(line string) (name, key string, ok bool) {
		t := strings.TrimSpace(line)
		if m := reClass.FindStringSubmatch(t); m != nil {
			return "class " + m[1], m[1], true
		}
		if m := reFunc.FindStringSubmatch(t); m != nil {
			return "function " + m[1], m[1], true
		}
		if m := reArrow.FindStringSubmatch(t); m != nil {
			return "const " + m[1], m[1], true
		}
		return "", "", false
	})
}

// ── Rust ──────────────────────────────────────────────────────────────────────

func getRustBlocks(lines []string) []blockEntry {
	re := regexp.MustCompile(`^(?:pub\s+)?(?:async\s+)?(?:fn|struct|enum|trait|impl|mod)\s+(\w+)`)
	return braceTrackedBlocks(lines, func(line string) (name, key string, ok bool) {
		t := strings.TrimSpace(line)
		if m := re.FindStringSubmatch(t); m != nil {
			keyword := strings.Fields(t)[0]
			if keyword == "pub" || keyword == "async" {
				keyword = strings.Fields(t)[1]
			}
			return keyword + " " + m[1], m[1], true
		}
		return "", "", false
	})
}

// ── Java ──────────────────────────────────────────────────────────────────────

func getJavaBlocks(lines []string) []blockEntry {
	re := regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+|static\s+|abstract\s+|final\s+)*(?:class|interface|enum|record)\s+(\w+)`)
	return braceTrackedBlocks(lines, func(line string) (name, key string, ok bool) {
		t := strings.TrimSpace(line)
		if m := re.FindStringSubmatch(t); m != nil {
			return "class " + m[1], m[1], true
		}
		return "", "", false
	})
}

// ── C# ────────────────────────────────────────────────────────────────────────

func getCSharpBlocks(lines []string) []blockEntry {
	re := regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+|internal\s+|static\s+|abstract\s+|sealed\s+|partial\s+)*(?:class|interface|struct|enum|record)\s+(\w+)`)
	return braceTrackedBlocks(lines, func(line string) (name, key string, ok bool) {
		t := strings.TrimSpace(line)
		if m := re.FindStringSubmatch(t); m != nil {
			return "class " + m[1], m[1], true
		}
		return "", "", false
	})
}

// ── Ruby ──────────────────────────────────────────────────────────────────────

func getRubyBlocks(lines []string) []blockEntry {
	reClass := regexp.MustCompile(`^class\s+(\w+)`)
	reDef := regexp.MustCompile(`^def\s+(\w+)`)
	reModule := regexp.MustCompile(`^module\s+(\w+)`)

	blockKws := regexp.MustCompile(`\b(?:class|module|def|do|if|unless|while|until|for|begin|case)\b`)
	endKw := regexp.MustCompile(`\bend\b`)

	var entries []blockEntry
	inBlock := false
	depth := 0
	curName, curKey := "", ""
	curStart := 0

	for i, line := range lines {
		ln := i + 1
		t := strings.TrimSpace(line)
		if !inBlock {
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				if m := reClass.FindStringSubmatch(t); m != nil {
					curName, curKey, curStart, inBlock, depth = "class "+m[1], m[1], ln, true, 1
					continue
				}
				if m := reModule.FindStringSubmatch(t); m != nil {
					curName, curKey, curStart, inBlock, depth = "module "+m[1], m[1], ln, true, 1
					continue
				}
				if m := reDef.FindStringSubmatch(t); m != nil {
					curName, curKey, curStart, inBlock, depth = "def "+m[1], m[1], ln, true, 1
					continue
				}
			}
		}
		if inBlock {
			depth += len(blockKws.FindAllString(t, -1))
			depth -= len(endKw.FindAllString(t, -1))
			if depth <= 0 {
				entries = append(entries, blockEntry{curName, curKey, curStart, ln})
				inBlock = false
				curName, curKey = "", ""
			}
		}
	}
	if inBlock && curStart > 0 {
		entries = append(entries, blockEntry{curName, curKey, curStart, len(lines)})
	}
	return entries
}

// ── PHP ───────────────────────────────────────────────────────────────────────

func getPHPBlocks(lines []string) []blockEntry {
	reClass := regexp.MustCompile(`^(?:abstract\s+|final\s+)?class\s+(\w+)`)
	reFunc := regexp.MustCompile(`^(?:public\s+|private\s+|protected\s+|static\s+)*function\s+(\w+)`)
	return braceTrackedBlocks(lines, func(line string) (name, key string, ok bool) {
		t := strings.TrimSpace(line)
		if m := reClass.FindStringSubmatch(t); m != nil {
			return "class " + m[1], m[1], true
		}
		if m := reFunc.FindStringSubmatch(t); m != nil {
			return "function " + m[1], m[1], true
		}
		return "", "", false
	})
}

// ── shared brace-tracker ──────────────────────────────────────────────────────

// braceTrackedBlocks is a generic brace-depth scanner used by JS, Rust, Java, C#, PHP.
// matchFn is called on each line; return (name, key, true) to start a new block.
func braceTrackedBlocks(lines []string, matchFn func(string) (string, string, bool)) []blockEntry {
	var entries []blockEntry
	inBlock := false
	depth := 0
	curName, curKey := "", ""
	curStart := 0

	for i, line := range lines {
		ln := i + 1
		t := strings.TrimSpace(line)

		if !inBlock {
			if name, key, ok := matchFn(line); ok {
				curName, curKey, curStart = name, key, ln
				inBlock = true
				depth = 0
			}
		}

		if inBlock {
			depth += strings.Count(t, "{") - strings.Count(t, "}")
			if depth <= 0 && ln > curStart {
				entries = append(entries, blockEntry{curName, curKey, curStart, ln})
				inBlock = false
				curName, curKey = "", ""
			}
		}
	}
	if inBlock && curStart > 0 {
		entries = append(entries, blockEntry{curName, curKey, curStart, len(lines)})
	}
	return entries
}

// ════════════════════════════════════════════════════════════
// REGEX REPLACEALL + GLOB MULTI-FILE — v1.3.0
// ════════════════════════════════════════════════════════════

// execReplaceAllRegex is the pure testable core for -match-regex mode.
// Returns (result lines, replacement count, error).
// Go regex uses $1/$2 for capture group references in replacement strings.
func execReplaceAllRegex(lines []string, pattern, replacement string) ([]string, int, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, 0, fmt.Errorf("invalid regex %q: %v", pattern, err)
	}
	count := 0
	result := make([]string, len(lines))
	for i, line := range lines {
		newLine := re.ReplaceAllString(line, replacement)
		result[i] = newLine
		if newLine != line {
			count++
		}
	}
	return result, count, nil
}

func doReplaceAllRegex(lines []string, path, pattern, replacement string) {
	result, count, err := execReplaceAllRegex(lines, pattern, replacement)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if count == 0 {
		fmt.Fprintf(os.Stderr, "No lines match regex: %s\n", pattern)
		os.Exit(1)
	}
	if err := writeLines(path, result); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Regex-replaced on %d line(s) (%d total)\n", count, len(result))
}

// doReplaceAllGlob applies a literal replaceall to every file matching a glob pattern.
func doReplaceAllGlob(globPattern, search, replacement string) {
	paths, err := filepath.Glob(globPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Glob error: %v\n", err)
		os.Exit(1)
	}
	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "No files matched: %s\n", globPattern)
		os.Exit(1)
	}
	totalFiles, totalLines := 0, 0
	for _, p := range paths {
		lines, err := readLines(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  SKIP %-40s %v\n", p, err)
			continue
		}
		count := 0
		for i, line := range lines {
			if strings.Contains(line, search) {
				lines[i] = strings.ReplaceAll(line, search, replacement)
				count++
			}
		}
		if count == 0 {
			continue
		}
		if err := writeLines(p, lines); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %-40s %v\n", p, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %-50s %d line(s)\n", p, count)
		totalFiles++
		totalLines += count
	}
	fmt.Fprintf(os.Stderr, "Glob %s: %d file(s) changed, %d total line(s) replaced\n",
		globPattern, totalFiles, totalLines)
}

// doReplaceAllRegexGlob applies a regex replaceall to every file matching a glob pattern.
func doReplaceAllRegexGlob(globPattern, pattern, replacement string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid regex %q: %v\n", pattern, err)
		os.Exit(1)
	}
	paths, err := filepath.Glob(globPattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Glob error: %v\n", err)
		os.Exit(1)
	}
	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "No files matched: %s\n", globPattern)
		os.Exit(1)
	}
	totalFiles, totalLines := 0, 0
	for _, p := range paths {
		lines, err := readLines(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  SKIP %-40s %v\n", p, err)
			continue
		}
		count := 0
		for i, line := range lines {
			newLine := re.ReplaceAllString(line, replacement)
			if newLine != line {
				lines[i] = newLine
				count++
			}
		}
		if count == 0 {
			continue
		}
		if err := writeLines(p, lines); err != nil {
			fmt.Fprintf(os.Stderr, "  ERROR %-40s %v\n", p, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "  %-50s %d line(s)\n", p, count)
		totalFiles++
		totalLines += count
	}
	fmt.Fprintf(os.Stderr, "Glob %s: %d file(s) changed, %d total line(s) replaced\n",
		globPattern, totalFiles, totalLines)
}
