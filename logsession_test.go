package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The logging session ties the settings, the sink, the spool and the merge
// together for one run of fedit. These tests use fake file operations so they
// need no real log, except the merge tests, which use real temp files. They
// reuse the helpers of spool_test.go and logcfg_test.go.

type lgxRec struct {
	lines    []string
	attempts int
	failFrom int // 1-based attempt number from which Append fails; 0 = never
}

func (r *lgxRec) io() logIO {
	return logIO{
		Append: func(path, line string) error {
			r.attempts++
			if r.failFrom > 0 && r.attempts >= r.failFrom {
				return errors.New("locked")
			}
			r.lines = append(r.lines, line)
			return nil
		},
		Size:   func(path string) (int64, error) { return 0, nil },
		Rotate: func(path string) error { return nil },
	}
}

// lgxClock returns 2026-10-21T08:00:00Z on the first call and 250 ms more on
// each following call.
func lgxClock() func() time.Time {
	start := time.Date(2026, 10, 21, 8, 0, 0, 0, time.UTC)
	calls := 0
	return func() time.Time {
		t := start.Add(time.Duration(calls) * 250 * time.Millisecond)
		calls++
		return t
	}
}

func lgxSettings(t *testing.T, mode logMode, level logThreshold, scope logScope) logSettings {
	t.Helper()
	dir := t.TempDir()
	return logSettings{
		Mode:     mode,
		File:     filepath.Join(dir, "fedit.log"),
		Level:    level,
		Scope:    scope,
		SpoolDir: filepath.Join(dir, "spool"),
	}
}

var lgxFields = []logKV{{"source", "cli"}, {"file", "a.txt"}}

func lgxNew(s logSettings, op string, rec *lgxRec, notice func()) *logSession {
	return newLogSession(s, op, lgxClock(), rec.io(), notice, lgxFields)
}

func TestLG_OpTable(t *testing.T) {
	for _, op := range []string{"show", "map", "find", "fields"} {
		if !logOpKnown(op) || logOpIsWrite(op) {
			t.Errorf("%s: known = %v, write = %v; want a known read-only operation", op, logOpKnown(op), logOpIsWrite(op))
		}
	}
	for _, op := range []string{"insert", "delete", "replace", "replaceall", "write", "writeraw", "writelines", "insertafter", "insertbefore", "move", "copy"} {
		if !logOpKnown(op) || !logOpIsWrite(op) {
			t.Errorf("%s: known = %v, write = %v; want a known writing operation", op, logOpKnown(op), logOpIsWrite(op))
		}
	}
	for _, op := range []string{"", "frobnicate"} {
		if logOpKnown(op) || logOpIsWrite(op) {
			t.Errorf("%q must not be a known operation", op)
		}
	}
}

func TestLG_Inactive(t *testing.T) {
	cases := []struct {
		name string
		s    logSettings
		op   string
	}{
		{"mode off", lgxSettings(t, logOff, logThresholdAll, logScopeAll), "replaceall"},
		{"strict, scope writes, read-only op", lgxSettings(t, logStrict, logThresholdAll, logScopeWrites), "show"},
		{"strict, unknown op", lgxSettings(t, logStrict, logThresholdAll, logScopeAll), "frobnicate"},
		{"strict, no op", lgxSettings(t, logStrict, logThresholdAll, logScopeAll), ""},
	}
	for _, c := range cases {
		rec := &lgxRec{failFrom: 1}
		ls := lgxNew(c.s, c.op, rec, func() {})
		if ls.Active() {
			t.Errorf("%s: session must not be active", c.name)
		}
		if err := ls.Begin(); err != nil || ls.MustAbort(err) {
			t.Errorf("%s: Begin = %v; an inactive session never fails or aborts", c.name, err)
		}
		if err := ls.End(0, nil); err != nil || ls.MustAbort(err) {
			t.Errorf("%s: End = %v; an inactive session never fails or aborts", c.name, err)
		}
		if rec.attempts != 0 {
			t.Errorf("%s: the log must not be touched, got %d write attempts", c.name, rec.attempts)
		}
	}
}

func TestLG_ActiveCases(t *testing.T) {
	cases := []struct {
		scope logScope
		op    string
		want  bool
	}{
		{logScopeAll, "find", true},
		{logScopeAll, "replaceall", true},
		{logScopeWrites, "replaceall", true},
		{logScopeWrites, "find", false},
	}
	for _, c := range cases {
		ls := lgxNew(lgxSettings(t, logPermissive, logThresholdAll, c.scope), c.op, &lgxRec{}, func() {})
		if ls.Active() != c.want {
			t.Errorf("scope %s, op %s: Active = %v; want %v", c.scope, c.op, ls.Active(), c.want)
		}
	}
}

