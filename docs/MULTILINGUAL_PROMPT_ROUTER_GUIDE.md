# Multilingual Prompt Router Guide

`prtr` is easiest to understand if you think of it as a routing pipeline rather than a plain translator.
It takes a user request, decides how that request should be translated, wraps it in a target-aware prompt shape, optionally opens the target CLI, and stores the whole run in local history so you can reuse it later.

This guide is focused on the currently implemented multilingual prompt router flow.
For install steps, see [INSTALLATION.md](../INSTALLATION.md).
For the full command reference, see [USAGE.md](../USAGE.md).

## 1. What The Router Actually Does

For a command such as:

```bash
prtr -t codex -r be "도커 컨테이너 실행하는 법을 초보자용 단계로 정리해줘"
```

`prtr` performs this sequence:

1. Reads the prompt from CLI args or `stdin`.
2. Resolves the target, role, and template preset.
3. Resolves the source and target language route.
4. Decides whether translation should run with `auto`, `force`, or `skip`.
5. Preserves code-like tokens such as code blocks, paths, URLs, env vars, and stack traces.
6. Translates the user request with DeepL when required.
7. Renders the translated request into the selected template preset.
8. Prints the final prompt to `stdout`.
9. Copies the final prompt to the clipboard unless `--no-copy` is set.
10. Optionally launches or pastes into a supported target CLI.
11. Stores the run in local history for rerun, pin, favorite, and search.

## 2. First-Time Setup

Check the binary:

```bash
prtr version
```

Run guided setup:

```bash
prtr setup
```

`setup` can store:

- your DeepL API key
- default input language
- default output language
- default target
- default role
- default template preset

Validate the environment after setup:

```bash
prtr doctor
```

Use `doctor` before recording demos or troubleshooting launch and paste behavior.

## 3. The Fastest Path To A Working Run

Start with a safe terminal-only run:

```bash
prtr --no-copy "이 변경의 핵심 리스크를 요약해줘"
```

If you want to inspect how the router resolved the run:

```bash
prtr --no-copy --explain "이 변경의 핵심 리스크를 요약해줘"
```

Typical explain output includes:

- which target was selected
- which role was selected
- which template preset was chosen
- which language route was applied
- whether translation ran or was skipped
- whether delivery stayed as copy-only, launch, or paste

If you want machine-readable output:

```bash
prtr --no-copy --json "이 변경의 핵심 리스크를 요약해줘"
```

The JSON payload includes:

- `original`
- `translated`
- `final_prompt`
- `target`
- `role`
- `template_preset`
- `source_lang`
- `target_lang`
- `translation_mode`
- `translation_decision`
- `delivery_mode`

## 4. Core Prompt Routing Patterns

### Route a non-English request into an English coding prompt

```bash
prtr --no-copy -t codex -r be "이 함수의 성능 병목을 찾고 안전하게 리팩터링해줘"
```

Use this when you want:

- Korean, Japanese, or Chinese input
- English output prompt for coding models
- target-specific role wording for Codex

### Force translation even if the input looks mostly English

```bash
prtr --no-copy --translation-mode force --to en "Fix login bug. 사용자 영향도 같이 설명해줘."
```

Use this when a mixed-language prompt should still become a fully translated target prompt.

### Skip translation and only use prompt shaping

```bash
prtr --no-copy --translation-mode skip -t claude "Keep this prompt exactly as written"
```

Use this when you already wrote the request in the final language and only want template and role routing.

### Read from stdin

```bash
printf '이 쿼리 성능을 분석하고 인덱스 전략을 제안해줘' | prtr --no-copy -t claude
```

Use this in shell pipelines or editor integrations.

## 5. Target, Role, And Template Resolution

These three settings define the final prompt shape more than the translation itself.

### Target

Choose the receiving model family:

```bash
prtr -t claude "이 요구사항을 구조적으로 분석해줘"
prtr -t codex "이 버그를 고쳐줘"
prtr -t gemini "이 설계를 단계별로 비교해줘"
```

Built-in targets currently include:

- `claude`
- `codex`
- `gemini`

### Role

Choose the working persona:

```bash
prtr -r be "이 API 설계를 검토해줘"
prtr -r review "이 변경에서 회귀 리스크를 찾아줘"
prtr -r writer "이 설명을 더 간결하게 다시 써줘"
```

Role prompts can also change by target.
For example, `be` with `codex` uses a more implementation-heavy instruction than `be` with `claude`.

### Template preset

Choose the prompt layout explicitly:

```bash
prtr --template claude-structured "이 기능을 설명해줘"
prtr --template claude-review "이 PR을 리뷰해줘"
prtr --template codex-implement "이 테스트를 고쳐줘"
prtr --template codex-review "이 변경을 코드 리뷰해줘"
prtr --template gemini-stepwise "이 데이터 흐름을 단계별로 설명해줘"
```

Inspect available presets:

```bash
prtr templates list
prtr templates show codex-implement
```

## 6. Resolution Order That Matters In Practice

Understanding the precedence rules helps explain why a run resolved a specific way.

Target resolution order:

1. CLI `-t/--target`
2. built-in or project shortcut
3. config `default_target`
4. `PRTR_TARGET`
5. built-in default `claude`

Role resolution order:

1. CLI `-r/--role`
2. built-in or project shortcut
3. config `default_role`

