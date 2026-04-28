package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
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
	op := flag.String("op", "", "Operation: insert, delete, replace, replaceall, show, write, map, find, insertafter, insertbefore")
	line := flag.Int("line", 0, "Line number (1-based)")
	endLine := flag.Int("end", 0, "End line for delete/replace range (inclusive)")
	text := flag.String("text", "", "Text to insert/replace (use \\n for newlines, \\t for tabs)")
	textFile := flag.String("textfile", "", "Read insert/replace text from this file instead of -text")
	lang := flag.String("lang", "", "Language for map: go, html, sql")
	match := flag.String("match", "", "Text to search for (find/insertafter/insertbefore)")
	nth := flag.Int("nth", 1, "Which occurrence to match (default 1, use -1 for last)")
	v := flag.Bool("v", false, "Verify: show affected lines after mutation")
	flag.Parse()

	if *file == "" || *op == "" {
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

	lines, err := readLines(*file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
		os.Exit(1)
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
		if *match == "" {
			fmt.Fprintln(os.Stderr, "replaceall requires -match (text to find)")
			os.Exit(1)
		}
		replacement := *text
		if *textFile != "" {
			rLines := resolveText("", *textFile)
			replacement = strings.Join(rLines, "\n")
		}
		doReplaceAll(lines, *file, *match, replacement)
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", *op)
		os.Exit(1)
	}
	if *v {
		switch *op {
		case "show", "map", "find", "write":
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
