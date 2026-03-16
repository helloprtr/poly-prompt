# Installation Guide

This guide covers how to install or update `prtr` for the latest published release.

Release page:

- [helloprtr/poly-prompt releases](https://github.com/helloprtr/poly-prompt/releases)

## Platform matrix

`prtr doctor` now summarizes the current surface using these labels:

- `macOS + Terminal.app`
- `Linux + X11`
- `Linux + Wayland`
- `Windows interactive session`

## macOS

### Install with Homebrew

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

### Update with Homebrew

```bash
brew update
brew upgrade prtr
prtr version
```

Expected result:

- `prtr version` prints the latest installed release version

### Apple Silicon vs Intel

Homebrew installs the correct package automatically for:

- Apple Silicon: `darwin_arm64`
- Intel: `darwin_amd64`

## Linux

Linux release archives are published on GitHub Releases.

### amd64

```bash
VERSION=<latest-version>
curl -LO "https://github.com/helloprtr/poly-prompt/releases/download/${VERSION}/prtr_${VERSION#v}_linux_amd64.tar.gz"
tar -xzf "prtr_${VERSION#v}_linux_amd64.tar.gz"
chmod +x prtr
sudo mv prtr /usr/local/bin/prtr
prtr version
```

### arm64

```bash
VERSION=<latest-version>
curl -LO "https://github.com/helloprtr/poly-prompt/releases/download/${VERSION}/prtr_${VERSION#v}_linux_arm64.tar.gz"
tar -xzf "prtr_${VERSION#v}_linux_arm64.tar.gz"
chmod +x prtr
sudo mv prtr /usr/local/bin/prtr
prtr version
```

### Update an existing install

Repeat the download and replace flow for your architecture, then verify:

```bash
prtr version
```

Expected result:

- `prtr version` prints the latest installed release version

### Clipboard dependencies

`prtr` auto-detects clipboard support on Linux in this order:

- `wl-copy`
- `xclip`
- `xsel`

Install at least one of them.

Examples:

```bash
sudo apt-get install wl-clipboard
```

```bash
sudo apt-get install xclip
```

```bash
sudo apt-get install xsel
```

### Launcher dependencies

`prtr --launch` on Linux requires at least one supported terminal backend. It checks in this order:

- `x-terminal-emulator`
- `gnome-terminal`
- `konsole`
- `kitty`
- `wezterm`

If none of these are installed, `prtr doctor` and `prtr --launch` return an actionable launcher error.

### Paste automation dependencies

`prtr --paste` on Linux requires a graphical session plus:

- X11: `xdotool`
- Wayland: `wtype`

Examples:

```bash
sudo apt-get install xdotool
```

```bash
sudo apt-get install wtype
```

## Windows

Windows release archives are published on GitHub Releases.

### amd64

1. Download the latest `prtr_<version>_windows_amd64.zip` from the releases page.
2. Unzip it.
3. Move `prtr.exe` into a folder on your `PATH`, or keep it in a tools directory and add that directory to `PATH`.
4. Open a new terminal and run:

```powershell
prtr version
```

### arm64

1. Download the latest `prtr_<version>_windows_arm64.zip` from the releases page.
2. Unzip it.
3. Move `prtr.exe` into a folder on your `PATH`, or keep it in a tools directory and add that directory to `PATH`.
4. Open a new terminal and run:

```powershell
prtr version
```

Expected result:

- `prtr version` prints the latest installed release version

### Clipboard support

Windows uses `clip.exe`, which is included with Windows.

### Launcher dependencies

`prtr --launch` on Windows checks terminal backends in this order:

- `wt.exe`
- `pwsh.exe`
- `powershell.exe`
- `cmd.exe`

If none of these are available, `prtr doctor` and `prtr --launch` return an actionable launcher error.

### Paste automation dependencies

`prtr --paste` on Windows requires:

- an interactive desktop session
- `powershell.exe` or `pwsh.exe`

`--submit confirm` is still macOS-only.

## Build from source

If you want the latest `main` branch instead of the latest published release:

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## Quick post-install smoke test

Run the beginner-first entry:

```bash
prtr start
```

`start` guides the first-run flow:

- prompts for minimal onboarding settings when needed
- runs `prtr doctor`
- asks for a first request if you do not pass one
- sends that first request through the same path as `prtr go`

If you want the full advanced defaults flow later:

```bash
prtr setup
```

Optional quick language-only update later:

```bash
prtr lang
```

Or set your API key manually:

```bash
export DEEPL_API_KEY="your-deepl-key"
```

Windows PowerShell:

```powershell
$env:DEEPL_API_KEY="your-deepl-key"
```

Then run:

```bash
prtr start "이 코드 리뷰해줘"
prtr go "이 에러 원인 분석해줘"
prtr go review "이 PR에서 위험한 부분만 짚어줘" --dry-run
```

Optional diagnostic check:

```bash
prtr doctor
prtr doctor --fix
```

Optional interactive check:

```bash
prtr go design "이 API 설계 검토해줘" --edit --dry-run
```

Optional history and launch checks:

```bash
prtr history
prtr go "이 변경의 핵심 리스크를 요약해줘"
```

Optional paste automation checks:

```bash
prtr --paste "이 변경의 핵심 리스크를 요약해줘"
prtr swap gemini --dry-run
```

Optional learn loop check from inside a Git repo:

```bash
prtr learn --dry-run
prtr go "BuildPrompt와 PRTR_TARGET를 설명해줘" --dry-run
```
