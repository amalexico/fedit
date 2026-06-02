# fedit -- Session Status File
# Upload alongside user_skill.md at the start of any fedit-focused chat.
# Last updated: June 1, 2026

═══════════════════════════════════════════════════════════════
FEDIT -- PROJECT STATE
═══════════════════════════════════════════════════════════════

REPO:    github.com/amalexico/fedit (PUBLIC -- MIT)
PATH:    C:\Users\kehsi\Desktop\amalex-brand\fedit\
WEBSITE: amalexhandler.com/fedit (LIVE -- current: v1.7.0, needs v1.8.0 update)
INSTALL: go install github.com/amalexico/fedit@latest
STARS:   8 (as of May 4)

CURRENT TAG:  v1.8.0 (PUSHED)
LATEST COMMITS (v1.8.0):
  c2860a1  feat: mcp.go -- endmatch+match+negative indices for show/delete/replace
  a288cc4  docs: update SKILL.md and README for v1.8.0
  757d5f5  feat: colon line syntax + negative indices for -line/-end (F2)
  9e30767  feat: endmatch+quiet support for show/delete/replace (F1+F3)

═══════════════════════════════════════════════════════════════
FEDIT -- PENDING TASKS
═══════════════════════════════════════════════════════════════

IMMEDIATE (do in this order):
  [X] 1. Rebuild fwencode.exe (BOM fix)             DONE (May 30, 2026)
  [X] 2. Tag v1.7.0                                  DONE (May 31, 2026)
  [X] 3. Update README.md                            DONE (May 31, 2026, commit d9c1cf9)
  [X] 4. Update SKILL.md                             DONE (May 29, 2026)
  [X] 5. Update CLAUDE_DESKTOP.md                    DONE (May 29, 2026)
  [X] 6. Update CLAUDE_DESKTOP.md (dup -- n/a)
  [X] 7. Update fedit.html on website for v1.7.0     DONE (May 31, 2026, deployed)
  [ ] 8. Post r/devops v1.7.0 update                 TODO

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

MAP LANGUAGES (17):
  Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust,
  Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile

KEY FLAGS (v1.8.0):
  -block NAME + -lang LANG   target named block (no line numbers)
  -raw                       show: bare content for piping to fwencode
  -texthex HEX               hex-encoded content (bypasses PS quoting)
                             PREFERRED: pass $h directly, no WriteAllBytes needed
                             flag.String type (was flag.Bool -- fixed in 50cccc9)
  -extract SPEC              sub-line extraction (see hierarchy below)
  -get REGEX                 pre-filter line before -extract
  -wdelim CHAR               word delimiter for -extract (default: whitespace)
  -x                         find: bare line numbers / fields: no stats footer
  -cleanfirst                truncate file before insert/write
  -stream                    large file mode (replaceall + find)
  -v                         verify after every mutation (ALWAYS USE)
  -nth N                     occurrence selector (default 1, -1 = last)
  -match-regex P             regex pattern with capture groups ($1 $2) for replaceall
  -files GLOB                apply replaceall across matching files
  -line N:+M               colon range: N to N+M (M additional lines)
  -line -N                 N-th line from end of file
  -line -N:                last N lines (from -N to EOF)
  -line :                  EOF -- append for insert, last line for show/delete/replace
  -end -N                  end line relative to EOF
  -endmatch TEXT           content-anchor end of range (show, replace, delete, move, copy)
  -quiet                   suppress stdout on success; exit code signals result (wins over -v)

EXTRACT HIERARCHY (File -> Line -> Word -> Char):
  WN           word N (normalized whitespace, 1-based)
  WN[s:c]      word N, chars s to s+c-1 (1-based start, count c)
  WN[s:]       word N, from char s to end
  WN/DELIM/F   word N, split by DELIM, take field F

WRITE VS WRITERAW:
  fedit_write    -- processes \n \t \\ escape sequences
  fedit_writeraw -- backslashes literal (use for Go regex, Windows paths)

═══════════════════════════════════════════════════════════════
FEDIT -- CRITICAL RULES (LEARNED FROM PRACTICE)
═══════════════════════════════════════════════════════════════

