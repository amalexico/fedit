package main

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// guided: natural-order front end. Translates words into a real explicit -op
// argv, echoes it to stderr, then main() falls through into the normal flag path.

var guidedLineRange = regexp.MustCompile(`(?i)^line\s+(\d+)-(\d+)$`)

const guidedUsage = "usage: fedit guided SRCFILE SRCBLOCK VERB DESTFILE [before|after TARGET | at-end] [silent]\n" +
	"       fedit guided SRCFILE SRCBLOCK replace DESTFILE [TARGET]\n" +
	"       fedit guided SRCFILE SRCBLOCK insertbefore|insertafter DESTFILE TARGET\n" +
	"       fedit guided \"new text\" insertbefore|insertafter FILE TARGET\n" +
	"verbs: copy, move, replace, insertbefore, insertafter. SRCBLOCK may be \"line 12-18\".\n" +
	"Several destinations: quote them as one comma list, e.g. \"b.go,c.go\"."

func isGuidedCall(args []string) bool {
	return len(args) >= 2 && args[1] == "guided"
}

func guidedQuote(s string) string {
	if s == "" || strings.ContainsAny(s, " \t\"") {
		return "\"" + strings.ReplaceAll(s, "\"", "`\"") + "\""
	}
	return s
}

func guidedDests(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "(")
	s = strings.TrimSuffix(s, ")")
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// guidedSource returns the source-side flags and the source language ("" when unknown).
func guidedSource(file, spec string) ([]string, string, error) {
	lang, _ := detectLang(file)
	if m := guidedLineRange.FindStringSubmatch(spec); m != nil {
		return []string{"-line", m[1], "-end", m[2]}, lang, nil
	}
	if lang == "" {
		return nil, "", fmt.Errorf("cannot detect the language of %s (needed for block %q)", file, spec)
	}
	return []string{"-block", spec}, lang, nil
}

// guidedBlockLang picks the -lang for block resolution and rejects a source/dest mismatch.
func guidedBlockLang(srcLang string, dests []string) (string, error) {
	lang := srcLang
	for _, d := range dests {
		dl, _ := detectLang(d)
		if dl == "" {
			return "", fmt.Errorf("cannot detect the language of %s", d)
		}
		if lang == "" {
			lang = dl
		}
		if dl != lang {
			return "", fmt.Errorf("%s is %s but the block language is %s; guided needs one language per command", d, dl, lang)
		}
	}
	return lang, nil
}

func guidedTransfer(args []string) ([]string, []string, error) {
	src, spec, verb := args[0], args[1], args[2]
	dests := guidedDests(args[3])
	if len(dests) == 0 {
		return nil, nil, fmt.Errorf("no destination file given")
	}
	srcFlags, lang, err := guidedSource(src, spec)
	if err != nil {
		return nil, nil, err
	}
	rest := args[4:]
	pos := ""
	if len(rest) > 0 {
		pos = rest[0]
	}
	var notes, posFlags []string
	switch pos {
	case "before", "after":
		if len(rest) < 2 {
			return nil, nil, fmt.Errorf("%q needs a target block name", pos)
		}
		if lang, err = guidedBlockLang(lang, dests); err != nil {
			return nil, nil, err
		}
		posFlags = []string{"-" + pos + "block", rest[1]}
	case "at-end", "":
		if len(dests) != 1 {
			return nil, nil, fmt.Errorf("at-end (the default) supports one destination; name a before/after target for several")
		}
		n := 0
		if lines, e := readLines(dests[0]); e == nil {
			n = len(lines)
		} else {
			notes = append(notes, dests[0]+" does not exist yet, it will be created")
		}
		if pos == "" {
			notes = append(notes, "no position given, appending at the end of "+dests[0])
		}
		posFlags = []string{"-after", strconv.Itoa(n)}
	default:
		return nil, nil, fmt.Errorf("unknown position %q (use before, after or at-end)", pos)
	}
	argv := []string{"-op", verb, "-file", src}
	if lang != "" {
		argv = append(argv, "-lang", lang)
	}
	argv = append(argv, srcFlags...)
	argv = append(argv, "-dest", strings.Join(dests, ","))
	argv = append(argv, posFlags...)
	return append(argv, "-v"), notes, nil
}

