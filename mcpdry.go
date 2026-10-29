package main

import (
	"bytes"
	"encoding/json"
	"strings"
)

// mcpDryTools are the MCP tools that change files. Only they take dry_run.
var mcpDryTools = func() map[string]bool {
	m := map[string]bool{}
	for _, n := range []string{"fedit_insert", "fedit_delete", "fedit_replace", "fedit_write", "fedit_writeraw", "fedit_replaceall", "fedit_insertafter", "fedit_insertbefore", "fedit_move", "fedit_copy"} {
		m[n] = true
	}
	return m
}()

const mcpDryProp = "\"dry_run\":{\"type\":\"boolean\",\"description\":\"Preview only: return a unified diff of what would change and write nothing\"},"

// mcpToolDefs is the tool list the server advertises: the base list with the
// dry_run property added to every tool that changes files.
func mcpToolDefs() []mcpToolDef {
	return mcpAddOutput(mcpAddDryRun(mcpBaseToolDefs()))
}

func mcpAddDryRun(defs []mcpToolDef) []mcpToolDef {
	for i, d := range defs {
		raw, ok := d.InputSchema.(json.RawMessage)
		if !ok || !mcpDryTools[d.Name] {
			continue
		}
		s := strings.Replace(string(raw), "\"properties\":{", "\"properties\":{"+mcpDryProp, 1)
		defs[i].InputSchema = json.RawMessage(s)
	}
	return defs
}

// mcpWantsDry reports whether this call is a changing tool with dry_run true.
func mcpWantsDry(name string, args map[string]any) bool {
	if !mcpDryTools[name] {
		return false
	}
	b, _ := args["dry_run"].(bool)
	return b
}

// mcpRunDry runs a changing tool with dryRun on and the diff captured in a
// buffer. Under MCP stdout is the JSON-RPC channel, so nothing may be printed
// there. The server handles one request at a time, so the two globals are safe.
func mcpRunDry(name string, args map[string]any) mcpCallResult {
	var buf bytes.Buffer
	oldDry, oldOut := dryRun, dryOut
	dryRun, dryOut = true, &buf
	defer func() { dryRun, dryOut = oldDry, oldOut }()
	logNote("dry", "true")
	res := mcpExecTool(name, args)
	head := "dry-run: applied=false (no file was changed)\n" + buf.String()
	if len(res.Content) > 0 {
		res.Content[0].Text = head + res.Content[0].Text
	} else {
		res.Content = []mcpContent{{Type: "text", Text: head}}
	}
	return res
}

// mcpBatchVerb ends a batch total: succeeded, or would succeed under dry_run.
func mcpBatchVerb() string {
	if dryRun {
		return "would succeed"
	}
	return "succeeded"
}
