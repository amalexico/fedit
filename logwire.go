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

// logParentEnv carries the id of a guided parent process to the child fedit
// processes it starts, one per step.
const logParentEnv = "FEDIT_LOG_PARENT"

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
	cfgRegisterFlags()
}

// logSourceGuided marks the run as started through the guided front end.
func logSourceGuided() { logSourceName = "guided" }

// logNoticeHook, when set, also receives every not-logged notice; the MCP server sets it for the call so the notice can go into the tool result.
var logNoticeHook func(string)

// logNotify prints a not-logged notice to stderr and passes it to logNoticeHook.
func logNotify(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	if logNoticeHook != nil {
		logNoticeHook(msg)
	}
}

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
	fmt.Fprintln(os.Stderr, "  -log merge      Append the spooled audit records to the main log now, in any mode; it is not logged itself; add -noconfig to skip the settings file")
	fmt.Fprintln(os.Stderr, "  -noconfig       Do not read the settings file (default <user config folder>\\fedit\\fedit.ini, or the file named by FEDIT_CONFIG)")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Settings file command:")
	fmt.Fprintln(os.Stderr, "  fedit -config path|show|get KEY|set KEY VALUE|unset KEY   Show where the settings file is, list or read the effective values, or change a key (set and unset are logged; not available over MCP)")
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
	Parent                  string // id of the guided parent process, when there is one
	File, Files, Dest       string
	Steps                   int    // guided parent: number of steps
	Create                  string // guided parent: comma list of files it creates
	Text, TextHex, TextFile string
	Client, ClientVersion   string
	ConfigKey, ConfigValue  string // -config set and unset: the key and the value
	OutputFile              string // MCP output_file: the result file the call writes to
	OutputAppend            bool   // MCP output_append: append to that file
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
// host, pid, cwd, then parent (a child of a guided run), file, files and dest
// (absolute, only when given), steps and create (a guided parent), then the
// text (length and SHA-256, or verbatim when Content is set) and textfile (the path, then the length and SHA-256 of its content; the content is never logged).
// -texthex is hashed after decoding; an invalid one is hashed as typed.
func logCLIFields(in logCLIInput) []logKV {
	kvs := []logKV{
		{"source", in.Source},
		{"user", in.User},
		{"host", in.Host},
		{"pid", strconv.Itoa(in.Pid)},
		{"cwd", in.Cwd},
	}
	if in.Parent != "" {
		kvs = append(kvs, logKV{"parent", in.Parent})
	}
	for _, p := range []struct{ key, spec string }{{"file", in.File}, {"files", in.Files}, {"dest", in.Dest}} {
		if v := logAbsList(p.spec, in.Cwd); v != "" {
			kvs = append(kvs, logKV{p.key, v})
		}
	}
	if in.Steps > 0 {
		kvs = append(kvs, logKV{"steps", strconv.Itoa(in.Steps)})
	}
	if v := logAbsList(in.Create, in.Cwd); v != "" {
		kvs = append(kvs, logKV{"create", v})
	}
	if in.Client != "" {
		kvs = append(kvs, logKV{"client", in.Client})
		if in.ClientVersion != "" {
			kvs = append(kvs, logKV{"client_version", in.ClientVersion})
		}
	}
	if v := logAbsList(in.OutputFile, in.Cwd); v != "" {
		kvs = append(kvs, logKV{"output_file", v})
		if in.OutputAppend {
			kvs = append(kvs, logKV{"output_append", "true"})
		}
	}
	if in.ConfigKey != "" {
		kvs = append(kvs, logKV{"config_key", in.ConfigKey})
		if in.ConfigValue != "" {
			kvs = append(kvs, logKV{"config_value", in.ConfigValue})
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
	// The content of a -textfile is never logged, only its length and SHA-256.
	if v := logAbsList(in.TextFile, in.Cwd); v != "" {
		if b, err := os.ReadFile(v); err == nil {
			kvs = append(kvs, logTextFields("textfile", b)...)
		}
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

// logFlagString reads a -log-* flag value; a flag that was never registered
// (a guided run starts logging before flag.Parse) reads as not set.
func logFlagString(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// logStart starts the logging of a command-line run from the -log-* flags and
// the FEDIT_LOG_* environment. See logStartWith.
func logStart(op string, in logCLIInput) int {
	return logStartWith(logInputs{
		Mode:    logFlagString(logFlagVals.Mode),
		File:    logFlagString(logFlagVals.File),
		MaxMB:   logFlagString(logFlagVals.MaxMB),
		Content: logFlagString(logFlagVals.Content),
		Level:   logFlagString(logFlagVals.Level),
		Scope:   logFlagString(logFlagVals.Scope),
	}, op, in)
}

// logStartGuided starts the logging of a guided parent process, the one that
// creates files and runs one child fedit per step. It is configured from the
// environment only, since a guided command line has no -log-* flags.
func logStartGuided(steps []guidedStep) int {
	var create []string
	for _, st := range steps {
		create = append(create, st.create...)
	}
	logSourceGuided()
	return logStartWith(logInputs{}, "guided", logCLIInput{Steps: len(steps), Create: strings.Join(create, ",")})
}

// logChildEnv is the environment for a child fedit process: the parent's own,
// plus the id of this run, so the child logs parent=<id>. It is nil (inherit)
// when this run is not logged.
func logChildEnv() []string {
	if logRun == nil {
		return nil
	}
	return append(os.Environ(), logParentEnv+"="+logRun.ID())
}

// logStartWith resolves the logging settings from the flag layer and the
// FEDIT_LOG_* environment, starts the session and writes the begin record. It
// returns 0 to go on, 1 for an invalid setting, or exitLogRefused when strict
// mode could not write the begin record; in both cases nothing has changed. A
// process that finds FEDIT_LOG_PARENT in its environment is a step of a guided
// run: it logs source=guided and parent=<id>.
func logStartWith(flags logInputs, op string, in logCLIInput) int {
	home, _ := os.UserHomeDir()
	ini, ok := cfgLayerRun(os.Getenv, home, os.UserConfigDir, os.Stderr)
	if !ok {
		return 1
	}
	settings, err := resolveLogSettings(os.Getenv, home, os.UserConfigDir, flags, logInputsFromEnv(os.Getenv), ini)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fedit: %v\n", err)
		return 1
	}
	if !newLogSession(settings, op, time.Now, fileLogIO(), func() {}, nil).Active() {
		return 0
	}

	in.Source = logSourceName
	if p := strings.TrimSpace(os.Getenv(logParentEnv)); p != "" {
		in.Parent, in.Source = p, "guided"
	}
	in.Content = settings.Content
	in.User, in.Host, in.Pid, in.Cwd = logIdentity()
	notice := func() {
		logNotify("fedit: The action was not logged: the audit log could not be written, so the records were saved in the spool and the next logged run will merge them into the log")
	}
	ls := newLogSession(settings, op, time.Now, fileLogIO(), notice, logCLIFields(in))
	if err := ls.Begin(); err != nil {
		if ls.MustAbort(err) {
			fmt.Fprintf(os.Stderr, "fedit: log_mode=strict: cannot write the audit log (%v); nothing was changed\n", err)
			return exitLogRefused
		}
		logNotify(fmt.Sprintf("fedit: The action was not logged: %v", err))
		logWarned = true
	}
	logRun, logRunStrict = ls, settings.Mode == logStrict
	logHashStart(op, in)
	if dryRun {
		logNote("dry", "true")
	}
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
	logHashFinish()
	err := logRun.End(code, logTakeNotes())
	if err != nil {
		switch {
		case logRunStrict:
			fmt.Fprintf(os.Stderr, "fedit: log_mode=strict: the end record could not be written (%v); the audit log has a gap for this run\n", err)
		case !logWarned:
			logNotify(fmt.Sprintf("fedit: The action was not logged: %v", err))
		}
	}
	return logFinishCode(code, err, logRunStrict)
}
