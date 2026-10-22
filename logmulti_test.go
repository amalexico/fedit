package main

import (
	"strings"
	"testing"
)

// lmxPath returns the path of the first result with the given status, as
// collectMultiMutate reports it (the spec may be expanded or cleaned).
func lmxPath(t *testing.T, rs []batchResult, status string) string {
	t.Helper()
	for _, r := range rs {
		if r.status == status {
			return r.path
		}
	}
	t.Fatalf("no %s result in %v", status, rs)
	return ""
}

func TestLM_CollectMultiNotes(t *testing.T) {
	t.Run("all_pass", func(t *testing.T) {
		lnxStart(t)
		a, b := wraFile(t, "a x\nb\n"), wraFile(t, "x\n")
		rs, ok, err := collectMultiMutate(mfOpts(a, b))
		if err != nil || ok != 2 || len(rs) != 2 {
			t.Fatalf("ok = %d, results = %d, err = %v", ok, len(rs), err)
		}
		lnxWant(t, "files_total", "2")
		lnxWant(t, "files_ok", "2")
		lnxWant(t, "files_skip", "0")
		lnxWant(t, "files_fail", "0")
		lnxWant(t, "verify", "PASS")
		lnxWant(t, "results", "OK "+rs[0].path+"; OK "+rs[1].path)
	})
	t.Run("one_file_fails_verification", func(t *testing.T) {
		lnxStart(t)
		a, b := wraFile(t, "a x\nb\n"), wraFile(t, "x\n")
		mfInject(t, a)
		rs, _, err := collectMultiMutate(mfOpts(a, b))
		if err != nil {
			t.Fatal(err)
		}
		lnxWant(t, "files_ok", "1")
		lnxWant(t, "files_fail", "1")
		lnxWant(t, "verify", "FAIL")
		if v, _ := logNoteValue("results"); !strings.Contains(v, "FAIL "+lmxPath(t, rs, "FAIL")) {
			t.Errorf("results lacks the failed file: %q", v)
		}
	})
	t.Run("a_file_without_the_text_is_skipped", func(t *testing.T) {
		lnxStart(t)
		a, b := wraFile(t, "q\n"), wraFile(t, "x\n")
		rs, ok, err := collectMultiMutate(mfOpts(a, b))
		if err != nil || ok != 1 {
			t.Fatalf("ok = %d, err = %v", ok, err)
		}
		lnxWant(t, "files_skip", "1")
		lnxWant(t, "files_ok", "1")
		lnxWant(t, "verify", "PASS")
		if v, _ := logNoteValue("results"); !strings.Contains(v, "SKIP "+lmxPath(t, rs, "SKIP")) {
			t.Errorf("results lacks the skipped file: %q", v)
		}
	})
	t.Run("an_op_without_verification_notes_none", func(t *testing.T) {
		lnxStart(t)
		a, b := wraFile(t, "x\n"), wraFile(t, "x\n")
		o := multiMutOpts{spec: a + "," + b, op: "insertafter", match: "x", newLines: []string{"new"}, nth: 1}
		if _, ok, err := collectMultiMutate(o); err != nil || ok != 2 {
			t.Fatalf("ok = %d, err = %v", ok, err)
		}
		lnxWant(t, "files_ok", "2")
		lnxWant(t, "verify", "none")
	})
	t.Run("no_session_notes_nothing", func(t *testing.T) {
		logRun, logNotes = nil, nil
		a, b := wraFile(t, "x\n"), wraFile(t, "x\n")
		if _, _, err := collectMultiMutate(mfOpts(a, b)); err != nil {
			t.Fatal(err)
		}
		if len(logNotes) != 0 {
			t.Errorf("notes without a session: %v", logNotes)
		}
	})
}
