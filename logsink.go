package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The audit log sink writes the begin and end records of one operation. It
// knows nothing about files: the clock and the three file operations are
// injected, so tests run without a real log. The permissive spool, the
// notice and the wiring into the operations live elsewhere.

type logMode string

const (
	logOff        logMode = "off"
	logPermissive logMode = "permissive"
	logStrict     logMode = "strict"
)

// parseLogMode reads a log_mode value. An empty value means off.
func parseLogMode(s string) (logMode, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "off":
		return logOff, nil
	case "permissive":
		return logPermissive, nil
	case "strict":
		return logStrict, nil
	}
	return logOff, fmt.Errorf("invalid log_mode %q: use off, permissive or strict", s)
}

// logIO is the set of file operations the sink needs. Append adds one line
// (without its newline) to the file, Size reports the current size, and
// Rotate moves the full file out of the way.
type logIO struct {
	Append func(path, line string) error
	Size   func(path string) (int64, error)
	Rotate func(path string) error
}

type logSink struct {
	Mode  logMode
	Path  string
	MaxMB int // rotate when the file has reached this size; 0 = never
	Now   func() time.Time
	IO    logIO
	ID    string // shared by the begin and end records

	begun    time.Time
	hasBegin bool
}

func newLogSink(mode logMode, path string, maxMB int, now func() time.Time, fileIO logIO) *logSink {
	if now == nil {
		now = time.Now
	}
	return &logSink{Mode: mode, Path: path, MaxMB: maxMB, Now: now, IO: fileIO, ID: logNewID()}
}

// write rotates the file first when it has reached MaxMB, then appends the
// record. A failed rotation is not an error: the record is still appended.
func (s *logSink) write(level logLevel, t time.Time, kvs []logKV) error {
	if s.IO.Append == nil {
		return fmt.Errorf("no log writer configured")
	}
	if s.MaxMB > 0 && s.IO.Size != nil && s.IO.Rotate != nil {
		if size, err := s.IO.Size(s.Path); err == nil && size >= int64(s.MaxMB)*1024*1024 {
			_ = s.IO.Rotate(s.Path)
		}
	}
	return s.IO.Append(s.Path, formatLogLine(t, level, kvs))
}

// Begin writes the begin record. It reads the clock once, even when the
// write fails. In mode off it does nothing.
func (s *logSink) Begin(kvs []logKV) error {
	if s.Mode == logOff {
		return nil
	}
	t := s.Now()
	all := append([]logKV{{"id", s.ID}, {"phase", "begin"}}, kvs...)
	err := s.write(logInfo, t, all)
	if err == nil {
		s.begun, s.hasBegin = t, true
	}
	return err
}

// End writes the end record: level INFO for exit 0, WARNING for exit 1 (the
// edit did not happen) and ERROR for any other code. ms is the time since the
// begin record and is left out when no begin record was written.
func (s *logSink) End(code int, kvs []logKV) error {
	if s.Mode == logOff {
		return nil
	}
	t := s.Now()
	level := logError
	switch code {
	case 0:
		level = logInfo
	case 1:
		level = logWarning
	}
	head := []logKV{{"id", s.ID}, {"phase", "end"}, {"exit", strconv.Itoa(code)}}
	if s.hasBegin {
		head = append(head, logKV{"ms", strconv.FormatInt(t.Sub(s.begun).Milliseconds(), 10)})
	}
	return s.write(level, t, append(head, kvs...))
}

// MustAbort reports whether a write error must stop the operation: only in
// strict mode.
func (s *logSink) MustAbort(err error) bool {
	return err != nil && s.Mode == logStrict
}
