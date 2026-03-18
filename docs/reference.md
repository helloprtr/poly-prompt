# prtr Command Reference

## Commands overview

| Command | One-line description |
|---|---|
| `prtr go [mode] <prompt>` | Translate intent, add context, compile prompt, open AI app |
| `prtr swap <app>` | Resend the last prompt to a different AI app |
| `prtr take <action>` | Turn clipboard content into a structured follow-up prompt |
| `prtr take <action> --deep` | Run the five-worker pipeline before delivery |
| `prtr again` | Replay the last run exactly |
| `prtr learn [paths...]` | Build or update the repo termbook of protected terms |
| `prtr inspect <prompt>` | Preview the compiled prompt without sending |
| `prtr demo` | Safe offline preview of the prtr loop |
| `prtr start` | Guided first send with onboarding if needed |
| `prtr setup` | Interactive full configuration |
| `prtr lang` | Update language defaults only |
| `prtr doctor` | Check which features are available on this system |
| `prtr init` | Create the default config file |
| `prtr version` | Print the installed version |
| `prtr history` | List recent runs |
| `prtr history search <query>` | Search past runs |
| `prtr rerun <id>` | Replay a specific history entry |
| `prtr templates list` | List available template presets |
| `prtr templates show <name>` | Show a template preset's content |
| `prtr profiles list` | List saved profiles |
| `prtr profiles use <name>` | Apply a profile's settings as new defaults |
| `prtr watch` | Start the background context watcher (v0.8) |
| `prtr watch --off` | Stop the background watcher (v0.8) |
| `prtr watch --status` | Show watcher state (v0.8) |
| `prtr save [label]` | Save current work state as a capsule (v0.8) |
| `prtr resume [id]` | Restore a saved capsule and continue (v0.8) |
| `prtr status` | Show latest capsule state and drift (v0.8) |
| `prtr list` | List all capsules for this repo (v0.8) |
| `prtr prune` | Delete old capsules per retention policy (v0.8) |

> **v0.8 note:** Context watcher and Work Capsule commands require `memory.enabled = true` in your config (the default). Set `memory.enabled = false` to disable all capsule operations.

---

## prtr go

Send a prompt to an AI app with optional stdin evidence and repo context.

**Usage:**

```bash
prtr go [mode] [flags] <prompt>
<stdin> | prtr go [mode] [flags] [prompt]
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--to <app>` | string | last used app | Target AI app: `claude`, `gemini`, or `codex` |
| `--dry-run` | bool | false | Compile and print; skip clipboard copy and launch |
| `--no-context` | bool | false | Skip repo context and termbook attachment |
| `--edit` | bool | false | Open compiled prompt in editor before delivery |

**Modes (positional argument before the prompt):**

| Mode | Framing |
|---|---|
| `fix` | Root cause analysis focused |
| `review` | Risk and regression focused |
| `design` | Architecture and structure focused |
| (none) | General request |

**Stdin behavior:**

| Input | Result |
|---|---|
| Message only | Message is the prompt |
| Stdin only | Stdin becomes the prompt |
| Stdin + message | Message is the prompt; stdin is appended as evidence |
| Neither | Error: missing prompt text |

---

## prtr swap

Resend the last prompt to a different AI app, recompiling the template for the new target.

**Usage:**

```bash
prtr swap <app> [flags]
```

**Arguments:**

