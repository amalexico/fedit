package main

import (
	"strings"
	"testing"
)

func TestDecodeHexAnchor_Valid(t *testing.T) {
	want := `case "replace":`
	got, err := decodeHexAnchor("matchhex", thEncode(want))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestDecodeHexAnchor_QuotesAndBackslashesSurvive(t *testing.T) {
	want := `fmt.Fprintf(os.Stderr, "x\n")`
	got, err := decodeHexAnchor("matchhex", thEncode(want))
	if err != nil || got != want {
		t.Errorf("got %q, %v; want %q", got, err, want)
	}
}

func TestDecodeHexAnchor_Empty(t *testing.T) {
	got, err := decodeHexAnchor("matchhex", "")
	if err != nil || got != "" {
		t.Errorf("got %q, %v; want empty and nil", got, err)
	}
}

func TestDecodeHexAnchor_InvalidNamesFlag(t *testing.T) {
	for _, flagName := range []string{"matchhex", "endmatchhex"} {
		_, err := decodeHexAnchor(flagName, "zz")
		if err == nil {
			t.Fatalf("%s: expected error for invalid hex", flagName)
		}
		if !strings.HasPrefix(err.Error(), flagName+": invalid hex string") {
			t.Errorf("%s: error %q does not name the flag", flagName, err)
		}
	}
}

func TestMatchHex_FindsLineWithDoubleQuote(t *testing.T) {
	lines := []string{"switch op {", `case "insert":`, `case "replace":`, "}"}
	anchor, _ := decodeHexAnchor("matchhex", thEncode(`case "replace":`))
	line, err := resolveNth(findMatches(lines, anchor), 1)
	if err != nil || line != 3 {
		t.Errorf("got line %d, %v; want line 3", line, err)
	}
}

func TestMatchHex_NthLast(t *testing.T) {
	lines := []string{`a "x"`, "b", `c "x"`, "d"}
	anchor, _ := decodeHexAnchor("matchhex", thEncode(`"x"`))
	line, err := resolveNth(findMatches(lines, anchor), -1)
	if err != nil || line != 3 {
		t.Errorf("got line %d, %v; want line 3", line, err)
	}
}

func TestMatchHex_RangeWithQuotedEndAnchors(t *testing.T) {
	lines := []string{"func a() {", `  x := "start"`, "  y := 1", `  z := "end"`, "}"}
	s, _ := decodeHexAnchor("matchhex", thEncode(`"start"`))
	e, _ := decodeHexAnchor("endmatchhex", thEncode(`"end"`))
	start, end, err := resolveSourceLines(lines, 0, 0, s, e, 1)
	if err != nil || start != 2 || end != 4 {
		t.Errorf("got %d-%d, %v; want 2-4", start, end, err)
	}
}
