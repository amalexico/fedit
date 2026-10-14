# fedit — Claude Desktop Integration Guide

fedit ships as a zero-dependency MCP server. Once wired in, Claude can read,
edit, move, and refactor any file on your machine — including Terraform configs,
Go source, Python scripts, and more — using structured tool calls instead of
copy-pasting code blocks.

---

## 1. Prerequisites

- **fedit v1.10.0+** installed and on your PATH (v1.7.0 is enough for the single-file tools; map for HCL and Nix, js/ts/cs as lang, sourcefile with a file list and the Go import hint need v1.11.0)  
  Verify: `fedit` prints the usage block  
- **Claude Desktop** (macOS or Windows)  
  Download: https://claude.ai/download

---

## 2. Find your config file

| Platform | Path |
|----------|------|
| macOS    | `~/Library/Application Support/Claude/claude_desktop_config.json` |
| Windows  | `%APPDATA%\Claude\claude_desktop_config.json` |

If the file does not exist, create it.

---

## 3. Add fedit to the config

### macOS / Linux

```json
{
  "mcpServers": {
    "fedit": {
      "command": "fedit",
      "args": ["mcp"]
    }
  }
}
```

### Windows

```json
{
  "mcpServers": {
    "fedit": {
      "command": "fedit.exe",
      "args": ["mcp"]
    }
  }
}
```

If `fedit` is not on your system PATH, use the full path instead:

```json
"command": "C:\\Users\\yourname\\bin\\fedit.exe"
```

---

## 4. Restart Claude Desktop

Fully quit and relaunch. In the new conversation window you should see a
hammer icon (🔨) — click it to confirm **fedit** appears in the tool list.

---

## 5. Quick-start prompts

Once connected, try these in Claude Desktop:

**Show a file with line numbers**
> Show me main.go with line numbers

**Map a Go, Terraform or Nix file's structure**
> Map the structure of main.tf

**Move a Terraform resource**
> Move the `aws_s3_bucket.data` resource before `aws_instance.web` in infra.tf

**Reorder a Nix binding**
> Move `programs.git` after `programs.ssh` in home.nix

**Global find-and-replace with regex**
> Replace all `fmt.Println` calls in main.go with `log.Info` using fedit

**Extract a CSV column**
> Extract column 3 from data.csv using fedit

**Stream-safe edit on a large log file**
> Find all lines containing "ERROR" in app.log using stream mode

**Replace a function body by name (no line numbers)**
> Replace the `handleLogin` function body in server.go with an updated version

**Extract a sub-field from matched lines**
> Find lines matching "endpoint:" in config.yaml and extract the third word

**Edit many files at once**
> Replace `old.example.com` with `new.example.com` in every .conf file in this folder using fedit

**Copy a function into other files**
> Copy the `Helper` function from a.go into b.go and c.go, after `Init`, using fedit

**Overwrite a function in other files**
> Replace the `Helper` function in b.go and c.go with the one from a.go

**Insert a block taken from another file**
> In b.go, insert the `Helper` function from a.go after the `// helpers` comment

---

## 6. All available tools

