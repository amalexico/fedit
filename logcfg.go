package main

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Resolution of the four logging settings (log_mode, log_file, log_max_mb,
// log_content) from their layers: command-line flags, then environment
// variables, later the ini file, then the built-in defaults. Nothing here
// touches the file system or is wired into the operations yet.

// logDefaultMaxMB is the rotation size used when log_max_mb is not set.
const logDefaultMaxMB = 10

// logInputs holds one layer's raw values. An empty or blank string means the
// layer does not set that value.
type logInputs struct {
	Mode    string
	File    string
	MaxMB   string
	Content string
}

// logSettings is the validated result.
type logSettings struct {
	Mode     logMode
	File     string // absolute log path; empty when it cannot be determined and Mode is off
	MaxMB    int    // 0 = never rotate
	Content  bool   // log_content=on: log text values instead of length + SHA-256
	SpoolDir string // where permissive mode spools records the main log refused
}

// logInputsFromEnv reads the FEDIT_LOG_* environment variables.
func logInputsFromEnv(getenv func(string) string) logInputs {
	return logInputs{
		Mode:    getenv("FEDIT_LOG_MODE"),
		File:    getenv("FEDIT_LOG_FILE"),
		MaxMB:   getenv("FEDIT_LOG_MAX_MB"),
		Content: getenv("FEDIT_LOG_CONTENT"),
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
// when the mode is off. The log file defaults to the per-user location; it is
// an error only when the mode is on and no location can be determined. The
// spool folder is "spool" in the per-user folder, or next to the log file
// when there is no per-user folder.
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
