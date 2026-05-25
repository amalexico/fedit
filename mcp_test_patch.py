# mcp_test_patch.py
# Fixes: def.InputSchema is any (holds json.RawMessage) -- needs type assertion
# Adds schemaBytes() helper and replaces all direct json.Unmarshal(def.InputSchema,...)
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os, sys

TEST_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\mcp_test.go"

def load(p):
    with open(p, "r", encoding="utf-8") as f: return f.read()

def save(p, c):
    with open(p, "w", encoding="utf-8", newline="\n") as f: f.write(c)

if not os.path.exists(TEST_FILE):
    print(f"ERROR: not found: {TEST_FILE}"); sys.exit(1)

content = load(TEST_FILE)

# 1. Add schemaBytes helper after the import block
OLD_IMPORT = """import (
\t"encoding/json"
\t"os"
\t"strings"
\t"testing"
)"""

NEW_IMPORT = """import (
\t"encoding/json"
\t"os"
\t"strings"
\t"testing"
)

// schemaBytes extracts the raw JSON bytes from a tool's InputSchema.
// InputSchema is typed as any but always holds a json.RawMessage.
func schemaBytes(def mcpToolDef) []byte {
\tif raw, ok := def.InputSchema.(json.RawMessage); ok {
\t\treturn []byte(raw)
\t}
\tb, _ := json.Marshal(def.InputSchema)
\treturn b
}"""

if OLD_IMPORT in content:
    content = content.replace(OLD_IMPORT, NEW_IMPORT, 1)
    print("  OK   added schemaBytes() helper")
else:
    print("  FAIL import block not found"); sys.exit(1)

# 2. Replace all json.Unmarshal(def.InputSchema, with json.Unmarshal(schemaBytes(def),
old = "json.Unmarshal(def.InputSchema,"
new = "json.Unmarshal(schemaBytes(def),"
count = content.count(old)
content = content.replace(old, new)
print(f"  OK   replaced {count} occurrence(s) of json.Unmarshal(def.InputSchema,...)")

save(TEST_FILE, content)
print(f"mcp_test.go patched")
