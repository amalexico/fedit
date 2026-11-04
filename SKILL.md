---
name: fedit-file-editor
description: Use this skill PROACTIVELY whenever editing files larger than ~100 lines or making targeted changes (insert a function, replace a block, rename across a file, delete a section). fedit performs surgical line-anchored or content-matched edits via 14 MCP tools -- with streaming engine, block-aware ops (-block/-lang), sub-line extraction (-extract/-get), CSV/TSV fields, HCL/Terraform and Nix block mappers -- eliminating the line-number hallucination class that plagues whole-file rewrites. Always prefer fedit over outputting the entire file when fedit is available. Trigger keywords -- edit, modify, insert, replace, delete, rename, refactor, patch, fix in file, add to file, update line, change function.
---

# fedit -- Surgical File Editor

## When to use this skill

Reach for fedit whenever the user asks you to:

- **Edit a file** -- any insert / delete / replace / rename / refactor task
- **Patch code** -- add a method, fix a bug on specific lines, modify a config block
- **Modify large files** -- anything over ~100 lines where rewriting the whole file is slow and risks truncation or formatting drift
- **Make multiple coordinated edits** -- chain several calls; every call re-reads the file, so content-matching anchors stay valid across the chain
- **Process large files** -- add `-stream` to replaceall/find for multi-GB files
- **Extract structured data** -- use `fields` to pull CSV/TSV columns; use `-extract` for sub-line word/char extraction without awk
- **Block-aware edits** -- use `-block` / `-lang` to target named functions, classes, or Terraform resources without line numbers

If you have an MCP connection to fedit, ALWAYS prefer it over generating a full-file rewrite. A 1000-line file rewrite is slow and error-prone; a single fedit_replaceall call achieves the same result in milliseconds.

## The fedit workflow: recon, mutate, verify

### Step 1: Recon (mandatory before any mutation)

Never mutate a file blind. Always recon first using read-only tools:

- **fedit_map** -- structural overview (functions, headings, sections, code blocks). Supports 21 languages: go, html, sql, python, javascript, typescript, css, rust, java, csharp, yaml, toml, markdown, ruby, php, dockerfile, makefile, json, csv, hcl, nix.
- **Block scanners** for `-block`: Python, Go, JS/TS, Rust, Java, C#, Ruby, PHP, **HCL/Terraform** (`-lang hcl`/`tf`/`terraform`), **Nix** (`-lang nix`).
- **fedit_find** -- locate exact line numbers for a substring. Returns context lines. Add `-x` for bare line numbers only (machine-readable).
- **fedit_show** -- display a line range to confirm what you are about to edit. Add `-raw` for bare content with no line-number prefix and no footer -- ideal for piping output to fwencode.

### Step 2: Mutate (prefer content-matching)

Mutation operations, in order of safety:

1. **fedit_replaceall** -- global find-and-replace across entire file. Best for renames. Supports `-match-regex` with capture groups.
2. **fedit_insertafter** / **fedit_insertbefore** -- anchor on a unique substring or named block (`-block` / `-lang`). Drift-immune.
3. **fedit_replace** -- replace a line range or named block (`-block` / `-lang`). Use AFTER fedit_find confirms the range.
4. **fedit_insert** -- insert AFTER a line number. Add `-cleanfirst` to truncate the file before writing. Last resort -- only with verified line numbers.
5. **fedit_delete** -- remove lines. Confirm range with fedit_show first.
6. **fedit_write** -- overwrite or create a file. Escape sequences (`\n`, `\t`, `\\`) are processed.
7. **fedit_writeraw** -- overwrite or create a file with **no escape expansion**. Backslashes are literal. Prefer this for Go regex patterns, Windows paths, or any content that must not have backslashes interpreted.

**Critical rule:** prefer content-matching ops (replaceall, insertafter, insertbefore) over line-numbered ops (insert, replace, delete). Line numbers drift across edit chains and are easy to miscount. Content matching is immune to this entire class of error.

**Block-aware rule:** when the target is a named function, class, resource, or attribute set, always prefer `-block` / `-lang` over line numbers or `-match`. Block-aware ops survive surrounding edits without recalibration.

**Ordering rule:** when a sequence mixes line-number ops (delete, insert, replace) with content-matching ops (insertafter, insertbefore), ALWAYS run the line-number ops FIRST. Once an insertion shifts line numbers, any subsequent line-number op targets the wrong lines.

