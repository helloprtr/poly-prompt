# prtr User Guide

## What is prtr?

prtr is the command layer for AI work. It sits between your terminal and your AI tool — Claude, Codex, or Gemini — and handles the friction of each step in the loop: translating your intent into a well-formed prompt, adding repo context, copying to the clipboard, launching the right app, and turning the answer back into the next action. It is not a chatbot wrapper. You still talk to the AI directly. prtr structures what goes in and organizes what comes out, so each pass through the loop is faster and less manual than the last. The central idea is that your clipboard is a handoff point: prtr writes a prompt to it before you send, and reads the answer from it when you take the next step.

---

## Installation

### Homebrew (macOS — recommended)

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

### GitHub Releases (Linux / Windows)

Download the archive for your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases), extract, and place the `prtr` binary on your `PATH`.

### Build from source

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

Verify the installation:

```bash
prtr version
```

---

## Your first 5 minutes

### Step 1 — safe preview, no setup required

```bash
prtr demo
```

This prints a sample loop to your terminal and shows you what a compiled prompt looks like. Nothing is sent anywhere. No API key is required.

Expected output:

```
prtr demo
The command layer for AI work.
No API key required. Safe preview only.

Loop:
  npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
  prtr swap gemini
  prtr take patch

Sample input:
  request (ko): 왜 깨지는지 정확한 원인만 찾아줘
  evidence: npm test output

Preview prompt:
  ...

Try next:
  prtr go "explain this error" --dry-run
  prtr setup
```

### Step 2 — preview a real prompt

```bash
prtr go "explain this error" --dry-run
```

`--dry-run` compiles the prompt and prints it to stdout without opening any app. Use this until you are comfortable with what gets generated.

### Step 3 — set up your defaults

```bash
prtr setup
```

`setup` asks for:
- DeepL API key (optional — AI apps handle multilingual input natively without one)
- Default input language
- Default output language
- Default app (claude / gemini / codex)
- Default role
- Default template preset

If you only want to adjust language defaults later:

```bash
prtr lang
```

### Step 4 — your first real send

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
```

This pipes your test output as evidence, translates your intent into English, attaches repo context, compiles the prompt, copies it to the clipboard, and opens your default AI app.

### Step 5 — take the next action

After you get an answer in your AI app and copy it:

```bash
prtr take patch
```

This reads from your clipboard, builds a follow-up prompt for applying the patch, and sends it.

---

## Core commands

### prtr go

`go` is the main send command. It takes your intent — in any language — translates it to English, adds repo context and evidence from stdin, compiles a template-driven prompt, copies it to the clipboard, and opens your AI app.

**Modes** control the framing of your request:

```bash
prtr go "이 에러 원인 분석해줘"
prtr go review "이 PR에서 위험한 부분만 짚어줘"
prtr go fix "왜 테스트가 깨지는지 원인만 찾아줘"
prtr go design "이 기능 구조 설계해줘"
```

**Routing to a specific app:**

```bash
prtr go "이 구조 문제점 봐줘" --to claude
prtr go fix "왜 테스트가 깨지는지 봐줘" --to codex
prtr go review "이 설계 위험도 평가해줘" --to gemini
```

**Available flags:**

| Flag | Description |
|---|---|
| `--to <app>` | Send to `claude`, `codex`, or `gemini` |
| `--dry-run` | Compile and print the prompt; skip clipboard and launch |
| `--no-context` | Skip automatic repo context and termbook attachment |
| `--edit` | Open the compiled prompt in your editor before delivery |

**Piping stdin:**

When you pipe text, it becomes evidence attached alongside your message. If you pass no message but do pipe text, the piped text becomes the request itself.

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
cat crash.log | prtr go fix
```

**What repo context includes:**

- repo name
- current branch
- list of changed files

**Example stderr output on a successful send:**

```
-> go:fix | claude | prompt+repo | copy+open+paste
```

---

### prtr swap

`swap` resends the last request to a different AI app. Use it when you want to compare how two models handle the same problem. prtr recompiles the prompt for the destination app's template rather than reusing the previous app's format blindly.

