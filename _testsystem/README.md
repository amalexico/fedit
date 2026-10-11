# fedit Test System -- Phase 1: `-lang` / `map`

Everything here was built and verified against a real `fedit` binary compiled
from `github.com/amalexico/fedit` @ main. Results below are measured, not assumed.

---

## Contents

```
Test-FeditLang.ps1          PowerShell harness (100% ASCII, PS7)
fixtures/<lang>/            clean_a.* and dup_b.* per language (19 langs, 38 files)
fedit_patch/
  json_csv_mappers.patch    adds doMapJSON + doMapCSV to main.go
  map_json_csv_test.go      11 Go regression tests for the new mappers
```

## Running it

```powershell
cd C:\Users\kehsi\Desktop\amalex-brand\fedit\testsystem
.\Test-FeditLang.ps1                    # all languages
.\Test-FeditLang.ps1 -Lang go,json,csv  # subset
.\Test-FeditLang.ps1 -ShowOutput        # include full map output
```

Exit code is 0 on all-pass, 1 on any failure. Current state: **34 passed, 0 failed,
4 skipped** (skips are markdown + dockerfile, explained below).

---

## Finding 1: `map` has no multi-file mode

`-file` is a `flag.String` -- one path, no globbing. Only `replaceall` has `-files`.
Confirmed by running it:

```
./fedit -file *.go -op map -lang go
# bash expands the glob -> fedit sees extra positional args -> usage error, exit 1
```

In PowerShell it is worse: PS does not expand `*.go` for a native exe, so fedit
receives the literal string `*.go` and fails to open it.

**This is not a blocker.** Each invocation already prints its own titled header:

```
=== MAP: clean_a.go (go, 13 lines) ===

-- Functions (3) --
   3 | func Add
   7 | func Sub
  11 | func Mul

OK No duplicate functions
```

So the harness loops `Get-ChildItem` and calls fedit once per file, which produces
exactly the per-file-titled output you described. If you later want native
`-file a.go,b.go,c.go`, that is a fedit source change -- say the word and it is a
small one (split on comma, loop `doMap`, keep the existing header per file).

## Finding 2: JSON and CSV did not exist -- now added

`-lang json` and `-lang csv` previously printed `Unknown language` and exited 1.
The patch adds two real mappers. Note CSV column extraction already existed via the
unrelated `fields` op (`-col N -delim ","`); that is extraction, not integrity checking.

**`map -lang json` reports:**
- Integrity: valid/malformed, with the parse error and its cause
- Structure: root type, top-level keys, max nesting depth, counts of keys/objects/arrays/scalars
- Duplicate keys **within the same object** -- caught by walking the token stream, because
  `json.Unmarshal` silently collapses duplicates and would hide them. Same key in two
  *different* objects is correctly not flagged.
- Trailing garbage after the root value (two root objects) is treated as malformed.

**`map -lang csv` reports:**
- Integrity: valid/malformed, with record number and the quoting error
- Structure: row count, data row count, column count, header list
- Ragged rows: every row whose field count differs from the header, with expected-vs-got
- Duplicate header columns, with the column positions
- Empty field count across data rows

Malformed CSV still reports the header and any rows parsed before the failure point
(a two-pass strict/lenient read), so a quoting error at line 900 does not blind you to
the first 899 rows.

## Finding 3: two pre-existing mapper gaps

Not caused by the patch -- found while building fixtures:

- **`markdown` and `dockerfile` have no duplicate-detection section at all.** Every other
  mapper ends with a dup check; these two just stop after listing. The harness marks
  these `SKIP` rather than pretending to test them. Worth adding (duplicate headings,
  duplicate `FROM ... AS <name>` stage names) -- both are real error classes.
- **CSS duplicate detection is scope-blind.** A selector inside `@media` counts as a
  duplicate of the same selector at top level, so `.header` outside and `.header` inside
  a media query is reported as a duplicate when it is legitimate CSS. The fixture avoids
  this so the suite stays green, but it is a genuine false positive.

## Finding 4: one bug found and fixed during development

First implementation reported duplicate-key line numbers one line early (`InputOffset()`
returns the offset *before* the token is consumed). Caught by testing against a fixture
with a known answer, fixed, verified: `"name"` now correctly reports line 2, `"org"` line 5.

---

## Verification performed

Clean-room: fresh `git clone`, `git apply` the patch, build, vet, test.

```
PATCH APPLIES CLEANLY
BUILD OK
VET OK
ok  github.com/amalexico/fedit  0.132s     <- full existing suite, no regressions
34 passed, 0 failed                        <- fixture validation, all 19 languages
```

The 11 new Go tests cover: clean JSON, same-object duplicates, cross-object non-duplicates,
malformed JSON, trailing garbage, clean CSV, ragged rows, duplicate headers, empty fields,
malformed quoting with header recovery, and empty file.

Note: `go.mod` requires go 1.24.3 and the sandbox had 1.22, so the build used a temporarily
lowered `go.mod`. The patch does **not** touch `go.mod` -- nothing in the new code uses
anything newer than 1.21. Build on your box at the real version to confirm.

---

## Phase 2 (not started)

Per your plan, deep testing of the remaining operations. The natural order:

1. **Read-only ops first** -- `show`, `find`, `fields`, `extract`. Safe, no mutation,
   establishes the before/after capture pattern.
2. **Mutation ops with real before/after diffing** -- `insert`, `insertafter`,
   `insertbefore`, `replace`, `delete`, `move`, `copy`. Each test: snapshot file,
   run op, snapshot again, assert the diff is exactly what was intended. This is where
   the harness earns its keep, and where your `-v` verify output becomes assertable.
3. **Multi-file ops** -- `replaceall -files GLOB` is the only true multi-file path today,
   so it gets its own matrix.
4. **Extreme cases** -- the multiple-occurrence reliability testing you asked about:
   `-nth` selection at boundaries (1, -1, out-of-range), overlapping matches, matches
   inside strings/comments, CRLF vs LF, files with no trailing newline, BOM, and the
   binary-byte case already in your critical rules.

One thing worth deciding before phase 2: whether the harness asserts against fedit's
`-v` verify block (human-readable, may change format) or against the file content
directly (stable, but you write more comparison logic). I would go with file content
for the assertions and capture `-v` purely for the human-readable log.
