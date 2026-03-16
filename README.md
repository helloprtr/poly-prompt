# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v0.6.2-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v0.6.2)

**prtr is the command layer for AI work.**

Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.

Project site: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)

`prtr` is the command layer between what you mean and what you should do next.

It takes native-language intent, shapes it for Claude, Codex, or Gemini, and keeps the repeat loop close with `go`, `swap`, `take`, `again`, `learn`, and `inspect`.

![prtr banner](images/prtr-banner.png)

## 30-Second Loop

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch
prtr learn
```

One intent. Another app. The next action. Repo memory.

## Try It In 60 Seconds

Install first:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

Then try the setup-free path:

```bash
prtr demo
prtr go "explain this error" --dry-run
```

Set up DeepL later when you want multilingual routing:

```bash
prtr setup
prtr doctor
```

`prtr demo` is safe preview-only. It does not launch an app, paste into a terminal, or require a DeepL key.

## Promotion Kit

- X loop card: [images/x-card-loop-v062.png](images/x-card-loop-v062.png)
- X compare card: [images/x-card-compare-v062.png](images/x-card-compare-v062.png)
- Show HN thumbnail: [images/show-hn-thumb-v062.png](images/show-hn-thumb-v062.png)
- Promotion copy pack: [docs/PROMOTION_PACK_V062.md](docs/PROMOTION_PACK_V062.md)
- Release message templates: [docs/RELEASE_MESSAGES.md](docs/RELEASE_MESSAGES.md)
- Discussion seed prompts: [docs/DISCUSSION_SEEDS.md](docs/DISCUSSION_SEEDS.md)
- GitHub surface checklist: [docs/GITHUB_SURFACE_CHECKLIST.md](docs/GITHUB_SURFACE_CHECKLIST.md)
- Docs hub promo section: [helloprtr.github.io/poly-prompt/docs/#promo](https://helloprtr.github.io/poly-prompt/docs/#promo)

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
```

In one command, `prtr` can:

- treat your Korean text as the request
- treat the piped test output as evidence
- translate the request to English
- add lightweight repo context when available
- open your AI app
- paste the final prompt
- save the run so you can keep moving

Then keep going:

```bash
prtr swap gemini
prtr take patch
prtr learn
prtr again --edit
prtr inspect --json "방금 흐름이 어떻게 조합됐는지 보여줘"
```

From multilingual intent to routed prompt to next action.

## Why people keep using it

- No copy-paste prompt babysitting
- No rewriting Korean thoughts into awkward English
- No rebuilding the same context for every app
- No tab juggling between logs, repo, and AI tools

`prtr` is the command layer between what you mean and the AI workflow you want to repeat.

## The surface

The beginner surface is intentionally small:

- `app`: where to send it: `claude`, `codex`, `gemini`
- `mode`: what kind of help you want: `ask`, `review`, `fix`, `design`
- `recipe`: the reusable defaults behind the scenes

The config and advanced internals still use the existing names `target`, `template_preset`, `role`, and `profile`. The product surface is now organized around `app`, `mode`, and repeat loops.

The repeat loop is the real product surface:

- `go`: send the first prompt fast
- `swap`: compare another app without rebuilding context
- `take`: turn an answer into the next action
- `learn`: keep project terms stable across future runs
- `inspect`: open the expert path when you want the raw details

## Quick Start

### Install

Detailed OS-specific install and update steps are in [INSTALLATION.md](INSTALLATION.md).
Detailed day-to-day command examples are in [USAGE.md](USAGE.md).

#### Homebrew (macOS)

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

#### GitHub Releases (Linux and Windows)

Download the archive that matches your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases).

- Linux: `prtr_<version>_linux_amd64.tar.gz` or `prtr_<version>_linux_arm64.tar.gz`
- Windows: `prtr_<version>_windows_amd64.zip` or `prtr_<version>_windows_arm64.zip`

#### Build from source

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

### First run

```bash
prtr version
prtr demo
prtr go "explain this error" --dry-run
```

Then unlock multilingual routing and diagnostics when you are ready:

```bash
prtr setup
prtr doctor
```

`setup` stores your DeepL API key and default language/app settings. `doctor` now separates the checks you need right now from the optional unlocks that matter later.

## Start here

### Hero examples

```bash
prtr demo
prtr go "explain this error" --dry-run
prtr go "이 함수 왜 느린지 설명해줘"
prtr go review "이 PR에서 위험한 부분만 짚어줘"
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
```

### 1. Translate and copy

```bash
prtr "이 에러 원인 분석해줘"
```

This uses your defaults, renders the final prompt, prints it to `stdout`, and copies it to the clipboard.

### 2. Send it now

```bash
prtr go "이 에러 원인 분석해줘"
```

`go` is the fast path:

1. resolves the prompt text
2. picks the app from `--to`, your latest run, or your default app
3. picks the mode from the command or falls back to `ask`
4. translates with DeepL only when needed
5. auto-routes to the best app unless you override it with `--to`
6. copies the final prompt
7. opens the target CLI
8. pastes into the active terminal session
9. saves the run to local history

English requests already work without a DeepL key. If you want a safe preview before setup, start with `prtr demo`.

`go` prints a one-line status summary to `stderr`, for example:

```text
-> fix | codex | prompt+stdin | launch+paste | ko->en
```

### 3. Change the mode

```bash
prtr go review "이 PR 위험한 부분만 짚어줘"
prtr go fix "왜 테스트가 깨지는지 진짜 원인만 찾아줘"
prtr go design "이 기능 구조 설계해줘"
```

Available modes:

