package main

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func radFile(t *testing.T, lines []string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rad.txt")
	if err := writeLines(p, lines); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestRAD_ChangedLinesAndOccurrences(t *testing.T) {
	in := []string{"a x", "b", "x x", "c", "x"}
	keep := append([]string(nil), in...)
	res, err := execReplaceAllDetail(in, "x", "Y")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Changed, []int{1, 3, 5}) {
		t.Errorf("Changed = %v", res.Changed)
	}
	if res.Occurrences != 4 {
		t.Errorf("Occurrences = %d, want 4", res.Occurrences)
	}
	if !reflect.DeepEqual(res.Lines, []string{"a Y", "b", "Y Y", "c", "Y"}) {
		t.Errorf("Lines = %q", res.Lines)
	}
	if !reflect.DeepEqual(in, keep) {
		t.Errorf("input was mutated: %q", in)
	}
}

func TestRAD_Errors(t *testing.T) {
	if _, err := execReplaceAllDetail([]string{"a"}, "", "b"); err == nil {
		t.Error("empty search must error")
	}
	_, err := execReplaceAllDetail([]string{"a"}, "zzz", "b")
	if err == nil || !strings.Contains(err.Error(), "No lines contain: zzz") {
		t.Errorf("err = %v", err)
	}
}

func TestRAD_ReplacementContainsMatch(t *testing.T) {
	res, err := execReplaceAllDetail([]string{"a a", "b"}, "a", "aa")
	if err != nil {
		t.Fatal(err)
	}
	if res.Occurrences != 2 || !reflect.DeepEqual(res.Changed, []int{1}) || res.Lines[0] != "aa aa" {
		t.Errorf("got %+v", res)
	}
}

func TestRAD_VerifyPass(t *testing.T) {
	res, _ := execReplaceAllDetail([]string{"a x", "b", "x x"}, "x", "Y")
	v, err := verifyReplaceAllOnDisk(radFile(t, res.Lines), "x", res)
	if err != nil {
		t.Fatal(err)
	}
	if !v.OK || len(v.Bad) != 0 || v.Remaining != 0 || v.WantRemaining != 0 {
		t.Errorf("got %+v", v)
	}
}

func TestRAD_VerifyPassWhenReplacementContainsMatch(t *testing.T) {
	res, _ := execReplaceAllDetail([]string{"a a", "b"}, "a", "aa")
	v, err := verifyReplaceAllOnDisk(radFile(t, res.Lines), "a", res)
	if err != nil {
		t.Fatal(err)
	}
	if !v.OK || v.Remaining != 4 || v.WantRemaining != 4 {
		t.Errorf("got %+v", v)
	}
}

func TestRAD_VerifyFailsWhenAHitWasSkipped(t *testing.T) {
	res, _ := execReplaceAllDetail([]string{"x", "b", "x", "c", "x"}, "x", "Y")
	disk := []string{"Y", "b", "x", "c", "Y"}
	v, err := verifyReplaceAllOnDisk(radFile(t, disk), "x", res)
	if err != nil {
		t.Fatal(err)
	}
	if v.OK || !reflect.DeepEqual(v.Bad, []int{3}) || v.Remaining != 1 || v.WantRemaining != 0 {
		t.Errorf("got %+v", v)
	}
}

func TestRAD_VerifyFailsOnLineCount(t *testing.T) {
	res, _ := execReplaceAllDetail([]string{"x", "b", "x", "c", "x"}, "x", "Y")
	short, err := verifyReplaceAllOnDisk(radFile(t, res.Lines[:2]), "x", res)
	if err != nil {
		t.Fatal(err)
	}
	if short.OK || short.WantLines != 5 || short.GotLines != 2 || len(short.Bad) == 0 {
		t.Errorf("short: %+v", short)
	}
	extra := append(append([]string(nil), res.Lines...), "stray")
	long, err := verifyReplaceAllOnDisk(radFile(t, extra), "x", res)
	if err != nil {
		t.Fatal(err)
	}
	if long.OK || long.GotLines != 6 {
		t.Errorf("long: %+v", long)
	}
}

func TestRAD_LineRanges(t *testing.T) {
	cases := []struct {
		in   []int
		want string
	}{
		{nil, ""},
		{[]int{5}, "5"},
		{[]int{1, 3, 5}, "1,3,5"},
		{[]int{1, 2, 3, 5}, "1-3,5"},
		{[]int{2, 3, 7, 8, 9}, "2-3,7-9"},
	}
	for _, c := range cases {
		if got := formatLineRanges(c.in); got != c.want {
			t.Errorf("formatLineRanges(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRAD_FormatVerify(t *testing.T) {
	res, _ := execReplaceAllDetail([]string{"a x", "b", "x x", "c", "x"}, "x", "Y")
	pass := formatReplaceVerify(res, replaceVerify{OK: true})
	for _, w := range []string{"VERIFY: PASS", "4 occurrence(s)", "3 line(s)", "changed lines: 1,3,5"} {
		if !strings.Contains(pass, w) {
			t.Errorf("PASS text lacks %q: %s", w, pass)
		}
	}
	fail := formatReplaceVerify(res, replaceVerify{Bad: []int{3}, Remaining: 1})
	for _, w := range []string{"VERIFY: FAIL", "mismatched lines: 3"} {
		if !strings.Contains(fail, w) {
			t.Errorf("FAIL text lacks %q: %s", w, fail)
		}
	}
}
