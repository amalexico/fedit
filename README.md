# fedit

A line-oriented file editor for LLM-assisted coding.

fedit lets AI assistants (and humans) make precise, scriptable edits to source files -- by line number or by content matching. No SDK, no dependencies, single binary.

## Install

```
go install github.com/amalexico/fedit@latest
```

Or download a binary from [Releases](https://github.com/amalexico/fedit/releases).

## Operations

| Operation | Description |
|-----------|-------------|
| `show` | Display lines (entire file or range) |
| `insert` | Insert content after a line |
| `delete` | Delete line(s) |
| `replace` | Replace line range with new content |
| `write` | Write/overwrite entire file |
| `map` | Structural overview (Go funcs, HTML sections) |
| `find` | Find lines matching a substring |
| `insertafter` | Insert after matching line (content-based) |
| `insertbefore` | Insert before matching line (content-based) |

## Examples

```bash
# Show file structure
fedit -file server.go -op map

# Find a function
fedit -file server.go -op find -match "handleCreate"

# Insert after a match (no line-number math)
fedit -file server.go -op insertafter -match 'log.Info()' -textfile patch.txt

# Show lines 100-150
fedit -file server.go -op show -line 100 -end 150

# Replace lines 50-52
fedit -file server.go -op replace -line 50 -end 52 -textfile patch.txt
```

## Flags

| Flag | Description |
|------|-------------|
| `-file` | Target file path |
| `-op` | Operation name |
| `-line` | Start line (1-based) |
| `-end` | End line |
| `-text` | Inline text (use \n for newlines) |
| `-textfile` | File containing content to insert/replace |
| `-match` | Substring to search for |
| `-nth` | Which occurrence (default 1, -1 for last) |

## Why?

LLMs are great at generating code but terrible at applying patches to the right location.
fedit gives them a deterministic, scriptable way to edit files --
especially `insertafter`/`insertbefore` which eliminate line-number drift entirely.

## License

MIT