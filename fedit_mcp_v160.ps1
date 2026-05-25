# fedit_mcp_v160.ps1 - Patch mcp.go for fedit v1.6.0
# Adds: fedit_writeraw, cleanfirst on insert, x on find/fields, getBool helper
#
# Run with: powershell -ExecutionPolicy Bypass -File .\fedit_mcp_v160.ps1

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$f = "mcp.go"

function Check([string]$step) {
    if ($LASTEXITCODE -ne 0) { Write-Error "FAILED: $step"; exit 1 }
}

Write-Host ""
Write-Host "=== mcp.go v1.6.0 patch ===" -ForegroundColor Cyan
Write-Host ""

# Line-range ops applied BOTTOM-TO-TOP. mcp.go is 682 lines, untouched.

# -- 1. mcpDoFields: add x bool param (lines 645-682) -----------------------
Write-Host "[1/12] mcpDoFields: add x param + conditional stats"
fedit -file $f -op replace -line 645 -end 682 -textfile _mcp_dofields.go
Check "mcpDoFields"

# -- 2. mcpDoFind: add x bool param (lines 449-472) -------------------------
Write-Host "[2/12] mcpDoFind: add x param + bare-number branch"
fedit -file $f -op replace -line 449 -end 472 -textfile _mcp_dofind.go
Check "mcpDoFind"

# -- 3. mcpDoInsert: add cleanfirst param (lines 330-353) -------------------
Write-Host "[3/12] mcpDoInsert: add cleanfirst param"
fedit -file $f -op replace -line 330 -end 353 -textfile _mcp_doinsert.go
Check "mcpDoInsert"

# -- 4. Insert mcpDoWriteRaw after line 328 (end of mcpDoWrite) -------------
Write-Host "[4/12] Insert mcpDoWriteRaw function"
fedit -file $f -op insert -line 328 -textfile _mcp_writeraw_fn.go
Check "mcpDoWriteRaw"

# -- 5. Switch: fedit_fields case (original lines 205-206) ------------------
Write-Host "[5/12] Switch: fedit_fields -> pass getBool(x)"
fedit -file $f -op replace -line 205 -end 206 -textfile _mcp_switch_fields.go
Check "switch fields"

# -- 6. Switch: fedit_find case (original lines 189-190) --------------------
Write-Host "[6/12] Switch: fedit_find -> pass getBool(x)"
fedit -file $f -op replace -line 189 -end 190 -textfile _mcp_switch_find.go
Check "switch find"

# -- 7. Switch: fedit_write/writeraw/insert (original lines 176-179) --------
Write-Host "[7/12] Switch: add fedit_writeraw case, wire getBool(cleanfirst)"
fedit -file $f -op replace -line 176 -end 179 -textfile _mcp_switch_write.go
Check "switch write"

# -- 8. Insert getBool helper after line 167 (closing } of getInt) ----------
Write-Host "[8/12] Insert getBool helper"
fedit -file $f -op insert -line 167 -textfile _mcp_getbool.go
Check "getBool"

# -- 9. Tool def: fedit_fields (original line 143) --------------------------
Write-Host "[9/12] Tool def: fedit_fields + x param"
fedit -file $f -op replace -line 143 -end 143 -textfile _mcp_td_fields.go
Check "td fields"

# -- 10. Tool def: fedit_find (original line 138) ---------------------------
Write-Host "[10/12] Tool def: fedit_find + x param"
fedit -file $f -op replace -line 138 -end 138 -textfile _mcp_td_find.go
Check "td find"

# -- 11. Insert fedit_writeraw tool def after line 136 ----------------------
Write-Host "[11/12] Tool def: insert fedit_writeraw"
fedit -file $f -op insert -line 136 -textfile _mcp_td_writeraw.go
Check "td writeraw"

# -- 12. Tool def: fedit_insert + cleanfirst (original line 132) ------------
Write-Host "[12/12] Tool def: fedit_insert + cleanfirst param"
fedit -file $f -op replace -line 132 -end 132 -textfile _mcp_td_insert.go
Check "td insert"

# -- Version bump (content-based, safe last) --------------------------------
Write-Host "[+] Version bump 1.5.0 -> 1.6.0"
fedit -file $f -op replaceall -match 'Version: "1.5.0"' -text 'Version: "1.6.0"'
Check "version"

# -- Build -------------------------------------------------------------------
Write-Host ""
Write-Host "Building..." -ForegroundColor Yellow
go build -o fedit.exe .
if ($LASTEXITCODE -ne 0) { Write-Error "Build failed - run: go vet ./..."; exit 1 }

Write-Host "Build OK - mcp.go patched to v1.6.0" -ForegroundColor Green
Write-Host ""
Write-Host "New MCP tools/params:" -ForegroundColor Cyan
Write-Host "  fedit_writeraw          - raw write, no escape expansion"
Write-Host "  fedit_insert cleanfirst - truncate then insert"
Write-Host "  fedit_find x            - bare line numbers only"
Write-Host "  fedit_fields x          - suppress stats footer"
