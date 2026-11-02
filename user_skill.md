# Amalex Brand -- Claude Working Preferences
# Upload this file alongside the project status file at the start of every chat.
# Applies to: fedit, amalex-handler, fwrite, website, and all Amalex Brand projects.
# Last updated: November 2, 2026

---

## Shell & Environment

  Shell:     Windows PowerShell 5.1 (powershell.exe, NOT pwsh 7). Use ; not &&. No SHA256.HashData (use SHA256.Create().ComputeHash).
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

## Chat Length and Handoff (set Oct 23, 2026)

  Limits:    soft limit about 20 interactions per chat, hard limit about 30. Heavy
             patch work (inline payloads, long test output) fills the context fast, so
             the count is an estimate; hand off earlier if details start to drift.
  Trigger:   a clean milestone, not the count alone. At the soft limit the assistant
             finishes the current step and offers the handoff at the next commit
             boundary (tests green, commit run, tree clean). The assistant says
             "handoff time" when the limit is near; Alex can ask for one at any point.
  Never:     hand off in the middle of a red/green pair or before a patch has been run.
  Last turn: the handoff IS the last response: pasted text, no patch block, no commands
             to run. Code files go as uploads; the handoff text stays pasted text.
  Order:     before the handoff, fedit_status.md is brought up to date in one docs
             commit, so the status file and the handoff text never disagree.
  Contents:  git state (origin/main hash, commits ahead, tree status); what is done;
             next steps in order; decisions in force; known gaps; the last _patchN.txt
             name; the list of files to upload; a first action for the new chat.

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
  fedit work (Oct 20, 2026): the apply line removes the patch file afterwards, so no
  cleanup script is needed. The names keep cycling _patch.txt ... _patch9.txt; the
  last name used is carried in the handoff text.
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
   NOTE:    replaceall always verifies (since Oct 19, 2026): after the write fedit
            re-reads the file from disk and prints VERIFY: PASS or VERIFY: FAIL
            with the changed lines. Exit codes: 0 ok, 1 the edit did not happen,
            3 written but verification failed or could not be done. Trust the
            VERIFY line and the exit code; a separate -op find is optional.
             CORRECT: $n = [int](fedit ... -x)  →  ($n+3) = 247
   Paste-safe pattern: one $var = cmd ; cmd per line -- reversal does not break same-line chains.
   Chained find+insert: combine into ONE line for paste safety:
     $n = [int](fedit -op find -match X -x) ; fedit -op insert -line $n -textfile f.txt -v
   Chained format+verify: after any Go file edit, one line covers everything:
     gofmt -w FILES ; gofmt -l FILES ; go build -o fedit.exe . ; go test ./... ; go vet ./... ; git status -sb
     (fedit standard chain, Oct 20, 2026; go test takes about 120 s; other projects: go build ./... is fine)
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

