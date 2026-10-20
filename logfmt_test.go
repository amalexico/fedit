package main

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestLF_Timestamp(t *testing.T) {
	est := time.FixedZone("EST", -5*3600)
	cases := []struct {
		name string
		in   time.Time
		want string
	}{
		{"utc", time.Date(2026, 10, 18, 8, 17, 15, 40_000_000, time.UTC), "2026-10-18T08:17:15.040Z"},
		{"other zone is converted", time.Date(2026, 10, 18, 3, 17, 15, 40_000_000, est), "2026-10-18T08:17:15.040Z"},
		{"zero millis keep the dot", time.Date(2026, 10, 18, 8, 17, 15, 0, time.UTC), "2026-10-18T08:17:15.000Z"},
		{"sub-millisecond is truncated", time.Date(2026, 10, 18, 8, 17, 15, 40_999_999, time.UTC), "2026-10-18T08:17:15.040Z"},
	}
	for _, c := range cases {
		if got := logTimestamp(c.in); got != c.want {
			t.Errorf("%s: logTimestamp = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestLF_Quote(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", `""`},
		{"abc", "abc"},
		{"C:\\Users\\kehsi\\a.txt", "C:\\Users\\kehsi\\a.txt"},
		{"a b", `"a b"`},
		{"say \"hi\"", `"say \"hi\""`},
		{"a=b", `"a=b"`},
		{"line1\nline2", `"line1\nline2"`},
		{"tab\there", `"tab\there"`},
		{"cr\r", `"cr\r"`},
		{"\x01", `"\u0001"`},
		{"h\u00e9llo", "h\u00e9llo"},
		{"back\\slash with space", `"back\\slash with space"`},
	}
	for _, c := range cases {
		if got := logQuote(c.in); got != c.want {
			t.Errorf("logQuote(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLF_Key(t *testing.T) {
	cases := []struct{ in, want string }{
		{"op", "op"},
		{"sha256_before", "sha256_before"},
		{"ok.key-1_", "ok.key-1_"},
		{"a b", "a_b"},
		{"x=y", "x_y"},
		{"", "_"},
	}
	for _, c := range cases {
		if got := logKey(c.in); got != c.want {
			t.Errorf("logKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLF_FormatLine(t *testing.T) {
	ts := time.Date(2026, 10, 18, 8, 17, 15, 40_000_000, time.UTC)

	got := formatLogLine(ts, logInfo, []logKV{{"id", "a1b2"}, {"op", "replaceall"}, {"file", "C:\\x y\\f.txt"}, {"msg", ""}})
	want := `2026-10-18T08:17:15.040Z [INFO] id=a1b2 op=replaceall file="C:\\x y\\f.txt" msg=""`
	if got != want {
		t.Errorf("formatLogLine =\n  %q\nwant\n  %q", got, want)
	}

	if got, want := formatLogLine(ts, logWarning, nil), "2026-10-18T08:17:15.040Z [WARNING]"; got != want {
		t.Errorf("no fields: got %q, want %q", got, want)
	}

	got = formatLogLine(ts, logError, []logKV{{"msg", "a\nb"}})
	if strings.Contains(got, "\n") {
		t.Errorf("a log record must stay on one line, got %q", got)
	}
	if want := `2026-10-18T08:17:15.040Z [ERROR] msg="a\nb"`; got != want {
		t.Errorf("newline value: got %q, want %q", got, want)
	}

	got = formatLogLine(ts, logInfo, []logKV{{"bad key", "v"}})
	if want := "2026-10-18T08:17:15.040Z [INFO] bad_key=v"; got != want {
		t.Errorf("key sanitizing: got %q, want %q", got, want)
	}
}

func TestLF_NewID(t *testing.T) {
	id1, id2 := logNewID(), logNewID()
	if len(id1) != 16 {
		t.Fatalf("id %q has length %d, want 16", id1, len(id1))
	}
	for _, c := range id1 {
		if !strings.ContainsRune("0123456789abcdef", c) {
			t.Errorf("id %q has a non-hex character %q", id1, c)
		}
	}
	if id1 == id2 {
		t.Errorf("two ids must differ, both were %q", id1)
	}
}

func TestLF_TextFields(t *testing.T) {
	got := logTextFields("text", []byte("hello"))
	want := []logKV{
		{"text_len", "5"},
		{"text_sha256", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("logTextFields(hello) = %v, want %v", got, want)
	}

	got = logTextFields("match", nil)
	want = []logKV{
		{"match_len", "0"},
		{"match_sha256", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("logTextFields(empty) = %v, want %v", got, want)
	}
}
