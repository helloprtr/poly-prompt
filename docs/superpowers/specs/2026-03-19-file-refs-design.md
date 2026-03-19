# Design Spec: `@file` Reference Syntax for `prtr go`

**Date:** 2026-03-19
**Status:** Approved
**Branch:** fix/code-review-v070

---

## Problem

`prtr go` currently requires piping file content via stdin to include it as evidence:

```bash
cat src/foo.go | prtr go "explain this"
cat src/foo.go src/bar.go | prtr go "explain this"
sed -n '10,20p' src/foo.go | prtr go "explain this"
```

Pain points:
1. Typing `cat file | prtr go` is verbose
2. Including multiple files is awkward
3. Selecting specific line ranges is cumbersome

---

## Solution

Support `@path` and `@path:start-end` tokens inline in the prompt argument.

```bash
prtr go "explain this" @src/foo.go
prtr go "explain this" @src/foo.go:10-20 @src/bar.go
prtr go fix "why broken" @src/foo.go:1-50 @src/bar.go:20-40
```

---

## Design

### 1. Parsing (`parseGoCommand`)

During prompt token iteration, any **standalone shell argument** starting with `@` is treated as a file reference and extracted into a `fileRef` struct. Tokens inside a quoted string (e.g. `"email @user"`) arrive as a single argument and are never split, so the `@` inside them is never detected as a file ref — this is intentional and requires no escape mechanism.

Tokens after `--` are passed through verbatim into `command.prompt` without `@` scanning. This is intentional: `prtr go -- @src/foo.go` sends the literal string `@src/foo.go` to the AI. `@` tokens that appear *before* `--` are still scanned and collected into `fileRefs` normally.

```go
type fileRef struct {
    path  string
    start int // 1-based; 0 = no range (read whole file)
    end   int // 1-based, inclusive
}
```

Parsing rules:
- `@src/foo.go` → `{path: "src/foo.go", start: 0, end: 0}`
- `@src/foo.go:10-20` → `{path: "src/foo.go", start: 10, end: 20}`
- Invalid range syntax (e.g. `@foo.go:abc`) → hard error, abort
- `@` with no path → hard error, abort
- `start > end` (e.g. `@foo.go:20-10`) → hard error, abort
- `start == 0` with a range present (e.g. `@foo.go:0-10`) → treated as `start = 1`
- Single-line range (e.g. `@foo.go:5-5`) → valid, reads one line

Paths with spaces must be shell-quoted as a single argument:
```bash
prtr go "question" '@path/with spaces/file.go'
```
The shell passes the whole quoted string (including `@`) as one token; the parser strips the leading `@` and uses the rest as the path.

`goCommandOptions` gains a new field:
```go
fileRefs []fileRef
```

Non-`@` tokens continue to be collected into `prompt` as before.

### 2. File Reading & Evidence Assembly (`resolveSurfaceInput`)

`resolveSurfaceInput` receives an additional `fileRefs []fileRef` parameter. The existing callers (`runAgain`, `runSwap`) pass `nil` — no behaviour change for those paths.

**`ErrNoInput` logic update:** An error is returned only when prompt text is empty AND stdin is absent AND fileRefs is empty. If file refs are provided with no prompt text, the call is valid (the file content becomes the full input).

When file refs are present:

1. Read each file; apply line range if specified (1-based, inclusive)
2. If line range exceeds file length, clamp silently to end of file
3. If file does not exist, print warning to stderr and skip:
   ```
   warning: @src/foo.go: file not found, skipping
   ```
4. Join non-empty file contents with `\n---\n` separator
5. Append stdin evidence after file content if both are present

No per-file path headers are added to the evidence body. This is intentional for simplicity; per-file attribution is a possible future enhancement.

Final prompt structure:

```
<prompt text if any>

Evidence:
<file1 content>
---
<file2 content>
---
<stdin content if any>
```

If all file refs are skipped (all files missing) and no stdin, and prompt text is empty → `ErrNoInput`.

**`inputSource` return values** for the new combinations:

| Inputs present | `inputSource` |
|----------------|---------------|
| file refs only (≥1 resolved) | `"file"` |
| prompt + file refs (≥1 resolved) | `"prompt+file"` |
| file refs (≥1 resolved) + stdin | `"file+stdin"` |
| prompt + file refs (≥1 resolved) + stdin | `"prompt+file+stdin"` |
| all file refs skipped, prompt present | `"prompt"` (falls back to existing label) |
| all file refs skipped, stdin present | `"stdin"` (falls back to existing label) |
| all file refs skipped, prompt + stdin | `"prompt+stdin"` (falls back to existing label) |

Existing values (`"prompt"`, `"stdin"`, `"prompt+stdin"`) are unchanged.

### 3. Error Handling

| Situation | Behaviour |
|-----------|-----------|
| File not found | `stderr` warning, skip, continue |
| Line range exceeds file length | Clamp to end of file, no warning |
| Invalid range syntax (`@foo:abc`) | Hard error, abort |
| `@` token with no path | Hard error, abort |
| `start > end` | Hard error, abort |
| `start == 0` in range | Silently treated as `start = 1`, no warning (consistent with line-range clamping) |
| All file refs skipped + no other input | `ErrNoInput` |

### 4. Unchanged Scope

- `runAgain` and `runSwap` call `resolveSurfaceInput` with `nil` fileRefs — no behaviour change. Their existing `if len(command.prompt) > 0 || stdinPiped` guards remain correct and unchanged.
- The `runGo` call site (line 911) does **not** need a guard condition; it always calls `resolveSurfaceInput` and relies on the function itself to return `ErrNoInput` when all inputs are absent. No guard should be added at line 911.
- `runTake` is unaffected
- The `--no-context` flag continues to suppress repo context and termbook only; file refs are always included (they are explicit user input, not automatic context)
- Stdin piping continues to work exactly as before
- `@` tokens after `--` are not expanded (passed through as literal prompt text)

---

## Testing

Follow existing patterns in `app_test.go`:

- Unit tests for `@` token parsing:
  - Single file, no range
  - Single file with line range
  - Multiple files
  - Invalid range syntax → error
  - `start > end` → error
  - `start == 0` → clamped to 1
  - Tokens after `--` not parsed as file refs
- Unit tests for `resolveSurfaceInput` with fileRefs:
  - File only (no prompt, no stdin) → valid
  - Prompt + file
  - File + stdin
  - Prompt + file + stdin
  - Missing file → warning, skip
  - All files missing + no other input → `ErrNoInput`
  - Line range clamp
  - `inputSource` values for each combination

---

## Example Usage

```bash
# Single file
prtr go "what does this do" @internal/app/app.go

# Line range
prtr go "explain this function" @internal/app/app.go:905-943

# Multiple files
prtr go review @cmd/prtr/main.go @internal/app/app.go

# File only, no prompt text
prtr go @internal/app/app.go

# File + piped stdin
npm test 2>&1 | prtr go fix "why broken" @internal/app/app.go:100-150

# File + no repo context
prtr go "quick question" @src/foo.go --no-context

# Path with spaces (shell-quoted)
prtr go "explain" '@src/my module/foo.go'

# Literal @-token after -- (not expanded)
prtr go -- "@Override annotation example"
```