func TestLG_BeginEnd(t *testing.T) {
	rec := &lgxRec{}
	ls := lgxNew(lgxSettings(t, logPermissive, logThresholdAll, logScopeAll), "replaceall", rec, func() {})
	if err := ls.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := ls.End(0, []logKV{{"replacements", "2"}}); err != nil {
		t.Fatalf("End: %v", err)
	}
	if len(rec.lines) != 2 {
		t.Fatalf("want a begin and an end record, got %q", rec.lines)
	}
	id := ls.ID()
	if len(id) != 16 {
		t.Errorf("ID = %q; want 16 hex characters", id)
	}
	b, e := rec.lines[0], rec.lines[1]
	if !strings.HasPrefix(b, "2026-10-21T08:00:00.000Z [INFO] id="+id+" phase=begin op=replaceall source=cli file=a.txt") {
		t.Errorf("begin record = %q", b)
	}
	if !strings.HasPrefix(e, "2026-10-21T08:00:00.250Z [INFO] id="+id+" phase=end exit=0 ms=250") || !strings.HasSuffix(e, " replacements=2") {
		t.Errorf("end record = %q", e)
	}
}

func TestLG_EndLevels(t *testing.T) {
	for code, level := range map[int]string{1: "[WARNING]", 3: "[ERROR]"} {
		rec := &lgxRec{}
		ls := lgxNew(lgxSettings(t, logPermissive, logThresholdAll, logScopeAll), "replaceall", rec, func() {})
		_ = ls.Begin()
		_ = ls.End(code, nil)
		if len(rec.lines) != 2 || !strings.Contains(rec.lines[1], level+" id="+ls.ID()+" phase=end exit="+string(rune('0'+code))) {
			t.Errorf("exit %d: records = %q; want the end record at level %s", code, rec.lines, level)
		}
	}
}

func TestLG_LevelWarning(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdWarning, logScopeAll)

	rec := &lgxRec{}
	ls := lgxNew(s, "replaceall", rec, func() {})
	_ = ls.Begin()
	_ = ls.End(0, nil)
	if len(rec.lines) != 0 {
		t.Errorf("level warning must not write begin or INFO records, got %q", rec.lines)
	}

	rec = &lgxRec{}
	ls = lgxNew(s, "replaceall", rec, func() {})
	_ = ls.Begin()
	if len(rec.lines) != 0 {
		t.Errorf("the begin record is not written at level warning, got %q", rec.lines)
	}
	_ = ls.End(1, []logKV{{"replacements", "0"}})
	if len(rec.lines) != 1 {
		t.Fatalf("a warning must be written as one record, got %q", rec.lines)
	}
	want := "[WARNING] id=" + ls.ID() + " phase=end exit=1 ms=250 op=replaceall source=cli file=a.txt replacements=0"
	if !strings.Contains(rec.lines[0], want) {
		t.Errorf("record = %q; want it to contain %q (the end record carries the begin fields)", rec.lines[0], want)
	}
}

func TestLG_LevelError(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdError, logScopeAll)

	rec := &lgxRec{}
	ls := lgxNew(s, "replaceall", rec, func() {})
	_ = ls.Begin()
	_ = ls.End(1, nil)
	if len(rec.lines) != 0 {
		t.Errorf("level error must not write warnings, got %q", rec.lines)
	}

	rec = &lgxRec{}
	ls = lgxNew(s, "replaceall", rec, func() {})
	_ = ls.Begin()
	_ = ls.End(3, nil)
	if len(rec.lines) != 1 || !strings.Contains(rec.lines[0], "[ERROR] id="+ls.ID()+" phase=end exit=3") {
		t.Errorf("an error must be written as one record, got %q", rec.lines)
	}
}

func TestLG_StrictAlwaysWritesEverything(t *testing.T) {
	rec := &lgxRec{}
	ls := lgxNew(lgxSettings(t, logStrict, logThresholdError, logScopeAll), "replaceall", rec, func() {})
	if err := ls.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if len(rec.lines) != 1 || !strings.Contains(rec.lines[0], "phase=begin") {
		t.Errorf("strict must write the begin record whatever the level says, got %q", rec.lines)
	}
	_ = ls.End(0, nil)
	if len(rec.lines) != 2 {
		t.Errorf("strict must write the INFO end record too, got %q", rec.lines)
	}
}

func TestLG_PermissiveSpoolsOnFailure(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdAll, logScopeAll)
	rec := &lgxRec{failFrom: 1}
	notices := 0
	ls := lgxNew(s, "replaceall", rec, func() { notices++ })
	if err := ls.Begin(); err != nil || ls.MustAbort(err) {
		t.Fatalf("Begin = %v; permissive spools instead of failing", err)
	}
	if err := ls.End(0, nil); err != nil || ls.MustAbort(err) {
		t.Fatalf("End = %v; permissive spools instead of failing", err)
	}
	if notices != 1 {
		t.Errorf("the notice must fire exactly once for the run, fired %d times", notices)
	}
	if len(rec.lines) != 0 {
		t.Errorf("the main log accepted nothing, got %q", rec.lines)
	}
	got := strings.Split(strings.TrimRight(spRead(t, filepath.Join(s.SpoolDir, ls.ID()+".log")), "\n"), "\n")
	if len(got) != 2 || !strings.Contains(got[0], "phase=begin") || !strings.Contains(got[1], "phase=end") {
		t.Fatalf("spool = %q; want a begin and an end record", got)
	}
	for _, l := range got {
		if !strings.HasSuffix(l, " logged=spool") {
			t.Errorf("spooled record %q must end with logged=spool", l)
		}
	}
}

