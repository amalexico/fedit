# Amalex Brand -- Claude Working Preferences
# Upload this file alongside the project status file at the start of every chat.
# Applies to: fedit, amalex-handler, fwrite, website, and all Amalex Brand projects.
# Last updated: September 28, 2026

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
  When sending patch-file content, put the command that applies it (plus the verify chain) in the SAME response, right after the content. Never split them across two turns. Recon commands come in their own earlier turn.

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
    Open Notepad++, create _patch.txt, save UTF-8 no BOM.
    NOTE: if Notepad++ accidentally saves with BOM, fedit strips it from the first line.
    fedit -file target -op insert -line N -textfile _patch.txt -v
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
    BROKEN:  $h = fwencode "C:\...\builtins\"              -- a fwencode arg ENDING in a backslash right before the closing quote collapses via Windows argv-escaping into a literal embedded quote, silently dropping the backslash (hit July 17, 2026 on a trailing-backslash path arg -- "...builtins\" arrived as "...builtins" with a stray quote, backslash gone)
    FIX for content ending in a backslash: route through Notepad++ _patch.txt + -textfile instead of fwencode.
  NEVER use Set-Content.
  NEVER put a literal tab character in a fedit -text value on the command line -- PowerShell's console interprets a mid-line tab as a tab-completion trigger, not a literal character. Symptom: the command hangs or the -text value gets replaced/garbled with a tab-completed filename, requiring Ctrl+C. Fix: use a space for inline indentation instead (gofmt -w normalizes it to a real tab afterward). Reserve literal tabs/complex whitespace for -textfile or -texthex content only -- never -text. AFTER any fedit command that hangs or produces garbled output, re-run fedit -op find on the intended anchor before assuming a retry is the only change -- an interrupted command may have already partially or fully executed (caused a duplicate reg.Register line in fwrite's main.go on July 16, 2026).
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

Patch file location & naming:
  ALWAYS create patch files in the CURRENT WORKING DIRECTORY (apps/fwrite),
  never in a subfolder like internal/builtins/ -- keeps cleanup to one folder.
  Cycling numbering (max 10 temp files): _patch.txt, _patch1.txt ... _patch9.txt, then wrap to _patch.txt and overwrite.
  Do NOT Remove-Item after each command -- user runs a batch cleanup script
  at end of session instead.
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
  Range:   Before any -line N:+M or line-number range for a replace/
           delete spanning to EOF or a known endpoint, get a FRESH
           -op show -line N: (open-ended) or explicit total-line-count
           check immediately before computing the range -- don't reuse
           a line count from earlier in the conversation/session, even
           a few turns back. Two off-by-one range errors happened in
           one session (July 14, 2026) from hand-computing end-start
           against a stale remembered total.
  Format:  gofmt on specific files only -- never on directories.
  Args:    -line takes N:+M for a relative range. N:M is INVALID (error: expected +N after colon). A computed range must be ONE double-quoted string built with a subexpression, because an unquoted parenthesized expression followed by :+13 is split by PowerShell and fedit only sees the first part.
  NOTE:    replace and delete support -match/-endmatch for content-anchored ranges (v1.8.0).
           Anchor replace:  -op replace -match "start" -endmatch "end" -textfile f.txt -v
           Anchor delete:   -op delete -match "start" -endmatch "end" -v
           Substring replace only: replaceall -match "old" -text "new"
           Single-line match-delete: $n = [int](fedit -op find -match X -x) ; fedit -op delete -line $n -v
           PS -x CAST RULE: fedit -x returns STRING -- wrap in [int]() before arithmetic.
             WRONG:   $n = fedit ... -x  →  ($n+3) concatenates: "2443" not 247
   NOTE:    replaceall -v always prints "lines: 0 (unchanged, N total)" in its
            STATS section even when the substitution succeeded -- this is a
            display quirk of that op's stats line, not a failure signal.
            Confirm a replaceall actually applied with a separate -op find
            or -op show on the same anchor, never by reading the STATS line.
             CORRECT: $n = [int](fedit ... -x)  →  ($n+3) = 247
   Paste-safe pattern: one $var = cmd ; cmd per line -- reversal does not break same-line chains.
   Chained find+insert: combine into ONE line for paste safety:
     $n = [int](fedit -op find -match X -x) ; fedit -op insert -line $n -textfile f.txt -v
   Chained format+verify: after any Go file edit, one line covers everything:
     gofmt -w file.go ; gofmt -l file.go ; go build ./... ; go test ./... ; go vet ./...
     gofmt -l prints nothing on success -- silence there means formatting is clean.
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
  VERIFY-CHAIN GOTCHA: `go build ./...` (used in the standard four/five-
  command verify chain) NEVER produces or updates a binary -- it only
  checks that every package compiles, then discards the output. The
  binary on disk is whatever it was from the last EXPLICIT
  `go build -o <name>.exe .` (or ./cmd/<name>). A "manual sanity check"
  against a stale binary can look like a full regression (wrong parse
  errors, phantom tokens, etc.) when the actual code is fine -- cost a
  full debugging detour in fwrite on July 13, 2026 before the binary
  itself turned out to be the problem. ALWAYS run the explicit -o build
  fresh, immediately before any manual/interactive test, on ANY project
  in this workspace -- do not treat `go build ./...` passing as
  sufficient prep for a live run.

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

  user_skill.md    C:\Users\kehsi\Desktop\amalex-brand\fedit\user_skill.md
  fwrite_status.md C:\Users\kehsi\Desktop\amalex-brand\fwrite\fwrite_status.md
  fwrite_design.md C:\Users\kehsi\Desktop\amalex-brand\fwrite\fwrite_design.md
  fwrite builtins  C:\Users\kehsi\Desktop\amalex-brand\fwrite\apps\fwrite\internal\builtins\
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
