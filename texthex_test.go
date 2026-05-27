package main

import (
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// thEncode is the test-side equivalent of fwencode: raw UTF-8 bytes → hex string.
// All tests that simulate "-texthex HEX" use this to produce the flag value,
// then call hex.DecodeString to simulate fedit's decode, then call the op
// function directly. This keeps tests fast (no subprocess) while covering the
// full round-trip.
func thEncode(s string) string {
	return hex.EncodeToString([]byte(s))
}

// thDecode simulates the decode step in main() after flag parsing.
func thDecode(h string) ([]byte, error) {
	return hex.DecodeString(h)
}

// thTempFile creates a temp file with given initial content for op tests.
func thTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "fedit_texthex_*.txt")
	if err != nil {
		t.Fatalf("thTempFile: %v", err)
	}
	f.Close()
	if content != "" {
		if err := writeLines(f.Name(), strings.Split(content, "\n")); err != nil {
			t.Fatalf("thTempFile write: %v", err)
		}
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

// ── bytesToLines unit tests ───────────────────────────────────────────────────

func TestBytesToLines_Empty(t *testing.T) {
	if got := bytesToLines(nil); got != nil {
		t.Errorf("nil input: want nil, got %v", got)
	}
	if got := bytesToLines([]byte{}); got != nil {
		t.Errorf("empty slice: want nil, got %v", got)
	}
}

func TestBytesToLines_SingleLineNoTrailingNewline(t *testing.T) {
	got := bytesToLines([]byte("hello"))
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %v, want [hello]", got)
	}
}

func TestBytesToLines_SingleLineTrailingNewline(t *testing.T) {
	// A single line ending with \n should produce exactly one element, not two.
	got := bytesToLines([]byte("hello\n"))
	if len(got) != 1 || got[0] != "hello" {
		t.Errorf("got %v, want [hello]", got)
	}
}

func TestBytesToLines_MultipleLines(t *testing.T) {
	got := bytesToLines([]byte("line1\nline2\nline3\n"))
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(got), got)
	}
	if got[0] != "line1" || got[1] != "line2" || got[2] != "line3" {
		t.Errorf("got %v", got)
	}
}

func TestBytesToLines_MultipleTrailingNewlines(t *testing.T) {
	// Two trailing newlines: content ends with \n\n → last element is ""
	// bytesToLines should strip ONLY one trailing empty element (one trailing newline).
	// The second newline represents a genuinely empty last line.
	got := bytesToLines([]byte("line1\n\n"))
	// Expect: ["line1", ""] — the internal empty line is real content
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %v", len(got), got)
	}
	if got[0] != "line1" || got[1] != "" {
		t.Errorf("got %v, want [line1 ]", got)
	}
}

func TestBytesToLines_TabsPreserved(t *testing.T) {
	got := bytesToLines([]byte("\thello\n\tworld\n"))
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0] != "\thello" {
		t.Errorf("tab stripped in line 0: %q", got[0])
	}
	if got[1] != "\tworld" {
		t.Errorf("tab stripped in line 1: %q", got[1])
	}
}

func TestBytesToLines_DoubleQuotesPreserved(t *testing.T) {
	// The key use case: Go source with string literals.
	got := bytesToLines([]byte(`fmt.Println("hello")`+"\n"))
	if len(got) != 1 || got[0] != `fmt.Println("hello")` {
		t.Errorf("got %v", got)
	}
}

func TestBytesToLines_BackslashesNotExpanded(t *testing.T) {
	// Critical: bytesToLines must not expand \n, \t, \\ sequences.
	// If the content has the literal two-character sequence backslash-n,
	// it must stay as backslash-n, not become a newline.
	input := []byte("path=C:\\\\Users\\\\admin\n")
	got := bytesToLines(input)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1: %v", len(got), got)
	}
	if !strings.Contains(got[0], "\\\\") {
		t.Errorf("backslashes were altered: %q", got[0])
	}
}

// ── resolveTextFull unit tests ────────────────────────────────────────────────

func TestResolveTextFull_NilBytesUsesText(t *testing.T) {
	got := resolveTextFull("hello\\nworld", "", nil)
	// expandText expands \n → real newline
	if len(got) != 2 || got[0] != "hello" || got[1] != "world" {
		t.Errorf("expected [hello world], got %v", got)
	}
}

func TestResolveTextFull_BytesOverrideText(t *testing.T) {
	// Even when -text has content, resolvedBytes takes priority.
	b := []byte("from_hex\n")
	got := resolveTextFull("from_text", "", b)
	if len(got) != 1 || got[0] != "from_hex" {
		t.Errorf("got %v, want [from_hex]", got)
	}
}

