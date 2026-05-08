package main

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// ── helpers ──────────────────────────────────────────────────────────────────

// testLines creates a slice of N lines: "line 1", "line 2", ..., "line N".
func testLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	return lines
}

// writeTestFile writes lines to a temp file and returns its path.
// The file is removed automatically when the test ends.
func writeTestFile(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp("", "fedit_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := writeLines(f.Name(), lines); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// readTestFile reads a file and returns its lines.
func readTestFile(t *testing.T, path string) []string {
	t.Helper()
	lines, err := readLines(path)
	if err != nil {
		t.Fatalf("readTestFile: %v", err)
	}
	return lines
}

// ── execMove ─────────────────────────────────────────────────────────────────

func TestExecMove_BasicMoveForward(t *testing.T) {
	// Move lines 3-4 to after line 7 in a 10-line file.
	lines := testLines(10)
	result, firstDest, err := execMove(lines, 3, 4, 7, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: 1, 2, 5, 6, 7, [3, 4], 8, 9, 10
	expected := []string{"line 1", "line 2", "line 5", "line 6", "line 7",
		"line 3", "line 4", "line 8", "line 9", "line 10"}
	if len(result) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(result))
	}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
	if firstDest != 6 {
		t.Errorf("firstDest = %d, want 6", firstDest)
	}
}

func TestExecMove_BasicMoveBackward(t *testing.T) {
	// Move lines 7-8 to before line 3 (destAfter=2, destLine=3).
	lines := testLines(10)
	result, _, err := execMove(lines, 7, 8, 2, 3, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: 1, 2, [7, 8], 3, 4, 5, 6, 9, 10
	expected := []string{"line 1", "line 2", "line 7", "line 8",
		"line 3", "line 4", "line 5", "line 6", "line 9", "line 10"}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestExecMove_LineCountInvariant(t *testing.T) {
	// For any valid single move (times=1), line count must not change.
	cases := []struct {
		name                          string
		n, src1, src2, dest, destLine int
	}{
		{"start to end", 10, 1, 2, 10, 10},
		{"end to start", 10, 9, 10, 0, 0},
		{"middle forward", 10, 3, 5, 8, 8},
		{"middle backward", 10, 6, 8, 2, 2},
		{"single line forward", 10, 1, 1, 9, 9},
		{"single line backward", 10, 10, 10, 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, _, err := execMove(testLines(tc.n), tc.src1, tc.src2, tc.dest, tc.destLine, 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(result) != tc.n {
				t.Errorf("line count changed: %d -> %d (want 0 delta)", tc.n, len(result))
			}
		})
	}
}

func TestExecMove_TimesN_LineCountDelta(t *testing.T) {
	// move -times N: delta == blockSize * (times - 1)
	cases := []struct {
		src1, src2, dest, destLine, times int
	}{
		{3, 4, 7, 7, 3}, // 2 lines * 2 extra pastes = +4
		{3, 4, 7, 7, 5}, // 2 * 4 = +8
		{1, 1, 9, 9, 3}, // 1 * 2 = +2
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("src%d-%d_times%d", tc.src1, tc.src2, tc.times), func(t *testing.T) {
			n := 10
			result, _, err := execMove(testLines(n), tc.src1, tc.src2, tc.dest, tc.destLine, tc.times)
			if err != nil {
				t.Fatal(err)
			}
			blockSize := tc.src2 - tc.src1 + 1
			want := n + blockSize*(tc.times-1)
			if len(result) != want {
				t.Errorf("expected %d lines, got %d", want, len(result))
			}
		})
	}
}

func TestExecMove_TimesN_CutOncePasteN(t *testing.T) {
	// Cut lines 3-4, paste 3 times after line 7.
	// Source should be gone from original position, 3 identical copies at dest.
	result, _, err := execMove(testLines(10), 3, 4, 7, 7, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 14 {
		t.Fatalf("expected 14 lines, got %d", len(result))
	}
	// Result: 1, 2, 5, 6, 7, [3,4], [3,4], [3,4], 8, 9, 10
	for c := 0; c < 3; c++ {
		offset := 5 + c*2
		if result[offset] != "line 3" || result[offset+1] != "line 4" {
			t.Errorf("copy %d at [%d,%d] = %q,%q; want 'line 3','line 4'",
				c+1, offset, offset+1, result[offset], result[offset+1])
		}
	}
	// Original position (indices 2-3) should NOT be "line 3" / "line 4"
	if result[2] == "line 3" {
		t.Error("source lines still present at original position")
	}
}

func TestExecMove_OverlapError_Interior(t *testing.T) {
	lines := testLines(20)
	for _, destLine := range []int{10, 12, 15, 20} { // srcStart=10, srcEnd=20
		_, _, err := execMove(lines, 10, 20, destLine, destLine, 1)
		if err == nil {
			t.Errorf("destLine=%d: expected overlap error, got nil", destLine)
		} else if !strings.Contains(err.Error(), "inside source range") {
			t.Errorf("destLine=%d: error = %q, want 'inside source range'", destLine, err.Error())
		}
	}
}

func TestExecMove_OverlapError_ExactMessage(t *testing.T) {
	lines := testLines(20)
	_, _, err := execMove(lines, 5, 10, 7, 7, 1)
	if err == nil {
		t.Fatal("expected error")
	}
	want := "destination line 7 is inside source range 5-10"
	if err.Error() != want {
		t.Errorf("error = %q\nwant  %q", err.Error(), want)
	}
}

func TestExecMove_OverlapError_BoundaryStart(t *testing.T) {
	_, _, err := execMove(testLines(10), 5, 8, 5, 5, 1)
	if err == nil {
		t.Error("destLine == srcStart should be rejected")
	}
}

func TestExecMove_OverlapError_BoundaryEnd(t *testing.T) {
	_, _, err := execMove(testLines(10), 5, 8, 8, 8, 1)
	if err == nil {
		t.Error("destLine == srcEnd should be rejected")
	}
}

func TestExecMove_BoundaryJustBefore(t *testing.T) {
	// destLine = srcStart-1 → not overlap, must succeed.
	_, _, err := execMove(testLines(10), 5, 8, 4, 4, 1)
	if err != nil {
		t.Errorf("dest just before src should succeed: %v", err)
	}
}

func TestExecMove_BoundaryJustAfter(t *testing.T) {
	// destLine = srcEnd+1 → not overlap, must succeed.
	result, _, err := execMove(testLines(10), 5, 8, 9, 9, 1)
	if err != nil {
		t.Errorf("dest just after src should succeed: %v", err)
	}
	if len(result) != 10 {
		t.Errorf("line count %d, want 10", len(result))
	}
}

func TestExecMove_TimesZeroError(t *testing.T) {
	_, _, err := execMove(testLines(10), 1, 2, 5, 5, 0)
	if err == nil {
		t.Error("expected error for times=0")
	}
}

func TestExecMove_TimesNegativeError(t *testing.T) {
	_, _, err := execMove(testLines(10), 1, 2, 5, 5, -3)
	if err == nil {
		t.Error("expected error for times=-3")
	}
}

func TestExecMove_InvalidRange(t *testing.T) {
	lines := testLines(10)
	cases := []struct{ s, e int }{{0, 5}, {1, 11}, {5, 3}}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d-%d", tc.s, tc.e), func(t *testing.T) {
			_, _, err := execMove(lines, tc.s, tc.e, 0, 0, 1)
			if err == nil {
				t.Error("expected error for invalid range")
			}
		})
	}
}

