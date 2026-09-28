# fedit -- Session Status File
# Upload alongside user_skill.md at the start of any fedit-focused chat.
# Last updated: September 27, 2026

═══════════════════════════════════════════════════════════════
FEDIT -- PROJECT STATE
═══════════════════════════════════════════════════════════════

REPO:    github.com/amalexico/fedit (PUBLIC -- MIT)
PATH:    C:\Users\kehsi\Desktop\amalex-brand\fedit\
WEBSITE: amalexhandler.com/fedit (LIVE -- current: v1.8.0)
INSTALL: go install github.com/amalexico/fedit@latest

CURRENT TAG:  v1.8.0 (PUSHED)
PUSHED (Sept 27, 2026): 9050667 and e53e37c are on origin/main. No new tag
  yet (v1.9.0 is the natural next number, not reserved). Code commit
  touched .gitignore, CLAUDE_DESKTOP.md, main.go (~3920 lines), user_skill.md.
IN THAT COMMIT:
  - JSON + CSV map support, Go import advisory (checkMissingGoImports is
    written and tested but still called from nowhere)
  - FIXED: replace with no -text/-textfile/-texthex now errors with exit 1
    (guard at the top of case "replace" in main()). The first entry in the
    BUGS section below is RESOLVED -- ignore it.
  - FIXED: block scanner undercount. Naive brace counting in five scanners
    (getGoBlocks, braceTrackedBlocks, getNixBlocks, getHCLBlocks) was thrown
    off by braces inside string literals. New helper stripStringLiterals
    (line, inBacktick) fixes it. main.go went from 27 to 67 blocks found.
  - NEW, unused so far: batchResult struct + printBatchReport(op, results)
    (shared OK/SKIP/ERROR per-file report for multi-file ops and guided)
UNTRACKED, NOT COMMITTED (repo is PUBLIC, decide before adding): _testsystem/,
  design_cross_file_ops.md, guide_mode_examples.md, description 1.9/2.0.txt,
  guided notes.txt, import_advisory.patch, import_advisory_test.go,
  map_json_csv_test.go, add_fedit_to_goose.ps1, download_cli.ps1,
  fedit-mcp.md, server.json. The two *_test.go files test shipped code and
  probably should be committed.
  .gitignore now covers .mcpregistry_* (token files, NEVER commit),
  _funding_patch.txt and _test_*.txt.

LESSONS FROM THIS SESSION (Sept 27, 2026):
  - Never git add -A here. Run git status -s first and add files by name.
  - A patch file keeps its old content. Overwrite _patch.txt fully before
    every use, and never write "do not run this" next to a command.
  - The -block "o" ambiguity trick lists blocks only when 2+ match. With a
    single match it MUTATES the file. Use it only on files with many blocks.
  - -match with an embedded double quote is mangled by PowerShell (see
    bugs below). Use a quote-free anchor, or -block now that it works.
  - -v verify display can show the wrong location. Confirm with show.

===============================================================
FEDIT -- PENDING TASKS
===============================================================