**Preview rule (since v1.12.0):** when a change is large, spans many files, or you are not sure an anchor resolves where you expect, call the changing tool with `dry_run=true` first (CLI: `-dry`). Nothing is written. The result starts with `dry-run: applied=false (no file was changed)` and shows a unified diff. Check the diff, then repeat the same call without `dry_run`. Under dry_run the verify step is skipped (`VERIFY: SKIP (dry-run)`), and the per-tool summary line is worded as a preview (Would insert ..., Would write ...): trust the applied=false line, and never report a dry run as done.

**Output rule (since v1.12.0):** when the result of fedit_show, fedit_find or fedit_map would be very large and does not need to enter the context (for example a find over many files whose list is only to be saved), pass `output_file=PATH` (and `output_append=true` to add to an existing file). The full result text goes to the file and only `result written to FILE (N lines)` comes back, so read the parts you need with a ranged fedit_show. Do not use it when you need the result to decide the next step, and never describe the content of a result file you have not read. CLI: `-outputresult FILE` (and `-outappend`).

### Step 3: Verify

All mutation tools auto-emit a verify block showing the surrounding lines and a stats block with line delta and elapsed time. Always read this output before reporting success to the user. fedit_replaceall goes further: it re-reads the written file from disk and reports VERIFY: PASS or VERIFY: FAIL, and a FAIL comes back as an error result. Never report a replaceall as done when the result says VERIFY: FAIL.

## Common patterns

### Add a new function after an existing one

    fedit_map (lang=go)                        -> confirm structure
    fedit_find (match="func Existing")         -> locate next function
    fedit_insertbefore (match="func Next", text="<new func body>")

Note: insertafter on "func ExistingFunc" matches the OPENING line of that function
and would insert INSIDE its body. Always use insertbefore on the NEXT structural
element when adding code AFTER something closes.

### Replace a named block (no line numbers needed)

    # Replace a Go function body with a new version
    fedit_replace (file="dst.go", block="OldFunc", lang="go", text="<new func body>")

    # Source the replacement from another file using show -raw + fwencode:
    fedit_show  (file="src.go", block="NewFunc", lang="go", raw=true)  -> pipe to fwencode -> $hex
    fedit_replace (file="dst.go", block="OldFunc", lang="go", texthex=$hex)

