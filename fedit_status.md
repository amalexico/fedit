# fedit -- Session Status File
# Upload alongside user_skill.md at the start of any fedit-focused chat.
# Last updated: October 15, 2026

═══════════════════════════════════════════════════════════════
FEDIT -- PROJECT STATE
═══════════════════════════════════════════════════════════════

REPO:    github.com/amalexico/fedit (PUBLIC -- MIT)
PATH:    C:\Users\kehsi\Desktop\amalex-brand\fedit\
WEBSITE: amalexhandler.com/fedit (LIVE, deployed Oct 12, 2026: v1.10 rows for MCP dest and sourcefile; the page shows no version badge)
INSTALL: go install github.com/amalexico/fedit@latest

CURRENT TAG:  v1.11.0 (tagged and pushed Oct 17, 2026; module proxy serves it). Adds MCP replaceall match_regex and textfile; the replace -dest multi-list guard; the Go import hint on -sourcefile/-sourceblock inserts; -sourcefile/-sourceblock with a multi -file target (CLI and MCP); map for hcl and nix (21 languages) with terraform and tf aliases; MCP inferLang fallback to detectLang; js, ts, cs and c# lang aliases for -block and map; fedit guided (bare Q and A, report-only); scanner fixes for PHP heredoc, nowdoc and # comments and the Java escaped triple quote; serverInfo reports 1.11.0. Previous tag: v1.10.0 (Oct 12, 2026, ee8cfde).
UNRELEASED: nothing; everything above is in v1.11.0. The website deploy and the LinkedIn post still wait for separate go-aheads.
PUSHED (Sept 27, 2026): 9050667 and e53e37c are on origin/main. v1.8.1 was tagged later (see the Sept 28 notes below).
  Code commit
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
TRACKED (decided Oct 11, 2026): _testsystem/, design_cross_file_ops.md and
  guide_mode_examples.md are committed and tracked going forward.
GITIGNORED, NOT COMMITTED (kept on local disk only, Oct 11, 2026 decision):
  description 1.9/2.0.txt, guided notes.txt, import_advisory.patch,
  map_json_csv_test.go, add_fedit_to_goose.ps1, (import_advisory_test.go became tracked Oct 12, 2026, 1aa7867)
  download_cli.ps1, fedit-mcp.md, server.json -- untracked via git rm
  --cached (fc7d6b6) and added to .gitignore by exact name.
  .gitignore now covers .mcpregistry_* (token files, NEVER commit),
  _funding_patch.txt, _test_*.txt, and _patch*.txt (widened from the
  singular _patch.txt pattern -- the numbered cycling files
  _patch1.txt..._patch9.txt and _patch_status.txt were NOT covered by
  the old pattern and leaked into commit edcceb7 on Oct 11, 2026;
  caught and cleaned up same session in fc7d6b6, see SESSION LOG).

LESSONS FROM THIS SESSION (Sept 27, 2026):
  - Never git add -A here. Run git status -s first and add files by name.
  - A patch file keeps its old content. Overwrite _patch.txt fully before
    every use, and never write "do not run this" next to a command.
  - The -block "o" ambiguity trick lists blocks only when 2+ match. With a
    single match it MUTATES the file. Use it only on files with many blocks.
  - -match with an embedded double quote is mangled by PowerShell (see
    bugs below). Use a quote-free anchor, or -block now that it works.
  - -v verify display fixed in edea6c4 except replaceall (still first-hit search). Confirm with show if in doubt.

