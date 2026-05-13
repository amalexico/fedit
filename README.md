# fedit — Fast File Editor for the Terminal

A zero-dependency CLI tool for **surgical file edits** from the command line.
No interactive editors. No sed/awk gymnastics. Just simple, predictable operations
with built-in verification.

Built for sysadmins, DevOps engineers, and anyone who scripts config changes.

![fedit demo](demo.gif)

```
go install github.com/amalexico/fedit@latest
```

---

## Why fedit?

- **One binary, zero dependencies** — pure Go, runs everywhere
- **17 language mappers** — see the structure of any file before editing
- **-v flag** — verify every mutation before moving on
- **Line-aware** — no regex surprises, no "which match did it hit?"
- **Stream engine** — process multi-GB files line-by-line with atomic integrity
- **Field extraction** — pull CSV/TSV column N without awk
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

# Move a function block before another (content-matched, atomic)
fedit -file main.go -op move -match "func OldHelper(" -end 45 -beforematch "func NewHelper(" -v

# Copy a config block and paste it 3 times at a new location
fedit -file values.yaml -op copy -line 50 -end 65 -after 200 -times 3 -v

# Extract column 2 from a TSV file (v1.4+)
fedit -file data.tsv -op fields -col 2

# Regex replace on a multi-GB log without loading it into memory (v1.4+)
fedit -file huge.log -op replaceall -match 'ERROR' -text 'WARN' -stream
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

---


### fields -- Extract a column from delimited files (v1.4.0)

```bash
# Extract column 2 from a tab-separated file (default delimiter: tab)
fedit -file data.tsv -op fields -col 2

# Extract the third field from a CSV
fedit -file report.csv -op fields -col 3 -delim ","

# Extract usernames from /etc/passwd (colon-delimited)
fedit -file /etc/passwd -op fields -col 1 -delim ":"
```

Output goes to stdout for piping. Lines shorter than `-col` are skipped silently.
Always streaming -- no memory limit regardless of file size.

### -stream -- Large-file streaming mode (v1.4.0)

Add `-stream` to `replaceall` or `find` to process files line-by-line without
loading them into memory. 10 MB per-line buffer handles JSON blobs and minified files.
Atomic integrity: writes to a temp file then renames -- original is untouched on interruption.

```bash
# Replace a pattern in a multi-GB log file
fedit -file server.log -op replaceall -match "10.0.0.1" -text "10.0.0.2" -stream

# Regex replace in a huge file with capture groups
fedit -file big.csv -op replaceall -match-regex 'id_(\d+)' -text 'ID_$1' -stream

# Streaming find -- grep-style output to stdout
fedit -file huge.log -op find -match "FATAL" -stream
```

Supported with `-stream`: `replaceall` (literal and regex), `find`.
Not supported: `move`, `copy`, `map` (these require full file structure in memory).

#### v1.5.0: HCL/Terraform block mapper (`-lang hcl`)

Move, copy, and refactor Terraform blocks by name — no line numbers needed.
Accepts `-lang hcl`, `-lang tf`, or `-lang terraform` (all equivalent).

Supported block types: `resource`, `data`, `module`, `provider`, `variable`,
`output`, `locals`, `terraform`, `moved`, `import`, `check`.

```bash
# Move a resource block before another
fedit -file main.tf -op move -block 'resource "aws_instance" "web"' \
      -beforeblock 'resource "aws_s3_bucket" "data"' -lang hcl -v

# Copy a variable definition (scaffold new variable from existing)
fedit -file variables.tf -op copy -block 'variable "instance_type"' \
      -after 20 -lang hcl -v

# Reorder provider blocks
fedit -file providers.tf -op move -block 'provider "google"' \
      -beforeblock 'provider "aws"' -lang hcl -v
```

Nested blocks (e.g. `ingress {}` inside a `resource`) are correctly ignored —
only top-level blocks are matched.

#### v1.5.0: Nix block mapper (`-lang nix`)

Move and copy top-level attribute bindings in Nix expression files.
Handles attribute sets (`name = { }`), lists (`name = [ ]`), and
dotted attributes (`programs.git = { }`).

```bash
# Reorder home-manager program configs
fedit -file home.nix -op move -block "programs.git" \
      -beforeblock "programs.ssh" -lang nix -v

# Copy a service config as a scaffold
fedit -file configuration.nix -op copy -block "services.nginx" \
      -after 50 -lang nix -v
```


### move â€” Move a line range to a new position

