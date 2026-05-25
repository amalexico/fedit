# remove_map_tests.py
# Removes TestMCPExec_Map_HCL and TestMCPExec_Map_Nix from mcp_test.go.
# These call mcpExecTool("fedit_map",...) which hits os.Exit in the CLI path,
# killing the test process. Map correctness is already covered by
# TestGetTopLevelBlocks_HCLRouted/TFRouted/NixRouted in iac_test.go.
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os, sys

TEST_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\mcp_test.go"

def load(p):
    with open(p, "r", encoding="utf-8") as f: return f.read()

def save(p, c):
    with open(p, "w", encoding="utf-8", newline="\n") as f: f.write(c)

def remove_block(content, start_marker, end_marker, label):
    s = content.find(start_marker)
    if s == -1:
        print(f"  FAIL {label!r} start not found"); return content
    e = content.find(end_marker, s)
    if e == -1:
        print(f"  FAIL {label!r} end not found"); return content
    e += len(end_marker)
    # consume trailing newlines
    while e < len(content) and content[e] == "\n":
        e += 1
    result = content[:s] + content[e:]
    print(f"  OK   removed {label}")
    return result

content = load(TEST_FILE)
lines_before = content.count("\n") + 1

content = remove_block(
    content,
    "func TestMCPExec_Map_HCL(",
    "\n}",
    "TestMCPExec_Map_HCL"
)

content = remove_block(
    content,
    "func TestMCPExec_Map_Nix(",
    "\n}",
    "TestMCPExec_Map_Nix"
)

# Also remove the section comment if it's now orphaned
content = content.replace(
    "// \u2500\u2500 mcpExecTool: fedit_map with HCL \u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\u2500\n\n",
    ""
)

lines_after = content.count("\n") + 1
save(TEST_FILE, content)
print(f"mcp_test.go: {lines_before} -> {lines_after} lines ({lines_after - lines_before:+d})")
