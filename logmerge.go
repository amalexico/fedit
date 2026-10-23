package main

import (
	"fmt"
	"io"
	"os"
)

// The "fedit -log merge" command: append the records the audit log could not
// write (they wait in the spool folder) to the main log now, in any log mode,
// instead of waiting for the next logged run. It uses the same settings as a
// logged run, from the FEDIT_LOG_* environment. It is not an operation and is
// not logged itself.

// runLogCommand runs "fedit -log ARGS" and returns the exit code.
func runLogCommand(args []string) int {
	home, _ := os.UserHomeDir()
	return logCommand(args, os.Getenv, home, os.UserConfigDir, fileLogIO(), os.Stderr)
}

func logCommand(args []string, getenv func(string) string, home string, userConfigDir func() (string, error), lio logIO, out io.Writer) int {
	if len(args) != 1 || args[0] != "merge" {
		fmt.Fprintln(out, "usage: fedit -log merge   (append the spooled audit records to the main log now)")
		return 1
	}
	ini, ok := cfgLayerRun(getenv, home, userConfigDir, out)
	if !ok {
		return 1
	}
	s, err := resolveLogSettings(getenv, home, userConfigDir, logInputsFromEnv(getenv), ini)
	if err != nil {
		fmt.Fprintf(out, "fedit: %v\n", err)
		return 1
	}
	if s.File == "" || s.SpoolDir == "" {
		fmt.Fprintln(out, "fedit: cannot determine the log file or the spool folder; set FEDIT_LOG_FILE")
		return 1
	}
	n, err := mergeSpool(s.SpoolDir, s.File, lio)
	if err != nil {
		fmt.Fprintf(out, "fedit: the merge stopped after %d record(s): %v\n", n, err)
		return 1
	}
	if n == 0 {
		fmt.Fprintln(out, "No spooled records to merge")
		return 0
	}
	fmt.Fprintf(out, "Merged %d spooled record(s) into %s\n", n, s.File)
	return 0
}
