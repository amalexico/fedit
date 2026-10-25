package main

import "strings"

// isDryArg reports whether a is the -dry flag in a spelling the flag package
// accepts: -dry, --dry or -dry=VALUE.
func isDryArg(a string) bool {
	if !strings.HasPrefix(a, "-") {
		return false
	}
	n := strings.TrimPrefix(strings.TrimPrefix(a, "-"), "-")
	return n == "dry" || strings.HasPrefix(n, "dry=")
}

// dryConflict reports whether the command line (os.Args style, program name
// first) combines -dry with the -config or -log command. Those two commands
// are recognised by their first argument only, so the check looks for -dry
// anywhere after them, and for a leading -dry followed by one of them.
func dryConflict(args []string) bool {
	if len(args) < 2 {
		return false
	}
	switch args[1] {
	case "-log", "-config":
		for _, a := range args[2:] {
			if isDryArg(a) {
				return true
			}
		}
	default:
		if isDryArg(args[1]) && len(args) >= 3 && (args[2] == "-log" || args[2] == "-config") {
			return true
		}
	}
	return false
}
