# prtr Work Capsule — Design Spec

**Date:** 2026-03-17
**Status:** Approved
**Version:** 0.8.x (separate worktree, after v0.8 base)
**Goal:** Single unified local state system for AI work continuation across sessions and AI apps.

---

## Problem

After closing Claude or ending a session, there is no way to resume "what I was doing in this repo." If working in Claude, switching to Codex or Gemini means losing context, progress, TODOs, and decisions.

---

## Solution: Work Capsule

One unified concept — `prtr save` / `prtr resume` — using a single local format called a **Work Capsule**.

No split between "memory" and "checkpoint". One thing to learn, one thing to use.

---

## CLI Behavior

### `prtr save`

```bash
prtr save                                         # auto-collect, no label
prtr save "auth refactor paused"                  # with label
prtr save --note "JWT decided; token refresh open"  # with note
prtr save "label" --note "note text"              # both
```

Output:
```
✓ capsule saved  cap_1773720001  auth refactor paused
  branch: fix/auth  sha: a1a1c21  3 todos
```

### `prtr status`

Shows the latest capsule state and detects repo drift.

Output (no drift):
```
last save:  2026-03-17 12:41  auth refactor paused
branch:     fix/auth  (no drift)
sha:        a1a1c21   (no drift)
todos:      2 open, 1 done
target:     claude
```

Output (drift detected):
```
last save:  2026-03-17 12:41  auth refactor paused
branch:     fix/auth → main  ⚠ branch changed
sha:        a1a1c21 → 8ff8a5a  ⚠ commits since save
todos:      2 open, 1 done
```

### `prtr list`

```bash
prtr list
```

Output:
```
cap_1773720001  2026-03-17 12:41  auth refactor paused    claude  3t
cap_1773718900  2026-03-17 12:31  [auto]                  gemini  5t  📌
cap_1773718300  2026-03-17 12:00  initial capsule         codex   1t
```

### `prtr resume`

```bash
prtr resume                                         # latest capsule, default target
prtr resume latest                                  # same
prtr resume cap_1773720001                          # specific capsule
prtr resume --to codex                              # latest, render for codex
prtr resume cap_1773720001 --to gemini
prtr resume cap_1773720001 --to gemini --dry-run    # print prompt, do not send
```

Behavior:
1. Load capsule.json
2. Detect drift (branch changed, sha changed, changed files differ)
3. Render resume prompt (template-based; LLM-enhanced if `llm_provider` is configured)
4. If drift exists, prepend a drift warning section to the prompt
5. Route through existing `prtr go` clipboard+launch path
6. `--dry-run` prints prompt to stdout and exits

### `prtr prune`

```bash
prtr prune                       # apply configured retention policy
prtr prune --older-than 30d      # delete entries older than 30 days (pinned excluded)
prtr prune --dry-run             # list what would be deleted
```

### Auto-save

Triggered automatically after successful: `prtr go`, `prtr swap`, `prtr again`, `prtr take`, `prtr take --deep`.

**Dedup rule:** if same repo + branch + normalized_goal + target_app and last auto-save within 10 minutes → update existing auto-save instead of creating new capsule.

Auto-save failures are silent (logged at debug level). They never block the main flow.

---

## Capsule Schema

### `.prtr/capsules/<id>/capsule.json`

```json
{
  "id": "cap_1773720001234567000",
  "label": "auth refactor paused",
  "note": "JWT decided; token refresh policy open",
  "kind": "manual",
  "pinned": false,
  "created_at": "2026-03-17T12:41:00Z",
  "updated_at": "2026-03-17T12:41:00Z",

  "repo": {
    "root": "/Users/koo/dev/myrepo",
    "name": "myrepo",
    "branch": "fix/auth",
    "head_sha": "a1a1c21",
    "touched_files": ["internal/auth/auth.go", "internal/config/config.go"],
    "diff_stat": "3 files changed, +47/-12"
  },

  "session": {
    "target_app": "claude",
    "engine": "deep",
    "mode": "patch",
    "source_history_id": "1773718302533411000",
    "source_run_id": "1773718305356602000",
    "artifact_root": ".prtr/runs/1773718305356602000"
  },

  "work": {
    "original_request": "implement JWT auth with token refresh",
    "normalized_goal": "implement JWT auth with token refresh",
    "next_action": "add refresh token rotation in internal/auth/token.go",
    "summary": "JWT base structure complete. Token refresh policy undecided.",
    "protected_terms": ["JWT", "Bearer"],
    "todos": [
      {"id": "plan", "title": "Design JWT structure", "status": "completed"},
      {"id": "patch", "title": "Token refresh logic", "status": "pending"},
      {"id": "tests", "title": "Write auth tests", "status": "pending"}
    ],
    "decisions": ["Use JWT (session-based approach rejected)"],
    "open_questions": ["What should the token refresh interval be?"],
    "risks": ["No token revocation on theft"]
  }
}
```

### `.prtr/capsules/<id>/summary.md`

Human-readable summary. Same format as `prtr resume --dry-run` output.

```markdown
# auth refactor paused
**Saved:** 2026-03-17 12:41 · branch: fix/auth · sha: a1a1c21

## What was being worked on
implement JWT auth with token refresh

## Progress
- ✓ Design JWT structure
- ○ Token refresh logic
- ○ Write auth tests

## Decisions made
- Use JWT (session-based approach rejected)

## Open questions
- What should the token refresh interval be?

## Risks
- No token revocation on theft

## Next action
add refresh token rotation in internal/auth/token.go
```

### ID Format

`cap_` prefix to distinguish from history IDs (raw UnixNano integers).

```
cap_1773720001234567000
```

Directory: `.prtr/capsules/cap_1773720001234567000/`

