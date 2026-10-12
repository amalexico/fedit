package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mcpMDWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mcpMDRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(b), "\r\n", "\n")
}

const mcpMDSrc = "package main\n\nfunc Helper() int {\n\treturn 42\n}\n\nfunc Keep() {}\n"
const mcpMDDest = "package main\n\nfunc Init() {}\n\nfunc Target() int {\n\treturn 1\n}\n"

func mcpMDFixture(t *testing.T) (dir string, p func(string) string) {
	t.Helper()
	dir = t.TempDir()
	p = func(n string) string { return filepath.Join(dir, n) }
	mcpMDWrite(t, p("src.go"), mcpMDSrc)
	mcpMDWrite(t, p("d1.go"), mcpMDDest)
	mcpMDWrite(t, p("d2.go"), mcpMDDest)
	return dir, p
}

func TestMCPMultiDest_CopyAfterBlock(t *testing.T) {
	_, p := mcpMDFixture(t)
	res := mcpExecTool("fedit_copy", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go",
		"dest": p("d1.go") + "," + p("d2.go"), "afterblock": "Init",
	})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "2/2 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	for _, n := range []string{"d1.go", "d2.go"} {
		if !strings.Contains(mcpMDRead(t, p(n)), "func Helper() int {") {
			t.Errorf("%s did not receive Helper:\n%s", n, mcpMDRead(t, p(n)))
		}
	}
	if mcpMDRead(t, p("src.go")) != mcpMDSrc {
		t.Error("copy must leave the source untouched")
	}
}

func TestMCPMultiDest_MoveRemovesSourceAfterSuccess(t *testing.T) {
	_, p := mcpMDFixture(t)
	res := mcpExecTool("fedit_move", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go",
		"dest": p("d1.go"), "afterblock": "Init",
	})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "1/1 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	if !strings.Contains(mcpMDRead(t, p("d1.go")), "func Helper() int {") {
		t.Error("d1.go did not receive Helper")
	}
	if strings.Contains(mcpMDRead(t, p("src.go")), "func Helper") {
		t.Error("move must remove Helper from the source")
	}
	if !strings.Contains(mcpMDRead(t, p("src.go")), "func Keep()") {
		t.Error("move must leave the rest of the source")
	}
}

func TestMCPMultiDest_MoveKeepsSourceWhenEveryDestinationFails(t *testing.T) {
	_, p := mcpMDFixture(t)
	res := mcpExecTool("fedit_move", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go",
		"dest": p("missing1.go") + "," + p("missing2.go"), "afterblock": "Init",
	})
	text := mcpMultiText(res)
	if !res.IsError || !strings.Contains(text, "0/2 file(s) succeeded") || !strings.Contains(text, "left unmodified") {
		t.Fatalf("want error, 0/2 and the left-unmodified note, got:\n%s", text)
	}
	if mcpMDRead(t, p("src.go")) != mcpMDSrc {
		t.Error("the source must stay intact when every destination fails")
	}
}

func TestMCPMultiDest_MissingDestinationIsSkipped(t *testing.T) {
	_, p := mcpMDFixture(t)
	res := mcpExecTool("fedit_copy", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go",
		"dest": p("d1.go") + "," + p("missing.go"), "afterblock": "Init",
	})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "SKIP") || !strings.Contains(text, "1/2 file(s) succeeded") {
		t.Fatalf("unexpected result:\n%s", text)
	}
}

func TestMCPMultiDest_ReplaceDestBlock(t *testing.T) {
	_, p := mcpMDFixture(t)
	mcpMDWrite(t, p("d3.go"), "package main\n\nfunc Other() {}\n")
	res := mcpExecTool("fedit_replace", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go",
		"dest": p("d1.go") + "," + p("d3.go"), "destblock": "Target",
	})
	text := mcpMultiText(res)
	if res.IsError || !strings.Contains(text, "1/2 file(s) succeeded") || !strings.Contains(text, "SKIP") {
		t.Fatalf("unexpected result:\n%s", text)
	}
	got := mcpMDRead(t, p("d1.go"))
	if strings.Contains(got, "func Target") || !strings.Contains(got, "return 42") {
		t.Errorf("Target was not replaced by Helper:\n%s", got)
	}
	if mcpMDRead(t, p("d3.go")) != "package main\n\nfunc Other() {}\n" {
		t.Error("a destination without the block must be left unmodified")
	}
}

