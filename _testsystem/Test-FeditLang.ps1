#requires -Version 7
<#
  Test-FeditLang.ps1
  ---------------------------------------------------------------
  Phase 1 of the fedit testing system: the -lang / map feature.
  Runs fedit -op map -lang <lang> against every fixture file for
  every supported language, single file at a time.

  WHY PER-FILE, NOT -file *.ext:
  fedit's -file flag takes exactly one path (flag.String). There
  is no glob/multi-file support in "map" -- only "replaceall" has
  -files. PowerShell also does not expand *.ext for a native exe
  the way bash does, so "-file *.go" would just fail to open a
  literal file named "*.go". "Multiple files" here means: this
  script enumerates every file in each language's fixture folder
  and calls fedit once per file. Each call already prints its own
  "=== MAP: <path> (...) ===" header, so looping naturally gives
  you per-file-titled output with zero fedit changes needed.

  BEFORE / AFTER:
  map is read-only, so "before" is the raw file (fedit -op show),
  "after" is the map output. True mutation before/after diffing
  (insert/replace/delete) is phase 2, per the conversation plan.

  USAGE:
    .\Test-FeditLang.ps1                      # run everything
    .\Test-FeditLang.ps1 -Lang go,python       # just these languages
    .\Test-FeditLang.ps1 -ShowOutput           # print full map output too
    .\Test-FeditLang.ps1 -FixturesPath C:\...\fixtures

  EXIT CODE: 0 if every check passed, 1 if any failed.
#>

param(
    [string[]]$Lang,
    [switch]$ShowOutput,
    [string]$FeditExe = "fedit",
    [string]$FixturesPath = (Join-Path $PSScriptRoot "fixtures")
)

# -- Language config ----------------------------------------------
# CleanMarker  = text fedit prints when the mapper finds NO duplicates
# DupMarker    = text fedit prints in the ! warning when it DOES find one
# Either can be $null -- some mappers (markdown, dockerfile) have no
# duplicate-detection section at all; verified against the real binary.
$LangConfig = [ordered]@{
    go         = @{ CleanMarker = "No duplicate functions";              DupMarker = "DUPLICATE FUNCTIONS" }
    html       = @{ CleanMarker = "No duplicate JS functions";           DupMarker = "DUPLICATE JS FUNCTIONS" }
    sql        = @{ CleanMarker = "No duplicate queries";                DupMarker = "DUPLICATE QUERIES" }
    python     = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    javascript = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    typescript = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    css        = @{ CleanMarker = "No duplicate selectors";              DupMarker = "Duplicates:" }
    rust       = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    java       = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    csharp     = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    yaml       = @{ CleanMarker = "No duplicate top-level keys";         DupMarker = "Duplicates:" }
    toml       = @{ CleanMarker = "No duplicate tables";                 DupMarker = "Duplicate tables:" }
    markdown   = @{ CleanMarker = $null;                                 DupMarker = $null }  # no dup check exists
    ruby       = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    php        = @{ CleanMarker = "No duplicate definitions";            DupMarker = "Duplicates:" }
    dockerfile = @{ CleanMarker = $null;                                 DupMarker = $null }  # no dup check exists
    makefile   = @{ CleanMarker = "No duplicate targets";                DupMarker = "Duplicates:" }
    json       = @{ CleanMarker = "No duplicate keys within any object"; DupMarker = "DUPLICATE KEYS" }
    csv        = @{ CleanMarker = "No duplicate header columns";         DupMarker = "DUPLICATE HEADER COLUMNS" }
}

$languagesToRun = if ($Lang) { $Lang } else { $LangConfig.Keys }

$results = [System.Collections.Generic.List[object]]::new()

function Write-Result {
    param($Status, $Message)
    $color = switch ($Status) {
        "PASS" { "Green" }
        "FAIL" { "Red" }
        "SKIP" { "DarkYellow" }
    }
    Write-Host ("  {0,-4}  {1}" -f $Status, $Message) -ForegroundColor $color
}

foreach ($lang in $languagesToRun) {
    if ($LangConfig.Keys -notcontains $lang) {
        Write-Warning "Unknown language '$lang' -- skipping"
        continue
    }
    $cfg = $LangConfig[$lang]
    $langDir = Join-Path $FixturesPath $lang
    if (-not (Test-Path $langDir)) {
        Write-Warning "No fixtures directory for '$lang' at $langDir -- skipping"
        continue
    }

    Write-Host "=== $lang ===" -ForegroundColor Cyan

    # Multi-file loop: every fixture for this language, called individually,
    # since map has no native glob/multi-file mode (see header comment).
    $files = Get-ChildItem -Path $langDir -File
    foreach ($file in $files) {
        # "before": raw content (proves the file exists / is readable before we mutate nothing)
        $before = & $FeditExe -file $file.FullName -op show -raw 2>$null

        # "after": the actual map output under test
        $after = & $FeditExe -file $file.FullName -op map -lang $lang 2>$null
        $exitCode = $LASTEXITCODE
        $afterText = $after -join "`n"

        if ($ShowOutput) {
            Write-Host $afterText -ForegroundColor Gray
        }

        if ($exitCode -ne 0) {
            Write-Result "FAIL" "$($file.Name) -- fedit exited $exitCode (expected 0)"
            $results.Add([pscustomobject]@{ Lang=$lang; File=$file.Name; Status="FAIL"; Detail="exit code $exitCode" })
            continue
        }

        $isDupFixture = $file.Name -like "dup_*"
        $expectedMarker = if ($isDupFixture) { $cfg.DupMarker } else { $cfg.CleanMarker }
        $expectLabel = if ($isDupFixture) { "duplicate warning" } else { "clean (no duplicates)" }

        if ($null -eq $expectedMarker) {
            Write-Result "SKIP" "$($file.Name) -> $expectLabel (mapper has no duplicate-detection section)"
            $results.Add([pscustomobject]@{ Lang=$lang; File=$file.Name; Status="SKIP"; Detail="no dup check in mapper" })
            continue
        }

        if ($afterText -like "*$expectedMarker*") {
            Write-Result "PASS" "$($file.Name) -> $expectLabel"
            $results.Add([pscustomobject]@{ Lang=$lang; File=$file.Name; Status="PASS"; Detail="" })
        } else {
            Write-Result "FAIL" "$($file.Name) -> expected '$expectedMarker' in output, not found"
            $results.Add([pscustomobject]@{ Lang=$lang; File=$file.Name; Status="FAIL"; Detail="missing marker: $expectedMarker" })
        }

        # sanity: "before" should never be empty for a fixture that exists
        if (-not $before) {
            Write-Result "FAIL" "$($file.Name) -- raw show returned nothing (fixture empty or unreadable?)"
            $results.Add([pscustomobject]@{ Lang=$lang; File=$file.Name; Status="FAIL"; Detail="empty before-state" })
        }
    }
}

$pass = ($results | Where-Object Status -eq "PASS").Count
$fail = ($results | Where-Object Status -eq "FAIL").Count
$skip = ($results | Where-Object Status -eq "SKIP").Count

Write-Host ""
Write-Host "===================================" -ForegroundColor White
Write-Host "TOTAL: $pass passed, $fail failed, $skip skipped" -ForegroundColor White
Write-Host "===================================" -ForegroundColor White

if ($fail -gt 0) {
    Write-Host "`nFailures:" -ForegroundColor Red
    $results | Where-Object Status -eq "FAIL" | Format-Table Lang, File, Detail -AutoSize
    exit 1
}
exit 0