```bash
prtr swap gemini
prtr swap codex
prtr swap claude --edit
prtr swap gemini --dry-run
```

The original request text, mode, and language settings are preserved. Only the target app and its template change.

---

### prtr take (classic)

`take` reads an answer from your clipboard and builds a structured follow-up prompt for the next action. Use it when you have already copied a useful AI response and want to move directly into the next step without composing a new request from scratch.

```bash
prtr take patch
prtr take test --to codex
prtr take commit --dry-run
prtr take summary --edit
```

**Classic actions:**

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

`--to` overrides the delivery target for a single run. Without it, prtr uses the target from your most recent history entry.

---

### prtr take --deep

`--deep` runs a multi-step internal pipeline before delivery. Instead of building one follow-up prompt from the clipboard text directly, it runs five sequential workers that produce structured artifacts, then assembles a final delivery prompt from their combined output.

#### What --deep does vs classic take

Classic `take` reads your clipboard and generates a prompt in one step. `--deep` reads your clipboard, runs analysis, risk review, and test planning across five workers, writes all intermediate results to disk, then produces a delivery prompt that incorporates everything.

Use `--deep` when:
- the answer you copied describes a non-trivial code change
- you want risk analysis and a test plan automatically attached to your next prompt
- you want to inspect exactly what went into the delivery (everything is written to disk)

#### The 5-worker pipeline

When you run `prtr take patch --deep`, these five workers run in sequence:

1. **planner** — reads your clipboard content and repo evidence, produces a `WorkPlan` that specifies what each subsequent worker will do. This is a hard blocker: if it fails, the run stops.

2. **patcher** — reads the source material and work plan, drafts the implementation intent: touched files, implementation notes, constraints from your current branch, and a diff skeleton. Hard blocker.

3. **critic** — inspects the patcher's output and identifies the top risks: schema changes, auth regressions, API contract breaks, concurrency hazards, destructive operations, and test gaps. Soft blocker: if it fails, the run continues with a warning.

4. **tester** — inspects the patcher's output and drafts a test plan: targeted test cases, edge cases, and verification steps. It checks whether a `_test.go` counterpart already exists for the primary touched file. Soft blocker.

5. **reconciler** — merges the patch draft, risk report, and test plan into a typed `PatchBundle`, notes any warnings from soft-blocker failures, and writes the final artifact. Hard blocker.

The critic and tester run concurrently after the patcher completes.

**Example stderr output during a deep run:**

```
-> take:patch --deep | claude | clipboard | running
   step: plan (1/5)
   step: patch (2/5)
   step: critique (3/5)
   step: tests (4/5)
   step: reconcile (5/5)
   prompt: enhanced for claude
```

#### Artifacts created

Every deep run writes its artifacts to `.prtr/runs/<run-id>/` inside your repo root (or to the prtr history directory if you are not in a repo).

| File | Description |
|---|---|
| `manifest.json` | Canonical run record: ID, status, action, timestamps |
| `plan.json` | The work plan produced by the planner worker |
| `events.jsonl` | Append-only structured event log for the full run |
| `lineage.json` | Provenance: parent history ID, source kind, target app |
| `evidence/source.md` | The clipboard text that started this run |
| `evidence/repo_context.json` | Repo name, branch, changed files |
| `evidence/history.json` | The parent history entry if one existed |
| `evidence/memory.json` | Protected terms from your termbook |
| `evidence/git.diff` | Output of `git diff` at run time |
| `result/patch_bundle.json` | Final structured output: summary, diff, risks, test plan |
| `result/patch.diff` | The diff skeleton extracted or inferred from source |
| `result/tests.md` | The test plan in Markdown |
| `result/summary.md` | Human-readable summary of the patch bundle |
| `workers/<name>/request.json` | Input sent to each worker |
| `workers/<name>/result.json` | Output produced by each worker |

Review after a run:

```bash
cat .prtr/runs/<run-id>/result/summary.md
cat .prtr/runs/<run-id>/result/tests.md
```

#### --llm flag: provider-specific formatting