IMMEDIATE (do in this order):
  [ ] 1. Push commit 9050667, then commit this status file.
  [ ] 2. Regression-check the block scanner on non-Go fixtures (Rust, Java,
         C#, PHP, JS, HCL, Nix). The only test today was Python, which was
         inconclusive because Python has its own scanner.
  [ ] 3. Fix -v verify display (showVerify): it shows context around the
         FIRST match instead of the resolved -nth match, and after
         insertafter -block it shows line 1. The edit itself lands correctly.
  [ ] 4. Fix -match with embedded double quotes: PowerShell mangles it before
         fedit sees it (e.g. -match 'case "replace":' searches for
         case replace:). Candidate: -matchhex mirroring -texthex.
  [ ] 5. Fix -get REGEX: it did nothing on map -lang go (returned the full
         map). Decide whether map should accept -get or only find should.
  [ ] 6. Multi-file work, LOCKED DESIGN (see below). Build order:
         a. multi -file (comma list or glob) for show/find/map, a simple loop
            with a per-file header
         b. multi -file for self-contained mutating ops. First extract pure
            execInsert/execDelete/execReplace/execInsertMatch (return an
            error, no os.Exit) from doInsert/doDelete/doReplace/
            doInsertMatch, keep the do* wrappers unchanged, then loop with
            batchResult + printBatchReport. Partial failure is report and
            continue, never abort the batch (ASSUMED, not yet confirmed).
         c. -dest fan-out for copy/move/replace
         d. -sourceblock/-sourcefile/-destblock
         e. wire checkMissingGoImports into cross-file Go transfers
  [ ] 7. Build the guided entry point (git-style bypass like mcp).
  [ ] 8. Phase 2 of the scanner fix: multi-line string constructs (C#
         verbatim, Java text blocks, Rust raw strings, HCL heredocs).
  [ ] 9. Older items: -afterblock/-beforeblock misleading error for
         insertafter/insertbefore; CSS @media duplicate false positive;
         duplicate-detection for markdown and dockerfile; doMapHCL missing,
         so map -lang terraform still fails.
  [ ] 10. Tag v1.9.0 once the fixes and the first multi-file steps land.

LOCKED DESIGN DECISION (Sept 27, 2026): multi -file and multi -dest are two
SEPARATE mechanisms that NEVER combine. Multi -file = self-contained
per-file operations (show/find/map plus delete/write/replace/replaceall/
insertafter/insertbefore with the match and content inside each file).
Multi -dest = cross-file transfer only (copy/move, and replace/insertafter/
insertbefore sourced from -sourceblock/-sourcefile). No fan-in, no
cross-product. replace -dest DOES fan out to multiple destinations.

═══════════════════════════════════════════════════════════════
FEDIT -- OPERATIONS & FLAGS
═══════════════════════════════════════════════════════════════

OPERATIONS (15):
  show, find, map, insert, insertafter, insertbefore,
  replace, replaceall, delete, write, writeraw, writelines,
  move, copy, fields

MCP TOOLS (14 -- writelines has no MCP tool, interactive stdin only):
  fedit_show, fedit_find, fedit_map, fedit_insert,
  fedit_insertafter, fedit_insertbefore, fedit_replace,
  fedit_replaceall, fedit_delete, fedit_write, fedit_writeraw,
  fedit_move, fedit_copy, fedit_fields

BLOCK SCANNER LANGUAGES (10):
  Go, Python, JS/TS, Rust, Java, C#, Ruby, PHP,
  HCL/Terraform (hcl/tf/terraform), Nix
  Supported on: replace, insert, insertbefore, insertafter, show, move, copy
  NOTE: plain C is NOT supported anywhere in fedit (no block scanner, no
  map function) -- confirmed by reading the source, not assumed. Decision
  on whether to add it is still open.

MAP LANGUAGES (19, was 17 at v1.8.0):
  Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust,
  Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile,
  JSON (NEW), CSV (NEW)
  NOTE: HCL/Terraform and Nix are block-scanner-only -- no doMapHCL exists,
  so `map -lang terraform` still fails today. Real gap, not yet closed.

JSON MAP DETAILS (new):
  - Integrity: valid/malformed JSON, with parse error and cause
  - Structure: root type, top-level keys, max nesting depth, key/object/
    array/scalar counts
  - Duplicate keys WITHIN THE SAME OBJECT -- caught via token-stream walk
    (json.Unmarshal silently collapses real duplicates, so this required
    custom decoding, not just json.Unmarshal + len())
  - Same key name in two DIFFERENT objects is correctly NOT flagged

CSV MAP DETAILS (new):
  - Integrity: valid/malformed CSV, record number + quoting error
  - Structure: row/column counts, header list
  - Ragged rows (field count mismatch vs header), with row-by-row detail
  - Duplicate header columns, with column positions
  - Empty field count across data rows
  - Malformed CSV still reports the header + any rows parsed before the
    failure point (two-pass strict/lenient read)

GO IMPORT ADVISORY (new, checkMissingGoImports -- NOT YET WIRED IN):
  - Advisory-only heuristic for the future guided cross-file transfer report
  - Scoped to Go standard library packages only (curated whitelist) --
    third-party packages are out of scope by design, no fixed list exists
  - Tested against fedit's own source: an unscoped ("any lowercase.Capitalized(
    pattern") version had a ~20% false-positive rate from local variables
    calling exported methods (e.g. scanner.Text(), dec.Token()). The
    whitelist-scoped version eliminates that.
  - Known accepted limitations (documented in code comments + tests):
    aliased imports on either side can produce a false negative (source
    aliases a stdlib pkg) or false positive (dest aliases it) -- both rare,
    both accepted since false negatives are safe and false positives are
    the one failure mode that matters for an advisory hint
  - 8 tests in import_advisory_test.go, all passing, zero regressions to
    the existing suite

KEY FLAGS (v1.8.0, all still current):
  -block NAME + -lang LANG   target named block (no line numbers)
  -raw                       show: bare content for piping to fwencode
  -texthex HEX               hex-encoded content (bypasses PS quoting)
  -extract SPEC              sub-line extraction (see hierarchy below)
  -get REGEX                 pre-filter line before -extract
  -wdelim CHAR               word delimiter for -extract (default: whitespace)
  -x                         find: bare line numbers / fields: no stats footer
  -cleanfirst                truncate file before insert/write
  -stream                    large file mode (replaceall + find)
  -v                         verify after every mutation (ALWAYS USE)
  -nth N                     occurrence selector (default 1, -1 = last)
  -match-regex P             regex pattern with capture groups ($1 $2) for replaceall
  -files GLOB                apply replaceall across matching files (replaceall ONLY
                             today -- no other op accepts multi-file input yet)
  -line N:+M               colon range: N to N+M (M additional lines)
  -line -N                 N-th line from end of file
  -line -N:                last N lines (from -N to EOF)
  -line :                  EOF -- append for insert, last line for show/delete/replace
  -end -N                  end line relative to EOF
  -endmatch TEXT           content-anchor end of range (show, replace, delete, move, copy)
  -quiet                   suppress stdout on success; exit code signals result (wins over -v)

PLANNED FLAGS (designed, NOT YET IMPLEMENTED -- see FEDIT 2.0 section):
  -dest PATH(,PATH,...)     destination file(s) for copy/move/replace
  -sourceblock NAME         block in -file to use as content (cross-file only)
  -destblock NAME           block in -dest to overwrite (replace -dest only)
  -sourcefile PATH          the OTHER file, for insertbefore/insertafter
                            (paired with -sourceblock; -file stays the
                            edited file, unchanged from today)

EXTRACT HIERARCHY (File -> Line -> Word -> Char):
  WN           word N (normalized whitespace, 1-based)
  WN[s:c]      word N, chars s to s+c-1 (1-based start, count c)
  WN[s:]       word N, from char s to end
  WN/DELIM/F   word N, split by DELIM, take field F

WRITE VS WRITERAW:
  fedit_write    -- processes \n \t \\ escape sequences
  fedit_writeraw -- backslashes literal (use for Go regex, Windows paths)
  BOTH create a new file if -file doesn't exist (os.Create semantics) --
  confirmed by reading writeLines(), not assumed. writelines is NOT "always
  a new file" the way it might sound -- it TRUNCATES an existing file at
  that path with no warning, same risk class as the replace bug below.

═══════════════════════════════════════════════════════════════
FEDIT -- KNOWN BUGS (found this session, real, verified against the binary)
═══════════════════════════════════════════════════════════════

HIGH PRIORITY -- SILENT DATA LOSS:
  `replace -line N -end N` with NO -text/-textfile/-texthex silently
  succeeds (exit 0) and deletes the range instead of erroring. Confirmed:
  deleted a real "package analytics" line this way by accident mid-test.
  `delete` already exists as an explicit, separate op -- this looks like
  an unguarded fallthrough, not an intended feature. Fix: require at least
  one content source for replace, or add an explicit -allow-empty flag if
  empty-replace-as-delete is ever wanted on purpose.

MEDIUM -- MISLEADING FLAG, NOT A DECOY BY DESIGN BUT READS LIKE ONE:
  -afterblock/-beforeblock exist in the CLI help text with wording that
  reads like they'd work for insertafter/insertbefore ("Destination:
  insert after named block"), but the actual switch-statement code for
  those two ops only ever reads -block, never -afterblock/-beforeblock --
  those two flags are wired to move/copy destinations only. Using
  -afterblock with insertafter fails with "-match is required", which
  names the wrong problem (the real issue is the unwired flag, not a
  missing -match). Command-level fix is easy (use -block instead); the
  CLI-level fix (better error, or actually wiring -afterblock/-beforeblock
  to insert ops too) is still open.

MEDIUM -- FALSE POSITIVE, PRE-EXISTING (not introduced this session):
  CSS map's duplicate-selector check doesn't track @media nesting depth,
  so `.header` at top level and `.header` inside `@media (...)  { }` are
  flagged as duplicates even though that's completely normal, correct CSS
  for responsive breakpoints.

LOW -- STRUCTURAL GAP, NOT A CRASH:
  markdown and dockerfile are the only two map languages with NO
  duplicate-detection section at all (every other language has one).
  Not wrong, just incomplete -- duplicate headings / duplicate `FROM ... AS
  name` stages are real error classes these two currently can't catch.

CONFIRMED WORKING CORRECTLY (tested, not assumed):
  - Read-only destination file: fedit fails cleanly with "permission
    denied", exit 1, zero partial writes -- but ONLY confirmed when tested
    as a genuinely non-privileged user. Root bypasses Unix permission bits
    entirely, so testing this as root gives a false "it just wrote anyway"
    result. Remember this if retesting permission behavior later.
  - Block-transfer content fidelity: byte-for-byte identical, tabs/blank
    lines/trailing newline all preserved correctly through a real
    show -raw -> hex -> insertafter/replace round trip.
  - Duplicate-function detection catches an accidental double-paste
    correctly when you remember to run `map` afterward -- it's just not
    proactive (insertafter itself doesn't warn at insert time, only a
    follow-up `map` surfaces it).
  - Go compiler/vet/gofmt have ZERO requirement for a blank line between
    top-level declarations -- confirmed empirically (go build, go vet, and
    actual execution all succeed with functions mashed directly together,
    and gofmt's own rewrite doesn't add one either). Same confirmed for
    Python (py_compile) and JavaScript (node). DECISION: fedit will NOT
    auto-insert blank lines on block-anchored insertion, in any language --
    would be fedit doing something nobody asked for with no functional
    justification. This closes what looked like a bug two rounds ago.
  - Cross-file block transfer moves TEXT, not DEPENDENCIES. Confirmed by
    transferring a function that called fmt.Println into a file with no
    fmt import -- textually perfect, does not compile. This is a hard,
    permanent boundary of text-level operation, not something fixable in
    general. The Go import advisory (above) is the narrow, scoped mitigation
    for exactly this one case.

═══════════════════════════════════════════════════════════════
FEDIT -- TESTING SYSTEM (built this session, lives in _testsystem/)
═══════════════════════════════════════════════════════════════

LOCATION: C:\Users\kehsi\Desktop\amalex-brand\fedit\_testsystem\
  Leading underscore is REQUIRED -- Go's build tooling (go build/vet/test
  ./...) automatically skips directories starting with _ or named
  testdata. Without it, every fixture file (including deliberately-broken
  ones) gets compiled as if it were real source and pollutes go vet/test
  output. Learned this the hard way, twice, in this exact repo.

