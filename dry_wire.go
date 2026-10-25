package main

import "flag"

// dryFlag is the -dry flag. It is nil until dryRegisterFlags has run.
var dryFlag *bool

// dryRegisterFlags defines -dry. Call it before flag.Parse.
func dryRegisterFlags() {
	dryFlag = flag.Bool("dry", false, "Dry run: print a unified diff of what the operation would change and write nothing; the report says applied=false")
}

// dryApply switches dry-run on when -dry was given. Call it right after
// flag.Parse, before anything is written.
func dryApply() {
	if dryFlag != nil && *dryFlag {
		dryRun = true
	}
}
