# prtr v0.8 — Context-First + Beautiful CLI Design Spec

**Date:** 2026-03-17
**Status:** Approved
**Goal:** Make prtr the standard AI work layer for any terminal — easy, fast, and visually striking.

---

## Vision

prtr becomes the go-to AI command layer that works alongside any terminal. Not a terminal replacement (not Warp), but the indispensable companion that developers reach for regardless of their setup. Global audience. Translation is a secondary feature; AI workflow orchestration is the core.

The shift: from "I command, prtr executes" → "prtr knows my context, I just say go."

---

## What's New in v0.8

Four additions on top of the existing stable surface (`go`, `swap`, `take`, `again`, `learn`, `inspect`, `history`). Nothing existing is removed or changed.

---

## 1. `prtr watch` — Event-Driven Context Engine

### Behavior

A background process monitors the current shell session for developer events. When an event is detected, it collects relevant context and offers an inline suggestion before asking.

### Event Detection Targets

**Immediate suggestion (high signal):**
- Test failure (`npm test`, `go test`, `pytest`, etc.)
- Build error
- Crash / panic output
- Git merge conflict

**Optional suggestion (medium signal, config-gated):**
- Lint warnings
- Large diffs (50+ lines)
- TODO comment growth

### Suggestion Format

```
⚡ prtr: 컨텍스트 준비됨
  • 실패 로그 2줄
  • git diff: auth.js +12/-3
  • branch: fix/login-flow
  → prtr go fix [y/N]
```

Context is displayed before asking. User types `y` to send or `N` / Enter to dismiss. No clipboard, no launch — just the next prompt queued.

### Commands

```bash
prtr watch          # start background watcher
prtr watch --off    # stop watcher
prtr watch --status # show watcher state
```

### Config

```toml
# ~/.config/prtr/config.toml
[watch]
enabled = true
notify = false          # opt-in system notification (future)
medium_signals = false  # opt-in lint/large-diff suggestions
```

### Privacy

Watcher respects `.prtrignore` at repo root (gitignore syntax). Files matching ignore patterns are excluded from auto-collected context. Sensitive file patterns (`.env`, `*.key`, `*secret*`) are excluded by default.

### Implementation Notes

- Implemented as a long-running subprocess, PID stored in `~/.config/prtr/watch.pid`
- Reads shell output via a shell hook (`.zshrc` / `.bashrc` injection on `prtr watch` first run)
- Context collection reuses existing `repoctx` and `input` packages
- No new external dependencies

---

## 2. Context Engine Enhancement

### Current State

`prtr go` already attaches lightweight repo context (repo name, branch, changed files summary).

### v0.8 Enhancement

When `prtr go` or `prtr watch` fires, the context engine auto-collects:

| Source | What's collected |
|--------|-----------------|
| git diff | Staged + unstaged changes (trimmed to 200 lines) |
| Test output | Last test run result if available in `$TMPDIR/prtr-last-test` |
| Error log | Last N lines of piped stderr (already supported via pipe) |
| Branch name | Already supported |

Context collection is transparent — `prtr inspect` shows exactly what was bundled.

### `.prtrignore`

Gitignore-syntax file at repo root. Excluded from all automatic context collection.

Default exclusions (built-in, no file needed):
```
.env
.env.*
*.key
*.pem
*secret*
*password*
```

---

## 3. `--deep` Pipeline Visualization

### Current State

`--deep` runs 5 workers sequentially (planner → patcher → critic → tester → reconciler) with no visual feedback.

### v0.8 Enhancement

Replace silent execution with a bubbletea TUI showing real-time progress:

```
$ prtr take patch --deep

[████████░░░░░░░] 2/5
planner ✓  →  patcher ⠼  →  critic ○  →  tester ○  →  reconciler ○
patcher: 변경 범위 분석 중 (3s)
```

**States per stage:**
- `○` — pending (dim)
- `⠼` — running (animated spinner, yellow)
- `✓` — complete (green)
- `✕` — failed (red)

### Implementation Notes

- Uses `charmbracelet/bubbles` progress bar (already in `go.mod`)
- Uses `charmbracelet/lipgloss` for stage token styling (already in `go.mod`)
- `internal/deep` workers emit state via a `chan PipelineEvent` passed at construction
- New `internal/deep/ui.go` — bubbletea model subscribing to the channel
- Falls back to plain text output when `--no-color` or non-TTY

---

## 4. `prtr dashboard` — Minimal Home Screen

### Behavior

Running `prtr` with no arguments launches a minimal interactive TUI instead of printing help text.

```
⚡ prtr  v0.8.0

CONTEXT
branch: fix/login-flow
target: claude
watch:  active

QUICK ACTIONS
g  go      send a prompt
t  take    next action from clipboard
s  swap    change AI target
h  history recent runs
q  quit
```

Key bindings activate the corresponding command inline (e.g., `g` opens a prompt for `prtr go`).

### Implementation Notes

- bubbletea model in `cmd/prtr/dashboard.go`
- Reads current config and `watch.pid` for status
- `q` / `Ctrl-C` exits cleanly
- Existing `prtr --help` remains accessible via `prtr help`

---

## What's Unchanged

All existing commands and flags are preserved with no breaking changes:

- `prtr go`, `swap`, `take`, `again`, `learn`, `inspect`
- `prtr history`, `rerun`, `pin`, `favorite`
- `prtr setup`, `start`, `doctor`, `lang`
- `prtr templates`, `profiles`
- `--deep`, `--llm`, `--dry-run`, `--edit`, `--no-copy`, `--no-context`
- All delivery flags: `--launch`, `--paste`, `--submit`

---

## Architecture Summary

```
prtr watch (subprocess)
  └─ shell hook → event stream
       └─ context engine (repoctx + input packages)
            └─ inline suggestion → prtr go/fix/debug

prtr go / take
  └─ context engine (enhanced)
       └─ .prtrignore filter
            └─ existing translate → template → deliver pipeline

prtr take --deep
  └─ internal/deep workers
       └─ chan PipelineEvent
            └─ bubbletea UI (internal/deep/ui.go)

prtr (no args)
  └─ dashboard TUI (cmd/prtr/dashboard.go)
```

---

## Out of Scope for v0.8

- System notifications (`watch.notify`) — config key reserved, not implemented
- Medium signal events (lint, large diff) — config key reserved, off by default
- iTerm2 support
- GUI AI app automation
- Team sharing / sync
