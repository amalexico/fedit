# fedit_pr_v160.ps1 - Apply fedit v1.6.0 PR
# New: -texthex | -op writeraw | -op writelines | -cleanfirst | -x
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit
# Drop ALL _p_*.go files in the same folder before running.
# Requires: fedit.exe on PATH
#
# Run with: powershell -ExecutionPolicy Bypass -File .\fedit_pr_v160.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$f = "main.go"

function Check([string]$step) {
    if ($LASTEXITCODE -ne 0) { Write-Error "FAILED: $step"; exit 1 }
}

Write-Host ""
Write-Host "=== fedit v1.6.0 PR ===" -ForegroundColor Cyan
Write-Host "  -texthex | -op writeraw | -op writelines | -cleanfirst | -x"
Write-Host ""

# ==================================================================
# Line-range ops applied BOTTOM-TO-TOP so earlier numbers stay valid.
# replaceall / insertafter / insertbefore are content-based (any order).
# ALL double-quote-in-arg problems solved via -textfile snippets.
# ==================================================================

# -- 1. doFields: add x bool param -----------------------------------
Write-Host "[1/12] doFields: signature"
fedit -file $f -op replaceall `
    -match 'func doFields(path string, col int, delim string) {' `
    -text  'func doFields(path string, col int, delim string, x bool) {'
Check "doFields sig"

# -- 2. doFields: conditional stderr (original lines 2697-2698) ------
Write-Host "[2/12] doFields: suppress stderr with -x"
fedit -file $f -op replace -line 2697 -end 2698 -textfile _p_dofields_stderr.go
Check "doFields stderr"

# -- 3. doStreamFind: replace ENTIRE function (original lines 2636-2659)
# After step 2: +2 lines -> doStreamFind now at 2638-2661
Write-Host "[3/12] doStreamFind: full replacement (sig + x branch + conditional summary)"
fedit -file $f -op replace -line 2638 -end 2661 -textfile _p_dostreamfind_full.go
Check "doStreamFind full"

# -- 4. doFind: add x bool param -------------------------------------
Write-Host "[4/12] doFind: signature"
fedit -file $f -op replaceall `
    -match 'func doFind(lines []string, match string, nth int) {' `
    -text  'func doFind(lines []string, match string, nth int, x bool) {'
Check "doFind sig"

# -- 5. doFind: insert -x short-circuit before context block ---------
Write-Host "[5/12] doFind: -x short-circuit (bare line numbers to stdout)"
fedit -file $f -op insertbefore `
    -match 'width := len(strconv.Itoa(len(lines)))' `
    -textfile _p_dofind_xbranch.go
Check "doFind x branch"

# -- 6. Switch: thread *x to all three call sites --------------------
Write-Host "[6/12] Switch: pass *x to doStreamFind, doFind, doFields"
fedit -file $f -op replaceall -match 'doStreamFind(*file, *match)'     -text 'doStreamFind(*file, *match, *x)'     ; Check "switch doStreamFind"
fedit -file $f -op replaceall -match 'doFind(lines, *match, *nth)'     -text 'doFind(lines, *match, *nth, *x)'     ; Check "switch doFind"
fedit -file $f -op replaceall -match 'doFields(*file, *col, delimStr)' -text 'doFields(*file, *col, delimStr, *x)' ; Check "switch doFields"

# -- 7. -v skip list: original line 251, unaffected by steps 1-6 ----
Write-Host "[7/12] -v skip list: add writeraw + writelines"
fedit -file $f -op replace -line 251 -end 251 -textfile _p_v_skip.go
Check "-v list"

# -- 8. Expand write handler (original lines 79-99) ------------------
Write-Host "[8/12] write handler -> write | writeraw | writelines"
fedit -file $f -op replace -line 79 -end 99 -textfile _p_write_handler.go
Check "write handler"

# -- 9. -cleanfirst block before first 'var lines []string' ----------
Write-Host "[9/12] -cleanfirst truncate block"
fedit -file $f -op insertbefore -match 'var lines []string' -nth 1 -textfile _p_cleanfirst.go
Check "-cleanfirst"

# -- 10. New flags + hex decode (replaces the flag.Parse() line) -----
Write-Host "[10/12] -texthex/-cleanfirst/-x flags + hex decode"
fedit -file $f -op replaceall -match '		flag.Parse()' -textfile _p_flags.go
Check "flags + hex"

# -- 11. op description: original line 22, unaffected by steps 1-10 -
Write-Host "[11/12] op description + usage text"
fedit -file $f -op replace -line 22 -end 22 -textfile _p_op_desc.go
Check "op desc"

fedit -file $f -op insertafter `
    -match '  write         Write text to -file (creates/overwrites)' `
    -textfile _p_usage_writelines.go
Check "usage write*"

fedit -file $f -op insertbefore `
    -match 'Escapes in -text: \\n = newline' `
    -textfile _p_usage_newflags.go
Check "usage flags"

# -- 12. encoding/hex import -----------------------------------------
Write-Host "[12/12] encoding/hex import"
fedit -file $f -op insertafter -match '"bufio"' -text "`t`"encoding/hex`""
Check "import"

# -- Cleanup + Build -------------------------------------------------
Write-Host ""
Write-Host "All patches applied. Building..." -ForegroundColor Yellow
go build -o fedit.exe .
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed - run: go vet ./..."; exit 1 }

Write-Host "Build OK - fedit.exe updated to v1.6.0" -ForegroundColor Green
Write-Host ""
Write-Host "Smoke tests:" -ForegroundColor Cyan
Write-Host '  fedit -file _f2.txt -op writeraw  -text "hello\nworld"       # backslash is literal'
Write-Host '  fedit -file main.go -op find -match "func " -x 2>$null       # bare line numbers'
Write-Host '  fedit -file _f2.txt -op writelines                            # interactive stdin'
Write-Host '  fedit -file _f2.txt -op insert -line 0 -cleanfirst -text "x" # truncate first'
