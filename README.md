# poly-prompt

`poly-prompt` is the repository for `prtr`, a cross-platform CLI that translates prompt text into English with DeepL, applies model-specific template presets, layers in role guidance, lets you inspect or edit the final prompt, prints the confirmed result to `stdout`, and copies it to your clipboard.

![prtr banner](images/prtr-banner.png)

## What it does

- Accepts prompt text as CLI args or from `stdin`
- Guides first-run setup with `prtr setup`
- Runs local environment diagnostics with `prtr doctor`
- Translates the input to English with DeepL
- Applies a target selected by `-t/--target`
- Applies a template preset selected by `--template`
- Optionally applies a role selected by `-r/--role`
- Supports reusable profiles, shortcuts, history, and rerun
- Optionally opens the final prompt in an interactive editor with `-i/--interactive`
- Can explain, diff, or emit JSON for the resolved prompt pipeline
- Prints only the confirmed final prompt to `stdout`
- Copies the final prompt to the clipboard on macOS, Linux, or Windows

`stdout` stays clean for shell workflows. Status output, explain output, diffs, and optional original text are written to `stderr`.

## Install

Detailed OS-specific install and update steps are in [INSTALLATION.md](/Users/koo/dev/translateCLI-brew/INSTALLATION.md).

### Homebrew (macOS)

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

### GitHub Releases (Linux and Windows)

Download the archive that matches your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases).

- Linux: `prtr_<version>_linux_amd64.tar.gz` or `prtr_<version>_linux_arm64.tar.gz`
- Windows: `prtr_<version>_windows_amd64.zip` or `prtr_<version>_windows_arm64.zip`

### Build from source

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## Quick Start

### 1. Check the binary

```bash
prtr version
```

### 2. Run guided setup

```bash
prtr setup
```

`setup` walks through:
- DeepL API key storage
- default target
- default role
- default template preset

### 3. Run diagnostics

```bash
prtr doctor
```

`doctor` checks:
- user and project config detection
- DeepL API key availability
- clipboard backend support
- target, template preset, profile, and shortcut validity
- translation reachability

### 4. Generate your first prompt

```bash
prtr --no-copy "도커 컨테이너 실행하는 법 알려줘"
```

### 5. Try a shortcut

```bash
prtr review "이 PR의 리스크를 찾아줘"
```

## Configuration

### Config locations

- User config: `$XDG_CONFIG_HOME/prtr/config.toml`
- Fallback user config: `~/.config/prtr/config.toml`
- Project config: `.prtr.toml` found by walking up from the current working directory

### Schema notes

- `deepl_api_key` can be stored in user config through `prtr setup`
- `default_target`, `default_role`, and `default_template_preset` are supported
- template presets are defined separately from targets
- `prompt` is the preferred key for role definitions
- the older `content` key is still accepted for backward compatibility when loading roles

### Starter config

```toml
deepl_api_key = ""
default_target = "claude"
default_template_preset = "claude-structured"

[targets.claude]
family = "claude"
default_template_preset = "claude-structured"

[targets.gemini]
family = "gemini"
default_template_preset = "gemini-stepwise"

[targets.codex]
family = "codex"
default_template_preset = "codex-implement"

[template_presets.claude-structured]
description = "XML-structured default for Claude."
template = """
<role>
{{role}}
</role>

<context>
{{context}}
</context>

<input_prompt>
{{prompt}}
</input_prompt>

<output_format>
{{output_format}}
</output_format>
"""

[template_presets.gemini-stepwise]
description = "Stepwise reasoning scaffold for Gemini."
template = """
{{role}}

Context:
{{context}}

Follow these steps:
1. Briefly summarize the core requirement.
2. Identify edge cases, bottlenecks, or risks.
3. Provide the most useful response for the user's request.

User Request:
{{prompt}}

Output Format:
{{output_format}}
"""

[template_presets.codex-implement]
description = "Implementation-focused prompt for coding models."
template = """
// Target: {{target}}
// Context: {{context}}
// Output Format: {{output_format}}

{{role}}

{{prompt}}
"""

[roles.be]
prompt = """
Expert Backend Engineer & Tech Lead.
Focus on API design, reliability, observability, maintainability, security, and production tradeoffs.
Tailor the response to the user's request instead of assuming they want code.
"""

[profiles.backend_review]
target = "claude"
role = "be"
template_preset = "claude-review"

[shortcuts.review]
target = "claude"
role = "be"
template_preset = "claude-review"
```

## How it works

When you run a command such as:

```bash
prtr "이 React 에러 원인 분석해줘"
```

the tool performs these steps:

1. Reads input from CLI args or, if no args are present, from `stdin`
2. Loads built-in defaults, user config, and optional project config
3. Resolves the target using this order:
   CLI `-t` -> shortcut -> config `default_target` -> `PRTR_TARGET` -> built-in default
