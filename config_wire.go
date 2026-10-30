package main

import (
	"flag"
	"fmt"
	"io"
)

// Wiring of the settings file (fedit.ini) into a run: the -noconfig flag, and
// the ini layer that sits between the environment and the defaults in the
// logging settings.

// logFlagNoConfig is the -noconfig flag. It is nil until logRegisterFlags has
// run, so guided runs and the MCP server, which have no
// flags, always read the settings file.
var logFlagNoConfig *bool

// cfgRegisterFlags defines -noconfig. Call it before flag.Parse.
func cfgRegisterFlags() {
	logFlagNoConfig = flag.Bool("noconfig", false, "Do not read the settings file (default <user config folder>/fedit/fedit.ini; env FEDIT_CONFIG names another file)")
}

// cfgLayerRun reads the settings file and returns its layer of the logging
// settings. Unknown keys are reported on errOut as warnings. ok is false when
// the file is unreadable or invalid, after the reason has been written to
// errOut; the run must then stop before it changes anything.
func cfgLayerRun(getenv func(string) string, home string, userConfigDir func() (string, error), errOut io.Writer) (layer logInputs, ok bool) {
	if logFlagNoConfig != nil && *logFlagNoConfig {
		return logInputs{}, true
	}
	r, err := cfgLoad(getenv, home, userConfigDir)
	if err != nil {
		fmt.Fprintf(errOut, "fedit: %v\n", err)
		return logInputs{}, false
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(errOut, "fedit: settings file %s: %s\n", r.Path, w)
	}
	return r.Inputs, true
}
