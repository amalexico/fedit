package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Locating and reading the settings file (fedit.ini). A missing file is not an
// error; an unreadable or invalid one is, so a typo never silently changes how
// fedit logs.

// cfgEnvPath is the environment variable that names the settings file.
const cfgEnvPath = "FEDIT_CONFIG"

// cfgLoaded is the result of reading the settings file.
type cfgLoaded struct {
	Path     string    // the file looked at; empty when no location could be determined
	Exists   bool      // false when there is no file at Path
	Inputs   logInputs // the ini layer of the logging settings
	Warnings []string  // unknown keys, each with its line
}

// cfgPath is where the settings file lives: FEDIT_CONFIG (with ~ and %VAR%
// expanded, made absolute), else fedit\fedit.ini in the user config folder.
// It is empty when neither can be determined.
func cfgPath(getenv func(string) string, home string, userConfigDir func() (string, error)) string {
	if p := strings.TrimSpace(getenv(cfgEnvPath)); p != "" {
		p = expandLogPath(p, home, getenv)
		if abs, err := filepath.Abs(p); err == nil {
			p = abs
		}
		return p
	}
	if userConfigDir != nil {
		if d, err := userConfigDir(); err == nil && d != "" {
			return filepath.Join(d, "fedit", "fedit.ini")
		}
	}
	return ""
}

// cfgLoad reads the settings file. A leading UTF-8 byte order mark (Notepad
// adds one) is ignored.
func cfgLoad(getenv func(string) string, home string, userConfigDir func() (string, error)) (cfgLoaded, error) {
	path := cfgPath(getenv, home, userConfigDir)
	if path == "" {
		return cfgLoaded{}, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfgLoaded{Path: path}, nil
	}
	if err != nil {
		return cfgLoaded{Path: path}, fmt.Errorf("settings file %s: %v", path, err)
	}
	vals, warns, perr := cfgParse(strings.TrimPrefix(string(data), "\ufeff"))
	if perr != nil {
		return cfgLoaded{Path: path, Exists: true}, fmt.Errorf("settings file %s: %v", path, perr)
	}
	return cfgLoaded{Path: path, Exists: true, Inputs: cfgLogInputs(vals), Warnings: warns}, nil
}