| Tool | What it does |
|------|-------------|
| `fedit_show` | Display file with line numbers (optional range); add `raw=true` for bare content output; comma list or glob in `file` |
| `fedit_insert` | Insert content after line N |
| `fedit_delete` | Delete a line or range (`line`/`end` or `match`/`endmatch`); comma list or glob in `file` with `match` |
| `fedit_replace` | Replace a line range or named block (`block` + `lang`); comma list or glob in `file` with `match`; `dest` + `destblock` overwrite a named block in other files (`text` is still required, pass any placeholder) |
| `fedit_write` | Write or overwrite an entire file |
| `fedit_writeraw` | Write or overwrite a file with no escape expansion |
| `fedit_map` | Structural overview — 21 languages: go, html, sql, python, javascript, typescript, css, rust, java, csharp, yaml, toml, markdown, ruby, php, dockerfile, makefile, json, csv, hcl, nix (hcl and nix since v1.11.0); `lang` is detected from the extension when omitted; comma list or glob in `file` |
| `fedit_find` | Find lines matching a substring; `extract`/`get`/`wdelim` for sub-line extraction; `x=true` for bare line numbers; `stream=true` for large files; comma list or glob in `file` |
| `fedit_insertafter` | Insert content after a matched line or named block (`block` + `lang`); `sourcefile` + `sourceblock` + `lang` insert a block from another file (Go import hint since v1.11.0); comma list or glob in `file` with `match` |
| `fedit_insertbefore` | Insert content before a matched line or named block (`block` + `lang`); `sourcefile` + `sourceblock` + `lang` insert a block from another file (Go import hint since v1.11.0); comma list or glob in `file` with `match` |
| `fedit_replaceall` | Global replace; `match_regex` for regex and `textfile` for replacement text (both since v1.11.0), comma list or glob in `file` |
| `fedit_move` | Move a line range or named block to a new position; `dest` (a comma list of files) moves it into other files, each with its own anchor |
| `fedit_copy` | Copy a line range or named block; `dest` (a comma list of files) copies it into other files, each with its own anchor |
| `fedit_fields` | Extract column N from CSV/TSV/delimited file (`col`, `delim`); `x=true` suppresses the stats footer |

Tool parameters use the schema names (for example `raw`, `x`, `stream`, `match_regex`), not the CLI flag spellings. Section 7 shows the cross-file tools (`dest`, `sourcefile`) and many-files edits with examples.

---

## 7. Block-aware editing (IaC + source), many files and cross-file edits

`fedit_replace`, `fedit_insertbefore`, `fedit_insertafter`, `fedit_move`, and `fedit_copy` all accept a `block` parameter instead of line
numbers. Pass the block name and the language — fedit resolves the exact line
range automatically.

**Terraform example (fedit_move)**
```json
{
  "file": "infra.tf",
  "block": "aws_instance.web",
  "beforeblock": "aws_s3_bucket.data",
  "lang": "hcl"
}
```

**Go example (fedit_move)**
```json
{
  "file": "main.go",
  "block": "handleRequest",
  "afterblock": "handleHealth",
  "lang": "go"
}
```

**Replace a named block (fedit_replace)**
```json
{
  "file": "main.go",
  "block": "handleRequest",
  "lang": "go",
  "text": "<new function body>"
}
```

**Insert before a named block (fedit_insertbefore; fedit_insertafter inserts after it)**
```json
{
  "file": "main.go",
  "block": "handleHealth",
  "lang": "go",
  "text": "<new function>"
}
```

Supported `lang` values for block-aware ops:
`go`, `python`, `js`, `ts`, `rust`, `java`, `cs`, `ruby`, `php`,
`hcl`, `tf`, `terraform`, `nix`

The long names `javascript`, `typescript` and `csharp`, and `c#`, work too; the short names `js`, `ts` and `cs` need v1.11.0.

### Many files at once, and across files

Pass a comma list or glob in `file` and the same call runs on each file on its own, with a per-file report (`OK` or `SKIP`, then `N/M file(s) succeeded`). A file that fails is skipped and never modified, and the batch carries on. This works for `fedit_show`, `fedit_find`, `fedit_map`, `fedit_replaceall`, `fedit_insertafter`, `fedit_insertbefore`, `fedit_delete` and `fedit_replace` (with `match`; `block`, `line` and `stream` are rejected). `fedit_write` takes a single path.

**Replace in every .conf file (fedit_replaceall)**
```json
{
  "file": "*.conf",
  "match": "old.example.com",
  "text": "new.example.com"
}
```

**Copy a block into other files (fedit_copy; fedit_move works the same and removes the source)**
```json
{
  "file": "a.go",
  "block": "Helper",
  "lang": "go",
  "dest": "b.go,c.go",
  "afterblock": "Init"
}
```
`dest` is a comma list of existing files (no globs) and `file` must be one path. Each destination gets its own anchor (`after`, `before`, `aftermatch`, `beforematch`, `afterblock` or `beforeblock`; the block anchors need `lang`). A destination that fails is skipped, and a move removes the source only if at least one destination write succeeded. Cross-file transfer moves text, not imports: for Go, fedit adds an advisory note when a destination lacks a standard library import that the text uses (since v1.11.0).

