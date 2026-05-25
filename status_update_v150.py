"""
status_update_v150.py
Updates Amalex_handler_current_status.txt to reflect:
  - fedit v1.5.0 shipped (HCL/Terraform + Nix block scanners)
  - All docs committed and website deployed
  - Session history recap (May 8-13 work)
  - Immediate next steps: MCP Standardization (Milestone 3)
  - Known bug in mcp.go fedit_find schema (stream misplaced)

Run from: C:\Users\kehsi\Desktop\amalex-brand
"""

import os
import sys

STATUS_FILE = r"C:\Users\kehsi\Desktop\amalex-brand\Amalex_handler_current_status.txt"

def load(path):
    with open(path, "r", encoding="utf-8") as f:
        return f.read()

def save(path, content):
    with open(path, "w", encoding="utf-8", newline="\n") as f:
        f.write(content)

def replace_once(content, old, new, label):
    if old not in content:
        print(f"  FAIL {label!r} — target not found")
        return content, False
    count = content.count(old)
    if count > 1:
        print(f"  WARN {label!r} — found {count} occurrences, replacing first only")
    result = content.replace(old, new, 1)
    print(f"  OK   {label}")
    return result, True

# ── Load ────────────────────────────────────────────────────────────────────

if not os.path.exists(STATUS_FILE):
    print(f"ERROR: status file not found at {STATUS_FILE}")
    sys.exit(1)

original = load(STATUS_FILE)
content = original
lines_before = content.count("\n") + 1

# ── Patch 1: Header — Last Updated ─────────────────────────────────────────

# Try a few possible states the header might be in after previous scripts
for old_date in [
    "Last Updated: May 4, 2026 (Phase 4B Step 2: Dropbox handler shipped)",
    "Last Updated: May 11, 2026",
    "Last Updated: May 12, 2026",
    "Last Updated: May 13, 2026",
    "Last Updated: May 10, 2026",
    "Last Updated: May 9, 2026",
]:
    if old_date in content:
        content, _ = replace_once(
            content, old_date,
            "Last Updated: May 18, 2026 (fedit v1.5.0 shipped — MCP Standardization next)",
            "last updated date"
        )
        break
else:
    # Fallback: find any "Last Updated: May" line
    import re
    m = re.search(r"Last Updated: May [^\n]+", content)
    if m:
        content, _ = replace_once(
            content, m.group(0),
            "Last Updated: May 18, 2026 (fedit v1.5.0 shipped — MCP Standardization next)",
            "last updated date (regex fallback)"
        )
    else:
        print("  FAIL last updated date — no pattern found")

# ── Patch 2: Status line ────────────────────────────────────────────────────

for old_status in [
    "Status: DROPBOX HANDLER SHIPPED (commit ae14a7e) → STEP 3 REGISTRY WIRING + FEDIT v1.2.0 PARALLEL",
    "Status: FEDIT v1.4.0 SHIPPED",
    "Status: FEDIT v1.4.0",
]:
    if old_status in content:
        content, _ = replace_once(
            content, old_status,
            "Status: FEDIT v1.5.0 SHIPPED (commit 84acdc8) \u2192 MCP STANDARDIZATION (Milestone 3)",
            "status line"
        )
        break
else:
    print("  SKIP status line — may have been updated by a previous script; check manually")

# ── Patch 3: Immediate Next Steps section ───────────────────────────────────

# Replace whatever the current IMMEDIATE NEXT STEPS block says
OLD_NEXT = """\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550
END OF PROJECT STATUS REPORT"""

# We'll insert our new IMMEDIATE NEXT STEPS before the final separator
# First find the IMMEDIATE NEXT STEPS header
NEXT_HEADER = "\u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\n\u2551 IMMEDIATE NEXT STEPS"

