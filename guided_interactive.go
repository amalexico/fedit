package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var (
	guidedQVerbs   = []string{"copy", "move", "replace", "insertbefore", "insertafter"}
	guidedQSources = []string{"file", "text"}
	guidedQPos     = []string{"before", "after", "at-end"}
)

// guidedMenu renders options as "1) copy  2) move ...".
func guidedMenu(opts []string) string {
	parts := make([]string, len(opts))
	for i, o := range opts {
		parts[i] = strconv.Itoa(i+1) + ") " + o
	}
	return strings.Join(parts, "  ")
}

// guidedInteractive asks the questions of a bare "fedit guided" and returns the
// words planGuided takes. Prompts go to out, answers come from in, one per line.
// Nothing is changed here: a failed or cut-short session just returns an error.
func guidedInteractive(in io.Reader, out io.Writer) ([]string, error) {
	sc := bufio.NewScanner(in)
	errEnd := fmt.Errorf("input ended before the last answer, nothing changed")
	var failed error

	line := func(prompt string) (string, bool) {
		fmt.Fprint(out, prompt+" ")
		if !sc.Scan() {
			fmt.Fprintln(out)
			return "", false
		}
		return strings.TrimSpace(sc.Text()), true
	}
	// get asks until the answer is not blank.
	get := func(prompt string) string {
		if failed != nil {
			return ""
		}
		for try := 0; try < 3; try++ {
			s, ok := line(prompt)
			if !ok {
				failed = errEnd
				return ""
			}
			if s != "" {
				return s
			}
			fmt.Fprintln(out, "an answer is required")
		}
		failed = fmt.Errorf("no answer after 3 tries, nothing changed")
		return ""
	}
	// maybe takes a blank answer as "not given".
	maybe := func(prompt string) string {
		if failed != nil {
			return ""
		}
		s, ok := line(prompt)
		if !ok {
			failed = errEnd
		}
		return s
	}
	// pick asks until the answer is one of opts, by name or by number.
	pick := func(prompt string, opts []string) string {
		if failed != nil {
			return ""
		}
		for try := 0; try < 3; try++ {
			s, ok := line(prompt + " " + guidedMenu(opts) + ":")
			if !ok {
				failed = errEnd
				return ""
			}
			if n, err := strconv.Atoi(s); err == nil && n >= 1 && n <= len(opts) {
				return opts[n-1]
			}
			for _, o := range opts {
				if strings.EqualFold(s, o) {
					return o
				}
			}
			fmt.Fprintln(out, "please answer with a name or a number from the list")
		}
		failed = fmt.Errorf("no valid answer after 3 tries, nothing changed")
		return ""
	}

	var args []string
	verb := pick("Operation", guidedQVerbs)
	switch verb {
	case "insertbefore", "insertafter":
		where := strings.TrimPrefix(verb, "insert")
		if pick("Insert from", guidedQSources) == "text" {
			text := get("Text to insert:")
			file := get("File to insert into:")
			target := get("Block in that file to insert " + where + ":")
			args = []string{text, verb, file, target}
		} else {
			src := get("Source file:")
			blk := get("Source block name, or \"line 12-18\":")
			dest := get("Destination file:")
			target := get("Block in the destination to insert " + where + ":")
			args = []string{src, blk, verb, dest, target}
		}
	case "replace":
		src := get("Source file:")
		blk := get("Source block name, or \"line 12-18\":")
		dest := get("Destination file:")
		target := maybe("Block in the destination to replace (blank = same name as the source):")
		args = []string{src, blk, verb, dest}
		if target != "" {
			args = append(args, target)
		}
	default:
		src := get("Source file:")
		blk := get("Source block name, or \"line 12-18\":")
		dest := get("Destination file (several: a comma list):")
		pos := pick("Position in the destination", guidedQPos)
		args = []string{src, blk, verb, dest, pos}
		if pos != "at-end" {
			args = append(args, get("Block in the destination to put it "+pos+":"))
		}
	}
	if failed != nil {
		return nil, failed
	}

	for try := 0; try < 3; try++ {
		s, ok := line("Modifier (blank for none, or silent, report-only):")
		if !ok {
			return args, nil
		}
		switch strings.ToLower(s) {
		case "", "none":
			return args, nil
		case "silent", "report-only":
			return append(args, strings.ToLower(s)), nil
		}
		fmt.Fprintln(out, "please answer blank, silent or report-only")
	}
	return nil, fmt.Errorf("no valid answer after 3 tries, nothing changed")
}
