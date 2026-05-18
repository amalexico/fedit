package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type mcpInitResult struct {
	ProtocolVersion string        `json:"protocolVersion"`
	Capabilities    mcpCaps       `json:"capabilities"`
	ServerInfo      mcpServerInfo `json:"serverInfo"`
}

type mcpCaps struct {
	Tools map[string]any `json:"tools,omitempty"`
}

type mcpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type mcpToolsResult struct {
	Tools []mcpToolDef `json:"tools"`
}

type mcpToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type mcpCallParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

type mcpCallResult struct {
	Content []mcpContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

type mcpContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func runMCP() {
	reader := bufio.NewReader(os.Stdin)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "MCP read error: %v\n", err)
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			mcpSendError(nil, -32700, "Parse error")
			continue
		}
		switch req.Method {
		case "initialize":
			mcpSendResult(req.ID, mcpInitResult{
				ProtocolVersion: "2024-11-05",
				Capabilities:    mcpCaps{Tools: map[string]any{}},
				ServerInfo:      mcpServerInfo{Name: "fedit", Version: "1.5.0"},
			})
		case "notifications/initialized":
		// no response needed
		case "tools/list":
			mcpSendResult(req.ID, mcpToolsResult{Tools: mcpToolDefs()})
		case "tools/call":
			var params mcpCallParams
			if err := json.Unmarshal(req.Params, &params); err != nil {
				mcpSendError(req.ID, -32602, "Invalid params")
				continue
			}
			result := mcpExecTool(params.Name, params.Arguments)
			mcpSendResult(req.ID, result)
		default:
			mcpSendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func mcpSendResult(id json.RawMessage, result any) {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Result: result}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(os.Stdout, "%s\n", data)
}

