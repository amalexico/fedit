package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// guided: natural-order front end. Translates words into a real explicit -op
// argv, echoes it to stderr, then main() falls through into the normal flag path.

var guidedLineRange = regexp.MustCompile(`(?i)^line\s+(\d+)-(\d+)$`)

const guidedUsage = "usage: fedit guided SRCFILE SRCBLOCK VERB DESTFILE [before|after TARGET | at-end] [silent|report-only]\n" +
	"       fedit guided   (no arguments: asks the questions one at a time)\n" +
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

// guidedLookup lists the top-level blocks the engine would match for name
// (substring of the display name or the key, same rule as resolveBlock).
func guidedLookup(lines []string, lang, name string) ([]blockEntry, error) {
	entries, err := getTopLevelBlocks(lines, lang)
	if err != nil {
		return nil, err
	}
	var ms []blockEntry
	for _, e := range entries {
		if strings.Contains(e.name, name) || strings.Contains(e.key, name) {
			ms = append(ms, e)
		}
	}
	return ms, nil
}

// guidedAsk shows a numbered choice and returns the 0-based pick. It only asks
// when stdin is a terminal; tests replace it.
var guidedAsk = func(question string, options []string) (int, bool) {
	fi, err := os.Stdin.Stat()
	if err != nil || fi.Mode()&os.ModeCharDevice == 0 {
		return 0, false
	}
	fmt.Fprintln(os.Stderr, question)
	for i, o := range options {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, o)
	}
	fmt.Fprint(os.Stderr, "Choose a number: ")
	var n int
	if _, e := fmt.Fscanln(os.Stdin, &n); e != nil || n < 1 || n > len(options) {
		return 0, false
	}
	return n - 1, true
}

// guidedChoose settles an ambiguous name: a single exact key or name match wins
// with a note, otherwise the person picks, otherwise it is an error listing the
// candidates. The chosen block is then addressed by line numbers, because the
// engine can only match by substring.
func guidedChoose(file, name string, ms []blockEntry) (blockEntry, string, error) {
	var exact []blockEntry
	for _, m := range ms {
		if m.key == name || m.name == name {
			exact = append(exact, m)
		}
	}
	if len(exact) == 1 {
		e := exact[0]
		return e, fmt.Sprintf("%s: %q matched %d blocks, using the exact match %s (line %d)", file, name, len(ms), e.name, e.start), nil
	}
	opts := make([]string, len(ms))
	for i, m := range ms {
		opts[i] = fmt.Sprintf("%s (line %d)", m.name, m.start)
	}
	q := fmt.Sprintf("%s: %q matched %d blocks, which one?", file, name, len(ms))
	if i, ok := guidedAsk(q, opts); ok {
		return ms[i], "", nil
	}
	return blockEntry{}, "", fmt.Errorf("%s\n  %s\nrun it in a terminal to pick one, or use a more specific name", q, strings.Join(opts, "\n  "))
}

