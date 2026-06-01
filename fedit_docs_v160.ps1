# fedit_docs_v160.ps1 - Update README, CLAUDE_DESKTOP, SKILL for v1.6.0
# Run with: powershell -ExecutionPolicy Bypass -File .\fedit_docs_v160.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Check([string]$step) {
    if ($LASTEXITCODE -ne 0) { Write-Error "FAILED: $step"; exit 1 }
}

Write-Host ""
Write-Host "=== docs v1.6.0 patch ===" -ForegroundColor Cyan
Write-Host ""

# ================================================================
# README.md
# ================================================================
$r = "README.md"
Write-Host "--- README.md ---" -ForegroundColor Yellow

# Version bumps: v1.5.0 -> v1.6.0 in non-historical contexts
# (Keep "as of v1.5.0" for HCL since that's factual history)
fedit -file $r -op replaceall -match 'v1.5.0+' -text 'v1.6.0+'
# (may not match, ignore exit code for optional bumps)
$LASTEXITCODE = 0

fedit -file $r -op replaceall -match 'Update to v1.5.0+.' -text 'Update to v1.6.0+.'
$LASTEXITCODE = 0

fedit -file $r -op replaceall -match 'v1.5.0 release notes' -text 'v1.5.0 release notes'
$LASTEXITCODE = 0

# 13 -> 14 ops in MCP section
fedit -file $r -op replaceall `
    -match 'all 13 editing operations as MCP tools' `
    -text  'all 14 editing operations as MCP tools'
Check "README 13->14"

# Add fedit_writeraw after fedit_write in MCP table
fedit -file $r -op insertafter `
    -match '| `fedit_write` | Create or overwrite a file |' `
    -textfile _doc_readme_td_writeraw.md
Check "README fedit_writeraw table row"

# Add v1.6.0 Quick Start examples after the fields example
fedit -file $r -op insertafter `
    -match 'fedit -file data.tsv -op fields -col 2' `
    -textfile _doc_readme_qs_v160.md
Check "README QS examples"

# Add new ops section (writeraw, writelines, flags) after the -stream section
fedit -file $r -op insertafter `
    -match 'Not supported: `move`, `copy`, `map`' `
    -textfile _doc_readme_ops_v160.md
Check "README new ops section"

Write-Host "README.md done" -ForegroundColor Green

# ================================================================
# CLAUDE_DESKTOP.md
# ================================================================
$cd = "CLAUDE_DESKTOP.md"
Write-Host "--- CLAUDE_DESKTOP.md ---" -ForegroundColor Yellow

# Prereq version bump
fedit -file $cd -op replaceall `
    -match 'fedit v1.5.0+ installed' `
    -text  'fedit v1.6.0+ installed'
Check "CD version prereq"

fedit -file $cd -op replaceall `
    -match 'Verify: `fedit` prints the usage block' `
    -text  'Verify: `fedit` with no args prints the usage block'
$LASTEXITCODE = 0

# Add fedit_writeraw to tool table
fedit -file $cd -op insertafter `
    -match '| `fedit_write` | Write or overwrite an entire file |' `
    -textfile _doc_cd_td_writeraw.md
Check "CD writeraw table row"

# Update fedit_find row to mention -x
fedit -file $cd -op replaceall `
    -match '| `fedit_find` | Find lines matching a substring; `stream=true` for large files |' `
    -text  '| `fedit_find` | Find lines matching a substring; `stream=true` for large files; `x=true` for bare line numbers |'
Check "CD find row"

# Update fedit_insert row to mention cleanfirst
fedit -file $cd -op replaceall `
    -match '| `fedit_insert` | Insert content after line N |' `
    -text  '| `fedit_insert` | Insert content after line N; `cleanfirst=true` to truncate first |'
Check "CD insert row"

# Update troubleshooting version mention
fedit -file $cd -op replaceall `
    -match 'show `fields` and `move`/`copy` operations. Update to v1.5.0+.' `
    -text  'show `writeraw`, `writelines`, `fields` and `move`/`copy`. Update to v1.6.0+.'
Check "CD troubleshooting"

# Add v1.6.0 changelog (insertafter v1.5.0 line)
fedit -file $cd -op insertafter `
    -match 'v1.5.0 release notes: HCL/Terraform' `
    -textfile _doc_cd_changelog.md
Check "CD changelog"

Write-Host "CLAUDE_DESKTOP.md done" -ForegroundColor Green

# ================================================================
# SKILL.md
# ================================================================
$sk = "SKILL.md"
Write-Host "--- SKILL.md ---" -ForegroundColor Yellow

# 13 -> 14 in description
fedit -file $sk -op replaceall `
    -match 'fedit performs surgical line-anchored or content-matched edits via 13 MCP tools' `
    -text  'fedit performs surgical line-anchored or content-matched edits via 14 MCP tools'
Check "SKILL 13->14 desc"

fedit -file $sk -op replaceall `
    -match 'All 13 operations available as MCP tools' `
    -text  'All 14 operations available as MCP tools'
Check "SKILL 13->14 ref"

# Add writeraw to mutation ops list (after fedit_write bullet)
fedit -file $sk -op insertafter `
    -match '**fedit_write** -- overwrite or create a file' `
    -textfile _doc_skill_writeraw.md
Check "SKILL writeraw op"

# Add -x note after fields op section
fedit -file $sk -op insertafter `
    -match '`fields` op: available as `fedit_fields` MCP tool' `
    -textfile _doc_skill_x.md
Check "SKILL -x note"

# Add writeraw to operations reference list
fedit -file $sk -op replaceall `
    -match '- **Operations:** show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write, move, copy, fields' `
    -text  '- **Operations:** show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write, writeraw, writelines, move, copy, fields'
Check "SKILL ops ref"

Write-Host "SKILL.md done" -ForegroundColor Green

Write-Host ""
Write-Host "All doc patches applied." -ForegroundColor Cyan
Write-Host "Review with: fedit -file README.md -op find -match v1.6.0"
Write-Host "Then: git add -A ; git commit -m 'docs: v1.6.0 - writeraw, cleanfirst, -x, -texthex' ; git push"
