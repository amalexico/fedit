package main

import (
	"bufio"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	op := flag.String("op", "", "Operation: insert, delete, replace, replaceall, show, write, writeraw, writelines, map, find, insertafter, insertbefore, move, copy, fields")
	line := flag.String("line", "", "Line number (1-based)")
	endLine := flag.String("end", "", "End line for delete/replace range (inclusive)")
	text := flag.String("text", "", "Text to insert/replace (use \\n for newlines, \\t for tabs)")
	textFile := flag.String("textfile", "", "Read insert/replace text from this file instead of -text")
	lang := flag.String("lang", "", "Language for map: go, html, sql, python, javascript, typescript, css, rust, java, csharp, yaml, toml, markdown, ruby, php, dockerfile, makefile, json, csv")
	match := flag.String("match", "", "Text to search for (find/insertafter/insertbefore)")
	nth := flag.Int("nth", 1, "Which occurrence to match (default 1, use -1 for last)")
	v := flag.Bool("v", false, "Verify: show affected lines after mutation")
	endmatch := flag.String("endmatch", "", "End line bound for move/copy source (content-based)")
	after := flag.Int("after", -1, "Destination: insert after line N (0 = beginning of file)")
	before := flag.Int("before", -1, "Destination: insert before line N")
	aftermatch := flag.String("aftermatch", "", "Destination: insert after first line matching TEXT")
	beforematch := flag.String("beforematch", "", "Destination: insert before first line matching TEXT")
	times := flag.Int("times", 1, "Repeat N times (default 1, min 1; for move: cut once, paste N times)")
	block := flag.String("block", "", "Source: auto-derive range from named block (requires -lang)")
	beforeblock := flag.String("beforeblock", "", "Destination: insert before named block (requires -lang)")
	afterblock := flag.String("afterblock", "", "Destination: insert after named block (requires -lang)")
	matchRegex := flag.String("match-regex", "", "Regex pattern for replaceall with capture groups (e.g. -match-regex \"(\\w+)\" -text \"[$1]\")")
	files := flag.String("files", "", "Apply replaceall to all files matching a glob (e.g. -files \"*.go\")")
	stream := flag.Bool("stream", false, "Stream mode: line-by-line I/O for large files (replaceall, find)")
	col := flag.Int("col", 0, "Column number (1-based) for fields op")
	delim := flag.String("delim", "\t", "Field delimiter for fields op (default: tab)")
	texthex := flag.String("texthex", "", "Content as UTF-8 hex (fwencode output); bypasses -text and all escape expansion")
	cleanfirst := flag.Bool("cleanfirst", false, "Truncate -file to zero bytes before writing")
	x := flag.Bool("x", false, "Machine-readable output: bare line numbers / counts, no labels")
	extractFlag := flag.String("extract", "", "Extract from matched line: WN  WN[s:c]  WN[s:]  WN/DELIM/F")
	get := flag.String("get", "", "Regex pre-filter: extract matching token from line before -extract applies")
	wdelim := flag.String("wdelim", "", "Word delimiter for -extract (default: normalized whitespace, awk-style)")
	raw := flag.Bool("raw", false, "Raw show output: no line numbers or prefix (for piping to fwencode)")
	quiet := flag.Bool("quiet", false, "Suppress stdout on success; exit code signals result (-quiet wins over -v)")
	flag.Parse()

	// -texthex: hex string is the content itself (produced by fwencode or PS hex encode).
	// Decoded bytes bypass expandText entirely — no escape expansion, no backslash mangling.
	// resolvedBytes is nil when -texthex is absent; all downstream code checks nil before use.
	var resolvedBytes []byte
	if *texthex != "" {
		var hErr error
		resolvedBytes, hErr = hex.DecodeString(*texthex)
		if hErr != nil {
			fmt.Fprintf(os.Stderr, "texthex: invalid hex string: %v\n", hErr)
			os.Exit(1)
		}
	}

	if *quiet {
		*v = false
	}
	if (*file == "" && *files == "") || *op == "" {
		fmt.Fprintln(os.Stderr, "Usage: fedit -file PATH -op OPERATION [flags]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Operations:")
		fmt.Fprintln(os.Stderr, "  show          Show file with line numbers (-line/-end to limit range)")
		fmt.Fprintln(os.Stderr, "  insert        Insert text AFTER line N (use -line 0 for top)")
		fmt.Fprintln(os.Stderr, "  delete        Delete line N (or range N to -end)")
		fmt.Fprintln(os.Stderr, "  replace       Replace line N (or range N to -end) with text")
		fmt.Fprintln(os.Stderr, "  write         Write text to -file (creates/overwrites)")
		fmt.Fprintln(os.Stderr, "  writeraw      Write -text raw (no escape expansion; backslashes are literal)")
		fmt.Fprintln(os.Stderr, "  writelines    Write lines interactively from stdin (Ctrl+Z/D to finish)")
		fmt.Fprintln(os.Stderr, "  map           Show structure map (-lang go|html|sql)")
		fmt.Fprintln(os.Stderr, "  find          Find lines containing -match text, print line numbers")
		fmt.Fprintln(os.Stderr, "  insertafter   Find -match text, insert -textfile content AFTER matched line")
		fmt.Fprintln(os.Stderr, "  insertbefore  Find -match text, insert -textfile content BEFORE matched line")
		fmt.Fprintln(os.Stderr, "  replaceall    Replace ALL occurrences of -match text with -text (-match-regex for regex, -files for glob)")
		fmt.Fprintln(os.Stderr, "  move          Move a line range (-line/-end or -match/-endmatch + -after/-before/-aftermatch/-beforematch)")
		fmt.Fprintln(os.Stderr, "  copy          Copy a line range (same flags as move; -times N for N copies)")
		fmt.Fprintln(os.Stderr, "  fields        Extract column N from delimited file (-col N -delim CHAR, outputs to stdout)")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Match flags (for find/insertafter/insertbefore):")
		fmt.Fprintln(os.Stderr, "  -match TEXT   Substring to search for (required)")
		fmt.Fprintln(os.Stderr, "  -nth N        Which occurrence (default 1, -1 for last)")
		fmt.Fprintln(os.Stderr, "  -stream       Line-by-line I/O for large files (replaceall, find)")
		fmt.Fprintln(os.Stderr, "  -col N        Column for fields op (1-based)")
		fmt.Fprintln(os.Stderr, "  -delim CHAR   Field delimiter (default: tab)")
		fmt.Fprintln(os.Stderr, "  -block TEXT   Source block name for move/copy (requires -lang)")
		fmt.Fprintln(os.Stderr, "  -match-regex  Regex for replaceall with capture groups")
		fmt.Fprintln(os.Stderr, "  -files GLOB   Apply replaceall to all files matching a glob")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  -texthex      Treat -text as hex from fwencode (bypasses PS quoting entirely)")
		fmt.Fprintln(os.Stderr, "  -cleanfirst   Truncate -file before writing (pair with insert for clean overwrite)")
		fmt.Fprintln(os.Stderr, "  -x            Machine-readable output: bare line numbers / counts, no labels")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Escapes in -text: \\n = newline, \\t = tab, \\\\ = literal backslash")
		os.Exit(1)
	}

	if *op == "write" || *op == "writeraw" || *op == "writelines" {
		var content []string
		switch *op {
		case "writelines":
			scanner := bufio.NewScanner(os.Stdin)
			for {
				fmt.Fprint(os.Stderr, "> ")
				if !scanner.Scan() {
					break
				}
				content = append(content, scanner.Text())
			}
		case "writeraw":
			if resolvedBytes != nil {
				content = bytesToLines(resolvedBytes)
			} else if *textFile != "" {
				var err error
				content, err = readLines(*textFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
					os.Exit(1)
				}
			} else {
				// raw: split on actual newlines only — no \n->newline expansion
				content = strings.Split(*text, "\n")
			}
		default: // "write"
			if resolvedBytes != nil {
				content = bytesToLines(resolvedBytes)
			} else if *textFile != "" {
				var err error
				content, err = readLines(*textFile)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error reading textfile: %v\n", err)
					os.Exit(1)
				}
			} else {
				content = expandText(*text)
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

	// -cleanfirst: truncate before reading so mutations start from an empty file.
	if *cleanfirst && *file != "" {
		if err := os.WriteFile(*file, []byte{}, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error truncating file: %v\n", err)
			os.Exit(1)
		}
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

	lineN, endN, lineErr := parseAndResolveLines(*line, *endLine, len(lines))
	if lineErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", lineErr)
		os.Exit(1)
	}

	linesBefore := len(lines)
	startTime := time.Now()

	switch *op {
	case "show":
		showStart, showEnd := lineN, endN
		if *block != "" {
			var bErr error
			showStart, showEnd, bErr = resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
		} else if *match != "" {
			var mErr error
			showStart, showEnd, mErr = resolveSourceLines(lines, 0, 0, *match, *endmatch, *nth)
			if mErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", mErr)
				os.Exit(1)
			}
		}
		doShow(lines, showStart, showEnd, *raw)
	case "insert":
		newText := resolveTextFull(*text, *textFile, resolvedBytes)
		doInsert(lines, *file, lineN, newText)
	case "delete":
		delStart, delEnd := lineN, endN
		if *match != "" {
			var mErr error
			delStart, delEnd, mErr = resolveSourceLines(lines, 0, 0, *match, *endmatch, *nth)
			if mErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", mErr)
				os.Exit(1)
			}
		}
		doDelete(lines, *file, delStart, delEnd)
	case "replace":
		newText := resolveTextFull(*text, *textFile, resolvedBytes)
		replStart, replEnd := lineN, endN
		if *block != "" {
			var bErr error
			replStart, replEnd, bErr = resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
		} else if *match != "" {
			var mErr error
			replStart, replEnd, mErr = resolveSourceLines(lines, 0, 0, *match, *endmatch, *nth)
			if mErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", mErr)
				os.Exit(1)
			}
		}
		doReplace(lines, *file, replStart, replEnd, newText)
	case "map":
		doMap(lines, *file, *lang)
	case "find":
		if *stream {
			doStreamFind(*file, *match, *x, *get, *extractFlag, *wdelim)
		} else {
			doFind(lines, *match, *nth, *x, *get, *extractFlag, *wdelim)
		}
	case "insertafter":
		newText := resolveTextFull(*text, *textFile, resolvedBytes)
		if *block != "" {
			_, blockEnd, bErr := resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			doInsert(lines, *file, blockEnd, newText) // after closing line of block
		} else {
			doInsertMatch(lines, *file, *match, *nth, newText, false)
		}
	case "insertbefore":
		newText := resolveTextFull(*text, *textFile, resolvedBytes)
		if *block != "" {
			blockStart, _, bErr := resolveBlock(lines, *lang, *block)
			if bErr != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", bErr)
				os.Exit(1)
			}
			doInsert(lines, *file, blockStart-1, newText) // before opening line of block
		} else {
			doInsertMatch(lines, *file, *match, *nth, newText, true)
		}
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
		if resolvedBytes != nil {
			replacement = string(resolvedBytes)
		} else if *textFile != "" {
			rLines := resolveText("", *textFile)
			replacement = strings.Join(rLines, "\n")
		}
		if *files != "" {
			if isRegex {
				doReplaceAllRegexGlob(*files, searchStr, replacement)
			} else {
				doReplaceAllGlob(*files, searchStr, replacement)
			}
		} else if *stream {
			if isRegex {
				doStreamReplaceAllRegex(*file, searchStr, replacement)
			} else {
				doStreamReplaceAll(*file, searchStr, replacement)
			}
		} else if isRegex {
			doReplaceAllRegex(lines, *file, searchStr, replacement)
		} else {
			doReplaceAll(lines, *file, searchStr, replacement)
		}
	case "move":
		srcL, srcE := lineN, endN
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
		srcL, srcE := lineN, endN
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
	case "fields":
		if *col == 0 {
			fmt.Fprintln(os.Stderr, "fields requires -col N (1-based column number)")
			os.Exit(1)
		}
		delimStr := *delim
		if delimStr == "\\t" {
			delimStr = "\t"
		}
		doFields(*file, *col, delimStr, *x)
	default:
		fmt.Fprintf(os.Stderr, "Unknown operation: %s\n", *op)
		os.Exit(1)
	}
	if *v {
		switch *op {
		case "show", "map", "find", "write", "writeraw", "writelines", "move", "copy":
		// read-only ops, no verify
		default:
			center := lineN
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

// ════════════════════════════════════════════════════════════
// STREAM ENGINE + FIELDS — v1.4.0
// ════════════════════════════════════════════════════════════

// streamLineBuffer is the per-line read buffer for the stream engine.
// 10 MB handles JSON blobs, base64 payloads, and minified JS.
const streamLineBuffer = 10 * 1024 * 1024

// execStreamOp processes a file line-by-line using fn, writing results to
// a temp file, then atomically replacing the original on success.
//
// fn receives (1-based lineNum, raw line text) and returns the lines that
// should appear in the output for that input line:
//
//	[]string{line}         -- pass-through (unchanged)
//	[]string{newLine}      -- replace
//	[]string{}             -- delete
//	[]string{extra, line}  -- insertbefore
//	[]string{line, extra}  -- insertafter
//
// Returns (linesRead, linesWritten, error).
func execStreamOp(path string, fn func(lineNum int, line string) []string) (int, int, error) {
	src, err := os.Open(path)
	if err != nil {
		return 0, 0, fmt.Errorf("open: %v", err)
	}

	// Temp file in same directory for atomic rename.
	tmpPath := path + ".fedit_stream_tmp"
	dst, err := os.Create(tmpPath)
	if err != nil {
		src.Close()
		return 0, 0, fmt.Errorf("create temp: %v", err)
	}

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	writer := bufio.NewWriter(dst)

	linesRead, linesWritten := 0, 0
	var scanErr error

	for scanner.Scan() {
		linesRead++
		out := fn(linesRead, scanner.Text())
		for _, ol := range out {
			writer.WriteString(ol)
			writer.WriteByte('\n')
			linesWritten++
		}
	}
	scanErr = scanner.Err()

	writer.Flush()
	dst.Close()
	src.Close()

	if scanErr != nil {
		os.Remove(tmpPath)
		return linesRead, 0, fmt.Errorf("scan: %v", scanErr)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return linesRead, 0, fmt.Errorf("atomic rename: %v", err)
	}
	return linesRead, linesWritten, nil
}

// doStreamReplaceAll is the streaming path for literal replaceall (-stream).
func doStreamReplaceAll(path, search, replacement string) {
	changed := 0
	linesRead, _, err := execStreamOp(path, func(_ int, line string) []string {
		if strings.Contains(line, search) {
			changed++
			return []string{strings.ReplaceAll(line, search, replacement)}
		}
		return []string{line}
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if changed == 0 {
		fmt.Fprintf(os.Stderr, "No lines match: %s\n", search)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Stream-replaced on %d line(s) of %d total\n", changed, linesRead)
}

// doStreamReplaceAllRegex is the streaming path for regex replaceall (-stream).
func doStreamReplaceAllRegex(path, pattern, replacement string) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid regex %q: %v\n", pattern, err)
		os.Exit(1)
	}
	changed := 0
	linesRead, _, sErr := execStreamOp(path, func(_ int, line string) []string {
		newLine := re.ReplaceAllString(line, replacement)
		if newLine != line {
			changed++
		}
		return []string{newLine}
	})
	if sErr != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", sErr)
		os.Exit(1)
	}
	if changed == 0 {
		fmt.Fprintf(os.Stderr, "No lines match regex: %s\n", pattern)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Stream regex-replaced on %d line(s) of %d total\n", changed, linesRead)
}

// doStreamFind is the streaming path for find (-stream). Outputs to stdout.
func doStreamFind(path, search string, x bool, get, extractStr, wdelim string) {
	src, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	// Pre-parse extract spec once (not per line).
	var es extractSpec
	if extractStr != "" {
		es, err = parseExtractSpec(extractStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}

	extractMode := extractStr != "" || get != ""
	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	count, lineNum, extracted, skipped := 0, 0, 0, 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if !strings.Contains(line, search) {
			continue
		}
		count++
		if extractMode {
			token := line
			if get != "" {
				filtered, ok := applyGet(token, get)
				if !ok {
					skipped++
					continue
				}
				token = filtered
			}
			if extractStr != "" {
				val, ok := applyExtract(token, es, wdelim)
				if !ok {
					skipped++
					continue
				}
				fmt.Println(val)
			} else {
				fmt.Println(token)
			}
			extracted++
		} else if x {
			fmt.Println(lineNum)
		} else {
			fmt.Printf("%d: %s\n", lineNum, line)
		}
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %v\n", scanner.Err())
		os.Exit(1)
	}
	if extractMode {
		if skipped > 0 {
			fmt.Fprintf(os.Stderr, "Extracted %d value(s), skipped %d (no match) across %d lines\n", extracted, skipped, lineNum)
		} else {
			fmt.Fprintf(os.Stderr, "Extracted %d value(s) from %d match(es) across %d lines\n", extracted, count, lineNum)
		}
	} else if !x {
		fmt.Fprintf(os.Stderr, "Found %d match(es) across %d lines\n", count, lineNum)
	}
}

// doFields extracts column col (1-based) from every line of path,
// using delim as the field separator. Output goes to stdout.
func doFields(path string, col int, delim string, x bool) {
	if col < 1 {
		fmt.Fprintln(os.Stderr, "Error: -col must be >= 1")
		os.Exit(1)
	}
	src, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer src.Close()

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, streamLineBuffer), streamLineBuffer)
	lineNum, extracted := 0, 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			fmt.Println()
			continue
		}
		parts := strings.Split(line, delim)
		if col <= len(parts) {
			fmt.Println(parts[col-1])
			extracted++
		}
		// Lines shorter than col are skipped silently.
	}
	if scanner.Err() != nil {
		fmt.Fprintf(os.Stderr, "Error reading: %v\n", scanner.Err())
		os.Exit(1)
	}
	if !x {
		fmt.Fprintf(os.Stderr, "Extracted col %d from %d/%d lines (delim=%q)\n",
			col, extracted, lineNum, delim)
	}
}

