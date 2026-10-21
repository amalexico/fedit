package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Wiring of the audit log into a command-line run: the six -log-* flags, the
// session that is started right after flag.Parse and finished in exitNow, and
// the fields of the begin record. The logging itself lives in logsession.go.

// Exit codes of strict logging. Exit 4: the begin record could not be written,
// so the operation was refused and nothing changed. Exit 5: the operation
// completed but the end record could not be written.
const (
	exitLogRefused = 4
	exitLogGap     = 5
)

var (
	// logFlagVals holds the -log-* flag values, registered before flag.Parse.
	logFlagVals struct{ Mode, File, MaxMB, Content, Level, Scope *string }

	logSourceName = "cli" // the source field of the begin record
	logRun        *logSession
	logRunStrict  bool
	logRunEnded   bool
	logWarned     bool
)

// logRegisterFlags defines the -log-* flags. Call it before flag.Parse.
func logRegisterFlags() {
	logFlagVals.Mode = flag.String("log-mode", "", "Audit log mode: off (default), permissive or strict (env FEDIT_LOG_MODE)")
	logFlagVals.File = flag.String("log-file", "", "Audit log file (env FEDIT_LOG_FILE)")
	logFlagVals.MaxMB = flag.String("log-max-mb", "", "Rotate the audit log at this many MB, 0 = never (env FEDIT_LOG_MAX_MB)")
	logFlagVals.Content = flag.String("log-content", "", "on: log -text values verbatim; off (default): length and SHA-256 only (env FEDIT_LOG_CONTENT)")
	logFlagVals.Level = flag.String("log-level", "", "Severities written: all (default), warning or error (env FEDIT_LOG_LEVEL)")
	logFlagVals.Scope = flag.String("log-scope", "", "Operations logged: all (default) or writes (env FEDIT_LOG_SCOPE)")
}

// logSourceGuided marks the run as started through the guided front end.
func logSourceGuided() { logSourceName = "guided" }

// logUsage prints the audit log part of the usage text.
func logUsage() {
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Audit log flags (each also has an environment variable FEDIT_LOG_*):")
	fmt.Fprintln(os.Stderr, "  -log-mode M     off (default), permissive (log; spool the records when the log cannot be written) or strict (refuse to run when the log cannot be written; exit 4, or 5 when only the end record fails)")
	fmt.Fprintln(os.Stderr, "  -log-file FILE  Log file (default %LOCALAPPDATA%\\fedit\\fedit.log)")
	fmt.Fprintln(os.Stderr, "  -log-max-mb N   Rotate the log at N MB (default 10, 0 = never)")
	fmt.Fprintln(os.Stderr, "  -log-content on|off  Log -text values verbatim instead of length and SHA-256 (default off)")
	fmt.Fprintln(os.Stderr, "  -log-level L    all (default), warning or error: which severities are written (strict always writes all)")
	fmt.Fprintln(os.Stderr, "  -log-scope S    all (default: every operation, reads included) or writes (only operations that can change a file)")
}

// logFinishCode is the exit code of a run that logs: in strict mode a run
// that succeeded but whose end record could not be written exits 5. A run
// that already failed keeps its own code.
func logFinishCode(code int, endErr error, strict bool) int {
	if strict && endErr != nil && code == 0 {
		return exitLogGap
	}
	return code
}

// logCLIInput is what the begin record of a command-line run is built from.
type logCLIInput struct {
	Source, User, Host, Cwd string
	Pid                     int
	File, Files, Dest       string
	Text, TextHex, TextFile string
	Content                 bool
}

// logAbsList makes every entry of a comma list absolute against cwd. An
// absolute entry is only cleaned, and a glob keeps its pattern.
func logAbsList(spec, cwd string) string {
	var out []string
	for _, p := range strings.Split(spec, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(cwd, p)
		}
		out = append(out, filepath.Clean(p))
	}
	return strings.Join(out, ",")
}

