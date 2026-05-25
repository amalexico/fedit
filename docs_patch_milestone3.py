# docs_patch_milestone3.py
# Updates README.md, SKILL.md, fedit.html for MCP Standardization (Milestone 3):
#   - 12 -> 13 operations everywhere
#   - fedit_fields added to MCP tools table in README
#   - FAQ HCL map "planned for v1.6.0" -> "supported as of v1.5.0"
#   - CLAUDE_DESKTOP.md link added to README + fedit.html
#   - SKILL.md: fedit_map language list corrected (removed CSS/YAML/TOML/Markdown/Dockerfile/Makefile)
#   - SKILL.md: fedit_fields marked as MCP tool (not CLI-only)
#   - SKILL.md: 12 -> 13 in description and reference section
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit

import os, sys

README  = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\README.md"
SKILL   = r"C:\Users\kehsi\Desktop\amalex-brand\fedit\SKILL.md"
HTML    = r"C:\Users\kehsi\Desktop\amalex-brand\website\fedit.html"

def load(p):
    with open(p, "r", encoding="utf-8") as f: return f.read()

def save(p, c):
    with open(p, "w", encoding="utf-8", newline="\n") as f: f.write(c)

def rep(content, old, new, label):
    if old not in content:
        print(f"  FAIL {label!r} -- not found"); return content
    result = content.replace(old, new, 1)
    print(f"  OK   {label}"); return result

results = {}

# ═══════════════════════════════════════════════════════════════════════════
# README.md
# ═══════════════════════════════════════════════════════════════════════════
if not os.path.exists(README):
    print(f"ERROR: README not found: {README}"); sys.exit(1)

r = load(README)
lb = r.count("\n") + 1

# 1. "12 editing operations" -> 13
r = rep(r,
    "The server exposes all 12 editing operations as MCP tools:",
    "The server exposes all 13 editing operations as MCP tools:",
    "README: 12->13 operations")

# 2. Add fedit_fields to MCP tools table (after fedit_copy row)
r = rep(r,
    "| `fedit_copy` | Copy a line range; snapshot semantics, overlap allowed, -times N |",
    "| `fedit_copy` | Copy a line range; snapshot semantics, overlap allowed, -times N |\n"
    "| `fedit_fields` | Extract column N from CSV/TSV/delimited file (`-col N -delim CHAR`); always streaming |",
    "README: add fedit_fields to MCP table")

# 3. Add CLAUDE_DESKTOP.md reference after the Claude Desktop config block
r = rep(r,
    "### Configuring with Cursor",
    "> For a full setup guide (Windows paths, troubleshooting, Cursor/Cline/Windsurf snippets): "
    "[CLAUDE_DESKTOP.md](CLAUDE_DESKTOP.md)\n\n### Configuring with Cursor",
    "README: add CLAUDE_DESKTOP.md link")

# 4. Fix FAQ: map HCL "planned for v1.6.0" is now done
r = rep(r,
    "The `map` op does not yet show HCL structure (planned for v1.6.0).",
    "The `map` op supports HCL/Terraform and Nix as of v1.5.0 -- `fedit_map -lang hcl` returns all top-level block names and line ranges.",
    "README: fix FAQ HCL map status")

la = r.count("\n") + 1
save(README, r)
print(f"  README.md: {lb} -> {la} lines ({la-lb:+d})\n")

# ═══════════════════════════════════════════════════════════════════════════
# SKILL.md
# ═══════════════════════════════════════════════════════════════════════════
if not os.path.exists(SKILL):
    print(f"ERROR: SKILL not found: {SKILL}"); sys.exit(1)

s = load(SKILL)
lb = s.count("\n") + 1

# 1. Description: 12 MCP tools -> 13
s = rep(s,
    "fedit performs surgical line-anchored or content-matched edits via 12 MCP tools",
    "fedit performs surgical line-anchored or content-matched edits via 13 MCP tools",
    "SKILL: description 12->13")

# 2. fedit_map language list -- remove wrong langs, add correct ones
s = rep(s,
    "- **fedit_map** \u2014 structural overview (functions, headings, sections, code blocks). Supports 17 languages: Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust, Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile.",
    "- **fedit_map** \u2014 structural overview (functions, headings, sections, code blocks). Supports 17 languages: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix.",
    "SKILL: fedit_map language list corrected")

