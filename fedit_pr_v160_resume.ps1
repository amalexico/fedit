# fedit_pr_v160_resume.ps1 - Resume v1.6.0 patch from step 4
# Steps 1-3 already applied (doFields x bool, doStreamFind sig).
# This script picks up from doStreamFind body onwards.
#
# Run with: powershell -ExecutionPolicy Bypass -File .\fedit_pr_v160_resume.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$f = "main.go"

function Check([string]$step) {
    if ($LASTEXITCODE -ne 0) { Write-Error "FAILED: $step"; exit 1 }
}

Write-Host ""
Write-Host "=== fedit v1.6.0 resume (steps 4-12) ===" -ForegroundColor Cyan
Write-Host ""

# Line-range ops applied bottom-to-top to keep earlier numbers stable.
# Current file: 2837 lines.

# -- R1. doStreamFind body (lines 2635-2660) -------------------------
# Replace comment + full function. Net +6 lines (26 -> 32).
Write-Host "[R1] doStreamFind: replace body with x branch + conditional summary"
fedit -file $f -op replace -line 2635 -end 2660 -textfile _p_dostreamfind_full.go
Check "doStreamFind body"

# -- R2. doFind: add x bool param ------------------------------------
Write-Host "[R2] doFind: signature"
fedit -file $f -op replaceall `
    -match 'func doFind(lines []string, match string, nth int) {' `
    -text  'func doFind(lines []string, match string, nth int, x bool) {'
Check "doFind sig"

# -- R3. doFind: insert -x short-circuit -----------------------------
Write-Host "[R3] doFind: -x short-circuit (bare line numbers to stdout)"
fedit -file $f -op insertbefore `
    -match 'width := len(strconv.Itoa(len(lines)))' `
    -textfile _p_dofind_xbranch.go
Check "doFind x branch"

# -- R4. Switch: thread *x to all three call sites -------------------
Write-Host "[R4] Switch: pass *x to doStreamFind, doFind, doFields"
fedit -file $f -op replaceall -match 'doStreamFind(*file, *match)'     -text 'doStreamFind(*file, *match, *x)'     ; Check "switch doStreamFind"
fedit -file $f -op replaceall -match 'doFind(lines, *match, *nth)'     -text 'doFind(lines, *match, *nth, *x)'     ; Check "switch doFind"
fedit -file $f -op replaceall -match 'doFields(*file, *col, delimStr)' -text 'doFields(*file, *col, delimStr, *x)' ; Check "switch doFields"

# -- R5. -v skip list: line 251 (unaffected by R1-R4) ---------------
Write-Host "[R5] -v skip list: add writeraw + writelines"
fedit -file $f -op replace -line 251 -end 251 -textfile _p_v_skip.go
Check "-v list"

# -- R6. Write handler: lines 79-99 (unaffected by R1-R5) -----------
Write-Host "[R6] write handler -> write | writeraw | writelines"
fedit -file $f -op replace -line 79 -end 99 -textfile _p_write_handler.go
Check "write handler"

# -- R7. -cleanfirst block before first var lines --------------------
Write-Host "[R7] -cleanfirst truncate block"
fedit -file $f -op insertbefore -match 'var lines []string' -nth 1 -textfile _p_cleanfirst.go
Check "-cleanfirst"

# -- R8. New flags + hex decode (replaces flag.Parse() line) ---------
Write-Host "[R8] -texthex/-cleanfirst/-x flags + hex decode"
fedit -file $f -op replaceall -match '		flag.Parse()' -textfile _p_flags.go
Check "flags + hex"

# -- R9. op description + usage text (line 22, unaffected by R1-R8) -
Write-Host "[R9] op description"
fedit -file $f -op replace -line 22 -end 22 -textfile _p_op_desc.go
Check "op desc"

Write-Host "[R10] usage text: writeraw + writelines"
fedit -file $f -op insertafter `
    -match '  write         Write text to -file (creates/overwrites)' `
    -textfile _p_usage_writelines.go
Check "usage write*"

Write-Host "[R11] usage text: new flags"
fedit -file $f -op insertbefore `
    -match 'Escapes in -text: \\n = newline' `
    -textfile _p_usage_newflags.go
Check "usage flags"

# -- R12. encoding/hex import ----------------------------------------
Write-Host "[R12] encoding/hex import"
fedit -file $f -op insertafter -match '"bufio"' -text "`t`"encoding/hex`""
Check "import"

# -- Build -----------------------------------------------------------
Write-Host ""
Write-Host "All patches applied. Building..." -ForegroundColor Yellow
go build -o fedit.exe .
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed - run: go vet ./..."; exit 1 }

Write-Host "Build OK - fedit.exe is now v1.6.0" -ForegroundColor Green
Write-Host ""
Write-Host "Smoke tests:" -ForegroundColor Cyan
Write-Host '  fedit -file _f2.txt -op writeraw  -text "hello\nworld"'
Write-Host '  fedit -file main.go -op find -match "func " -x 2>$null'
Write-Host '  fedit -file _f2.txt -op writelines'
