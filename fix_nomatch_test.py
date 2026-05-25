# fix_nomatch_test.py
# fedit_find returns IsError=true when no lines match. Fix test expectation.
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os

TEST_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\mcp_test.go"

old = """\t// No match is not an error \u2014 just zero results
\tif r.IsError {
\t\tt.Errorf("find with no match should not be an error: %s", mcpText(r))
\t}"""

new = """\t// fedit_find returns IsError=true when no lines match \u2014 correct behavior
\tif !r.IsError {
\t\tt.Error("find with no match should return IsError=true")
\t}"""

with open(TEST_FILE, "r", encoding="utf-8") as f:
    content = f.read()

if old not in content:
    print("  FAIL target not found"); raise SystemExit(1)

content = content.replace(old, new, 1)

with open(TEST_FILE, "w", encoding="utf-8", newline="\n") as f:
    f.write(content)

print("  OK   TestMCPExec_Find_NoMatch expectation fixed")