- `ask`
- `review`
- `fix`
- `design`

### 4. Change the app

```bash
prtr go "이 구조 문제점 봐줘" --to claude
prtr go fix "왜 테스트가 깨지는지 봐줘" --to codex
prtr go review "이 설계 위험도 평가해줘" --to gemini
```

### 5. Pipe evidence directly

If you provide both prompt text and piped `stdin`, `prtr go` treats `stdin` as evidence and appends it to the prompt.

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 진짜 원인만 찾아줘"
pytest -q 2>&1 | prtr go review "실패 원인과 리스크만 정리해줘"
```

If you pipe without prompt text, the piped content becomes the prompt itself.

If you run `go` inside a Git repo, `prtr` also adds lightweight repo context such as the repo name, current branch, and changed files. Use `--no-context` if you want to skip both repo context and piped evidence.

## The loop

The sticky part of `prtr` is not just sending once. It is making the next action cheaper.

### `prtr again`

Replay the latest run with the same mode and app.

```bash
prtr again
prtr again --edit
```

### `prtr swap`

Take the latest run and resend it to another app.

```bash
prtr swap claude
prtr swap gemini
prtr swap codex
```

When you swap apps, `prtr` keeps the core prompt but re-resolves the prompt shape for the destination app instead of blindly reusing the old app's template.

### `prtr take`

Turn copied AI output into the next prompt without manually rewriting it.

```bash
prtr take patch
prtr take clarify --from last-prompt
prtr take test --to codex
prtr take issue --dry-run
prtr take plan --edit
```

`take` now supports `patch`, `test`, `commit`, `summary`, `clarify`, `issue`, and `plan`.
It reads from `--from auto|clipboard|last-prompt`, builds a new English prompt for the selected action, then routes it through the same target-aware delivery flow.

### `prtr learn`

Teach `prtr` the project terms and repo memory that should survive translation.

```bash
prtr learn
prtr learn README.md docs
prtr learn --dry-run
prtr learn --reset
```

`learn` builds repo-local `.prtr/termbook.toml` and `.prtr/memory.toml` files from README, docs, and code identifiers. Then `go`, `take`, and `inspect` reuse protected terms, repo summary, and prompt guidance automatically.

## The only flags most people need

The first-screen send surface for `go`, `again`, `swap`, and `take` is intentionally small:

- `--to <app>`: choose `claude`, `codex`, or `gemini`
- `--edit`: review and edit the final prompt before delivery
- `--dry-run`: preview only; do not launch or paste
- `--no-context`: ignore piped `stdin` evidence when prompt text is already present

`learn` has its own minimal surface:

- `--dry-run`: preview the generated termbook
- `--reset`: rebuild the termbook and memory instead of merging them

## Pack sharing

Export, import, and install repo-local workflow packs:

```bash
prtr pack export backend-review --output ./backend-review.prtr-pack.toml
prtr pack import ./backend-review.prtr-pack.toml
prtr pack install backend-review --force
```

Packs capture:

- project defaults
- routing rules
- repo memory
- profiles
- shortcuts

## Inspect mode

Advanced output still exists, but it moved behind `inspect`.

```bash
prtr inspect "이 에러 원인 분석해줘"
prtr inspect --json "이 PR 리뷰해줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

`inspect` is preview-only:

- no clipboard copy
- no launch
- no paste
- explain and diff output by default unless you request `--json`

## Modes, recipes, and advanced config

Built-in mode defaults are backed by the existing shortcut system:

- `ask`
- `review`
- `fix`
- `design`

You can still inspect reusable profiles and advanced templates:

```bash
prtr templates list
prtr templates show codex-implement

prtr profiles list
prtr profiles show backend_review
prtr profiles use backend_review
```

Think of those advanced profiles as recipes. The user-facing language is simpler, but the engine underneath is still the same reliable config system.

### Config locations

- User config: `$XDG_CONFIG_HOME/prtr/config.toml`
- Fallback user config: `~/.config/prtr/config.toml`
- Project config: `.prtr.toml` found by walking up from the current working directory

### Starter config

```toml
deepl_api_key = ""
translation_source_lang = "auto"
translation_target_lang = "en"
default_target = "claude"
default_template_preset = "claude-structured"

[targets.claude]
family = "claude"
default_template_preset = "claude-structured"
translation_target_lang = "en"
default_delivery = "open-copy"

[targets.gemini]
family = "gemini"
default_template_preset = "gemini-stepwise"
translation_target_lang = "en"
default_delivery = "open-copy"

[targets.codex]
family = "codex"
default_template_preset = "codex-implement"
translation_target_lang = "en"
default_delivery = "open-copy"

[shortcuts.review]
target = "claude"
role = "be"
template_preset = "claude-review"
translation_target_lang = "en"
```

## Platform support

`prtr` supports launch on macOS, Linux, and Windows.

`prtr` supports paste on:

- macOS: `Terminal.app`
- Linux: graphical sessions with `xdotool` on X11 or `wtype` on Wayland
- Windows: interactive desktop sessions via PowerShell SendKeys

Clipboard support is detected automatically:

- macOS: `pbcopy`
- Linux: `wl-copy`, then `xclip`, then `xsel`
- Windows: `clip.exe`

## Current boundaries

- `prtr go` is optimized for fast send loops, not full prompt inspection
- `prtr inspect` is the expert path for JSON, explain, diff, and legacy flags
- `--submit confirm` is still macOS-only
- full auto submit is not supported yet
- DeepL is currently the translation backend
- history is local-only and capped to the most recent 200 entries

## Development

```bash
go test ./...
```