```bash
# Move lines 100-120 to after line 200 (explicit range)
fedit -file server.go -op move -line 100 -end 120 -after 200 -v

# Move a function block to before another function (content-matched)
fedit -file routes.go -op move -match "func OldHelper(" -end 45 -beforematch "func NewHelper(" -v

# Swap two nginx server blocks
fedit -file nginx.conf -op move -match "server {" -endmatch "# end server 1" -aftermatch "# end server 2" -v

# Cut once, scaffold 3 copies at destination
fedit -file main.go -op move -line 5 -end 12 -after 100 -times 3 -v
```

**Rules:**
- Destination may not overlap the source range â€” fedit reports a precise error with line numbers.
- `-times N`: cut once, paste N times. Net delta = blockSize Ã— (Nâˆ’1). Default times=1 = zero delta.

---

### copy â€” Copy a line range to a new position

```bash
# Copy a config block to after a section header
fedit -file values.yaml -op copy -line 50 -end 65 -aftermatch "# staging" -v

# Duplicate a test fixture 10 times for parameterised tests
fedit -file fixtures_test.go -op copy -match "func TestCase(" -end 30 -after 200 -times 10 -v

# Reorder Python classes (copy source before target; overlap is allowed)
fedit -file processor.py -op copy -match "class ModuleProcessor_15" -endmatch "class ModuleProcessor_16" -beforematch "class ModuleProcessor_13" -v
```

**Rules:**
- Snapshot semantics: source range is read once before any writes. All N copies are identical clones of the original, even when destination overlaps source.
- Net delta = blockSize Ã— times.
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

How well do current LLMs use fedit vs. rewriting whole files?

We tested **Claude (Sonnet 4.6)**, **ChatGPT (GPT-4o)**, and **Gemini (2.5 Pro)** on
7 realistic editing tasks across files of 565-1206 lines. Each model was tested
twice per task: once asked to **output the whole file** ("raw"), once asked to
**output fedit commands only** ("fedit").

### Results

Legend: `PASS+` = optimal one-line solution | `PASS` = correct | `PASS*` = correct content, formatting artifact | `PARTIAL` = correct intent, off-by-one or fragile | `FAIL` = wrong output

| Test | File | Task | Claude raw | Claude fedit | GPT raw | GPT fedit | Gemini raw | Gemini fedit |
|------|------|------|:---:|:---:|:---:|:---:|:---:|:---:|
| T1 | processor_600.go (575 L)   | Insert method after specific struct method | PASS  | PARTIAL | FAIL  | PARTIAL | PASS  | FAIL  |
| T2 | config_800.yaml (1059 L)   | Replace 24-line deployment block           | PASS  | PASS    | FAIL  | FAIL    | PASS  | FAIL  |
| T3 | styles_500.css (565 L)     | Find & delete CSS rule block               | PASS  | PASS    | PASS* | FAIL    | PASS* | FAIL  |
| T4 | system_1000.go (980 L)     | Global rename (36 occurrences)             | PASS  | PASS+   | FAIL  | PASS+   | PASS* | PASS+ |
| T5 | analytics_700.py (682 L)   | 3-step chain (insert + delete + replace)   | PASS  | PARTIAL | FAIL  | FAIL    | PASS  | FAIL  |
| T6 | dashboard_900.html (891 L) | Insert before 3rd matching button (`-nth`) | PASS  | PARTIAL | FAIL  | PASS+   | PASS  | FAIL  |
| T7 | engine_1200.go (1196 L)    | Map + targeted insert after method         | PASS  | PASS    | FAIL  | FAIL    | PASS  | FAIL  |

### Top-line numbers

| Model       | Raw mode      | Fedit mode                       |
|-------------|---------------|----------------------------------|
| Claude      | **7/7 PASS**  | 4 PASS, 3 PARTIAL                |
| ChatGPT     | 1/7 PASS*     | 2 PASS+, 1 PARTIAL, 4 FAIL       |
| Gemini      | 7/7 PASS*     | 1 PASS+, 6 FAIL                  |

### Key findings

**1. ChatGPT cannot output large files reliably.**
6 of 7 raw tests truncated. The most striking failure was T6, where it inserted
the literal placeholder line `[... TRUNCATED FOR BREVITY ...]` into otherwise
valid HTML — a uniquely dangerous failure mode where output looks structurally
complete but contains placeholder strings. T7 truncated 1206 lines down to 166.

