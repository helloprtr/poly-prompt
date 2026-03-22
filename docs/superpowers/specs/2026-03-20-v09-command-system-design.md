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

**Prompt construction (with message):**

```
[last-response content]

---
[user message]
```

**No-arg behavior:** When called with no arguments, `resume` re-runs the last stored prompt verbatim. It reads the last prompt from the existing history store (same source that `again` uses today). It does **not** consult `last-response.json` — there is no response context needed for a plain re-run. If no prior prompt exists in history, `resume` exits with: `No previous prompt found. Run prtr go first.`

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

**Prompt construction:**

```
The following is an AI response:

[last-response content]

Using the above as context: [user action]
```

**`take` vs `resume` distinction:**

| Command | Intent | Prompt framing |
|---------|--------|----------------|
| `take "테스트 짜줘"` | Use AI response as *material* for a new task | "Using the above as context: [action]" |
| `resume "더 자세히"` | *Continue* the same conversation thread | response + `---` separator + follow-up message |

Both read from the same `last-response.json` source. The difference is in how the prompt is framed: `take` presents the response as reference material for a new task; `resume` presents it as prior conversation context.

### 1.4 `prtr inspect --response` — new flag

Previews the currently captured response. The `Source` label displays the raw `source` field value from `last-response.json` (`terminal` or `clipboard`).

```
$ prtr inspect --response
Source:      terminal
Captured:    2 minutes ago
───────────────────────────────
토큰 갱신 로직은 refresh token이 만료되기 전에...
(truncated at 500 chars — full content in ~/.config/prtr/last-response.json)
```

```
$ prtr inspect --response
Source:      clipboard
Captured:    8 minutes ago
───────────────────────────────
Here's the updated auth middleware...
(truncated at 500 chars — full content in ~/.config/prtr/last-response.json)
```

If no response is captured:

```
No response captured yet. Run prtr go, then copy or let prtr capture the AI's response.
```

---

## 2. AI Response Auto-Capture (B)

### 2.1 Terminal AI path

When `prtr go` launches a terminal AI (claude-code, gemini-cli, codex), the v0.8 `watch` shell hook is extended to capture that command's stdout+stderr output in full via tee into a temp file. The hook's `precmd` callback reads the temp file and writes its entire contents to `last-response.json` after the AI command exits.

The full stdout+stderr of the AI command is captured — no trimming, no delimiter detection. These tools write their final answer as the last portion of their output; downstream consumers (`take`, `resume`) receive everything and the AI app itself handles presentation.

**Source field value:** `"terminal"`

### 2.2 GUI AI path (clipboard watcher)

When `prtr go` targets a GUI AI (Claude.app, browser), prtr starts a background clipboard watcher process.

**Behavior:**
- Polls clipboard every 500ms
- Detects a new AI response when: clipboard content changed AND length > 100 chars AND differs from the prompt just sent
- On detection: writes to `last-response.json` and exits
- Hard timeout: 5 minutes after `prtr go`, watcher exits regardless
- `prtr resume` or `prtr take` execution also signals the watcher to exit (via the PID file — see below)

**Deduplication and stale PID handling:**

Before starting, the watcher checks `~/.config/prtr/clipboard-watcher.pid`:

1. If the file does not exist → start the watcher, write PID file.
2. If the file exists → read the PID and run `kill -0 <pid>` to check liveness.
   - Process alive → do not start a second instance, exit silently.
   - Process dead (stale PID file) → delete the stale file, start fresh, write new PID file.

On normal exit (detection, timeout, or signal), the watcher deletes the PID file.

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

The §2.4 table applies when `take` or `resume` (with a message argument) reads a response. `resume` with no arguments bypasses this table entirely — it reads from history, not from `last-response.json`.

| Condition | Behavior |
|-----------|----------|
| `last-response.json` present, < 5 minutes old | Use as-is |
| `last-response.json` present, ≥ 5 minutes old | Use, but warn: `Using response from 8 minutes ago` |
| `last-response.json` absent | Fall back to clipboard |
| Clipboard also empty | Error: `No response captured yet. Run prtr go first, then copy the AI's response.` |

---

## 3. Discoverability

### 3.1 Post-capture hint (beginner UX)

For the **terminal AI path**, `prtr go` waits for the AI command to exit, then the shell hook writes `last-response.json`. The hint is printed by `prtr go` synchronously after the command exits:

```
✓ Response captured — prtr resume to continue · prtr take to reuse
```

For the **clipboard watcher path**, `prtr go` returns immediately and the watcher runs in the background. The watcher cannot write directly to the user's terminal. Instead, it uses the same IPC mechanism defined in v0.8 `watch`: on detection it writes a notification file at `~/.config/prtr/watch-suggest`, and the shell's `precmd` hook reads and prints it on the next prompt draw. The notification content is the same hint line above.

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
