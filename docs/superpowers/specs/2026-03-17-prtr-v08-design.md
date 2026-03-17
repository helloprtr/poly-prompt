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

Four additions on top of the existing stable surface (`go`, `swap`, `take`, `again`, `learn`, `inspect`, `history`). Nothing existing is removed or changed in behavior.

---

## 1. `prtr watch` — Event-Driven Context Engine

### Behavior

A background process monitors the current shell session for developer events. When an event is detected, it collects relevant context, displays a summary, and offers an inline `y/N` prompt.

### Shell Hook Mechanism

On first `prtr watch` run, prtr appends a hook to the user's shell config (`~/.zshrc` or `~/.bashrc`). The hook uses a **tee-based wrapper** to capture output without PTY manipulation:

**zsh (via `preexec` + `precmd`):**
```zsh
# written to ~/.zshrc by prtr watch
_prtr_preexec() { _PRTR_CMD="$1"; }
_prtr_precmd() {
  local exit=$?
  if [[ -n "$_PRTR_CMD" ]]; then
    printf '{"exit_code":%d,"cmd":"%s","output_file":"%s"}\n' \
      "$exit" "$_PRTR_CMD" "$TMPDIR/prtr-last-output" \
      | socat - UNIX-CONNECT:~/.config/prtr/watch.sock 2>/dev/null || true
    unset _PRTR_CMD
  fi
}
add-zsh-hook preexec _prtr_preexec
add-zsh-hook precmd _prtr_precmd
```

Output capture: commands are wrapped via `ZDOTDIR` trick or `alias`-based tee:
```zsh
# User runs: npm test
# Hook wraps as: npm test 2>&1 | tee $TMPDIR/prtr-last-output; exit ${PIPESTATUS[0]}
```

**bash (via `DEBUG` trap + `PROMPT_COMMAND`):** equivalent pattern using `BASH_COMMAND` and `eval` wrapping.

**Tradeoffs documented for the implementer:**
- The alias/tee approach intercepts stdout+stderr but not raw PTY writes (acceptable — test runners and build tools all write to stdout/stderr)
- `socat` is used for socket communication; if not available, falls back to writing a temp notification file (polling fallback)
- Windows: shell hook is not supported in v0.8; `prtr watch` prints an unsupported message on Windows

The watcher subprocess binds to `~/.config/prtr/watch.sock` and reads JSON events.

### IPC: Suggestion Delivery

The watcher subprocess cannot write directly to the user's terminal (the TTY is owned by the interactive shell). Instead:

- The shell's `precmd` hook also **checks a response file** at `~/.config/prtr/watch-suggest` after each command
- If the file exists, the hook reads it, prints the suggestion to the terminal (via `echo`), and prompts `[y/N]`
- The shell reads the response and, if `y`, runs `prtr go <action>` directly
- The watcher writes `watch-suggest` only when it has a high-confidence event

This keeps the suggestion in the shell's own stdout stream — no PTY injection, no `/dev/tty` tricks.

### Event Detection

The watcher reads `output_file` and matches against patterns:

| Event | Detection |
|-------|-----------|
| Test failure | Exit code ≠ 0 AND output matches `FAIL \|✕ \|failed\|Error:` |
| Build error | Exit code ≠ 0 AND output matches `error\[E\|cannot find\|undefined:` |
| Crash / panic | Output matches `panic:\|Segmentation fault\|SIGSEGV` |
| Git conflict | Output matches `CONFLICT \|Automatic merge failed` |

### Subprocess Lifecycle

- PID stored at `~/.config/prtr/watch.pid`
- `prtr watch --off` sends `SIGTERM` to the PID, then removes the PID file. If PID file is stale (process not running), it is removed silently.
- `prtr watch --status` checks if PID is alive and prints current state

### Suggestion Format

```
⚡ prtr: context ready
  • 2 failure lines
  • git diff: auth.js +12/-3
  • branch: fix/login-flow
  → prtr go fix [y/N]
```

Suggestion text is English only (same as all other prtr output). No i18n in v0.8.

User types `y` → shell runs `prtr go fix` with the pre-collected context. `N` or Enter → suggestion file removed, no action.

### Atomic Write for watch-suggest

The watcher writes the suggest file atomically to avoid race conditions when rapid events fire:
1. Write to `~/.config/prtr/watch-suggest.tmp`
2. `os.Rename` to `~/.config/prtr/watch-suggest` (atomic on POSIX)

