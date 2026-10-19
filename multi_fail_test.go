package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// Fault-injection tests for the multi -file replaceall verification. The
// writer seam is replaced with one that stores the file minus its last line,
// so the write itself succeeds but the content on disk is wrong. Such a file
// must be reported FAIL (not counted as a success) and the run must exit 3.

// mfInject makes writes to the given paths drop the last line. Other paths
// are written normally. The original writer is restored when the test ends.
func mfInject(t *testing.T, bad ...string) {
	t.Helper()
	old := multiWriteLines
	t.Cleanup(func() { multiWriteLines = old })
	multiWriteLines = func(path string, lines []string) error {
		for _, b := range bad {
			if strings.EqualFold(filepath.Clean(path), filepath.Clean(b)) {
				return writeLines(path, lines[:len(lines)-1])
			}
		}
		return writeLines(path, lines)
	}
}

func mfOpts(a, b string) multiMutOpts {
	return multiMutOpts{spec: a + "," + b, op: "replaceall", match: "x", replacement: "Y", nth: 1}
}

func TestMF_FailPath(t *testing.T) {
	t.Run("collect_marks_only_the_bad_file_FAIL", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\n")
		mfInject(t, a)
		results, ok, err := collectMultiMutate(mfOpts(a, b))
		if err != nil {
			t.Fatal(err)
		}
		if ok != 1 {
			t.Errorf("ok count %d, want 1", ok)
		}
		fails, passes := 0, 0
		for _, r := range results {
			switch r.status {
			case "FAIL":
				fails++
				if !strings.Contains(r.detail, "VERIFY: FAIL") {
					t.Errorf("FAIL detail lacks VERIFY: FAIL: %q", r.detail)
				}
			case "OK":
				passes++
				if !strings.Contains(r.detail, "VERIFY: PASS") {
					t.Errorf("OK detail lacks VERIFY: PASS: %q", r.detail)
				}
			}
		}
		if fails != 1 || passes != 1 {
			t.Errorf("FAIL=%d OK=%d, want 1 and 1: %+v", fails, passes, results)
		}
	})

	t.Run("exit3_when_one_file_fails", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\n")
		mfInject(t, a)
		if code := runMultiMutate(mfOpts(a, b)); code != 3 {
			t.Errorf("exit %d, want 3 (one file written but failed verification)", code)
		}
	})

	t.Run("exit3_when_every_file_fails", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\nc\n")
		mfInject(t, a, b)
		if code := runMultiMutate(mfOpts(a, b)); code != 3 {
			t.Errorf("exit %d, want 3 (every file failed verification)", code)
		}
	})

	t.Run("exit0_without_injection", func(t *testing.T) {
		a := wraFile(t, "a x\nb\n")
		b := wraFile(t, "x\n")
		if code := runMultiMutate(mfOpts(a, b)); code != 0 {
			t.Errorf("exit %d, want 0", code)
		}
	})
}
