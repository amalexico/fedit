# fedit: Cross-File Operations Design (pre-implementation)

Status: DESIGN ONLY -- nothing here is implemented yet. Written to be argued with
before code gets touched, per the "design all together" instruction.

This covers three asks that turned out to be entangled, not independent:
1. Native cross-file copy/move/insert/replace (no more show-raw + hex + insertafter)
2. The missing-blank-line bug found while manually doing that pipeline
3. `-file` accepting multiple files as input

---

## 1. Reframing the actual gap

Original ask was "add -dest so copy can go from main1.go to main3.go in one command."
Looking at the real switch-statement code for each op, the gap is different for
different operations, so one flag doesn't cover all of them the same way:

- **copy / move**: today, `-file` is read AND written back to -- these are strictly
  same-file relocation. There genuinely is no destination concept at all yet.
- **insertafter / insertbefore / replace**: `-file` is the file being modified, and
  the content to insert already CAN come from another file -- via `-textfile` --
  just not a *named block* of another file. Today you can say "insert the whole
  contents of patch.txt" but not "insert just the `parseEndFlag` function out of
  main1.go." That's the real gap here, and it's narrower than a generic `-dest`.

So the design below is two separate, smaller mechanisms rather than one `-dest`
flag bolted onto everything -- each reuses an existing flag family instead of
inventing new vocabulary where possible.

---

## 2. copy / move: add `-dest`

New flag: `-dest PATH` (destination file). Only meaningful for `-op copy` and
`-op move`.

**When `-dest` is absent:** exact current behavior, byte-for-byte. Zero risk to
every existing script, MCP call, and fixture we've already built.

**When `-dest` is present:**
- Source range resolution (`-line`/`-end`, `-match`/`-endmatch`, or `-block`+`-lang`)
  resolves against `-file` as it does today.
- Destination position resolution (`-after`/`-before`/`-aftermatch`/`-beforematch`/
  `-afterblock`/`-beforeblock`) resolves against **`-dest`'s content**, not `-file`'s.
  This is the part that's easy to get backwards -- `-afterblock` today means
  "a block inside `-file`" for same-file moves; with `-dest` present it has to mean
  "a block inside `-dest`" instead, since that's the only reading that makes the
  feature useful. Needs a clear doc note and a dedicated test, because it's a silent
  behavior change in what `-afterblock` searches depending on whether `-dest` is set.
- `copy -dest`: read from `-file`, write into `-dest`. `-file` is untouched.
- `move -dest`: read from `-file`, write into `-dest`, THEN delete the original range
  from `-file`. Two file writes for one command -- if the second (delete) fails,
  we've already duplicated content into `-dest`. Needs an explicit failure-mode
  decision: roll back the `-dest` write if the source delete fails, or leave the
  duplicate and report a clear error telling the user to manually clean up `-file`?
  I'd lean toward attempting the delete first when both files are writable-checked
  up front (pre-flight write-permission check on BOTH files before touching either
  one -- directly serves the "check read-only before acting" ask from a few rounds
  back), so the common failure mode (one file is read-only) is caught before any
  write happens at all, rather than after a partial write.

Example, matching the actual scenario from last round:
```
fedit -op copy -file main1.go -block "parseEndFlag" -lang go \
      -dest main3.go -afterblock "getNixBlocks" -v
```

---

## 3. insertafter / insertbefore / replace: sourcing content from a named block

Reuse `-textfile` (already means "read content from this file") and add ONE new
flag, `-sourceblock`, that narrows it from "whole file" to "just this block":

