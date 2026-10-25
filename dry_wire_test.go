package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// drbNL folds CRLF to LF so assertions do not depend on the platform.
func drbNL(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

// drbHas reports whether the diff has a content line that starts with sign
// (+ or -) and contains sub. The --- and +++ header lines are skipped.
func drbHas(diff, sign, sub string) bool {
	for _, l := range strings.Split(drbNL(diff), "\n") {
		if strings.HasPrefix(l, "---") || strings.HasPrefix(l, "+++") {
			continue
		}
		if strings.HasPrefix(l, sign) && strings.Contains(l, sub) {
			return true
		}
	}
	return false
}

func drbRead(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestDRW_ReplaceAllVerifiedSkipsVerify(t *testing.T) {
	buf := drxSetup(t, true)
	p := drxFile(t, "alpha\nbeta\n")
	lines, err := readLines(p)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := runReplaceAllVerified(lines, p, "alpha", "gamma", false, &out, writeLines); code != 0 {
		t.Fatalf("exit %d, output %q", code, out.String())
	}
	if b := drbRead(t, p); string(b) != "alpha\nbeta\n" {
		t.Fatalf("dry run changed the file: %q", b)
	}
	if !strings.Contains(out.String(), "VERIFY: SKIP (dry-run)") || strings.Contains(out.String(), "VERIFY: FAIL") {
		t.Fatalf("verify must be skipped under dry-run, got %q", out.String())
	}
	if !drbHas(buf.String(), "-", "alpha") || !drbHas(buf.String(), "+", "gamma") {
		t.Fatalf("diff missing:\n%s", buf.String())
	}
}

func TestDRW_RegexVerifiedSkipsVerify(t *testing.T) {
	buf := drxSetup(t, true)
	p := drxFile(t, "alpha\nbeta\n")
	lines, err := readLines(p)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if code := runReplaceAllRegexVerified(lines, p, "alp.a", "gamma", false, &out, writeLines); code != 0 {
		t.Fatalf("exit %d, output %q", code, out.String())
	}
	if b := drbRead(t, p); string(b) != "alpha\nbeta\n" {
		t.Fatalf("dry run changed the file: %q", b)
	}
	if !strings.Contains(out.String(), "VERIFY: SKIP (dry-run)") || strings.Contains(out.String(), "VERIFY: FAIL") {
		t.Fatalf("verify must be skipped under dry-run, got %q", out.String())
	}
	if !drbHas(buf.String(), "+", "gamma") {
		t.Fatalf("diff missing:\n%s", buf.String())
	}
}

func drbBinReplaceAll(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	before := drbRead(t, f)
	so, se, code := orxRun(t, bin, "-dry", "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if !bytes.Equal(before, drbRead(t, f)) {
		t.Fatal("-dry changed the file")
	}
	if !strings.Contains(drbNL(so), "--- "+f+"\n+++ "+f+"\n") {
		t.Errorf("diff header missing:\n%s", so)
	}
	if !drbHas(so, "-", "alpha") || !drbHas(so, "+", "gamma") {
		t.Errorf("diff lines missing:\n%s", so)
	}
	if !strings.Contains(se, "VERIFY: SKIP (dry-run)") || strings.Contains(se, "VERIFY: FAIL") {
		t.Errorf("verify must be skipped, stderr %q", se)
	}
	if !strings.Contains(se, "applied=false") {
		t.Errorf("the report must say applied=false, stderr %q", se)
	}
}

func drbBinInsertAfter(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	before := drbRead(t, f)
	so, se, code := orxRun(t, bin, "-dry", "-file", f, "-op", "insertafter", "-match", "alpha", "-text", "zzzline")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if !bytes.Equal(before, drbRead(t, f)) {
		t.Fatal("-dry changed the file")
	}
	if !drbHas(so, "+", "zzzline") {
		t.Errorf("diff missing the inserted line:\n%s", so)
	}
	if !strings.Contains(se, "applied=false") {
		t.Errorf("the report must say applied=false, stderr %q", se)
	}
}

func drbBinWriteNewFile(t *testing.T, bin string) {
	dir, _ := lwxSetup(t)
	nf := filepath.Join(dir, "newfile.txt")
	so, se, code := orxRun(t, bin, "-dry", "-file", nf, "-op", "write", "-text", "hello")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if _, err := os.Stat(nf); err == nil {
		t.Fatal("-dry created the file")
	}
	if !strings.Contains(drbNL(so), "--- /dev/null\n+++ "+nf+"\n") || !drbHas(so, "+", "hello") {
		t.Errorf("new-file diff missing:\n%s", so)
	}
}

func drbBinNoMatchExit(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	before := drbRead(t, f)
	args := []string{"-file", f, "-op", "replaceall", "-match", "nosuchtext", "-text", "x"}
	_, _, dryCode := orxRun(t, bin, append([]string{"-dry"}, args...)...)
	_, _, realCode := orxRun(t, bin, args...)
	if dryCode != 1 || realCode != 1 {
		t.Fatalf("a run with no match must exit 1 with and without -dry, got dry %d, real %d", dryCode, realCode)
	}
	if !bytes.Equal(before, drbRead(t, f)) {
		t.Fatal("the file changed")
	}
}

func drbBinOffNoNote(t *testing.T, bin string) {
	_, f := lwxSetup(t)
	_, se, code := orxRun(t, bin, "-file", f, "-op", "replaceall", "-match", "alpha", "-text", "gamma")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, se)
	}
	if strings.Contains(se, "applied=false") {
		t.Errorf("a normal run must not say applied=false, stderr %q", se)
	}
	if !strings.Contains(string(drbRead(t, f)), "gamma") {
		t.Error("a normal run must still write")
	}
}