STRUCTURE:
  _testsystem/
    Test-FeditLang.ps1       PowerShell harness for the -lang/map feature
    fixtures/                19 languages x 2 files (clean_a + dup_b) for
                              map/duplicate-detection testing -- 34 pass,
                              0 fail, 4 skip (skips = markdown + dockerfile,
                              no dup-check exists, not a test failure)
    fedit_patch/              json_csv_mappers.patch + its test file
    test/
      go/                     main1-4.go -- REAL fedit source snapshots
                              (main1/main2 identical, main3/main4 identical,
                              main3/4 is a different/longer snapshot than
                              main1/2, not just a longer prefix of the same
                              one -- confirmed by checking which functions
                              exist in each)
      python/                 main1-4.py -- Alex's real production code
                              (HBO Latin America disk-space-report script,
                              2019). main1/2 identical; main3/4 renames
                              class + 3 methods with a "1" suffix, but did
                              NOT update internal call sites (e.g.
                              mainProgram1 still calls self.calculateValues()
                              not calculateValues1()) -- harmless since
                              fedit doesn't execute these files, only reads
                              their structure

TEST METHODOLOGY (Alex's stated philosophy, not a generic best-practice
copy-paste -- worth preserving verbatim for a new session):
  Real multi-file fixtures with authentic block-level operations (copy,
  move, find across files) are preferred over conventional unit-test
  suites (PyUnit-style). The process should be interactive, not pure
  batch pass/fail -- run something, actually read the resulting file
  content, not just check exit codes, because exit-code-only checks miss
  real bugs (both bugs in this file's BUGS section were found exactly
  this way, not by a scripted assertion). Validations for practical
  failure modes (read-only files, malformed commands) matter as much as
  correctness of the happy path.

