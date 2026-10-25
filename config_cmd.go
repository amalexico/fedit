package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// The -config command: path, show, get, set and unset for the settings file.

const cfgUsageText = "usage: fedit -config path | show | get KEY | set KEY VALUE | unset KEY"

// cfgBOM is the UTF-8 byte order mark that Notepad puts at the start of a file.
const cfgBOM = "\ufeff"

// cfgWriteFile writes text to path atomically: a temp file in the same
// folder, then a rename over the target. The folder is created when needed;
// a failed write leaves no temp file behind.
func cfgWriteFile(path, text string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".fedit-ini-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	_, werr := tmp.WriteString(text)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		os.Remove(name)
		if werr != nil {
			return werr
		}
		return cerr
	}
	if err := os.Rename(name, path); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// cfgInputValue is the value an ini layer holds for a key.
func cfgInputValue(in logInputs, name string) string {
	switch name {
	case "log_mode":
		return in.Mode
	case "log_file":
		return in.File
	case "log_max_mb":
		return in.MaxMB
	case "log_content":
		return in.Content
	case "log_level":
		return in.Level
	case "log_scope":
		return in.Scope
	}
	return ""
}

// cfgDefaultValue is what a key is when nothing sets it.
func cfgDefaultValue(name string, getenv func(string) string, userConfigDir func() (string, error)) string {
	switch name {
	case "log_mode", "log_content":
		return "off"
	case "log_file":
		if p, err := defaultLogPath(getenv, userConfigDir); err == nil {
			return p
		}
		return ""
	case "log_max_mb":
		return strconv.Itoa(logDefaultMaxMB)
	case "log_level", "log_scope":
		return "all"
	}
	return ""
}

type cfgRow struct{ Key, Value, Source string }

// cfgEffective returns every key with its effective value and where it comes
// from (environment, ini or default). A settings file that cannot be used, or
// an invalid environment value, is reported on errOut and ok is false.
func cfgEffective(getenv func(string) string, home string, userConfigDir func() (string, error), errOut io.Writer) ([]cfgRow, bool) {
	r, err := cfgLoad(getenv, home, userConfigDir)
	if err != nil {
		fmt.Fprintf(errOut, "fedit: %v\n", err)
		return nil, false
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(errOut, "fedit: settings file %s: %s\n", r.Path, w)
	}
	var rows []cfgRow
	for _, k := range cfgKeys {
		if v := strings.TrimSpace(getenv(k.Env)); v != "" {
			if k.Kind == "enum" {
				v = strings.ToLower(v) // runs accept FEDIT_LOG_MODE=Strict
			}
			if err := cfgCheck(k.Name, v); err != nil {
				fmt.Fprintf(errOut, "fedit: %s: %v\n", k.Env, err)
				return nil, false
			}
			rows = append(rows, cfgRow{k.Name, v, "environment"})
			continue
		}
		if v := cfgInputValue(r.Inputs, k.Name); v != "" {
			rows = append(rows, cfgRow{k.Name, v, "ini"})
			continue
		}
		rows = append(rows, cfgRow{k.Name, cfgDefaultValue(k.Name, getenv, userConfigDir), "default"})
	}
	return rows, true
}

// cfgEditable loads the settings file for set and unset: its path, its text
// without a leading byte order mark, and whether it had one. A file that is
// unreadable or invalid stops the edit.
func cfgEditable(getenv func(string) string, home string, userConfigDir func() (string, error), errOut io.Writer) (path, text string, bom, ok bool) {
	r, err := cfgLoad(getenv, home, userConfigDir)
	if err != nil {
		fmt.Fprintf(errOut, "fedit: %v\n", err)
		return "", "", false, false
	}
	if r.Path == "" {
		fmt.Fprintln(errOut, "fedit: cannot determine the settings file location; set FEDIT_CONFIG")
		return "", "", false, false
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(errOut, "fedit: settings file %s: %s\n", r.Path, w)
	}
	data, err := os.ReadFile(r.Path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(errOut, "fedit: settings file %s: %v\n", r.Path, err)
		return "", "", false, false
	}
	text = string(data)
	if strings.HasPrefix(text, cfgBOM) {
		bom, text = true, strings.TrimPrefix(text, cfgBOM)
	}
	return r.Path, text, bom, true
}