Block-aware ops: replace, insert, insertbefore, insertafter, show. Brace languages find a block end by counting braces and skip strings (including multi-line ones), comments, and Rust char literals and lifetimes (since v1.11.0). If a range still looks wrong, check it with fedit_show first and fall back to line numbers.
Languages: go, python, js/ts, rust, java, cs, ruby, php, hcl/tf/terraform, nix (the long names javascript, typescript and csharp, and c#, are accepted as well).

### Insert before/after a named block

    fedit_insertbefore (file="dst.go", block="TargetFunc", lang="go", text="<new func>")
    fedit_insertafter  (file="dst.go", block="TargetFunc", lang="go", text="<epilogue>")

### Rename a symbol across a file

    fedit_replaceall (match="OldName", text="NewName")

One call. Done. No line numbers needed.

### Replace a config block (line-number approach)

    fedit_find (match="<unique start of block>")   -> confirm start line
    fedit_show (line=N, end=N+30)                  -> verify end of block
    fedit_replace (line=N, end=M, text="<new block>")
### Show / delete / replace by content anchors (no line numbers)

    # Show everything from one function signature to another
    fedit_show (match="func Start", endmatch="func End")

    # Delete a section bounded by content anchors
    fedit_delete (match="## Section Header", endmatch="## Next Header")

    # Replace a plain-text block by content anchors
    fedit_replace (match="Version 1.6", text="Version 1.7")      # the one matched line

    # match with end=N runs from the matched line through line N
    fedit_delete (match="## Section Header", end=40)

Both anchor lines are included in the range. -nth applies to -match only.
fedit_delete and fedit_replace (and the CLI delete and replace) act on the single matched line when there is neither endmatch nor end (after v1.12.0; older versions need endmatch or end N). show, move and copy still require endmatch or end N as the end bound.

### Pass content with special characters (texthex -- preferred)

When text contains double-quotes, backticks, triple backticks, or shell-special chars,
use `-texthex` directly. This is pure fedit -- no temp files needed.

    # Encode inline (fwencode strips UTF-8 BOM automatically):
    $hex = fwencode "content with `"quotes`" or \backslashes\"
    fedit_insertbefore (match="func Foo", texthex=$hex)

    # Encode from show -raw pipeline (block-to-block transfer):
    $hex = fedit -file src.go -op show -block "Func" -lang go -raw 2>$null | fwencode
    fedit_replace (file="dst.go", block="Func", lang="go", texthex=$hex)

    # Pre-compute hex and pass directly -- no WriteAllBytes, no temp file:
    $h = '<precomputed_hex_string>'
    fedit_insertbefore (match="anchor line", texthex=$h)

NOTE: WriteAllBytes + textfile is only needed when the hex string exceeds PS
argument length limits (rare, >32KB). For all normal patches, use -texthex directly.

### Anchors that contain double quotes (matchhex)

PowerShell mangles a double quote inside -match/-endmatch before fedit sees it
(-match 'case "replace":' searches for: case replace:). On the CLI, use -matchhex
and -endmatchhex: the anchor goes in as hex and is decoded into -match/-endmatch.

    $m = [BitConverter]::ToString([Text.Encoding]::UTF8.GetBytes('case "replace":')).Replace('-','').ToLower()
    fedit -file main.go -op find -matchhex $m -x

A quote-free anchor is still fine when one exists.

### Anchor selection for insertbefore/insertafter

Always anchor on the LAST LINE OF CONTENT in a section, not on headings or separators.

    # WRONG -- separator sits between content and heading; prompt lands after ---
    fedit_insertbefore (match="## 6. All available tools", text="<new content>")

    # CORRECT -- anchor on the last line of actual content before the separator
    fedit_insertafter (match="> last prompt in section", text="<new content>")

If you must anchor near a heading, use insertafter on the content line ABOVE the
separator, not insertbefore on the heading BELOW it.

### Extract a sub-line value (-extract / -get / -wdelim)

`-extract` navigates the File -> Line -> Word -> Char hierarchy without awk:

    # Get word N from a matched line (normalized whitespace, 1-based like awk $N)
    fedit_find (match="server_address", extract="W3")

    # Slice chars from a word: WN[start:count] (1-based start, count chars)
    fedit_find (match="v1.6.0", extract="W2[1:4]")         -> first 4 chars of word 2

    # Slice to end of word: WN[start:]
    fedit_find (match="PATH", extract="W5[3:]")             -> word 5 from char 3 to end

    # Split a word on a delimiter: WN/DELIM/FIELD
    fedit_find (match="09876154", extract="W7/./1")         -> part before first dot
    fedit_find (match="09876154", extract="W7/./2")         -> part after first dot

    # Pre-filter with -get (isolate a regex match before -extract)
    fedit_find (match="config", get="\d+\.\d+", extract="W1[1:3]")

    # Custom word delimiter via -wdelim
    fedit_find (match="PATH=", wdelim=":", extract="W2")    -> second colon-delimited token

Use `-x` flag to return bare line numbers only (machine-readable, no context lines).

### Get bare file content for piping (-raw on show)

    fedit_show (file="src.go", block="FuncName", lang="go", raw=true)

`-raw` suppresses line-number prefixes and the footer. Designed for piping to fwencode
or other tools. Available on `show` only.

### Insert before the Nth occurrence

    fedit_insertbefore (match="View Details", nth=3, text="<banner>")

nth=-1 targets the last occurrence.

### Move a block before/after another block

    fedit_find (match="func Target")    -> confirm line numbers
    fedit_move (line=N, end=M, beforematch="func Destination")

For Python class reordering, content-match both bounds:

    fedit_move (match="class A", endmatch="class B", beforematch="class C")

For Terraform resource reordering, use block-aware move:

    fedit_move (file="main.tf", block='resource "aws_instance" "web"',
                beforeblock='resource "aws_s3_bucket" "data"', lang="hcl")

**Move overlap rule:** destination inside source range is always rejected.

**Cross-file copy/move:** `-dest F1,F2` copies or moves the source block into other files. Each destination needs its own anchor flag (-after, -before, -aftermatch, -beforematch, -afterblock, -beforeblock; the block anchors need -lang), resolved inside that destination file:

    fedit -file a.go -op copy -block Helper -lang go -dest b.go,c.go -afterblock Init -v
    fedit -file base.conf -op move -match '[cache]' -endmatch '[/cache]' -dest x.conf,y.conf -beforematch '[logging]'

- Per-destination policy: a problem in one destination is a SKIP for that file and never aborts the batch. Exit code 0 if at least one destination succeeded, 1 if none did.
- move writes all destinations first and removes the source only if at least one write succeeded. If every destination fails the source stays intact.
- copy and move take -dest with an anchor flag; replace takes -dest with -destblock (below). It is rejected with -match-regex, -cleanfirst, -stream, -files or a multi -file list. The MCP tools take the same thing as parameters: dest and destblock on fedit_copy, fedit_move and fedit_replace, sourcefile and sourceblock on fedit_insertafter and fedit_insertbefore.
- It moves text, not dependencies. After a Go transfer, check imports in the destination: fedit adds an advisory note when a Go destination lacks a standard library import that the transferred text uses (results and exit codes are unchanged).

**Replace a block in other files:** `replace -dest` overwrites a named block in each destination with the source block or range from `-file`. `-destblock` is required:

    fedit -file a.go -op replace -block Helper -lang go -dest b.go,c.go -destblock Helper -v

- A destination without that block is a SKIP, never an insert. -destblock is only accepted with replace -dest.

**Insert a block from another file:** insertafter and insertbefore can take their content from a block in another file. -file stays the file being edited:

    fedit -file b.go -op insertafter -match '// helpers' -sourcefile a.go -sourceblock Helper -lang go -v

- -sourcefile and -sourceblock must be used together and need -lang. They cannot combine with -text, -textfile or -texthex. -file may be a comma list or glob: the block is read once and inserted into every file at its own -match, with a per-file report. When both files are Go, a standard library import the destination lacks is flagged with an advisory hint (the insert is still done); the hint goes to stderr on the CLI and is appended to the OK message over MCP.

### Edit many files at once (comma list or glob in file)

    fedit_replaceall (file="*.conf", match="old.example.com", text="new.example.com")
    fedit_insertafter (file="a.yaml,b.yaml", match="servers:", text="  - new")
    fedit_delete (file="*.md", match="<!-- start -->", endmatch="<!-- end -->")
    fedit_find (file="*.go", match="TODO")

- Each file is its own source: a problem in one (no match, unreadable, read-only, write failed) is a SKIP for that file, the batch goes on, and a skipped file is never modified. The report lists OK, SKIP or FAIL per file, then N/M file(s) succeeded. For replaceall each written file is re-read from disk; a FAIL means the file was written but did not verify. Exit code 0 if at least one file succeeded and none failed, 1 if none succeeded, 3 if any file is FAIL. Over MCP the result is an error if any file is FAIL or none succeeded.
- Accepted: show, find and map, plus replaceall (also match_regex), insertafter, insertbefore, delete and replace with match, and write (same content to every file). block, line, stream, files, move and copy are rejected before any file is read. fedit_write over MCP takes one path.
- This is not -dest. A multi file edit applies the same match in every file; -dest moves text from one source file into other files. They never combine.

### Duplicate a block N times (scaffolding / test fixtures)

    fedit_copy (line=N, end=M, after=InsertPoint, times=10)

Snapshot semantics: all 10 copies are identical clones of the original block
at read time -- even if destination overlaps source range.

### Extract a CSV/TSV column (fields op)

    fedit_fields (file="data.tsv", col=2, delim="\t")

Output goes to stdout for piping. For CSV: `delim=","`. Default is tab.
Lines shorter than the requested column are skipped silently.
Add `-x` to suppress the stats footer (machine-readable output).

### Reorder Terraform blocks (HCL)

    fedit_move (file="main.tf",
                block='resource "aws_instance" "web"',
                beforeblock='resource "aws_s3_bucket" "data"',
                lang="hcl")

All top-level Terraform block types supported. Accepts lang: `hcl`, `tf`, `terraform`.
Nested blocks (ingress, lifecycle) are ignored -- only top-level blocks matched.
Recon first with fedit_map on the .tf file: it lists the top-level blocks by kind with line ranges and flags duplicate resource, data, module, variable and output blocks (.tf and .hcl are detected from the extension).

### Reorder Nix attribute bindings

    fedit_move (file="home.nix", block="programs.git",
                beforeblock="programs.ssh", lang="nix")

Handles attribute sets and list bindings. Dotted attrs (`programs.git`) work natively.
fedit_map on a .nix file lists the unindented top-level attributes with line ranges and flags a path defined twice; indented attributes inside a function body are not listed, the same rule -block uses.

### Stream mode for large files

Add `-stream` to `fedit_replaceall` or `fedit_find` for files too large for memory.
Atomic: writes temp file then renames -- original untouched on interruption.

    fedit_replaceall (file="huge.log", match="10.0.0.1", text="10.0.0.2", stream=true)

Not supported with move/copy/map (those need full structure in memory).

### Preview a change with dry_run (since v1.12.0)

Add `dry_run=true` to any of the ten tools that change files (fedit_insert, fedit_delete, fedit_replace, fedit_write, fedit_writeraw, fedit_replaceall, fedit_insertafter, fedit_insertbefore, fedit_move, fedit_copy) and nothing is written, truncated or created. The result starts with `dry-run: applied=false (no file was changed)`, then a unified diff of each file that would change (a new file is shown against /dev/null). A call that would fail is still an error.

    fedit_replaceall (file="*.conf", match="old.example.com", text="new.example.com", dry_run=true)

- Multi-file and dest calls give one diff per file and a total that reads "N/M file(s) would succeed", and each file line is worded as a preview, for example "would replace on 1 line(s)".
- The CLI summary of a single-file -dry run is worded as a preview too (Would insert 1 line(s) after line 1 (4 total after)), like the MCP summary.
- Verify is skipped, so there is no VERIFY: PASS to report. Say that the change was previewed, not applied, then repeat the call without dry_run to apply it.
- CLI: -dry on the command line, and the word dry at the end of a `fedit guided` command. -dry cannot be combined with -config or -log merge.
- The read-only tools (fedit_show, fedit_find, fedit_map, fedit_fields) have no dry_run parameter.
- A dry run is logged like any run; its end record carries dry=true.

### Send a result to a file with output_file (since v1.12.0)

Every MCP tool (all 14) takes an optional `output_file` (string) and `output_append` (boolean). The full result text (all content, UTF-8, no BOM) is written to the file, overwritten or appended, and the tool returns only `result written to FILE (N lines)` or `result appended to FILE (N lines)`. The operation itself runs as usual, so a changing tool still changes its files; with dry_run the dry-run text is what is written.

    fedit_find (file="*.go", match="TODO", output_file="todo.txt")

- Refused before the tool runs, nothing written: output_append without output_file; a result file that is also an edited file (file, dest or a glob match); a result file whose folder does not exist.
- A call that fails returns its error in full and leaves the result file untouched. If the file cannot be written after the operation succeeded, the error says so and carries the text: do not repeat the edit.
- A relative path resolves against the working directory of the MCP server, like every other path. Use an absolute path when unsure.
- Only the summary line comes back: read the file (a ranged fedit_show) before you say what it contains.
- CLI: -outputresult FILE and -outappend.

### Audit log (since v1.12.0, off by default)

fedit can write an audit record per operation: a begin line and an end line that share an id. It is off unless the user turns it on. Do not enable it or pass -log-* flags yourself unless asked.

- Settings: -log-mode off|permissive|strict, -log-file, -log-max-mb, -log-content on|off, -log-level all|warning|error, -log-scope all|writes. Each has an environment variable FEDIT_LOG_*. Order: flag, environment, settings file, default. Over MCP there are no flags: the environment and the settings file are read, so put FEDIT_LOG_* (and FEDIT_CONFIG, which names another settings file) in the server's env block, or the settings in the settings file.
- Settings file (since v1.12.0): an optional fedit.ini of `key = value` lines (log_mode, log_file, log_max_mb, log_content, log_level, log_scope; # or ; starts a comment) in <user config folder>/fedit/fedit.ini, or the file named by FEDIT_CONFIG. A missing file is fine. An unreadable file, a bad line or an invalid value is an error that names the file and line (exit 1, nothing edited); an unknown key is a warning on stderr. -noconfig skips it on the command line. Do not create or edit it, and do not run fedit -config set or unset, unless asked; fedit -config show prints the effective values.
- Every operation is logged, reads included, unless the scope is writes. -text values are logged as length and SHA-256 unless -log-content is on. A -textfile is logged by path plus the length and SHA-256 of its content (textfile_len, textfile_sha256), never the content.
- permissive: the edit always runs. If the log cannot be written, stderr says "The action was not logged" (over MCP the tool result carries it too, as an extra text item) and the records are spooled; the next logged run, or `fedit -log merge`, appends them to the log.
- strict, exit 4: the begin record could not be written, so the run was refused and nothing changed. Report it to the user.
- strict, exit 5: the edit succeeded but the end record could not be written. The file IS changed, so do not repeat the edit; tell the user the log has a gap. Over MCP this comes back as an error result that says the operation completed.
- The end record carries exit, ms and, where available, lines_changed (every plain write operation and replaceall; none for a dry run or a failed call), replacements (occurrences replaced; replaceall only, CLI and MCP), verify, per-file results and sha256_before/sha256_after.
- A dry run (dry_run=true or -dry) is logged like any run: its end record carries dry=true, notes no replacements and no lines_changed, and the two hashes are equal.
- A call with output_file is logged like any call: its begin record also carries output_file (an absolute path) and, when it is set, output_append=true.

## Anti-patterns (do not do these)

1. **Do not output the whole file when a fedit_* tool is available.** That defeats the purpose.
2. **Do not guess line numbers.** Always run fedit_find or fedit_show first.
3. **Do not chain multiple line-numbered ops without re-running fedit_show between them.** Each insert/delete shifts subsequent line numbers. Use -block/-lang or -match to stay drift-immune.
4. **Do not use match with multi-line patterns.** fedit matches single lines only. Anchor on a unique single line instead.
5. **Do not pass escaped newlines to match.** Backslash-n is treated as literal characters, not a newline.
6. **Do not use fedit_write when fedit_writeraw is needed.** If your text contains literal backslashes (Go regex, Windows paths, raw string literals), fedit_write will corrupt them.
7. **Do not skip -raw when piping show output to fwencode.** Without -raw the line-number prefixes and footer become part of the encoded content.
8. **Do not use WriteAllBytes + textfile when -texthex works.** Pre-compute the hex and pass it directly via -texthex. Temp files are only necessary when the hex string exceeds PS argument length limits (~32KB).
9. **Do not run line-number ops after insertions in the same sequence.** Insertions shift all subsequent line numbers. Always run delete/replace-by-line BEFORE insertafter/insertbefore in the same patch sequence.
10. **Do not anchor insertbefore on headings or separators when content precedes them.** A `---` or `## N.` heading anchor places content after the separator, outside the intended section. Anchor on the last line of actual content with insertafter instead.
11. **Do not combine -dest with a comma list or glob in -file, and do not expect -dest to carry imports.** Multi -file and -dest never combine; after a Go transfer add the missing import in the destination yourself (the hint tells you which).
12. **Do not pass block, line, end, stream or files together with a comma list or glob in file.** They are rejected; use match (plus endmatch) so each file resolves its own anchor.
13. **Do not describe a result file you have not read.** With output_file only a one-line summary comes back; read the file (a ranged fedit_show) before you report what is in it, and do not point output_file at a file you are editing.

## Why this skill exists

We benchmarked 3 frontier LLMs (Claude, ChatGPT, Gemini) on 7 realistic editing tasks across files of 565 to 1206 lines. Findings:

- ChatGPT truncated 6 of 7 raw file rewrites, including inserting the literal text "[... TRUNCATED FOR BREVITY ...]" into otherwise valid HTML.
- Gemini hallucinated line numbers by up to 73 lines on a 3-step edit chain.
- Models that succeeded with fedit consistently used the recon, content-match, verify pattern.

Full results: https://amalexhandler.com/fedit#benchmark

## Reference

**Operations (15):**
  show, find, map, insert, insertafter, insertbefore,
  replace, replaceall, delete, write, writeraw, writelines,
  move, copy, fields

**MCP tools (14):**
  fedit_show, fedit_find, fedit_map, fedit_insert,
  fedit_insertafter, fedit_insertbefore, fedit_replace,
  fedit_replaceall, fedit_delete, fedit_write, fedit_writeraw,
  fedit_move, fedit_copy, fedit_fields
  (writelines is interactive stdin only -- no MCP tool)
  Every MCP tool also takes output_file (string) and output_append (boolean) (since v1.12.0): the full result goes to the file and only a one-line summary comes back

**Map languages (21):**
  Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust,
  Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile, JSON, CSV, HCL, Nix

**Block scanner languages (10):**
  Go, Python, JS/TS, Rust, Java, C#, Ruby, PHP,
  HCL/Terraform (hcl/tf/terraform), Nix

**Key flags:**
  -block NAME    target named block (requires -lang)
  -lang LANG     language for map, block scanner, and block-aware ops
  -raw           show: bare content, no line numbers, no footer
  -x             find: bare line numbers only / fields: suppress stats footer
  -texthex HEX   hex-encoded content, bypasses shell quoting -- PREFERRED over -textfile for patches
  -matchhex HEX  hex-encoded -match anchor, for anchors containing double quotes
  -endmatchhex HEX  hex-encoded -endmatch anchor
  -extract SPEC  sub-line extraction -- WN, WN[s:c], WN[s:], WN/DELIM/F
  -get REGEX     pre-filter: extract regex match from line before -extract (over MCP an invalid regex is an error, since v1.11.0)
  -wdelim CHAR   word delimiter for -extract (default: normalized whitespace)
  -cleanfirst    truncate file before insert/write
  -stream        large file mode for replaceall + find
  -v             verify block after every mutation (always include); replaceall always verifies the written file, -v adds a window for every hit
  -nth N         which occurrence (default 1, -1 = last)
  -match-regex   regex pattern for replaceall with capture groups ($1 $2)
  -files GLOB    apply replaceall to all files matching a glob (every other op takes a comma list or glob in -file)
  -line N:+M     lines N to N+M inclusive (N:+0 = N only); -N counts from end; -N: = last N lines; : = EOF/append
  -end -N        end line relative to EOF (-1 = last line, -5 = fifth from end)
  -endmatch TEXT content-anchor end of range -- show, replace, delete, move, copy (optional for replace and delete: -match alone = the matched line, after v1.12.0)
  -quiet         suppress output on success, including the per-file batch report (skips and errors still print); exit code signals result (wins over -v)
  -outputresult FILE  write the operation report to FILE (UTF-8, no BOM), one summary line on the terminal; FILE may not be an edited file; nothing is written on a failed run
  -outappend     with -outputresult: append to FILE instead of overwriting it
  -log-mode M    audit log mode: off (default), permissive or strict; also -log-file, -log-max-mb, -log-content on|off, -log-level, -log-scope, each with an environment variable FEDIT_LOG_* (since v1.12.0, see Audit log)
  -log merge     append the spooled audit records to the main log now (since v1.12.0)
  -noconfig      do not read the settings file (default <user config folder>/fedit/fedit.ini; env FEDIT_CONFIG names another file) (since v1.12.0)
  -config ARGS   path | show | get KEY | set KEY VALUE | unset KEY: show or change the settings file (set and unset are logged, not over MCP) (since v1.12.0)
  -times N       move/copy: paste N times (move cuts once)
  -after N, -before N, -aftermatch TEXT, -beforematch TEXT, -afterblock NAME, -beforeblock NAME   move/copy destination anchors (the block anchors need -lang)
  -col N         fields: column number (1-based); -delim CHAR is the field delimiter (default tab)
  -file LIST     CLI: comma list and/or glob; show/find/map, plus replaceall (also -match-regex), insertafter, insertbefore, delete and replace with -match, and write (per-file SKIP, never abort); -block, -line, move, copy and -stream rejected. MCP: fedit_show/find/map/replaceall/insertafter/insertbefore/delete/replace take a list too; fedit_write and the other tools take one file
  -dest F1,F2    copy/move into other files; each destination needs its own anchor flag; per-destination SKIP, source removed only if a write succeeded; MCP: the dest parameter
  -destblock NAME  replace -dest only: block in each destination to overwrite (required); a destination without it is a SKIP
  -sourcefile F    with -sourceblock NAME and -lang, insertafter/insertbefore take content from that block of F; not with -text/-textfile/-texthex; -file may be a comma list or glob (per-file report, Go import hint)
