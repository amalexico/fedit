package main

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	cssReSelector  = regexp.MustCompile(`^([a-zA-Z.#@\[\*:][^{]*)\{`)
	cssReMedia     = regexp.MustCompile(`^@media\s+`)
	cssReKeyframes = regexp.MustCompile(`^@keyframes\s+(\S+)`)
	cssReContainer = regexp.MustCompile(`^@(supports|layer|container|-webkit-keyframes)\b`)
)

// cssBraces returns the { and } characters of one CSS line in order, skipping
// quoted strings and /* */ comments. inComment carries a multi-line comment
// from one line to the next.
func cssBraces(line string, inComment bool) ([]byte, bool) {
	var ev []byte
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case inComment:
			if c == '*' && i+1 < len(line) && line[i+1] == '/' {
				inComment = false
				i++
			}
		case quote != 0:
			if c == '\\' {
				i++
			} else if c == quote {
				quote = 0
			}
		case c == '/' && i+1 < len(line) && line[i+1] == '*':
			inComment = true
			i++
		case c == '"' || c == '\'':
			quote = c
		case c == '{' || c == '}':
			ev = append(ev, c)
		}
	}
	return ev, inComment
}

type cssDup struct {
	sel, ctx string
	n        int
}

// doMapCSSSelectors prints the selectors and at-rules of a stylesheet and
// reports duplicates. A selector is only a duplicate of another one in the same
// container block: the same selector in two different @media or @keyframes
// blocks (or once at top level and once inside @media) is normal CSS.
func doMapCSSSelectors(lines []string) {
	scopes := []int{0}
	labels := map[int]string{0: ""}
	next := 1
	counts := map[string]*cssDup{}
	var order []string
	add := func(sel string) {
		scope := scopes[len(scopes)-1]
		k := fmt.Sprintf("%d|%s", scope, sel)
		d, ok := counts[k]
		if !ok {
			d = &cssDup{sel: sel, ctx: labels[scope]}
			counts[k] = d
			order = append(order, k)
		}
		d.n++
	}
	inComment := false
	pending := false
	pendingLabel := ""
	for i, line := range lines {
		t := strings.TrimSpace(line)
		startInComment := inComment
		events, now := cssBraces(t, inComment)
		inComment = now
		if !startInComment {
			if m := cssReKeyframes.FindStringSubmatch(t); m != nil {
				fmt.Printf("%4d │ @keyframes %s\n", i+1, m[1])
				add("@keyframes " + m[1])
				pending, pendingLabel = true, "@keyframes "+m[1]
			} else if cssReMedia.MatchString(t) {
				fmt.Printf("%4d │ %s\n", i+1, t)
				label := t
				if idx := strings.Index(t, "{"); idx >= 0 {
					label = strings.TrimSpace(t[:idx])
				}
				pending, pendingLabel = true, label
			} else if m := cssReSelector.FindStringSubmatch(t); m != nil {
				sel := strings.TrimSpace(m[1])
				fmt.Printf("%4d │ %s\n", i+1, sel)
				add(sel)
				if cssReContainer.MatchString(sel) {
					pending, pendingLabel = true, sel
				}
			}
		}
		for _, ev := range events {
			if ev == '{' {
				if pending {
					scopes = append(scopes, next)
					labels[next] = pendingLabel
					next++
					pending = false
				} else {
					scopes = append(scopes, scopes[len(scopes)-1])
				}
			} else if len(scopes) > 1 {
				scopes = scopes[:len(scopes)-1]
			}
		}
	}

	fmt.Println()
	dups := false
	for _, k := range order {
		d := counts[k]
		if d.n < 2 {
			continue
		}
		if !dups {
			fmt.Println("⚠ Duplicates:")
			dups = true
		}
		if d.ctx == "" {
			fmt.Printf("  %s (%d times)\n", d.sel, d.n)
		} else {
			fmt.Printf("  %s (%d times, inside %s)\n", d.sel, d.n, d.ctx)
		}
	}
	if !dups {
		fmt.Println("✓ No duplicate selectors")
	}
}
