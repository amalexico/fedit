package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func lrRead(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func lrEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLR_Append(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "deeper", "a.log")
	if err := logAppendFile(p, "one"); err != nil {
		t.Fatalf("first append: %v", err)
	}
	if err := logAppendFile(p, "two"); err != nil {
		t.Fatalf("second append: %v", err)
	}
	if got := lrRead(t, p); got != "one\ntwo\n" {
		t.Errorf("file = %q, want %q", got, "one\ntwo\n")
	}
}

func TestLR_Size(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.log")
	if _, err := logFileSize(p); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("size of a missing file: err = %v, want os.ErrNotExist", err)
	}
	if err := logAppendFile(p, "one"); err != nil {
		t.Fatalf("append: %v", err)
	}
	size, err := logFileSize(p)
	if err != nil || size != 4 {
		t.Errorf("size = %d, %v; want 4, nil", size, err)
	}
}

func TestLR_Rotate(t *testing.T) {
	const stamp = ".20261018T081715Z"
	p := filepath.Join(t.TempDir(), "a.log")
	est := time.FixedZone("EST", -5*3600)
	at := time.Date(2026, 10, 18, 3, 17, 15, 0, est) // 08:17:15 UTC

	for i, suffix := range []string{"", ".1", ".2"} {
		line := []string{"one", "two", "three"}[i]
		if err := logAppendFile(p, line); err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
		got, err := logRotateFileAt(p, at)
		if err != nil {
			t.Fatalf("rotate %d: %v", i, err)
		}
		if want := p + stamp + suffix; got != want {
			t.Errorf("rotate %d: new name %q, want %q", i, got, want)
		}
		if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("rotate %d: the original file should be gone, stat err = %v", i, err)
		}
		if c := lrRead(t, got); c != line+"\n" {
			t.Errorf("rotate %d: rotated content %q, want %q", i, c, line+"\n")
		}
	}
	if c := lrRead(t, p+stamp); c != "one\n" {
		t.Errorf("the first rotated file must stay untouched, got %q", c)
	}

	if _, err := logRotateFileAt(filepath.Join(t.TempDir(), "none.log"), at); err == nil {
		t.Errorf("rotating a missing file should fail")
	}
}

func TestLR_FileLogIO(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a.log")
	lio := fileLogIO()
	if lio.Append == nil || lio.Size == nil || lio.Rotate == nil {
		t.Fatalf("fileLogIO must fill Append, Size and Rotate")
	}
	if err := lio.Append(p, "x"); err != nil {
		t.Fatalf("Append: %v", err)
	}
	if size, err := lio.Size(p); err != nil || size != 2 {
		t.Errorf("Size = %d, %v; want 2, nil", size, err)
	}
	if err := lio.Rotate(p); err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	matches, _ := filepath.Glob(p + ".*")
	if len(matches) != 1 {
		t.Errorf("want exactly one rotated file next to %s, got %v", p, matches)
	}
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the original file should be gone after Rotate, stat err = %v", err)
	}
}

func TestLR_ExpandPath(t *testing.T) {
	env := lrEnv(map[string]string{"APPDATA": "C:/Users/u/AppData/Roaming", "X": "y"})
	home := "/home/u"
	cases := []struct{ in, want string }{
		{"~", home},
		{"~/x.log", filepath.Join(home, "x.log")},
		{`~\x.log`, filepath.Join(home, "x.log")},
		{"~/a/b.log", filepath.Join(home, "a/b.log")},
		{"%APPDATA%/fedit/x.log", "C:/Users/u/AppData/Roaming/fedit/x.log"},
		{"%NOPE%/x", "%NOPE%/x"},
		{"a%X%b%X%c", "aybyc"},
		{"100%", "100%"},
		{"~other/x", "~other/x"},
		{"/abs/x.log", "/abs/x.log"},
	}
	for _, c := range cases {
		if got := expandLogPath(c.in, home, env); got != c.want {
			t.Errorf("expandLogPath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if got := expandLogPath("~/x.log", "", env); got != "~/x.log" {
		t.Errorf("without a home directory ~ must be left alone, got %q", got)
	}
}

func TestLR_DefaultPath(t *testing.T) {
	failCfg := func() (string, error) { return "", errors.New("no config dir") }
	okCfg := func() (string, error) { return "/cfg", nil }

	got, err := defaultLogPath(lrEnv(map[string]string{"LOCALAPPDATA": "C:/L"}), failCfg)
	if want := filepath.Join("C:/L", "fedit", "fedit.log"); err != nil || got != want {
		t.Errorf("with LOCALAPPDATA: %q, %v; want %q, nil", got, err, want)
	}

	got, err = defaultLogPath(lrEnv(map[string]string{"LOCALAPPDATA": ""}), okCfg)
	if want := filepath.Join("/cfg", "fedit", "fedit.log"); err != nil || got != want {
		t.Errorf("fallback to the user config dir: %q, %v; want %q, nil", got, err, want)
	}

	_, err = defaultLogPath(lrEnv(nil), failCfg)
	if err == nil || !strings.Contains(err.Error(), "log_file") {
		t.Errorf("with no location at all the error should tell the user to set log_file, got %v", err)
	}
}
