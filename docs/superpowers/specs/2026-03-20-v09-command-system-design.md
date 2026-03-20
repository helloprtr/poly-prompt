# Design Spec: prtr v0.9 — Command System Redesign + AI Response Auto-Capture

**Date:** 2026-03-20
**Status:** Approved
**Scope:** Command system redesign (`resume`, `again` alias, `take` source change, `inspect --response`) + AI response auto-capture (B)

---

## Problem

The current command surface has two gaps:

1. **No conversation continuation.** After `prtr go`, users must manually copy the AI's response and re-paste it into the next request. There is no native "continue this conversation" command.

2. **Manual response bridging.** `prtr take` reads from the clipboard, requiring the user to copy the AI's response before every follow-up. Terminal AI users (claude-code, gemini-cli) never need to copy — the output is already in the terminal — but prtr doesn't capture it.

---

## Goals

- Add `prtr resume` for conversation continuation
- Retire `again` as a visible command by making it a hidden alias of `resume`
- Automatically capture AI responses so `take` and `resume` require no manual copy step
- Make the captured state visible to both beginners (inline hint) and power users (`inspect --response`)

---

## Non-Goals

- Resuming from a specific history entry (future work)
- Storing multiple past responses (only last-response is kept)
- Windows clipboard watcher support in v0.9

---

## 1. Command System Changes

### 1.1 `prtr resume [message]` — new command

Sends a follow-up message using the last captured AI response as context.

```
prtr resume "더 자세히 설명해줘"
prtr resume              # no-arg: re-runs the last prompt (same as current `again`)
```

**Prompt construction:**

```
[last-response content]

---
[user message]
```

When called with no arguments, `resume` re-runs the last stored prompt verbatim (preserving current `again` behavior).

**Delivery:** same mechanism as `go` — opens the AI app and pastes the composed prompt.

### 1.2 `again` → hidden alias of `resume`

`again` is registered as a hidden subcommand that routes directly to `resume`. It remains fully functional for existing users but disappears from `prtr --help`.

```
prtr again              →  prtr resume        (no-arg: rerun last prompt)
prtr again "다시해줘"   →  prtr resume "다시해줘"
```

No behavior change for existing `again` users.

### 1.3 `prtr take <action>` — source change only

`take` reads from `last-response.json` instead of the clipboard directly. Clipboard remains the fallback.

```
prtr take "테스트도 짜줘"
  → reads last-response.json (if present and fresh)
  → falls back to clipboard if not
```

**`take` vs `resume` distinction:**

| Command | Intent | Prompt shape |
|---------|--------|-------------|
| `take "테스트 짜줘"` | Use AI response as *material* for a new task | response as context + new task |
| `resume "더 자세히"` | *Continue* the same conversation thread | response as context + follow-up message |

The mechanical difference is framing: `take` starts a new task, `resume` continues a thread.

### 1.4 `prtr inspect --response` — new flag

Previews the currently captured response.

```
$ prtr inspect --response
Source:      clipboard
Captured:    2 minutes ago
───────────────────────────────
토큰 갱신 로직은 refresh token이 만료되기 전에...
(truncated at 500 chars — full content in ~/.config/prtr/last-response.json)
```

If no response is captured:

```
No response captured yet. Run prtr go, then copy or let prtr capture the AI's response.
```

---

## 2. AI Response Auto-Capture (B)

### 2.1 Terminal AI path

When `prtr go` launches a terminal AI (claude-code, gemini-cli, codex), the v0.8 `watch` shell hook is extended to also capture that command's output.

The hook writes the output to a temp file, and the hook's `precmd` callback writes to `last-response.json` after the AI command exits.

**Source field value:** `"terminal"`

### 2.2 GUI AI path (clipboard watcher)

When `prtr go` targets a GUI AI (Claude.app, browser), prtr starts a background clipboard watcher process.

**Behavior:**
- Polls clipboard every 500ms
- Detects a new AI response when: clipboard content changed AND length > 100 chars AND differs from the prompt just sent
- On detection: writes to `last-response.json` and exits
- Hard timeout: 5 minutes after `prtr go`, watcher exits regardless
- `prtr resume` or `prtr take` execution also signals the watcher to exit

**Deduplication:** watcher checks for a PID file at `~/.config/prtr/clipboard-watcher.pid` before starting. If the process is alive, it does not start a second instance.

**Source field value:** `"clipboard"`

**Failure mode:** if the watcher fails to start, it fails silently. `take` and `resume` fall back to reading the clipboard directly (existing behavior).

### 2.3 `last-response.json` format

```json
{
  "captured_at": "2026-03-20T14:23:00Z",
  "source": "terminal",
  "response": "토큰 갱신 로직은 refresh token이..."
}
```

**Location:** `~/.config/prtr/last-response.json`

`source` values: `"terminal"` | `"clipboard"`

### 2.4 Freshness and fallback

| Condition | Behavior |
|-----------|----------|
| `last-response.json` present, < 5 minutes old | Use as-is |
| `last-response.json` present, ≥ 5 minutes old | Use, but warn: `Using response from 8 minutes ago` |
| `last-response.json` absent | Fall back to clipboard |
| Clipboard also empty | Error: `No response captured yet. Run prtr go first, then copy the AI's response.` |

---

## 3. Discoverability

### 3.1 Post-capture hint (beginner UX)

After `prtr go` completes and a response is captured (either path), prtr prints a single hint line:

```
✓ Response captured — prtr resume to continue · prtr take to reuse
```

This hint is suppressed if `prtr go` is piped or if `--quiet` is set.

### 3.2 `inspect --response` (power user UX)

Described in §1.4. Lets power users verify the capture state before running `resume` or `take`.

---

## 4. New Files and Paths

```
~/.config/prtr/last-response.json      # captured AI response
~/.config/prtr/clipboard-watcher.pid   # watcher process ID
```

---

## 5. Compatibility

- `again` continues to work without any change in behavior
- `take` continues to work with clipboard as fallback — no change for users who manually copy
- `inspect` without `--response` is unaffected
- No existing commands are removed or renamed