// logCLIFields builds the fields of the begin record, after op: source, user,
// host, pid, cwd, then file, files and dest (absolute, only when given), then
// the text (length and SHA-256, or verbatim when Content is set) and textfile.
// -texthex is hashed after decoding; an invalid one is hashed as typed.
func logCLIFields(in logCLIInput) []logKV {
	kvs := []logKV{
		{"source", in.Source},
		{"user", in.User},
		{"host", in.Host},
		{"pid", strconv.Itoa(in.Pid)},
		{"cwd", in.Cwd},
	}
	for _, p := range []struct{ key, spec string }{{"file", in.File}, {"files", in.Files}, {"dest", in.Dest}} {
		if v := logAbsList(p.spec, in.Cwd); v != "" {
			kvs = append(kvs, logKV{p.key, v})
		}
	}
	var raw []byte
	have := false
	switch {
	case in.TextHex != "":
		have = true
		if b, err := hex.DecodeString(in.TextHex); err == nil {
			raw = b
		} else {
			raw = []byte(in.TextHex)
		}
	case in.Text != "":
		have, raw = true, []byte(in.Text)
	}
	if have {
		if in.Content {
			kvs = append(kvs, logKV{"text", string(raw)})
		} else {
			kvs = append(kvs, logTextFields("text", raw)...)
		}
	}
	if v := logAbsList(in.TextFile, in.Cwd); v != "" {
		kvs = append(kvs, logKV{"textfile", v})
	}
	return kvs
}

// logIdentity returns the OS user (with the domain on Windows), the host
// name, the process id and the working directory.
func logIdentity() (usr, host string, pid int, cwd string) {
	if u, err := user.Current(); err == nil {
		usr = u.Username
	}
	if usr == "" {
		usr = os.Getenv("USERNAME")
	}
	host, _ = os.Hostname()
	cwd, _ = os.Getwd()
	return usr, host, os.Getpid(), cwd
}

// logStart resolves the logging settings from the -log-* flags and the
// FEDIT_LOG_* environment, starts the session and writes the begin record. It
// returns 0 to go on, 1 for an invalid setting, or exitLogRefused when strict
// mode could not write the begin record; in both cases nothing has changed.
func logStart(op string, in logCLIInput) int {
	home, _ := os.UserHomeDir()
	settings, err := resolveLogSettings(os.Getenv, home, os.UserConfigDir,
		logInputs{
			Mode:    *logFlagVals.Mode,
			File:    *logFlagVals.File,
			MaxMB:   *logFlagVals.MaxMB,
			Content: *logFlagVals.Content,
			Level:   *logFlagVals.Level,
			Scope:   *logFlagVals.Scope,
		},
		logInputsFromEnv(os.Getenv))
	if err != nil {
		fmt.Fprintf(os.Stderr, "fedit: %v\n", err)
		return 1
	}
	if !newLogSession(settings, op, time.Now, fileLogIO(), func() {}, nil).Active() {
		return 0
	}

	in.Source = logSourceName
	in.Content = settings.Content
	in.User, in.Host, in.Pid, in.Cwd = logIdentity()
	notice := func() {
		fmt.Fprintln(os.Stderr, "fedit: The action was not logged: the audit log could not be written, so the records were saved in the spool and the next logged run will merge them into the log")
	}
	ls := newLogSession(settings, op, time.Now, fileLogIO(), notice, logCLIFields(in))
	if err := ls.Begin(); err != nil {
		if ls.MustAbort(err) {
			fmt.Fprintf(os.Stderr, "fedit: log_mode=strict: cannot write the audit log (%v); nothing was changed\n", err)
			return exitLogRefused
		}
		fmt.Fprintf(os.Stderr, "fedit: The action was not logged: %v\n", err)
		logWarned = true
	}
	logRun, logRunStrict = ls, settings.Mode == logStrict
	return 0
}

// logFinish writes the end record for the final exit code and returns the
// code to exit with. It does nothing when no session is running, and only
// writes once.
func logFinish(code int) int {
	if logRun == nil || logRunEnded {
		return code
	}
	logRunEnded = true
	err := logRun.End(code, nil)
	if err != nil {
		switch {
		case logRunStrict:
			fmt.Fprintf(os.Stderr, "fedit: log_mode=strict: the end record could not be written (%v); the audit log has a gap for this run\n", err)
		case !logWarned:
			fmt.Fprintf(os.Stderr, "fedit: The action was not logged: %v\n", err)
		}
	}
	return logFinishCode(code, err, logRunStrict)
}
