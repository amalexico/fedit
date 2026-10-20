package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The real file operations behind the audit log sink, plus path handling for
// the log file. Nothing here is wired into the operations yet.

// logAppendFile adds one line plus a newline (LF on every platform) to the
// file, creating the file and its parent directories when needed. The line is
// written with a single write call on a file opened in append mode.
func logAppendFile(path, line string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	_, err = f.WriteString(line + "\n")
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

// logFileSize returns the size of the log file; a missing file is an error
// that satisfies errors.Is(err, os.ErrNotExist).
func logFileSize(path string) (int64, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.Size(), nil
}

// logRotateFileAt renames the log file to <path>.<UTC stamp> (for example
// a.log.20261018T081715Z) and returns the new name. An existing rotated file
// is never overwritten: a numeric suffix .1, .2, ... is added instead.
func logRotateFileAt(path string, t time.Time) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	base := path + "." + t.UTC().Format("20060102T150405Z")
	target := base
	for i := 1; ; i++ {
		_, err := os.Lstat(target)
		if errors.Is(err, os.ErrNotExist) {
			break
		}
		if err != nil {
			return "", err
		}
		if i > 1000 {
			return "", fmt.Errorf("too many rotated files named %s.*", base)
		}
		target = fmt.Sprintf("%s.%d", base, i)
	}
	if err := os.Rename(path, target); err != nil {
		return "", err
	}
	return target, nil
}

func logRotateFile(path string) error {
	_, err := logRotateFileAt(path, time.Now())
	return err
}

// fileLogIO is the logIO the sink uses on a real log file.
func fileLogIO() logIO {
	return logIO{Append: logAppendFile, Size: logFileSize, Rotate: logRotateFile}
}

func logEnvNameOK(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '(', r == ')', r == '.', r == '-':
		default:
			return false
		}
	}
	return true
}

// expandLogPercent replaces %NAME% with the value of the environment
// variable. An unset or empty variable, an invalid name or a lone % is left
// exactly as written.
func expandLogPercent(p string, getenv func(string) string) string {
	var b strings.Builder
	for i := 0; i < len(p); {
		if p[i] != '%' {
			b.WriteByte(p[i])
			i++
			continue
		}
		j := strings.IndexByte(p[i+1:], '%')
		if j < 0 {
			b.WriteString(p[i:])
			break
		}
		j += i + 1
		if logEnvNameOK(p[i+1 : j]) {
			if v := getenv(p[i+1 : j]); v != "" {
				b.WriteString(v)
			} else {
				b.WriteString(p[i : j+1])
			}
			i = j + 1
			continue
		}
		b.WriteByte('%')
		i++
	}
	return b.String()
}

// expandLogPath expands %VAR% from the environment and a leading ~, ~/ or ~\
// to the home directory. Without a home directory the ~ is left alone, and
// ~name is never expanded.
func expandLogPath(p, home string, getenv func(string) string) string {
	p = expandLogPercent(p, getenv)
	if home == "" {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(home, p[2:])
	}
	return p
}

// logBaseDir is the per-user folder fedit keeps its log and spool in:
// %LOCALAPPDATA%\fedit, or the user config directory when that is not set.
func logBaseDir(getenv func(string) string, userConfigDir func() (string, error)) (string, error) {
	if la := getenv("LOCALAPPDATA"); la != "" {
		return filepath.Join(la, "fedit"), nil
	}
	if userConfigDir != nil {
		if d, err := userConfigDir(); err == nil && d != "" {
			return filepath.Join(d, "fedit"), nil
		}
	}
	return "", errors.New("cannot determine a default log location; set log_file")
}

// defaultLogPath is the log file used when log_file is not configured.
func defaultLogPath(getenv func(string) string, userConfigDir func() (string, error)) (string, error) {
	dir, err := logBaseDir(getenv, userConfigDir)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "fedit.log"), nil
}