func mcpSendError(id json.RawMessage, code int, msg string) {
	resp := jsonRPCResponse{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}}
	data, _ := json.Marshal(resp)
	fmt.Fprintf(os.Stdout, "%s\n", data)
}
func mcpToolDefs() []mcpToolDef {
	s := func(s string) json.RawMessage { return json.RawMessage(s) }
	return []mcpToolDef{
		{Name: "fedit_show", Description: "Display file contents with line numbers. Optionally limit to a line range.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"line":{"type":"integer","description":"Start line (1-based, optional)"},"end":{"type":"integer","description":"End line (inclusive, optional)"}},"required":["file"]}`)},
		{Name: "fedit_insert", Description: "Insert content after a given line number.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"line":{"type":"integer","description":"Insert after this line (0 = beginning)"},"text":{"type":"string","description":"Content to insert (use \\n for newlines)"}},"required":["file","line","text"]}`)},
		{Name: "fedit_delete", Description: "Delete one or more lines.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"line":{"type":"integer","description":"Start line to delete"},"end":{"type":"integer","description":"End line (inclusive, defaults to start line)"}},"required":["file","line"]}`)},
		{Name: "fedit_replace", Description: "Replace a line range with new content.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"line":{"type":"integer","description":"Start line"},"end":{"type":"integer","description":"End line (inclusive)"},"text":{"type":"string","description":"Replacement content (use \\n for newlines)"}},"required":["file","line","end","text"]}`)},
		{Name: "fedit_replaceall", Description: "Global find-and-replace. Use match_regex for capture groups. Use files for glob. Add stream=true for large files.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"match":{"type":"string","description":"Literal text to find (use match_regex for regex)"},"match_regex":{"type":"string","description":"Regex pattern with capture groups $1 $2 (alternative to match)"},"files":{"type":"string","description":"Glob pattern to apply replaceall across multiple files (e.g. *.go)"},"stream":{"type":"boolean","description":"Line-by-line I/O for multi-GB files without loading into memory"},"text":{"type":"string","description":"Replacement text"},"textfile":{"type":"string","description":"File containing replacement text"}},"required":["file"]}`)},
		{Name: "fedit_write", Description: "Write or overwrite an entire file.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"text":{"type":"string","description":"Full file content (use \\n for newlines)"}},"required":["file","text"]}`)},
		{Name: "fedit_map", Description: "Structural overview of a source file. Supports 17 languages: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix. Use lang param for ambiguous extensions.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"lang":{"type":"string","description":"Language hint: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix (auto-detected from extension if omitted)"}},"required":["file"]}`)},
		{Name: "fedit_find", Description: "Find all lines matching a substring. Add stream=true for large files.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"match":{"type":"string","description":"Substring to search for"},"nth":{"type":"integer","description":"Which occurrence (default 1, -1 for last)"},"stream":{"type":"boolean","description":"Streaming grep-style output for large files"}},"required":["file","match"]}`)},
		{Name: "fedit_insertafter", Description: "Insert content after a line matching a substring.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"match":{"type":"string","description":"Substring to match"},"text":{"type":"string","description":"Content to insert (use \\n for newlines)"},"nth":{"type":"integer","description":"Which occurrence (default 1, -1 for last)"}},"required":["file","match","text"]}`)},
		{Name: "fedit_insertbefore", Description: "Insert content before a line matching a substring.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},"match":{"type":"string","description":"Substring to match"},"text":{"type":"string","description":"Content to insert (use \\n for newlines)"},"nth":{"type":"integer","description":"Which occurrence (default 1, -1 for last)"}},"required":["file","match","text"]}`)},
		{Name: "fedit_move", Description: "Move a line range to a new position. Atomic. Overlap (dest inside src) rejected. -times N: cut once, paste N times. Use -block/-beforeblock/-afterblock with -lang for mapper-aware moves.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string"},"line":{"type":"integer"},"end":{"type":"integer"},"match":{"type":"string"},"endmatch":{"type":"string"},"after":{"type":"integer","description":"Destination after line N (0=beginning)"},"before":{"type":"integer"},"aftermatch":{"type":"string"},"beforematch":{"type":"string"},"block":{"type":"string","description":"Source block name (requires lang)"},"beforeblock":{"type":"string","description":"Dest: before named block (requires lang)"},"afterblock":{"type":"string","description":"Dest: after named block (requires lang)"},"lang":{"type":"string","description":"go, python, js, ts, rust, java, cs, ruby, php, hcl, tf, terraform, nix"},"times":{"type":"integer"},"nth":{"type":"integer"}},"required":["file"]}`)},
		{Name: "fedit_copy", Description: "Copy a line range to a new position. Atomic. Snapshot semantics: all N copies identical even with overlap. Supports -block/-beforeblock/-afterblock with -lang.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string"},"line":{"type":"integer"},"end":{"type":"integer"},"match":{"type":"string"},"endmatch":{"type":"string"},"after":{"type":"integer","description":"Destination after line N (0=beginning)"},"before":{"type":"integer"},"aftermatch":{"type":"string"},"beforematch":{"type":"string"},"block":{"type":"string","description":"Source block name (requires lang)"},"beforeblock":{"type":"string","description":"Dest: before named block (requires lang)"},"afterblock":{"type":"string","description":"Dest: after named block (requires lang)"},"lang":{"type":"string","description":"go, python, js, ts, rust, java, cs, ruby, php, hcl, tf, terraform, nix"},"times":{"type":"integer"},"nth":{"type":"integer"}},"required":["file"]}`)},
		{Name: "fedit_fields", Description: "Extract column N from a delimited file (CSV, TSV, colon-separated). Always streams -- no memory limit. Output goes to stdout.", InputSchema: s(`{"type":"object","properties":{"file":{"type":"string"},"col":{"type":"integer","description":"Column number (1-based)"},"delim":{"type":"string","description":"Field delimiter: comma for CSV, default is tab"}},"required":["file","col"]}`)},
	}
}

func mcpExecTool(name string, args map[string]any) mcpCallResult {
	start := time.Now()
	getStr := func(key string) string {
		if v, ok := args[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}
	getInt := func(key string, def int) int {
		if v, ok := args[key]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			}
		}
		return def
	}
	file := getStr("file")
	if file == "" {
		return mcpErrorResult("missing required parameter: file")
	}
	file, _ = filepath.Abs(file)
	switch name {
	case "fedit_show":
		return mcpDoShow(file, getInt("line", 0), getInt("end", 0), start)
	case "fedit_write":
		return mcpDoWrite(file, getStr("text"), start)
	case "fedit_insert":
		return mcpDoInsert(file, getInt("line", 0), getStr("text"), start)
	case "fedit_delete":
		ln := getInt("line", 0)
		return mcpDoDelete(file, ln, getInt("end", ln), start)
	case "fedit_replace":
		return mcpDoReplace(file, getInt("line", 0), getInt("end", 0), getStr("text"), start)
	case "fedit_replaceall":
		return mcpDoReplaceAll(file, getStr("match"), getStr("text"), start)
	case "fedit_map":
		return mcpDoMap(file, getStr("lang"), start)
	case "fedit_find":
		return mcpDoFind(file, getStr("match"), getInt("nth", 1), start)
	case "fedit_insertafter":
		return mcpDoInsertMatch(file, getStr("match"), getInt("nth", 1), getStr("text"), false, start)
	case "fedit_insertbefore":
		return mcpDoInsertMatch(file, getStr("match"), getInt("nth", 1), getStr("text"), true, start)
	case "fedit_move":
		return mcpDoMove(file, getInt("line", 0), getInt("end", 0), getStr("match"), getStr("endmatch"),
			getInt("after", -1), getInt("before", -1), getStr("aftermatch"), getStr("beforematch"),
			getStr("block"), getStr("beforeblock"), getStr("afterblock"), getStr("lang"),
			getInt("nth", 1), getInt("times", 1), start)
	case "fedit_copy":
		return mcpDoCopy(file, getInt("line", 0), getInt("end", 0), getStr("match"), getStr("endmatch"),
			getInt("after", -1), getInt("before", -1), getStr("aftermatch"), getStr("beforematch"),
			getStr("block"), getStr("beforeblock"), getStr("afterblock"), getStr("lang"),
			getInt("nth", 1), getInt("times", 1), start)
		case "fedit_fields":
		return mcpDoFields(file, getInt("col", 0), getStr("delim"), start)
	default:
		return mcpErrorResult(fmt.Sprintf("unknown tool: %s", name))
	}
}

func mcpErrorResult(msg string) mcpCallResult {
	return mcpCallResult{Content: []mcpContent{{Type: "text", Text: msg}}, IsError: true}
}

func mcpOK(msg string) mcpCallResult {
	return mcpCallResult{Content: []mcpContent{{Type: "text", Text: msg}}}
}

func mcpStats(op, file string, linesBefore, linesAfter int, match string, start time.Time) string {
	elapsed := time.Since(start)
	var elapsedStr string
	if elapsed < time.Millisecond {
		elapsedStr = "<1ms"
	} else {
		elapsedStr = elapsed.Round(time.Millisecond).String()
	}
	var b strings.Builder
	b.WriteString("\n=== STATS ===\n")
	fmt.Fprintf(&b, "  op:      %s\n", op)
	fmt.Fprintf(&b, "  file:    %s\n", file)
	if match != "" {
		fmt.Fprintf(&b, "  match:   %q\n", match)
	}
	delta := linesAfter - linesBefore
	if delta > 0 {
		fmt.Fprintf(&b, "  lines:   +%d (%d -> %d)\n", delta, linesBefore, linesAfter)
	} else if delta < 0 {
		fmt.Fprintf(&b, "  lines:   %d (%d -> %d)\n", delta, linesBefore, linesAfter)
	} else {
		fmt.Fprintf(&b, "  lines:   0 (unchanged, %d total)\n", linesBefore)
	}
	fmt.Fprintf(&b, "  elapsed: %s\n", elapsedStr)
	return b.String()
}

func inferLang(file string) string {
	ext := strings.ToLower(filepath.Ext(file))
	switch ext {
	case ".go":
		return "go"
	case ".html", ".htm":
		return "html"
	case ".sql":
		return "sql"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	case ".css":
		return "css"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".cs":
		return "csharp"
	case ".yaml", ".yml":
		return "yaml"
	case ".toml":
		return "toml"
	case ".md":
		return "markdown"
	case ".rb":
		return "ruby"
	case ".php":
		return "php"
	}
	base := strings.ToLower(filepath.Base(file))
	switch base {
	case "dockerfile":
		return "dockerfile"
	case "makefile":
		return "makefile"
	}
	return ""
}
func mcpDoShow(file string, startLine, endLine int, start time.Time) mcpCallResult {
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	if startLine == 0 && endLine == 0 {
		startLine = 1
		endLine = len(lines)
	}
	if endLine == 0 {
		endLine = startLine
	}
	if startLine < 1 {
		startLine = 1
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	var b strings.Builder
	width := len(fmt.Sprintf("%d", endLine))
	for i := startLine; i <= endLine; i++ {
		fmt.Fprintf(&b, "%*d | %s\n", width, i, lines[i-1])
	}
	fmt.Fprintf(&b, "--- %d total lines ---", len(lines))
	return mcpOK(b.String())
}

func mcpDoWrite(file, text string, start time.Time) mcpCallResult {
	content := expandText(text)
	if len(content) == 0 {
		return mcpErrorResult("Nothing to write")
	}
	if err := writeLines(file, content); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Wrote %d line(s) to %s", len(content), file)
	msg += mcpStats("write", file, 0, len(content), "", start)
	return mcpOK(msg)
}

func mcpDoInsert(file string, afterLine int, text string, start time.Time) mcpCallResult {
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	newLines := expandText(text)
	if len(newLines) == 0 {
		return mcpErrorResult("Nothing to insert")
	}
	if afterLine < 0 || afterLine > len(lines) {
		return mcpErrorResult(fmt.Sprintf("Line %d out of range (file has %d lines)", afterLine, len(lines)))
	}
	linesBefore := len(lines)
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:afterLine]...)
	result = append(result, newLines...)
	result = append(result, lines[afterLine:]...)
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Inserted %d line(s) after line %d (%d total now)", len(newLines), afterLine, len(result))
	msg += mcpStats("insert", file, linesBefore, len(result), "", start)
	return mcpOK(msg)
}

func mcpDoDelete(file string, startLine, endLine int, start time.Time) mcpCallResult {
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	if startLine < 1 || endLine > len(lines) || startLine > endLine {
		return mcpErrorResult(fmt.Sprintf("Invalid range %d-%d (file has %d lines)", startLine, endLine, len(lines)))
	}
	linesBefore := len(lines)
	result := make([]string, 0, len(lines)-(endLine-startLine+1))
	result = append(result, lines[:startLine-1]...)
	result = append(result, lines[endLine:]...)
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Deleted lines %d-%d (%d total now)", startLine, endLine, len(result))
	msg += mcpStats("delete", file, linesBefore, len(result), "", start)
	return mcpOK(msg)
}

func mcpDoReplace(file string, startLine, endLine int, text string, start time.Time) mcpCallResult {
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	if startLine < 1 || endLine > len(lines) || startLine > endLine {
		return mcpErrorResult(fmt.Sprintf("Invalid range %d-%d (file has %d lines)", startLine, endLine, len(lines)))
	}
	newLines := expandText(text)
	linesBefore := len(lines)
	result := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newLines))
	result = append(result, lines[:startLine-1]...)
	result = append(result, newLines...)
	result = append(result, lines[endLine:]...)
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Replaced lines %d-%d with %d line(s) (%d total now)", startLine, endLine, len(newLines), len(result))
	msg += mcpStats("replace", file, linesBefore, len(result), "", start)
	return mcpOK(msg)
}

func mcpDoReplaceAll(file, search, replacement string, start time.Time) mcpCallResult {
	if search == "" {
		return mcpErrorResult("replaceall requires match (text to find)")
	}
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	count := 0
	for i, line := range lines {
		if strings.Contains(line, search) {
			lines[i] = strings.ReplaceAll(line, search, replacement)
			count++
		}
	}
	if count == 0 {
		return mcpErrorResult(fmt.Sprintf("No lines contain: %s", search))
	}
	if err := writeLines(file, lines); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Replaced '%s' on %d line(s) (%d total)", search, count, len(lines))
	msg += mcpStats("replaceall", file, len(lines), len(lines), search, start)
	return mcpOK(msg)
}

func mcpDoMap(file, lang string, start time.Time) mcpCallResult {
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	if lang == "" {
		lang = inferLang(file)
	}
	if lang == "" {
		return mcpErrorResult("Cannot determine language. Use lang parameter.")
	}
	oldOut := os.Stdout
	oldErr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Stderr = w
	doMap(lines, file, lang)
	w.Close()
	os.Stdout = oldOut
	os.Stderr = oldErr
	var buf strings.Builder
	io.Copy(&buf, r)
	r.Close()
	return mcpOK(buf.String())
}

func mcpDoFind(file, match string, nth int, start time.Time) mcpCallResult {
	if match == "" {
		return mcpErrorResult("find requires match parameter")
	}
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	hits := findMatches(lines, match)
	if len(hits) == 0 {
		return mcpErrorResult(fmt.Sprintf("No matches found for: %s", match))
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Found %d match(es) for: %s\n\n", len(hits), match)
	for _, h := range hits {
		fmt.Fprintf(&b, " >  %4d | %s\n", h, lines[h-1])
	}
	if nth == -1 {
		fmt.Fprintf(&b, "\nLast occurrence is at line %d", hits[len(hits)-1])
	} else if nth >= 1 && nth <= len(hits) {
		fmt.Fprintf(&b, "\nOccurrence %d is at line %d", nth, hits[nth-1])
	}
	return mcpOK(b.String())
}

func mcpDoInsertMatch(file, match string, nth int, text string, before bool, start time.Time) mcpCallResult {
	if match == "" {
		return mcpErrorResult("requires match parameter")
	}
	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	newLines := expandText(text)
	if len(newLines) == 0 {
		return mcpErrorResult("Nothing to insert")
	}
	hits := findMatches(lines, match)
	if len(hits) == 0 {
		return mcpErrorResult(fmt.Sprintf("No match found for: %s", match))
	}
	var targetLine int
	if nth == -1 {
		targetLine = hits[len(hits)-1]
	} else if nth >= 1 && nth <= len(hits) {
		targetLine = hits[nth-1]
	} else {
		return mcpErrorResult(fmt.Sprintf("Occurrence %d out of range (found %d)", nth, len(hits)))
	}
	var insertAfter int
	direction := "after"
	if before {
		insertAfter = targetLine - 1
		direction = "before"
	} else {
		insertAfter = targetLine
	}
	linesBefore := len(lines)
	result := make([]string, 0, len(lines)+len(newLines))
	result = append(result, lines[:insertAfter]...)
	result = append(result, newLines...)
	result = append(result, lines[insertAfter:]...)
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	msg := fmt.Sprintf("Inserted %d line(s) %s line %d (%d total now)", len(newLines), direction, targetLine, len(result))
	msg += mcpStats("insert"+direction, file, linesBefore, len(result), match, start)
	return mcpOK(msg)
}

func mcpDoMove(file string, lineFlag, endFlag int, matchFlag, endmatchFlag string,
	afterFlag, beforeFlag int, afterMatchFlag, beforeMatchFlag string,
	blockFlag, beforeBlockFlag, afterBlockFlag, lang string,
	nth, times int, start time.Time) mcpCallResult {

	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	srcL, srcE := lineFlag, endFlag
	srcM, srcEM := matchFlag, endmatchFlag
	dstA, dstB := afterFlag, beforeFlag
	dstAM, dstBM := afterMatchFlag, beforeMatchFlag

	if blockFlag != "" {
		s, e, bErr := resolveBlock(lines, lang, blockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Source error: %v", bErr))
		}
		srcL, srcE, srcM, srcEM = s, e, "", ""
	}
	if beforeBlockFlag != "" {
		bs, _, bErr := resolveBlock(lines, lang, beforeBlockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Destination error: %v", bErr))
		}
		dstB, dstA, dstAM, dstBM = bs, -1, "", ""
	}
	if afterBlockFlag != "" {
		_, be, bErr := resolveBlock(lines, lang, afterBlockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Destination error: %v", bErr))
		}
		dstA, dstB, dstAM, dstBM = be, -1, "", ""
	}

	srcStart, srcEnd, err := resolveSourceLines(lines, srcL, srcE, srcM, srcEM, nth)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Source error: %v", err))
	}
	destAfter, destLine, destDesc, err := resolveDestLine(lines, dstA, dstB, dstAM, dstBM)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Destination error: %v", err))
	}
	linesBefore := len(lines)
	result, _, err := execMove(lines, srcStart, srcEnd, destAfter, destLine, times)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Move error: %v", err))
	}
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	blockSize := srcEnd - srcStart + 1
	var msg string
	if times == 1 {
		msg = fmt.Sprintf("Moved lines %d-%d (%d lines) %s (%d total now)", srcStart, srcEnd, blockSize, destDesc, len(result))
	} else {
		msg = fmt.Sprintf("Moved lines %d-%d (%d lines) %s x%d (%d total now)", srcStart, srcEnd, blockSize, destDesc, times, len(result))
	}
	msg += mcpStats("move", file, linesBefore, len(result), "", start)
	return mcpOK(msg)
}

