# fedit_pr_v160.ps1 — Apply fedit v1.6.0 PR
# New: -texthex  |  -op writeraw  |  -op writelines  |  -cleanfirst  |  -x
#
# Run from: C:\Users\kehsi\Desktop\amalex-brand\fedit
# Drop the _p_*.go snippet files into the same directory, then run this script.
# Requires: fedit.exe on PATH

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

# =================================================================
# Line-range ops are applied BOTTOM-TO-TOP so earlier line numbers
# stay valid.  Content-based ops (replaceall/insertafter/before)
# are order-independent and can go anywhere.
# =================================================================

# ── 1. doFields: add x bool param ───────────────────────────────
Write-Host "[1/13] doFields: signature"
fedit -file $f -op replaceall `
    -match 'func doFields(path string, col int, delim string) {' `
    -text  'func doFields(path string, col int, delim string, x bool) {'
Check "doFields sig"

# ── 2. doFields: conditional stderr (lines 2697-2698 in original)
Write-Host "[2/13] doFields: suppress stderr with -x"
fedit -file $f -op replace -line 2697 -end 2698 -textfile _p_dofields_stderr.go
Check "doFields stderr"

# ── 3. doStreamFind: add x bool param ───────────────────────────
Write-Host "[3/13] doStreamFind: signature"
fedit -file $f -op replaceall `
    -match 'func doStreamFind(path, search string) {' `
    -text  'func doStreamFind(path, search string, x bool) {'
Check "doStreamFind sig"

# ── 4. doStreamFind: machine-readable Printf branch ─────────────
Write-Host "[4/13] doStreamFind: -x output branch"
fedit -file $f -op replaceall `
    -match 'fmt.Printf("%d: %s\n", lineNum, line)' `
    -textfile _p_dostreamfind_printf.go
Check "doStreamFind printf"

# ── 5. doStreamFind: suppress summary when -x ───────────────────
Write-Host "[5/13] doStreamFind: conditional summary"
fedit -file $f -op replaceall `
    -match 'fmt.Fprintf(os.Stderr, "Found %d match(es) across %d lines\n", count, lineNum)' `
    -textfile _p_dostreamfind_summary.go
Check "doStreamFind summary"

# ── 6. doFind: add x bool param ─────────────────────────────────
Write-Host "[6/13] doFind: signature"
fedit -file $f -op replaceall `
    -match 'func doFind(lines []string, match string, nth int) {' `
    -text  'func doFind(lines []string, match string, nth int, x bool) {'
Check "doFind sig"

# ── 7. doFind: insert -x short-circuit before context block ─────
Write-Host "[7/13] doFind: -x short-circuit (bare line numbers)"
fedit -file $f -op insertbefore `
    -match 'width := len(strconv.Itoa(len(lines)))' `
    -textfile _p_dofind_xbranch.go
Check "doFind x branch"

# ── 8. Switch: wire *x through to all three functions ───────────
Write-Host "[8/13] Switch: thread *x to doStreamFind, doFind, doFields"
fedit -file $f -op replaceall -match 'doStreamFind(*file, *match)'     -text 'doStreamFind(*file, *match, *x)'     ; Check "switch doStreamFind"
fedit -file $f -op replaceall -match 'doFind(lines, *match, *nth)'     -text 'doFind(lines, *match, *nth, *x)'     ; Check "switch doFind"
fedit -file $f -op replaceall -match 'doFields(*file, *col, delimStr)' -text 'doFields(*file, *col, delimStr, *x)' ; Check "switch doFields"

# ── 9. -v skip list: add writeraw + writelines ──────────────────
Write-Host "[9/13] -v skip list"
fedit -file $f -op replaceall `
    -match 'case "show", "map", "find", "write", "move", "copy":' `
    -text  'case "show", "map", "find", "write", "writeraw", "writelines", "move", "copy":'
Check "-v list"

# ── 10. Expand write handler (original lines 79-99) ─────────────
Write-Host "[10/13] write handler -> write | writeraw | writelines"
fedit -file $f -op replace -line 79 -end 99 -textfile _p_write_handler.go
Check "write handler"

# ── 11. -cleanfirst block before first 'var lines []string' ─────
Write-Host "[11/13] -cleanfirst truncate block"
fedit -file $f -op insertbefore -match 'var lines []string' -nth 1 -textfile _p_cleanfirst.go
Check "-cleanfirst"

# ── 12. New flags + -texthex decode (replaces flag.Parse() line) ─
Write-Host "[12/13] -texthex/-cleanfirst/-x flags + hex decode block"
fedit -file $f -op replaceall -match '		flag.Parse()' -textfile _p_flags.go
Check "flags + hex"

# ── 13. op description + usage text + encoding/hex import ────────
Write-Host "[13/13] op description, usage help, import"

fedit -file $f -op replaceall `
    -match '"Operation: insert, delete, replace, replaceall, show, write, map, find, insertafter, insertbefore, move, copy, fields"' `
    -text  '"Operation: insert, delete, replace, replaceall, show, write, writeraw, writelines, map, find, insertafter, insertbefore, move, copy, fields"'
Check "op desc"

fedit -file $f -op insertafter `
    -match '  write         Write text to -file (creates/overwrites)' `
    -text '		fmt.Fprintln(os.Stderr, "  writeraw      Write -text raw (no escape expansion; backslashes are literal)")\n		fmt.Fprintln(os.Stderr, "  writelines    Write lines interactively from stdin (Ctrl+Z/D to finish)")'
Check "usage write*"

fedit -file $f -op insertbefore `
    -match 'Escapes in -text: \\n = newline' `
    -text '		fmt.Fprintln(os.Stderr, "  -texthex      Treat -text as hex from fwencode (bypasses PS quoting entirely)")\n		fmt.Fprintln(os.Stderr, "  -cleanfirst   Truncate -file before writing (pair with insert for clean overwrite)")\n		fmt.Fprintln(os.Stderr, "  -x            Machine-readable output: bare line numbers / counts, no labels")\n		fmt.Fprintln(os.Stderr, "")'
Check "usage flags"

fedit -file $f -op insertafter -match '"bufio"' -text "`t`"encoding/hex`""
Check "import"

# ── Cleanup + Build ──────────────────────────────────────────────
Write-Host ""
Write-Host "All patches applied. Building..." -ForegroundColor Yellow
go build -o fedit.exe .
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed — run: go vet ./..."; exit 1 }

Write-Host "Build OK — fedit.exe updated to v1.6.0" -ForegroundColor Green
Write-Host ""
Write-Host "Smoke tests:" -ForegroundColor Cyan
Write-Host '  fedit -file _f2.txt -op writeraw  -text "hello\nworld"       # backslash is literal'
Write-Host '  fedit -file main.go -op find -match "func " -x 2>$null       # bare line numbers'
Write-Host '  fedit -file _f2.txt -op writelines                            # interactive stdin'
Write-Host '  fedit -file _f2.txt -op insert -line 0 -cleanfirst -text "x" # truncate first'