**Overwrite a named block in other files (fedit_replace)**
```json
{
  "file": "a.go",
  "block": "Helper",
  "lang": "go",
  "dest": "b.go,c.go",
  "destblock": "Helper",
  "text": "unused"
}
```
`destblock` is required with `dest`, and a destination without that block is skipped, never inserted into. `text` is ignored here, but the tool schema still requires it, so pass any placeholder.

**Insert a block taken from another file (fedit_insertafter or fedit_insertbefore)**
```json
{
  "file": "b.go",
  "match": "// helpers",
  "sourcefile": "a.go",
  "sourceblock": "Helper",
  "lang": "go"
}
```
`sourcefile`, `sourceblock` and `lang` go together and replace `text`. `file` may also be a comma list or glob: the block is read once and inserted at each file's own `match` (list support since v1.11.0).

---

## 8. Cursor / Cline / Windsurf

These editors use the same MCP config format. Add the fedit entry to
whichever config file your editor reads (usually `.cursor/mcp.json` or
`.cline/mcp_settings.json`):

```json
{
  "mcpServers": {
    "fedit": {
      "command": "fedit",
      "args": ["mcp"]
    }
  }
}
```

Restart the editor after saving.

---

## 9. Troubleshooting

**fedit does not appear in the tool list**  
- Confirm `fedit mcp` exits cleanly (Ctrl-C after launch — no error output)  
- Check the config file is valid JSON (`python3 -m json.tool claude_desktop_config.json`)  
- Fully quit Claude Desktop (not just close the window) and relaunch  

**"Method not found" errors**  
- You are running an older fedit binary. Run `fedit` — the usage block must
  show writeraw, writelines, fields and move/copy. Update to v1.7.0+.

**Block not found for HCL/Nix files**  
- Pass `lang` explicitly: `"lang": "hcl"` or `"lang": "nix"`  
- For Terraform and Nix files, `fedit_map` detects the language from the extension (.tf, .hcl, .nix); block operations (`block`, `beforeblock`, `afterblock`) always need `lang`
- Check for typos: block names are matched by substring, so `aws_instance` will
  match `resource "aws_instance" "web"`. Use a longer substring if ambiguous.

**A parameter is ignored or rejected**
- `dest`, `destblock`, `sourcefile` and `sourceblock` need v1.10.0 or newer; `sourcefile` with a file list, map for hcl and nix, and `js`, `ts` and `cs` as `lang` need v1.11.0. Run `fedit` and compare the usage block with section 6.

---

## 10. Source & changelog

- Repo: https://github.com/amalexico/fedit  
- v1.5.0 release notes: HCL/Terraform + Nix block scanners, 333 tests  
- v1.6.0 release notes: `-texthex`, `writeraw`, `writelines`, `-cleanfirst`, `-x` (machine-readable find/fields)
- v1.7.0 release notes: `-raw` on show, `-extract`/`-get`/`-wdelim` (sub-line extraction), `-block`/`-lang` on replace/insertbefore/insertafter, BOM strip  
- v1.4.0 release notes: stream engine, fields op  
- v1.3.0 release notes: regex replaceall, multi-file glob  
- v1.8.1 release notes: `-matchhex`/`-endmatchhex`, multi-file `show`, `find` and `map`
- v1.9.0 release notes: multi-file mutation (comma list or glob in `file`), `-dest` cross-file copy, move and replace
- v1.10.0 release notes: MCP `dest`, `destblock`, `sourcefile` and `sourceblock`
- Unreleased (expected v1.11.0): `fedit_map` for hcl and nix (`.tf`, `.hcl`, `.nix`), `terraform` and `tf` as map aliases, `js`, `ts` and `cs` as `lang` values, `sourcefile` and `sourceblock` with a file list, the Go import hint on cross-file transfers and on `sourcefile` inserts, `match_regex` and `textfile` on `fedit_replaceall`
