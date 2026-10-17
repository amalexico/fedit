package main

import (
	"strings"
	"unicode/utf8"
)

// strMode is the multi-line string construct the block scanners are inside.
type strMode int

const (
	smNone         strMode = iota
	smBacktick             // Go raw string, JS template literal
	smVerbatim             // C# @"..." (a doubled quote is an escaped quote)
	smQuotes               // C# raw string and Java text block (three or more quotes)
	smRustRaw              // Rust r"..." and r#"..."#
	smHeredoc              // HCL heredoc (<<EOT, <<-EOT)
	smNixIndent            // Nix ''...''
	smBlockComment         // /* ... */ (nested in Rust)
)

// strState carries multi-line string state from one line to the next. The lang
// given to strip picks the constructs: "hcl" heredocs, "nix" indented strings,
// "clike" the C# and Java forms and Rust raw strings, "rust" Rust raw strings plus
// char literals and lifetimes. Every mode except "" also skips comments. (it also covers JS, PHP and the other
// brace-tracked languages), anything else backtick strings only.
// The "java" lang is "clike" plus backslash escapes inside text blocks, and "php"
// is "clike" plus heredocs, nowdocs and # comments.
type strState struct {
	mode    strMode
	n       int    // quote count (smQuotes) or hash count (smRustRaw)
	tag     string // heredoc terminator
	pending bool   // heredoc opened on this line, its body starts on the next
	nest    bool   // block comments nest (Rust)
	esc     bool   // Java text block: a backslash escapes the next character
}

