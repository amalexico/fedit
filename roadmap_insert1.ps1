$s = "C:\Users\kehsi\Desktop\amalex-brand\Amalex_handler_current_status.txt"

# Step 1: Write roadmap file using Add-Content (no heredoc = no reversal)
$r = "$PSScriptRoot\_roadmap.txt"
Remove-Item $r -ErrorAction SilentlyContinue

Add-Content $r "  FEDIT UNIVERSAL TOOL ROADMAP (agreed May 7, 2026):"
Add-Content $r "  Vision: fedit as the universal file manipulation primitive for LLM agents."
Add-Content $r "  Closes the last functional gaps vs sed/awk with fedit readable syntax."
Add-Content $r ""
Add-Content $r "  v1.3.0 — Regex + Multi-file (closes sed gap)"
Add-Content $r "    -match-regex PATTERN    Regex with capture groups in replaceall/replace"
Add-Content $r '    -text "[$1] $2"         Reference captures in replacement text'
Add-Content $r '    -files "*.go"           Apply operation across a file glob (atomic per file)'
Add-Content $r "    Literal -match stays default; -match-regex is explicit opt-in"
Add-Content $r "    Example:"
Add-Content $r '      fedit -op replaceall -match-regex "v(\d+\.\d+)" -text "v[$1]" -file CHANGELOG.md'
Add-Content $r '      fedit -files "*.go" -op replaceall -match "OldName" -text "NewName"'
Add-Content $r ""
Add-Content $r "  v1.4.0 — Fields + Stream (closes awk gap)"
Add-Content $r "    -op fields              Extract column N from delimited files (CSV, TSV)"
Add-Content $r "    -col N                  Which column (1-based)"
Add-Content $r "    -delim CHAR             Field delimiter (default tab; comma for CSV)"
Add-Content $r "    -stream                 Large file mode (no full load into memory)"
Add-Content $r "    Example:"
Add-Content $r '      fedit -file data.csv -op fields -col 2 -delim ","'
Add-Content $r ""
Add-Content $r "  WHAT STAYS OUT OF SCOPE:"
Add-Content $r "    Arithmetic/computation — awk sum/avg stays in awk territory"
Add-Content $r "    fedit is a file editor, not a data processor"
Add-Content $r ""

# Step 2: Insert into status file
Start-Sleep -Seconds 1
fedit -file $s -op insertafter -match STRATEGIC

# Step 3: Also add -afterblock flag if not already present
$content = [IO.File]::ReadAllText($s)
if ($content -notmatch afterblock
    fedit -file $s -op insertafter -match -beforeblock
}

# Step 4: Cleanup
Remove-Item $r -ErrorAction SilentlyContinue

Write-Host Done.
