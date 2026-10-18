package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

var rrIn = []string{"v1 x", "b", "v2 v3", "c"}

func TestRRD_ChangedMatchesAndCaptureGroups(t *testing.T) {
	keep := append([]string(nil), rrIn...)
	res, err := execReplaceAllRegexDetail(rrIn, `v(\d)`, "V$1")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Lines, []string{"V1 x", "b", "V2 V3", "c"}) {
		t.Errorf("Lines = %q", res.Lines)
	}
	if !reflect.DeepEqual(res.Changed, []int{1, 3}) || res.Occurrences != 3 {
		t.Errorf("Changed = %v, Occurrences = %d", res.Changed, res.Occurrences)
	}
	if !reflect.DeepEqual(rrIn, keep) {
		t.Errorf("input was mutated: %q", rrIn)
	}
}

func TestRRD_Errors(t *testing.T) {
	cases := []struct{ pattern, repl, want string }{
		{"(", "x", "invalid regex"},
		{"zzz+", "x", "No lines match regex: zzz+"},
		{"b", "b", "No lines match regex: b"},
	}
	for _, c := range cases {
		_, err := execReplaceAllRegexDetail(rrIn, c.pattern, c.repl)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("pattern %q: err = %v, want %q", c.pattern, err, c.want)
		}
	}
}

func TestRRD_Verify(t *testing.T) {
	res, _ := execReplaceAllRegexDetail(rrIn, `v(\d)`, "V$1")
	pass, err := verifyReplaceAllRegexOnDisk(rvFile(t, res.Lines), `v(\d)`, res)
	if err != nil {
		t.Fatal(err)
	}
	if !pass.OK || len(pass.Bad) != 0 || pass.Remaining != 0 || pass.WantRemaining != 0 {
		t.Errorf("pass: %+v", pass)
	}
	disk := []string{"V1 x", "b", "v2 v3", "c"}
	fail, err := verifyReplaceAllRegexOnDisk(rvFile(t, disk), `v(\d)`, res)
	if err != nil {
		t.Fatal(err)
	}
	if fail.OK || !reflect.DeepEqual(fail.Bad, []int{3}) || fail.Remaining != 2 || fail.WantRemaining != 0 {
		t.Errorf("fail: %+v", fail)
	}
}

func TestRRD_VerifyWhenReplacementRecreatesPattern(t *testing.T) {
	res, _ := execReplaceAllRegexDetail([]string{"a", "b"}, "a", "aa")
	v, err := verifyReplaceAllRegexOnDisk(rvFile(t, res.Lines), "a", res)
	if err != nil {
		t.Fatal(err)
	}
	if !v.OK || v.Remaining != 2 || v.WantRemaining != 2 {
		t.Errorf("got %+v", v)
	}
}

func TestRRV_PassAndVerbose(t *testing.T) {
	p := rvFile(t, rrIn)
	var out bytes.Buffer
	if code := runReplaceAllRegexVerified(rrIn, p, `v(\d)`, "V$1", true, &out, writeLines); code != 0 {
		t.Fatalf("code = %d:\n%s", code, out.String())
	}
	got, _ := readLines(p)
	if strings.Join(got, "|") != "V1 x|b|V2 V3|c" {
		t.Errorf("file = %q", got)
	}
	rvNeed(t, out.String(), "Regex-replaced on 2 line(s) (4 total)", "VERIFY: PASS", "3 occurrence(s)", "changed lines: 1,3", "=== VERIFY ===")
	if n := strings.Count(out.String(), "\n> "); n != 2 {
		t.Errorf("marked lines = %d, want 2:\n%s", n, out.String())
	}
	var quiet bytes.Buffer
	runReplaceAllRegexVerified(rrIn, rvFile(t, rrIn), `v(\d)`, "V$1", false, &quiet, writeLines)
	if strings.Contains(quiet.String(), "=== VERIFY ===") {
		t.Errorf("windows must only print with verbose:\n%s", quiet.String())
	}
}

func TestRRV_SkippedHitFailsWithExit3(t *testing.T) {
	skip := func(path string, ls []string) error {
		cp := append([]string(nil), ls...)
		cp[2] = "v2 v3"
		return writeLines(path, cp)
	}
	var out bytes.Buffer
	code := runReplaceAllRegexVerified(rrIn, rvFile(t, rrIn), `v(\d)`, "V$1", false, &out, skip)
	if code != 3 {
		t.Fatalf("code = %d, want 3:\n%s", code, out.String())
	}
	rvNeed(t, out.String(), "VERIFY: FAIL", "mismatched lines: 3")
}

func TestRRV_ErrorsExit1AndLeaveFile(t *testing.T) {
	cases := []struct{ pattern, want string }{
		{"(", "invalid regex"},
		{"zzz+", "No lines match regex: zzz+"},
	}
	for _, c := range cases {
		p := rvFile(t, rrIn)
		var out bytes.Buffer
		if code := runReplaceAllRegexVerified(rrIn, p, c.pattern, "x", false, &out, writeLines); code != 1 {
			t.Errorf("pattern %q: code = %d, want 1", c.pattern, code)
		}
		rvNeed(t, out.String(), c.want)
		got, _ := readLines(p)
		if strings.Join(got, "|") != strings.Join(rrIn, "|") {
			t.Errorf("pattern %q changed the file: %q", c.pattern, got)
		}
	}
	boom := func(string, []string) error { return errors.New("boom") }
	var out bytes.Buffer
	if code := runReplaceAllRegexVerified(rrIn, rvFile(t, rrIn), "v", "x", false, &out, boom); code != 1 {
		t.Errorf("write error: code = %d, want 1", code)
	}
	rvNeed(t, out.String(), "Error writing file: boom")
}