func guidedReplace(args []string) ([]string, []string, error) {
	src, spec := args[0], args[1]
	dests := guidedDests(args[3])
	if len(dests) == 0 {
		return nil, nil, fmt.Errorf("no destination file given")
	}
	srcFlags, lang, err := guidedSource(src, spec)
	if err != nil {
		return nil, nil, err
	}
	target := spec
	if len(args) > 4 {
		target = args[4]
	} else if guidedLineRange.MatchString(spec) {
		return nil, nil, fmt.Errorf("a line range has no name, so replace needs an explicit target block")
	}
	if lang, err = guidedBlockLang(lang, dests); err != nil {
		return nil, nil, err
	}
	argv := []string{"-op", "replace", "-file", src, "-lang", lang}
	argv = append(argv, srcFlags...)
	argv = append(argv, "-dest", strings.Join(dests, ","), "-destblock", target, "-v")
	return argv, nil, nil
}

func guidedInsertFile(args []string) ([]string, []string, error) {
	src, spec, verb, dest := args[0], args[1], args[2], args[3]
	if len(args) < 5 {
		return nil, nil, fmt.Errorf("%s needs a target block name after the destination file", verb)
	}
	if guidedLineRange.MatchString(spec) {
		return nil, nil, fmt.Errorf("%s from another file needs a named source block, not a line range", verb)
	}
	dests := guidedDests(dest)
	if len(dests) != 1 {
		return nil, nil, fmt.Errorf("%s takes exactly one destination file", verb)
	}
	_, lang, err := guidedSource(src, spec)
	if err != nil {
		return nil, nil, err
	}
	if lang, err = guidedBlockLang(lang, dests); err != nil {
		return nil, nil, err
	}
	return []string{"-op", verb, "-file", dests[0], "-lang", lang, "-block", args[4],
		"-sourcefile", src, "-sourceblock", spec, "-v"}, nil, nil
}

func guidedInsertText(args []string) ([]string, []string, error) {
	text, verb, file := args[0], args[1], args[2]
	if len(args) < 4 {
		return nil, nil, fmt.Errorf("%s needs a target block name", verb)
	}
	lang, err := guidedBlockLang("", []string{file})
	if err != nil {
		return nil, nil, err
	}
	return []string{"-op", verb, "-file", file, "-lang", lang, "-block", args[3], "-text", text, "-v"}, nil, nil
}

// translateGuided takes the words after "guided" and returns the explicit argv.
func translateGuided(args []string) ([]string, []string, error) {
	silent := false
	if n := len(args); n > 0 && args[n-1] == "silent" {
		silent = true
		args = args[:n-1]
	}
	if len(args) < 4 {
		return nil, nil, fmt.Errorf("%s", guidedUsage)
	}
	var argv, notes []string
	var err error
	switch {
	case args[1] == "insertbefore" || args[1] == "insertafter":
		argv, notes, err = guidedInsertText(args)
	case args[2] == "copy" || args[2] == "move":
		argv, notes, err = guidedTransfer(args)
	case args[2] == "replace":
		argv, notes, err = guidedReplace(args)
	case args[2] == "insertbefore" || args[2] == "insertafter":
		argv, notes, err = guidedInsertFile(args)
	default:
		return nil, nil, fmt.Errorf("unknown verb %q\n%s", args[2], guidedUsage)
	}
	if err != nil {
		return nil, nil, err
	}
	if silent {
		for i, a := range argv {
			if a == "-v" {
				argv[i] = "-quiet"
			}
		}
		notes = nil
	}
	return argv, notes, nil
}

// runGuided rewrites os.Args so main() continues into the normal flag path.
func runGuided() error {
	argv, notes, err := translateGuided(os.Args[2:])
	if err != nil {
		return err
	}
	for _, n := range notes {
		fmt.Fprintln(os.Stderr, "guided note: "+n)
	}
	quiet := false
	parts := []string{"fedit"}
	for _, a := range argv {
		if a == "-quiet" {
			quiet = true
		}
		parts = append(parts, guidedQuote(a))
	}
	if !quiet {
		fmt.Fprintln(os.Stderr, "guided: "+strings.Join(parts, " "))
	}
	os.Args = append([]string{os.Args[0]}, argv...)
	return nil
}
