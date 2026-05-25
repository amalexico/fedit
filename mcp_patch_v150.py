# mcp_patch_v150.py
# Fixes mcp.go for MCP Standardization (Milestone 3):
#   1. fedit_find: stream param was inside required[] -- moved to properties{}
#      (broken JSON caused strict MCP clients to reject tools/list)
#   2. fedit_map: description now lists all 17 languages including hcl/tf/terraform/nix
#      lang param description now lists valid values explicitly
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os, sys, json

MCP_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\mcp.go"

def load(path):
    with open(path, "r", encoding="utf-8") as f:
        return f.read()

def save(path, content):
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(content)

def replace_once(content, old, new, label):
    if old not in content:
        print(f"  FAIL {label!r} -- target not found")
        return content, False
    if content.count(old) > 1:
        print(f"  WARN {label!r} -- found multiple occurrences, replacing first")
    result = content.replace(old, new, 1)
    print(f"  OK   {label}")
    return result, True

if not os.path.exists(MCP_FILE):
    print(f"ERROR: not found: {MCP_FILE}")
    sys.exit(1)

content = load(MCP_FILE)
lines_before = content.count("\n") + 1

# ── Fix 1: fedit_find schema (critical) ─────────────────────────────────────
# stream was embedded inside required[], causing malformed JSON.
# Also adds stream description to the tool description string.

OLD_FIND = (
    '\t\t{Name: "fedit_find", Description: "Find all lines matching a substring.", '
    'InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},'
    '"match":{"type":"string","description":"Substring to search for"},'
    '"nth":{"type":"integer","description":"Which occurrence (default 1, -1 for last)"}},'
    '"required":["file","match","stream":{"type":"boolean","description":"Streaming grep-style output for large files"}},'
    '"required":["file","match"]}`)}'
)

NEW_FIND = (
    '\t\t{Name: "fedit_find", Description: "Find all lines matching a substring. Add stream=true for large files.", '
    'InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},'
    '"match":{"type":"string","description":"Substring to search for"},'
    '"nth":{"type":"integer","description":"Which occurrence (default 1, -1 for last)"},'
    '"stream":{"type":"boolean","description":"Streaming grep-style output for large files"}},'
    '"required":["file","match"]}`)}'
)

content, ok1 = replace_once(content, OLD_FIND, NEW_FIND, "fedit_find schema (stream moved to properties)")

# ── Fix 2: fedit_map description + lang param ────────────────────────────────
# Description didn't mention HCL/Nix. Lang param had no list of valid values.

OLD_MAP = (
    '\t\t{Name: "fedit_map", Description: "Structural overview of a source file. Supports 17 languages.", '
    'InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},'
    '"lang":{"type":"string","description":"Language (auto-detected from extension if omitted)"}},'
    '"required":["file"]}`)}'
)

NEW_MAP = (
    '\t\t{Name: "fedit_map", Description: "Structural overview of a source file. '
    'Supports 17 languages: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix. '
    'Use lang param for ambiguous extensions.", '
    'InputSchema: s(`{"type":"object","properties":{"file":{"type":"string","description":"Path to file"},'
    '"lang":{"type":"string","description":"Language hint: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix (auto-detected from extension if omitted)"}},'
    '"required":["file"]}`)}'
)

content, ok2 = replace_once(content, OLD_MAP, NEW_MAP, "fedit_map lang list in description + schema")

# ── Validate the fixed fedit_find JSON ───────────────────────────────────────
import re
m = re.search(r'fedit_find.*?InputSchema: s\(`({.*?})`\)', content, re.DOTALL)
if m:
    raw = m.group(1)
    try:
        json.loads(raw)
        print("  OK   fedit_find schema JSON validates cleanly")
    except json.JSONDecodeError as e:
        print(f"  FAIL fedit_find JSON still invalid: {e}")
else:
    print("  WARN could not locate fedit_find schema for validation")

# ── Save ─────────────────────────────────────────────────────────────────────
lines_after = content.count("\n") + 1
save(MCP_FILE, content)
print(f"mcp.go patched: {lines_before} -> {lines_after} lines ({lines_after - lines_before:+d})")
print()
print("Next steps:")
print("  go build -o fedit.exe ./...")
print("  go test ./...")
print("  git add mcp.go")
print('  git commit -m "fix: fedit_find schema (stream in properties), fedit_map lang list"')
print("  git push")