**2. LLMs hallucinate line numbers — and it gets worse with chain length.**
Gemini's line-number errors grew across the suite: off by 36-56 lines on T2
(single replace), 45 lines on T3, then **73 lines** on T5 (3-step chain).
ChatGPT showed similar drift on T5. Claude was the only model that produced
runnable line-numbered commands consistently.

**3. Content-matching ops are immune to the hallucination class.**
Gemini failed every fedit test that required line numbers (T1, T2, T3, T5, T6,
T7), but PASSED T4 — which used `replaceall` with a content match. Same model,
same task complexity, dramatically different reliability. The bottleneck is
counting, not understanding.

**4. `insertafter` on a function declaration matches the OPENING line.**
Models repeatedly tried `insertafter -match "func MyFunc()"` to insert content
AFTER the function ended. This is correct fedit behavior (it inserts after the
matched line) but inserts the new code INSIDE the function body. Workaround:
`insertbefore` the NEXT structural element. Confirmed in T1 and T7.

**5. Even Claude flubs line-number direction.**
T6: Claude used `insert -line 30` to add a banner before the 3rd button at line
30. But `insert -line N` adds content AFTER line N. The correct command was
`insertbefore -match "View Details" -nth 3`. Even the strongest model gets
direction wrong about 1 in 6 single-step ops when reaching for line numbers.

**6. `-match` is single-line only — and that's a feature.**
ChatGPT (T3 fedit) tried to pass a multi-line CSS block as `-match` with literal
`\n` escapes. fedit doesn't interpret escapes in `-match` and matches only
single lines. This kept the operation safe (zero matches → no edit) rather than
allowing a fragile multi-line pattern that could easily mismatch.

**7. Markdown rendering is a hidden adversary.**
ChatGPT and Gemini outputs lost `__name__` → `name`, `__init__` → `**init**`,
and stripped CSS/Python indentation when rendered in chat UIs. The underlying
files (when downloaded directly) were correct. This affects copy-paste workflows
but not API integrations. **For best results, use the model's "copy code"
button or download links — never select-and-copy from rendered output.**

**8. Three different models converged on the same one-liner for T4.**
Claude, ChatGPT, and Gemini all independently produced
`fedit -op replaceall -match "FetchUser" -text "GetAccount"`.
When the right tool is obvious, models reach for it. fedit's design surface
makes the right tool obvious for content-driven edits.

### Recommendations for LLM-driven workflows

Based on these results, prefer **content-matching operations over line-number
operations** when generating fedit commands from an LLM:

- Use `insertbefore -match "next anchor"` instead of `insert -line N`
- Use `replaceall -match "old" -text "new"` instead of `replace -line N -end M`
- Use `find -match` and `show` to confirm line numbers before any line-numbered op
- Reserve line-numbered ops for cases where an MCP-connected LLM has just run
  `fedit_find` or `fedit_show` and has verified the line in context

