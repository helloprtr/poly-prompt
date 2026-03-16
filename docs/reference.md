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