- `-textfile PATH` alone: unchanged, whole-file content (today's behavior).
- `-textfile PATH -sourceblock NAME` (with `-lang` for the block scanner): content
  becomes just that named block from PATH, resolved the same way `-block` resolves
  today for copy/move sourcing. `-lang` is already an existing flag being reused,
  not a new one.

This directly replaces the show-raw + fwencode + texthex pipeline for the common
case. `-texthex` still exists for cases where the source content isn't sitting in
a file fedit can re-read a block from (piped/generated content).

Example -- what we did by hand last round becomes:
```
fedit -op insertafter -file main3.go -block "getNixBlocks" -lang go \
      -textfile main1.go -sourceblock "parseEndFlag" -v
```
No hex round-trip, no intermediate variable, one command.

`replace` gets the same treatment for free since it already shares `resolveTextFull`.

---

## 4. The blank-line bug -- fix goes in the shared insertion path, not the new flags

Root cause: `doInsert` has no concept of "this insertion point sits between two
top-level declarations" -- it just splices lines in. This bug already exists TODAY
in plain same-file `insertafter -block`, independent of anything cross-file. The
cross-file work just gave us a reason to notice it. Fix belongs in `doInsert`
itself so it's shared by every caller (same-file and the new cross-file paths
alike), not duplicated per-operation.