if NEXT_HEADER in content:
    # Find the block and replace through the closing box
    import re
    # Match from IMMEDIATE NEXT STEPS header to the next double-line separator
    pattern = re.compile(
        r"\u2554[^\n]*\n\u2551 IMMEDIATE NEXT STEPS[^\n]*\n\u255a[^\n]*\n.*?(?=\n\u2550{10,}|\Z)",
        re.DOTALL
    )
    m = pattern.search(content)
    if m:
        new_next = """\u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550
\u2551 IMMEDIATE NEXT STEPS                                         \u2551
\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550

  1. MCP Standardization (Milestone 3) -- NEXT
     a. Fix fedit_find schema bug in mcp.go:
        stream param is inside required[] instead of properties{}
        This causes JSON parse errors in strict MCP clients
     b. Audit all 13 tool schemas for correctness
     c. Add stream param to fedit_find properly
     d. Update fedit_map + fedit_move + fedit_copy descriptions to mention HCL/Nix
     e. Write CLAUDE_DESKTOP.md integration guide
     f. Add MCP-layer tests (fedit_fields, stream, HCL/Nix block ops via MCP)
     g. Update SKILL.md with exact JSON schemas agents should use

  2. r/devops re-engagement post (after MCP standardization)
     Hook: "fedit now understands Terraform blocks"
     Demo: fedit -op move -block 'resource "aws_instance" "web"'
           -beforeblock 'resource "aws_s3_bucket" "data"' -lang hcl -v
     Thread had 12K views. Terraform angle is the strongest hook yet.
     DO NOT post until MCP standardization is documented (agents need a guide).

  3. Phase 4B Step 3: Register Dropbox handler (resume after MCP + r/devops)
     See ADDENDUM May 5, 2026 for full step-by-step.
     Known risk: mw.ValidateCSRF(r) in handleOAuthRevoke -- verify exists.
     Known risk: cli.go oauth import -- verify it compiles.

  4. fedit v2.0.0 / Security hardening (Milestone 4)
     Strategic milestone for enterprise positioning."""
        content = content[:m.start()] + new_next + content[m.end():]
        print("  OK   IMMEDIATE NEXT STEPS section")
    else:
        print("  SKIP IMMEDIATE NEXT STEPS — regex didn't match block; will be in addendum")
else:
    print("  SKIP IMMEDIATE NEXT STEPS header not found; will be in addendum")

# ── Patch 4: fedit roadmap step markers ─────────────────────────────────────

content, _ = replace_once(
    content,
    "Step 19 -- IaC mappers: HCL/Terraform + Nix         \u2b05 NEXT",
    "Step 19 -- IaC mappers: HCL/Terraform + Nix         \u2705 DONE (commit 84acdc8, tag v1.5.0)",
    "fedit roadmap Step 19 marker"
)

content, _ = replace_once(
    content,
    "Step 20 -- MCP standardization + integration guides   \u25ab TODO",
    "Step 20 -- MCP standardization + integration guides   \u2b05 NEXT",
    "fedit roadmap Step 20 marker"
)

# ── Addendum: full v1.5.0 session recap ─────────────────────────────────────