func isIdentByte(c byte) bool {
	return c == '_' || (c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// skipQuoted returns the index just past the single-line string that opens at i.
func skipQuoted(line string, i int) int {
	quote := line[i]
	for j := i + 1; j < len(line); j++ {
		if line[j] == '\\' {
			j++
			continue
		}
		if line[j] == quote {
			return j + 1
		}
	}
	return len(line)
}

// lineComment reports whether rest starts a comment that runs to the end of the
// line. Nix has no // comment (it is the update operator), and # is only a
// comment in HCL and Nix (it is an attribute, a directive or a field elsewhere).
func lineComment(rest, lang string) bool {
	switch lang {
	case "clike", "rust", "go":
		return strings.HasPrefix(rest, "//")
	case "php":
		return strings.HasPrefix(rest, "//") || (strings.HasPrefix(rest, "#") && !strings.HasPrefix(rest, "#["))
	case "hcl":
		return strings.HasPrefix(rest, "//") || strings.HasPrefix(rest, "#")
	case "nix":
		return strings.HasPrefix(rest, "#")
	}
	return false
}

// rustQuote handles a quote at line[i] in Rust: a char literal is skipped as a
// string, while a lifetime or loop label (no closing quote) is plain code.
func rustQuote(line string, i int) int {
	if i+1 >= len(line) {
		return i + 1
	}
	if line[i+1] == '\\' {
		for j := i + 3; j < len(line); j++ {
			if line[j] == '\'' {
				return j + 1
			}
		}
		return len(line)
	}
	_, size := utf8.DecodeRuneInString(line[i+1:])
	if i+1+size < len(line) && line[i+1+size] == '\'' {
		return i + 2 + size
	}
	return i + 1
}

// strip returns line with every string literal removed, so braces inside
// strings are never counted, and updates the multi-line state.
func (s *strState) strip(line, lang string) string {
	java := lang == "java"
	if java {
		lang = "clike"
	}
	if s.mode == smHeredoc {
		if lang != "php" {
			if strings.TrimSpace(line) == s.tag {
				s.mode = smNone
			}
			return ""
		}
		rest, closed := phpHeredocEnd(line, s.tag)
		if !closed {
			return ""
		}
		s.mode = smNone
		line = rest
	}
	var b strings.Builder
	i := 0
	for i < len(line) {
		if s.mode != smNone {
			n, closed := s.consume(line[i:])
			i += n
			if !closed {
				break
			}
			s.mode = smNone
			continue
		}
		if lineComment(line[i:], lang) {
			break
		}
		if strings.HasPrefix(line[i:], "/*") && lang != "" {
			s.mode, s.n, s.nest = smBlockComment, 1, lang == "rust"
			i += 2
			continue
		}
		if n := s.open(line, i, lang); n > 0 {
			s.esc = java && s.mode == smQuotes
			i += n
			continue
		}
		c := line[i]
		if c == '\'' && lang == "rust" {
			i = rustQuote(line, i)
			continue
		}
		if c == '"' || c == '\'' {
			i = skipQuoted(line, i)
			continue
		}
		b.WriteByte(c)
		i++
	}
	if s.pending {
		s.pending = false
		s.mode = smHeredoc
	}
	return b.String()
}

// open recognises the start of a multi-line string at line[i], sets the mode and
// returns how many bytes the opener takes (0 when there is none).
func (s *strState) open(line string, i int, lang string) int {
	rest := line[i:]
	switch rest[0] {
	case '`':
		s.mode = smBacktick
		return 1
	case '@':
		if lang == "clike" {
			j := 1
			if j < len(rest) && rest[j] == '$' {
				j++
			}
			if j < len(rest) && rest[j] == '"' {
				s.mode = smVerbatim
				return j + 1
			}
		}
	case '$':
		if lang == "clike" && strings.HasPrefix(rest, `$@"`) {
			s.mode = smVerbatim
			return 3
		}
	case '"':
		if lang == "clike" {
			q := 0
			for q < len(rest) && rest[q] == '"' {
				q++
			}
			if q >= 3 {
				s.mode, s.n = smQuotes, q
				return q
			}
		}
	case 'r':
		if (lang == "clike" || lang == "rust") && (i == 0 || !isIdentByte(line[i-1]) || (line[i-1] == 'b' && (i < 2 || !isIdentByte(line[i-2])))) {
			j, h := 1, 0
			for j < len(rest) && rest[j] == '#' {
				h++
				j++
			}
			if j < len(rest) && rest[j] == '"' {
				s.mode, s.n = smRustRaw, h
				return j + 1
			}
		}
	case '<':
		if lang == "php" && strings.HasPrefix(rest, "<<<") {
			if tag, n := phpHeredocOpen(rest); n > 0 {
				s.tag = tag
				s.pending = true
				return n
			}
		}
		if lang == "hcl" && strings.HasPrefix(rest, "<<") {
			j := 2
			if j < len(rest) && rest[j] == '-' {
				j++
			}
			k := j
			for k < len(rest) && isIdentByte(rest[k]) {
				k++
			}
			if k > j {
				s.tag = rest[j:k]
				s.pending = true
				return k
			}
		}
	case '\'':
		if lang == "nix" && strings.HasPrefix(rest, "''") {
			s.mode = smNixIndent
			return 2
		}
	}
	return 0
}

// consume reads the rest of a multi-line string. It returns the bytes used and
// whether the string ended on this line.
func (s *strState) consume(rest string) (int, bool) {
	switch s.mode {
	case smBacktick:
		if k := strings.IndexByte(rest, '`'); k >= 0 {
			return k + 1, true
		}
	case smVerbatim:
		for j := 0; j < len(rest); j++ {
			if rest[j] == '"' {
				if j+1 < len(rest) && rest[j+1] == '"' {
					j++
					continue
				}
				return j + 1, true
			}
		}
	case smQuotes:
		pat := strings.Repeat(`"`, s.n)
		if s.esc {
			for j := 0; j < len(rest); j++ {
				if rest[j] == '\\' {
					j++
					continue
				}
				if strings.HasPrefix(rest[j:], pat) {
					return j + len(pat), true
				}
			}
			return len(rest), false
		}
		if k := strings.Index(rest, pat); k >= 0 {
			return k + len(pat), true
		}
	case smRustRaw:
		pat := `"` + strings.Repeat("#", s.n)
		if k := strings.Index(rest, pat); k >= 0 {
			return k + len(pat), true
		}
	case smBlockComment:
		for j := 0; j < len(rest); j++ {
			if s.nest && strings.HasPrefix(rest[j:], "/*") {
				s.n++
				j++
				continue
			}
			if strings.HasPrefix(rest[j:], "*/") {
				s.n--
				j++
				if s.n == 0 {
					return j + 1, true
				}
			}
		}
	case smNixIndent:
		for j := 0; j+1 < len(rest); j++ {
			if rest[j] == '\'' && rest[j+1] == '\'' {
				if j+2 < len(rest) && (rest[j+2] == '\'' || rest[j+2] == '$' || rest[j+2] == '\\') {
					j += 2
					continue
				}
				return j + 2, true
			}
		}
	}
	return len(rest), false
}

// phpHeredocOpen parses a PHP <<<TAG, <<<"TAG" or <<<'TAG' opener at the start of
// rest. It returns the tag and the bytes the opener takes (0 when it is not one).
func phpHeredocOpen(rest string) (string, int) {
	j := 3
	for j < len(rest) && (rest[j] == ' ' || rest[j] == '\t') {
		j++
	}
	var q byte
	if j < len(rest) && (rest[j] == '\'' || rest[j] == '"') {
		q = rest[j]
		j++
	}
	k := j
	for k < len(rest) && isIdentByte(rest[k]) {
		k++
	}
	if k == j || (rest[j] >= '0' && rest[j] <= '9') {
		return "", 0
	}
	tag := rest[j:k]
	if q != 0 {
		if k >= len(rest) || rest[k] != q {
			return "", 0
		}
		k++
	}
	return tag, k
}

// phpHeredocEnd reports whether line closes the PHP heredoc or nowdoc named tag.
// The closing marker may be indented and must not be followed by an identifier
// character. The returned rest is the text after the marker, which is ordinary code.
func phpHeredocEnd(line, tag string) (string, bool) {
	t := strings.TrimLeft(line, " \t")
	if !strings.HasPrefix(t, tag) {
		return "", false
	}
	if len(t) > len(tag) && isIdentByte(t[len(tag)]) {
		return "", false
	}
	return t[len(tag):], true
}
