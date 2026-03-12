# poly-prompt

`poly-prompt` is the repository for `prtr`, a cross-platform CLI that translates prompt text into English with DeepL, applies target-optimized prompt templates, optionally lets you review the final prompt in a TUI editor, prints the confirmed result to `stdout`, and copies it to your clipboard.

## What it does

- Accepts prompt text as CLI args or from `stdin`
- Translates the input to English with DeepL
- Applies a target profile template selected by `-t/--target`
- Optionally applies a role profile selected by `-r/--role`
- Optionally opens the final prompt in an interactive editor with `-i/--interactive`
- Prints only the confirmed final prompt to `stdout`
- Copies the final prompt to the clipboard on macOS, Linux, or Windows

`stdout` is intentionally kept clean so the command can be chained in shell workflows. Status messages and optional original text are written to `stderr`.

## Install

Detailed OS-specific install and update steps are in [INSTALLATION.md](/Users/koo/dev/translateCLI-brew/INSTALLATION.md).

### Homebrew (macOS)

Install from the official Homebrew tap:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

### GitHub Releases (Linux and Windows)

Download the archive that matches your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases).

- Linux: `prtr_<version>_linux_amd64.tar.gz` or `prtr_<version>_linux_arm64.tar.gz`
- Windows: `prtr_<version>_windows_amd64.zip` or `prtr_<version>_windows_arm64.zip`

### Build from source

If Go is not installed yet:

```bash
brew install go
```

Then build locally:

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## Install Verification

After installation, run through this quick smoke test checklist.

### 1. Check that the binary is installed

```bash
prtr version
```

Expected result:
- A version string such as `0.2.1`

### 2. Confirm the command is on your PATH

```bash
which prtr
```

Expected result:
- A Homebrew-installed path such as `/opt/homebrew/bin/prtr`

### 3. Set your DeepL API key

```bash
export DEEPL_API_KEY="your-deepl-key"
```

Optional default target:

```bash
export PRTR_TARGET="claude"
```

### 4. Create the starter config

```bash
prtr init
```

Expected result:
- A config file is created at `~/.config/prtr/config.toml`
- Running `prtr init` a second time refuses to overwrite the existing config

### 5. Run a no-clipboard smoke test

```bash
prtr --no-copy "도커 컨테이너 실행하는 법 알려줘"
```

Expected result:
- The translated English prompt is printed to `stdout`
- No clipboard write is attempted

### 6. Run a clipboard smoke test

```bash
prtr "리액트 useEffect 설명해줘"
```

Expected result:
- The translated English prompt is printed
- The same prompt is copied to the system clipboard
- You can paste it into Claude, Codex, Gemini, or any text field with `Cmd+V`

### 7. Test stdin input

```bash
echo "이 에러 원인 분석해줘" | prtr -t codex --no-copy
```

Expected result:
- `stdin` input is translated and printed correctly

## Configuration

### Required environment variable

```bash
export DEEPL_API_KEY="your-deepl-key"
```

### Optional environment variable

```bash
export PRTR_TARGET="claude"
```

### Create the starter config

```bash
prtr init
```

Config location:

- `$XDG_CONFIG_HOME/prtr/config.toml`
- Fallback: `~/.config/prtr/config.toml`

Starter config:

```toml
default_target = "claude"

[targets.claude]
template = """
<role>{{role}}</role>
<task>
Analyze and respond to the following prompt.
If the prompt involves code, provide a production-ready, clean, and optimized solution.
</task>

<input_prompt>
{{prompt}}
</input_prompt>

Please respond in English.
"""

[targets.codex]
template = """
// Language: Auto-detect
// Role: {{role}}
// Objective: Efficient, secure, and idiomatic code implementation.

{{prompt}}
"""

[targets.gemini]
template = """
You are an {{role}}

User Request:
{{prompt}}
"""

[roles.be]
content = "Expert Backend Engineer & Tech Lead"

[roles.fe]
content = "Expert Frontend Engineer & UX-minded Implementer"
```

## How It Works

`prtr` is a translator, prompt formatter, and clipboard helper for terminal workflows.

When you run a command such as:

```bash
prtr "이 React 에러 원인 분석해줘"
```

the tool performs these steps:

1. It reads the input text from command-line arguments or, if no arguments are present, from `stdin`.
2. It loads your config from `~/.config/prtr/config.toml` or `$XDG_CONFIG_HOME/prtr/config.toml`.
3. It resolves the target profile using this order:
   CLI flag `-t` -> config `default_target` -> `PRTR_TARGET` -> `claude`
4. It sends the original text to DeepL and asks for an `EN-US` translation.
5. It inserts the translated text into the selected target template using `{{prompt}}`, and if requested, inserts a role preset using `{{role}}`.
6. If `-i/--interactive` is set, it opens the rendered prompt in a single-pane TUI editor and waits for confirmation.
7. It prints only the confirmed final prompt to `stdout`.
8. Unless `--no-copy` is set, it copies the same final prompt to the platform clipboard.
9. It writes status messages, and optionally the original input, to `stderr`.