Same pattern as existing `config.go`'s `writeFileAtomic`.

### Commands

```bash
prtr watch          # start background watcher (installs hook on first run)
prtr watch --off    # stop watcher (SIGTERM to PID)
prtr watch --status # show watcher state
```

### Config

```toml
# ~/.config/prtr/config.toml
[watch]
enabled = true
notify = false          # opt-in system notification (reserved, not implemented in v0.8)
medium_signals = false  # opt-in lint/large-diff suggestions (reserved, off by default)
```

### Privacy & `.prtrignore`

Watcher respects `.prtrignore` at repo root (simplified glob syntax — see Section 2). Files matching ignore patterns are excluded from auto-collected context. The following patterns are excluded by default without any file:

```
.env
.env.*
*.key
*.pem
*secret*
*password*
```

### Implementation Notes

- New package: `internal/watcher` — subprocess entry point, socket server, event matcher, suggest-file writer (atomic)
- Shell hook template stored as a Go embed in `internal/watcher/hook.zsh` and `hook.bash`
- Context collection reuses existing `repoctx` and `input` packages
- Test output stored by shell hook to `$TMPDIR/prtr-last-output`; `internal/watcher` reads this path
- `socat` used for socket IPC; polling fallback via temp file if `socat` not available
- No new Go module dependencies
- Windows: `prtr watch` prints "not supported on Windows" and exits 0

---

## 2. Context Engine Enhancement

### Current State

`prtr go` already attaches lightweight repo context (repo name, branch, changed files summary) via `internal/repoctx`.

### v0.8 Enhancement

When `prtr go` fires (or `prtr watch` pre-collects), the context engine auto-assembles:

| Source | What's collected | Limit |
|--------|-----------------|-------|
| git diff | Staged + unstaged changes | 200 lines |
| Test output | Contents of `$TMPDIR/prtr-last-output` if file exists | Last 50 lines |
| Branch name | Already supported | — |
| Changed files summary | Already supported | — |

### Wiring into `prtr go`

The new context is injected at the `prepareRun` stage in `internal/app/app.go`. Specifically:

1. `internal/repoctx` gains two new exported functions: `GitDiff(root string) (string, error)` and `LastTestOutput() (string, error)`
2. `prepareRun` calls both, applies `.prtrignore` filtering, and appends the results to the existing `RepoContext` string before it is passed to `internal/template`
3. The `--no-context` flag continues to disable all automatic context including these new sources

### `.prtrignore` Parser

Syntax: simplified glob only — no directory traversal patterns, no negation. Supported forms:

```
.env          # exact filename match anywhere in tree
.env.*        # glob suffix match
*secret*      # glob contains match
*.key         # glob extension match
```

Implemented without external dependencies using Go's `path.Match` per filename segment. A comment in the parser documents the unsupported gitignore features (negation `!`, `**`, directory-only `/`).

### Transparency via `prtr inspect`

`prtr inspect` already shows the assembled prompt. With v0.8, it also prints a `CONTEXT BUNDLE` section showing which sources were included and which files were excluded by `.prtrignore`:

```
CONTEXT BUNDLE
  git diff:     auth.js (+12/-3)  [included]
  test output:  $TMPDIR/prtr-last-output  [included, 18 lines]
  .env:         [excluded by .prtrignore]
```

Format is plain text (same style as existing `--explain` output). No JSON changes needed.

---

## 3. `--deep` Pipeline Visualization

### Current State

`--deep` runs 5 workers via `internal/deep/runtime.go`. Currently, all five `emit()` calls (which fire the `run.Options.Progress` callback) are issued **before** `graphFactory().Run()` is called — as pre-announcements, not as workers complete. This means in the current code, all 5 stages would appear to fire simultaneously at startup.

**v0.8 requires a refactor of `runtime.go`:** move each `emit()` call to fire inside its corresponding worker function (or immediately before the worker goroutine returns), so the Progress callback reflects actual per-stage completion timing. This is a prerequisite for meaningful TUI visualization.

### v0.8 Enhancement

The `Progress` callback is wired to a bubbletea model instead of being a no-op.

**Stage display:**
```
$ prtr take patch --deep

[████████░░░░░░░] 2/5
planner ✓  →  patcher ⠼  →  critic ○  →  tester ○  →  reconciler ○
patcher: 변경 범위 분석 중 (3s)
```

**Stage states:**
- `○` — pending (dim, `#484f58`)
- `⠼` — running (animated charmbracelet spinner, yellow)
- `✓` — complete (green)
- `✕` — failed (red), pipeline halts

