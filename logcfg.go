package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Resolution of the six logging settings (log_mode, log_file, log_max_mb,
// log_content, log_level, log_scope) from their layers: command-line flags,
// then environment variables, later the ini file, then the built-in defaults.
// Nothing here touches the file system or is wired into the operations yet.

// logDefaultMaxMB is the rotation size used when log_max_mb is not set.
const logDefaultMaxMB = 10

// logThreshold is the log_level setting: which severities are written.
// all writes INFO, WARNING and ERROR; warning writes WARNING and ERROR;
// error writes ERROR only.
type logThreshold string

const (
	logThresholdAll     logThreshold = "all"
	logThresholdWarning logThreshold = "warning"
	logThresholdError   logThreshold = "error"
)

// logScope is the log_scope setting: which operations are logged. all logs
// every operation, reads included; writes logs only the operations that can
// change a file.
type logScope string

const (
	logScopeAll    logScope = "all"
	logScopeWrites logScope = "writes"
)

// parseLogThreshold reads a log_level value. An empty value means all, and
// the plural forms warnings and errors are accepted.
func parseLogThreshold(s string) (logThreshold, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all":
		return logThresholdAll, nil
	case "warning", "warnings":
		return logThresholdWarning, nil
	case "error", "errors":
		return logThresholdError, nil
	}
	return logThresholdAll, fmt.Errorf("invalid log_level %q: use all, warning or error", s)
}

// parseLogScope reads a log_scope value. An empty value means all.
func parseLogScope(s string) (logScope, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "all":
		return logScopeAll, nil
	case "writes":
		return logScopeWrites, nil
	}
	return logScopeAll, fmt.Errorf("invalid log_scope %q: use all or writes", s)
}

// logInputs holds one layer's raw values. An empty or blank string means the
// layer does not set that value.
type logInputs struct {
	Mode    string
	File    string
	MaxMB   string
	Content string
	Level   string
	Scope   string
}

// logSettings is the validated result.
type logSettings struct {
	Mode     logMode
	File     string // absolute log path; empty when it cannot be determined and Mode is off
	MaxMB    int    // 0 = never rotate
	Content  bool   // log_content=on: log text values instead of length + SHA-256
	Level    logThreshold
	Scope    logScope
	SpoolDir string // where permissive mode spools records the main log refused
}

// logInputsFromEnv reads the FEDIT_LOG_* environment variables.
func logInputsFromEnv(getenv func(string) string) logInputs {
	return logInputs{
		Mode:    getenv("FEDIT_LOG_MODE"),
		File:    getenv("FEDIT_LOG_FILE"),
		MaxMB:   getenv("FEDIT_LOG_MAX_MB"),
		Content: getenv("FEDIT_LOG_CONTENT"),
		Level:   getenv("FEDIT_LOG_LEVEL"),
		Scope:   getenv("FEDIT_LOG_SCOPE"),
	}
}

// logPick returns the first non-blank value among the layers, highest
// priority first, or "" when none sets it.
func logPick(layers []logInputs, get func(logInputs) string) string {
	for _, l := range layers {
		if v := strings.TrimSpace(get(l)); v != "" {
			return v
		}
	}
	return ""
}

// resolveLogSettings merges the layers (first layer wins, per value), checks
// every value and fills in the defaults. An invalid value is an error even
// when the mode is off, or when strict would override it. In strict mode the
// level is always all: strict records every operation before it happens,
// which a severity filter would defeat. The log file defaults to the per-user
// location; it is an error only when the mode is on and no location can be
// determined. The spool folder is "spool" in the per-user folder, or next to
// the log file when there is no per-user folder.
func resolveLogSettings(getenv func(string) string, home string, userConfigDir func() (string, error), layers ...logInputs) (logSettings, error) {
	var s logSettings

	mode, err := parseLogMode(logPick(layers, func(l logInputs) string { return l.Mode }))
	if err != nil {
		return s, err
	}
	s.Mode = mode

	s.MaxMB = logDefaultMaxMB
	if v := logPick(layers, func(l logInputs) string { return l.MaxMB }); v != "" {
		n, perr := strconv.Atoi(v)
		if perr != nil || n < 0 {
			return s, fmt.Errorf("invalid log_max_mb %q: use a whole number of megabytes, or 0 for no rotation", v)
		}
		s.MaxMB = n
	}

	switch v := logPick(layers, func(l logInputs) string { return l.Content }); strings.ToLower(v) {
	case "", "off":
	case "on":
		s.Content = true
	default:
		return s, fmt.Errorf("invalid log_content %q: use on or off", v)
	}

	level, err := parseLogThreshold(logPick(layers, func(l logInputs) string { return l.Level }))
	if err != nil {
		return s, err
	}
	if s.Mode == logStrict {
		level = logThresholdAll
	}
	s.Level = level

	scope, err := parseLogScope(logPick(layers, func(l logInputs) string { return l.Scope }))
	if err != nil {
		return s, err
	}
	s.Scope = scope

	file := logPick(layers, func(l logInputs) string { return l.File })
	if file == "" {
		p, perr := defaultLogPath(getenv, userConfigDir)
		if perr != nil {
			if s.Mode != logOff {
				return s, perr
			}
		} else {
			file = p
		}
	} else {
		file = expandLogPath(file, home, getenv)
		if abs, aerr := filepath.Abs(file); aerr == nil {
			file = abs
		}
	}
	s.File = file

	if base, berr := logBaseDir(getenv, userConfigDir); berr == nil {
		s.SpoolDir = filepath.Join(base, "spool")
	} else if file != "" {
		s.SpoolDir = filepath.Join(filepath.Dir(file), "spool")
	}
	return s, nil
}