This separation is intentional:
- `stdout` stays clean for piping or scripting
- `stderr` carries status and debugging-friendly output

In practice, that means you can write in Korean, let `prtr` generate polished English, and then paste the result into Claude, Codex, Gemini, or any other prompt field.

## Usage

### Basic examples

```bash
prtr "도커 컨테이너 실행하는 법 알려줘"
prtr -t codex "리액트 컴포넌트 생명주기 설명해줘"
echo "이 코드를 리뷰해줘" | prtr -t gemini
prtr -r be "고 API 서버 설계 리뷰해줘"
prtr -t claude -r se -i "이 인증 플로우 보안 검토해줘"
prtr --no-copy --show-original "이 에러 원인 분석해줘"
```

### Input modes

`prtr` supports two input styles:

1. Positional text arguments

```bash
prtr "고랭으로 간단한 웹서버 예제 만들어줘"
```

2. Piped `stdin`

```bash
echo "이 쿼리 성능 문제 분석해줘" | prtr
```

If both are present, positional arguments win and `stdin` is ignored.

Flag behavior:

- `-t, --target <name>` chooses the target profile
- `-r, --role <alias>` chooses the role profile
- `-i, --interactive` opens a TUI editor before writing to `stdout`
- `--no-copy` skips writing to the clipboard
- `--show-original` prints the original input to `stderr`

### Target profiles

You can select a target profile with `-t`:

```bash
prtr -t claude "이 코드 리뷰해줘"
prtr -t codex "이 함수 리팩터링해줘"
prtr -t gemini "이 문서 요약해줘"
```

Built-in targets ship with optimized defaults for each model family:

- `claude`: XML-structured prompt with stronger instruction/data separation
- `gemini`: step-by-step reasoning scaffold with performance/scalability emphasis
- `codex`: code-first prompt with terse technical guidance

You can also choose an optional role profile with `-r`:

```bash
prtr -r be "고 API 서버 설계 리뷰해줘"
prtr -r da -t gemini "이 데이터 파이프라인 병목 분석해줘"
prtr -r ui -t claude "이 온보딩 화면 개선안 제안해줘"
```

Built-in roles:

- `da`: data engineering
- `be`: backend engineering
- `fe`: frontend engineering
- `ui`: product/UI design
- `se`: security engineering
- `pm`: product management

Target selection order:

1. CLI flag
2. Config `default_target`
3. `PRTR_TARGET`
4. `claude`

### Common workflows

Translate and copy to clipboard:

```bash
prtr "이 타입스크립트 에러 원인 알려줘"
```

Translate for a specific target profile:

```bash
prtr -t codex "이 함수 리팩터링하고 테스트도 제안해줘"
```

Use `stdin` from another command:

```bash
cat note.txt | prtr -t claude
```

Print only, without touching the clipboard:

```bash
prtr --no-copy "도커 네트워크 구조 설명해줘"
```

See the original input as well:

```bash
prtr --show-original "이 파이썬 코드 리뷰해줘"
```

Review and edit the final prompt before it is emitted:

```bash
prtr -i "이 코드 리뷰해줘"
echo "이 쿼리 성능 문제 분석해줘" | prtr -t gemini -r da -i
```

Interactive controls:

- `Ctrl+S`: confirm and emit the edited prompt
- `Ctrl+C`: cancel without writing anything to `stdout`
- plain text editing keys: edit the final prompt

## Custom target templates

Every target template must include `{{prompt}}`.

Example:

```toml
[targets.codex]
template = "Please answer in English and keep the response concise.\n\n{{prompt}}"
```

Role templates are opt-in and can be customized separately:

```toml
[roles.be]
content = "Expert Backend Engineer & Tech Lead"
```

When you use:

```bash
prtr -t codex "이 코드 문제 찾아줘"
```

the output becomes something like:

```text
Please answer in English and keep the response concise.

Find the problems in this code.
```

## Limitations

- v1 does not launch Claude, Codex, or Gemini automatically
- v1 does not simulate keypresses or auto-paste into another app
- v1 only supports DeepL as the translation backend

## Clipboard support

`prtr` detects clipboard backends automatically:

- macOS: `pbcopy`
- Linux: `wl-copy`, then `xclip`, then `xsel`
- Windows: `clip.exe`

If no supported clipboard tool is available, `prtr` returns an actionable error with installation guidance. `--no-copy` skips clipboard detection entirely.

## Release flow

Releases are intended to be cut from Git tags. GitHub Actions runs GoReleaser, publishes macOS binaries, and updates the Homebrew tap repo.

Required GitHub secrets:

- `GITHUB_TOKEN` for GitHub Releases
- `HOMEBREW_TAP_GITHUB_TOKEN` with push access to `helloprtr/homebrew-tap`

Release outputs:

- macOS binaries plus Homebrew formula update
- Linux tarballs in GitHub Releases
- Windows zip archives in GitHub Releases

For end-user install and upgrade instructions by OS, see [INSTALLATION.md](/Users/koo/dev/translateCLI-brew/INSTALLATION.md).

## Development

Run tests locally:

```bash
go test ./...
```
