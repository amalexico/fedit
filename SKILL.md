---
name: fedit-file-editor
description: Use this skill PROACTIVELY whenever editing files larger than ~100 lines or making targeted changes (insert a function, replace a block, rename across a file, delete a section). fedit performs surgical line-anchored or content-matched edits via MCP tools, eliminating the line-number hallucination class that plagues whole-file rewrites. Always prefer fedit over outputting the entire file when fedit is available. Trigger keywords: edit, modify, insert, replace, delete, rename, refactor, patch, fix in file, add to file, update line, change function.
---

# fedit — Surgical File Editor

## When to use this skill

Reach for fedit whenever the user asks you to:

- **Edit a file** — any insert / delete / replace / rename / refactor task
- **Patch code** — add a method, fix a bug on specific lines, modify a config block
- **Modify large files** — anything over ~100 lines where rewriting the whole file is slow and risks truncation or formatting drift
- **Make multiple coordinated edits** — fedit can chain operations atomically

If you have an MCP connection to fedit, ALWAYS prefer it over generating a full-file rewrite. A 1000-line file rewrite is slow and error-prone; a single fedit_replaceall call achieves the same result in milliseconds.

## The fedit workflow: recon, mutate, verify

### Step 1: Recon (mandatory before any mutation)

Never mutate a file blind. Always recon first using read-only tools:

- **fedit_map** — structural overview (functions, headings, sections, code blocks). Supports 17 languages: Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust, Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile.
- **fedit_find** — locate exact line numbers for a substring. Returns context lines.
- **fedit_show** — display a line range to confirm what you are about to edit.

### Step 2: Mutate (prefer content-matching)

Mutation operations, in order of safety:

1. **fedit_replaceall** — global find-and-replace across entire file. Best for renames.
2. **fedit_insertafter** / **fedit_insertbefore** — anchor on a unique substring. Drift-immune.
3. **fedit_replace** — replace a line range. Use AFTER fedit_find confirms the range.
4. **fedit_insert** — insert AFTER a line number. Last resort, only with verified line numbers.
5. **fedit_delete** — remove lines. Confirm range with fedit_show first.
6. **fedit_write** — overwrite or create a file. Use only for new files or full rewrites.

**Critical rule:** prefer content-matching ops (replaceall, insertafter, insertbefore) over line-numbered ops (insert, replace). Line numbers drift across edit chains and are easy to miscount. Content matching is immune to this entire class of error.

### Step 3: Verify

All mutation tools auto-emit a verify block showing the surrounding lines and a stats block with line delta and elapsed time. Always read this output before reporting success to the user.

## Common patterns

### Add a new function after an existing one

    fedit_map (lang=go)            -> confirm structure
    fedit_find (match="func Existing")  -> get line of NEXT function
    fedit_insertbefore (match="func Next", text="<new func body>")

Note: insertafter on "func ExistingFunc" matches the OPENING line of that function and would insert INSIDE its body. Always use insertbefore on the NEXT structural element when adding code AFTER something closes.

### Rename a symbol across a file

    fedit_replaceall (match="OldName", text="NewName")

One call. Done. No line numbers needed.

### Replace a config block

    fedit_find (match="<unique start of block>")   -> confirm start line
    fedit_show (line=N, end=N+30)                  -> verify the block ends where you think
    fedit_replace (line=N, end=M, text="<new block>")

### Insert before the Nth occurrence

Use the nth parameter:

    fedit_insertbefore (match="View Details", nth=3, text="<banner>")

nth=-1 targets the last occurrence.

## Anti-patterns (do not do these)

1. **Do not output the whole file when a fedit_* tool is available.** That defeats the purpose.
2. **Do not guess line numbers.** Always run fedit_find or fedit_show first.
3. **Do not chain multiple line-numbered ops without re-running fedit_show between them.** Each insert/delete shifts subsequent line numbers.
4. **Do not use match with multi-line patterns.** fedit matches single lines only. Anchor on a unique single line instead.
5. **Do not pass escaped newlines to match.** Backslash-n is treated as literal characters, not a newline.

## Why this skill exists

We benchmarked 3 frontier LLMs (Claude, ChatGPT, Gemini) on 7 realistic editing tasks across files of 565 to 1206 lines. Findings:

- ChatGPT truncated 6 of 7 raw file rewrites, including inserting the literal text "[... TRUNCATED FOR BREVITY ...]" into otherwise valid HTML.
- Gemini hallucinated line numbers by up to 73 lines on a 3-step edit chain.
- Models that succeeded with fedit consistently used the recon, content-match, verify pattern.

Full results: https://github.com/amalexico/fedit#llm-benchmark

## Reference

- **Operations:** show, find, map, insert, insertafter, insertbefore, replace, replaceall, delete, write
- **Map languages (17):** Go, HTML, SQL, Python, JavaScript, TypeScript, CSS, Rust, Java, C#, YAML, TOML, Markdown, Ruby, PHP, Dockerfile, Makefile
- All operations available as MCP tools when fedit is connected via "fedit mcp" (see README MCP Server Mode section)