func mcpDoCopy(file string, lineFlag, endFlag int, matchFlag, endmatchFlag string,
	afterFlag, beforeFlag int, afterMatchFlag, beforeMatchFlag string,
	blockFlag, beforeBlockFlag, afterBlockFlag, lang string,
	nth, times int, start time.Time) mcpCallResult {

	lines, err := readLines(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading file: %v", err))
	}
	srcL, srcE := lineFlag, endFlag
	srcM, srcEM := matchFlag, endmatchFlag
	dstA, dstB := afterFlag, beforeFlag
	dstAM, dstBM := afterMatchFlag, beforeMatchFlag

	if blockFlag != "" {
		s, e, bErr := resolveBlock(lines, lang, blockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Source error: %v", bErr))
		}
		srcL, srcE, srcM, srcEM = s, e, "", ""
	}
	if beforeBlockFlag != "" {
		bs, _, bErr := resolveBlock(lines, lang, beforeBlockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Destination error: %v", bErr))
		}
		dstB, dstA, dstAM, dstBM = bs, -1, "", ""
	}
	if afterBlockFlag != "" {
		_, be, bErr := resolveBlock(lines, lang, afterBlockFlag)
		if bErr != nil {
			return mcpErrorResult(fmt.Sprintf("Destination error: %v", bErr))
		}
		dstA, dstB, dstAM, dstBM = be, -1, "", ""
	}

	srcStart, srcEnd, err := resolveSourceLines(lines, srcL, srcE, srcM, srcEM, nth)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Source error: %v", err))
	}
	destAfter, _, destDesc, err := resolveDestLine(lines, dstA, dstB, dstAM, dstBM)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Destination error: %v", err))
	}
	linesBefore := len(lines)
	result, _, err := execCopy(lines, srcStart, srcEnd, destAfter, times)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Copy error: %v", err))
	}
	if err := writeLines(file, result); err != nil {
		return mcpErrorResult(fmt.Sprintf("Error writing file: %v", err))
	}
	blockSize := srcEnd - srcStart + 1
	var msg string
	if times == 1 {
		msg = fmt.Sprintf("Copied lines %d-%d (%d lines) %s (%d total now)", srcStart, srcEnd, blockSize, destDesc, len(result))
	} else {
		msg = fmt.Sprintf("Copied lines %d-%d (%d lines) %s x%d (%d total now)", srcStart, srcEnd, blockSize, destDesc, times, len(result))
	}
	msg += mcpStats("copy", file, linesBefore, len(result), "", start)
	return mcpOK(msg)
}

func mcpDoFields(file string, col int, delim string, start time.Time) mcpCallResult {
	if col < 1 {
		return mcpErrorResult("col must be >= 1")
	}
	if delim == "" {
		delim = "\t"
	}
	src, err := os.Open(file)
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("Error opening file: %v", err))
	}
	defer src.Close()

	scanner := bufio.NewScanner(src)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)
	var results []string
	lineNum, extracted := 0, 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if line == "" {
			results = append(results, "")
			continue
		}
		parts := strings.Split(line, delim)
		if col <= len(parts) {
			results = append(results, parts[col-1])
			extracted++
		}
	}
	if scanner.Err() != nil {
		return mcpErrorResult(fmt.Sprintf("Error reading: %v", scanner.Err()))
	}
	msg := strings.Join(results, "\n")
	msg += fmt.Sprintf("\n\n(col %d from %d/%d lines, delim=%q, elapsed %s)",
		col, extracted, lineNum, delim, time.Since(start).Round(time.Millisecond))
	return mcpOK(msg)
}