SESSION LOG (Sept 27, 2026, later):
  - Task 2 DONE: block scanner regression check passed for JS, Rust, Java, C#, PHP, HCL and Nix (unbalanced braces inside single-line strings, exact block ranges). Fixtures are in the _testsystem\blockfixtures folder (untracked). Multi-line strings still untested (task 8).
  - Block -lang names: javascript or typescript, rust, java, csharp or c#, ruby, php, hcl or terraform or tf, nix. js, ts, cs and c# are accepted aliases since f0d4bcd (Oct 14, 2026).
  - Task 3 DONE (edea6c4, pushed): package var verifyCenter is set by doInsert, doInsertMatch, doDelete and doReplace; main() prefers it. replaceall keeps the first-hit search on purpose.
  - The PATH entry is the repo folder itself, so go build -o fedit.exe . replaces the PATH binary. Always use the explicit -o build.
  - Task 4 DONE (c6442f7, 06b890e, c027e97, all pushed): -matchhex and -endmatchhex decode into -match/-endmatch right after flag.Parse via decodeHexAnchor(flagName, h). Tests in matchhex_test.go. Documented in README.md and SKILL.md. CLAUDE_DESKTOP.md and mcp.go deliberately skipped (mcp.go was checked later: its params are JSON and never pass through PowerShell, so it needs no change).
  - LESSON: inline fwencode breaks on strings with many double quotes (argv mangling truncated a patch and broke main.go; restored with git checkout). Use a Notepad++ patch file plus -textfile. Trailing blank lines in patch files are dropped, so add a blank line separately with -texthex 0a.
  - Task 5 DONE (bfaa0a3, pushed): -get was never broken on map, it was silently ignored by every op except find. main() now errors if -get is used with any other op and compiles the regex up front (invalid regex exits 1 with the parser error). find -get alone (no -extract) is valid and prints the filtered part of each matched line.
  - Task 6a DONE (8f631c4, pushed): multi -file for show, find and map. -file accepts a comma list and/or globs (a real file with that exact name wins). Code is in multifile.go (runMultiFile, expandMultiSpec, detectLang) with a hook in main() just before the -cleanfirst block; tests in multifile_test.go (14 tests). Output has a header line per file (==> path <==). find skips files with no hits (grep-style) and exits 1 only if nothing matched anywhere; map skips unrecognized extensions with a SKIP line. Mutating ops and -stream are rejected with exit 1 before any file is read. CLI only: MCP handlers bypass main().
  - MCP fix DONE (b82a626, pushed): mcpDoMap read its stdout pipe only after doMap finished, so large maps hung the server, and an explicit unknown lang made doMap call os.Exit and kill it. Now lang is validated against mapLangs first and the pipe is drained in a goroutine. Tests in mcpmap_test.go. Both bugs are in the released v1.8.0; released as v1.8.1 on Sept 28.
  - MCP multi -file plan: mcpDoFind is its own implementation that returns strings, and mcpDoMap swaps os.Stdout, so runMultiFile cannot be reused as is. Do MCP multi-file together with 6b (pure exec* functions), then one batch layer serves the CLI and MCP.
  - Patch files now cycle: _patch.txt, _patch1.txt ... _patch9.txt (max 10), overwrite contents each time. Next file to use is _patch.txt (wrapped; _patch3.txt to _patch9.txt were used Sept 28).
  - v1.8.1 TAGGED AND PUSHED (Sept 28, 2026): includes the MCP map fix, task 6a, -matchhex and the -get guard. Docs for multi -file show/find/map pushed in c50a24f.
  - Task 6b PARTLY DONE (1da049a, pushed): pure functions execInsert, execDelete, execReplace, execInsertMatch and execReplaceAll return errors and never write files or call os.Exit; the do* wrappers keep the exact old messages. New multimutate.go: runMultiMutate does replaceall, insertafter, insertbefore and delete (delete accepts -endmatch, inclusive) over a multi -file spec, reusing batchResult and printBatchReport, wired into main() next to the runMultiFile hook. Tests in multimutate_test.go (7 tests, including a read-only file). -match-regex, -block, -line, -cleanfirst, -stream, -files and the ops replace, write, move and copy are rejected with exit 1 before any file is read. Live-tested in a scratch folder under TEMP. README and SKILL updated (mutation marked unreleased).
  - DECIDED: batch policy is report and continue, never abort. Every per-file problem is a SKIP line and that file is left unmodified. Exit code 0 if at least one file succeeded, 1 if none did.
  - DECIDED (-dest move step): destination writes happen first. A destination that fails (read-only, no permission) is a SKIP, and the source text is removed ONLY if at least one destination write succeeded. If every destination fails, all are SKIP and the source stays intact, so text is never lost.
  - Task 6c PARTLY DONE (6a5470e, pushed; docs d12be7f): -dest cross-file copy and move. New multidest.go (multiDestOpts, multiDestOne, runMultiDest) and dest_flag.go (package-level destFlag = flag.String, registered at init so main()'s flag block is untouched). The hook in main() sits just before the -cleanfirst block, so a rejected combination never truncates anything; it rejects -dest with any op other than copy/move, with no -file, and with -match-regex, -cleanfirst, -stream or -files. Source comes from -line/-end (parseAndResolveLines), -match/-endmatch or -block/-lang. Destination anchors are resolved inside each destination file: -after, -before, -aftermatch, -beforematch, -afterblock, -beforeblock. Per-destination SKIP for a missing file, missing anchor, read-only file, failed write, or a destination that is the source itself; a file listed twice is used once. move follows the no-loss rule: all destination writes first, source trimmed only if at least one succeeded; if every destination fails the source stays untouched and the exit code is 1. Exit 0 if at least one destination succeeded. Tests in multidest_test.go (10 tests, all passing, including a genuine read-only destination). Live-tested in scratch folders under TEMP, including -afterblock and -beforeblock on Go fixtures. README and SKILL updated (-dest marked unreleased).
  - Task 1 (of NEXT SESSION plan) DONE (95f2407, pushed): multi -file -match-regex wired into
    multiMutateOne's replaceall case (new matchRegex field on multiMutOpts); multi -file replace
    added to runMultiMutate/multiMutateOne, reusing delete's match/endmatch range-resolution
    then execReplace; multi -file write added as a special case inside main()'s previously-
    unconditional single-file write block (that block ran before the isMultiSpec check and would
    have silently mis-written to a literal comma-joined filename -- real latent bug, now fixed,
    write is exempt from the multi -file -match requirement). All live-tested in TEMP scratch
    (regex replace across 2 files, write-then-replace on 2 files, single-file write/replace
    regression). Not yet tagged.
  - LESSON: fwencode itself is subject to PowerShell argv mangling before it produces hex --
    wrapping a -match/-matchhex anchor with embedded double quotes in an inline fwencode "..."
    call can still corrupt it (quotes silently stripped), even though -matchhex protects the
    PowerShell-to-fedit hop. For anchors with embedded quotes, prefer a fresh -op show + -line
    N:+M range, or route through Notepad++ _patch.txt + -textfile.
  - -dest with a multi -file list is rejected by the multi -file path ("multi -file supports show, find and map only"), exit 1, nothing modified (live-tested).
  - NOT built for -dest: MCP support (mcpDoReplace/mcpDoInsertAfter/mcpDoInsertBefore need their own -dest wiring since they bypass main()), the Go import advisory wiring (6e). Cosmetic: no blank line between the -dest hook's closing brace and the -cleanfirst comment in main().
  - Task 6d DONE (edcceb7, pushed): replace -dest fans out via new runMultiDestReplace
    (multidest.go) -- requires -destblock, overwrites that block in every destination with
    the source block/range; missing target block is a per-destination SKIP, never an insert
    fallback (locked design decision, confirmed in design_cross_file_ops.md RESOLVED section).
    parseDestList factored out of runMultiDest for reuse. New destblock_flag.go. main()'s
    -dest validation extended to allow replace alongside copy/move.
    insertafter/insertbefore -sourcefile/-sourceblock also DONE: pulls a named block from
    another file as insert content, -file stays the file being edited (NOT a -dest fan-out --
    confirmed against RESOLVED section, which supersedes an earlier draft that suggested
    -sourcefile/-sourceblock might pair with -dest). New sourcefile_flag.go,
    sourceblock_flag.go. resolveTextFull signature extended with (sourceFile, sourceBlock,
    lang string) and now returns ([]string, error) instead of []string -- touched all 6
    call sites in main.go plus 14 call sites in texthex_test.go (new mustResolveTextFull
    test helper added there). New tests: 7 in multidest_test.go (runMultiDestReplace), 6 in
    new sourceblock_test.go (resolveTextFull's source path + one full doInsertMatch
    integration test). All live-verified in TEMP scratch folders. go test ./... and gofmt -l
    clean throughout.
  - LESSON (repeat, now with a concrete cost): -match/-text containing embedded double quotes
    cannot be typed inline as -texthex $(fwencode "...") in PowerShell -- the argv mangling
    corrupts multi-quote strings worse than the single-anchor case documented earlier; five
    separate replaceall attempts with embedded-quote replacement text all silently failed
    ("No lines contain" with visibly mangled search strings) before switching to
    line-numbered -texthex (single-quote content, safe) and finally Notepad++ _patch.txt +
    -textfile for the multi-quote cases. Default to _patch.txt + -textfile for ANY content
    with more than one embedded double quote; don't attempt inline fwencode for it.
  - Git cleanup (fc7d6b6, pushed): a plain git add -A before 6d's commit swept in 152
    unrelated files -- the entire untracked _testsystem/ tree (now intentionally tracked,
    see below), all 9 numbered _patch*.txt cycling files plus _patch_status.txt (NOT caught
    by the old singular _patch.txt gitignore pattern), and 10 loose scratch files. Commit
    edcceb7 (6d itself) already carried this debris before it was caught -- cleaned up in
    the very next commit (fc7d6b6) via git rm --cached (files kept on local disk) plus a
    widened .gitignore (_patch.txt -> _patch*.txt). DECIDED: _testsystem/ and the two design
    docs are legitimate tracked content; the 10 loose scratch files stay local-only.
    LESSON (already in this file above, reconfirmed): never git add -A blind -- run git
    status first and review before every commit, every time, no exceptions.
  - SESSION Oct 11, 2026 (v1.9.0 prep): MCP multi-file and docs, all pushed.
    7455f2e: fedit_show/fedit_find/fedit_map take a comma list or glob. New mcpmulti.go (mcpRunMulti: one ==> path <== header per file, find skips files with no hits, report-and-continue, error only if no file produced results); hook in mcpExecTool above the filepath.Abs line. Tests in mcpmulti_test.go (8).
    a8da65c: collectMultiMutate split out of runMultiMutate (returns []batchResult and the OK count, prints nothing); the CLI wrapper keeps its exact messages and exit codes.
    c346752: MCP multi-file mutation for fedit_replaceall, insertafter, insertbefore, delete and replace. New mcpmultimut.go (mcpMultiMutTools, mcpMultiMutOpts, mcpRunMultiMutate), second hook in mcpExecTool. DECIDED: fedit_write is NOT in the list, so a write to a path with a comma still creates that file. match is required; block, line/end, stream and files are rejected before any file is read. Tests in mcpmultimut_test.go (13 incl. a real read-only file).
    1288847, 3ab8353: MCP tool descriptions mention multi-file. f7fa423, 55e283e, 230eaa3, 7fb1491: README and SKILL brought in line with the code (replace and write over a list, -match-regex over a list, replace -dest with -destblock, -sourcefile/-sourceblock, all unreleased markers dropped; MCP -dest still marked not supported).
    Smoke test (scratch folder under TEMP) of the documented replace -dest and insertafter -sourcefile examples passed, incl. the missing-block SKIP and both rejections.
    LESSON: fedit find without -x prints its report to the console, not stdout, so a -match test on its output silently fails; guard with find -x and count lines.
    LESSON: a patch-file guard must check the file CONTENT, not just Test-Path; a stale _patch8.txt put 27 junk lines into SKILL.md (repaired).
    LESSON: new test helpers need a unique prefix (mmRead clashed with multimutate_test.go; renamed mcpMMRead).
    KNOWN: single-file fedit_replaceall ignores match_regex, files, stream and textfile although its schema advertises them (task 9).
  - SESSION Oct 12, 2026 (MCP dest, unreleased): pushed, not tagged.
    3ef09cc: collectMultiDest and collectMultiDestReplace split out of runMultiDest and runMultiDestReplace (new multiDestOutcome and printMultiDest; the CLI keeps its exact messages and exit codes). The move no-loss rule is unchanged: destination writes first, source trimmed only if at least one succeeded.
    6fae3bb: MCP dest on fedit_copy, fedit_move and fedit_replace, destblock on replace. New mcpmultidest.go (mcpDestTools, mcpMultiDestOpts, mcpRunMultiDest, mcpSourceBlockContent), hook in mcpExecTool after the filepath.Abs line, only when dest is present. Tests in mcpmultidest_test.go.
    d69ab91, 14aa93f: schemas advertise dest and destblock; fedit_insertafter and fedit_insertbefore take sourcefile and sourceblock (needs lang, not with text). mcpDoInsertMatch is now a thin wrapper over mcpDoInsertMatchLines. Tests in mcpsourceblock_test.go. 5adce73: README and SKILL no longer say CLI only.
    DECIDED: text stays required on fedit_replace, because an empty text silently deletes the range (the old CLI bug); with dest the text is ignored and any placeholder works. Dropping text from required is safe on insertafter and insertbefore, which return Nothing to insert.
    RESOLVED (checked Oct 12, 2026): a comma list in file on fedit_copy or fedit_move with dest already returns "dest does not combine with a multi file list: file must be the single source file" and modifies nothing; the old KNOWN note about a poor file-not-found message was stale. Locked in by TestMCPExec_Dest_RejectsMultiSource.
  - DONE (245ec57, pushed Oct 12, 2026, not tagged): fedit_map description and lang schema now list the 19 languages in mapLangs (the old 17 and the js, ts, cs, hcl, tf, terraform and nix names were wrong for map); TestMCPToolDefs_MapLangsMatchSupportedSet in mcp_maplangs_test.go replaces the two IaC description tests; README and SKILL say 19.
  - DONE (Oct 12, 2026, not tagged): BUG FIXED: fedit_replace with dest and a comma list in file fell into the multi file replace path, ignored dest and overwrote the sources with the placeholder text. mcpMultiMutOpts now rejects dest with a multi file list, with the same message as copy and move. Tests in mcp_destmulti_test.go cover copy, move and replace.
  - DONE (6e31571, pushed Oct 12, 2026, not tagged): single-file MCP fedit_replaceall now honours match_regex (execReplaceAllRegex; errors for neither or both of match and match_regex, "No lines match: ...") and textfile (trailing newlines trimmed; error if text is also given). files is rejected with a message pointing to file. stream stays accepted and is processed normally (TestMCPExec_ReplaceAll_Stream sends it). The multi-file comma-list path in mcpmultimut.go already rejected stream and files but ignored textfile, which would have replaced matches with empty text; it now honours textfile too. The literal match path is unchanged. Schema text, README and CLAUDE_DESKTOP rows updated. 12 new tests in mcp_replaceall_test.go (helper prefix mra; prefixes already used: mcpMD, mcpMM, mcpSB, mdt, mrd, mra).
  - LESSONS (Oct 12, 2026): fedit find without -x prints to the console, not stdout, so guards must use find -x and count lines; a find -match with embedded double quotes is mangled by PowerShell; patch-file guards must check the file content, not just Test-Path; the website folder is not in the git repo (deploy only on explicit go-ahead); never tag without an explicit go-ahead.
  - DONE (90e60fc, 1aa7867, pushed Oct 12, 2026, not tagged): Go import advisory wiring (6e), first step. Copy, move and replace with dest (CLI and MCP) add an advisory note in out.notes when the transferred lines use stdlib packages a Go destination does not import. goImportHint in multidest_advisory.go calls checkMissingGoImports only when source and destination are .go files; it never changes OK or SKIP results, exit codes or move semantics. Tests: multidest_advisory_test.go (5 new) and import_advisory_test.go, now tracked (removed from .gitignore) so its 8 tests run for everyone.
  - DONE (4935255, 9075bbb, 51a32eb, a7841f6, f8fbdec, baf97c4, d251f06, a44a175; pushed Oct 13, 2026, not tagged): Go import advisory wiring (6e), second step. sourceBlockHint (multidest_advisory.go) returns the existing goImportHint only when -sourcefile and -sourceblock were both used, and the hint never changes an insert, an OK or SKIP result or an exit code. Wired into: single-file CLI insertafter and insertbefore (hint printed to stderr after the Inserted line, both the -match and the -block branch); single-file MCP (mcpDoInsertMatchLines now takes srcFile and srcBlock and appends the hint to the OK message); multi -file insertafter and insertbefore on both CLI and MCP (multiMutOpts gained srcFile and srcBlock; collectMultiMutate computes the hint per file from that file own pre-edit lines and appends it to that file report detail). insert and replace already reject -sourcefile, so they carry no hint. DECISION (user, Oct 13, 2026): -sourcefile and -sourceblock now work with a multi -file target on the CLI (rejection removed from main(); the block is read once and inserted into every file at its own -match, per-file report, a file without the match is a SKIP) and on MCP (mcpMultiMutOpts reads sourcefile and sourceblock, rejects text together with them and requires lang; before this it ignored them). MCP tool descriptions, README and SKILL updated. Tests (helper prefix sbh): sourceblock_hint_test.go (4), sourceblock_multi_test.go (2), sourceblock_mcpmulti_test.go (3), sourceblock_mcpsingle_test.go (4). The two CLI hint sites sit in main() and have no automated test; they were live-tested in scratch folders under TEMP with a fresh go build -o fedit.exe . (hint for a Go destination missing the import, none for a .txt destination or plain -text, multi -file report 3/4 with one SKIP).
  - DONE (7d488ba, pushed Oct 13, 2026, not tagged): map for HCL/Terraform and Nix. New map_hcl.go: doMapHCL groups the top-level blocks of a .tf or .hcl file by kind with getHCLBlocks and flags duplicate resource, data, module, variable and output blocks; provider aliases, locals, terraform, moved, import and check can legitimately repeat and are never flagged; a one-line block header is cut at its brace. doMapNix lists the block attributes and the single-line attributes of the unindented top level with getNixBlocks and flags an attribute path defined twice; a file whose attributes are all indented (a function or let/in body) prints a note instead of a list. mapLangs has 21 languages (hcl and nix added). normMapLang maps the terraform and tf aliases to hcl in doMap, multi -file map and the MCP map handler; the aliases are not mapLangs keys. .tf, .hcl and .nix are auto-detected in doMap and detectLang. mcp.go inferLang is a third extension table (it had no json, csv, jsx, tsx, markdown, tf, hcl or nix, so fedit_map without lang failed on them); it now falls back to detectLang, and its only caller is the MCP map handler. The fedit_map description and lang schema list 21 languages. Tests: map_hcl_test.go (12, prefix mh), infer_lang_test.go (1); TestDetectLang in multifile_test.go now expects hcl and nix. Live-tested on the _testsystem .tf and .nix fixtures (CLI, -lang terraform, multi -file) and over MCP without lang. LESSON: a hex payload pasted by hand had one corrupted character in a comment; a SHA-256 check of the written file against the tested copy caught it, so large pasted files get a hash check.
  - Open items: no automated test covers the two CLI hint sites in main() (a test that builds and runs the binary would close it). A fixture written through fwencode with embedded double quotes lost them (a Go fixture got import strings without quotes); use [IO.File]::WriteAllText for such fixtures. replace -match needs -endmatch, so for a single line use find -x then replace -line N. go build ./... does not refresh fedit.exe, so run go build -o fedit.exe . before any live test.
  - NEXT SESSION (older plan; item (0) is DONE as of Oct 14, 2026, see the SESSION Oct 14 entry), in order: (0) documentation complete and accurate for everything unreleased since v1.10.0: README, SKILL, CLAUDE_DESKTOP, the MCP tool descriptions in mcp.go and the CLI flag help text, with the since v1.11.0 label fixed before tagging, the two README LLM Benchmark sections merged, and the CLAUDE_DESKTOP block examples checked against the tool schemas; (1) the remaining task 9 older items (MCP fedit_find invalid get regex, detectLang duplication, the misleading -afterblock and -beforeblock error, CSS @media duplicate, markdown and dockerfile duplicate detection); then tasks 7 and 8. Release plan: no tag and no website deploy without an explicit go-ahead; the LinkedIn announcement (what is new in fedit, plus a request for feedback) waits until all multi-path work is complete.
  - SESSION Oct 14, 2026 (documentation pass; HEAD was 77206cf and the last tag v1.10.0 at the start; nothing tagged, website not deployed):
    b5e464b: status file up to 7d488ba. 25f92fa: README brought in line with the code (21 languages, full flags table, provisional since v1.11.0 labels, the two LLM Benchmark sections merged, an unclosed code fence fixed, double-encoded dashes repaired, FAQ corrected).
    f0d4bcd: FIX, js, ts, cs and c# are accepted as lang aliases for -block and map (getTopLevelBlocks calls normMapLang first); tests in lang_alias_test.go. The older note that js is NOT a block lang name was wrong.
    4f9acab: SKILL.md (many-files section, import hint, 21 languages, flag rows, anti-patterns 11 and 12). a6c485c: CLAUDE_DESKTOP.md (tool table with MCP parameter names, section 7 examples fixed and extended, troubleshooting, changelog lines).
    77206cf: REAL STATE: only the lang schema description of fedit_insertafter and fedit_insertbefore in mcp.go changed (2 lines). Its commit message wrongly claims more. The rest of that text work did not apply because the PowerShell helper was named Rp, a built-in alias of Remove-ItemProperty.
    52c5cf3: the redo, 7 guarded edits, 0 skipped, gate passed: mcp.go fedit_replace description (file must then be a single path); mcp.go fedit_map lang schema (accepted aliases terraform, tf, js, ts, cs, c#); main.go -lang help (21 map languages, block languages, aliases), -endmatch help (show, replace, delete, move, copy), -file help (comma list or glob); dest_flag.go help (copy, move and replace; replace needs -destblock); sourcefile_flag.go help.
    1d49a71: GAP FOUND AND FIXED, the no-argument usage text was stale (map -lang go|html|sql, -block for move/copy only, and no -dest, -destblock, -sourcefile, -sourceblock, destination anchors, -times, -raw, -extract, -get, -wdelim, -lang or list support in -file). Seven operation lines rewritten and new sections added (block and language, range, destination, cross-file, find and show, output flags); a live run of the freshly built fedit.exe confirmed the new text and that go|html|sql is gone. Text only, no behaviour change.
    Documentation for everything unreleased since v1.10.0 is now complete: README, SKILL, CLAUDE_DESKTOP, mcp.go tool descriptions and CLI help. The since v1.11.0 labels stay provisional until the tag is confirmed, and the serverInfo version in mcp.go (1.10.0) changes at tag time.
    LESSONS: the prompt in this session was Windows PowerShell 5.1 (5.1.26100.9444), not pwsh 7.6.2, so check $PSVersionTable.PSVersion first. Never name a helper after a PowerShell alias (Rp, rp, sc); check Get-Command and Get-Alias first. The guarded helper pattern (edit only when the anchor matches exactly once, count skips and edits, gate the commit on counts, gofmt, build, vet, test and a live check) did 15 edits with 0 skips.
  - SESSION Oct 15, 2026 (continuation, task 9 done; nothing tagged, website not deployed):
    5f743f5: status file (the Oct 14 entry above). 0ed69a3: TEST, cli_hint_test.go builds and runs the real binary and covers the two CLI hint sites in main() (insertafter and insertbefore, -block branch, already imported, .txt destination, plain -text). The whole suite now takes about 30s because those tests rebuild the binary; go test -short skips them.
    0a5507d: FIX, MCP fedit_find rejects an invalid get regex with "get: invalid regex" (mcpCheckGet in mcpgetcheck.go, called in mcpExecTool before the multi-file branch; applyGet itself still skips silently); the get description in mcp.go says so.
    1a8fe58: REFACTOR, doMap auto-detect and inferLang use detectLang (multifile.go) as the single extension table; lang_table_test.go locks the table for detectLang, inferLang and the CLI map.
    d45cfc2: FIX, -afterblock and -beforeblock with insertafter/insertbefore fail with a clear error pointing to -block (guard in main() next to the -get guard); other ops still ignore these two flags silently (not changed). Test: block_anchor_guard_test.go.
    0ca333d: FIX, CSS map counts duplicate selectors per container block (css_scope.go: doMapCSSSelectors, cssBraces); the same selector in different @media or @keyframes blocks, and from/to in keyframes, are no longer duplicates; repeats inside one block report where. Tests: css_scope_test.go.
    4486609: FEATURE, duplicate detection for markdown headings (same text, same level, same parent, case-insensitive, closing hashes ignored) and Dockerfile (duplicate stage names, case-insensitive; CMD/ENTRYPOINT/HEALTHCHECK repeated in one stage). The markdown heading list now skips fenced code (backticks and tildes), so shell comments are no longer listed as headings. Code in map_dupes.go, tests in map_dupes_test.go.
    TASK 9 IS COMPLETE (invalid get regex over MCP, detectLang duplication, misleading -afterblock/-beforeblock error, CSS @media duplicate false positive, markdown and dockerfile duplicate detection); the Task 9 entry in the numbered task list further up is DONE and can be tidied.
    LESSONS: pasted output shows $$ where the source has a backslash-bracket (a display artifact), so never copy regexes from pasted output. A test-first red/green gate (new tests fail before the edit, pass after) worked for every fix. Hex payload plus SHA-256 check for every new file worked with zero corruption. New helper prefixes used: clh, mgc, abg, mkd, dkr, dup.
  - SESSION Oct 15, 2026 (later, task 7 slice 1 done; nothing tagged, website not deployed):
    f39864b: FEATURE, guided entry point slice 1 (guided.go, guided_test.go, hook in main() before flag parsing). fedit guided translates natural order input into an explicit -op argv, echoes it to stderr, rewrites os.Args and falls through into the normal flag path (engine untouched). Verbs: copy, move, replace, insertbefore, insertafter. Source is a named block or "line N-M". Position is before/after TARGET or at-end; no position appends at end with a note; a missing dest file counts as line 0 and the engine creates it; fan-out takes one quoted comma list; silent swaps -v for -quiet. The translation matches the shipped engine: copy, move and replace use -block with -lang for the source (NOT -sourceblock), insert from another file uses -sourcefile plus -sourceblock with -block as the anchor, insert of new text uses -text. Smoke-tested live on scratch files in TEMP (copy before, no position, replace default target, move, insertbefore from file, silent).
    TASK 7 SLICE 1 DONE. OPEN for slice 2: bare fedit guided (interactive Q and A); numbered choice for ambiguous blocks and the "impl Point" qualifier; per-destination missing-anchor append fallback inside fan-out, and at-end for several destinations (guided currently errors on both); wording for replace with a missing target (SKIP); report-only modifier; silent still prints the engine per-destination report for -dest because -quiet does not cover it; docs (README, SKILL, CLAUDE_DESKTOP) not yet updated for guided.
    LESSONS: the design doc flag names (-sourceblock for copy) did not match the shipped engine, so always smoke-test a translation against the real binary. This terminal is Windows PowerShell 5.1 (5.1.26100), not pwsh 7: no SHA256.HashData, no &&; user_skill.md still says 7.6.2.
  - SESSION Oct 15, 2026 (later still, guided fan-out fallbacks done; nothing tagged, website not deployed):
    59363a9: FEATURE, guided now plans one or more engine commands (planGuided and guidedStep with a create list; translateGuided is the single-command wrapper used by tests). Per destination: a missing dest file is created empty and gets -after 0; a missing before/after anchor falls back to appending at the end (-after N) with a note; destinations that have the anchor stay in one -dest group; at-end works for several destinations (one command each); move runs its copies first and the move last, so the source is erased only by a write that landed. Several commands run as child processes of the same binary (flag.Parse runs once); one plain command still falls through in-process. Smoke-tested live: fan-out with a missing anchor, missing dest file, at-end fan-out, move fan-out.
    SLICE 2 STILL OPEN: bare fedit guided Q and A; numbered choice for ambiguous blocks and the "impl Point" qualifier; report-only; silent still shows the engine -dest report; docs (README, SKILL, CLAUDE_DESKTOP) for guided. LESSON: send payloads over about 10 KB of hex as attached files, a 32 KB paste froze the app.
    6ee1f11: FEATURE, guided resolves ambiguous block names. The engine matches block names by substring of the display name or key (resolveBlock), so two also matches twoSum. guided now looks blocks up itself (guidedLookup via getTopLevelBlocks): a single exact key or name match wins with a note; otherwise guidedAsk shows a numbered choice when stdin is a terminal; otherwise it fails with the candidate list (exit 2) and changes nothing. The chosen block is addressed by line numbers: source as -line S -end E, destination anchors as -before S or -after E (one command per such destination). Fixed a bug in the fan-out pre-check where an ambiguous anchor silently appended at the end. Not covered: ambiguous source in insert-from-file, replace -destblock (stays a per-destination SKIP), insert anchors. Smoke-tested live: exact match, no-terminal failure, line-range source.
    SLICE 2 REMAINING (both DONE in e44f082, see the e44f082 entry below): bare fedit guided Q and A; report-only. The silent vs engine -dest report item is DONE in 72243a7: printBatchReport prints nothing under -quiet when every file succeeded (batchQuiet, set where -quiet is handled; skips and errors still print, exit code unchanged), so guided silent is now quiet, and -quiet on a multi -file run behaves the same; -quiet help text updated; test quiet_dest_test.go builds the binary, red first then green. Docs for guided are DONE. README and SKILL wording for -quiet is DONE (README guided line and flag row, SKILL flag row).
    - DOCS PASS FOR GUIDED AND TASK 9 DONE (Oct 15, 2026, committed after 6ee1f11): README has a new guided section, the markdown, Dockerfile and CSS map rows with the new duplicate reports, the corrected mapper sentence, the -afterblock/-beforeblock error note and the MCP get error; SKILL has the -get line; CLAUDE_DESKTOP has the get error and the Unreleased list including guided. The since v1.11.0 labels stay provisional and serverInfo stays 1.10.0 until the tag.
    949a7ac: FIX, task 8: multi-line string constructs no longer move block boundaries. New strstate.go (strState.strip, language aware) replaces the inBacktick bool in the four scanner loops (Go, braceTrackedBlocks, HCL, Nix); stripStringLiterals is no longer used by the scanners. Constructs: C# at-quote and dollar-at-quote strings (doubled quote escape), C# raw strings and Java text blocks (three or more quotes), Rust r and r-hash raw strings, HCL heredocs (<<EOT and <<-EOT), Nix indented strings (two single quotes, with the escapes). Language modes: hcl, nix, clike (the shared braceTrackedBlocks, so JS, PHP, Java, C# and Rust), go. Also fixes a one-line Nix indented string that was miscounted before. Tests: blocks_multiline_test.go, 14 cases, red first (10 failed) then green; the seven fixture maps are unchanged and correct.
    TASK 8 FOLLOW-UPS (the Rust lifetime one is DONE in 0f364b9: strState got a rust mode that skips char literals and treats lifetimes and loop labels as code, getRustBlocks now calls the new braceTrackedBlocksLang with rust while the other callers keep clike, tests in rust_lifetime_test.go, red first then green; the rest are DONE too, see the end of this entry): braces inside comments are DONE in 75654f3 (strState skips // and /* */ comments for Go, Rust (nested), Java, C#, JS and PHP and HCL, # only for HCL and Nix, never // for Nix because it is the update operator; tests in comment_test.go, red first then green; known gaps: a JS regex with an unescaped /* starts a bogus comment, a block comment that opens outside any block is not tracked, and a block start that sits inside a comment is still matched); PHP heredoc and nowdoc, PHP # comments and a Java text block with a backslash-escaped triple quote are DONE in 6e729d1 (strState got a java lang with an esc flag, and a php lang with phpHeredocOpen and phpHeredocEnd; getJavaBlocks and getPHPBlocks now call braceTrackedBlocksLang with java and php; tests in blocks_multiline2_test.go, red first with 6 failing and 3 guards passing, then green); the unused stripStringLiterals is REMOVED in cda6857; docs in 3f433ad (README and CLAUDE_DESKTOP, SKILL needed no change); known gap: a PHP heredoc that opens outside any block is not tracked, because only lines inside a matched block are fed to the scanner.
  e44f082: FEATURE, task 7 slice 2 DONE (Oct 17, 2026). report-only is a trailing modifier like silent: planGuided strips it and drops -v from every step, and runGuided skips the echo line; the notes and the engine summary (the OK lines, which go to stderr) stay. Bare fedit guided calls guidedInteractive (new guided_interactive.go) from runGuided: prompts go to stderr, answers come from stdin one per line, so piped input works; the order is operation, source file and block, destination, position and target, then the modifier; operation, insert source (file or text) and position accept a name or a number; a wrong choice is asked again up to 3 times; EOF before the last answer fails with "nothing changed" (EOF at the modifier prompt means none); fast syntax never prompts. Usage text updated. Tests: guided_slice2_test.go (11, red first with 10 failing and the fast-syntax guard passing; the Q and A tests run the built binary with piped stdin; one test looked for the engine summary on stdout although it is on stderr, fixed in the same commit). Docs ec934c6 (README guided section, CLAUDE_DESKTOP unreleased list; SKILL has no guided entry because guided is CLI only). LESSON: in Windows PowerShell 5.1 the [IO.File] methods resolve relative paths against C:\WINDOWS\system32, not the current location, so build absolute paths with Join-Path (Get-Location).Path.
- NEXT, in order: (1) no feature work is left: slice 2 is DONE (bare fedit guided Q and A and report-only in e44f082, docs in ec934c6; the silent item in 72243a7) and the scanner follow-ups are all DONE (PHP heredoc and nowdoc and PHP # comments and the Java escaped triple quote in 6e729d1, stripStringLiterals removed in cda6857, docs in 3f433ad); docs for the scanner fixes are DONE (README section How block ends are found, SKILL, CLAUDE_DESKTOP); RELEASE READINESS CHECKED Oct 15, 2026 at 6302be2: full go test passes, go vet clean, 53 commits since v1.10.0, serverInfo still 1.10.0 (bump at tag time, no test asserts it). Since that check: 6e729d1, cda6857 and 3f433ad (scanner follow-ups and their docs), go test -short passes at ec934c6; also since: e44f082 and ec934c6 (guided slice 2 and its docs). gofmt: demo_sample.go is intentionally unformatted (go:build ignore, the demo.gif sample) and must stay as is; extract_test.go wants only comment alignment but gofmt -w also rewrites a non-ASCII byte sequence in a comment next to the extractTestContent fixture, so it is left as is on purpose; the underscore patch fragments (_mcp_*.go, _p_*.go) are ignored by Go, _p_*.go and _mcp_*.go are in .gitignore and none is tracked; the 7 tracked files matching _*.go are _testsystem/ test material kept on purpose for future tests (decided Oct 17, 2026), nothing to remove; (2) before any tag: bump serverInfo in mcp.go (1.10.0) and confirm the since v1.11.0 labels; (3) tag, website deploy and the LinkedIn post only on the explicit go-ahead.

===============================================================
FEDIT -- PENDING TASKS
===============================================================

IMMEDIATE (do in this order):
  [x] 1. DONE: commits pushed, status file committed.
  [x] 2. DONE (see SESSION LOG). Regression-check the block scanner on non-Go fixtures (Rust, Java,
         C#, PHP, JS, HCL, Nix). The only test today was Python, which was
         inconclusive because Python has its own scanner.
  [x] 3. DONE (edea6c4). Fix -v verify display (showVerify): it shows context around the
         FIRST match instead of the resolved -nth match, and after
         insertafter -block it shows line 1. The edit itself lands correctly.
  [x] 4. DONE (c6442f7, 06b890e, c027e97). Fix -match with embedded double quotes: PowerShell mangles it before
         fedit sees it (e.g. -match 'case "replace":' searches for
         case replace:). Candidate: -matchhex mirroring -texthex.
  [x] 5. DONE (bfaa0a3). Fix -get REGEX: it did nothing on map -lang go (returned the full
         map). Decide whether map should accept -get or only find should.
  [ ] 6. (6a DONE 8f631c4, 6b PARTLY DONE 1da049a) Multi-file work, LOCKED DESIGN (see below). Build order:
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
         e. DONE (90e60fc, pushed Oct 12, 2026): checkMissingGoImports is called from collectMultiDest and collectMultiDestReplace via goImportHint (multidest_advisory.go); advisory note only. The sourcefile and sourceblock insert path was wired Oct 13, 2026 (sourceBlockHint); open: no automated test for the two CLI hint sites in main().
  [~] 7. Guided entry point (git-style bypass like mcp): slice 1 DONE (f39864b, Oct 15, 2026); slice 2 DONE (e44f082, docs ec934c6, Oct 17, 2026), task 7 complete.
  [x] 8. DONE (Oct 15, 2026, 949a7ac; see the SESSION Oct 15 entry). Phase 2 of the scanner fix: multi-line string constructs (C#
         verbatim, Java text blocks, Rust raw strings, HCL heredocs).
  [ ] 9. Older items: -afterblock/-beforeblock misleading error for
         insertafter/insertbefore; CSS @media duplicate false positive;
         duplicate-detection for markdown and dockerfile. (doMapHCL and doMapNix are DONE,
         Oct 13, 2026, 7d488ba: map now supports hcl and nix.)
         ALSO (remaining): MCP fedit_find bypasses main() so an invalid get regex still skips every line silently there.
         ALSO (remaining): detectLang in multifile.go duplicates the doMap extension switch: make doMap call it. mcp.go inferLang is a third table; since 7d488ba it falls back to detectLang for anything it does not know. The fedit_map schema alias problem and the 17-language wording were fixed Oct 12, 2026 (245ec57).
  [ ] 10. Tag the next release (v1.11.0 expected; v1.9.0 and v1.10.0 are already tagged) once the remaining multi-path work and the documentation are complete. Explicit go-ahead required; never tag before that.

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
  HCL/Terraform (hcl/tf/terraform), Nix (aliases js, ts, cs, c# accepted since f0d4bcd)
  Supported on: replace, insert, insertbefore, insertafter, show, move, copy
  NOTE: plain C is NOT supported anywhere in fedit (no block scanner, no
  map function) -- confirmed by reading the source, not assumed. Decision
  on whether to add it is still open.

MAP LANGUAGES (21, was 17 at v1.8.0):
  Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust,
  Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile,
  JSON (NEW), CSV (NEW), HCL/Terraform (NEW, Oct 13), Nix (NEW, Oct 13)
  NOTE: hcl and nix map through doMapHCL and doMapNix (map_hcl.go, 7d488ba); terraform and tf are aliases for hcl (normMapLang).
  Nix maps only the unindented top level, the same rule as the block scanner.

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

GO IMPORT ADVISORY (checkMissingGoImports -- WIRED Oct 12-13, 2026):
  - Advisory-only heuristic: printed on every cross-file -dest run and on -sourcefile/-sourceblock inserts (Go to Go only), and still meant for the future guided report
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
  -files GLOB                apply replaceall across matching files (replaceall only; every other op takes a comma list or glob in -file)
  -line N:+M               colon range: N to N+M (M additional lines)
  -line -N                 N-th line from end of file
  -line -N:                last N lines (from -N to EOF)
  -line :                  EOF -- append for insert, last line for show/delete/replace
  -end -N                  end line relative to EOF
  -endmatch TEXT           content-anchor end of range (show, replace, delete, move, copy)
  -quiet                   suppress stdout on success; exit code signals result (wins over -v)
  -file LIST               comma list and/or glob: show/find/map, replaceall, insertafter, insertbefore, delete, replace (with -match) and write, per-file SKIP, never abort; MCP takes the same lists

CROSS-FILE FLAGS (implemented Oct 11-13, 2026; designed in the FEDIT 2.0 section):
  -dest PATH(,PATH,...)     destination file(s) for copy/move/replace
  -sourceblock NAME         block in -sourcefile to insert (insertafter and insertbefore; needs -lang)
  -destblock NAME           block in -dest to overwrite (replace -dest only)
  -sourcefile PATH          the OTHER file, for insertbefore/insertafter
                            (paired with -sourceblock; -file stays the
                            edited file, unchanged from today)
                            -file may also be a comma list or glob; the block is read once.

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

main.go              (~4220 lines -- primary source, was 3357 at v1.8.0)
mcp.go               (934 lines)
README.md, SKILL.md, CLAUDE_DESKTOP.md, user_skill.md
go.mod, LICENSE, .gitignore
design_cross_file_ops.md, guide_mode_examples.md   (NEW -- fedit 2.0 design)
import_advisory.patch, import_advisory_test.go     (NEW -- applied)
map_hcl.go, map_hcl_test.go, infer_lang_test.go (NEW Oct 13, 2026 -- HCL and Nix map, inferLang fallback)
lang_alias_test.go (NEW Oct 14, 2026 -- js, ts, cs, c# lang aliases)
Test files: mcp_test.go, texthex_test.go, extract_test.go, block_test.go,
  iac_test.go, move_copy_test.go, regex_glob_test.go, stream_fields_test.go,
  v180_test.go, map_json_csv_test.go (NEW), import_advisory_test.go (NEW)
_testsystem/          (NEW -- see TESTING SYSTEM section above)
Demo: demo.gif, demo.tape, demo_sample.go

═══════════════════════════════════════════════════════════════
END OF FEDIT STATUS -- October 15, 2026
═══════════════════════════════════════════════════════════════