func TestLG_PermissiveNeverAborts(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	spWrite(t, blocker, "x")
	s := lgxSettings(t, logPermissive, logThresholdAll, logScopeAll)
	s.SpoolDir = filepath.Join(blocker, "spool")
	ls := lgxNew(s, "replaceall", &lgxRec{failFrom: 1}, func() {})
	err := ls.Begin()
	if err == nil {
		t.Errorf("when neither the log nor the spool can be written, Begin must report the error")
	}
	if ls.MustAbort(err) {
		t.Errorf("permissive mode never aborts the operation")
	}
}

func TestLG_StrictBeginFailureAborts(t *testing.T) {
	s := lgxSettings(t, logStrict, logThresholdAll, logScopeAll)
	ls := lgxNew(s, "replaceall", &lgxRec{failFrom: 1}, func() {})
	err := ls.Begin()
	if err == nil || !ls.MustAbort(err) {
		t.Errorf("strict: Begin = %v, MustAbort = %v; want an error that aborts", err, ls.MustAbort(err))
	}
	if m := spGlob(t, filepath.Join(s.SpoolDir, "*")); len(m) != 0 {
		t.Errorf("strict never spools, found %v", m)
	}
}

func TestLG_StrictEndFailure(t *testing.T) {
	s := lgxSettings(t, logStrict, logThresholdAll, logScopeAll)
	ls := lgxNew(s, "replaceall", &lgxRec{failFrom: 2}, func() {})
	if err := ls.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	err := ls.End(0, nil)
	if err == nil || !ls.MustAbort(err) {
		t.Errorf("strict: End = %v, MustAbort = %v; want an error that is flagged", err, ls.MustAbort(err))
	}
	if m := spGlob(t, filepath.Join(s.SpoolDir, "*")); len(m) != 0 {
		t.Errorf("strict never spools, found %v", m)
	}
}

func TestLG_MergeBeforeBegin(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdAll, logScopeAll)
	old := "2026-10-18T08:00:00.000Z [INFO] id=a1 phase=begin op=x logged=spool"
	spWrite(t, filepath.Join(s.SpoolDir, "a1.log"), old+"\n")

	ls := newLogSession(s, "replaceall", lgxClock(), fileLogIO(), func() {}, lgxFields)
	if err := ls.Begin(); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	lines := strings.Split(strings.TrimRight(spRead(t, s.File), "\n"), "\n")
	if len(lines) != 2 || lines[0] != old+" late=true" || !strings.Contains(lines[1], "id="+ls.ID()+" phase=begin") {
		t.Errorf("main log = %q; want the merged late record first, then this run's begin record", lines)
	}
	if m := spGlob(t, filepath.Join(s.SpoolDir, "*")); len(m) != 0 {
		t.Errorf("the spool folder should be empty after the merge, found %v", m)
	}
}

func TestLG_MergeSkippedWhenInactive(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdAll, logScopeWrites)
	spWrite(t, filepath.Join(s.SpoolDir, "a1.log"), "2026-10-18T08:00:00.000Z [INFO] id=a1 phase=begin op=x logged=spool\n")

	ls := newLogSession(s, "show", lgxClock(), fileLogIO(), func() {}, lgxFields)
	_ = ls.Begin()
	if m := spGlob(t, filepath.Join(s.SpoolDir, "a1.log")); len(m) != 1 {
		t.Errorf("an inactive session must leave the spool alone, found %v", m)
	}
	if m := spGlob(t, s.File); len(m) != 0 {
		t.Errorf("an inactive session must not create the log, found %v", m)
	}
}

func TestLG_MergeFailureDoesNotBlock(t *testing.T) {
	s := lgxSettings(t, logPermissive, logThresholdAll, logScopeAll)
	spWrite(t, filepath.Join(s.SpoolDir, "a1.log"), "2026-10-18T08:00:00.000Z [INFO] id=a1 phase=begin op=x logged=spool\n")

	ls := lgxNew(s, "replaceall", &lgxRec{failFrom: 1}, func() {})
	if err := ls.Begin(); err != nil || ls.MustAbort(err) {
		t.Fatalf("Begin = %v; a failed merge must not block the run", err)
	}
	if m := spGlob(t, filepath.Join(s.SpoolDir, "a1.log.merging")); len(m) != 1 {
		t.Errorf("the old records must be kept for the next merge, found %v", m)
	}
	if m := spGlob(t, filepath.Join(s.SpoolDir, ls.ID()+".log")); len(m) != 1 {
		t.Errorf("this run's own record must be spooled, found %v", m)
	}
}