═══════════════════════════════════════════════════════════════
FEDIT -- "FEDIT 2.0" / GUIDED MODE (design phase, see the two docs below
for full detail -- this is a summary, not a replacement for them)
═══════════════════════════════════════════════════════════════

DOCS (in repo root):
  design_cross_file_ops.md   -- the -dest/-sourceblock/-destblock/-sourcefile
                                engine design, resolved decisions, still-open
                                items
  guide_mode_examples.md     -- full worked-example corpus for `guided`'s
                                fast-syntax across copy/move/replace/
                                insertbefore/insertafter

CORE IDEA: `fedit guided` is an ADDITIVE layer, not a replacement for the
existing -op engine. Same architectural pattern as the existing `mcp`
first-argument bypass (`if os.Args[1] == "mcp" { runMCP(); return }`) --
`guided` would intercept the same way, translate natural-order input into
a real explicit -op command, SHOW that command, and only run it after
confirmation. Every existing script, MCP tool call, and fixture in
_testsystem/ keeps working completely unchanged. Explicitly rejected
alternative: rewriting the core flag parser to support git-style verb
placement everywhere -- too much blast radius for the benefit, and AI
agents (MCP tools) don't benefit from natural word order anyway since they
already call structured named parameters.

WHY -op ISN'T GOING AWAY: tested concretely that multiple operations share
IDENTICAL required flag signatures with no way to disambiguate from flags
alone -- e.g. `-file f -line 5 -end 8` could mean `show` (read-only) or
`delete` (destroys the range). Removing -op would mean guessing between a
safe and a destructive operation from identical input. This is the guided
LAYER's job (natural syntax, translated to explicit -op), not a change to
the engine itself.

