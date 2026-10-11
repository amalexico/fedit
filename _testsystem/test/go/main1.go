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

// resolveTextFull is resolveText extended with -texthex support.
// Priority: resolvedBytes (from -texthex) > textFile > text.
// When resolvedBytes is non-nil the content is used as-is — no escape expansion,
// no backslash processing. This is intentional: hex-encoded bytes are already exact.
func resolveTextFull(text, textFile string, resolvedBytes []byte) []string {
	if resolvedBytes != nil {
		return bytesToLines(resolvedBytes)
	}
	return resolveText(text, textFile)
}

// bytesToLines splits raw bytes on newline and strips the trailing empty element
// that strings.Split produces for content ending with '\n'. This matches the
// behaviour of readLines (scanner-based) so callers get consistent line slices.
// parseLineFlag parses a -line flag value into raw components.
// Supports: "N", "-N", "N:+M", "-N:+M", "-N:", ":"
func parseLineFlag(s string) (startRaw, relEnd int, hasColon, endIsEOF bool, err error) {
	if s == "" {
		return 0, 0, false, false, nil
	}
	if s == ":" {
		return 0, 0, true, true, nil
	}
	if idx := strings.Index(s, ":"); idx >= 0 {
		left, right := s[:idx], s[idx+1:]
		startRaw, err = strconv.Atoi(left)
		if err != nil {
			return 0, 0, false, false, fmt.Errorf("invalid -line %q: %v", s, err)
		}
		if right == "" {
			return startRaw, 0, true, true, nil
		}
		if !strings.HasPrefix(right, "+") {
			return 0, 0, false, false, fmt.Errorf("invalid -line range %q: expected +N after colon", s)
		}
		relEnd, err = strconv.Atoi(right[1:])
		if err != nil {
			return 0, 0, false, false, fmt.Errorf("invalid -line range %q: %v", s, err)
		}
		return startRaw, relEnd, true, false, nil
	}
	startRaw, err = strconv.Atoi(s)
	if err != nil {
		return 0, 0, false, false, fmt.Errorf("invalid -line %q: %v", s, err)
	}
	return startRaw, 0, false, false, nil
}

// parseEndFlag parses a -end flag value: "N" or "-N"
func parseEndFlag(s string) (raw int, err error) {
	if s == "" {
		return 0, nil
	}
	raw, err = strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid -end %q: %v", s, err)
	}
	return raw, nil
}