func TestExecMove_MoveToBeginning(t *testing.T) {
	result, firstDest, err := execMove(testLines(10), 8, 10, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(result))
	}
	if result[0] != "line 8" {
		t.Errorf("result[0] = %q, want 'line 8'", result[0])
	}
	if firstDest != 1 {
		t.Errorf("firstDest = %d, want 1", firstDest)
	}
}

func TestExecMove_MoveToEnd(t *testing.T) {
	result, _, err := execMove(testLines(10), 1, 2, 10, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 10 {
		t.Fatalf("expected 10 lines, got %d", len(result))
	}
	if result[8] != "line 1" || result[9] != "line 2" {
		t.Errorf("last two = %q, %q; want 'line 1','line 2'", result[8], result[9])
	}
}

func TestExecMove_SingleLine(t *testing.T) {
	result, _, err := execMove(testLines(5), 2, 2, 4, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: 1, 3, 4, 2, 5
	expected := []string{"line 1", "line 3", "line 4", "line 2", "line 5"}
	for i, want := range expected {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestExecMove_ContentPreserved(t *testing.T) {
	// All original lines must be present exactly once after a move.
	result, _, err := execMove(testLines(10), 4, 6, 9, 9, 1)
	if err != nil {
		t.Fatal(err)
	}
	counts := make(map[string]int)
	for _, l := range result {
		counts[l]++
	}
	for i := 1; i <= 10; i++ {
		want := fmt.Sprintf("line %d", i)
		if counts[want] != 1 {
			t.Errorf("%q appears %d times, want 1", want, counts[want])
		}
	}
}

func TestExecMove_AdjacentDestIsNoOp(t *testing.T) {
	// Move lines 5-7 to after line 4 (= just before themselves) → same order.
	lines := testLines(10)
	result, _, err := execMove(lines, 5, 7, 4, 4, 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range lines {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestExecMove_LargeFile(t *testing.T) {
	result, _, err := execMove(testLines(1000), 100, 200, 800, 800, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1000 {
		t.Errorf("expected 1000 lines, got %d", len(result))
	}
}

// ── execCopy ─────────────────────────────────────────────────────────────────

func TestExecCopy_Basic(t *testing.T) {
	result, firstDest, err := execCopy(testLines(10), 3, 4, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 12 {
		t.Fatalf("expected 12 lines, got %d", len(result))
	}
	// Source preserved at original position.
	if result[2] != "line 3" || result[3] != "line 4" {
		t.Errorf("source not preserved: %q, %q", result[2], result[3])
	}
	// Copy at destination (after line 7 → index 7,8).
	if result[7] != "line 3" || result[8] != "line 4" {
		t.Errorf("copy not at destination: %q, %q", result[7], result[8])
	}
	if firstDest != 8 {
		t.Errorf("firstDest = %d, want 8", firstDest)
	}
}

func TestExecCopy_LineCountDelta(t *testing.T) {
	// delta must equal (srcEnd-srcStart+1) * times exactly.
	cases := []struct{ n, s, e, dest, times int }{
		{10, 3, 5, 7, 1},
		{10, 3, 5, 7, 3},
		{10, 1, 1, 9, 5},
		{10, 2, 4, 0, 2},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("src%d-%d_x%d", tc.s, tc.e, tc.times), func(t *testing.T) {
			result, _, err := execCopy(testLines(tc.n), tc.s, tc.e, tc.dest, tc.times)
			if err != nil {
				t.Fatal(err)
			}
			wantDelta := (tc.e - tc.s + 1) * tc.times
			if len(result) != tc.n+wantDelta {
				t.Errorf("expected %d lines, got %d", tc.n+wantDelta, len(result))
			}
		})
	}
}

func TestExecCopy_OverlapIsAllowed(t *testing.T) {
	// Copy destination inside source range must NOT return an error.
	result, _, err := execCopy(testLines(20), 10, 20, 15, 1)
	if err != nil {
		t.Errorf("copy with overlap should be allowed: %v", err)
		return
	}
	// delta = 11 * 1 = 11
	if len(result) != 31 {
		t.Errorf("expected 31 lines, got %d", len(result))
	}
}

func TestExecCopy_SnapshotIntegrity(t *testing.T) {
	// Copy source 200-250 to before line 220 (overlap), times=3.
	// All 3 copies must be clones of the ORIGINAL block, not nested.
	n := 300
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("orig_%d", i+1)
	}
	// destAfter=219 (insert before original line 220)
	result, _, err := execCopy(lines, 200, 250, 219, 3)
	if err != nil {
		t.Fatal(err)
	}
	// 300 + 51*3 = 453
	if len(result) != 453 {
		t.Fatalf("expected 453 lines, got %d", len(result))
	}
	for c := 0; c < 3; c++ {
		offset := 219 + c*51 // 0-based
		for j := 0; j < 51; j++ {
			want := fmt.Sprintf("orig_%d", 200+j)
			if result[offset+j] != want {
				t.Errorf("copy %d offset %d: got %q, want %q", c+1, j, result[offset+j], want)
				break
			}
		}
	}
}

func TestExecCopy_SnapshotNotRecursive(t *testing.T) {
	// Copy lines 2-4 (BBB,CCC,DDD) to after line 2, times=2.
	// The 2nd copy must be [BBB,CCC,DDD] not [BBB, copy1..., CCC,DDD].
	lines := []string{"AAA", "BBB", "CCC", "DDD", "EEE"}
	result, _, err := execCopy(lines, 2, 4, 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 11 {
		t.Fatalf("expected 11 lines, got %d", len(result))
	}
	// copy 1: result[2..4]
	if result[2] != "BBB" || result[3] != "CCC" || result[4] != "DDD" {
		t.Errorf("copy 1 = %v, want BBB,CCC,DDD", result[2:5])
	}
	// copy 2: result[5..7] — must NOT include copy 1
	if result[5] != "BBB" || result[6] != "CCC" || result[7] != "DDD" {
		t.Errorf("copy 2 = %v, want BBB,CCC,DDD (not recursive)", result[5:8])
	}
}

func TestExecCopy_AllCopiesIdentical(t *testing.T) {
	lines := make([]string, 20)
	for i := range lines {
		lines[i] = fmt.Sprintf("unique_%04d", i)
	}
	result, _, err := execCopy(lines, 5, 9, 15, 10)
	if err != nil {
		t.Fatal(err)
	}
	original := lines[4:9] // 0-based snapshot of lines 5-9
	for c := 0; c < 10; c++ {
		offset := 15 + c*5
		for j, want := range original {
			if result[offset+j] != want {
				t.Errorf("copy %d elem %d: got %q, want %q", c+1, j, result[offset+j], want)
			}
		}
	}
}

func TestExecCopy_TimesZeroError(t *testing.T) {
	_, _, err := execCopy(testLines(10), 1, 2, 5, 0)
	if err == nil {
		t.Error("expected error for times=0")
	}
}

func TestExecCopy_InvalidRange(t *testing.T) {
	_, _, err := execCopy(testLines(10), 11, 12, 5, 1)
	if err == nil {
		t.Error("expected error for out-of-range source")
	}
}

func TestExecCopy_DestAtZero(t *testing.T) {
	result, firstDest, err := execCopy(testLines(5), 3, 4, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != "line 3" || result[1] != "line 4" {
		t.Errorf("copy not at top: %q, %q", result[0], result[1])
	}
	// Original preserved at indices 4,5 (shifted by 2)
	if result[4] != "line 3" || result[5] != "line 4" {
		t.Errorf("original not preserved: %q, %q", result[4], result[5])
	}
	if firstDest != 1 {
		t.Errorf("firstDest = %d, want 1", firstDest)
	}
}

func TestExecCopy_DestAtEnd(t *testing.T) {
	result, _, err := execCopy(testLines(5), 1, 2, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result[5] != "line 1" || result[6] != "line 2" {
		t.Errorf("copy not at end: %q, %q", result[5], result[6])
	}
}

func TestExecCopy_LargeFile(t *testing.T) {
	result, _, err := execCopy(testLines(1000), 200, 299, 500, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 1500 {
		t.Errorf("expected 1500 lines, got %d", len(result))
	}
}

// ── resolveSourceLines ────────────────────────────────────────────────────────

func TestResolveSource_ExplicitRange(t *testing.T) {
	s, e, err := resolveSourceLines(testLines(10), 3, 7, "", "", 1)
	if err != nil || s != 3 || e != 7 {
		t.Errorf("expected 3-7, got %d-%d err=%v", s, e, err)
	}
}

func TestResolveSource_SingleLineDefault(t *testing.T) {
	// -line 5 without -end → single-line range 5-5
	s, e, err := resolveSourceLines(testLines(10), 5, 0, "", "", 1)
	if err != nil || s != 5 || e != 5 {
		t.Errorf("expected 5-5, got %d-%d err=%v", s, e, err)
	}
}

func TestResolveSource_MatchWithExplicitEnd(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	s, e, err := resolveSourceLines(lines, 0, 3, "beta", "", 1)
	if err != nil || s != 2 || e != 3 {
		t.Errorf("expected 2-3, got %d-%d err=%v", s, e, err)
	}
}

func TestResolveSource_MatchWithEndmatch(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	s, e, err := resolveSourceLines(lines, 0, 0, "beta", "delta", 1)
	if err != nil || s != 2 || e != 4 {
		t.Errorf("expected 2-4, got %d-%d err=%v", s, e, err)
	}
}

func TestResolveSource_MatchNoEndBound(t *testing.T) {
	_, _, err := resolveSourceLines(testLines(10), 0, 0, "line 3", "", 1)
	if err == nil {
		t.Error("expected error: -match without end bound")
	}
}

func TestResolveSource_NoSourceSpecified(t *testing.T) {
	_, _, err := resolveSourceLines(testLines(10), 0, 0, "", "", 1)
	if err == nil {
		t.Error("expected error: no source specified")
	}
}

func TestResolveSource_MatchNotFound(t *testing.T) {
	_, _, err := resolveSourceLines(testLines(10), 0, 4, "NOTEXIST", "", 1)
	if err == nil {
		t.Error("expected error: match not found")
	}
}

func TestResolveSource_EndmatchBeforeStart(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma", "delta"}
	_, _, err := resolveSourceLines(lines, 0, 0, "gamma", "alpha", 1)
	if err == nil {
		t.Error("expected error: endmatch before start")
	}
}

func TestResolveSource_NthOccurrence(t *testing.T) {
	lines := []string{"foo", "bar", "foo", "baz", "foo"}
	s, e, err := resolveSourceLines(lines, 0, 4, "foo", "", 2)
	if err != nil || s != 3 || e != 4 {
		t.Errorf("expected 3-4, got %d-%d err=%v", s, e, err)
	}
}

// ── resolveDestLine ───────────────────────────────────────────────────────────

func TestResolveDest_After(t *testing.T) {
	da, dl, desc, err := resolveDestLine(testLines(10), 5, -1, "", "")
	if err != nil || da != 5 || dl != 5 {
		t.Errorf("after=5: da=%d dl=%d err=%v", da, dl, err)
	}
	if !strings.Contains(desc, "after line 5") {
		t.Errorf("desc = %q, want 'after line 5'", desc)
	}
}

func TestResolveDest_AfterZero(t *testing.T) {
	// -after 0 = insert at beginning of file
	da, dl, _, err := resolveDestLine(testLines(10), 0, -1, "", "")
	if err != nil || da != 0 || dl != 0 {
		t.Errorf("after=0: da=%d dl=%d err=%v", da, dl, err)
	}
}

func TestResolveDest_Before(t *testing.T) {
	da, dl, _, err := resolveDestLine(testLines(10), -1, 5, "", "")
	if err != nil || da != 4 || dl != 5 {
		t.Errorf("before=5: da=%d dl=%d err=%v", da, dl, err)
	}
}

func TestResolveDest_AfterMatch(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	da, dl, _, err := resolveDestLine(lines, -1, -1, "beta", "")
	if err != nil || da != 2 || dl != 2 {
		t.Errorf("aftermatch=beta: da=%d dl=%d err=%v", da, dl, err)
	}
}

func TestResolveDest_BeforeMatch(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma"}
	da, dl, _, err := resolveDestLine(lines, -1, -1, "", "beta")
	if err != nil || da != 1 || dl != 2 {
		t.Errorf("beforematch=beta: da=%d dl=%d err=%v", da, dl, err)
	}
}

func TestResolveDest_NoneSpecified(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(10), -1, -1, "", "")
	if err == nil {
		t.Error("expected error: no destination")
	}
}

func TestResolveDest_MultipleSpecified(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(10), 5, 7, "", "")
	if err == nil {
		t.Error("expected error: multiple destinations")
	}
}

func TestResolveDest_AfterOutOfRange(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(10), 11, -1, "", "")
	if err == nil {
		t.Error("expected error: -after out of range")
	}
}

func TestResolveDest_BeforeOutOfRange(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(10), -1, 0, "", "")
	if err == nil {
		t.Error("expected error: -before 0 out of range")
	}
}

func TestResolveDest_AfterMatchNotFound(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(5), -1, -1, "NOPE", "")
	if err == nil {
		t.Error("expected error: aftermatch not found")
	}
}

func TestResolveDest_BeforeMatchNotFound(t *testing.T) {
	_, _, _, err := resolveDestLine(testLines(5), -1, -1, "", "NOPE")
	if err == nil {
		t.Error("expected error: beforematch not found")
	}
}

// ── integration (full file round-trip) ───────────────────────────────────────

func TestMoveIntegration_FileRoundTrip(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	lines, _ := readLines(path)
	result, _, err := execMove(lines, 2, 4, 7, 7, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeLines(path, result); err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	// Expected: 1, 5, 6, 7, [2, 3, 4], 8, 9, 10
	expected := []string{"line 1", "line 5", "line 6", "line 7",
		"line 2", "line 3", "line 4", "line 8", "line 9", "line 10"}
	for i, want := range expected {
		if got[i] != want {
			t.Errorf("line %d = %q, want %q", i+1, got[i], want)
		}
	}
}

func TestCopyIntegration_FileRoundTrip(t *testing.T) {
	path := writeTestFile(t, testLines(5))
	lines, _ := readLines(path)
	result, _, err := execCopy(lines, 2, 3, 4, 2)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeLines(path, result); err != nil {
		t.Fatal(err)
	}
	got := readTestFile(t, path)
	// Original [1,2,3,4,5], copy 2-3 after line 4, times=2 → 9 lines
	if len(got) != 9 {
		t.Fatalf("expected 9 lines, got %d", len(got))
	}
	if got[4] != "line 2" || got[5] != "line 3" {
		t.Errorf("copy 1: %q, %q", got[4], got[5])
	}
	if got[6] != "line 2" || got[7] != "line 3" {
		t.Errorf("copy 2: %q, %q", got[6], got[7])
	}
}

func TestMoveIntegration_ContentMatchSource(t *testing.T) {
	lines := []string{
		"func alpha() {",
		"  body alpha",
		"}",
		"func beta() {",
		"  body beta",
		"}",
	}
	path := writeTestFile(t, lines)
	fileLines, _ := readLines(path)

	// Move the alpha block (lines 1-3) to after beta (line 6).
	s, e, err := resolveSourceLines(fileLines, 1, 3, "", "", 1)
	if err != nil {
		t.Fatal(err)
	}
	da, dl, _, err := resolveDestLine(fileLines, 6, -1, "", "")
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := execMove(fileLines, s, e, da, dl, 1)
	if err != nil {
		t.Fatal(err)
	}
	if result[0] != "func beta() {" {
		t.Errorf("result[0] = %q, want 'func beta() {'", result[0])
	}
	if result[3] != "func alpha() {" {
		t.Errorf("result[3] = %q, want 'func alpha() {'", result[3])
	}
}

// ── MCP handler tests ─────────────────────────────────────────────────────────

func TestMcpDoMove_Basic(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	res := mcpDoMove(path, 3, 5, "", "", 8, -1, "", "", "", "", "", "", 1, 1, time.Now())
	if res.IsError {
		t.Errorf("unexpected error: %s", res.Content[0].Text)
	}
	got := readTestFile(t, path)
	if len(got) != 10 {
		t.Errorf("expected 10 lines, got %d", len(got))
	}
}

func TestMcpDoMove_OverlapReturnsError(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	res := mcpDoMove(path, 3, 7, "", "", -1, -1, "", "", "", "", "", "", 1, 1, time.Now())
	// No destination → should return error
	if !res.IsError {
		t.Error("expected error for missing destination")
	}
}

func TestMcpDoMove_OverlapDestInsideSrc(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	// src=3-7, dest after line 5 (inside src) → overlap error
	res := mcpDoMove(path, 3, 7, "", "", 5, -1, "", "", "", "", "", "", 1, 1, time.Now())
	if !res.IsError {
		t.Error("expected error: destination inside source range")
	}
	if !strings.Contains(res.Content[0].Text, "inside source range") {
		t.Errorf("error text = %q, want 'inside source range'", res.Content[0].Text)
	}
}

func TestMcpDoCopy_Basic(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	res := mcpDoCopy(path, 3, 5, "", "", 8, -1, "", "", "", "", "", "", 1, 1, time.Now())
	if res.IsError {
		t.Errorf("unexpected error: %s", res.Content[0].Text)
	}
	got := readTestFile(t, path)
	// 10 + 3*1 = 13
	if len(got) != 13 {
		t.Errorf("expected 13 lines, got %d", len(got))
	}
}

func TestMcpDoCopy_TimesN(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	res := mcpDoCopy(path, 2, 4, "", "", 9, -1, "", "", "", "", "", "", 1, 3, time.Now())
	if res.IsError {
		t.Errorf("unexpected error: %s", res.Content[0].Text)
	}
	got := readTestFile(t, path)
	// 10 + 3*3 = 19
	if len(got) != 19 {
		t.Errorf("expected 19 lines, got %d", len(got))
	}
}

func TestMcpDoCopy_OverlapAllowed(t *testing.T) {
	path := writeTestFile(t, testLines(10))
	// src=3-7, dest after line 5 (inside src) → must succeed for copy
	res := mcpDoCopy(path, 3, 7, "", "", 5, -1, "", "", "", "", "", "", 1, 1, time.Now())
	if res.IsError {
		t.Errorf("copy with overlap should succeed: %s", res.Content[0].Text)
	}
}

func TestMcpDoMove_ContentMatchSource(t *testing.T) {
	lines := []string{"AAA", "BBB", "CCC", "DDD", "EEE"}
	path := writeTestFile(t, lines)
	// Move "BBB" line (via match, end=3) to after "EEE" (after line 5)
	res := mcpDoMove(path, 0, 0, "BBB", "", 5, -1, "", "", "", "", "", "", 1, 1, time.Now())
	if res.IsError {
		// endmatch missing → should fail with source error (acceptable)
		if !strings.Contains(res.Content[0].Text, "end bound") {
			t.Errorf("unexpected error: %s", res.Content[0].Text)
		}
	}
}

func TestMcpDoCopy_BeforeMatch(t *testing.T) {
	lines := []string{"func A", "body A", "func B", "body B"}
	path := writeTestFile(t, lines)
	// Copy lines 1-2 to before "func B"
	res := mcpDoCopy(path, 1, 2, "", "", -1, -1, "", "func B", "", "", "", "", 1, 1, time.Now())
	if res.IsError {
		t.Errorf("unexpected error: %s", res.Content[0].Text)
	}
	got := readTestFile(t, path)
	// [func A, body A, func A, body A, func B, body B]
	if len(got) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(got))
	}
	if got[2] != "func A" {
		t.Errorf("got[2] = %q, want 'func A'", got[2])
	}
	if got[4] != "func B" {
		t.Errorf("got[4] = %q, want 'func B'", got[4])
	}
}

// ── extra edge-case battery ───────────────────────────────────────────────────

func TestExecMove_WholFileIsSource(t *testing.T) {
	// Moving ALL lines to "beginning" is a no-op (except destLine=0 avoids overlap since 0 < 1).
	lines := testLines(5)
	result, _, err := execMove(lines, 1, 5, 0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range lines {
		if result[i] != want {
			t.Errorf("result[%d] = %q, want %q", i, result[i], want)
		}
	}
}

func TestExecCopy_WholFileIsSource(t *testing.T) {
	lines := testLines(3)
	// Copy all lines to end, times=2 → 9 total
	result, _, err := execCopy(lines, 1, 3, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 9 {
		t.Fatalf("expected 9 lines, got %d", len(result))
	}
}

func TestExecCopy_OverlapBeginningOfSource(t *testing.T) {
	// destAfter = srcStart-1 means inserting just before the source block.
	// For copy that's fine (no overlap check).
	result, _, err := execCopy(testLines(10), 5, 8, 4, 1) // destAfter=4, srcStart=5
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 14 {
		t.Fatalf("expected 14 lines, got %d", len(result))
	}
}

func TestExecMove_BeforeMatchDestDesc(t *testing.T) {
	lines := []string{"alpha", "beta", "gamma", "delta"}
	da, dl, desc, err := resolveDestLine(lines, -1, -1, "", "gamma")
	if err != nil {
		t.Fatal(err)
	}
	if da != 2 || dl != 3 {
		t.Errorf("beforematch=gamma: da=%d dl=%d, want da=2 dl=3", da, dl)
	}
	if !strings.Contains(desc, "before line 3") {
		t.Errorf("desc = %q, want 'before line 3'", desc)
	}
}

// ══════════════════════════════════════════════════════════════
// v1.2.1 — block resolution tests
// ══════════════════════════════════════════════════════════════

// ── getPythonBlocks ───────────────────────────────────────────

func TestGetPythonBlocks_Basic(t *testing.T) {
	src := []string{
		"class Foo:",
		"    def method(self):",
		"        pass",
		"",
		"class Bar:",
		"    pass",
		"",
		"def top_func():",
		"    return 1",
	}
	entries := getPythonBlocks(src)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].name != "class Foo" || entries[0].start != 1 {
		t.Errorf("entry 0 = %+v", entries[0])
	}
	if entries[1].name != "class Bar" || entries[1].start != 5 {
		t.Errorf("entry 1 = %+v", entries[1])
	}
	if entries[2].name != "def top_func" || entries[2].start != 8 {
		t.Errorf("entry 2 = %+v", entries[2])
	}
	// Foo ends before Bar starts
	if entries[0].end != 3 {
		t.Errorf("Foo end = %d, want 3 (last non-blank line before Bar)", entries[0].end)
	}
}

func TestGetPythonBlocks_IgnoresNested(t *testing.T) {
	src := []string{
		"class Outer:",
		"    class Inner:",
		"        pass",
		"    pass",
	}
	entries := getPythonBlocks(src)
	// Only Outer is top-level
	if len(entries) != 1 {
		t.Fatalf("expected 1 top-level entry, got %d", len(entries))
	}
	if entries[0].key != "Outer" {
		t.Errorf("expected Outer, got %q", entries[0].key)
	}
}

func TestGetPythonBlocks_EndOfFile(t *testing.T) {
	src := []string{
		"class A:",
		"    pass",
	}
	entries := getPythonBlocks(src)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].end != 2 {
		t.Errorf("end = %d, want 2", entries[0].end)
	}
}

func TestGetPythonBlocks_Empty(t *testing.T) {
	entries := getPythonBlocks([]string{})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty input")
	}
}

// ── getGoBlocks ───────────────────────────────────────────────

func TestGetGoBlocks_Basic(t *testing.T) {
	src := []string{
		"package main",
		"",
		"func Alpha() {",
		"    return",
		"}",
		"",
		"func Beta(x int) string {",
		"    return \"\"",
		"}",
	}
	entries := getGoBlocks(src)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(entries), entries)
	}
	if entries[0].key != "Alpha" || entries[0].start != 3 || entries[0].end != 5 {
		t.Errorf("Alpha = %+v", entries[0])
	}
	if entries[1].key != "Beta" || entries[1].start != 7 || entries[1].end != 9 {
		t.Errorf("Beta = %+v", entries[1])
	}
}

func TestGetGoBlocks_NestedBraces(t *testing.T) {
	src := []string{
		"func Complex() {",
		"    if true {",
		"        for i := 0; i < 10; i++ {",
		"        }",
		"    }",
		"}",
		"func Simple() {}",
	}
	entries := getGoBlocks(src)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].end != 6 {
		t.Errorf("Complex end = %d, want 6", entries[0].end)
	}
	if entries[1].end != 7 {
		t.Errorf("Simple end = %d, want 7", entries[1].end)
	}
}

func TestGetGoBlocks_TypeDecl(t *testing.T) {
	src := []string{
		"type Foo struct {",
		"    X int",
		"}",
	}
	entries := getGoBlocks(src)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].key != "Foo" || entries[0].start != 1 || entries[0].end != 3 {
		t.Errorf("Foo = %+v", entries[0])
	}
}

