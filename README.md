# poly-prompt

`poly-prompt` is the repository for `prtr`, a macOS-first CLI that translates prompt text into English with DeepL, prints the final prompt to `stdout`, and copies it to the clipboard so you can paste it into Claude, Codex, Gemini, or any other tool with `Cmd+V`.

## What it does

- Accepts prompt text as CLI args or from `stdin`
- Translates the input to English with DeepL
- Applies a target profile template selected by `-t/--target`
- Prints the final prompt to `stdout`
- Copies the final prompt to the macOS clipboard with `pbcopy`

`stdout` is intentionally kept clean so the command can be chained in shell workflows. Status messages and optional original text are written to `stderr`.

## Install

### Homebrew

Once releases are published and the tap is live:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

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
template = "{{prompt}}"

[targets.codex]
template = "{{prompt}}"

[targets.gemini]
template = "{{prompt}}"
```

## Usage

```bash
prtr "도커 컨테이너 실행하는 법 알려줘"
prtr -t codex "리액트 컴포넌트 생명주기 설명해줘"
echo "이 코드를 리뷰해줘" | prtr -t gemini
prtr --no-copy --show-original "이 에러 원인 분석해줘"
```

Flag behavior:

- `-t, --target <name>` chooses the target profile
- `--no-copy` skips writing to the clipboard
- `--show-original` prints the original input to `stderr`

Target selection order:

1. CLI flag
2. Config `default_target`
3. `PRTR_TARGET`
4. `claude`

## Custom target templates

Every target template must include `{{prompt}}`.

Example:

```toml
[targets.codex]
template = "Please answer in English and keep the response concise.\n\n{{prompt}}"
```

## Limitations

- v1 is macOS-first and uses `pbcopy`
- v1 does not launch Claude, Codex, or Gemini automatically
- v1 does not simulate keypresses or auto-paste into another app
- v1 only supports DeepL as the translation backend

## Release flow

Releases are intended to be cut from Git tags. GitHub Actions runs GoReleaser, publishes macOS binaries, and updates the Homebrew tap repo.

Required GitHub secrets:

- `GITHUB_TOKEN` for GitHub Releases
- `HOMEBREW_TAP_GITHUB_TOKEN` with push access to `helloprtr/homebrew-tap`

## Development

Run tests locally:

```bash
go test ./...
```