TEXTHEX RULE:
  Use -texthex $h directly for hex patches. NO temp files needed.
  $h='<hex>' ; fedit -file f -op insertbefore -match 'anchor' -texthex $h -v
  WriteAllBytes + textfile only when hex string > ~32KB (very rare).

ORDERING RULE:
  Line-number ops (delete, replace -line) MUST run BEFORE content-matching
  ops (insertafter, insertbefore) in the same patch sequence.
  Once an insertion shifts line numbers, delete -line N hits the wrong lines.

ANCHOR RULE:
  NEVER anchor insertbefore on section headings or --- separators when
  content precedes them. A "---" sits BETWEEN the content and the heading.
  insertbefore '## 6.' puts content AFTER the ---, outside the section.
  CORRECT: insertafter 'last content line in section'

BINARY BYTES RULE:
  Files can contain non-printable bytes invisible in text editors.
  Detect with: python3 -c "data=open('f','rb').read(); print(data[idx:idx+20].hex())"
  Match them in PS with: "$([char]0x0c)" in double-quoted fedit -match argument.

VERIFY RULE:
  Always read the === VERIFY === block before running the next command.
  The verify shows surrounding context -- placement errors are visible immediately.
  If something looks off: stop, diagnose, fix before continuing.

═══════════════════════════════════════════════════════════════
FEDIT -- BENCHMARK PROJECT
═══════════════════════════════════════════════════════════════

STATUS: COMPLETE -- results live on amalexhandler.com/fedit
STATUS: COMPLETE -- README.md ## LLM Benchmark section added (May 31, 2026, commit d9c1cf9)

TEST FILES PATH: C:\Users\kehsi\Desktop\amalex-brand\test_files\
  test_files\
  ├── analytics_700.py       (682 lines)
  ├── config_800.yaml        (1059 lines)
  ├── dashboard_900.html     (891 lines)
  ├── engine_1200.go         (1196 lines)
  ├── processor_600.go       (575 lines)
  ├── styles_500.css         (565 lines)
  ├── system_1000.go         (980 lines)
  ├── _patch_t1.txt through _patch_t7.txt (fedit patch files)
  ├── originals\             (untouched backup copies + originals.zip)
  └── results\
      ├── *_GROUND_TRUTH.*   (T3 css, T4 go, T5 py, T6 html, T7 go)
      ├── T1_claude_raw.go, T1_chatgpt_raw.go
      ├── T2_claude_raw.yaml
      ├── T3_raw_{claude,chatgpt,gemini}.css
      ├── T4_raw_{claude,chatgpt,gemini}.go
      ├── T5_raw_{claude,chatgpt,gemini}.py
      ├── T6_raw_{claude,chatgpt,gemini}.html
      └── T7_raw_{claude,chatgpt,gemini}.go

BENCHMARK RESULTS SUMMARY:

  Claude:  7/7 raw PASS   |  4/7 fedit PASS + 3 PARTIAL
  ChatGPT: 1/7 raw PASS   |  2/7 fedit PASS + 1 PARTIAL + 4 FAIL
           (6 of 7 raw tests truncated)
  Gemini:  7/7 raw PASS*  |  1/7 fedit PASS + 6 FAIL

PER-TEST RESULTS TABLE:
  Test  File                 Task                          CLraw  CLfedit  GPTraw  GPTfedit  GMraw   GMfedit
  T1    processor.go (575)   Insert method after method    PASS   PARTIAL  FAIL    PARTIAL   PASS    FAIL
  T2    config.yaml  (1059)  Replace 24-line block         PASS   PASS     FAIL    FAIL      PASS    FAIL
  T3    styles.css   (565)   Find + delete CSS rule        PASS   PASS     PASS*   FAIL      PASS*   FAIL
  T4    system.go    (980)   Global rename (36 occ)        PASS   PASS+    FAIL    PASS+     PASS*   PASS+
  T5    analytics.py (682)   3-step chain                  PASS   PARTIAL  FAIL    FAIL      PASS    FAIL
  T6    dashboard.html(891)  Insert before 3rd match       PASS   PARTIAL  FAIL    PASS+     PASS    FAIL
  T7    engine.go    (1196)  Map + targeted insert         PASS   PASS     FAIL    FAIL      PASS    FAIL