By default, the delivery prompt uses a universal Markdown format. The `--llm` flag tells prtr to format the prompt for a specific AI provider's preferred structure:

```bash
prtr take patch --deep --llm=claude
prtr take patch --deep --llm=gemini
prtr take patch --deep --llm=codex
```

| Provider | Format | When to use |
|---|---|---|
| `claude` | XML semantic tags: `<role>`, `<context>`, `<task>`, `<risks>`, `<constraints>`, `<validation>` | Claude performs better with structured XML delimiters for multi-section prompts |
| `gemini` | Markdown headers: `## Role`, `## Source Intent`, `## Patch Bundle`, `## Task`, `## Risks`, `## Validation` | Gemini performs well with clear Markdown section breaks |
| `codex` | Numbered instruction list + fenced diff block | Codex is optimized for instruction-following with explicit step numbering and diff context |
| (none) | Universal Markdown with `## Goal`, `## Source Intent`, `## Key Risks`, `## Tests Required` | Works across all providers; use when you have not yet identified which provider handles the prompt |

#### Provider configuration

Set a default provider so you do not need to pass `--llm` on every run.

In `~/.config/prtr/config.toml`:

```toml
llm_provider = "claude"
```

Or use the environment variable:

```bash
export PRTR_LLM_PROVIDER=claude
```

Priority order (highest to lowest): `--llm` flag on the command line, `llm_provider` in `config.toml`, `PRTR_LLM_PROVIDER` environment variable.

Once set, `--deep` picks up the format automatically.

#### Deep actions

`--deep` supports four actions:

```bash
prtr take patch --deep     # implementation follow-up
prtr take test --deep      # test-writing follow-up
prtr take debug --deep     # root cause and fix follow-up
prtr take refactor --deep  # refactor scope and safety follow-up
```

Each action changes the planner's objective and the result type label in the manifest (`PatchBundle`, `TestBundle`, `DebugBundle`, `RefactorBundle`). The same five workers run regardless of action.

#### Complete examples

```bash
# Basic deep patch run using the configured default app
prtr take patch --deep

# Deep debug run formatted for Claude
prtr take debug --deep --llm=claude

# Deep test run targeting Codex, preview only
prtr take test --deep --to codex --dry-run

# Deep refactor run formatted for Gemini, open editor before delivery
prtr take refactor --deep --llm=gemini --edit
```

With `--dry-run`, the pipeline still runs all five workers and writes all artifacts to disk. The only difference is that the delivery prompt is not copied to the clipboard and no app is opened.

---

### prtr again

`again` replays the last run exactly. Use it when you closed the AI app by accident, want to resend to the same app, or want to tweak the prompt before sending again.

```bash
prtr again
prtr again --edit
prtr again --dry-run
```

`--edit` opens the compiled prompt in your editor before delivery. `--dry-run` prints the prompt without sending.

---

### prtr learn

`learn` scans your repo and builds a termbook: a list of project-specific identifiers that should not be translated or renamed by prtr's translation pipeline.

```bash
prtr learn
prtr learn README.md docs/
prtr learn --dry-run
prtr learn --reset
```

`learn` stores:
- source files used to build the termbook
- protected terms such as function names, CLI flags, package names, and identifiers in snake_case or camelCase

The termbook is written to `.prtr/termbook.toml` at your repo root. On every subsequent `prtr go`, these terms are attached to the prompt to prevent the translator from rewriting them.

`--dry-run` prints the termbook without writing it. `--reset` rebuilds from scratch, discarding any previous entries. Without `--reset`, new terms are merged with the existing termbook.

---

### prtr inspect

`inspect` compiles the prompt exactly as `go` would, but stops before clipboard copy and app launch. Use it to understand how your settings interact before sending.

