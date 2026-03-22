# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v1.0.0-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v1.0.0)

[English README](README.md) · [한국어 README](README.ko.md) · [Docs Hub](https://helloprtr.github.io/poly-prompt/docs/) · [Releases](https://github.com/helloprtr/poly-prompt/releases)

![prtr banner](images/prtr-banner.png)

**One line:** `prtr` is the AI Work Session Manager: start a focused work session, let Claude drive, checkpoint progress, and hand off cleanly to Gemini or Codex.

`prtr` keeps track of what you are working on across the AI loop. Start a session, checkpoint your progress, and hand off to another model — without rebuilding context by hand.

## Try It in 60 Seconds

macOS with Homebrew:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
prtr review internal/app/app.go
```

Linux and Windows:

Download the right archive from [GitHub Releases](https://github.com/helloprtr/poly-prompt/releases), put `prtr` on your `PATH`, then run the same command above.

Start a real session:

```bash
prtr edit internal/session/store.go   # start a focused edit session
prtr checkpoint "store refactor done"  # save progress
prtr @gemini                           # hand off to Gemini
prtr done                              # mark complete
```

No API key is required for `--dry-run` flows.

## Session Commands

| Command | What it does |
|---|---|
| `prtr` | Resume active session or start a new one |
| `prtr review [files]` | Start a code review session |
| `prtr edit [files]` | Start a focused edit session |
| `prtr fix [files]` | Start a bug-fix session |
| `prtr design [topic]` | Start a design session |
| `prtr @gemini` / `@codex` | Hand off current session to another model |
| `prtr checkpoint "note"` | Save a progress memo |
| `prtr done` | Mark the session complete |
| `prtr sessions` | List sessions for the current repo |
| `prtr status` | Show current session state and git diff summary |

## Documentation

- Start here: [Docs Hub](https://helloprtr.github.io/poly-prompt/docs/)
- Install and update: [INSTALLATION.md](INSTALLATION.md)
- Day-to-day usage: [docs/guide.md](docs/guide.md)
- Command reference: [docs/reference.md](docs/reference.md)
- Korean guide: [docs/guide.ko.md](docs/guide.ko.md)
- Project site: [helloprtr.github.io/poly-prompt](https://helloprtr.github.io/poly-prompt/)

## Contributing

- Contribution guide: [CONTRIBUTING.md](CONTRIBUTING.md)
- Starter tasks: [good first issue](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22)
- Broader help: [help wanted](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
- Early ideas and workflow feedback: [Discussions](https://github.com/helloprtr/poly-prompt/discussions)

If you maintain this repo, keep 3 to 5 small, current `good first issue` tickets live. The best starter issues explain the user pain clearly, define done criteria, and include one local verification command.
