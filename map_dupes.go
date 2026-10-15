package main

import (
	"fmt"
	"regexp"
	"strings"
)

type mkdHeading struct {
	line  int
	level int
	text  string // raw text after the hashes, as printed by the heading list
}

var mkdReHeading = regexp.MustCompile(`^(#{1,6})\s+(.+)`)

// mkdFence returns the fence marker when the line starts a fenced code block
// line (up to three spaces of indent, then ``` or ~~~).
func mkdFence(line string) string {
	t := strings.TrimLeft(line, " ")
	if len(line)-len(t) > 3 {
		return ""
	}
	if strings.HasPrefix(t, "```") {
		return "```"
	}
	if strings.HasPrefix(t, "~~~") {
		return "~~~"
	}
	return ""
}

// mkdHeadings returns the ATX headings of a markdown file, skipping fenced
// code blocks so that shell comments inside them are not treated as headings.
func mkdHeadings(lines []string) []mkdHeading {
	var hs []mkdHeading
	fence := ""
	for i, line := range lines {
		if f := mkdFence(line); f != "" {
			if fence == "" {
				fence = f
			} else if f == fence {
				rest := strings.TrimSpace(strings.TrimLeft(strings.TrimLeft(line, " "), string(f[0])))
				if rest == "" {
					fence = ""
				}
			}
			continue
		}
		if fence != "" {
			continue
		}
		if m := mkdReHeading.FindStringSubmatch(line); m != nil {
			hs = append(hs, mkdHeading{line: i + 1, level: len(m[1]), text: m[2]})
		}
	}
	return hs
}

// mkdNormText drops an ATX closing sequence ("## Title ##") but keeps a
// trailing # that belongs to the text ("## Learning C#").
func mkdNormText(text string) string {
	t := strings.TrimSpace(text)
	i := len(t)
	for i > 0 && t[i-1] == '#' {
		i--
	}
	if i < len(t) && (i == 0 || t[i-1] == ' ' || t[i-1] == '\t') {
		t = strings.TrimSpace(t[:i])
	}
	return t
}

func dupJoinInts(ns []int) string {
	s := make([]string, len(ns))
	for i, n := range ns {
		s[i] = fmt.Sprint(n)
	}
	return strings.Join(s, ", ")
}

// mkdDuplicateReport lists headings that repeat with the same text, at the
// same level, under the same parent heading instance. The same heading under
// different parents is normal and is not reported.
func mkdDuplicateReport(hs []mkdHeading) []string {
	type group struct {
		level      int
		text       string
		parentText string
		lines      []int
	}
	groups := map[string]*group{}
	var order []string
	var stack []int
	for idx, h := range hs {
		for len(stack) > 0 && hs[stack[len(stack)-1]].level >= h.level {
			stack = stack[:len(stack)-1]
		}
		parent := 0
		parentText := ""
		if len(stack) > 0 {
			p := hs[stack[len(stack)-1]]
			parent = p.line
			parentText = strings.Repeat("#", p.level) + " " + mkdNormText(p.text)
		}
		stack = append(stack, idx)
		key := fmt.Sprintf("%d|%d|%s", parent, h.level, strings.ToLower(mkdNormText(h.text)))
		g := groups[key]
		if g == nil {
			g = &group{level: h.level, text: mkdNormText(h.text), parentText: parentText}
			groups[key] = g
			order = append(order, key)
		}
		g.lines = append(g.lines, h.line)
	}
	var out []string
	for _, k := range order {
		g := groups[k]
		if len(g.lines) < 2 {
			continue
		}
		s := fmt.Sprintf("  %s %s (%d times, lines %s", strings.Repeat("#", g.level), g.text, len(g.lines), dupJoinInts(g.lines))
		if g.parentText != "" {
			s += ", under " + g.parentText
		}
		out = append(out, s+")")
	}
	return out
}

func mkdPrintHeadings(hs []mkdHeading) {
	for _, h := range hs {
		fmt.Printf("%4d │ %s%s %s\n", h.line, strings.Repeat("  ", h.level-1), strings.Repeat("#", h.level), h.text)
	}
}

func mkdPrintDuplicates(hs []mkdHeading) {
	fmt.Println()
	d := mkdDuplicateReport(hs)
	if len(d) == 0 {
		fmt.Println("✓ No duplicate headings")
		return
	}
	fmt.Println("⚠ Duplicates:")
	for _, s := range d {
		fmt.Println(s)
	}
}

var (
	dkrReFrom = regexp.MustCompile(`(?i)^\s*FROM\s+`)
	dkrReAs   = regexp.MustCompile(`(?i)\s+AS\s+(\S+)`)
	dkrReOnce = regexp.MustCompile(`(?i)^\s*(CMD|ENTRYPOINT|HEALTHCHECK)\b`)
)

// dkrDuplicateReport finds duplicate stage names (Docker treats them
// case-insensitively and rejects them) and CMD, ENTRYPOINT or HEALTHCHECK
// repeated inside one stage, where only the last one takes effect. Comments and
// backslash continuation lines are skipped.
func dkrDuplicateReport(lines []string) []string {
	type stage struct {
		n    int
		name string
		once map[string][]int
	}
	var stages []*stage
	names := map[string][]int{}
	orig := map[string]string{}
	var nameOrder []string
	var cur *stage
	cont := false
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if cont {
			cont = strings.HasSuffix(t, "\\")
			continue
		}
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		cont = strings.HasSuffix(t, "\\")
		if dkrReFrom.MatchString(t) {
			s := &stage{n: len(stages) + 1, once: map[string][]int{}}
			if a := dkrReAs.FindStringSubmatch(t); a != nil {
				s.name = a[1]
				k := strings.ToLower(a[1])
				if _, ok := names[k]; !ok {
					nameOrder = append(nameOrder, k)
					orig[k] = a[1]
				}
				names[k] = append(names[k], i+1)
			}
			stages = append(stages, s)
			cur = s
			continue
		}
		if cur != nil {
			if m := dkrReOnce.FindStringSubmatch(t); m != nil {
				k := strings.ToUpper(m[1])
				cur.once[k] = append(cur.once[k], i+1)
			}
		}
	}
	var out []string
	for _, k := range nameOrder {
		if ls := names[k]; len(ls) > 1 {
			out = append(out, fmt.Sprintf("  stage %s (%d times, lines %s)", orig[k], len(ls), dupJoinInts(ls)))
		}
	}
	for _, s := range stages {
		for _, ins := range []string{"CMD", "ENTRYPOINT", "HEALTHCHECK"} {
			ls := s.once[ins]
			if len(ls) < 2 {
				continue
			}
			label := fmt.Sprintf("stage %d", s.n)
			if s.name != "" {
				label += " (" + s.name + ")"
			}
			out = append(out, fmt.Sprintf("  %s in %s (%d times, lines %s; only the last takes effect)", ins, label, len(ls), dupJoinInts(ls)))
		}
	}
	return out
}

func dkrPrintDuplicates(lines []string) {
	fmt.Println()
	d := dkrDuplicateReport(lines)
	if len(d) == 0 {
		fmt.Println("✓ No duplicate stages or repeated CMD/ENTRYPOINT/HEALTHCHECK")
		return
	}
	fmt.Println("⚠ Duplicates:")
	for _, s := range d {
		fmt.Println(s)
	}
}
