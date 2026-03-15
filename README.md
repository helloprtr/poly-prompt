# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)

**Write in your language. Ship the next action.**

`prtr` is the beginner-first AI command layer that turns user intent into the next action for Claude, Codex, or Gemini.

Project site: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)

![prtr banner](images/prtr-banner.png)

## Why prtr

Most beginners do not get stuck because they cannot write a perfect prompt.

They get stuck because:

- they do not know which app to use
- they do not know whether to start with `review`, `fix`, `design`, or a plain `ask`
- they get one answer, then fail to turn it into the next action
- launch, paste, trust, and settings failures are hard to recover from

`prtr` is built to remove that friction.

It is not a prompt-polishing toy.
It is the command layer between what you mean and what you should do next.

## Status

### Available

- `start`
- `go`
- `swap`
- `take`
- `again`
- `learn`
- `inspect`
- `sync`
- `doctor --fix`
- `platform`
- `exec`
- repo-local termbook generation and memory support
- `open-copy` delivery

### Planned

- `fanout`
- `collect`

### Alpha

- `server`

### Current boundaries

- history is local-only
- full auto submit is not supported
- `--submit confirm` is still macOS-only

## 30-Second Example

```bash
prtr go review "Point out only the risky parts of this PR."
prtr swap codex
prtr take patch
prtr again --edit
```

You can also pipe evidence directly:

```bash
npm test 2>&1 | prtr go fix "Find the real reason this is failing."
```

## Core Concepts

**app**
Choose where to send the request.
`claude`, `gemini`, `codex`

**mode**
Choose what kind of help you want before you think about prompt wording.
`ask`, `review`, `fix`, `design`

**delivery**
Choose how the action gets executed.
`open-copy`, `exec`, `server`

**loop**
Do not stop at the first answer.
`go -> swap -> take -> again -> learn`

## Quick Start

Install details live in [INSTALLATION.md](INSTALLATION.md).
Daily command examples live in [USAGE.md](USAGE.md).
Product direction lives in [docs/PRD_V1.md](docs/PRD_V1.md) and [docs/BACKLOG_90_DAYS.md](docs/BACKLOG_90_DAYS.md).

### Install

#### Homebrew

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

#### GitHub Releases

Download the archive for your platform from the [releases page](https://github.com/helloprtr/poly-prompt/releases).

#### Build from source

```bash
git clone https://github.com/helloprtr/poly-prompt.git
cd poly-prompt
go build ./cmd/prtr
```

### First run

```bash
prtr start
```

If you want the older full configuration wizard, `prtr setup` still exists as the advanced compatibility path.

## Recommended Start Flow

### 1. Just send it

```bash
prtr go "Explain why this function is slow."
```

### 2. Pick a mode first

```bash
prtr go review "Point out only the risky parts of this PR."
prtr go fix "Find the real reason these tests are failing."
prtr go design "Design the structure for this feature."
```

### 3. Compare another app quickly

```bash
prtr swap claude
prtr swap gemini
prtr swap codex
```

### 4. Turn the first answer into the next action

```bash
prtr take patch
prtr take test
prtr take commit
prtr take summary
prtr take issue
prtr take plan
```

### 5. Keep project terms stable

```bash
prtr learn
prtr learn README.md docs
```

## The Working Surface

The beginner surface is intentionally small:

- `app`: `claude`, `gemini`, `codex`
- `mode`: `ask`, `review`, `fix`, `design`
- `delivery`: start with `open-copy`

The working loop is the real product surface:

- `go`: send the first request fast
- `swap`: compare another app with the same request
- `take`: turn an answer into the next action
- `again`: replay the recent flow
- `learn`: protect repo vocabulary for future runs
- `inspect`: open the expert path for raw details

The underlying config still uses names such as `target`, `template_preset`, `role`, and `profile`, but the public product language is now organized around `app`, `mode`, `delivery`, and the repeat loop.

## Delivery Modes

### `open-copy` — Available

Open the destination app, copy the rendered prompt, and hand it off through the visible terminal flow.
This is the default because it is the fastest activation path for beginners.

### `exec` — Available

Headless subprocess execution for background and script-friendly runs.
Use it when you want the same compiled prompt pipeline without visible open-copy handoff.

### `server` — Alpha

Long-running sessions and orchestration-friendly delivery for SDK, MCP, and app-server style integrations.

## Roadmap Surface

The next public surface is centered on:

- `prtr fanout`
- `prtr collect`
- deeper `server` orchestration

These commands are part of the roadmap. They are not implemented in the current public release.

## Design Principles

**Mode-first**
Ask what kind of help the user wants before asking which app to use.

**Next-action-first**
Optimize for what happens after the first answer, not just for the first answer itself.

**Human-first, script-ready**
The CLI should be easy for a person to use and easy for scripts, CI, or other agents to call later.

**Honest boundaries**
Do not pretend unsupported automation already exists.

## Current Product Boundaries

- `prtr go` is optimized for fast send loops, not full prompt inspection
- `prtr inspect` is the expert surface for target, role, template, diff, explain, JSON, and legacy flags
- Linux and Windows paste support still depend on platform-specific clipboard and terminal tooling
- DeepL is currently the translation backend
- history is capped locally instead of synced remotely

## Product Docs

- [INSTALLATION.md](INSTALLATION.md)
- [USAGE.md](USAGE.md)
- [docs/PRD_V1.md](docs/PRD_V1.md)
- [docs/BACKLOG_90_DAYS.md](docs/BACKLOG_90_DAYS.md)
- [docs/MULTILINGUAL_PROMPT_ROUTER_GUIDE.md](docs/MULTILINGUAL_PROMPT_ROUTER_GUIDE.md)
- [docs/MULTILINGUAL_PROMPT_ROUTER_DEMO_SCRIPT.md](docs/MULTILINGUAL_PROMPT_ROUTER_DEMO_SCRIPT.md)

## One Sentence

**prtr is the command layer that turns a beginner's intent into the next useful AI action as quickly as possible.**

## Development

```bash
go test ./...
```