AWK/SED HONEST COMPARISON (for README narrative):
  What awk/sed do that fedit doesn't (honestly):
    - Stream processing of multi-GB files (fedit loads whole file -- use -stream for replaceall/find)
    - Arithmetic on content (awk '{sum += $1}')
    - Multi-file glob in one native call (fedit has -files for replaceall)
    - Regex capture group replacement -- fedit HAS -match-regex with $1 $2 (partial coverage)
  What fedit does that awk/sed can't:
    - Move/copy blocks of lines to new positions
    - Insert anchored to structural elements (insertafter/before/nth)
    - Block-aware ops (-block "class Foo" -lang python)
    - Atomic single-write (no partial file states on crash)
    - MCP tool interface for LLM agents
    - -v verification output
    - Sub-line extraction without separate awk call (-extract/-get)

═══════════════════════════════════════════════════════════════
FEDIT -- README BENCHMARK SECTION (DONE -- May 31, 2026)
═══════════════════════════════════════════════════════════════

═══════════════════════════════════════════════════════════════
FEDIT -- 14 BENCHMARK PROMPTS (REFERENCE)
═══════════════════════════════════════════════════════════════

NOTE: All raw LLM tests complete. Prompts kept for future model testing.

FEDIT PROMPT HEADER (same for all 7 fedit prompts):
---
You have a CLI tool called fedit for precise file editing. Available operations:
  fedit -file FILE -op show [-line N] [-end N]
  fedit -file FILE -op find -match "text" [-nth N]
  fedit -file FILE -op map -lang LANG
  fedit -file FILE -op insert -line N -text "content"
  fedit -file FILE -op delete -line N [-end N]
  fedit -file FILE -op replace -line N -end N -text "content"
  fedit -file FILE -op insertafter -match "text" [-nth N] -text "content"
  fedit -file FILE -op insertbefore -match "text" [-nth N] -text "content"
  fedit -file FILE -op replaceall -match "old" -text "new"
Flags: -v (verify) | -nth N (Nth occurrence) | -textfile F (content from file)
---

T1 (575L) -- Insert method after struct method
T2 (1059L) -- Replace 24-line K8s deployment block
T3 (565L) -- Find + delete CSS rule
T4 (980L) -- Global rename 36 occurrences (FetchUser -> GetAccount)
T5 (679L) -- 3-step chain (insert method, delete class, replace main block)
T6 (891L) -- Insert before 3rd occurrence (-nth 3)
T7 (1196L) -- Map + targeted insert after specific struct method

═══════════════════════════════════════════════════════════════
FEDIT -- FILES IN REPO
═══════════════════════════════════════════════════════════════

main.go          (3357 lines -- primary source)
mcp.go           (845 lines)
README.md        (needs ## LLM Benchmark section)
SKILL.md         (updated May 29 -- v1.7.0 features + new anti-patterns)
CLAUDE_DESKTOP.md (updated May 29 -- v1.7.0 features, committed)
go.mod, LICENSE, .gitignore
Test files:
  mcp_test.go, texthex_test.go (34), extract_test.go (40), block_test.go (22),
  iac_test.go, move_copy_test.go, regex_glob_test.go, stream_fields_test.go
Demo:
  demo.gif, demo.tape, demo_sample.go

═══════════════════════════════════════════════════════════════

===============================================================
FEDIT -- NEXT VERSION TODO
===============================================================

SHIPPED IN v1.8.0 (June 1, 2026):
  [X] 1. Anchor-to-anchor replace (-match/-endmatch on show/replace/delete + MCP)
  [X] 2. Colon line syntax (-line N:+M, -N, -N:, : and -end -N)
  [X] 3. Quiet mode (-quiet flag)

PENDING v1.8.0:
  [ ] 1. Update fedit.html on website for v1.8.0    TODO
  [ ] 2. Post r/devops v1.8.0 update                TODO
  [ ] 3. Add tests for new flags                    TODO

END OF FEDIT STATUS -- June 1, 2026
═══════════════════════════════════════════════════════════════