// ── getJSBlocks ───────────────────────────────────────────────

func TestGetJSBlocks_ClassAndFunction(t *testing.T) {
	src := []string{
		"class Renderer {",
		"    render() {}",
		"}",
		"",
		"function process(data) {",
		"    return data;",
		"}",
	}
	entries := getJSBlocks(src)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].key != "Renderer" || entries[0].end != 3 {
		t.Errorf("Renderer = %+v", entries[0])
	}
	if entries[1].key != "process" || entries[1].end != 7 {
		t.Errorf("process = %+v", entries[1])
	}
}

// ── resolveBlock ──────────────────────────────────────────────

func TestResolveBlock_ExactMatch(t *testing.T) {
	src := []string{
		"class Alpha:",
		"    pass",
		"",
		"class Beta:",
		"    pass",
	}
	s, e, err := resolveBlock(src, "python", "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	if s != 1 || e != 2 {
		t.Errorf("Alpha: start=%d end=%d, want 1-2", s, e)
	}
}

func TestResolveBlock_FullNameMatch(t *testing.T) {
	src := []string{
		"class Alpha:",
		"    pass",
	}
	s, e, err := resolveBlock(src, "python", "class Alpha")
	if err != nil || s != 1 || e != 2 {
		t.Errorf("full name match: s=%d e=%d err=%v", s, e, err)
	}
}

