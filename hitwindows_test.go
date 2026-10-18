package main

import (
	"fmt"
	"strings"
	"testing"
)

const hwHead = "\n=== VERIFY ===\n"

func hwLines(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("l%d", i+1)
	}
	return out
}

func hwRow(marker string, n int) string {
	return fmt.Sprintf("%s%4d | l%d\n", marker, n, n)
}

func TestHW_SingleHit(t *testing.T) {
	got := formatHitWindows(hwLines(10), []int{5}, 2, 2)
	want := hwHead + hwRow("  ", 3) + hwRow("  ", 4) + hwRow("> ", 5) +
		hwRow("  ", 6) + hwRow("  ", 7) + "--- 10 total lines ---\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestHW_OverlappingWindowsMerge(t *testing.T) {
	got := formatHitWindows(hwLines(10), []int{4, 6}, 2, 2)
	if strings.Contains(got, "...") {
		t.Errorf("overlapping windows must merge without a gap line:\n%s", got)
	}
	if n := strings.Count(got, "\n> "); n != 2 {
		t.Errorf("marked lines = %d, want 2:\n%s", n, got)
	}
	for _, ln := range []int{2, 8} {
		if !strings.Contains(got, hwRow("  ", ln)) {
			t.Errorf("line %d missing:\n%s", ln, got)
		}
	}
	for _, ln := range []int{1, 9} {
		if strings.Contains(got, hwRow("  ", ln)) {
			t.Errorf("line %d must not be shown:\n%s", ln, got)
		}
	}
}

func TestHW_DistantWindowsGetGapLine(t *testing.T) {
	got := formatHitWindows(hwLines(12), []int{2, 9}, 1, 1)
	want := hwHead + hwRow("  ", 1) + hwRow("> ", 2) + hwRow("  ", 3) + "  ...\n" +
		hwRow("  ", 8) + hwRow("> ", 9) + hwRow("  ", 10) + "--- 12 total lines ---\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestHW_TouchingWindowsJoin(t *testing.T) {
	got := formatHitWindows(hwLines(10), []int{3, 6}, 1, 1)
	if strings.Contains(got, "...") {
		t.Errorf("touching windows must join:\n%s", got)
	}
	for _, ln := range []int{2, 4, 5, 7} {
		if !strings.Contains(got, hwRow("  ", ln)) {
			t.Errorf("line %d missing:\n%s", ln, got)
		}
	}
	if !strings.Contains(got, hwRow("> ", 3)) || !strings.Contains(got, hwRow("> ", 6)) {
		t.Errorf("hits not marked:\n%s", got)
	}
}

func TestHW_EdgesAreClamped(t *testing.T) {
	got := formatHitWindows(hwLines(5), []int{1, 5}, 1, 1)
	want := hwHead + hwRow("> ", 1) + hwRow("  ", 2) + "  ...\n" +
		hwRow("  ", 4) + hwRow("> ", 5) + "--- 5 total lines ---\n"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestHW_EmptyAndOutOfRange(t *testing.T) {
	want := hwHead + "--- 5 total lines ---\n"
	if got := formatHitWindows(hwLines(5), nil, 2, 2); got != want {
		t.Errorf("nil: %q", got)
	}
	if got := formatHitWindows(hwLines(5), []int{0, 99}, 2, 2); got != want {
		t.Errorf("out of range: %q", got)
	}
}

func TestHW_EveryHitIsShownUncapped(t *testing.T) {
	var hits []int
	for i := 10; i <= 1000; i += 10 {
		hits = append(hits, i)
	}
	got := formatHitWindows(hwLines(1000), hits, 1, 1)
	if n := strings.Count(got, "\n> "); n != 100 {
		t.Errorf("marked lines = %d, want 100", n)
	}
	if !strings.Contains(got, hwRow("> ", 1000)) || !strings.Contains(got, hwRow("> ", 10)) {
		t.Error("first or last hit missing")
	}
}