func TestResolveTextFull_BytesOverrideTextFile(t *testing.T) {
	// resolvedBytes also overrides -textfile.
	f := thTempFile(t, "from_file")
	b := []byte("from_hex\n")
	got := resolveTextFull("", f, b)
	if len(got) != 1 || got[0] != "from_hex" {
		t.Errorf("got %v, want [from_hex]", got)
	}
}

func TestResolveTextFull_NilBytesUsesTextFile(t *testing.T) {
	f := thTempFile(t, "from_file")
	got := resolveTextFull("ignored_text", f, nil)
	if len(got) == 0 || !strings.Contains(strings.Join(got, ""), "from_file") {
		t.Errorf("expected textfile content, got %v", got)
	}
}

func TestResolveTextFull_DoubleQuotesInHexContent(t *testing.T) {
	// The primary motivation: Go source with double quotes can be hex-encoded
	// and decoded without any shell-escaping issues.
	src := `	ClientID: "my-client-id",`
	b, _ := thDecode(thEncode(src + "\n"))
	got := resolveTextFull("", "", b)
	if len(got) != 1 || got[0] != src {
		t.Errorf("got %q, want %q", got, src)
	}
}

func TestResolveTextFull_BackslashNotExpanded(t *testing.T) {
	// Content with C:\Users\admin — backslashes must survive intact.
	// If expandText were called, \U, \a, etc. might be processed (they aren't
	// currently, but \\ → \ would be). Ensure bytesToLines path is safe.
	src := `path = "C:\\Users\\admin\\file.txt"`
	b, _ := thDecode(thEncode(src + "\n"))
	got := resolveTextFull("", "", b)
	if len(got) != 1 || got[0] != src {
		t.Errorf("got %q, want %q", got[0], src)
	}
}

func TestResolveTextFull_MultilineGoSource(t *testing.T) {
	// A realistic multi-line Go snippet with tabs, quotes, and braces.
	src := "// GoogleDrive provider configuration.\nvar GoogleDrive = Provider{\n\tName: \"googledrive\",\n}\n"
	b, _ := thDecode(thEncode(src))
	got := resolveTextFull("", "", b)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %v", len(got), got)
	}
	if got[0] != "// GoogleDrive provider configuration." {
		t.Errorf("line 0: %q", got[0])
	}
	if got[1] != "var GoogleDrive = Provider{" {
		t.Errorf("line 1: %q", got[1])
	}
	if got[2] != "\tName: \"googledrive\"," {
		t.Errorf("line 2 (tab+quote): %q", got[2])
	}
	if got[3] != "}" {
		t.Errorf("line 3: %q", got[3])
	}
}

// ── hex encode / decode round-trip tests ─────────────────────────────────────

func TestThEncodeDecode_ASCII(t *testing.T) {
	cases := []string{
		"hello world",
		"",
		"line1\nline2",
		"\thello\tworld",
	}
	for _, c := range cases {
		encoded := thEncode(c)
		decoded, err := thDecode(encoded)
		if err != nil {
			t.Errorf("decode error for %q: %v", c, err)
			continue
		}
		if string(decoded) != c {
			t.Errorf("round-trip failed for %q: got %q", c, decoded)
		}
	}
}

func TestThDecode_InvalidHex(t *testing.T) {
	cases := []string{
		"zz",       // non-hex chars
		"0",        // odd length
		"0g",       // invalid char
		"hello",    // not hex
	}
	for _, c := range cases {
		if _, err := thDecode(c); err == nil {
			t.Errorf("expected error for invalid hex %q, got nil", c)
		}
	}
}

func TestThDecode_ValidEdgeCases(t *testing.T) {
	// Empty hex string decodes to empty bytes (not an error).
	b, err := thDecode("")
	if err != nil {
		t.Errorf("empty string should decode to empty bytes, got error: %v", err)
	}
	if len(b) != 0 {
		t.Errorf("empty string should decode to 0 bytes, got %d", len(b))
	}
}

// ── write op integration tests ────────────────────────────────────────────────

func TestTexthex_Write_Basic(t *testing.T) {
	path := thTempFile(t, "")
	src := "line one\nline two\nline three\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(got), got)
	}
	if got[0] != "line one" || got[1] != "line two" || got[2] != "line three" {
		t.Errorf("got %v", got)
	}
}

func TestTexthex_Write_WithDoubleQuotes(t *testing.T) {
	// Ensures double quotes in source survive the hex round-trip.
	path := thTempFile(t, "")
	src := `	Name: "googledrive",` + "\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 1 || got[0] != `	Name: "googledrive",` {
		t.Errorf("double quotes corrupted: %q", got)
	}
}