4. Resolves the template preset using this order:
   CLI `--template` -> shortcut -> config `default_template_preset` -> target default preset
5. Resolves the role using this order:
   CLI `-r` -> shortcut -> config `default_role`
6. Resolves the DeepL API key from environment or config
7. Translates the original text to `EN-US`
8. Renders the selected template preset with `{{prompt}}` and optional placeholders such as `{{role}}`, `{{target}}`, `{{context}}`, and `{{output_format}}`
9. Optionally opens the result in the interactive editor
10. Prints the final prompt to `stdout`, or emits structured JSON with `--json`
11. Copies the final prompt unless `--no-copy` is set
12. Stores the run in local history for `prtr history` and `prtr rerun`

## Usage

### Basic examples

```bash
prtr "도커 컨테이너 실행하는 법 알려줘"
prtr --template codex-implement "리액트 컴포넌트 생명주기 설명해줘"
echo "이 코드를 리뷰해줘" | prtr --template gemini-stepwise
prtr -r be --template claude-review "고 API 서버 설계 리뷰해줘"
prtr review -i "이 인증 플로우 보안 검토해줘"
prtr --json --explain --diff "이 에러 원인 분석해줘"
```

### Input modes

```bash
prtr "고랭으로 간단한 웹서버 예제 만들어줘"
echo "이 쿼리 성능 문제 분석해줘" | prtr
```

If both are present, positional arguments win and `stdin` is ignored.

### Core flags

- `-t, --target <name>` chooses the target
- `-r, --role <alias>` chooses the role
- `--template <name>` chooses the template preset
- `-i, --interactive` opens the TUI editor before output
- `--no-copy` skips clipboard copy
- `--show-original` prints the original input to `stderr`
- `--explain` prints resolved config details to `stderr`
- `--diff` prints original, translated, and final prompt sections to `stderr`
- `--json` emits structured JSON instead of plain text

## Targets, presets, and roles

Built-in targets define model family and default preset:

- `claude` -> `claude-structured`
- `gemini` -> `gemini-stepwise`
- `codex` -> `codex-implement`

You can select a target explicitly:

```bash
prtr -t claude "이 구조를 분석해줘"
prtr -t gemini "이 데이터 파이프라인 병목 분석해줘"
prtr -t codex "이 함수를 리팩터링해줘"
```

You can also select a preset directly:

```bash
prtr --template claude-structured "이 구조를 분석해줘"
prtr --template claude-review "이 PR 리뷰해줘"
prtr --template gemini-stepwise "이 파이프라인 병목 분석해줘"
prtr --template gemini-scalable "이 시스템 확장 전략 제안해줘"
prtr --template codex-implement "이 함수를 리팩터링해줘"
prtr --template codex-review "이 코드의 리스크를 찾아줘"
```

Built-in roles:

- `da`: data engineering
- `be`: backend engineering
- `fe`: frontend engineering
- `ui`: product and UI design
- `se`: security engineering
- `pm`: product management

The split is intentional:

- targets define model family and default strategy
- template presets define the actual optimized prompt shape
- roles define the expert lens and response priorities
- the translated user request stays in `{{prompt}}` without assuming they want code, design output, planning, or review unless the request asks for it

## Profiles, shortcuts, and history

List and inspect reusable profiles:

```bash
prtr profiles list
prtr profiles show backend_review
prtr profiles use backend_review
```

Use built-in shortcuts:

```bash
prtr ask "이 기능 설명해줘"
prtr review "이 설계 문제점 찾아줘"
prtr fix "이 에러 고쳐줘"
prtr design "이 온보딩 화면 개선해줘"
```

Inspect local history and rerun a past prompt:

```bash
prtr history
prtr rerun <history-id> -i
```

## Custom templates and project config

Every template preset must include `{{prompt}}`.

Custom user preset:

```toml
[template_presets.team-review]
template = """
<role>
{{role}}
</role>

<context>
Team coding standards and release checklist apply.
</context>

<input_prompt>
{{prompt}}
</input_prompt>
"""
```

Project-local override in `.prtr.toml`:

```toml
[shortcuts.review]
target = "claude"
role = "se"
template_preset = "claude-review"
```

## Clipboard support

`prtr` detects clipboard backends automatically:

- macOS: `pbcopy`
- Linux: `wl-copy`, then `xclip`, then `xsel`
- Windows: `clip.exe`

If no supported clipboard tool is available, `prtr` returns an actionable error with installation guidance. `--no-copy` skips clipboard detection entirely.

## Limitations

- `prtr` does not launch Claude, Codex, or Gemini automatically
- `prtr` does not simulate keypresses or auto-paste into another app
- `prtr` currently uses DeepL as the translation backend
- history is local-only and is capped to the most recent 200 entries

## Development

```bash
go test ./...
```