// ════════════════════════════════════════════════════════════
// IaC BLOCK SCANNERS — v1.5.0 (HCL/Terraform + Nix)
// ════════════════════════════════════════════════════════════

// getHCLBlocks extracts top-level block definitions from HCL/Terraform files.
// Handles: resource, data, module, provider, variable, output,
//
//	locals, terraform, moved, import, check.
//
// Both multi-line blocks and single-line blocks (e.g. locals {}) are supported.
func getHCLBlocks(lines []string) []blockEntry {
	reTopBlock := regexp.MustCompile(`^(\w+)\s*(.*)`)
	validKw := map[string]bool{
		"resource": true, "data": true, "module": true, "provider": true,
		"variable": true, "output": true, "locals": true, "terraform": true,
		"moved": true, "import": true, "check": true,
	}

	var entries []blockEntry
	inBlock := false
	depth := 0
	curName, curKey := "", ""
	curStart := 0

	for i, line := range lines {
		ln := i + 1
		t := strings.TrimSpace(line)

		if !inBlock {
			// Skip blank lines, comments, and indented lines (not top-level)
			if t == "" || strings.HasPrefix(t, "#") || strings.HasPrefix(t, "//") {
				continue
			}
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				continue
			}
			m := reTopBlock.FindStringSubmatch(t)
			if m == nil || !validKw[m[1]] {
				continue
			}
			keyword := m[1]
			// Strip trailing { and whitespace to get the label
			rest := strings.TrimRight(strings.TrimSpace(m[2]), " \t{}")
			// Build display name and searchable key
			// e.g. resource "aws_instance" "web" → key: aws_instance_web
			var key, name string
			if rest == "" {
				key = keyword
				name = keyword
			} else {
				key = strings.ReplaceAll(rest, `" "`, "_")
				key = strings.Trim(key, `"`)
				name = keyword + " " + rest
			}
			curName, curKey, curStart = name, key, ln
			inBlock = true
			depth = 0
		}

		if inBlock {
			depth += strings.Count(t, "{") - strings.Count(t, "}")
			// Close when balanced. Allow single-line blocks (ln == curStart + brace close).
			if depth <= 0 && (ln > curStart || strings.Contains(t, "}")) {
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

// getNixBlocks extracts top-level attribute bindings that contain block bodies
// from Nix expression files. Handles patterns like:
//
//	someAttr = {       (attribute set)
//	programs.git = {   (nested attribute)
//	buildInputs = [    (list binding — tracked by bracket depth)
//	mkDerivation = {   (derivation)
func getNixBlocks(lines []string) []blockEntry {
	reAttr := regexp.MustCompile(`^([\w.]+(?:\."[^"]+")?)\s*=`)

	var entries []blockEntry
	inBlock := false
	depth := 0     // tracks { }
	listDepth := 0 // tracks [ ]
	curName, curKey := "", ""
	curStart := 0

	for i, line := range lines {
		ln := i + 1
		t := strings.TrimSpace(line)

		if !inBlock {
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			// Only top-level lines (no leading whitespace)
			if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
				continue
			}
			m := reAttr.FindStringSubmatch(t)
			if m == nil {
				continue
			}
			// Only start a block if the line opens a { or [
			hasBrace := strings.Contains(t, "{")
			hasList := strings.Contains(t, "[")
			if !hasBrace && !hasList {
				continue
			}
			key := m[1]
			curName = key + " = {..}"
			curKey = key
			curStart = ln
			inBlock = true
			depth, listDepth = 0, 0
		}

		if inBlock {
			depth += strings.Count(t, "{") - strings.Count(t, "}")
			listDepth += strings.Count(t, "[") - strings.Count(t, "]")
			closed := depth <= 0 && listDepth <= 0
			if closed && (ln > curStart || strings.ContainsAny(t, "}]")) {
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