func TestTexthex_Write_BackslashNotExpanded(t *testing.T) {
	// C:\Users\admin — the two backslashes must NOT be collapsed to one.
	path := thTempFile(t, "")
	src := `path = "C:\\Users\\admin"` + "\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	want := `path = "C:\\Users\\admin"`
	if got[0] != want {
		t.Errorf("backslash expanded:\n  got:  %q\n  want: %q", got[0], want)
	}
}

func TestTexthex_Write_TabsPreserved(t *testing.T) {
	path := thTempFile(t, "")
	src := "\tvar x = 1\n\tvar y = 2\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if !strings.HasPrefix(got[0], "\t") || !strings.HasPrefix(got[1], "\t") {
		t.Errorf("tabs stripped: %v", got)
	}
}

func TestTexthex_Write_MultilineGoSource(t *testing.T) {
	// Realistic snippet: the Google Drive provider var we actually need to insert.
	path := thTempFile(t, "")
	src := "// GoogleDrive provider.\nvar GoogleDrive = Provider{\n\tName:    \"googledrive\",\n\tUsePKCE: true,\n}\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5: %v", len(got), got)
	}
	if got[2] != "\tName:    \"googledrive\"," {
		t.Errorf("line 2 corrupt: %q", got[2])
	}
	if got[3] != "\tUsePKCE: true," {
		t.Errorf("line 3 corrupt: %q", got[3])
	}
}

// ── insert op integration tests ───────────────────────────────────────────────

func TestTexthex_Insert_AfterLine(t *testing.T) {
	path := thTempFile(t, "line1\nline3\nline4")
	src := "line2\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsert(lines, path, 1, newText)
	got, _ := readLines(path)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %v", len(got), got)
	}
	if got[1] != "line2" {
		t.Errorf("inserted line wrong: %q", got[1])
	}
}

func TestTexthex_Insert_WithDoubleQuotes(t *testing.T) {
	path := thTempFile(t, "before\nafter")
	src := `	ClientID: "test-client",` + "\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsert(lines, path, 1, newText)
	got, _ := readLines(path)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(got), got)
	}
	want := `	ClientID: "test-client",`
	if got[1] != want {
		t.Errorf("got %q, want %q", got[1], want)
	}
}

// ── insertafter / insertbefore integration tests ──────────────────────────────

func TestTexthex_InsertAfter_WithDoubleQuotes(t *testing.T) {
	path := thTempFile(t, "var Dropbox = Provider{\n\tName: \"dropbox\",\n}\n\nfunc ProviderByName")
	// Insert GoogleDrive var before ProviderByName
	src := "var GoogleDrive = Provider{\n\tName: \"googledrive\",\n}\n\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsertMatch(lines, path, "}", 1, newText, false) // insertafter first "}"
	got, _ := readLines(path)
	// Original had 5 lines; inserted 4 → 9 total
	if len(got) < 7 {
		t.Fatalf("too few lines after insert: %d: %v", len(got), got)
	}
	// Verify double quotes survived
	found := false
	for _, l := range got {
		if l == `	Name: "googledrive",` {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("inserted line with double quotes not found in: %v", got)
	}
}

func TestTexthex_InsertBefore_WithDoubleQuotes(t *testing.T) {
	path := thTempFile(t, "var Dropbox = Provider{\n\tName: \"dropbox\",\n}\n\nfunc ProviderByName() {}")
	src := "var GoogleDrive = Provider{\n\tName: \"googledrive\",\n}\n\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsertMatch(lines, path, "func ProviderByName", 1, newText, true) // insertbefore
	got, _ := readLines(path)
	// Verify GoogleDrive comes before ProviderByName
	gdLineIdx, pbnLineIdx := -1, -1
	for i, l := range got {
		if strings.Contains(l, "googledrive") {
			gdLineIdx = i
		}
		if strings.Contains(l, "func ProviderByName") {
			pbnLineIdx = i
		}
	}
	if gdLineIdx == -1 {
		t.Fatal("GoogleDrive line not found")
	}
	if pbnLineIdx == -1 {
		t.Fatal("ProviderByName line not found")
	}
	if gdLineIdx >= pbnLineIdx {
		t.Errorf("GoogleDrive (line %d) should be before ProviderByName (line %d)", gdLineIdx, pbnLineIdx)
	}
}

