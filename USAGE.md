# prtr Usage Guide

`prtr` is the command layer for AI work.

Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.

This guide focuses on the current public surface:

- `prtr start`
- `prtr go`
- `prtr again`
- `prtr swap`
- `prtr take`
- `prtr learn`
- `prtr inspect`

Advanced template, role, profile, and history commands are still available, but they are secondary to the send loop.

The current working loop is:

- `start` for the first successful send
- `go` for the first send
- `swap` for model comparison
- `take` for next-action prompting
- `learn` for repo-specific memory
- `inspect` for expert visibility

## 1. First run

Check the installed binary:

```bash
prtr version
```

Try the setup-free path first:

```bash
prtr demo
prtr go "explain this error" --dry-run
```

Then use the beginner-first flow when you want a guided first send:

```bash
prtr start
```

`start` handles the first successful send:

- minimal onboarding when needed
- `doctor` before the first send
- a first request prompt if you do not pass one
- the same delivery flow as `go`

Then run guided setup when you want multilingual routing and advanced defaults:

```bash
prtr setup
```

`setup` still asks for:

- DeepL API key
- default input language
- default output language
- default target
- default role
- default template preset

If you only want to adjust language defaults later:

```bash
prtr lang
```

Run diagnostics directly at any time:

```bash
prtr doctor
```

`doctor` splits checks into:

- ready-now checks
- optional DeepL and delivery unlocks

`doctor` now also prints a platform matrix summary and supports:

```bash
prtr doctor --fix
```

`--fix` applies safe automatic fixes when possible, such as creating or resetting the user config, then prints fallback suggestions for anything it cannot repair automatically.

## 2. The fastest path after start: `prtr go`

Send a request in your own language:

```bash
prtr go "이 에러 원인 분석해줘"
```

Choose a mode:

```bash
prtr go review "이 PR에서 위험한 부분만 짚어줘"
prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘"
prtr go design "이 기능 구조 설계해줘"
```

Choose an app:

```bash
prtr go "이 구조 문제점 봐줘" --to claude
prtr go fix "왜 테스트가 깨지는지 봐줘" --to codex
prtr go review "이 설계 위험도 평가해줘" --to gemini
```

Edit before delivery:

```bash
prtr go design "이 기능 구조 설계해줘" --to gemini --edit
```

Preview only:

```bash
prtr go "이 문서 설명해줘" --dry-run
prtr go "explain this error" --dry-run
```

English requests already work without a DeepL key. For a canned preview, use `prtr demo`.

How input works:

- if you pass a message, that message is the request
- if you pipe text and also pass a message, the piped text becomes evidence
- if you only pipe text, the piped text becomes the request
- if you are inside a Git repo, `go` also adds lightweight repo context
- `--no-context` disables automatic repo context and piped-evidence attachment

