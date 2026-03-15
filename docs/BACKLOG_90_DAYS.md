# prtr 90-Day Backlog

*2026-03-15 to 2026-06-13*

This backlog is optimized for sequence, not feature count.

The order is:

`activation -> continuity -> automation`

## Priority Rules

### P0 first

- README
- `start`
- `doctor --fix`
- honest status labels

### P1 next

- `sync`
- JSON event log
- `exec`

### P2 last

- `server`
- `fanout` polish
- ecosystem and content scale

## Sprint 1: Positioning and truth

*Week 1 to Week 2*

### Goal

Make the external product story match the real command surface.

### Must ship

- README v2
- command vocabulary freeze for `app`, `mode`, `delivery`, `sync`, and `doctor`
- feature maturity labels: `Available`, `Planned`, `Alpha`
- release note template cleanup
- issue template and PR template cleanup
- first demo script draft

### Tasks

- rebuild the README around the next-action command layer story
- reorder the public surface around `go`, `swap`, `take`, `again`, `learn`, `inspect`
- mark `start`, `sync`, `doctor --fix`, `exec`, and `server` as roadmap features
- simplify the OSS contribution guide
- define three demo GIF scenarios:
  - first success
  - swap comparison
  - failure to recovery

### Definition of done

- the product is explainable in one sentence from the first screen
- install to first-use flow is visible in one pass
- non-shipped features do not look shipped

## Sprint 2: Activation

*Week 3 to Week 4*

### Goal

Build the first `start` and `doctor --fix` skeleton.

### Must ship

- `prtr start` alpha
- `prtr doctor --fix` core
- OS and platform probe layer
- first-run success telemetry

### Tasks

- implement `start` flow:
  - config presence check
  - setup guidance
  - doctor run
  - sample send
- implement `doctor` probes:
  - Claude, Gemini, Codex CLI presence
  - auth state
  - clipboard
  - launcher
  - paste backend
- implement first `doctor --fix` actions:
  - missing clipboard dependency guidance
  - unsupported paste fallback
  - dry-run fallback
- unify failure message formatting
- store first-success metric

### Definition of done

- at least four out of five new users can reach first success from docs only
- `doctor --fix` resolves at least three real issue types
- every failure shows one next step immediately

## Sprint 3: Sync foundation

*Week 5 to Week 6*

### Goal

Lock the canonical source and generated target model.

### Must ship

- `.prtr/` canonical layout
- `sync init`
- `sync status`
- `sync`
- diff-first preview

### Tasks

- define canonical files:
  - `.prtr/guide.md`
  - `.prtr/termbook.toml`
  - `.prtr/memory.toml`
- build target renderers:
  - `CLAUDE.md`
  - `GEMINI.md`
  - `AGENTS.md`
- design import paths
- add generated-file markers
- define conflict policy:
  - no blind overwrite
  - preview first
- add sync tests for:
  - clean repo
  - existing guide files
  - partial target writes

### Definition of done

- `sync init -> sync status -> sync` works in an empty repo
- existing files produce safe diffs
- drift is understandable in one line

## Sprint 4: Exec engine and logs

*Week 7 to Week 8*

### Goal

Ship the first automation engine outside `open-copy`.

### Must ship

- internal runner interface
- Claude exec adapter
- Gemini exec adapter
- Codex exec adapter
- `.jsonl` event log v0
- `collect` alpha

### Tasks

- define the run model:
  - run id
  - target
  - mode
  - delivery
  - timestamps
  - exit status
- define the event schema
- implement vendor adapters:
  - Claude headless subprocess
  - Gemini headless subprocess
  - Codex non-interactive subprocess
- add human-readable stdout formatter
- add `--json` formatter
- add `collect --latest`
- add `tail <run-id>`

### Definition of done

- all three targets can store real `exec` results
- each run leaves both human and machine logs
- `collect` returns the last result successfully

## Sprint 5: Trust, fanout, and repair depth

*Week 9 to Week 10*

### Goal

Add parallel execution, trust abstraction, and stronger repair logic.

### Must ship

- trust profiles: `safe`, `work`, `yolo`
- `fanout` alpha
- `doctor` trust and sandbox checks
- three failure to recovery examples

### Tasks

- implement vendor permission mapping table
- design trust profile selection UX
- implement `fanout --to claude,gemini,codex --delivery exec`
- add partial failure summary
- merge `collect` output across targets
- detect Gemini trusted-folder and project-settings issues
- detect Codex sandbox and approval mismatches
- detect Claude permission misconfigurations
- document three real failure and recovery stories

### Definition of done

- one prompt can run across two to three agents in parallel
- trust profiles are understandable without vendor jargon
- doctor catches at least two trust-related problem types

## Sprint 6: Server alpha and launch assets

*Week 11 to Week 12*

### Goal

Ship the first long-running session layer and public launch assets.

### Must ship

- `server` alpha
- four content packs
- three demo GIFs
- launch checklist
- KPI dashboard v0

### Tasks

- build Claude server alpha:
  - Agent SDK session wrapper
- build Codex server alpha:
  - MCP server or app-server wrapper
- build Gemini alpha:
  - persistent subprocess plus MCP bridge
- create content assets:
  - why beginners struggle with AI
  - why next action matters more than prompt polish
  - one layer across Claude, Gemini, and Codex
  - an AI CLI with `doctor --fix`
- create community assets:
  - examples repo cleanup
  - recipes page
  - release note style guide
- build launch metrics dashboard:
  - activation
  - usage
  - reliability

### Definition of done

- at least one vendor keeps a stable long-running session
- the README explains the product without needing a demo
- baseline launch KPIs are measurable before public launch

## Cut Order

If the schedule slips, cut in this order:

1. `server` alpha
2. `fanout` polish
3. deeper pack portability
4. advanced sync target expansion

Do not cut:

- `start`
- `doctor --fix`
- `sync`
- `exec`
- event log

## Launch Checklist

- README-first onboarding works
- `start` success rate meets target
- `doctor --fix` resolution rate meets target
- `exec` smoke tests pass for all three targets
- OS support and non-support boundaries are documented
- three demo GIFs are ready
- release notes are short and clear
- issue and PR templates are ready

## Operating Targets

Early 90-day internal targets:

- install to first successful send conversion: 50% or better
- median time to first success: under 3 minutes
- `doctor --fix` resolution rate: 60% or better
- 7-day reuse rate: 25% or better
- `go` success rate: 85% or better
- launch and paste failure rate: trending down over time

## Final Judgement

If this sentence feels sharper after 90 days, the roadmap is on track:

> `prtr` is the command layer that turns intent into the next action as quickly as possible.