func TestTexthex_InsertBefore_CaseStatement(t *testing.T) {
	// The specific case we need: inserting a switch case with double quotes
	// before the closing brace of ProviderByName.
	path := thTempFile(t, "func ProviderByName(name string) (Provider, bool) {\n\tswitch name {\n\tcase \"dropbox\":\n\t\treturn Dropbox, true\n\t}\n\treturn Provider{}, false\n}")
	src := "\tcase \"googledrive\":\n\t\treturn GoogleDrive, true\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsertMatch(lines, path, "return Dropbox, true", 1, newText, false)
	got, _ := readLines(path)
	// Verify both cases exist and googledrive follows dropbox
	dropboxIdx, gdriveIdx := -1, -1
	for i, l := range got {
		if strings.Contains(l, `"dropbox"`) {
			dropboxIdx = i
		}
		if strings.Contains(l, `"googledrive"`) {
			gdriveIdx = i
		}
	}
	if dropboxIdx == -1 {
		t.Fatal("dropbox case not found")
	}
	if gdriveIdx == -1 {
		t.Fatal("googledrive case not found")
	}
	if gdriveIdx <= dropboxIdx {
		t.Errorf("googledrive case (line %d) should follow dropbox case (line %d)", gdriveIdx, dropboxIdx)
	}
}

// ── replace op integration tests ─────────────────────────────────────────────

func TestTexthex_Replace_WithDoubleQuotes(t *testing.T) {
	path := thTempFile(t, "line1\nOLD_LINE\nline3")
	src := `NEW: "quoted content"` + "\n"
	b, _ := thDecode(thEncode(src))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doReplace(lines, path, 2, 2, newText)
	got, _ := readLines(path)
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3: %v", len(got), got)
	}
	want := `NEW: "quoted content"`
	if got[1] != want {
		t.Errorf("got %q, want %q", got[1], want)
	}
}

// ── server registration snippet — the actual production use case ──────────────

func TestTexthex_ServerRegistrationSnippet(t *testing.T) {
	// The exact content we need to insert into server.go.
	// This test verifies the full path works for the most complex real-world case.
	path := thTempFile(t, "package main\n\nfunc init() {\n\ts.registry.Register(protocol.NewDropbox(s.oauthStore, oauth.Config{ClientID: \"bihqc9zghmhhgcr\", Provider: oauth.Dropbox}, log.Logger))\n}")
	insert := "\ts.registry.Register(protocol.NewGoogleDrive(s.oauthStore, oauth.Config{ClientID: \"GDRIVE_CLIENT_ID_HERE\", Provider: oauth.GoogleDrive}, log.Logger))\n"
	b, _ := thDecode(thEncode(insert))
	newText := resolveTextFull("", "", b)
	lines, _ := readLines(path)
	doInsertMatch(lines, path, "NewDropbox", 1, newText, false)
	got, _ := readLines(path)
	// Both Dropbox and GoogleDrive registrations must exist
	hasDropbox, hasGDrive := false, false
	for _, l := range got {
		if strings.Contains(l, "NewDropbox") {
			hasDropbox = true
		}
		if strings.Contains(l, "NewGoogleDrive") {
			hasGDrive = true
		}
	}
	if !hasDropbox {
		t.Error("Dropbox registration not found after insert")
	}
	if !hasGDrive {
		t.Error("GoogleDrive registration not found after insert")
	}
	// Verify the client ID string with double quotes is intact
	for _, l := range got {
		if strings.Contains(l, "NewGoogleDrive") {
			if !strings.Contains(l, `"GDRIVE_CLIENT_ID_HERE"`) {
				t.Errorf("double quotes around client ID corrupted: %q", l)
			}
		}
	}
}

// ── providers.go snippet — the other production use case ─────────────────────

func TestTexthex_ProvidersGoSnippet(t *testing.T) {
	// Verify the GoogleDrive provider var with complex content
	// round-trips correctly through hex encode/decode/bytesToLines/writeLines.
	path := thTempFile(t, "")
	src := "// GoogleDrive provider configuration.\nvar GoogleDrive = Provider{\n\tName:        \"googledrive\",\n\tDisplayName: \"Google Drive\",\n\tAuthURL:     \"https://accounts.google.com/o/oauth2/v2/auth\",\n\tTokenURL:    \"https://oauth2.googleapis.com/token\",\n\tUsePKCE:     true,\n\tExtraAuthParams: map[string]string{\n\t\t\"access_type\": \"offline\",\n\t\t\"prompt\":      \"consent\",\n\t},\n}\n"
	b, _ := thDecode(thEncode(src))
	content := bytesToLines(b)
	if err := writeLines(path, content); err != nil {
		t.Fatalf("writeLines: %v", err)
	}
	got, _ := readLines(path)
	if len(got) != 12 {
		t.Fatalf("len = %d, want 12: %v", len(got), got)
	}
	// Spot-check lines with double quotes
	checks := map[int]string{
		2:  "\tName:        \"googledrive\",",
		3:  "\tDisplayName: \"Google Drive\",",
		8:  "\t\t\"access_type\": \"offline\",",
		9:  "\t\t\"prompt\":      \"consent\",",
	}
	for lineIdx, want := range checks {
		if got[lineIdx] != want {
			t.Errorf("line %d: got %q, want %q", lineIdx, got[lineIdx], want)
		}
	}
}
