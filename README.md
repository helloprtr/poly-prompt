# poly-prompt

[![Project Site](https://img.shields.io/badge/project%20site-live-ff7a1a?style=flat-square)](https://helloprtr.github.io/poly-prompt/)
[![Docs Hub](https://img.shields.io/badge/docs-pages-58f2c5?style=flat-square)](https://helloprtr.github.io/poly-prompt/docs/)
[![Latest Release](https://img.shields.io/badge/release-v1.0.1-1b2c49?style=flat-square)](https://github.com/helloprtr/poly-prompt/releases/tag/v1.0.1)

[English README](README.md) · [한국어 README](README.ko.md) · [Docs Hub](https://helloprtr.github.io/poly-prompt/docs/) · [Releases](https://github.com/helloprtr/poly-prompt/releases)

![prtr banner](images/prtr-banner.png)

**One line:** `prtr` is the AI Work Session Manager — start a focused work session, let Claude drive, checkpoint your progress, and hand off cleanly to Gemini or Codex.

`prtr` keeps track of what you are doing across the entire AI loop. Instead of rebuilding context by hand every time you switch tools or pick up where you left off, prtr holds your task goal, the files you care about, your progress notes, and the git diff — and builds the right prompt when you need it.

## Install in 60 Seconds

macOS with Homebrew:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
prtr doctor        # verify setup
```

Linux and Windows — download the right archive from [GitHub Releases](https://github.com/helloprtr/poly-prompt/releases), put `prtr` on your `PATH`.

## The Core Idea: Sessions

A **session** is a focused work block tied to a git repository. When you run any session command, prtr:

1. Prompts for your goal (or takes it from the command line)
2. Saves a session file with your task, files, mode, and base git SHA
3. Builds a structured start prompt and copies it to your clipboard
4. Opens Claude (or your configured AI app)

When you hand off or resume later, prtr recomputes the git diff since the session started and includes it in the prompt — so the AI picks up exactly where you left off.

## Session Commands (Start Here)

```bash
prtr                              # resume active session, or start a new one
prtr review [files...]            # start a code review session
prtr edit [files...]              # start a focused edit session
prtr fix [files...]               # start a bug-fix session
prtr design [topic]               # start a design/architecture session
```

**Manage your session:**

```bash
prtr checkpoint "refactor done"   # save a progress note (tied to current git SHA)
prtr @gemini                      # hand off current session to Gemini
prtr @codex                       # hand off current session to Codex
prtr done                         # mark session complete
prtr sessions                     # list sessions for this repo
prtr status                       # show current session + git diff summary
```

**A full session from start to finish:**

```bash
prtr edit internal/app/app.go     # start — goal prompted interactively
prtr checkpoint "split help.go"   # mid-session note
prtr @gemini                      # hand off with full diff context
prtr done                         # close session
```

## Key Concepts

### Session
A session tracks one focused task inside a git repo. It stores:
- **Goal** — what you are trying to accomplish
- **Files** — which files matter (optional)
- **Mode** — `review`, `edit`, `fix`, or `design`
- **Base git SHA** — the commit when the session started (used to compute the handoff diff)
- **Checkpoints** — timestamped progress notes you add along the way

Sessions are stored locally at `~/.config/prtr/sessions/`. One active session per repo at a time.

### Handoff
When you run `prtr @gemini` or `prtr @codex`, prtr:
1. Computes the git diff since the session base SHA
2. Reads your last AI response (auto-captured from the model's conversation log)
3. Builds a structured handoff prompt with full context
4. Copies it to your clipboard and opens the target app

No manual context reconstruction needed.

**Auto-capture (v1.0.1):** When you exit Claude Code or Codex, prtr automatically reads the model's conversation log and saves the last response to `~/.config/prtr/last-response.json`. The next handoff picks it up with no copy-paste required.

- Claude Code: reads `~/.claude/projects/<repo-slug>/*.jsonl`
- Codex: reads `~/.codex/sessions/YYYY/MM/DD/rollout-*.jsonl`

### Checkpoint
A progress note saved mid-session. Each checkpoint records the note text, the current git SHA, and a timestamp. Checkpoints are included in handoff prompts so the next AI knows exactly where you stopped.

### Work Capsule
Beyond sessions, `prtr save` captures a full snapshot of your current work state — branch, HEAD SHA, last run, and open todos — so you can resume days later or on a different machine, with drift detection built in.

```bash
prtr save "auth refactor in progress"   # snapshot current state
prtr list                               # list all capsules for this repo
prtr resume                             # resume latest capsule with drift report
prtr prune --older-than 30d             # clean up old capsules
```

## All Commands at a Glance

| Command | What it does |
|---|---|
| `prtr` | Resume active session or start new |
| `prtr review [files]` | Code review session |
| `prtr edit [files]` | Code edit session |
| `prtr fix [files]` | Bug fix session |
| `prtr design [topic]` | Design/architecture session |
| `prtr @gemini` / `@codex` | Hand off to another AI model |
| `prtr checkpoint "note"` | Save progress note |
| `prtr done` | Mark session complete |
| `prtr sessions` | List this repo's sessions |
| `prtr status` | Current session + diff summary |
| `prtr save [label]` | Snapshot work state (Work Capsule) |
| `prtr resume [id]` | Resume a capsule with drift detection |
| `prtr list` | List all capsules for this repo |
| `prtr doctor` | Check setup (AI binaries, clipboard, config) |
| `prtr setup` | Guided first-time configuration |

### Still Available (Advanced)

These classic commands remain fully functional:

| Command | What it does |
|---|---|
| `prtr go [mode] [message]` | Fast-path to send any request to Claude/Codex/Gemini |
| `prtr swap <app>` | Resend last prompt to another AI app |
| `prtr take <action>` | Convert AI output to patch/test/commit/plan prompt |
| `prtr take --deep` | Run 5-worker AI pipeline (planner, patcher, critic, tester, reconciler) |
| `prtr again` | Replay the last run |
| `prtr learn [paths]` | Build repo memory to protect project terms during translation |
| `prtr inspect [message]` | Show assembled prompt and config without sending |
| `prtr history` | Browse past runs |

## Multilingual Support

Write your goal in Korean (or any language). prtr translates via DeepL before sending. No API key is needed for English-only flows.

```bash
prtr edit                         # then type: "이 함수 성능 개선해줘"
prtr go "왜 테스트가 깨지는지 찾아줘"
```

Configure with:

```bash
prtr setup        # set DeepL key, default language, default AI app
prtr doctor       # verify everything works
```

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
- Help wanted: [help wanted](https://github.com/helloprtr/poly-prompt/issues?q=is%3Aopen+is%3Aissue+label%3A%22help+wanted%22)
- Ideas and workflow feedback: [Discussions](https://github.com/helloprtr/poly-prompt/discussions)
