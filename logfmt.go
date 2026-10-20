package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Pure formatting helpers for the audit log. One record is one line:
//
//	2026-10-18T08:17:15.040Z [INFO] id=a1b2c3d4e5f60718 op=replaceall file=a.txt
//
// Nothing here touches the file system or the operation code.

type logLevel string

const (
	logInfo    logLevel = "INFO"
	logWarning logLevel = "WARNING"
	logError   logLevel = "ERROR"
)

// logKV is one key=value field. Fields keep the order they are given in.
type logKV struct {
	Key   string
	Value string
}

// logTimestamp formats t in UTC as ISO 8601 with a dot before the
// milliseconds (truncated, never rounded).
func logTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05.000Z")
}

// logKey keeps [A-Za-z0-9_.-] and turns every other character into an
// underscore, so a key can never break the line.
func logKey(k string) string {
	if k == "" {
		return "_"
	}
	var b strings.Builder
	for _, r := range k {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '.', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}

func logNeedsQuote(v string) bool {
	if v == "" {
		return true
	}
	for _, r := range v {
		if r == '"' || r == '=' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// logQuote returns v as it appears after the equals sign: bare when it is
// safe, otherwise in double quotes with backslash escapes. A bare value is
// verbatim up to the next space, so Windows paths without spaces stay
// readable. Invalid UTF-8 is replaced by U+FFFD.
func logQuote(v string) string {
	v = strings.ToValidUTF8(v, "\uFFFD")
	if !logNeedsQuote(v) {
		return v
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range v {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if unicode.IsControl(r) {
				fmt.Fprintf(&b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// formatLogLine builds one log record: timestamp, [LEVEL], then the fields.
// The result never contains a line break.
func formatLogLine(t time.Time, level logLevel, kvs []logKV) string {
	var b strings.Builder
	b.WriteString(logTimestamp(t))
	b.WriteString(" [")
	b.WriteString(string(level))
	b.WriteString("]")
	for _, kv := range kvs {
		b.WriteByte(' ')
		b.WriteString(logKey(kv.Key))
		b.WriteByte('=')
		b.WriteString(logQuote(kv.Value))
	}
	return b.String()
}

// logNewID returns 16 lowercase hex characters, shared by the begin and end
// records of one operation. If the system random source fails it falls back
// to the clock, so logging never blocks an edit.
func logNewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%016x", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

// logTextFields stands in for a text value that must not be logged verbatim:
// its length in bytes and its SHA-256.
func logTextFields(name string, b []byte) []logKV {
	sum := sha256.Sum256(b)
	return []logKV{
		{name + "_len", strconv.Itoa(len(b))},
		{name + "_sha256", hex.EncodeToString(sum[:])},
	}
}
