package main

import (
	"encoding/json"
	"strings"
)

// Audit logging of MCP tool calls. Every tools/call is one logged operation:
// the begin record is written before the tool runs, the end record when it
// returns. The settings come from the FEDIT_LOG_* environment only (the mcp
// command has no -log-* flags). The client name and version come from the
// clientInfo of the initialize request.

var mcpClientName, mcpClientVersion string

// mcpClean makes a client-supplied value safe for a log field: control
// characters are dropped and the length is capped at 100 bytes.
func mcpClean(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if r < 0x20 || r == 0x7f {
			continue
		}
		if b.Len()+len(string(r)) > 100 {
			break
		}
		b.WriteRune(r)
	}
	return b.String()
}

// mcpNoteClient remembers the clientInfo of an initialize request. Params that
// are missing or not valid JSON leave the client unknown.
func mcpNoteClient(params []byte) {
	mcpClientName, mcpClientVersion = "", ""
	var p struct {
		ClientInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"clientInfo"`
	}
	if len(params) == 0 || json.Unmarshal(params, &p) != nil {
		return
	}
	mcpClientName = mcpClean(p.ClientInfo.Name)
	mcpClientVersion = mcpClean(p.ClientInfo.Version)
}

func mcpArgString(args map[string]any, key string) string {
	if s, ok := args[key].(string); ok {
		return s
	}
	return ""
}

// mcpLogReset clears the per-run logging state. The logging session is built
// for one run per process, and the MCP server runs many.
func mcpLogReset() {
	logRun, logRunStrict, logRunEnded, logWarned = nil, false, false, false
	logNotes = nil
	logHashPath, logHashBefore = "", ""
}

// mcpLoggedExec runs one tool call inside a logging session. In strict mode a
// begin record that cannot be written refuses the call and nothing changes; an
// end record that cannot be written after a successful call turns the result
// into an error that says the operation completed. A failed verification is
// logged as exit 3, any other error result as exit 1.
func mcpLoggedExec(name string, args map[string]any) mcpCallResult {
	mcpLogReset()
	savedSource := logSourceName
	logSourceName = "mcp"
	defer func() {
		logSourceName = savedSource
		mcpLogReset()
	}()
	in := logCLIInput{
		File:          mcpArgString(args, "file"),
		Dest:          mcpArgString(args, "dest"),
		Text:          mcpArgString(args, "text"),
		TextFile:      mcpArgString(args, "textfile"),
		Client:        mcpClientName,
		ClientVersion: mcpClientVersion,
	}
	switch logStartWith(logInputs{}, strings.TrimPrefix(name, "fedit_"), in) {
	case 0:
	case exitLogRefused:
		return mcpErrorResult("fedit: log_mode=strict: the audit log could not be written, so the operation was refused and nothing was changed")
	default:
		return mcpErrorResult("fedit: invalid audit log setting (see the server's stderr); nothing was changed")
	}
	res := mcpExecTool(name, args)
	code := 0
	if res.IsError {
		code = 1
	}
	if v, _ := logNoteValue("verify"); v == "FAIL" {
		code = 3
	}
	if logFinish(code) == exitLogGap {
		res.IsError = true
		res.Content = append(res.Content, mcpContent{Type: "text", Text: "fedit: log_mode=strict: the operation completed, but the end record could not be written; the audit log has a gap for this run"})
	}
	return res
}
