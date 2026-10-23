package main

import (
	"fmt"
	"strconv"
	"strings"
)

// The settings file (an .ini): a flat list of "key = value" lines with # or ;
// comment lines. This file holds the key registry and the pure text functions
// (parse, check, set, unset); locating and writing the file, and the -config
// command, are built on top of it.

// cfgKey describes one setting: its name in the ini, its kind, its
// environment variable and flag, and whether the MCP server may read it.
type cfgKey struct {
	Name    string
	Kind    string   // "enum", "uint" or "path"
	Allowed []string // the values of an "enum"
	Env     string
	Flag    string
	MCP     bool
}

var cfgKeys = []cfgKey{
	{"log_mode", "enum", []string{"off", "permissive", "strict"}, "FEDIT_LOG_MODE", "-log-mode", true},
	{"log_file", "path", nil, "FEDIT_LOG_FILE", "-log-file", true},
	{"log_max_mb", "uint", nil, "FEDIT_LOG_MAX_MB", "-log-max-mb", true},
	{"log_content", "enum", []string{"on", "off"}, "FEDIT_LOG_CONTENT", "-log-content", true},
	{"log_level", "enum", []string{"all", "warning", "error"}, "FEDIT_LOG_LEVEL", "-log-level", true},
	{"log_scope", "enum", []string{"all", "writes"}, "FEDIT_LOG_SCOPE", "-log-scope", true},
}

func cfgNorm(name string) string { return strings.ToLower(strings.TrimSpace(name)) }

// cfgFind returns the registry entry of a key name (case-insensitive).
func cfgFind(name string) (cfgKey, bool) {
	n := cfgNorm(name)
	for _, k := range cfgKeys {
		if k.Name == n {
			return k, true
		}
	}
	return cfgKey{}, false
}

// cfgCheck validates a value for a key.
func cfgCheck(name, value string) error {
	k, ok := cfgFind(name)
	if !ok {
		return fmt.Errorf("unknown key %q", strings.TrimSpace(name))
	}
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%s: the value must be on one line", k.Name)
	}
	value = strings.TrimSpace(value)
	switch k.Kind {
	case "enum":
		for _, a := range k.Allowed {
			if value == a {
				return nil
			}
		}
		return fmt.Errorf("%s: invalid value %q (want %s)", k.Name, value, strings.Join(k.Allowed, ", "))
	case "uint":
		if _, err := strconv.ParseUint(value, 10, 32); err != nil {
			return fmt.Errorf("%s: invalid value %q (want a whole number, 0 or more)", k.Name, value)
		}
	case "path":
		if value == "" {
			return fmt.Errorf("%s: the value must not be empty", k.Name)
		}
	}
	return nil
}

// cfgKeyOfLine returns the normalised key of a "key = value" line, or "" for
// a blank line, a comment or a line without "=".
func cfgKeyOfLine(line string) string {
	t := strings.TrimSpace(line)
	if t == "" || t[0] == '#' || t[0] == ';' {
		return ""
	}
	i := strings.Index(t, "=")
	if i < 0 {
		return ""
	}
	return cfgNorm(t[:i])
}

// cfgParse reads the text of an ini. It returns the values of the known keys
// (the last of a repeated key wins) and a warning for each unknown key. An
// invalid value or a line that is not "key = value" is an error naming the line.
func cfgParse(text string) (map[string]string, []string, error) {
	vals := map[string]string{}
	var warns []string
	for i, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || line[0] == '#' || line[0] == ';' {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			return nil, nil, fmt.Errorf("line %d: expected key = value", i+1)
		}
		key, val := cfgNorm(line[:eq]), strings.TrimSpace(line[eq+1:])
		if key == "" {
			return nil, nil, fmt.Errorf("line %d: the key is empty", i+1)
		}
		if _, ok := cfgFind(key); !ok {
			warns = append(warns, fmt.Sprintf("line %d: unknown key %q (ignored)", i+1, key))
			continue
		}
		if err := cfgCheck(key, val); err != nil {
			return nil, nil, fmt.Errorf("line %d: %v", i+1, err)
		}
		vals[key] = val
	}
	return vals, warns, nil
}

// cfgSet returns the text with key set to value. The value is validated first.
// An existing line (the last one of a repeated key) is replaced in place; a new
// key is appended. Comments, blank lines and the line ending style are kept.
func cfgSet(text, name, value string) (string, error) {
	if err := cfgCheck(name, value); err != nil {
		return text, err
	}
	key, value := cfgNorm(name), strings.TrimSpace(value)
	lines := strings.Split(text, "\n")
	at := -1
	for i, l := range lines {
		if cfgKeyOfLine(l) == key {
			at = i
		}
	}
	if at >= 0 {
		cr := ""
		if strings.HasSuffix(lines[at], "\r") {
			cr = "\r"
		}
		lines[at] = key + " = " + value + cr
		return strings.Join(lines, "\n"), nil
	}
	eol := "\n"
	if strings.Contains(text, "\r\n") {
		eol = "\r\n"
	}
	switch {
	case text == "":
		return key + " = " + value + eol, nil
	case strings.HasSuffix(text, "\n"):
		return text + key + " = " + value + eol, nil
	}
	return text + eol + key + " = " + value + eol, nil
}

// cfgUnset removes every line of a key. found is false, and the text is
// returned unchanged, when the key has no line.
func cfgUnset(text, name string) (string, bool) {
	key := cfgNorm(name)
	var keep []string
	found := false
	for _, l := range strings.Split(text, "\n") {
		if cfgKeyOfLine(l) == key {
			found = true
			continue
		}
		keep = append(keep, l)
	}
	if !found {
		return text, false
	}
	return strings.Join(keep, "\n"), true
}

// cfgLogInputs turns parsed ini values into the ini layer of the logging
// settings (the layer between the environment and the defaults).
func cfgLogInputs(vals map[string]string) logInputs {
	return logInputs{
		Mode:    vals["log_mode"],
		File:    vals["log_file"],
		MaxMB:   vals["log_max_mb"],
		Content: vals["log_content"],
		Level:   vals["log_level"],
		Scope:   vals["log_scope"],
	}
}