| Argument | Values |
|---|---|
| `<app>` | `claude`, `gemini`, `codex` |

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--edit` | bool | false | Open the recompiled prompt in editor before delivery |
| `--dry-run` | bool | false | Print the recompiled prompt; skip clipboard and launch |

---

## prtr take

Read clipboard content and build a structured follow-up prompt.

**Usage:**

```bash
prtr take <action> [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--to <app>` | string | last used app | Target AI app |
| `--dry-run` | bool | false | Print the prompt; skip clipboard and launch |
| `--edit` | bool | false | Open compiled prompt in editor before delivery |
| `--deep` | bool | false | Run the five-worker pipeline before delivery |
| `--llm <provider>` | string | see config | Format the deep delivery prompt for a specific provider |

**Classic actions (no --deep required):**

| Action | What the follow-up prompt asks for |
|---|---|
| `patch` | Apply the described change file by file |
| `test` | Write tests covering the described behavior |
| `debug` | Identify and fix the root cause |
| `refactor` | Refactor the described code safely |
| `commit` | Draft a commit message from the change |
| `summary` | Summarize the key points of the answer |
| `clarify` | Ask clarifying questions about the answer |
| `issue` | Draft a GitHub issue from the described problem |
| `plan` | Expand the answer into a sequenced plan |

---

## prtr take --deep

Run the five-worker internal pipeline, then deliver the assembled prompt to an AI app.

**Usage:**

```bash
prtr take <action> --deep [flags]
```

**Additional flags (used with --deep):**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--llm <provider>` | string | resolved from config | Format the delivery prompt for a specific AI provider |

**--llm provider options:**

| Value | Prompt format | Characteristic tags / structure |
|---|---|---|
| `claude` | XML semantic tags | `<role>`, `<context>`, `<task>`, `<risks>`, `<constraints>`, `<validation>` |
| `gemini` | Markdown headers | `## Role`, `## Source Intent`, `## Patch Bundle`, `## Task`, `## Risks`, `## Validation` |
| `codex` | Numbered list + diff block | Numbered instructions, fenced ` ```diff ``` ` block |
| (none/empty) | Universal Markdown | `## Goal`, `## Source Intent`, `## Key Risks`, `## Tests Required`, `## Verify` |

**Deep actions:**

| Action | Pipeline objective | Result type |
|---|---|---|
| `patch` | Draft and critique an implementation patch | `PatchBundle` |
| `test` | Plan and draft test cases | `TestBundle` |
| `debug` | Identify root cause and draft fix | `DebugBundle` |
| `refactor` | Define refactor scope, safety, and rollback plan | `RefactorBundle` |

**Worker pipeline (always runs all five in order):**

| Step | Worker | Blocker type | Reads | Writes |
|---|---|---|---|---|
| 1 | `planner` | Hard | `source.md`, evidence files | `plan.json` |
| 2 | `patcher` | Hard | `source.md`, `plan.json`, `evidence/git.diff` | `result/patch.diff`, `workers/patcher/result.json` |
| 3 | `critic` | Soft | `workers/patcher/result.json` | `workers/critic/result.json` |
| 4 | `tester` | Soft | `workers/patcher/result.json`, `evidence/memory.json` | `result/tests.md`, `workers/tester/result.json` |
| 5 | `reconciler` | Hard | patcher + critic + tester outputs | `result/patch_bundle.json`, `workers/reconciler/result.json` |

Critic and tester run concurrently after the patcher completes. A hard blocker failure stops the run entirely. A soft blocker failure allows the run to complete with warnings recorded in the bundle.

**Note on --dry-run with --deep:** The pipeline runs fully and all artifacts are written to disk. Only clipboard copy and app launch are skipped.

---

## prtr again

Replay the last run with the same prompt, target, and settings.

**Usage:**

```bash
prtr again [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--edit` | bool | false | Open the last prompt in editor before resending |
| `--dry-run` | bool | false | Print the prompt; skip clipboard and launch |

---

## prtr learn

Scan repo source files and build a termbook of protected identifiers.

**Usage:**

```bash
prtr learn [paths...] [flags]
```

**Arguments:**

| Argument | Description |
|---|---|
| `[paths...]` | Optional list of files or directories to scan; defaults to repo root |

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--dry-run` | bool | false | Print the termbook without writing it to disk |
| `--reset` | bool | false | Rebuild from scratch, discarding previous entries |

**Termbook location:** `.prtr/termbook.toml` at repo root.

---

## prtr inspect

Compile and preview the prompt without sending it anywhere.

**Usage:**

```bash
prtr inspect [flags] <prompt>
<stdin> | prtr inspect [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `-t <app>` | string | default target | Resolve for this target app |
| `-r <role>` | string | default role | Resolve with this role |
| `--template <preset>` | string | default preset | Use this template preset |
| `--explain` | bool | false | Print how each setting was resolved |
| `--diff` | bool | true | Show before/after translation diff |
| `--json` | bool | false | Output the full resolved run as JSON |

`inspect` never copies, launches, or pastes.

---

## Configuration reference

### Config file locations

| Path | Priority | Description |
|---|---|---|
| `$XDG_CONFIG_HOME/prtr/config.toml` | 1 (if `XDG_CONFIG_HOME` is set) | User config |
| `~/.config/prtr/config.toml` | 1 (default) | User config |
| `.prtr.toml` (repo root) | 2 (project override) | Project-local overrides |

### Top-level fields

| Field | Type | Default | Description |
|---|---|---|---|
| `deepl_api_key` | string | `""` | DeepL API key for translation; optional |
| `translation_source_lang` | string | `"auto"` | Input language: `auto`, `ko`, `ja`, `zh`, `en` |
| `translation_target_lang` | string | `"en"` | Output language: `en`, `ja`, `zh`, `de`, `fr` |
| `default_target` | string | `"claude"` | Default AI app: `claude`, `gemini`, `codex` |
| `default_role` | string | `""` | Default role: `be`, `review`, `writer` |
| `default_template_preset` | string | `"claude-structured"` | Default template preset name |
| `llm_provider` | string | `""` | Deep mode prompt format: `claude`, `gemini`, `codex`, or `""` for rule-based universal Markdown |

### [targets.\<name\>] fields

| Field | Type | Description |
|---|---|---|
| `family` | string | AI family identifier (`claude`, `gemini`, `codex`) |
| `default_template_preset` | string | Template preset to use for this target |
| `translation_target_lang` | string | Output language override for this target |
| `default_delivery` | string | Default delivery mode (`open-copy`) |

### [launchers.\<name\>] fields

| Field | Type | Description |
|---|---|---|
| `command` | string | CLI command to launch (`claude`, `gemini`, `codex`) |
| `args` | []string | Extra arguments passed to the command |
| `paste_delay_ms` | int | Milliseconds to wait before paste after launch |
| `submit_mode` | string | Submit behavior: `manual` |

### Project-local .prtr.toml

```toml
[shortcuts.review]
target = "claude"
role = "review"
template_preset = "claude-review"
translation_target_lang = "en"

[profiles.incident]
target = "gemini"
role = "be"
template_preset = "gemini-stepwise"
context = "Focus on mitigation, rollback, and customer impact."
```

---

## Environment variables

| Variable | Description | Example |
|---|---|---|
| `DEEPL_API_KEY` | DeepL API key; takes precedence over `deepl_api_key` in config | `DEEPL_API_KEY=abc123:fx` |
| `PRTR_LLM_PROVIDER` | Default LLM provider for `--deep` formatting; lowest priority (overridden by config and `--llm` flag) | `PRTR_LLM_PROVIDER=claude` |
| `XDG_CONFIG_HOME` | Base directory for user config; if set, prtr reads from `$XDG_CONFIG_HOME/prtr/config.toml` | `XDG_CONFIG_HOME=$HOME/.config` |

**Priority order for `llm_provider` resolution:**

```
--llm flag  >  config.toml llm_provider  >  PRTR_LLM_PROVIDER env var
```

---

## Deep mode artifacts

All artifacts are written to `.prtr/runs/<run-id>/` relative to your repo root. If prtr cannot find a repo root, artifacts are written to the prtr data directory (`~/.local/share/prtr/runs/<run-id>/` on Linux, `~/Library/Application Support/prtr/runs/<run-id>/` on macOS).

### Root artifacts

| File | Description |
|---|---|
| `manifest.json` | Canonical run record: ID, version, action, engine, status, result type, target app, delivery mode, source kind, parent history ID, repo root, artifact root, event log path, created/updated/completed timestamps, embedded WorkPlan, result ref, warning count, error message |
| `plan.json` | The WorkPlan produced by the planner worker: version, action, result type, summary, evidence refs, todo list, and worker dependency graph |
| `lineage.json` | Provenance record: parent history ID, source kind, target app |
| `events.jsonl` | Append-only JSONL event log; one JSON object per line, each with `type`, `timestamp`, and `data` |

### Evidence files (read-only inputs)

| File | Description |
|---|---|
| `evidence/source.md` | The clipboard text that started this run |
| `evidence/repo_context.json` | Repo name, branch, and list of changed files at run time |
| `evidence/history.json` | The parent history entry (the most recent `prtr go` or `prtr take`) if one existed; `{}` otherwise |
| `evidence/memory.json` | Protected terms extracted from the repo termbook |
| `evidence/git.diff` | Output of `git diff -- .` run in the repo root at run time |

### Result artifacts (final outputs)

| File | Description |
|---|---|
| `result/patch_bundle.json` | Final PatchBundle: summary, diff, touched files, risks (title/severity/detail/confidence per item), test plan (test cases, edge cases, verification steps), open questions, warnings |
| `result/patch.diff` | Diff skeleton extracted or inferred from the clipboard source |
| `result/tests.md` | Test plan in Markdown: Test Cases, Edge Cases, Verification Steps |
| `result/summary.md` | Human-readable Markdown summary of the patch bundle |

### Per-worker artifacts

| File | Description |
|---|---|
| `workers/planner/request.json` | Inputs passed to the planner |
| `workers/planner/result.json` | WorkPlan written by the planner |
| `workers/patcher/request.json` | Inputs passed to the patcher |
| `workers/patcher/result.json` | PatchDraft written by the patcher |
| `workers/critic/request.json` | Inputs passed to the critic |
| `workers/critic/result.json` | RiskReport written by the critic |
| `workers/tester/request.json` | Inputs passed to the tester |
| `workers/tester/result.json` | TestPlan written by the tester |
| `workers/reconciler/request.json` | Inputs passed to the reconciler |
| `workers/reconciler/result.json` | Final PatchBundle written by the reconciler |

### Event types (events.jsonl)

| Event type | When it is emitted |
|---|---|
| `run.started` | Start of the deep run |
| `context.compiled` | Evidence files have been written |
| `plan.created` | Planner worker completed; WorkPlan is available |
| `todo.updated` | A todo item's status changed |
| `worker.started` | A worker began execution |
| `worker.completed` | A worker finished execution |
| `artifact.ready` | A result artifact has been written to disk |
| `approval.requested` | Run is paused awaiting approval (reserved for future use) |
| `approval.granted` | Approval was given (reserved for future use) |
| `delivery.started` | Prompt is being copied and the app is being launched |
| `delivery.completed` | Delivery finished |
| `memory.suggested` | New terms were suggested for the termbook |
| `run.completed` | Run finished successfully (or with warnings) |
| `run.failed` | Run failed due to a hard blocker |

---

## prtr watch (v0.8)

Start a background context watcher that tracks shell activity and keeps repo context current for fast `prtr go` runs.

**Usage:**

```bash
prtr watch
prtr watch --off
prtr watch --status
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--off` | bool | false | Stop the running watcher |
| `--status` | bool | false | Print watcher state (active/inactive and PID) |

**Behavior:**

- On start, installs a shell hook in the detected shell config file (`~/.zshrc` or `~/.bashrc`) and prints instructions to reload the shell.
- Runs as a foreground process; use `prtr watch --off` or `SIGTERM` to stop it.
- On `--off`: sends `SIGTERM` to the recorded PID and removes the PID file. If the watcher is not running, prints `prtr watch: not running`.
- On `--status`: prints `prtr watch: active (PID <n>)` or `prtr watch: inactive`.
- Not available on Windows.

---

## prtr save (v0.8)

Capture the current work state as a capsule: repo branch, HEAD SHA, last AI run, open todos, and session metadata.

**Usage:**

```bash
prtr save [label] [flags]
```

**Arguments:**

| Argument | Description |
|---|---|
| `[label]` | Optional human-readable label for the capsule |

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--note <text>` | string | `""` | Attach a free-text note to the capsule |

**Behavior:**

- Requires `memory.enabled = true` (config default).
- Capsules are saved to `.prtr/capsules/` at the repo root (or `~/Library/Application Support/prtr/capsules/` when no repo is found).
- If `prune_on_write = true`, runs `prtr prune` automatically after saving.
- Prints: `✓ capsule saved  <id>  <label>  branch: <branch>  sha: <sha>  <n> todos`
- prtr also saves a capsule automatically after every successful `go`, `swap`, `again`, or `take` run. Auto-saved capsules have a blank label shown as `[auto]`.

---

## prtr resume (v0.8)

Restore a saved capsule and build a structured resume prompt, then deliver it to an AI app.

**Usage:**

```bash
prtr resume [id] [flags]
```

**Arguments:**

| Argument | Description |
|---|---|
| `[id]` | Capsule ID prefix to restore; omit to use the latest capsule |

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--to <app>` | string | capsule's recorded app | Target AI app to deliver the resume prompt to |
| `--dry-run` | bool | false | Print the resume prompt; skip clipboard and launch |

**Behavior:**

- Loads the specified capsule (or the most recent one if no ID is given).
- Computes repo drift: if the current branch, HEAD SHA, or working tree differs from the saved state, appends a drift warning to the resume prompt.
- Prints: `✓ resume prompt copied  <label>  → <app>` and a drift warning if applicable.
- If `prune_on_resume = true`, runs `prtr prune` automatically after resuming.

---

## prtr status (v0.8)

Show the latest capsule and current repo drift.

**Usage:**

```bash
prtr status
```

Prints the most recent capsule's ID, label, timestamp, branch, HEAD SHA, open/done todo counts, and target app. If the repo has drifted since the save, prints a drift summary.

---

## prtr list (v0.8)

List all capsules saved for the current repo.

**Usage:**

```bash
prtr list
```

Prints each capsule as one line: `<id>  <timestamp>  <label>  <todos>  <pin-mark>`. Pinned capsules are marked with `★`. Most recent first.

---

## prtr prune (v0.8)

Delete old capsules according to the retention policy.

**Usage:**

```bash
prtr prune [flags]
```

**Flags:**

| Flag | Type | Default | Description |
|---|---|---|---|
| `--older-than <duration>` | string | config `retention` | Delete capsules older than this duration (e.g. `30d`, `7d`) |
| `--dry-run` | bool | false | Print capsules that would be deleted; do not delete |

**Behavior:**

- Pinned capsules are never deleted by prune.
- Default retention comes from `memory.retention` in config (default: `30d`).
- Prints each deleted capsule ID and label, and a summary count.

---

## Work Capsule config fields (v0.8)

These fields live under `[memory]` in `config.toml` or `.prtr.toml`.

| Field | Type | Default | Description |
|---|---|---|---|
| `memory.enabled` | bool | `true` | Enable or disable all capsule operations |
| `memory.retention` | string | `"30d"` | Default retention window for `prtr prune` |
| `memory.prune_on_write` | bool | `false` | Auto-prune after every `prtr save` |
| `memory.prune_on_resume` | bool | `false` | Auto-prune after every `prtr resume` |