func TestResolveBlock_NoMatch(t *testing.T) {
	src := []string{"class Alpha:", "    pass"}
	_, _, err := resolveBlock(src, "python", "Nonexistent")
	if err == nil {
		t.Error("expected error: no match")
	}
	if !strings.Contains(err.Error(), "no match found") {
		t.Errorf("error = %q, want 'no match found'", err.Error())
	}
}

func TestResolveBlock_MultipleMatchError(t *testing.T) {
	src := []string{
		"class ProcessorA:",
		"    pass",
		"class ProcessorB:",
		"    pass",
	}
	_, _, err := resolveBlock(src, "python", "Processor")
	if err == nil {
		t.Error("expected error: multiple matches")
	}
	if !strings.Contains(err.Error(), "matched 2 blocks") {
		t.Errorf("error = %q, want 'matched 2 blocks'", err.Error())
	}
}

func TestResolveBlock_ErrorListsMatches(t *testing.T) {
	src := []string{
		"class FooA:",
		"    pass",
		"class FooB:",
		"    pass",
		"class FooC:",
		"    pass",
	}
	_, _, err := resolveBlock(src, "python", "Foo")
	if err == nil {
		t.Fatal("expected error")
	}
	// Error should list all three matches
	if !strings.Contains(err.Error(), "FooA") || !strings.Contains(err.Error(), "FooB") || !strings.Contains(err.Error(), "FooC") {
		t.Errorf("error missing match details: %v", err)
	}
}