```bash
prtr inspect "이 PR 리뷰해줘"
prtr inspect --json "이 에러 분석해줘"
prtr inspect --explain "이 설정이 어떻게 해석되는지 보여줘"
prtr inspect --diff "이 문장이 번역 후 어떻게 바뀌는지 보여줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

`inspect` outputs:
- the compiled final prompt
- with `--explain`: a breakdown of which settings were resolved and why
- with `--diff`: a before/after view of the translation
- with `--json`: the full resolved run as JSON

Nothing is copied, launched, or pasted.

---

## Configuration

### Config file location

prtr looks for a user config file at:

```
~/.config/prtr/config.toml
```

If `XDG_CONFIG_HOME` is set, it uses `$XDG_CONFIG_HOME/prtr/config.toml` instead.

Project-level overrides can be placed in `.prtr.toml` at the repo root.

Create the default config:

```bash
prtr init
```

Or run the guided setup:

```bash
prtr setup
```

### Key configuration fields

```toml
deepl_api_key = ""                           # DeepL API key (optional)
translation_source_lang = "auto"             # Input language: auto, ko, ja, zh, en
translation_target_lang = "en"               # Output language for prompts
default_target = "claude"                    # Default AI app: claude, gemini, codex
llm_provider = "claude"                      # Deep mode prompt format (new in v0.7)
```

`llm_provider` accepts `"claude"`, `"gemini"`, `"codex"`, or `""` (empty string means rule-based universal Markdown). This setting only affects `--deep` runs.

### Environment variables

| Variable | Description |
|---|---|
| `DEEPL_API_KEY` | DeepL API key; takes precedence over the config file value |
| `PRTR_LLM_PROVIDER` | Default LLM provider for deep mode; lowest priority (overridden by config and `--llm`) |
| `XDG_CONFIG_HOME` | Base directory for config; if set, prtr reads from `$XDG_CONFIG_HOME/prtr/config.toml` |

---

## Common scenarios

### Fix a failing test

```bash
# Run tests and pipe the failure output as evidence
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"

# Copy the answer from your AI app, then run the structured follow-up
prtr take patch --deep --llm=claude

# Review what the pipeline produced
cat .prtr/runs/$(ls -t .prtr/runs | head -1)/result/summary.md
```

### Review a pull request

```bash
# Send the diff to Claude for review
git diff main...HEAD | prtr go review "이 PR에서 위험한 부분만 짚어줘" --to claude

# Try the same review with Gemini for a second opinion
prtr swap gemini

# After copying the review answer, turn it into a GitHub issue
prtr take issue
```

### Write tests for new code

```bash
# Send your new code to get a test plan
cat internal/mypackage/newfile.go | prtr go "이 코드에 테스트가 필요한 부분 찾아줘"

# Copy the answer and generate a structured test-writing prompt
prtr take test --deep --llm=codex

# Review the test plan before it reaches the AI app
prtr take test --deep --dry-run
```

---

## Troubleshooting

### "clipboard is empty; copy an answer and try again"

`prtr take` reads from the system clipboard. Copy the AI response before running `take`.

### "missing prompt text"

`prtr go` requires either a prompt argument or piped stdin. Both `prtr go` and `prtr go --dry-run` without any input will return this error.

### Clipboard write or read fails on Linux

Install one of: `wl-copy` / `wl-paste` (Wayland), `xclip`, or `xsel` (X11). Then re-run `prtr doctor` to confirm the clipboard check passes.

### App does not open or paste does not work

Run `prtr doctor` to see which delivery components are available. On macOS, confirm Terminal has Accessibility permission in System Settings. On Linux, confirm a graphical session is running and `xdotool` (X11) or `wtype` (Wayland) is installed.

### Translation is rewriting identifiers I want to keep

Run `prtr learn` inside your repo to build a termbook. Protected terms are automatically attached to every subsequent `prtr go` call. Use `prtr inspect --diff` to see the before/after translation for any specific prompt.

### Deep run fails with "deep supports: patch, test, debug, refactor"

The action you passed is not supported by `--deep`. Classic-only actions such as `commit`, `summary`, `clarify`, `issue`, and `plan` do not use the five-worker pipeline. Use them without `--deep`.

### Artifacts are missing after a deep run

If you are not inside a Git repo, artifacts are written to the prtr history directory rather than `.prtr/runs/`. Run `prtr doctor` to see where prtr is storing data, or check `~/.local/share/prtr/runs/` (Linux) or `~/Library/Application Support/prtr/runs/` (macOS).
