# Amalex Brand -- Claude Working Preferences
# Upload this file alongside the project status file at the start of every chat.
# Applies to: fedit, amalex-handler, fwrite, website, and all Amalex Brand projects.
# Last updated: June 1, 2026

---

## Shell & Environment

  Shell:     PowerShell 7.6.2 (pwsh). Use ; not &&.
  OS:        Windows 11
  Base dir:  C:\Users\kehsi\Desktop\amalex-brand\
  Browser:   Firefox

---

## Response Conventions

  Every Claude response ends with: -- response complete --
  ONE patch block per response. Never show a reference block AND a run-this block.
  NEVER provide two patch blocks in one message even with DO-NOT-RUN labels.
  Provide ONE canonical block only per turn.

---

## PowerShell Rules (Critical)

  NEVER use @'...'@ or @"..."@ heredocs -- terminal pastes in reverse order.

  Multi-line content -- use -texthex directly (PREFERRED, no temp files):
    $h='<hex>'
    fedit -file target -op insertafter -match 'anchor' -texthex $h -v

  Multi-line content -- WriteAllBytes fallback (only when hex > ~32KB):
    $h='<hex>'
    [IO.File]::WriteAllBytes('C:\...\absolute\_patch.txt',
      [byte[]]($h -split '(..)' | ?{$_} | %{[convert]::ToByte($_,16)}))
    fedit -file target -op insertafter -match 'anchor' -textfile _patch.txt -v
    Remove-Item _patch.txt

  Single-line content -- WriteAllText with absolute path:
    [IO.File]::WriteAllText('C:\...\absolute\_patch.txt', "content here")

Multi-line content -- Notepad++ _patch.txt (PREFERRED when content has quotes/backticks):
    Open Notepad++, create _patch.txt, type exact content, save UTF-8 no BOM.
    fedit -file target -op insertafter -match 'anchor' -textfile _patch.txt -v
    Remove-Item _patch.txt

  Python replacement -- Notepad++ _fix.py (for substrings containing double-quotes):
    Open Notepad++, create _fix.py with c.replace() calls.
    python3 _fix.py ; Remove-Item _fix.py

  Inline fwencode via $() subexpression:
    fedit -file f -op insertafter -match 'anchor' -texthex $(fwencode "content") -v
    WARNING: $$ in double-quoted PS strings expands to process ID -- avoid.
    SAFE:    $h = fwencode "content with $ signs" ; fedit ... -texthex $h
  fwencode USAGE RULES:
    WORKS:   $h = fwencode "single line"
    WORKS:   $h = fwencode "with `"double quotes`" escaped"
    WORKS:   $h = fwencode "line one`nline two"   -- PS expands `n to newline before fwencode sees it
    BROKEN:  $h = fwencode "line one\nline two"   -- \n is literal backslash-n, NOT a newline
    CORRECT for complex multi-line: Notepad++ _patch.txt + -textfile (avoids all escaping)
  NEVER use Set-Content.
  NEVER chain write + fedit in the same command block (separate commands).
  Double-quotes in -text/-match: use -texthex (see fwencode section below).
  .ps1 scripts must be 100% ASCII -- no em-dashes, box-drawing chars, or arrows.

  Go struct tags (backticks in PS):
    $bt = [string][char]96
    $c  = [IO.File]::ReadAllText('C:\...\file.go')
    $c  = $c.Replace("BKTICK", $bt)
    [IO.File]::WriteAllText('C:\...\file.go', $c)

  Git commit message with special chars:
    [IO.File]::WriteAllText("$PWD\_msg.txt", "feat: my commit message")
    git commit -F _msg.txt ; git push ; Remove-Item _msg.txt

  PS execution policy for .ps1 scripts:
    powershell -ExecutionPolicy Bypass -File .\script.ps1

---

