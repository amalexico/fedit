package main

import "time"

// The logging session ties the settings, the sink, the spool and the merge
// together for one run of fedit: Begin before the operation, End when the
// process exits. It is not wired into the operations yet.

// The operations fedit knows, as listed by -op. Reads never change a file.
var (
	logReadOps  = map[string]bool{"show": true, "map": true, "find": true, "fields": true}
	logWriteOps = map[string]bool{
		"insert": true, "delete": true, "replace": true, "replaceall": true,
		"write": true, "writeraw": true, "writelines": true,
		"insertafter": true, "insertbefore": true, "move": true, "copy": true,
	}
)

// logOpKnown reports whether op is an operation fedit performs. An unknown or
// empty -op is a usage error, not an operation, and is never logged.
func logOpKnown(op string) bool { return logReadOps[op] || logWriteOps[op] }

// logOpIsWrite reports whether op can change a file.
func logOpIsWrite(op string) bool { return logWriteOps[op] }

// MarkBegin reads the clock once and remembers it as the start of the
// operation without writing a record, so the end record can still report ms.
func (s *logSink) MarkBegin() time.Time {
	t := s.Now()
	s.begun, s.hasBegin = t, true
	return t
}

// logLevelPasses reports whether an end record with this exit code is
// written under the threshold. The levels follow logSink.End: exit 0 is
// INFO, exit 1 WARNING, anything else ERROR.
func logLevelPasses(th logThreshold, code int) bool {
	switch th {
	case logThresholdWarning:
		return code != 0
	case logThresholdError:
		return code != 0 && code != 1
	}
	return true
}

type logSession struct {
	settings logSettings
	op       string
	fields   []logKV
	primary  logIO
	sink     *logSink
	active   bool
	filtered bool // the begin record is held back and the end record carries its fields
}

// newLogSession prepares the logging for one run. fileIO is the real (or
// fake) log file access; notice is called once when permissive mode had to
// spool a record. The session is active when the mode is on, op is a known
// operation, and the scope covers it. In strict mode the level is always all.
func newLogSession(s logSettings, op string, now func() time.Time, fileIO logIO, notice func(), fields []logKV) *logSession {
	ls := &logSession{settings: s, op: op, fields: fields, primary: fileIO}
	ls.active = s.Mode != logOff && logOpKnown(op) && (s.Scope != logScopeWrites || logOpIsWrite(op))
	ls.sink = newLogSink(s.Mode, s.File, s.MaxMB, now, fileIO)
	if s.Mode == logPermissive && s.SpoolDir != "" {
		ls.sink.IO = newSpoolIO(fileIO, s.SpoolDir, ls.sink.ID, notice)
	}
	ls.filtered = s.Mode != logStrict && s.Level != logThresholdAll && s.Level != ""
	return ls
}

// Active reports whether this run is logged at all.
func (ls *logSession) Active() bool { return ls.active }

// ID is the id shared by the begin and end records of this run.
func (ls *logSession) ID() string { return ls.sink.ID }

// MustAbort reports whether a Begin or End error must stop the operation.
func (ls *logSession) MustAbort(err error) bool { return ls.sink.MustAbort(err) }

func (ls *logSession) beginFields() []logKV {
	return append([]logKV{{"op", ls.op}}, ls.fields...)
}

// Begin first tries to merge records spooled by earlier runs into the main
// log (a failed merge never blocks the run), then writes the begin record.
// When the level filters severities the begin record is not written.
func (ls *logSession) Begin() error {
	if !ls.active {
		return nil
	}
	if ls.settings.SpoolDir != "" {
		_, _ = mergeSpool(ls.settings.SpoolDir, ls.settings.File, ls.primary)
	}
	if ls.filtered {
		ls.sink.MarkBegin()
		return nil
	}
	return ls.sink.Begin(ls.beginFields())
}

// End writes the end record, unless the level filters it out. When the begin
// record was held back the end record carries the begin fields as well.
func (ls *logSession) End(code int, kvs []logKV) error {
	if !ls.active {
		return nil
	}
	if ls.filtered {
		if !logLevelPasses(ls.settings.Level, code) {
			return nil
		}
		kvs = append(ls.beginFields(), kvs...)
	}
	return ls.sink.End(code, kvs)
}