ADDENDUM = """
\u2554\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550
\u2551 ADDENDUM -- May 18, 2026 (Sessions May 8-13 recap + MCP Standardization prep) \u2551
\u255a\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550\u2550

SESSIONS MAY 8-13 -- COMPLETED WORK:

  fedit v1.3.0 (May 8):
    \u2705 -match-regex: regex replaceall with capture groups ($1, $2)
    \u2705 -files GLOB: apply replaceall across multiple files
    \u2705 fedit.html updated + deployed
    \u2705 README.md + SKILL.md docs committed (commit 4d32e88)

  fedit v1.4.0 (May 9-10):
    \u2705 Stream engine: line-by-line I/O with 10MB buffer (replaceall, find)
       Atomic integrity: writes to .fedit_stream_tmp then os.Rename
    \u2705 fields op: extract column N from delimited files (-col N -delim CHAR)
       Always streaming, outputs to stdout for piping
    \u2705 Usage block updated for all 13 operations + new flags
    \u2705 Git tag v1.4.0 fixed (was pointing to wrong commit; corrected to 3d09863)
    \u2705 273 tests passing
    \u2705 README.md + SKILL.md docs (commit 2976d94)
    \u2705 fedit.html deployed
    \u2705 Status file updated by status_update_v140.py (1235 -> 1267 lines)

  fedit v1.5.0 (May 11-13):
    \u2705 getHCLBlocks(): Handles all 11 Terraform block types:
       resource, data, module, provider, variable, output,
       locals, terraform, moved, import, check
       Nested blocks correctly ignored. Single-line blocks work.
       Three aliases: -lang hcl | -lang tf | -lang terraform
    \u2705 getNixBlocks(): Top-level attribute bindings with { or [ bodies:
       programs.git = { }, services.nginx = { }, environment.systemPackages = [ ]
    \u2705 Wired into getTopLevelBlocks() switch
    \u2705 333 tests passing (iac_test.go: 40 tests added)
    \u2705 Committed: feat: HCL/Terraform and Nix block scanners (v1.5.0) (84acdc8)
    \u2705 Tag: v1.5.0 (points to 84acdc8)
    \u2705 Docs: README.md + SKILL.md committed (a14a7d1)
    \u2705 fedit.html deployed (Cloudflare Pages)

FEDIT CURRENT STATE (May 18, 2026):
  Version:       v1.5.0
  Commit:        a14a7d1 (docs -- latest)
  Code commit:   84acdc8 (feat: IaC block scanners)
  Tag:           v1.5.0
  Tests:         333 passing
  Operations:    13 (show, insert, delete, replace, write, map, find,
                     insertafter, insertbefore, replaceall, move, copy, fields)
  Flags:         -file, -op, -line, -end, -text, -textfile, -match, -nth,
                 -stream, -col, -delim, -block, -lang, -match-regex, -files,
                 -after, -before, -aftermatch, -beforematch,
                 -afterblock, -beforeblock, -endmatch, -times, -v
  Languages:     go, python, js, ts, rust, java, cs, ruby, php,
                 hcl, tf, terraform, nix, html, sql (17 total for map)
  Repo:          github.com/amalexico/fedit (PUBLIC)
  Local:         C:\\Users\\kehsi\\Desktop\\amalex-brand\\fedit

MCP SERVER (mcp.go) KNOWN BUG:
  fedit_find schema has malformed JSON -- stream field is embedded
  inside the required[] array instead of the properties{} object.
  Current (broken):
    "required":["file","match","stream":{"type":"boolean",...}]
  Fix needed:
    stream must be in properties{} and required[] should be ["file","match"]
  Impact: strict MCP clients (Cursor, Claude Desktop) may fail to parse
  the tools/list response. Fix is the first task of Milestone 3.

MCP STANDARDIZATION PLAN (Milestone 3) -- STARTING NOW:
  Code (mcp.go):
    1. Fix fedit_find schema (stream misplaced -- see bug above)
    2. Audit all 13 tool schemas for correctness + completeness
    3. Update fedit_map description: add hcl, tf, terraform, nix to lang list
    4. Update fedit_move + fedit_copy: lang field already lists hcl/tf/terraform/nix
       -- verify descriptions match
    5. fedit_fields is already a proper MCP tool -- verify schema is clean
    6. Version: ServerInfo already shows "1.5.0" -- correct

  Documentation:
    7. CLAUDE_DESKTOP.md: claude_desktop_config.json snippet + usage examples
    8. SKILL.md: update with exact MCP JSON schemas agents should use
    9. fedit.html: add MCP Server section details

  Tests:
    10. MCP-layer tests: fedit_fields, stream params, HCL/Nix block ops via MCP
        Test that tools/list JSON is valid and parseable
        Test that fedit_find stream param works via MCP path

POWERSHELL REMINDERS (never forget):
  - Shell: pwsh. NEVER use && -- use ; instead
  - Patches: [IO.File]::WriteAllText + fedit. NEVER @"..."@ heredocs
  - ALWAYS use @'...'@ single-quote heredocs if heredoc is needed
  - Double quotes inside -text args get stripped -- always use -textfile
  - fedit insert op inserts AFTER the given line number (not before)
  - insertafter/insertbefore require -match, not -line
  - NEVER chain write + fedit in same block

KEY FACTS:
  - Dropbox app key: bihqc9zghmhhgcr
  - OAuth redirect: http://localhost:53682/oauth/callback
  - Migration is 016 (not 015 -- conflict fixed)
  - sqlc package: internal/database/sqlc
  - server.go: 4626 lines. engine.go: 947 lines.
  - connections_form.html: 652 lines.
  - mw.ValidateCSRF(r) used in handleOAuthRevoke -- verify exists
  - Indentation in handleConnectionEditForm is unformatted -- run gofmt after build
"""

# Append addendum before the final END OF PROJECT STATUS REPORT line
END_MARKER = "\u2550" * 63 + "\nEND OF PROJECT STATUS REPORT"
if END_MARKER in content:
    content = content.replace(END_MARKER, ADDENDUM + "\n" + END_MARKER)
    print("  OK   addendum appended before END marker")
else:
    # Just append at end
    content = content.rstrip() + "\n" + ADDENDUM
    print("  OK   addendum appended at end of file (END marker not found)")

# Update the END line date
content, _ = replace_once(
    content,
    "END OF PROJECT STATUS REPORT -- May 4, 2026",
    "END OF PROJECT STATUS REPORT -- May 18, 2026",
    "end date"
)
# Also try other dates
for d in ["May 5, 2026", "May 9, 2026", "May 10, 2026", "May 11, 2026", "May 13, 2026"]:
    if f"END OF PROJECT STATUS REPORT -- {d}" in content:
        content, _ = replace_once(
            content,
            f"END OF PROJECT STATUS REPORT -- {d}",
            "END OF PROJECT STATUS REPORT -- May 18, 2026",
            "end date"
        )

# ── Save ────────────────────────────────────────────────────────────────────

lines_after = content.count("\n") + 1
save(STATUS_FILE, content)
print(f"Status file updated: {lines_before} -> {lines_after} lines ({lines_after - lines_before:+d})")
