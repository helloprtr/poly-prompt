# Installation Guide

This guide covers how to install or update `prtr` for the latest published release.

Release page:

- [helloprtr/poly-prompt releases](https://github.com/helloprtr/poly-prompt/releases)

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
curl -LO https://github.com/helloprtr/poly-prompt/releases/download/v0.2.2/prtr_0.2.2_linux_amd64.tar.gz
tar -xzf prtr_0.2.2_linux_amd64.tar.gz
chmod +x prtr
sudo mv prtr /usr/local/bin/prtr
prtr version
```

### arm64

```bash
curl -LO https://github.com/helloprtr/poly-prompt/releases/download/v0.2.2/prtr_0.2.2_linux_arm64.tar.gz
tar -xzf prtr_0.2.2_linux_arm64.tar.gz
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

## Build from source

If you want the latest `main` branch instead of the latest published release:

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## Quick post-install smoke test

Run guided setup:

```bash
prtr setup
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
prtr --no-copy "이 코드 리뷰해줘"
```

Optional diagnostic check:

```bash
prtr doctor
```

Optional interactive check:

```bash
prtr -t claude -r be -i "이 API 설계 검토해줘"
```