GUIDED MODE BEHAVIOR RULES (locked in):
  - Fast-syntax (`fedit guided a.go functionOne copy b.go before
    functionTwo`) NEVER asks questions and NEVER blocks. Always takes a
    sensible fallback action and reports what happened.
  - Only the fully bare `fedit guided` (no other args) does interactive
    step-by-step Q&A.
  - `silent` modifier suppresses output; `report-only` shows just the
    summary; verbose+report is the default.
  - Missing destination anchor -> append at end of that destination file,
    report exactly what happened (never skip, never hard-fail) -- for
    copy/move/insertbefore/insertafter.
  - Missing destination FILE entirely -> create it with just the
    transferred content, report "File X created with Y as content."
  - EXCEPTION: `replace -dest` with a target block that doesn't exist ->
    SKIP with a report line, not append (no sensible "insert instead"
    behavior exists for "overwrite this thing that isn't there").
  - Fan-out (`-dest a,b,c`) is always per-destination, never
    all-or-nothing -- every destination gets attempted and reported
    independently.
  - `move -dest` CAN fan out to multiple destinations (reverses an earlier
    single-destination-only stance) -- redefined as "erase source once,
    land copies in as many destinations as succeed," failing only if
    EVERY destination fails. Ordering: attempt all destination writes
    FIRST, erase source only if >=1 succeeded (avoids the one truly unsafe
    state -- content gone from source, landed nowhere).
  - guided should accept an optional block-name qualifier ("impl Point" vs
    bare "Point") to preempt ambiguous-block errors instead of always
    falling back to skip+report -- direction agreed, exact syntax not
    yet spec'd.
  - No auto-blank-line insertion, ever, in any language -- see BUGS
    section, this was tested and explicitly decided against.

FLAG NAMING (a real collision was caught and fixed mid-design, worth
remembering why): -destblock means EXACTLY ONE THING -- "block in -dest to
overwrite," used only by replace -dest. An early draft also used -destblock
for insertbefore/insertafter's in-file anchor, which was wrong -- those two
ops never needed a new flag for that at all, since -file never stops being
the edited file for them. Plain -block (unchanged, today's meaning) is
correct for that role. This is the same class of bug as the -afterblock
issue above -- a flag silently meaning different things depending on
context -- caught during design instead of after shipping.

STILL OPEN (per design_cross_file_ops.md):
  1. Should every cross-file transfer report include the "check
     imports/dependencies" reminder by default, or only on request? (The
     Go-specific heuristic version -- checkMissingGoImports -- is built
     and tested either way; this question is about whether/how often to
     surface it once wired in.)
  2. Exact syntax for guided's block-name disambiguation qualifier.
  3. "At end of destination" needs NO engine change (confirmed by testing
     -- -after N already accepts the file's real last line number; guided
     just computes N itself). Not open, just noting it's resolved cleanly.

═══════════════════════════════════════════════════════════════
FEDIT -- FILES IN REPO (partial -- repo has accumulated many one-off
patch scripts over time, see `ls` output for the full list)
═══════════════════════════════════════════════════════════════

main.go              (~3850 lines -- primary source, was 3357 at v1.8.0)
mcp.go               (845 lines)
README.md, SKILL.md, CLAUDE_DESKTOP.md, user_skill.md
go.mod, LICENSE, .gitignore
design_cross_file_ops.md, guide_mode_examples.md   (NEW -- fedit 2.0 design)
import_advisory.patch, import_advisory_test.go     (NEW -- applied)
Test files: mcp_test.go, texthex_test.go, extract_test.go, block_test.go,
  iac_test.go, move_copy_test.go, regex_glob_test.go, stream_fields_test.go,
  v180_test.go, map_json_csv_test.go (NEW), import_advisory_test.go (NEW)
_testsystem/          (NEW -- see TESTING SYSTEM section above)
Demo: demo.gif, demo.tape, demo_sample.go

═══════════════════════════════════════════════════════════════
END OF FEDIT STATUS -- September 27, 2026
═══════════════════════════════════════════════════════════════