# 3. fields op: not CLI-only anymore
s = rep(s,
    "- `fields` op: CLI only (stdout); `-stream` flag: works on replaceall + find via MCP and CLI",
    "- `fields` op: available as `fedit_fields` MCP tool and CLI (`-op fields`). CLI outputs to stdout for piping.\n"
    "- `-stream` flag: works on replaceall + find via MCP and CLI",
    "SKILL: fedit_fields MCP tool note")

# 4. Reference: 12 operations -> 13, update ops list
s = rep(s,
    "- **Operations:** show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write, move, copy",
    "- **Operations:** show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write, move, copy, fields",
    "SKILL: operations list add fields")

s = rep(s,
    "- All 12 operations available as MCP tools when fedit is connected via \"fedit mcp\"",
    "- All 13 operations available as MCP tools when fedit is connected via `fedit mcp`",
    "SKILL: 12->13 in reference section")

la = s.count("\n") + 1
save(SKILL, s)
print(f"  SKILL.md: {lb} -> {la} lines ({la-lb:+d})\n")

# ═══════════════════════════════════════════════════════════════════════════
# fedit.html
# ═══════════════════════════════════════════════════════════════════════════
if not os.path.exists(HTML):
    print(f"ERROR: fedit.html not found: {HTML}"); sys.exit(1)

h = load(HTML)
lb = h.count("\n") + 1

# 1. "All 12 operations available as MCP tools" -> 13
h = rep(h,
    "All 12 operations available as MCP tools",
    "All 13 operations available as MCP tools",
    "HTML: 12->13 MCP tools")

# 2. "Twelve operations" -> "Thirteen operations" in section subtitle
h = rep(h,
    "Twelve operations, streaming engine, CSV/TSV fields, and HCL/Terraform + Nix block mappers.",
    "Thirteen operations, streaming engine, CSV/TSV fields, and HCL/Terraform + Nix block mappers.",
    "HTML: Twelve->Thirteen in subtitle")

# 3. Add CLAUDE_DESKTOP.md link to Claude Desktop card
h = rep(h,
    "        <li>Every edit returns line delta and timing stats</li>\n      </ul>\n    </div>\n    <div class=\"problem-card solution\">\n      <h3>Cursor",
    "        <li>Every edit returns line delta and timing stats</li>\n"
    "        <li><a href=\"https://github.com/amalexico/fedit/blob/main/CLAUDE_DESKTOP.md\" "
    "style=\"color:#4ECDC4;\">Full setup guide \u2192 CLAUDE_DESKTOP.md</a></li>\n"
    "      </ul>\n    </div>\n    <div class=\"problem-card solution\">\n      <h3>Cursor",
    "HTML: add CLAUDE_DESKTOP.md link in Claude Desktop card")

# 4. Fix logo: replace onerror hide with a text fallback SVG
# The current logo silently hides on 404. Replace with an inline SVG monogram.
h = rep(h,
    '<img src="/assets/logo-icon.png" alt="Amalex" onerror="this.style.display=\'none\'">',
    '<svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg" aria-label="Amalex">'
    '<rect width="32" height="32" rx="8" fill="#4ECDC4"/>'
    '<text x="16" y="22" text-anchor="middle" font-family="system-ui,sans-serif" font-weight="700" font-size="16" fill="#0d1f2d">A</text>'
    '</svg>',
    "HTML: inline SVG logo (replaces broken img)")

la = h.count("\n") + 1
save(HTML, h)
print(f"  fedit.html: {lb} -> {la} lines ({la-lb:+d})\n")

print("All done. Deploy:")
print("  cd C:\\Users\\kehsi\\Desktop\\amalex-brand\\fedit")
print("  git add README.md SKILL.md")
print('  git commit -m "docs: Milestone 3 -- 13 ops, fedit_fields MCP, CLAUDE_DESKTOP link, fix map FAQ"')
print("  git push")
print()
print("  cd C:\\Users\\kehsi\\Desktop\\amalex-brand\\website")
print("  npx wrangler pages deploy . --project-name=icy-glade-1626 --branch=production --commit-dirty=true")
