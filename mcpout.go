package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mcpOutputProp is added to the schema of every tool: the result text can be
// written to a file instead of coming back to the model.
const mcpOutputProp = "\"output_file\":{\"type\":\"string\",\"description\":\"Write the full result text to this file (UTF-8, no BOM) and return only a one-line summary. An error result is returned in full and the file is left alone. The file may not be one of the edited files and its folder must exist\"},\"output_append\":{\"type\":\"boolean\",\"description\":\"With output_file: append to the file instead of overwriting it\"},"

func mcpAddOutput(defs []mcpToolDef) []mcpToolDef {
	for i, d := range defs {
		raw, ok := d.InputSchema.(json.RawMessage)
		if !ok {
			continue
		}
		s := strings.Replace(string(raw), "\"properties\":{", "\"properties\":{"+mcpOutputProp, 1)
		defs[i].InputSchema = json.RawMessage(s)
	}
	return defs
}

// mcpWantsOutput reports whether the call names an output_file or sets
// output_append.
func mcpWantsOutput(args map[string]any) bool {
	if s, ok := args["output_file"].(string); ok && s != "" {
		return true
	}
	b, _ := args["output_append"].(bool)
	return b
}

// mcpRunOutput runs a tool and writes its result text to output_file. Every
// check that can fail happens before the tool runs, so a refused call changes
// nothing. An error result from the tool comes back in full and the file is
// left alone; on success only a one-line summary comes back.
func mcpRunOutput(name string, args map[string]any) mcpCallResult {
	path := mcpArgString(args, "output_file")
	appendMode, _ := args["output_append"].(bool)
	if path == "" {
		return mcpErrorResult("output_append requires output_file")
	}
	if msg := outputResultRefuse(path, mcpArgString(args, "file"), "", mcpArgString(args, "dest")); msg != "" {
		return mcpErrorResult(strings.Replace(msg, "-outputresult", "output_file", 1))
	}
	dir := filepath.Dir(path)
	if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
		return mcpErrorResult(fmt.Sprintf("output_file: directory %s does not exist", dir))
	}
	inner := map[string]any{}
	for k, v := range args {
		if k != "output_file" && k != "output_append" {
			inner[k] = v
		}
	}
	res := mcpExecTool(name, inner)
	if res.IsError {
		return res
	}
	var parts []string
	for _, c := range res.Content {
		parts = append(parts, c.Text)
	}
	text := strings.TrimPrefix(strings.Join(parts, "\n"), "\xef\xbb\xbf")
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	verb := "written to"
	if appendMode {
		flags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
		verb = "appended to"
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err == nil {
		_, err = f.WriteString(text)
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}
	if err != nil {
		return mcpErrorResult(fmt.Sprintf("output_file: the operation succeeded but the result could not be written to %s: %v\n%s", path, err, text))
	}
	return mcpOK(fmt.Sprintf("result %s %s (%d lines)", verb, path, strings.Count(text, "\n")))
}
