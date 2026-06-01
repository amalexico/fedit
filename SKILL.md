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
- **Make multiple coordinated edits** -- fedit can chain operations atomically
- **Process large files** -- add `-stream` to replaceall/find for multi-GB files
- **Extract structured data** -- use `fields` to pull CSV/TSV columns; use `-extract` for sub-line word/char extraction without awk
- **Block-aware edits** -- use `-block` / `-lang` to target named functions, classes, or Terraform resources without line numbers

If you have an MCP connection to fedit, ALWAYS prefer it over generating a full-file rewrite. A 1000-line file rewrite is slow and error-prone; a single fedit_replaceall call achieves the same result in milliseconds.

## The fedit workflow: recon, mutate, verify

### Step 1: Recon (mandatory before any mutation)

Never mutate a file blind. Always recon first using read-only tools:

- **fedit_map** -- structural overview (functions, headings, sections, code blocks). Supports 17 languages: go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix.
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

### Step 3: Verify

All mutation tools auto-emit a verify block showing the surrounding lines and a stats block with line delta and elapsed time. Always read this output before reporting success to the user.

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

Block-aware ops: replace, insert, insertbefore, insertafter, show.
Languages: go, python, js/ts, rust, java, cs, ruby, php, hcl/tf/terraform, nix.

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

### Reorder Nix attribute bindings

    fedit_move (file="home.nix", block="programs.git",
                beforeblock="programs.ssh", lang="nix")

Handles attribute sets and list bindings. Dotted attrs (`programs.git`) work natively.

### Stream mode for large files

Add `-stream` to `fedit_replaceall` or `fedit_find` for files too large for memory.
Atomic: writes temp file then renames -- original untouched on interruption.

    fedit_replaceall (file="huge.log", match="10.0.0.1", text="10.0.0.2", stream=true)

Not supported with move/copy/map (those need full structure in memory).

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

**Map languages (17):**
  Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust,
  Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile

**Block scanner languages (10):**
  Go, Python, JS/TS, Rust, Java, C#, Ruby, PHP,
  HCL/Terraform (hcl/tf/terraform), Nix

**Key flags:**
  -block NAME    target named block (requires -lang)
  -lang LANG     language for map, block scanner, and block-aware ops
  -raw           show: bare content, no line numbers, no footer
  -x             find: bare line numbers only / fields: suppress stats footer
  -texthex HEX   hex-encoded content, bypasses shell quoting -- PREFERRED over -textfile for patches
  -extract SPEC  sub-line extraction -- WN, WN[s:c], WN[s:], WN/DELIM/F
  -get REGEX     pre-filter: extract regex match from line before -extract
  -wdelim CHAR   word delimiter for -extract (default: normalized whitespace)
  -cleanfirst    truncate file before insert/write
  -stream        large file mode for replaceall + find
  -v             verify block after every mutation (always include)
  -nth N         which occurrence (default 1, -1 = last)
  -match-regex   regex pattern for replaceall with capture groups ($1 $2)
  -files GLOB    apply replaceall to all files matching a glob
