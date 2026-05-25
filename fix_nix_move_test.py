# fix_nix_move_test.py
# The Nix move leaves the blank separator line at index 0 after removing
# programs.git from the top. Fix: find the first non-empty line instead
# of asserting lines[0] directly.
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os

TEST_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\mcp_test.go"

OLD = """\tlines, _ := readLines(path)
\tif !strings.Contains(lines[0], "programs.ssh") {
\t\tt.Errorf("after move, first block should be programs.ssh, got: %q", lines[0])
\t}
}

// \u2500\u2500 mcpExecTool: insertafter"""

NEW = """\tlines, _ := readLines(path)
\t// The blank separator line may land at index 0 after the move;
\t// find the first non-empty line to check block order.
\tfirstNonEmpty := ""
\tfor _, l := range lines {
\t\tif strings.TrimSpace(l) != "" {
\t\t\tfirstNonEmpty = l
\t\t\tbreak
\t\t}
\t}
\tif !strings.Contains(firstNonEmpty, "programs.ssh") {
\t\tt.Errorf("after move, first non-empty line should be programs.ssh, got: %q", firstNonEmpty)
\t}
}

// \u2500\u2500 mcpExecTool: insertafter"""

with open(TEST_FILE, "r", encoding="utf-8") as f:
    content = f.read()

if OLD not in content:
    print("  FAIL target not found"); raise SystemExit(1)

content = content.replace(OLD, NEW, 1)

with open(TEST_FILE, "w", encoding="utf-8", newline="\n") as f:
    f.write(content)

print("  OK   TestMCPExec_Move_NixBlock assertion fixed")