func TestResolveBlock_NoLangError(t *testing.T) {
	_, _, err := resolveBlock(testLines(5), "", "something")
	if err == nil || !strings.Contains(err.Error(), "-lang") {
		t.Errorf("expected -lang error, got: %v", err)
	}
}

func TestResolveBlock_UnsupportedLang(t *testing.T) {
	_, _, err := resolveBlock(testLines(5), "cobol", "SECTION")
	if err == nil {
		t.Error("expected unsupported lang error")
	}
}

// ── integration: move with -block ─────────────────────────────

func TestBlockMove_PythonClassReorder(t *testing.T) {
	// Simulate: move class B before class A
	src := []string{
		"class A:",
		"    x = 1",
		"",
		"class B:",
		"    y = 2",
	}
	// resolveBlock for source "B" → lines 4-5
	srcStart, srcEnd, err := resolveBlock(src, "python", "B")
	if err != nil {
		t.Fatal(err)
	}
	if srcStart != 4 || srcEnd != 5 {
		t.Errorf("B: start=%d end=%d, want 4-5", srcStart, srcEnd)
	}
	// resolveBlock for dest "A" → start=1 (beforeblock)
	destStart, _, err := resolveBlock(src, "python", "A")
	if err != nil {
		t.Fatal(err)
	}
	// -beforeblock "A" → before line 1 → destAfter=0, destLine=1
	destAfter := destStart - 1
	destLine := destStart

	result, _, err := execMove(src, srcStart, srcEnd, destAfter, destLine, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Expected: [class B, y=2, class A, x=1, blank]
	if result[0] != "class B:" {
		t.Errorf("result[0] = %q, want 'class B:'", result[0])
	}
	if result[2] != "class A:" {
		t.Errorf("result[2] = %q, want 'class A:'", result[2])
	}
}

func TestBlockCopy_GoFuncDuplicate(t *testing.T) {
	src := []string{
		"func Alpha() {",
		"    return",
		"}",
		"",
		"func Beta() {",
		"    return",
		"}",
	}
	// Copy Alpha (lines 1-3) to after Beta (line 7)
	srcStart, srcEnd, err := resolveBlock(src, "go", "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	_, destEnd, err := resolveBlock(src, "go", "Beta")
	if err != nil {
		t.Fatal(err)
	}
	result, _, err := execCopy(src, srcStart, srcEnd, destEnd, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(result) != 10 {
		t.Errorf("expected 10 lines, got %d", len(result))
	}
	// Copy of Alpha should appear after Beta's closing brace
	if result[7] != "func Alpha() {" {
		t.Errorf("result[7] = %q, want 'func Alpha() {'", result[7])
	}
}

func TestBlockMove_OverlapRejected(t *testing.T) {
	// Attempt to move a block to a destination inside itself → overlap error
	src := []string{
		"class Alpha:",
		"    x = 1",
		"    y = 2",
	}
	srcStart, srcEnd, err := resolveBlock(src, "python", "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	// Try to move to inside the block itself
	_, _, err = execMove(src, srcStart, srcEnd, srcStart, srcStart, 1)
	if err == nil {
		t.Error("expected overlap error")
	}
}

func TestBlockCopy_OverlapAllowed(t *testing.T) {
	src := []string{
		"class Alpha:",
		"    x = 1",
		"",
		"class Beta:",
		"    y = 2",
	}
	// Copy Alpha into itself (overlap) → snapshot semantics, allowed
	srcStart, srcEnd, err := resolveBlock(src, "python", "Alpha")
	if err != nil {
		t.Fatal(err)
	}
	// dest = after line 1 (inside Alpha) — allowed for copy
	result, _, err := execCopy(src, srcStart, srcEnd, 1, 1)
	if err != nil {
		t.Errorf("copy with block overlap should be allowed: %v", err)
		return
	}
	// Line count: original 5 + 2 (Alpha block) = 7
	if len(result) != 7 {
		t.Errorf("expected 7 lines, got %d", len(result))
	}
}

// ── getTopLevelBlocks router ──────────────────────────────────

func TestGetTopLevelBlocks_Router(t *testing.T) {
	src := []string{"class A:", "    pass"}
	for _, lang := range []string{"python", "go", "javascript", "typescript", "rust", "java", "c#", "ruby", "php"} {
		_, err := getTopLevelBlocks(src, lang)
		if err != nil {
			t.Errorf("lang %q returned unexpected error: %v", lang, err)
		}
	}
}

func TestGetTopLevelBlocks_UnsupportedLang(t *testing.T) {
	_, err := getTopLevelBlocks(testLines(3), "brainfuck")
	if err == nil {
		t.Error("expected error for unsupported lang")
	}
}
