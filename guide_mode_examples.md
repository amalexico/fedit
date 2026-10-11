# fedit guide mode -- example corpus (design review)

Purpose: every shape of command `guide`'s fast-syntax needs to handle, laid out
for review before the conversation-tree/parser gets built. Each example shows
the guide-syntax input, the real explicit command it translates to, and notes
on anything ambiguous or worth deciding.

Convention used below: `guide SOURCE_FILE SOURCE_BLOCK VERB DEST_FILE [position] [TARGET_BLOCK]`
-- exact word order is what we're reviewing, not fixed yet.

---

## COPY (positioning verb -- position is required, not optional)

### 1a. Copy, positioned before an existing block, single destination
```
fedit guide a.go functionOne copy b.go before functionTwo
-> fedit -op copy -file a.go -lang go -sourceblock functionOne -dest b.go -beforeblock functionTwo -v
```

### 1b. Copy, positioned after
```
fedit guide a.go functionOne copy b.go after functionTwo
-> fedit -op copy -file a.go -lang go -sourceblock functionOne -dest b.go -afterblock functionTwo -v
```

### 1c. Copy to end of destination file (no existing block to anchor to)
```
fedit guide a.go functionOne copy b.go at-end
-> fedit -op copy -file a.go -lang go -sourceblock functionOne -dest b.go -after -1 -v
```
Note: "-1" here would need to mean "last line of dest" -- need to confirm this
resolves the same way -line -N: end-of-file syntax already does elsewhere, or
if `-dest` mode needs its own explicit "at end" flag instead of overloading -1.

### 1d. Copy, NO position given -- the ambiguous case from last round
```
fedit guide a.go functionOne copy b.go
```
No one-line translation exists. Drops into a follow-up question:
```
Where in b.go should functionOne go?
  1) Before an existing function   2) After an existing function   3) At the end of the file
```
Never silently defaults. This is the pattern for every verb below marked
"position required."

### 1e. Copy, fan-out to multiple destinations, same position rule applied to each
```
fedit guide a.go functionOne copy (b.go,c.go,d.go) before functionTwo
-> fedit -op copy -file a.go -lang go -sourceblock functionOne -dest "b.go,c.go,d.go" -beforeblock functionTwo -v
```
Open question: what happens when `functionTwo` doesn't exist in one of the three
files? Proposed: per-file report, same pattern as `doReplaceAllGlob` today --
```
  b.go   OK     inserted before functionTwo (line 42)
  c.go   SKIP   functionTwo not found
  d.go   OK     inserted before functionTwo (line 18)
Fan-out: 2/3 files updated, 1 skipped
```
Never a hard-fail-everything-because-one-failed. Confirm this is the behavior
you want -- the alternative (all-or-nothing) is also defensible and safer in a
different way (no partial-fan-out state to reason about).

### 1f. Copy with an explicit line/match anchor instead of a named block (source has no clean block boundary)
```
fedit guide a.go "line 12-18" copy b.go before functionTwo
-> fedit -op copy -file a.go -line 12 -end 18 -dest b.go -beforeblock functionTwo -v
```
Source doesn't have to be a named block -- raw line ranges should still work
through guide, just described differently ("line 12-18" instead of a name).

---

## MOVE (same shape as copy, single destination only)

### 2a. Move, positioned
```
fedit guide a.go functionOne move b.go before functionTwo
-> fedit -op move -file a.go -lang go -sourceblock functionOne -dest b.go -beforeblock functionTwo -v
```
Behavior: content appears in b.go, AND is deleted from a.go. Pre-flight check
(per the design doc) should confirm both files are writable before touching
either one, so a read-only a.go or b.go fails cleanly with nothing half-done.

### 2b. Move, no position -- same follow-up-question fallback as copy
```
fedit guide a.go functionOne move b.go
-> [asks: before/after/end, same as 1d]
```

### 2c. Move, multiple destinations -- REJECTED, not translated
```
fedit guide a.go functionOne move (b.go,c.go) before functionTwo
-> Error: "move" needs exactly one destination (a moved block can only exist
   in one place). Did you mean "copy"?
```
This is the semantic conflict from the top of this doc -- guide should catch
this at input time with a helpful redirect, not pass "move" with multiple
destinations down to the engine and let it fail confusingly (or worse, silently
do something the engine's own -dest parsing wasn't designed to handle for move).

---

## REPLACE (targeting verb -- same-name target is a safe default, not a guess)

### 3a. Replace, default target (same name as source) -- no follow-up needed
```
fedit guide a.go functionOne replace b.go
-> fedit -op replace -file a.go -lang go -sourceblock functionOne -dest b.go -destblock functionOne -v
```

### 3b. Replace, explicit different target name
```
fedit guide a.go functionOne replace b.go functionThree
-> fedit -op replace -file a.go -lang go -sourceblock functionOne -dest b.go -destblock functionThree -v
```