### `kind` Values

| Value | Meaning |
|---|---|
| `manual` | Created by `prtr save` |
| `auto` | Created automatically after `prtr go` / `prtr swap` / etc. |

Retention rules and dedup logic branch on this field.

---

## Config Shape

Add `[memory]` section to `config.toml`:

```toml
[memory]
enabled = true
auto_save = true
prune_on_write = true
prune_on_resume = true

capsule_retention_days = 30
autosave_retention_days = 14
run_retention_days = 7

max_capsules_per_repo = 200
max_storage_mb_per_repo = 256

store_diff = "stat"
```

### `store_diff` Options

| Value | Stored |
|---|---|
| `"stat"` (default) | `"3 files changed, +47/-12"` — one line summary |
| `"full"` | Actual git diff saved to `capsule/diff.patch` |
| `"none"` | No diff information |

### Go Struct

```go
type MemoryConfig struct {
    Enabled               bool   `toml:"enabled"`
    AutoSave              bool   `toml:"auto_save"`
    PruneOnWrite          bool   `toml:"prune_on_write"`
    PruneOnResume         bool   `toml:"prune_on_resume"`
    CapsuleRetentionDays  int    `toml:"capsule_retention_days"`
    AutosaveRetentionDays int    `toml:"autosave_retention_days"`
    RunRetentionDays      int    `toml:"run_retention_days"`
    MaxCapsulesPerRepo    int    `toml:"max_capsules_per_repo"`
    MaxStorageMBPerRepo   int    `toml:"max_storage_mb_per_repo"`
    StoreDiff             string `toml:"store_diff"`
}
```

Defaults provided by `defaultMemoryConfig()`.

### Retention Rules

| Condition | Behavior |
|---|---|
| `pinned = true` | Never auto-deleted |
| `kind = "manual"` | Apply `capsule_retention_days` |
| `kind = "auto"` | Apply `autosave_retention_days` |
| Storage cap exceeded | Delete oldest auto-saves first (pinned excluded) |
| Run referenced by pinned capsule | Never auto-deleted |
| Run referenced by any capsule | Exempt from `run_retention_days` deletion |

---

## File Mapping

### New Files

```
internal/capsule/
  capsule.go    — Capsule type, constants, zero values
  store.go      — filesystem CRUD (.prtr/capsules/)
  builder.go    — history + run + repoctx → Capsule assembly
  render.go     — Capsule → resume prompt (template-based; LLM optional)
  prune.go      — retention policy execution, storage calculation
```

### Modified Files

| File | Change |
|---|---|
| `internal/config/config.go` | Add `MemoryConfig`, `defaultMemoryConfig()`, wire into `Load()` |
| `internal/app/app.go` | Implement `runSave`, `runResume`, `runCapsuleStatus`, `runCapsuleList`, `runPrune` |
| `internal/app/command.go` | Register `save`, `resume`, `status`, `list`, `prune` cobra commands |
| `internal/repoctx/repoctx.go` | Add `HeadSHA()` for drift detection |

### Package Dependency Graph

```
internal/capsule/builder.go
  ├── internal/history      (read history entry)
  ├── internal/repoctx      (branch, sha, changed files)
  ├── internal/deep/run     (read run manifest)
  └── internal/config       (MemoryConfig)

internal/capsule/render.go
  ├── internal/capsule      (Capsule type)
  └── internal/config       (target template preset, LLMProvider)

internal/app/app.go
  └── internal/capsule      (store, builder, render, prune)
```

No circular dependencies. `capsule` package does not import `app`.

---

## Drift Detection

On `prtr resume` and `prtr status`, compare saved capsule state to current repo:

| Field | Check |
|---|---|
| `repo.branch` | `git rev-parse --abbrev-ref HEAD` |
| `repo.head_sha` | `git rev-parse --short HEAD` |
| `repo.touched_files` | `git status --short` changed files |

If any differ, display drift warning. On resume, prepend drift section to the generated prompt.

---

## Resume Prompt Generation

### Template-based (default)

Capsule fields are rendered into a structured prompt using the target app's template preset:

- `--to claude` → `claude-structured` template
- `--to codex` → `codex-implement` template
- `--to gemini` → `gemini-stepwise` template

Prompt includes: original request, current repo state, progress, TODOs, decisions, open questions, risks, next action, artifact paths to inspect.

### LLM-enhanced (optional)

When `llm_provider` is configured in `config.toml`, render.go calls the LLM to synthesize a target-optimized prompt from the same capsule data. Same opt-in pattern as existing deep run LLM enhancement.

---

## Tradeoffs

**① Auto-save added to `prtr go` success path**
Failure is silent and non-blocking. Capsule store injected as interface for test isolation.

**② `normalized_goal` extraction**
Default: lowercase normalization of first 100 chars of `original_request` (deterministic, no LLM needed). LLM path: one-sentence summary when `llm_provider` is set. Deterministic default is safer for dedup key use.

**③ Resume delivery path**
`render.go` generates the prompt string → reuses existing `runGo` clipboard+launch path. `--dry-run` prints to stdout and exits. No new delivery infrastructure needed.

**④ Runs directory retention**
Runs referenced by any capsule are exempt from `run_retention_days` auto-deletion. Pinned capsule references are permanent.

**⑤ Data size**
`capsule.json` stores summaries and path references only. Expected size: 2–5 KB per capsule. Heavy data (git diff, LLM responses) stays in `.prtr/runs/<id>/` and is referenced, not duplicated. `store_diff = "full"` opt-in for offline/archival use.

---

## Out of Scope

- Cloud sync or remote storage
- Team sharing
- Conversation log replay
- Cross-repo capsule search
- Capsule merge or diff