### Implementation Notes

- Prerequisite: refactor `runtime.go` to move `emit()` calls inside workers (before the TUI work begins)
- New file: `internal/deep/ui.go` — bubbletea model with a `chan deeprun.Progress` field (package path: `internal/deep/run`)
- `runtime.go` constructs the bubbletea program, passes its channel as `run.Options.Progress` callback
- Uses `charmbracelet/bubbles` progress bar (already in `go.mod`)
- Uses `charmbracelet/lipgloss` for stage token colors (already in `go.mod`)
- Falls back to plain sequential log lines when `--no-color` flag is set or stdout is not a TTY (`!term.IsTerminal(os.Stdout.Fd())`)
- The existing `deepevent.Append` (events.jsonl) continues unchanged — the TUI is additive

---

## 4. `prtr dashboard` — Minimal Home Screen

### Behavior

Running `prtr` with no arguments and no stdin currently routes to `runMain` via `shouldRunRootDirect`. Inside `runMain`, `input.Resolve` returns a `usageError` when no prompt text is present.

**Required routing change in `app.go`:** In the `Execute()` entry point, intercept the no-args + no-stdin case **before** calling `shouldRunRootDirect`. If `len(os.Args) == 1` and stdin is not a pipe (`!term.IsTerminal(os.Stdin.Fd())` is false), launch the dashboard TUI directly and return. This avoids touching `shouldRunRootDirect` logic.

Additionally, add `"watch"` to the known-subcommands switch inside `shouldRunRootDirect` so `prtr watch` is not misrouted as a shortcut prompt.

**Dashboard display:**
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

### Key Binding Behavior

Pressing a key (e.g., `g`) exits the dashboard TUI and **exec-replaces the process with `prtr go`**, passing control back to the shell's TTY. This avoids nested bubbletea sessions and keeps the interaction model simple.

`q` / `Ctrl-C` exits cleanly with code 0.

`prtr --help` continues to work (cobra handles `--help` before `shouldRunRootDirect`).
`prtr help` continues to work as a subcommand.

### Implementation Notes

- New file: `cmd/prtr/dashboard.go` — bubbletea model
- Reads `config.Load()` for target, llm_provider
- Reads `~/.config/prtr/watch.pid` existence for watch status
- Key dispatch: bubbletea model's `Update` returns a `tea.Quit` command, then after `p.Run()` returns, `syscall.Exec` is called to replace the process with `prtr <command>` (Unix/macOS)
- Windows limitation (known, acceptable for v0.8): `syscall.Exec` is not available on Windows. On Windows, the dashboard exits and spawns `prtr <command>` as a child process via `os.StartProcess`, then waits for it. The behavior differs slightly (parent process briefly visible) but is functionally correct.
- **`internal/config` struct change required:** add `Watch WatchConfig` field to `fileConfig` struct, where `WatchConfig` has `Enabled bool`, `Notify bool`, `MediumSignals bool` with TOML keys `enabled`, `notify`, `medium_signals`

---

## Architecture Summary

```
prtr watch (subprocess, internal/watcher)
  └─ Unix socket ← shell hook (precmd/PROMPT_COMMAND)
       └─ event matcher (exit code + output patterns)
            └─ context engine (repoctx + input + .prtrignore filter)
                 └─ watch-suggest file → shell hook reads → y/N → prtr go <action>

prtr go / take
  └─ context engine (enhanced: git diff 200L + last test output 50L)
       └─ .prtrignore filter (path.Match, no external deps)
            └─ existing translate → template → deliver pipeline

prtr take --deep
  └─ internal/deep/runtime.go
       └─ run.Options.Progress callback → chan deep.Progress
            └─ bubbletea TUI (internal/deep/ui.go)
                 └─ bubbles progress bar + lipgloss stage tokens

prtr (no args, no stdin)
  └─ dashboard TUI (cmd/prtr/dashboard.go)
       └─ keypress → syscall.Exec prtr <command>

prtr inspect
  └─ existing prompt preview + new CONTEXT BUNDLE section
```

---

## Out of Scope for v0.8

- System notifications (`watch.notify`) — config key reserved, not implemented
- Medium signal watch events (lint, large diff) — config key reserved, off by default
- iTerm2 support
- GUI AI app automation
- Team sharing / sync
- `.prtrignore` negation (`!`) or `**` glob patterns
