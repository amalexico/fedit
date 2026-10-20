package main

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

// lsFake is an in-memory stand-in for the log file operations.
type lsFake struct {
	lines      []string
	size       int64
	rotated    int
	failAppend bool
}

func (f *lsFake) io() logIO {
	return logIO{
		Append: func(path, line string) error {
			if f.failAppend {
				return errors.New("disk full")
			}
			f.lines = append(f.lines, line)
			return nil
		},
		Size: func(path string) (int64, error) { return f.size, nil },
		Rotate: func(path string) error {
			f.rotated++
			return nil
		},
	}
}

// lsClock returns a clock that starts at a fixed instant and moves forward
// 250 ms on every call.
func lsClock() func() time.Time {
	t := time.Date(2026, 10, 18, 8, 17, 15, 40_000_000, time.UTC)
	return func() time.Time {
		r := t
		t = t.Add(250 * time.Millisecond)
		return r
	}
}

func lsNew(mode logMode, maxMB int, f *lsFake) *logSink {
	s := newLogSink(mode, "audit.log", maxMB, lsClock(), f.io())
	s.ID = "t1"
	return s
}

func TestLS_BeginEnd(t *testing.T) {
	f := &lsFake{}
	s := lsNew(logPermissive, 0, f)
	if err := s.Begin([]logKV{{"op", "replaceall"}, {"file", "a.txt"}}); err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if err := s.End(0, []logKV{{"replacements", "3"}}); err != nil {
		t.Fatalf("End: %v", err)
	}
	want := []string{
		"2026-10-18T08:17:15.040Z [INFO] id=t1 phase=begin op=replaceall file=a.txt",
		"2026-10-18T08:17:15.290Z [INFO] id=t1 phase=end exit=0 ms=250 replacements=3",
	}
	if !reflect.DeepEqual(f.lines, want) {
		t.Errorf("records =\n  %q\nwant\n  %q", f.lines, want)
	}
}

func TestLS_NewSinkHasID(t *testing.T) {
	s := newLogSink(logStrict, "audit.log", 0, lsClock(), (&lsFake{}).io())
	if len(s.ID) != 16 {
		t.Errorf("a new sink should get a 16 character id, got %q", s.ID)
	}
}

func TestLS_EndLevels(t *testing.T) {
	cases := []struct {
		code  int
		level string
	}{{0, "INFO"}, {1, "WARNING"}, {2, "ERROR"}, {3, "ERROR"}}
	for _, c := range cases {
		f := &lsFake{}
		s := lsNew(logPermissive, 0, f)
		if err := s.Begin(nil); err != nil {
			t.Fatalf("Begin: %v", err)
		}
		if err := s.End(c.code, nil); err != nil {
			t.Fatalf("End: %v", err)
		}
		last := f.lines[len(f.lines)-1]
		if !strings.Contains(last, " ["+c.level+"] ") || !strings.Contains(last, fmt.Sprintf(" exit=%d ", c.code)) {
			t.Errorf("exit %d: want level %s and exit=%d, got %q", c.code, c.level, c.code, last)
		}
	}
}

func TestLS_OffWritesNothing(t *testing.T) {
	f := &lsFake{failAppend: true, size: 1 << 40}
	s := lsNew(logOff, 1, f)
	if err := s.Begin([]logKV{{"op", "show"}}); err != nil {
		t.Errorf("off: Begin returned %v, want nil", err)
	}
	if err := s.End(0, nil); err != nil {
		t.Errorf("off: End returned %v, want nil", err)
	}
	if len(f.lines) != 0 || f.rotated != 0 {
		t.Errorf("off must not write or rotate, got %d lines and %d rotations", len(f.lines), f.rotated)
	}
}

func TestLS_WriteFailurePolicy(t *testing.T) {
	for _, c := range []struct {
		mode      logMode
		mustAbort bool
	}{{logStrict, true}, {logPermissive, false}} {
		f := &lsFake{failAppend: true}
		s := lsNew(c.mode, 0, f)
		err := s.Begin(nil)
		if err == nil {
			t.Fatalf("%s: Begin should report the write failure", c.mode)
		}
		if got := s.MustAbort(err); got != c.mustAbort {
			t.Errorf("%s: MustAbort = %v, want %v", c.mode, got, c.mustAbort)
		}
	}
	s := lsNew(logStrict, 0, &lsFake{})
	if s.MustAbort(nil) {
		t.Errorf("MustAbort(nil) must be false")
	}
}

func TestLS_EndWithoutBegin(t *testing.T) {
	f := &lsFake{failAppend: true}
	s := lsNew(logPermissive, 0, f)
	if err := s.Begin(nil); err == nil {
		t.Fatalf("Begin should fail while the log is not writable")
	}
	f.failAppend = false
	if err := s.End(0, nil); err != nil {
		t.Fatalf("End: %v", err)
	}
	want := []string{"2026-10-18T08:17:15.290Z [INFO] id=t1 phase=end exit=0"}
	if !reflect.DeepEqual(f.lines, want) {
		t.Errorf("an end record without a begin has no ms: got %q, want %q", f.lines, want)
	}
}

func TestLS_Rotation(t *testing.T) {
	const mb = 1024 * 1024
	cases := []struct {
		maxMB       int
		size        int64
		wantRotated int
	}{
		{5, 5 * mb, 1},
		{5, 5*mb - 1, 0},
		{0, 100 * mb, 0},
		{1, 10 * mb, 1},
	}
	for _, c := range cases {
		f := &lsFake{size: c.size}
		s := lsNew(logPermissive, c.maxMB, f)
		if err := s.Begin(nil); err != nil {
			t.Fatalf("Begin: %v", err)
		}
		if f.rotated != c.wantRotated {
			t.Errorf("max %d MB, size %d: rotated %d times, want %d", c.maxMB, c.size, f.rotated, c.wantRotated)
		}
		if len(f.lines) != 1 {
			t.Errorf("max %d MB, size %d: the record must still be written, got %d lines", c.maxMB, c.size, len(f.lines))
		}
	}
}

func TestLS_ParseMode(t *testing.T) {
	ok := []struct {
		in   string
		want logMode
	}{{"off", logOff}, {"permissive", logPermissive}, {"strict", logStrict}, {" Strict ", logStrict}, {"PERMISSIVE", logPermissive}, {"", logOff}}
	for _, c := range ok {
		got, err := parseLogMode(c.in)
		if err != nil || got != c.want {
			t.Errorf("parseLogMode(%q) = %q, %v; want %q, nil", c.in, got, err, c.want)
		}
	}
	_, err := parseLogMode("verbose")
	if err == nil {
		t.Fatalf("parseLogMode(verbose) should fail")
	}
	for _, name := range []string{"off", "permissive", "strict"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("the error should list %q, got %q", name, err.Error())
		}
	}
}