Template preset resolution order:

1. CLI `--template`
2. built-in or project shortcut
3. role target override
4. config `default_template_preset`
5. target default template preset

Language route resolution order:

1. CLI `--source-lang`
2. config `translation_source_lang`
3. built-in `auto`

and:

1. CLI `--to`
2. shortcut `translation_target_lang`
3. target `translation_target_lang`
4. config `translation_target_lang`
5. built-in `en`

## 7. Built-In Shortcuts You Can Use Immediately

These are the fastest way to demonstrate the router:

```bash
prtr ask "이 변경을 요약해줘"
prtr review "이 변경의 리스크를 찾아줘"
prtr fix "이 테스트를 고쳐줘"
prtr design "이 온보딩 흐름을 개선해줘"
```

Current built-in shortcut intent:

- `ask`: general-purpose structured prompt for Claude
- `review`: review-oriented prompt for Claude with backend role defaults
- `fix`: implementation-oriented prompt for Codex
- `design`: stepwise product and design prompt for Gemini

## 8. Local Project Routing With `.prtr.toml`

Put a `.prtr.toml` file in your repository when the team wants shared defaults and reusable flows.

Example:

```toml
[shortcuts.review]
target = "claude"
role = "review"
template_preset = "claude-review"
translation_target_lang = "en"
context = "Assume this repository is production-facing."
output_format = "Start with findings, then list missing tests."

[profiles.backend_fix]
target = "codex"
role = "be"
template_preset = "codex-implement"
translation_target_lang = "en"
```

Useful patterns for project-local routing:

- repo-wide review shortcut
- incident response profile
- UI copy rewrite shortcut
- codex implementation profile
- release-note writing shortcut

## 9. Inspecting And Debugging The Router

Use `--explain` when the result does not look like what you expected:

```bash
prtr --no-copy --explain review "이 변경을 검토해줘"
```

Use `--diff` when you want to compare the original input, translated request, and final rendered prompt:

```bash
prtr --no-copy --diff "이 문장이 라우팅 후 어떻게 바뀌는지 보여줘"
```

Use `--show-original` if you want the raw input echoed to `stderr` before the final prompt.

Use `--json` if another tool needs to consume the router output.

## 10. Delivery Modes

The router has three practical delivery shapes:

Terminal-only:

```bash
prtr --no-copy "이 변경을 설명해줘"
```

Copy to clipboard:

```bash
prtr "이 변경을 설명해줘"
```

Launch or paste into a supported target:

```bash
prtr --launch "이 변경을 설명해줘"
prtr --paste "이 변경을 설명해줘"
prtr --paste --submit confirm "이 변경을 보내기 전에 마지막으로 검토해줘"
```

Current implementation notes:

- `--paste` implies `--launch`
- `--submit` requires `--paste`
- `--submit auto` is parsed but intentionally unsupported
- `--launch` supports `claude`, `codex`, and `gemini`
- `--paste` supports:
  - macOS: `Terminal.app`
  - Linux: graphical sessions with `xdotool` on X11 or `wtype` on Wayland
  - Windows: interactive desktop sessions via PowerShell SendKeys
- `--submit confirm` remains macOS-only

## 11. History As Local Prompt Memory

Every successful run is stored locally.

List recent runs:

```bash
prtr history
```

Search:

```bash
prtr history search review
```

Rerun:

```bash
prtr rerun <history-id>
```

Rerun and edit the stored final prompt:

```bash
prtr rerun <history-id> --edit
```

Pin or favorite:

```bash
prtr pin <history-id>
prtr favorite <history-id>
```

Stored history metadata includes:

- original input
- translated input
- final rendered prompt
- target
- role
- template preset
- shortcut
- source and target language
- translation mode and decision
- delivery mode
- paste and submit state

## 12. Demo-Friendly Commands

These are reliable commands to use in docs, live demos, or terminal recordings.

Explain routing without touching the clipboard:

```bash
prtr --no-copy --explain -t codex -r be --template codex-implement "이 로그인 버그를 고쳐줘"
```

Render a shortcut-based prompt:

```bash
prtr ask --no-copy "이 변경의 핵심을 요약해줘"
```

Show available presets:

```bash
prtr templates list
```

Show reusable profiles:

```bash
prtr profiles list
```

Inspect history after a run:

```bash
prtr history
```

## 13. Troubleshooting

`DEEPL_API_KEY is not set`

- run `prtr setup`
- or export `DEEPL_API_KEY`
- then rerun `prtr doctor`

Translation skipped unexpectedly:

- inspect with `--explain`
- rerun with `--translation-mode force`

Clipboard or launcher issues:

- rerun `prtr doctor`
- verify the target CLI is installed and on `PATH`
- on macOS, grant Automation and Accessibility permissions when using `--paste`

No project-specific behavior applied:

- confirm the current directory is inside a repository that contains `.prtr.toml`
- use `--explain` to verify whether project config was loaded

## 14. Recommended Onboarding Path For New Users

If someone is new to `prtr`, this is the shortest useful sequence:

1. `prtr setup`
2. `prtr doctor`
3. `prtr --no-copy --explain "이 변경의 리스크를 요약해줘"`
4. `prtr review "이 변경의 리스크를 요약해줘"`
5. `prtr history`

That path teaches setup, routing, explainability, shortcuts, and reuse in under five minutes.