Examples:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
cat crash.log | prtr go fix
```

Repo context currently includes:

- repo name
- current branch
- changed files summary

## 3. Repeat and compare

Run the latest flow again:

```bash
prtr again
prtr again --edit
prtr again --dry-run
```

Send the latest prompt to another app:

```bash
prtr swap claude
prtr swap codex
prtr swap gemini
prtr swap gemini --edit
prtr swap claude --dry-run
```

`swap` keeps the latest request and mode, then recompiles the prompt for the destination app instead of reusing the old app's template blindly.

## 4. Turn an answer into the next action

Use `take` when you already copied a useful answer and want to move straight into the next prompt.

```bash
prtr take patch
prtr take test --to codex
prtr take commit --dry-run
prtr take summary --edit
```

Supported actions:

- `patch`
- `test`
- `commit`
- `summary`

`take` always reads from the clipboard, generates a fresh English request for the chosen action, then sends or previews it with the same app-aware flow as `go`.

## 5. Teach repo terms with `learn`

Build a repo-local termbook of names that should not be translated away:

```bash
prtr learn
prtr learn README.md docs
prtr learn --dry-run
prtr learn --reset
```

What `learn` stores:

- source files used to build the termbook
- protected project terms such as `BuildPrompt`, `PRTR_TARGET`, `snake_case`, or `--dry-run`

Where it stores it:

- `.prtr/termbook.toml` at the repo root

Then `prtr go` automatically loads that termbook and protects those names during translation unless you use `--no-context`.

## 6. Inspect instead of send

Use `inspect` when you want to understand how the prompt was compiled without opening any app.

```bash
prtr inspect "이 PR 리뷰해줘"
prtr inspect --json "이 에러 분석해줘"
prtr inspect -t codex --template codex-implement -r be "이 함수 개선해줘"
```

Useful inspection flags:

```bash
prtr inspect --explain "이 설정이 어떻게 해석되는지 보여줘"
prtr inspect --diff "이 문장이 번역 후 어떻게 바뀌는지 보여줘"
prtr inspect --json "JSON으로 결과를 받고 싶어"
```

`inspect` is preview-only:

- no clipboard copy
- no launch
- no paste

## 7. Launch and paste support

Supported delivery targets:

- `claude`
- `codex`
- `gemini`

`--launch` support:

- macOS: `Terminal.app`
- Linux: first available backend from `x-terminal-emulator`, `gnome-terminal`, `konsole`, `kitty`, `wezterm`
- Windows: first available backend from `wt.exe`, `pwsh.exe`, `powershell.exe`, `cmd.exe`

`--paste` support:

- macOS: `Terminal.app`
- Linux: graphical sessions with `xdotool` on X11 or `wtype` on Wayland
- Windows: interactive desktop sessions via PowerShell SendKeys

`--submit` support:

- macOS: `--submit confirm`
- Linux and Windows: `--submit` is not supported yet
- `--submit auto` is parsed but intentionally rejected

Important rules:

- `--paste` implies `--launch`
- `--submit` requires `--paste`
- `--launch`, `--paste`, and `--submit` require clipboard copy

Examples:

```bash
prtr --launch "이 변경의 핵심 리스크를 요약해줘"
prtr --paste "이 변경의 핵심 리스크를 요약해줘"
prtr --paste --submit confirm "이 변경을 지금 보내도 되는지 검토해줘"
```

Current limitations:

- no iTerm2 support yet
- no GUI AI app automation
- no full auto submit

## 8. Advanced prompt controls

The older advanced surface is still supported.

Choose a target:

```bash
prtr -t claude "이 아키텍처를 분석해줘"
prtr -t codex "이 함수를 리팩터링해줘"
prtr -t gemini "이 데이터 흐름을 단계별로 정리해줘"
```

Choose a role:

```bash
prtr -r be "이 API 설계의 안정성을 검토해줘"
prtr -r review "이 변경의 회귀 리스크를 찾아줘"
prtr -r writer "이 설명을 더 간결하게 다시 써줘"
```

Choose a template explicitly:

```bash
prtr --template claude-review "이 PR 리뷰해줘"
prtr --template codex-implement "이 버그를 고쳐줘"
```

Keep the result in the terminal only:

```bash
prtr --no-copy "이 API 설계를 설명해줘"
```

Read from `stdin`:

```bash
echo "이 쿼리 성능을 분석해줘" | prtr
```

Override the output language for one run:

```bash
prtr --to ja "이 문장을 일본어 프롬프트로 바꿔줘"
```

Force or skip translation:

```bash
prtr --translation-mode force "짧은 명령도 번역해줘"
prtr --translation-mode skip "Keep this prompt exactly as written"
```

Open the interactive editor on the legacy path:

```bash
prtr -i "이 프롬프트를 조금 다듬고 싶어"
```

## 7. History as local prompt memory

List recent runs:

```bash
prtr history
```

Search past runs:

```bash
prtr history search review
```

Rerun a previous entry:

```bash
prtr rerun <history-id>
```

Rerun and edit the stored final prompt:

```bash
prtr rerun <history-id> --edit
```

Mark entries for easier reuse:

```bash
prtr pin <history-id>
prtr favorite <history-id>
```

Stored metadata includes:

- source and target language
- translation mode and decision
- launch target
- delivery mode
- pasted/submitted status

## 8. Profiles, templates, and project-local config

Inspect reusable templates:

```bash
prtr templates list
prtr templates show codex-implement
```

Inspect reusable profiles:

```bash
prtr profiles list
prtr profiles show backend_review
prtr profiles use backend_review
```

Project-local `.prtr.toml` example:

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

## 9. Troubleshooting

Missing DeepL key:

```bash
prtr doctor
prtr setup
```

Clipboard error:

- macOS: check `pbcopy`
- Linux: install `wl-copy`, `xclip`, or `xsel`
- Windows: confirm `clip.exe` is available

Launch or paste error:

- confirm the target CLI is installed and on `PATH`
- macOS: confirm Terminal is installed and Accessibility permission is granted
- Linux: confirm a graphical session is running and `xdotool` or `wtype` is installed
- Windows: confirm an interactive desktop session is active and `powershell.exe` or `pwsh.exe` is available
- rerun `prtr doctor`

Unexpected translation behavior:

- inspect with `prtr inspect --diff`
- use `--translation-mode force`
- or use `--translation-mode skip`