## fwencode / fwdecode

  Repo:    C:\Users\kehsi\Desktop\amalex-brand\fwrite\ (PRIVATE)
  Binary:  dist\fwencode.exe + dist\fwdecode.exe (both on PATH)
  Purpose: fwencode = UTF-8 text -> hex string (for fedit -texthex)
           fwdecode = hex string -> UTF-8 text
           BOM strip: fwencode strips UTF-8 BOM from stdin automatically
                      (PS 7 adds BOM when piping -- this neutralizes it)

  fwencode.exe: OK -- rebuilt May 30, 2026 (BOM fix live)
     Build: cd C:\Users\kehsi\Desktop\amalex-brand\fwrite\apps\fwencode
     go build -o ..\..\dist\fwencode.exe .

  Usage:
    $hex = fwencode "content with `"quotes`" and \backslashes\"
    fedit -file f.go -op insertbefore -match "func Foo" -texthex $hex

    # Pipe show -raw output (block-to-block transfer):
    $hex = fedit -file src.go -op show -block "FuncName" -lang go -raw 2>$null | fwencode
    fedit -file dst.go -op replace -block "OldFunc" -lang go -texthex $hex

---

## fedit Rules

  Binary:  fedit.exe is on PATH -- call as plain `fedit` from any directory.
           NEVER use `.\fedit.exe` or `$f = ".\fedit.exe"` (obsolete).
  Flag:    ALWAYS use -v on every mutation.
  Recon:   ALWAYS run fedit -op find or fedit -op show BEFORE any mutation.
  Order:   PREFER insertafter/insertbefore over line-numbered insert.
           PREFER -block/-lang when targeting named functions/classes/resources.
  Blocks:  insertafter on "func Foo" inserts INSIDE the body (matches opening line).
           To insert AFTER the function, use insertbefore on the NEXT function.
  Anchor:  NEVER use insertbefore on headings (## N.) or separators (---) when
           content precedes them. A --- sits between content and heading.
           insertbefore '## 6.' puts content AFTER the ---, outside the section.
           CORRECT: insertafter 'last content line in section'
  Seq:     When mixing delete/replace-line WITH insertafter/insertbefore in one
           sequence: run the LINE-NUMBER ops FIRST. Insertions shift all
           subsequent line numbers -- delete -line N hits the wrong line after.
  Format:  gofmt on specific files only -- never on directories.
  NOTE:    replace and delete support -match/-endmatch for content-anchored ranges (v1.8.0).
           Anchor replace:  -op replace -match "start" -endmatch "end" -textfile f.txt -v
           Anchor delete:   -op delete -match "start" -endmatch "end" -v
           Substring replace only: replaceall -match "old" -text "new"
           Single-line match-delete: $n = [int](fedit -op find -match X -x) ; fedit -op delete -line $n -v
           PS -x CAST RULE: fedit -x returns STRING -- wrap in [int]() before arithmetic.
             WRONG:   $n = fedit ... -x  →  ($n+3) concatenates: "2443" not 247
             CORRECT: $n = [int](fedit ... -x)  →  ($n+3) = 247
           Paste-safe pattern: one $var = cmd ; cmd per line -- reversal does not break same-line chains.
           BRACE NESTING RISK: replacing } with } else if -- verify indentation level in verify block.
           gofmt nests else if inside inner block if patch indentation places it at wrong brace depth.


  Operations (15):
    show, find, map, insert, insertafter, insertbefore,
    replace, replaceall, delete, write, writeraw, writelines,
    move, copy, fields

  MCP tools (14 -- writelines is interactive stdin only, no MCP tool):
    fedit_show, fedit_find, fedit_map, fedit_insert,
    fedit_insertafter, fedit_insertbefore, fedit_replace,
    fedit_replaceall, fedit_delete, fedit_write, fedit_writeraw,
    fedit_move, fedit_copy, fedit_fields

  Key flags:
    -block NAME + -lang LANG   target named block (no line numbers needed)
    -raw                       show: bare content for piping to fwencode
    -texthex HEX               hex-encoded content (bypasses PS quoting)
    -extract SPEC              sub-line extraction (see hierarchy below)
    -get REGEX                 pre-filter line before -extract
    -wdelim CHAR               word delimiter for -extract (default: whitespace)
    -x                         find: bare line numbers / fields: no stats
    -cleanfirst                truncate file before insert/write
    -stream                    large file mode (replaceall + find)
    -v                         verify after every mutation
    -nth N                     occurrence selector (default 1, -1 = last)
    -line N:+M                 colon range: N to N+M (M additional lines)
    -line -N                   N-th line from end of file
    -line -N:                  last N lines (from -N to EOF)
    -line :                    EOF (append for insert; last line for show/delete/replace)
    -end -N                    end line relative to EOF
    -endmatch TEXT             content-anchor end of range (show, replace, delete, move, copy)
    -quiet                     suppress stdout on success; exit code signals result

  Extract hierarchy (File -> Line -> Word -> Char):
    WN           word N (normalized whitespace, 1-based)
    WN[s:c]      word N, chars s through s+c-1 (1-based start, count c)
    WN[s:]       word N, from char s to end
    WN/DELIM/F   word N, split by DELIM, take field F

  write vs writeraw:
    fedit_write    -- processes escape sequences (\n \t \\)
    fedit_writeraw -- backslashes are literal (use for Go regex, Windows paths)

---

## Code Conventions (all projects)

  All IDs are TEXT (UUID) -- never auto-increment integers.
  Logging: package-level `log` (zerolog), NOT `s.log`.
  SQLC: always run `sqlc generate` from project root after changing .sql files.
  render.go has a HARDCODED pages list -- new pages must be added manually.
  CSRF: field "csrf_token", cookie "amalex_csrf", header X-CSRF-Token.
  TERMINOLOGY: UI says "Jobs", DB/code says "pairs".
  Build: `go build -o <name>.exe ./cmd/<name>` -- never `go run`.
  audit_log table uses INTEGER PRIMARY KEY AUTOINCREMENT (not TEXT UUID).

---

## Project Paths & Repos

  fedit           C:\Users\kehsi\Desktop\amalex-brand\fedit\
                  github.com/amalexico/fedit (PUBLIC -- MIT)
                  Install: go install github.com/amalexico/fedit@latest

  amalex-handler  C:\Users\kehsi\Desktop\amalex-brand\amalex-handler\
                  github.com/amalexico/amalex-handler (PRIVATE)
                  Build: go build -o amalex.exe ./cmd/amalex
                  Run:   .\amalex.exe serve

  fwrite          C:\Users\kehsi\Desktop\amalex-brand\fwrite\ (PRIVATE)
                  Apps: fwencode.exe, fwdecode.exe (both on PATH)

  website         C:\Users\kehsi\Desktop\amalex-brand\website\
                  Deploy: cd website ; .\deploy.ps1
                  Host: Cloudflare Pages (project: icy-glade-1626)

  status file     C:\Users\kehsi\Desktop\amalex-brand\Amalex_handler_current_status.txt

---

## Release Workflows

  fedit release:
    cd fedit
    git add -A ; git commit -m "message"
    git tag vX.Y.Z ; git push ; git push --tags
    (Go module proxy handles distribution -- no GoReleaser needed)

  amalex-handler release:
    git tag vX.Y.Z ; git push ; git push --tags
    docker login -u amalexico
    $env:GITHUB_TOKEN = "ghp_..."
    goreleaser release --clean
    Copy dist\* to website\downloads\
    Update version.json + index.html
    cd website ; .\deploy.ps1

  website deploy:
    cd C:\Users\kehsi\Desktop\amalex-brand\website
    .\deploy.ps1

---
