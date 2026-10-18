package main

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// A replacement that contains a newline adds lines to the file. The expected
// lines must be flattened and Changed renumbered to the new file's numbering.

func TestML_LiteralMultiLineReplacementIsFlattened(t *testing.T) {
	res, err := execReplaceAllDetail([]string{"a x", "b", "c x"}, "x", "1\n2")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Lines, []string{"a 1", "2", "b", "c 1", "2"}) {
		t.Errorf("Lines = %q", res.Lines)
	}
	if !reflect.DeepEqual(res.Changed, []int{1, 2, 4, 5}) || res.Occurrences != 2 {
		t.Errorf("Changed = %v, Occurrences = %d", res.Changed, res.Occurrences)
	}
}

func TestML_RegexMultiLineReplacementIsFlattened(t *testing.T) {
	res, err := execReplaceAllRegexDetail([]string{"x1", "b"}, `x(\d)`, "n$1\nz")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(res.Lines, []string{"n1", "z", "b"}) {
		t.Errorf("Lines = %q", res.Lines)
	}
	if !reflect.DeepEqual(res.Changed, []int{1, 2}) || res.Occurrences != 1 {
		t.Errorf("Changed = %v, Occurrences = %d", res.Changed, res.Occurrences)
	}
}

func TestML_RunnersVerifyMultiLineReplacement(t *testing.T) {
	lit := []string{"a x", "b", "c x"}
	p := rvFile(t, lit)
	var out bytes.Buffer
	if code := runReplaceAllVerified(lit, p, "x", "1\n2", false, &out, writeLines); code != 0 {
		t.Fatalf("literal: code = %d, want 0:\n%s", code, out.String())
	}
	got, _ := readLines(p)
	if strings.Join(got, "|") != "a 1|2|b|c 1|2" {
		t.Errorf("literal file = %q", got)
	}
	rvNeed(t, out.String(), "VERIFY: PASS", "changed lines: 1-2,4-5")

	rx := []string{"x1", "b"}
	p2 := rvFile(t, rx)
	var out2 bytes.Buffer
	if code := runReplaceAllRegexVerified(rx, p2, `x(\d)`, "n$1\nz", false, &out2, writeLines); code != 0 {
		t.Fatalf("regex: code = %d, want 0:\n%s", code, out2.String())
	}
	got2, _ := readLines(p2)
	if strings.Join(got2, "|") != "n1|z|b" {
		t.Errorf("regex file = %q", got2)
	}
	rvNeed(t, out2.String(), "VERIFY: PASS", "changed lines: 1-2")
}
