# add_fedit_to_goose.ps1
# Registers fedit as a goose MCP extension by editing goose's config.yaml.
# Safe to re-run: skips if fedit is already registered.

$ErrorActionPreference = "Stop"

# --- 1. Sanity check: is fedit actually on PATH? ---
$feditCmd = Get-Command fedit -ErrorAction SilentlyContinue
if (-not $feditCmd) {
    Write-Host "WARNING: 'fedit' was not found on PATH." -ForegroundColor Yellow
    Write-Host "goose will fail to launch the extension until fedit.exe is reachable as 'fedit' from any directory."
} else {
    Write-Host "Found fedit at: $($feditCmd.Source)"
}

# --- 2. Locate goose's config directory (Windows) ---
$configDir  = Join-Path $env:APPDATA "Block\goose\config"
$configPath = Join-Path $configDir "config.yaml"

if (-not (Test-Path $configDir)) {
    New-Item -ItemType Directory -Path $configDir -Force | Out-Null
    Write-Host "Created goose config directory: $configDir"
}

# --- 3. Build the fedit extension block (as a line array -- no heredocs) ---
$feditLines = @(
    '  fedit:',
    '    name: fedit',
    '    type: stdio',
    '    cmd: fedit',
    '    args: ["-mcp"]',
    '    enabled: true',
    '    timeout: 300'
)
$feditBlock = $feditLines -join "`n"

# --- 4. Create or patch config.yaml ---
if (-not (Test-Path $configPath)) {
    # No config file at all yet -- create one with just the fedit extension.
    $newContent = "extensions:`n$feditBlock`n"
    [IO.File]::WriteAllText($configPath, $newContent)
    Write-Host "Created new config.yaml with fedit extension:"
    Write-Host "  $configPath"
}
else {
    $text = [IO.File]::ReadAllText($configPath)

    if ($text -match '(?m)^\s*fedit:\s*$') {
        Write-Host "fedit is already registered in $configPath -- no changes made."
    }
    elseif ($text -match '(?m)^extensions:\s*$') {
        # extensions: block exists -- insert fedit right under it.
        $newText = $text -replace '(?m)^(extensions:\s*)$', "`$1`n$feditBlock"
        [IO.File]::WriteAllText($configPath, $newText)
        Write-Host "Added fedit to the existing extensions block in:"
        Write-Host "  $configPath"
    }
    else {
        # No extensions: block at all -- append one at the end of the file.
        $newText = $text.TrimEnd() + "`n`nextensions:`n$feditBlock`n"
        [IO.File]::WriteAllText($configPath, $newText)
        Write-Host "Appended a new extensions block with fedit to:"
        Write-Host "  $configPath"
    }
}

Write-Host ""
Write-Host "Done. To verify:" -ForegroundColor Green
Write-Host "  1. Run: goose session"
Write-Host "  2. Type: /tools"
Write-Host "  3. Confirm the fedit_* tools appear in the list"