// guidedSource returns the source-side flags, the source language ("" when
// unknown) and any notes.
func guidedSource(file, spec string) ([]string, string, []string, error) {
	lang, _ := detectLang(file)
	if m := guidedLineRange.FindStringSubmatch(spec); m != nil {
		return []string{"-line", m[1], "-end", m[2]}, lang, nil, nil
	}
	if lang == "" {
		return nil, "", nil, fmt.Errorf("cannot detect the language of %s (needed for block %q)", file, spec)
	}
	if lines, e := readLines(file); e == nil {
		if ms, e2 := guidedLookup(lines, lang, spec); e2 == nil && len(ms) > 1 {
			m, note, err := guidedChoose(file, spec, ms)
			if err != nil {
				return nil, "", nil, err
			}
			var notes []string
			if note != "" {
				notes = append(notes, note)
			}
			return []string{"-line", strconv.Itoa(m.start), "-end", strconv.Itoa(m.end)}, lang, notes, nil
		}
	}
	return []string{"-block", spec}, lang, nil, nil
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

// guidedStep is one explicit engine command; create lists destination files to
// create (empty) before it runs.
type guidedStep struct {
	argv   []string
	create []string
}

// guidedTransfer plans copy and move. A missing destination file is created and
// a missing anchor falls back to appending at the end, per destination, so the
// plan may hold several commands. For move, the last command is the move and the
// earlier ones are copies, so the source is erased only by a write that landed.
func guidedTransfer(args []string) ([]guidedStep, []string, error) {
	src, spec, verb := args[0], args[1], args[2]
	dests := guidedDests(args[3])
	if len(dests) == 0 {
		return nil, nil, fmt.Errorf("no destination file given")
	}
	srcFlags, lang, srcNotes, err := guidedSource(src, spec)
	if err != nil {
		return nil, nil, err
	}
	rest := args[4:]
	pos := ""
	if len(rest) > 0 {
		pos = rest[0]
	}
	notes := srcNotes
	kind, target := "", ""
	switch pos {
	case "before", "after":
		if len(rest) < 2 {
			return nil, nil, fmt.Errorf("%q needs a target block name", pos)
		}
		if lang, err = guidedBlockLang(lang, dests); err != nil {
			return nil, nil, err
		}
		kind, target = pos, rest[1]
	case "at-end":
	case "":
		notes = append(notes, "no position given, appending at the end of each destination")
	default:
		return nil, nil, fmt.Errorf("unknown position %q (use before, after or at-end)", pos)
	}
	base := func(extra ...string) []string {
		argv := []string{"-op", "copy", "-file", src}
		if lang != "" {
			argv = append(argv, "-lang", lang)
		}
		argv = append(argv, srcFlags...)
		return append(argv, extra...)
	}
	var steps []guidedStep
	var okDests []string
	for _, d := range dests {
		lines, e := readLines(d)
		if e != nil {
			notes = append(notes, d+" does not exist yet, it will be created with the transferred content")
			steps = append(steps, guidedStep{argv: base("-dest", d, "-after", "0", "-v"), create: []string{d}})
			continue
		}
		if kind != "" {
			ms, e2 := guidedLookup(lines, lang, target)
			switch {
			case e2 == nil && len(ms) == 1:
				okDests = append(okDests, d)
				continue
			case e2 == nil && len(ms) > 1:
				m, note, e3 := guidedChoose(d, target, ms)
				if e3 != nil {
					return nil, nil, e3
				}
				if note != "" {
					notes = append(notes, note)
				}
				posFlags := []string{"-before", strconv.Itoa(m.start)}
				if kind == "after" {
					posFlags = []string{"-after", strconv.Itoa(m.end)}
				}
				steps = append(steps, guidedStep{argv: base(append([]string{"-dest", d}, append(posFlags, "-v")...)...)})
				continue
			}
			notes = append(notes, fmt.Sprintf("%s: block %q not found, appending at the end", d, target))
		}
		steps = append(steps, guidedStep{argv: base("-dest", d, "-after", strconv.Itoa(len(lines)), "-v")})
	}
	if len(okDests) > 0 {
		group := guidedStep{argv: base("-dest", strings.Join(okDests, ","), "-"+kind+"block", target, "-v")}
		steps = append([]guidedStep{group}, steps...)
	}
	if verb == "move" {
		steps[len(steps)-1].argv[1] = "move"
	}
	return steps, notes, nil
}

func guidedReplace(args []string) ([]string, []string, error) {
	src, spec := args[0], args[1]
	dests := guidedDests(args[3])
	if len(dests) == 0 {
		return nil, nil, fmt.Errorf("no destination file given")
	}
	srcFlags, lang, srcNotes, err := guidedSource(src, spec)
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
	return argv, srcNotes, nil
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
	_, lang, _, err := guidedSource(src, spec)
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

// planGuided takes the words after "guided" and returns the commands to run.
func planGuided(args []string) ([]guidedStep, []string, error) {
	silent, reportOnly := false, false
	if n := len(args); n > 0 {
		switch args[n-1] {
		case "silent":
			silent = true
			args = args[:n-1]
		case "report-only":
			reportOnly = true
			args = args[:n-1]
		}
	}
	if len(args) < 4 {
		return nil, nil, fmt.Errorf("%s", guidedUsage)
	}
	var steps []guidedStep
	var notes []string
	var err error
	one := func(argv []string, n []string, e error) {
		if e == nil {
			steps = []guidedStep{{argv: argv}}
		}
		notes, err = n, e
	}
	switch {
	case args[1] == "insertbefore" || args[1] == "insertafter":
		one(guidedInsertText(args))
	case args[2] == "copy" || args[2] == "move":
		steps, notes, err = guidedTransfer(args)
	case args[2] == "replace":
		one(guidedReplace(args))
	case args[2] == "insertbefore" || args[2] == "insertafter":
		one(guidedInsertFile(args))
	default:
		return nil, nil, fmt.Errorf("unknown verb %q\n%s", args[2], guidedUsage)
	}
	if err != nil {
		return nil, nil, err
	}
	if reportOnly {
		for i := range steps {
			var kept []string
			for _, a := range steps[i].argv {
				if a != "-v" {
					kept = append(kept, a)
				}
			}
			steps[i].argv = kept
		}
	}
	if silent {
		for _, st := range steps {
			for i, a := range st.argv {
				if a == "-v" {
					st.argv[i] = "-quiet"
				}
			}
		}
		notes = nil
	}
	return steps, notes, nil
}

// translateGuided is the single-command form of planGuided (used by tests).
func translateGuided(args []string) ([]string, []string, error) {
	steps, notes, err := planGuided(args)
	if err != nil {
		return nil, nil, err
	}
	if len(steps) != 1 {
		return nil, nil, fmt.Errorf("plan has %d commands", len(steps))
	}
	return steps[0].argv, notes, nil
}

func guidedEcho(argv []string) {
	parts := []string{"fedit"}
	for _, a := range argv {
		if a == "-quiet" {
			return
		}
		parts = append(parts, guidedQuote(a))
	}
	fmt.Fprintln(os.Stderr, "guided: "+strings.Join(parts, " "))
}

// runGuided runs the plan. One plain command rewrites os.Args and returns so
// main() continues into the normal flag path; several commands run in order as
// child processes (flag.Parse can only run once) and the process exits.
func runGuided() error {
	args := os.Args[2:]
	if len(args) == 0 {
		var ierr error
		args, ierr = guidedInteractive(os.Stdin, os.Stderr)
		if ierr != nil {
			return ierr
		}
	}
	quietEcho := len(args) > 0 && args[len(args)-1] == "report-only"
	steps, notes, err := planGuided(args)
	if err != nil {
		return err
	}
	for _, n := range notes {
		fmt.Fprintln(os.Stderr, "guided note: "+n)
	}
	if len(steps) == 1 && len(steps[0].create) == 0 {
		if !quietEcho {
			guidedEcho(steps[0].argv)
		}
		os.Args = append([]string{os.Args[0]}, steps[0].argv...)
		return nil
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	code := 0
	for _, st := range steps {
		failed := false
		for _, f := range st.create {
			if e := os.WriteFile(f, nil, 0o644); e != nil {
				fmt.Fprintf(os.Stderr, "guided: cannot create %s: %v\n", f, e)
				failed = true
			}
		}
		if failed {
			code = 1
			continue
		}
		if !quietEcho {
			guidedEcho(st.argv)
		}
		cmd := exec.Command(exe, st.argv...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e := cmd.Run(); e != nil {
			code = 1
		}
	}
	os.Exit(code)
	return nil
}