## fedit Working Rules (set Oct 20, 2026)

  Delivery:
    ONE patch block per response, with its apply command and verify chain in the SAME
    response. Recon (read-only) commands come in their own earlier turn.
    Generate and verify every payload with a tool BEFORE posting. Never post a second
    or corrected block in the same response and never post a draft line; if something
    is wrong, fix it before posting.
    SMALL payloads (about 1.5 KB or less): ONE paste-safe line with a hex value and a
    length guard:
      $h='<hex>' ; if ($h.Length -eq N) { fedit ... -texthex $h } else { 'HEX LENGTH MISMATCH' }
    BIGGER payloads: an ATTACHED file that Alex saves in the fedit folder under its
    exact name, applied by ONE hash-guarded line (hash literal copied from the tool
    output, typed once). New .go files and test files are the attachment itself. The
    hash is checked BEFORE gofmt -w. State the expected hash, size and line count.
    For new identifiers add a guard that Select-String finds no name clash in the
    other non-test .go files (files starting with an underscore are ignored by Go).
    Alex pastes commands starting at `cd`.

  Test-first:
    Red test first (confirm it fails for the right reason), then green. The assistant
    can usually install a Go toolchain in its container (apt-get install -y golang-go,
    Go 1.22 on Oct 20, 2026) and should compile, vet and run new files with a stub
    main before posting; Alex's run is still the real check.

  PowerShell 5.1 details:
    A double quote inside -match is eaten by native argument passing: anchor on text
    without quotes or use -matchhex.
    Compute line numbers in the same command from a content anchor:
      $n=0 ; if (@(fedit ... -op find -match 'x' -x).Count -eq 1) { $n=[int](fedit ... -x) } ; fedit ... -line "$($n+2)"
    Set $n=0 first so a failed cast cannot reuse an old value. -match/-endmatch ranges
    replace whole lines.
    replaceall always verifies; for a mass replaceall omit -v (it prints a window per hit).

  Git:
    No tag, no website deploy and no push without Alex's explicit go-ahead.
    The assistant proposes commit commands (files staged by name, never git add -A);
    Alex runs them.

  Attachments and replacing files (added Oct 21, 2026):
    Every file an apply line needs is attached in the SAME message as the apply line
    (call present_files and check it); a missing file makes the guard fail harmlessly.
    To replace a committed file, attach the whole new file under an underscore name
    (for example _logwire_new.go; Go ignores it). The apply line checks its SHA-256 and
    that `git status --porcelain -- FILE` is empty, then Move-Item -Force over the old
    file, then gofmt -w. For an untracked file guard on the old file's hash instead.
    Name-clash guards use Select-String -CaseSensitive with explicit names (a short
    prefix plus a capital, such as lgd[A-Z], for test helpers).

  Tests that run the binary (added Oct 21, 2026):
    clhBuild runs go build on every call, about 17 s each on this machine. Build once
    per parent test (clhBuild at the top of one Test function) and run the cases as
    t.Run subtests; never call clhBuild from each test. 13 separate builds took the
    suite from 118 s to 408 s.
    When the suite or a command is suddenly slow, measure first (go test -v shows
    per-test times; Measure-Command times one run) before guessing a cause.
    The assistant cannot run the real fedit; it may build a scratch program with the
    same wiring to run the new tests, and Alex's run is still the real check.

  Exit codes (Oct 21, 2026): 0 ok, 1 nothing done, refused or usage error, 2 guided plan
    error or panic, 3 FAIL (verification failed or the -outputresult file could not be
    written),
    4 strict logging refused the run (begin record not written, nothing changed),
    5 strict logging: the run succeeded but its end record could not be written.

  Logging decisions in force (see fedit_status.md): logs are audits, so every
    operation is logged, reads included (-log-scope writes limits it); in strict mode
    -log-level is forced to all; log settings are flag > FEDIT_LOG_* environment >
    default, and off by default.
  Lessons from the v1.12.0 release chat (added Nov 2, 2026):
    Run the FULL suite (go test -count=1 ./..., vet, gofmt -l on a clean clone) BEFORE the tag;
    a pushed tag cannot be moved once the module proxy has cached it. gofmt -l lists six
    entries: demo_sample.go, extract_test.go, _testsystem/test/go/main1.go to main4.go.
    Pre-tests: real Go toolchain when the skeleton compiles (only the fake fedit on PATH);
    blocks under about 12 KB; the edit loop stops at the first failure; files outside the git
    repo get a backup in the Windows temp folder (not the website folder) and a restore.
    .NET file methods ([IO.File]::ReadAllText etc.) resolve a relative path against the PROCESS
    working directory, which cd does not change: pass an absolute path. Cmdlets follow cd.
    Pre-test from a different working directory than the target folder.
    r/devops: self promotion only in the weekly self promotion thread, affiliation disclosed at
    the top, one full URL; use the Markdown editor mode for comments with code blocks.

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
    git add FILES ; git commit -m "message"   (stage files by name, never git add -A)
    git tag vX.Y.Z ; git push ; git push --tags
    (Go module proxy handles distribution -- no GoReleaser needed)
    Before the tag: the full suite, vet and gofmt -l on a clean clone (see the Nov 2 lessons).
    After the tag: GitHub release from the EXISTING tag, as a draft first, with unsigned
    binaries built from a clean clone of the tag (windows amd64, linux amd64 and arm64, macos
    amd64 and arm64, plus checksums.txt) in the Windows temp folder; the first one was v1.12.0.

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