For best results, give the LLM an MCP connection to fedit (see
[MCP Server Mode](#mcp-server-mode)) so it can `fedit_find` and `fedit_show`
before mutating. This eliminates the line-number hallucination class entirely
and was the path Claude consistently took when it had recon available.

### Methodology

- Each test run in a fresh chat with the original file uploaded
- Prompts identical across models; only "raw" vs "fedit" framing differed
- Output saved verbatim, then diffed against ground truth via `Compare-Object`
- Ground truth verified by running fedit commands locally and re-reading output
- All 7 raw outputs that PASSED Compare-Object empty (byte-identical to ground
  truth): Claude T2, T3, T4, T5, T6, T7 + Gemini T5, T6, T7
- `PASS*` indicates byte-difference from indentation-stripping in chat UI render,
  with content structurally correct (verified by re-running fedit ops on the
  saved file)

### Test corpus

7 synthetic files (565-1206 lines) covering Go, YAML, CSS, Python, and HTML.
All test files, prompts, and ground-truth outputs are available in the `bench/`
directory of this repository.
## MCP Server Mode

fedit includes a built-in [Model Context Protocol](https://modelcontextprotocol.io/) (MCP) server, so AI coding assistants can use fedit as a tool for precise file edits.

```bash
fedit mcp
```

This starts a JSON-RPC 2.0 server on stdin/stdout. The server exposes all 12 editing operations as MCP tools:

| Tool | Description |
|------|-------------|
| `fedit_show` | Display file contents (full or line range) |
| `fedit_insert` | Insert content after a line number |
| `fedit_delete` | Delete one or more lines |
| `fedit_replace` | Replace a line range with new content |
| `fedit_replaceall` | Global find-and-replace; regex capture groups; multi-file glob; `-stream` for large files |
| `fedit_write` | Create or overwrite a file |
| `fedit_map` | Structural overview (17 languages) |
| `fedit_find` | Find lines matching a substring |
| `fedit_insertafter` | Insert after a matching line |
| `fedit_insertbefore` | Insert before a matching line |
| `fedit_move` | Move a line range to a new position; destination-overlap rejected |
| `fedit_copy` | Copy a line range; snapshot semantics, overlap allowed, -times N |

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


## FAQ

### Do I need fedit if my team has a solid PR review process?

Probably not for human-driven edits where you review every diff. fedit shines in two specific cases:

1. **Whole-file rewrites in long configs.** When an LLM regenerates a 600-line `values.yaml` to change one key, the diff is technically reviewable but practically nobody scrolls to line 412 to confirm nothing else moved. fedit makes the diff exactly N lines for an N-line change.

2. **Agent loops without a human in the middle.** If you run agents semi-autonomously (issue → branch → PR opened, you review at the end), giving the agent surgical tools (`fedit_replace`, `fedit_insertafter`) instead of "rewrite the whole file" measurably reduces review surface area.

If your workflow already catches these, the CLI is overkill. The MCP server may still be useful as an agent primitive.

### Does fedit support streaming for all operations?

Currently `replaceall` and `find` support `-stream` mode. `move` and `copy` require the full file in memory because they need to read both source and destination ranges. `map` requires structure analysis. `fields` is always streaming.

### What is the maximum file size fedit can handle?

In default (in-memory) mode: limited by available RAM, typically fine up to a few hundred MB. With `-stream`: unlimited -- fedit processes one line at a time with a 10 MB per-line buffer.

### Does fedit support Terraform / HCL?

Yes, as of v1.5.0. Use `-lang hcl` (or `-lang tf`/`terraform`) with `-block`, `-beforeblock`, `-afterblock` to move and copy Terraform blocks by name. All 11 top-level block types are supported. The `map` op does not yet show HCL structure (planned for v1.6.0).

### Does fedit preserve comments and formatting?

Yes, by design. fedit does not parse-and-reformat — it operates on raw text bytes at line granularity.

If you `replace -line 47 -end 49`, lines 1–46 and 50–EOF are untouched byte-for-byte: comments, trailing whitespace, mixed indentation, BOM markers, all preserved exactly. This is the main reason fedit is line-addressable instead of AST-based: AST round-trips lose too much (trailing commas, comment positioning, key ordering, blank-line spacing).

This matters most for nginx configs (inline comments documenting *why* a directive exists), Ansible playbooks (YAML anchors and merge keys), and any file where a human formatting choice carries semantic weight.

### Is the MCP server Claude-specific or Cursor-specific?

Neither — it targets the open MCP spec. Plain JSON-RPC 2.0 over stdin/stdout against the public Model Context Protocol schema. No vendor-specific bits.

Tested with Claude Desktop; should work with any compliant MCP client (Cursor, Continue, Cline, custom integrations). Tool definitions and JSON Schema live in `mcp.go` if you want to inspect the surface area before wiring it up.

### What if the target string appears multiple times in the file?

Use the `-nth N` flag:

- `-nth 1` (default) — first match
- `-nth 2` — second match
- `-nth -1` — last match
- Any positive integer picks that occurrence

Example targeting the third `http.Redirect` call:

```bash
fedit -file router.go -op insertbefore -match "http.Redirect" -nth 3 -textfile patch.txt -v
```

If you are not sure how many matches exist, run `find` first — it lists every match with context so you can disambiguate before mutating:

```bash
fedit -file router.go -op find -match "http.Redirect"
```

### Does chaining commands cause line-number drift?

No. Each fedit invocation reads the file fresh, so line numbers always reflect the current state on disk. You can chain operations safely with `;` (PowerShell) or `&&` (bash):

```bash
fedit -op replaceall -file f.conf -match "old.com" -text "new.com" -v ; fedit -op replace -file f.conf -line 42 -end 42 -text "fixed" -v
```

The tradeoff is one disk read per operation, which is negligible for editing workflows.

If you want drift-immunity by design, prefer content-based targeting (`insertafter` / `insertbefore` / `replaceall` with `-match`) over line numbers — those resolve the target on each invocation regardless of what previous ops did.
## License

MIT — see [LICENSE](LICENSE)

---

Built by [Amalex](https://amalexhandler.com) — makers of Amalex Handler, the universal file transporter.
