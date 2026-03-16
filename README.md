# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v0.6.2-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v0.6.2)

**prtr is the command layer for AI work.**

Turn logs, diffs, and intent into the next AI action — across Claude, Codex, and Gemini.

## 30-Second Loop

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch --deep
prtr learn
```

One intent. Another app. Structured artifacts. Repo memory.

## Install

#### Homebrew (macOS)

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

#### GitHub Releases (Linux / Windows)

Download the archive for your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases).

#### Build from source

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

## First Run

```bash
prtr demo                          # safe preview, no API key needed
prtr go "explain this error" --dry-run
```

Then unlock the full loop:

```bash
prtr setup     # store DeepL key, set default app
prtr doctor    # confirm what is ready, what is optional

prtr go fix "왜 테스트가 깨지는지 원인만 찾아줘"
prtr take patch --deep             # structured pipeline: risk + test plan + delivery prompt
```

## Core Commands

| Command | What it does |
|---|---|
| `go` | Translate intent, add context, open AI app, paste prompt |
| `swap` | Resend the last run to a different AI app |
| `take` | Turn AI output into the next structured action |
| `again` | Replay the last run; `--edit` to tweak before send |
| `learn` | Build repo memory and protected term list |
| `inspect` | Preview-only expert path: JSON, explain, diff |
| `take --deep` | Run internal pipeline: risk analysis, test plan, delivery prompt |

## `take --deep`

`--deep` runs a multi-step internal pipeline before delivery:

1. **Analyze** — reads diffs, logs, and repo context
2. **Risk** — identifies what could break
3. **Test plan** — generates verification steps
4. **Delivery prompt** — formats output for the target AI provider

```bash
prtr take patch --deep             # patch workflow
prtr take test --deep              # test-writing workflow
prtr take debug --deep             # debug workflow
prtr take refactor --deep          # refactor workflow
```

### Provider-specific formatting

Each AI app expects a different prompt shape. `--llm` formats the delivery prompt accordingly:

```bash
prtr take patch --deep --llm=claude   # XML tags (<task>, <context>, <constraints>)
prtr take patch --deep --llm=gemini   # Markdown sections
prtr take patch --deep --llm=codex    # Numbered instruction list
```

### Set a default provider

```toml
# ~/.config/prtr/config.toml
llm_provider = "claude"
```

Or use the environment variable:

```bash
export PRTR_LLM_PROVIDER=claude
```

Once set, `--deep` picks up the format automatically — no `--llm` flag needed on every run.

## Links

- Full install and update steps: [INSTALLATION.md](INSTALLATION.md)
- Day-to-day command reference: [docs/guide.md](docs/guide.md)
- 한국어 가이드: [docs/guide.ko.md](docs/guide.ko.md)
- Project site: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)