Proposed rule: when the insertion point was resolved via `-block` (i.e., we know
we're landing exactly at a block boundary, not an arbitrary line the user picked),
and the language is one where top-level declarations are conventionally
blank-line-separated, ensure exactly one blank line exists between the inserted
content and its neighbor on the relevant side -- without adding a second one if a
blank line is already there.

This can't be a blanket rule across all 19 languages -- the convention differs:

| Wants auto blank line | Does NOT want one |
|---|---|
| go, java, csharp, rust, javascript, typescript, ruby, php, python, sql | yaml, toml, json, csv, makefile, dockerfile |

(markdown and terraform/hcl aren't in either column -- markdown has no block
duplicate-detection at all per the earlier finding, and terraform blocks conventionally
DO get blank lines between top-level `resource`/`variable` blocks, so terraform
should probably join the left column once it's map-capable; leaving it out for now
since it's block-scanner-only today.)

This table needs your sign-off before implementation -- it's a judgment call per
language, not something I can verify against a spec the way the JSON/CSV integrity
checks were.

---

## 5. `-file` accepting multiple files

This is a third, mostly-independent axis: breadth (many files, same operation)
rather than depth (two files, one operation). Today only `replaceall` has this,
via the separate `-files GLOB` flag and `doReplaceAllGlob`.

Proposal: extend `-file` itself to accept a comma-separated list or a glob pattern,
for operations where "do the same thing independently to each matching file" is a
coherent idea: `show`, `map`, `find` (read-only -- trivially safe, just loops and
prints a header per file, exactly like our test harnesses already do manually) and
`insert`, `delete`, `replace`, `insertafter`, `insertbefore`, `replaceall` (mutating
-- each file gets the same anchor/content applied independently, with a per-file
SKIP/OK/ERROR summary line the way `doReplaceAllGlob` already reports today).

`-files` stays as a deprecated alias for backward compatibility (existing scripts
using `-files` for replaceall keep working unchanged) but the canonical form
becomes `-file "*.go"` or `-file "a.go,b.go,c.go"` for any operation that supports it.

**Explicit non-goal for v1: multi-file `-file` combined with `-dest` or
`-sourceblock`.** Copying one block into three destination files, or pulling from
three source files into one destination, are real things someone might eventually
want (fan-out / fan-in), but they add a combinatorial pile of edge cases (what
happens when 2 of 3 sources match the block and 1 doesn't? what's the destination
insertion point if it differs per source?) on top of two features that are
already substantial on their own. I'd ship `-dest` and `-sourceblock` first,
ship multi-file `-file` for the single-file-target operations second, and treat
"both at once" as a v2 question once we've seen how the first two actually get used.

---

## RESOLVED (superseding sections 2-4 above where they conflict)

Naming settled through worked examples (see guide_mode_examples.md for the
full corpus). Final flag set for cross-file operations:

- `-dest PATH` (or comma-separated list for fan-out) -- destination file(s),
  for `copy`/`move`/`replace` only.
- `-sourceblock NAME` -- the block in `-file` being read as content. New flag,
  used only when doing cross-file work; same-file `-block` usage is completely
  unchanged.
- `-destblock NAME` -- the block in `-dest` to be OVERWRITTEN. Used only by
  `replace -dest`. Reserved for this one meaning only -- confirmed and fixed a
  real naming collision where an early example reused `-destblock` to also mean
  "anchor position inside `-file`" for `insertbefore`/`insertafter`. That role
  needs no new flag at all: `-block` already means exactly that today and
  nothing about it changes when doing cross-file work, since `-file` never
  stops being the edited file for these two verbs.
- `-beforeblock`/`-afterblock` -- unchanged flags, existing meaning ("position
  anchor in the file being written to"); when `-dest` is present they resolve
  against `-dest`'s content instead of `-file`'s, which is a continuation of
  their existing role, not a new one.
- `-sourcefile PATH` -- new flag, `insertbefore`/`insertafter` only. The other
  file to pull `-sourceblock` content from. `-file` keeps meaning "the file
  being edited" for these two verbs, exactly as it does today.

Behavior decisions:

- **Missing destination anchor (copy/move/insertbefore/insertafter):** never
  skip, never hard-fail. Fall back to appending at the end of that destination
  file and report what happened, e.g. `c.go: functionTwo not found, functionOne
  added at the end`.
- **Missing destination file entirely:** create it with just the transferred
  content, report `File d.go created with functionOne as content`.
- **Fan-out (`-dest a,b,c`):** always per-destination, never all-or-nothing.
  Every destination gets attempted and reported independently.
- **`replace -dest` missing target block:** the one case that DOES skip rather
  than falling back -- there's no sensible "insert instead" behavior when the
  thing you were told to overwrite doesn't exist. Report as skip, not error.
- **`move -dest` with multiple destinations:** allowed (reverses the original
  v1-non-goal stance above) -- redefined as "erase the source once, distribute
  copies to as many destinations as succeed." Fails only if EVERY destination
  fails. Ordering: attempt all destination writes first; erase the source only
  if at least one destination write succeeded. This avoids the one truly unsafe
  state (content deleted from source but landed nowhere).
- **`-lang` requirement confirmed empirically** (not assumed): `resolveBlock`
  requires `-lang` for ANY block resolution, including `-beforeblock`/
  `-destblock`, even when the source side of the same command uses a plain
  `-line` range and never touches `-sourceblock` at all. Tested directly:
  `-beforeblock` on a `-lang`-less command fails with `-block requires -lang`
  before even attempting a match. This makes auto-detecting `-lang` from the
  relevant file's extension (source extension for `-sourceblock`, dest
  extension for `-destblock`/`-beforeblock`/`-afterblock`) a real ergonomic
  need for `guided`, not a nicety -- otherwise `-lang` has to be threaded
  through every guided cross-file command even when nobody typed a language
  anywhere. `map` already auto-detects from extension; `resolveBlock` does not
  yet -- this would be new code, shared by `-block` too once it exists.
- **"At end of destination" needs no engine change at all.** Tested directly:
  `-after N` already accepts any N up to and including the file's real last
  line number, and that's already "append at the very end." `guided` just
  needs to read the destination's line count first and pass the real number --
  there's no `-1`-means-EOF convention to add.
- **Cross-file transfer does not carry dependencies.** Tested directly:
  moving a function that calls `fmt.Println` into a file with no `fmt` import
  produces text-perfect, structurally-correct output that will not compile.
  This is a hard boundary of text-level operation, not a bug to fix -- fedit
  has no semantic awareness of what a block depends on. Open question below:
  should every cross-file report include a standing reminder about this, or
  only surface it on request?

## Still open

1. Sign off on the blank-line-language table in section 4 -- not yet reviewed.
2. Should every cross-file transfer report include the "check dependencies/
   imports" reminder by default, or only when asked?
3. `guided`'s optional block-name qualifier (e.g. `"impl Point"` vs `"Point"`)
   for preempting ambiguous-block errors -- confirmed as the right direction,
   not yet spec'd in detail (which languages need it, exact syntax).
