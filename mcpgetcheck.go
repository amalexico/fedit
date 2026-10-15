package main

import (
	"fmt"
	"regexp"
)

// mcpCheckGet validates the get parameter of fedit_find the way the CLI does
// for -get, so an invalid regex is an error instead of a silent skip of every
// line. It returns an empty string when get is empty or valid.
func mcpCheckGet(get string) string {
	if get == "" {
		return ""
	}
	if _, err := regexp.Compile(get); err != nil {
		return fmt.Sprintf("get: invalid regex: %v", err)
	}
	return ""
}
