# fedit — Fast File Editor for the Terminal

A zero-dependency CLI tool for **surgical file edits** from the command line.
No interactive editors. No sed/awk gymnastics. Just simple, predictable operations
with built-in verification.

Built for sysadmins, DevOps engineers, and anyone who scripts config changes.

```
go install github.com/amalexico/fedit@latest
```

---

## Why fedit?

- **One binary, zero dependencies** — pure Go, runs everywhere
- **17 language mappers** — see the structure of any file before editing
- **-v flag** — verify every mutation before moving on
- **Line-aware** — no regex surprises, no "which match did it hit?"
- **Safe** — no in-place unless you say so, never touches files you did not name

---

## Install

**Go install (recommended):**
```bash
go install github.com/amalexico/fedit@latest
```

Or download the binary from [GitHub Releases](https://github.com/amalexico/fedit/releases) and put it in your PATH.

**Verify:**
```bash
fedit -file /etc/hostname -op show
```

---

## Quick Start

```bash
# See what is in a file
fedit -file config.yaml -op show

# See just lines 10-25
fedit -file config.yaml -op show -line 10 -end 25

# Find every line containing "timeout"
fedit -file config.yaml -op find -match "timeout"

# See the structure of a Go file
fedit -file main.go -op map -lang go

# Replace line 42 with new content
fedit -file config.yaml -op replace -line 42 -end 42 -text "timeout: 60s" -v

# Insert a line after every occurrence of "server {"
fedit -file nginx.conf -op insertafter -match "server {" -text "    include security.conf;" -v
```

---

## All Operations

### show — Display file contents

```bash
# Entire file
fedit -file app.conf -op show

# Lines 50-75 only
fedit -file app.conf -op show -line 50 -end 75
```

---

### find — Search for lines matching a substring

```bash
# Find all lines containing "ERROR"
fedit -file /var/log/app.log -op find -match "ERROR"

# Output includes context lines and occurrence numbers
# Use -nth to target a specific match in other operations
```

**Pro tip:** Run find first to get line numbers, then use replace or delete with exact lines.

---

### insert — Insert content after a line number

```bash
# Insert a comment after line 1
fedit -file script.sh -op insert -line 1 -text "# Added by deploy script" -v

# Insert multiple lines from a file
fedit -file config.yaml -op insert -line 10 -textfile extra-config.yaml -v
```

---

### insertafter — Insert after a matching line (RECOMMENDED)

```bash
# Add a firewall rule after the matching comment
fedit -file iptables.rules -op insertafter -match "# Custom rules" -text "-A INPUT -p tcp --dport 8080 -j ACCEPT" -v

# Target the 2nd occurrence
fedit -file nginx.conf -op insertafter -match "server {" -nth 2 -text "    listen 8443 ssl;" -v

# Target the last occurrence
fedit -file docker-compose.yml -op insertafter -match "volumes:" -nth -1 -text "      - /data:/data" -v
```

---

### insertbefore — Insert before a matching line (RECOMMENDED)

```bash
# Add a header before the first route definition
fedit -file routes.rb -op insertbefore -match "get '/'" -text "  # === Public Routes ===" -v

# Insert a dependency before the closing bracket
fedit -file package.json -op insertbefore -match "}" -nth -1 -textfile new-deps.txt -v
```

---

### replace — Replace a line range with new content

```bash
# Replace a single line
fedit -file config.ini -op replace -line 15 -end 15 -text "max_connections = 200" -v

# Replace lines 30-35 with content from a patch file
fedit -file server.conf -op replace -line 30 -end 35 -textfile patched-block.txt -v
```

---

### replaceall — Global find-and-replace

```bash
# Change all occurrences of old domain to new
fedit -file nginx.conf -op replaceall -match "old.example.com" -text "new.example.com" -v

# Update a version string everywhere
fedit -file Makefile -op replaceall -match "1.1.0" -text "1.2.0" -v
```

---

### delete — Remove lines

```bash
# Delete a single line
fedit -file hosts -op delete -line 12 -end 12 -v

# Delete a block (lines 40-55)
fedit -file config.yaml -op delete -line 40 -end 55 -v
```

---

### write — Create or overwrite a file

```bash
# Create a new file
fedit -file /tmp/note.txt -op write -text "Deployment started" -v

# Write multi-line content from another file
fedit -file /etc/motd -op write -textfile new-motd.txt -v
```

---

### map — Structural overview of a file

```bash
# Map a Go file — see all functions, types, imports
fedit -file main.go -op map -lang go

# Map a Dockerfile — see stages and instructions
fedit -file Dockerfile -op map -lang dockerfile

# Map a Makefile — see variables, targets, duplicates
fedit -file Makefile -op map -lang makefile
```

---

## Supported Map Languages (17)

| Language | Key Structures Detected |
|------------|------------------------|
| go | package, imports, types, interfaces, functions |
| python | imports, classes, functions, decorators |
| javascript | imports, exports, classes, functions, arrows |
| typescript | (same as javascript) |
| css | imports, custom properties, selectors, @media, @keyframes |
| rust | use, mod, struct, enum, trait, impl, fn, macro_rules! |
| java | package, imports, classes, interfaces, enums, methods |
| csharp | using, namespace, classes, interfaces, records, properties |
| html | doctype, head/body, headings, scripts, links, forms, ids |
| sql | CREATE, ALTER, DROP, INSERT, SELECT, indexes, triggers |
| yaml | document separators, top-level keys |
| toml | tables, array tables, top-level keys |
| markdown | headings, code blocks, links |
| ruby | requires, modules, classes, methods, attributes |
| php | namespace, use, classes, interfaces, traits, functions |
| dockerfile | FROM stages, all instructions |
| makefile | includes, variables, .PHONY, targets |

All mappers detect **duplicates** and flag them with warnings.

---

## Flags Reference

| Flag | Description |
|------------|------------------------------------------|
| -file PATH | Target file (required) |
| -op OP | Operation to perform (required) |
| -line N | Starting line number (1-based) |
| -end N | Ending line number (for ranges) |
| -text "s" | Inline text content |
| -textfile F | Read content from a file |
| -match "s" | Substring to search for |
| -nth N | Which occurrence (default 1, -1 = last) |
| -lang LANG | Language for map operation |
| -v | Verify: show affected lines after edit |

---

## Real-World Examples

### Patch an Nginx config in a deploy script

```bash
#!/bin/bash
# Update upstream server and reload

fedit -file /etc/nginx/conf.d/app.conf \
  -op replaceall \
  -match "server 10.0.1.50:8080" \
  -text "server 10.0.1.51:8080" -v

nginx -t && systemctl reload nginx
```

### Add a cron job to crontab

```bash
fedit -file /etc/crontab \
  -op insertafter \
  -match "# Custom jobs" \
  -text "0 2 * * * root /opt/backup.sh" -v
```

### Bulk update version in multiple files

```bash
for f in Makefile config.yaml package.json; do
  fedit -file "$f" -op replaceall -match "1.1.0" -text "1.2.0" -v
done
```

### Inspect a Dockerfile before editing

```bash
fedit -file Dockerfile -op map -lang dockerfile
# See the structure, then surgically edit:
fedit -file Dockerfile -op insertafter -match "FROM alpine" -text "RUN apk add --no-cache curl" -v
```

### Remove a block from a config

```bash
# First, find the lines
fedit -file app.conf -op find -match "deprecated-feature"
# Output says lines 44-49, so:
fedit -file app.conf -op delete -line 44 -end 49 -v
```

### PowerShell workflow (Windows sysadmins)

```powershell
# Set an alias
$f = "C:\tools\fedit.exe"

# Find all TODO comments in Go code
& $f -file main.go -op find -match "TODO"

# Replace a config value
& $f -file config.toml -op replaceall -match 'debug = true' -text 'debug = false' -v

# Map a file to understand its structure
& $f -file main.go -op map -lang go
```

---

## Tips

- **Always use -v** on mutations — it costs nothing and saves you from blind edits
- **Use find before replace** — get the exact line numbers first
- **Prefer insertafter/insertbefore over insert** — matching is more resilient than hardcoded line numbers
- **Use -nth -1** to target the last occurrence of a match
- **Use -textfile** for multi-line inserts — avoids shell quoting headaches
- **map before editing unfamiliar files** — see the structure first

---


## LLM Benchmark

How well do LLMs use fedit vs. rewriting whole files?

We tested **Claude (Sonnet 4.6)**, **ChatGPT (GPT-4o)**, and **Gemini (2.5 Pro)** on
7 realistic editing tasks across files of 500-1200 lines. Each model was tested
twice per task: once asked to **output the whole file** ("raw"), once asked to
**output fedit commands only** ("fedit").

### Results

Legend: PASS = correct output | PARTIAL = correct intent, fragile execution | FAIL = wrong output

| Test | File | Task | Claude raw | Claude fedit | GPT raw | GPT fedit | Gemini raw | Gemini fedit |
|------|------|------|:---:|:---:|:---:|:---:|:---:|:---:|
| T1 | processor_600.go (575 L) | Insert method after specific struct method | PASS | PARTIAL | FAIL | PARTIAL | PASS | FAIL |
| T2 | config_800.yaml (1059 L) | Replace 24-line deployment block | PASS | PASS | FAIL | FAIL | PASS | FAIL |
| T3 | styles_500.css (565 L)   | Find & delete CSS rule block | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ |
| T4 | system_1000.go (980 L)   | Global rename (36 occurrences) | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ |
| T5 | analytics_700.py (679 L) | 3-step chain (insert + delete + replace) | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ |
| T6 | dashboard_900.html (891 L) | Insert before 3rd matching button | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ |
| T7 | engine_1200.go (1206 L)  | Map + targeted insert | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ | _pending_ |

### Key findings

1. **ChatGPT cannot output large files reliably.** On both T1 and T2, GPT-4o
   truncated mid-file and silently rewrote function signatures it had already
   emitted. Output was unusable as a drop-in replacement.

2. **LLMs hallucinate line numbers.** In T2, both ChatGPT and Gemini emitted
   fedit `replace` commands with confidently-stated line ranges that were off
   by 36-56 lines, producing completely wrong edits. Only Claude got the line
   range exactly right.

3. **`insertafter` on a function declaration matches the OPENING line.** Gemini
   repeatedly tried `insertafter -match 'func FetchUser()'` to insert content
   AFTER the function ended — but fedit (correctly) inserts after the matched
   line, placing the new code inside the function body. The fix is to use
   `insertbefore` on the NEXT declaration.

### Recommendation for LLM-driven workflows

Based on these results, prefer **content-matching operations over line-number
operations** when generating fedit commands from an LLM:

- Use `insertbefore -match 'next anchor'` instead of `insert -line N`
- Use `replaceall -match 'old' -text 'new'` instead of `replace -line N -end M`
- Reserve line-number ops for cases where you've just run `find` or `show`
  and have verified line numbers in context

For best results, give the LLM an MCP connection to fedit (see [MCP Server Mode](#mcp-server-mode))
so it can run `fedit_find` and `fedit_show` to verify line numbers before
mutating — this eliminates the hallucination class of errors entirely.

### Methodology

- Each test run in a fresh chat with the original file uploaded
- Prompts identical across models; only "raw" vs "fedit" framing differed
- Output saved verbatim, then diffed against ground truth
- Ground truth verified by running fedit commands locally and re-reading output

Test files and prompts: see `bench/` directory.

## MCP Server Mode

fedit includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server, so AI coding assistants can use fedit as a tool for precise file edits.

```bash
fedit mcp
```

This starts a JSON-RPC 2.0 server on stdin/stdout. The server exposes all 10 editing operations as MCP tools:

| Tool | Description |
|------|-------------|
| `fedit_show` | Display file contents (full or line range) |
| `fedit_insert` | Insert content after a line number |
| `fedit_delete` | Delete one or more lines |
| `fedit_replace` | Replace a line range with new content |
| `fedit_replaceall` | Global find-and-replace across a file |
| `fedit_write` | Create or overwrite a file |
| `fedit_map` | Structural overview (17 languages) |
| `fedit_find` | Find lines matching a substring |
| `fedit_insertafter` | Insert after a matching line |
| `fedit_insertbefore` | Insert before a matching line |

### Configuring with Claude Desktop

Add to your `claude_desktop_config.json`:

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

### Configuring with Cursor

Add to `.cursor/mcp.json` in your project root:

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

### Why MCP?

Without fedit, LLMs rewrite entire files — burning tokens and introducing drift. With fedit as an MCP tool, the model calls `fedit_map` to see structure, `fedit_find` to locate targets, and `fedit_replace` or `fedit_insertafter` to make surgical edits. Every mutation returns stats (line delta, elapsed time) so the model can verify its work.

---

## License

MIT — see [LICENSE](LICENSE)

---

Built by [Amalex](https://amalexhandler.com) — makers of Amalex Handler, the universal file transporter.
