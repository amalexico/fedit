# ═══════════════════════════════════════════════════════════════
# fedit v1.2.0 — COMPLETE SHIP SCRIPT
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit
# ═══════════════════════════════════════════════════════════════

# ── STEP 0: Fix go vet (demo_sample.go declares its own func main) ──────────
fedit -file demo_sample.go -op insertbefore -match "package main" -text "//go:build ignore" -v

# ── STEP 1: Copy new files (already downloaded from Claude) ─────────────────
# Copy main.go, mcp.go, move_copy_test.go from your downloads to this folder.
# Then verify:
go vet ./...
go test ./...

# ── STEP 2: README.md ────────────────────────────────────────────────────────

# 2a. Add move + copy operation sections (after replaceall section)
[IO.File]::WriteAllText("$PWD\_move_copy_ops.txt", @'

---

### move — Move a line range to a new position

```bash
# Move lines 100-120 to after line 200 (explicit range)
fedit -file server.go -op move -line 100 -end 120 -after 200 -v

# Move a function block to before another function (content-matched)
fedit -file routes.go -op move -match "func OldHelper(" -end 45 -beforematch "func NewHelper(" -v

# Swap two nginx server blocks
fedit -file nginx.conf -op move -match "server {" -endmatch "# end server 1" -aftermatch "# end server 2" -v

# Cut once, scaffold 3 copies at destination
fedit -file main.go -op move -line 5 -end 12 -after 100 -times 3 -v
```

**Rules:**
- Destination may not overlap the source range — fedit reports a precise error with line numbers.
- `-times N`: cut once, paste N times. Net delta = blockSize × (N−1). Default times=1 = zero delta.

---

### copy — Copy a line range to a new position

```bash
# Copy a config block to after a section header
fedit -file values.yaml -op copy -line 50 -end 65 -aftermatch "# staging" -v

# Duplicate a test fixture 10 times for parameterised tests
fedit -file fixtures_test.go -op copy -match "func TestCase(" -end 30 -after 200 -times 10 -v

# Reorder Python classes (copy source before target; overlap is allowed)
fedit -file processor.py -op copy -match "class ModuleProcessor_15" -endmatch "class ModuleProcessor_16" -beforematch "class ModuleProcessor_13" -v
```

**Rules:**
- Snapshot semantics: source range is read once before any writes. All N copies are identical clones of the original, even when destination overlaps source.
- Net delta = blockSize × times.

'@)
fedit -file README.md -op insertafter -match "fedit -file Makefile -op replaceall -match" -textfile _move_copy_ops.txt -v
Remove-Item _move_copy_ops.txt

# 2b. Update MCP section count
fedit -file README.md -op replaceall -match "all 10 editing operations as MCP tools" -text "all 12 editing operations as MCP tools" -v
fedit -file README.md -op replaceall -match "exposes all 10 editing operations" -text "exposes all 12 editing operations" -v

# 2c. Add fedit_move and fedit_copy rows to MCP tools table
[IO.File]::WriteAllText("$PWD\_mcp_rows.txt", @'
| `fedit_move` | Move a line range to a new position; destination-overlap rejected |
| `fedit_copy` | Copy a line range; snapshot semantics, overlap allowed, -times N |
'@)
fedit -file README.md -op insertafter -match "fedit_insertbefore" -textfile _mcp_rows.txt -v
Remove-Item _mcp_rows.txt

# 2d. Update Quick Start to mention move/copy
[IO.File]::WriteAllText("$PWD\_qs_movecopy.txt", @'

# Move a function block before another (content-matched, atomic)
fedit -file main.go -op move -match "func OldHelper(" -end 45 -beforematch "func NewHelper(" -v

# Copy a config block and paste it 3 times at a new location
fedit -file values.yaml -op copy -line 50 -end 65 -after 200 -times 3 -v
'@)
fedit -file README.md -op insertafter -match "fedit -file nginx.conf -op insertafter" -textfile _qs_movecopy.txt -v
Remove-Item _qs_movecopy.txt

# ── STEP 3: SKILL.md ─────────────────────────────────────────────────────────

# 3a. Update operations list in Reference section
fedit -file SKILL.md -op replaceall -match "show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write" -text "show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write, move, copy" -v

# 3b. Update MCP tool count in Reference section
fedit -file SKILL.md -op replaceall -match "All operations available as MCP tools" -text "All 12 operations available as MCP tools" -v

# 3c. Add move/copy usage pattern to the Common patterns section
[IO.File]::WriteAllText("$PWD\_skill_movecopy.txt", @'

### Move a block before/after another block

    fedit_find (match="func Target")    -> confirm line numbers
    fedit_move (line=N, end=M, beforematch="func Destination")

For Python class reordering, content-match both bounds:
    fedit_move (match="class A", endmatch="class B", beforematch="class C")

**Move overlap rule:** destination inside source range is always rejected.
Use fedit_find to verify bounds before calling fedit_move on large files.

### Duplicate a block N times (scaffolding / test fixtures)

    fedit_copy (line=N, end=M, after=InsertPoint, times=10)

Snapshot semantics: all 10 copies are identical clones of the original block
at read time — even if destination overlaps source range.

'@)
fedit -file SKILL.md -op insertbefore -match "## Anti-patterns" -textfile _skill_movecopy.txt -v
Remove-Item _skill_movecopy.txt

# 3d. Update description line (fedit is now 12 tools)
fedit -file SKILL.md -op replaceall -match "fedit performs surgical line-anchored or content-matched edits via MCP tools" -text "fedit performs surgical line-anchored or content-matched edits via 12 MCP tools" -v

# ── STEP 4: Update project status file ──────────────────────────────────────
# (fedit has its own status entries — update via fedit on the status file)

$statusFile = "C:\Users\kehsi\Desktop\amalex-brand\Amalex_handler_current_status.txt"

# 4a. Bump fedit current version
fedit -file $statusFile -op replaceall -match "CURRENT VERSION: v1.1.0 (tag pushed, go install verified)" -text "CURRENT VERSION: v1.2.0 (tag pushed, go install verified)" -v

# 4b. Update fedit operations count
fedit -file $statusFile -op replaceall -match "OPERATIONS (11 total" -text "OPERATIONS (12 total" -v

# 4c. Add move/copy to operations list
fedit -file $statusFile -op replaceall -match "show, insert, delete, replace, write, map, find,`n  insertafter (RECOMMENDED), insertbefore (RECOMMENDED), replaceall" -text "show, insert, delete, replace, write, map, find,`n  insertafter (RECOMMENDED), insertbefore (RECOMMENDED), replaceall,`n  move, copy" -v

# 4d. Mark Step 15 (fedit v1.2.0) as DONE in AGREED TASK ORDER
fedit -file $statusFile -op replaceall -match "Step 15 — fedit v1.2.0 (move + copy block ops)     ⬜ PARALLEL TRACK" -text "Step 15 — fedit v1.2.0 (move + copy block ops)     ✅ DONE (move+copy+MCP, 86 tests)" -v

# 4e. Update NEXT pointer (Step 16 is now next)
fedit -file $statusFile -op replaceall -match "Step 14 — Phase 4B Step 3                          ⬅ NEXT" -text "Step 14 — Phase 4B Step 3                          ⬅ NEXT (resume after v1.2.0 tag)" -v

# 4f. Mark fedit v1.2.0 section as shipped
fedit -file $statusFile -op replaceall -match "v1.2.0 — Block operations (move + copy)" -text "v1.2.0 — Block operations (move + copy)            ✅ SHIPPED" -v

# ── STEP 5: Git — commit + tag + push ────────────────────────────────────────

[IO.File]::WriteAllText("$PWD\_msg.txt", @'
feat: move + copy block operations with -times N (v1.2.0)

New operations:
  move  Atomic cut-once-paste-N; dest-inside-src rejected with precise error
  copy  Atomic snapshot copy; overlap allowed; all N copies identical clones

New flags:
  -after/-before/-aftermatch/-beforematch  destination (exactly one required)
  -endmatch TEXT   content-based source end bound
  -times N         repeat 1-N times (default 1)

MCP: fedit_move + fedit_copy added (12 tools total)
Tests: 86 tests in move_copy_test.go; all passing
'@)

git add main.go mcp.go move_copy_test.go README.md SKILL.md demo_sample.go
git commit -F _msg.txt
Remove-Item _msg.txt

git tag v1.2.0
git push
git push --tags

Write-Host "`n✅ fedit v1.2.0 shipped. go install github.com/amalexico/fedit@latest now serves v1.2.0`n"
