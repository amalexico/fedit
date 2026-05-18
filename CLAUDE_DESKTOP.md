# fedit — Claude Desktop Integration Guide

fedit ships as a zero-dependency MCP server. Once wired in, Claude can read,
edit, move, and refactor any file on your machine — including Terraform configs,
Go source, Python scripts, and more — using structured tool calls instead of
copy-pasting code blocks.

---

## 1. Prerequisites

- **fedit v1.5.0+** installed and on your PATH  
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
      "args": ["-mcp"]
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
      "args": ["-mcp"]
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

**Map a Go or Terraform file's structure**
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

---

## 6. All available tools

| Tool | What it does |
|------|-------------|
| `fedit_show` | Display file with line numbers (optional range) |
| `fedit_insert` | Insert content after line N |
| `fedit_delete` | Delete line or range |
| `fedit_replace` | Replace line or range with new content |
| `fedit_write` | Write or overwrite an entire file |
| `fedit_map` | Structural overview — go, python, js, ts, rust, java, cs, ruby, php, html, sql, hcl, tf, terraform, nix |
| `fedit_find` | Find lines matching a substring; `stream=true` for large files |
| `fedit_insertafter` | Insert content after a matched line |
| `fedit_insertbefore` | Insert content before a matched line |
| `fedit_replaceall` | Global replace; supports regex capture groups and glob (`files`) |
| `fedit_move` | Move a line range or named block to a new position |
| `fedit_copy` | Copy a line range or named block |
| `fedit_fields` | Extract column N from CSV/TSV/delimited file |

---

## 7. Block-aware editing (IaC + source)

`fedit_move` and `fedit_copy` accept a `block` parameter instead of line
numbers. Pass the block name and the language — fedit resolves the exact line
range automatically.

**Terraform example**
```json
{
  "file": "infra.tf",
  "block": "aws_instance.web",
  "beforeblock": "aws_s3_bucket.data",
  "lang": "hcl"
}
```

**Go example**
```json
{
  "file": "main.go",
  "block": "handleRequest",
  "afterblock": "handleHealth",
  "lang": "go"
}
```

Supported `lang` values for block-aware ops:
`go`, `python`, `js`, `ts`, `rust`, `java`, `cs`, `ruby`, `php`,
`hcl`, `tf`, `terraform`, `nix`

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
      "args": ["-mcp"]
    }
  }
}
```

Restart the editor after saving.

---

## 9. Troubleshooting

**fedit does not appear in the tool list**  
- Confirm `fedit -mcp` exits cleanly (Ctrl-C after launch — no error output)  
- Check the config file is valid JSON (`python3 -m json.tool claude_desktop_config.json`)  
- Fully quit Claude Desktop (not just close the window) and relaunch  

**"Method not found" errors**  
- You are running an older fedit binary. Run `fedit` — the usage block must
  show `fields` and `move`/`copy` operations. Update to v1.5.0+.

**Block not found for HCL/Nix files**  
- Pass `lang` explicitly: `"lang": "hcl"` or `"lang": "nix"`  
- For Terraform files saved as `.tf`, fedit auto-detects — but explicit is safer  
- Check for typos: block names are matched by substring, so `aws_instance` will
  match `resource "aws_instance" "web"`. Use a longer substring if ambiguous.

---

## 10. Source & changelog

- Repo: https://github.com/amalexico/fedit  
- v1.5.0 release notes: HCL/Terraform + Nix block scanners, 333 tests  
- v1.4.0 release notes: stream engine, fields op  
- v1.3.0 release notes: regex replaceall, multi-file glob  