// configCommand runs "fedit -config ARGS". Results go to out, messages and
// errors to errOut. It returns the exit code.
func configCommand(args []string, getenv func(string) string, home string, userConfigDir func() (string, error), out, errOut io.Writer) int {
	usage := func() int {
		fmt.Fprintln(errOut, cfgUsageText)
		return 1
	}
	if len(args) == 0 {
		return usage()
	}
	switch args[0] {
	case "path":
		if len(args) != 1 {
			return usage()
		}
		p := cfgPath(getenv, home, userConfigDir)
		if p == "" {
			fmt.Fprintln(errOut, "fedit: cannot determine the settings file location; set FEDIT_CONFIG")
			return 1
		}
		fmt.Fprintln(out, p)
		return 0
	case "show":
		if len(args) != 1 {
			return usage()
		}
		rows, ok := cfgEffective(getenv, home, userConfigDir, errOut)
		if !ok {
			return 1
		}
		for _, r := range rows {
			v := r.Value
			if v == "" {
				v = "(none)"
			}
			fmt.Fprintf(out, "%s = %s (%s)\n", r.Key, v, r.Source)
		}
		return 0
	case "get":
		if len(args) != 2 {
			return usage()
		}
		k, found := cfgFind(args[1])
		if !found {
			fmt.Fprintf(errOut, "fedit: unknown key %q\n", strings.TrimSpace(args[1]))
			return 1
		}
		rows, ok := cfgEffective(getenv, home, userConfigDir, errOut)
		if !ok {
			return 1
		}
		for _, r := range rows {
			if r.Key == k.Name {
				fmt.Fprintln(out, r.Value)
			}
		}
		return 0
	case "set":
		if len(args) != 3 {
			return usage()
		}
		if err := cfgCheck(args[1], args[2]); err != nil {
			fmt.Fprintf(errOut, "fedit: %v\n", err)
			return 1
		}
		path, text, bom, ok := cfgEditable(getenv, home, userConfigDir, errOut)
		if !ok {
			return 1
		}
		next, err := cfgSet(text, args[1], args[2])
		if err != nil {
			fmt.Fprintf(errOut, "fedit: %v\n", err)
			return 1
		}
		if bom {
			next = cfgBOM + next
		}
		if err := cfgWriteFile(path, next); err != nil {
			fmt.Fprintf(errOut, "fedit: settings file %s: %v\n", path, err)
			return 1
		}
		fmt.Fprintf(out, "Set %s = %s in %s\n", cfgNorm(args[1]), strings.TrimSpace(args[2]), path)
		return 0
	case "unset":
		if len(args) != 2 {
			return usage()
		}
		k, found := cfgFind(args[1])
		if !found {
			fmt.Fprintf(errOut, "fedit: unknown key %q\n", strings.TrimSpace(args[1]))
			return 1
		}
		path, text, bom, ok := cfgEditable(getenv, home, userConfigDir, errOut)
		if !ok {
			return 1
		}
		next, removed := cfgUnset(text, k.Name)
		if !removed {
			fmt.Fprintf(out, "%s is not set in %s\n", k.Name, path)
			return 0
		}
		if bom {
			next = cfgBOM + next
		}
		if err := cfgWriteFile(path, next); err != nil {
			fmt.Fprintf(errOut, "fedit: settings file %s: %v\n", path, err)
			return 1
		}
		fmt.Fprintf(out, "Removed %s from %s\n", k.Name, path)
		return 0
	}
	return usage()
}

// cfgLogOp returns the logged operation for a set or unset command line that
// has the right number of arguments, with the fields of its begin record. Any
// other command line (path, show, get, a usage error) is not an operation.
func cfgLogOp(args []string) (string, logCLIInput) {
	switch {
	case len(args) == 3 && args[0] == "set":
		return "config-set", logCLIInput{ConfigKey: args[1], ConfigValue: args[2]}
	case len(args) == 2 && args[0] == "unset":
		return "config-unset", logCLIInput{ConfigKey: args[1]}
	}
	return "", logCLIInput{}
}

// runConfigCommand is "fedit -config ARGS" for the real process: set and
// unset are logged like any other write operation, the rest is not.
func runConfigCommand(args []string) int {
	home, _ := os.UserHomeDir()
	if op, in := cfgLogOp(args); op != "" {
		if code := logStartWith(logInputs{}, op, in); code != 0 {
			return code
		}
	}
	code := configCommand(args, os.Getenv, home, os.UserConfigDir, os.Stdout, os.Stderr)
	return logFinish(code)
}
