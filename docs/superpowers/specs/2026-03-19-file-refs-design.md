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

During prompt token iteration, any token starting with `@` is treated as a file reference and extracted into a `fileRef` struct:

```go
type fileRef struct {
    path  string
    start int // 1-based, 0 = no range (read whole file)
    end   int // 1-based inclusive
}
```

Parsing rules:
- `@src/foo.go` → `{path: "src/foo.go", start: 0, end: 0}`
- `@src/foo.go:10-20` → `{path: "src/foo.go", start: 10, end: 20}`
- Invalid range syntax (e.g. `@foo.go:abc`) → hard error, abort with message

`goCommandOptions` gains a new field:
```go
fileRefs []fileRef
```

Non-`@` tokens continue to be collected into `prompt` as before.

### 2. File Reading & Evidence Assembly (`resolveSurfaceInput`)

`resolveSurfaceInput` receives the additional `fileRefs []fileRef` parameter. When file refs are present:

1. Read each file; apply line range if specified (1-based, inclusive)
2. If line range exceeds file length, clamp silently to end of file
3. If file does not exist, print warning to stderr and skip:
   ```
   warning: @src/foo.go: file not found, skipping
   ```
4. Join file contents with `\n---\n` separator
5. Append stdin evidence after file content if both are present

Final prompt structure:

```
<prompt>

Evidence:
<file1 content>
---
<file2 content>
---
<stdin content if any>
```

No per-file path headers are added to the evidence body.

### 3. Error Handling

| Situation | Behaviour |
|-----------|-----------|
| File not found | `stderr` warning, skip, continue |
| Line range exceeds file length | Clamp to end of file, no warning |
| Invalid range syntax (`@foo:abc`) | Hard error, abort |
| `@` token with no path (`@`) | Hard error, abort |

### 4. Unchanged Scope

- `runAgain`, `runSwap`, `runTake`, and all other commands are not modified
- The `--no-context` flag continues to suppress repo context and termbook only; file refs are always included (they are explicit user input, not automatic context)
- Stdin piping continues to work exactly as before

---

## Testing

Follow existing patterns in `app_test.go`:

- Unit tests for `@` token parsing: single file, line range, multiple files, invalid syntax
- Unit tests for `resolveSurfaceInput`: file-only, file+stdin, file+prompt, missing file warning, line range clamp

---

## Example Usage

```bash
# Single file
prtr go "what does this do" @internal/app/app.go

# Line range
prtr go "explain this function" @internal/app/app.go:905-943

# Multiple files
prtr go review @cmd/prtr/main.go @internal/app/app.go

# File + piped stdin
npm test 2>&1 | prtr go fix "why broken" @internal/app/app.go:100-150

# File + no repo context
prtr go "quick question" @src/foo.go --no-context
```