### 3c. Replace, fan-out (propagate a fixed function to several files at once)
```
fedit guide a.go functionOne replace (b.go,c.go,d.go)
-> fedit -op replace -file a.go -lang go -sourceblock functionOne -dest "b.go,c.go,d.go" -destblock functionOne -v
```
Same per-file OK/SKIP reporting question as 1e applies here too -- what if
`functionOne` doesn't exist yet in one of the destinations to be replaced?
That's arguably a harder skip-vs-error case than the copy one, since "replace
something that doesn't exist" is a more obviously wrong operation than "insert
before something that doesn't exist" -- leaning toward SKIP-with-loud-warning
here too for consistency, but flagging it as the shakier of the two calls.

### 3d. Replace target block doesn't exist anywhere -- real error, not a skip
```
fedit guide a.go functionOne replace b.go functionNineHundred
-> Error: functionNineHundred not found in b.go. Did you mean to `copy` and
   create it instead of `replace`?
```
This is the kind of guided error message that's the actual point of this whole
mode -- not just translating syntax, but catching the "wrong verb for what you
meant" mistake before running anything.

---

## INSERTBEFORE / INSERTAFTER (position is the whole point of these verbs -- always required by definition, but the source content doesn't have to come from another file)

### 4a. Insert an existing block from another file (the "cross-file" case this whole design started from)
```
fedit guide a.go functionOne insertbefore b.go functionTwo
-> fedit -op insertbefore -file b.go -lang go -destblock functionTwo -sourcefile a.go -sourceblock functionOne -v
```
Note the flag reversal from copy/move: here `-file` is `b.go` (the file being
edited) and `a.go` becomes a new `-sourcefile` flag, because insertbefore/after
were never "two files" operations before -- `-file` has ALWAYS meant "the file
being modified" for these two verbs specifically (unlike copy/move, which don't
modify -file at all when -dest is used). Keeping that meaning stable matters
more here than reusing -dest/-sourceblock's exact shape from copy/move, since
insertbefore/insertafter callers (including MCP tools) already depend on -file
always being the edited file. Worth double-checking this doesn't read as
inconsistent given -dest's naming everywhere else -- open question below.

### 4b. Insert your own new text, not sourced from another file at all
```
fedit guide "some new code here" insertbefore b.go functionTwo
-> fedit -op insertbefore -file b.go -text "some new code here" -block functionTwo -v
```
This is the existing, already-shipped capability (-text/-textfile, no second
file involved) -- guide needs to expose this path too, not just the cross-file
one, or it'd be a regression for the single-file case that already works today.

### 4c. Insert, ambiguous target block (mirrors the real Rust "Point" ambiguity we found)
```
fedit guide a.go functionOne insertbefore b.go Calculator
-> [b.go has both "class Calculator" and "def Calculator.reset"-style nested match]
   Which "Calculator" did you mean?
     1) class Calculator (line 4)
     2) def reset inside Calculator (line 19)
```
guide should catch resolveBlock's real "-block matched N blocks" error and turn
it into a numbered choice instead of the raw engine error text.

---

## Open questions -- RESOLVED (see design_cross_file_ops.md "RESOLVED" section for detail)

1. **1e/3c fan-out failure reporting**: RESOLVED -- always per-destination,
   never all-or-nothing, for both copy and replace. `copy`/`move`/`insert*`
   fall back to append-at-end when an anchor is missing; `replace` skips (no
   sensible append-instead exists for "overwrite this thing that isn't there").
2. **1c "copy to end of file"**: RESOLVED, no engine change needed -- confirmed
   by testing that `-after N` already accepts the file's true last line number.
   `guided` computes that number itself; no new "-1 means EOF" flag value.
3. **4a's flag naming**: RESOLVED, and a real collision was caught and fixed
   in the process -- an earlier example reused `-destblock` for two unrelated
   things (target-to-overwrite for `replace`, vs. in-file anchor for
   `insertbefore`). Fix: `insertbefore`/`insertafter` keep plain `-block` for
   the in-file anchor (unchanged from today) and gain `-sourcefile`+
   `-sourceblock` only for pulling content from elsewhere. `-destblock` is
   reserved exclusively for `replace -dest`'s "block to overwrite" meaning.
   Confirmed via testing that the corrected two-flag-family split actually
   works end-to-end (see the "4a mechanic" test in design_cross_file_ops.md).

## New from this round

- `move -dest` can fan out to multiple destinations after all (reverses the
  earlier single-destination-only stance) -- redefined as "erase source once,
  land copies in as many destinations as succeed," failing only if ALL
  destinations fail. Source erase happens only after at least one destination
  write succeeds.
- `-lang` is required for ANY block resolution on the destination side
  (`-beforeblock`/`-afterblock`/`-destblock`), confirmed by testing, even when
  the source side of the same command never touches a block at all. Auto-
  detecting `-lang` from the relevant file's extension is now a real need for
  `guided`, not a nicety -- tracked as an open item in the main design doc.
- Cross-file transfer moves text, not dependencies -- tested directly by
  transferring a function that called `fmt.Println` into a file with no `fmt`
  import. Structurally perfect, does not compile. Every cross-file report
  should probably say so; exact wording/frequency still open.
- `guided` should accept an optional block-name qualifier (`"impl Point"` vs
  bare `"Point"`) to preempt the ambiguous-block case instead of always
  falling back to skip+report -- direction agreed, details not yet spec'd.