func TestMCPMultiDest_RejectedRequestsModifyNothing(t *testing.T) {
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"destblock-on-copy", "fedit_copy", map[string]any{"block": "Helper", "lang": "go", "afterblock": "Init", "destblock": "Target"}, "destblock is only supported with fedit_replace"},
		{"replace-without-destblock", "fedit_replace", map[string]any{"block": "Helper", "lang": "go"}, "requires destblock"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, p := mcpMDFixture(t)
			c.args["file"] = p("src.go")
			c.args["dest"] = p("d1.go") + "," + p("d2.go")
			before := mcpMDRead(t, p("src.go")) + mcpMDRead(t, p("d1.go")) + mcpMDRead(t, p("d2.go"))
			res := mcpExecTool(c.tool, c.args)
			text := mcpMultiText(res)
			if !res.IsError || !strings.Contains(text, c.want) {
				t.Errorf("want error containing %q, got:\n%s", c.want, text)
			}
			after := mcpMDRead(t, p("src.go")) + mcpMDRead(t, p("d1.go")) + mcpMDRead(t, p("d2.go"))
			if before != after {
				t.Error("a rejected request must not modify any file")
			}
		})
	}
}

func TestMCPMultiDest_NoDestKeepsSingleFilePath(t *testing.T) {
	_, p := mcpMDFixture(t)
	res := mcpExecTool("fedit_copy", map[string]any{
		"file": p("src.go"), "block": "Helper", "lang": "go", "afterblock": "Keep",
	})
	text := mcpMultiText(res)
	if strings.Contains(text, "file(s) succeeded") {
		t.Errorf("a call without dest must not take the dest path:\n%s", text)
	}
}

func TestMCPMultiDest_CommaListInFileFailsSafe(t *testing.T) {
	_, p := mcpMDFixture(t)
	before := mcpMDRead(t, p("src.go")) + mcpMDRead(t, p("d1.go"))
	res := mcpExecTool("fedit_copy", map[string]any{
		"file": p("src.go") + "," + p("d2.go"), "block": "Helper", "lang": "go",
		"dest": p("d1.go"), "afterblock": "Init",
	})
	if !res.IsError {
		t.Errorf("a comma list in file must not succeed:\n%s", mcpMultiText(res))
	}
	after := mcpMDRead(t, p("src.go")) + mcpMDRead(t, p("d1.go"))
	if before != after {
		t.Error("nothing may be modified when file is a list")
	}
}

func TestMCPSourceBlockContent(t *testing.T) {
	_, p := mcpMDFixture(t)
	block, msg := mcpSourceBlockContent(p("src.go"), "Helper", "go")
	if msg != "" || len(block) != 3 || !strings.HasPrefix(block[0], "func Helper") {
		t.Fatalf("unexpected block %q msg %q", block, msg)
	}
	if _, msg := mcpSourceBlockContent(p("src.go"), "Helper", ""); !strings.Contains(msg, "require lang") {
		t.Errorf("missing lang must be rejected, got %q", msg)
	}
	if _, msg := mcpSourceBlockContent(p("src.go"), "", "go"); !strings.Contains(msg, "used together") {
		t.Errorf("missing sourceblock must be rejected, got %q", msg)
	}
	if _, msg := mcpSourceBlockContent(p("src.go"), "NoSuchBlock", "go"); !strings.Contains(msg, "sourceblock") {
		t.Errorf("unknown block must be rejected, got %q", msg)
	}
}
