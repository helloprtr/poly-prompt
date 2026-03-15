# prtr PRD v1

*2026-03-15*

## Purpose

This document defines the product requirements for `prtr vNext`.

The core goal is simple:

> Minimize the time between install and the first reliable next action.

## Product Definition

`prtr` is a mode-first AI command layer.

People can speak in their own language. `prtr` turns that intent into the right `mode`, `app`, and `delivery`, then helps them continue into the next useful action instead of stopping at the first answer.

The product is not about prettier prompts.
It is about keeping people moving.

## Problem

AI CLI usage usually breaks in three places.

### 1. Starting is hard

Users hit app choice, prompt rewriting, context rebuilding, and launch or paste friction at the same time.

### 2. The flow breaks after the first answer

People get a review but do not turn it into a patch.
They get a fix analysis but do not continue into tests, commits, issues, or plans.

### 3. The three agents drift apart

Claude, Gemini, and Codex each have their own memory, guide, permission, extension, and automation surfaces. Managing them separately becomes exhausting quickly.

## Target Users

### A. Beginner builder

Someone who wants AI to speed up development but does not want to learn each vendor CLI in depth.

Success means:

- they reach first success from the README
- mode choice is enough to get started
- failure recovery is guided by doctor
- the next action after the first answer is easy

### B. Working developer

Someone who compares Claude, Gemini, and Codex regularly.

Success means:

- the same request can be resent to another app quickly
- the same working context stays intact
- patch, test, review, and summary follow-ups are cheap

### C. Power user or builder

Someone who wants to call `prtr` from scripts, CI, or another agent.

Success means:

- structured output exists
- background execution exists
- event logging is stable
- there is a server integration surface

## Goals

### Product goals

1. `Activation`
Lower the friction from install to first successful send.

2. `Continuity`
Lower the cost of the next action after the first answer.

3. `Reliability`
Explain failure clearly and repair what can be repaired.

4. `Portability`
Provide a common command layer across Claude, Gemini, and Codex.

5. `Automation readiness`
Keep the human-first UX while making the surface scriptable.

### Non-goals

- building a new agent runtime
- building a new IDE
- building a new terminal multiplexer
- chasing perfect feature parity across vendors

## North Star Metric

**Time from install to first reliable next action**

Supporting metrics are grouped like this.

### Activation

- install to first successful send conversion
- time to first success
- `start` completion rate
- `doctor --fix` resolution rate

### Usage

- `go` success rate
- `swap` usage rate
- `take` usage rate
- `again` usage rate
- `learn` adoption rate
- 7-day reuse rate

### Reliability

- launch failure rate
- paste failure rate
- translation failure rate
- history corruption rate
- `doctor` false positive rate

### Growth

- release downloads
- GitHub stars per week
- issue to retained user conversion
- number of community recipes

## Product Principles

**P1. Mode-first**
Ask what kind of help is wanted before asking which app should be used.

**P2. Next-action-first**
Optimize for continuity after the first answer, not just for first-answer quality.

**P3. Sync-first**
Reduce drift across vendor guide and memory surfaces.

**P4. Doctor-first**
Do not stop at explanation. Offer repair and fallback.

**P5. Human plus script duality**
The same product should feel easy for a person and dependable for automation.

## Current State and Opportunity

Today `prtr` already ships the `go / swap / take / again / learn / inspect` loop plus `open-copy` delivery. It also documents local-only history and the lack of full auto submit support.

That means the first-send experience already has value, but the real product opportunity is now in:

- reliability and repair
- continuity after the first answer
- sync across agent surfaces
- machine-readable execution and logs

## Requirements

### R1. Command surface

The vNext public surface should be centered on:

- `prtr start`
- `prtr go`
- `prtr swap`
- `prtr take`
- `prtr again`
- `prtr learn`
- `prtr sync`
- `prtr doctor --fix`
- `prtr fanout`
- `prtr collect`
- `prtr inspect`

Acceptance criteria:

- a beginner should feel six or fewer commands on the first screen
- `go`, `swap`, `take`, and `again` should share one mental model
- `inspect` should remain the expert-facing raw path

### R2. Sync

`sync` is a core product feature, not an extra utility.

It should not invent a new memory system. It should render from one canonical source into vendor-native surfaces.

Canonical source:

- `.prtr/guide.md`
- `.prtr/termbook.toml`
- `.prtr/memory.toml`
- `.prtr/packs/`

Render targets:

- `CLAUDE.md`
- `GEMINI.md`
- `AGENTS.md`

Commands:

- `prtr sync init`
- `prtr sync status`
- `prtr sync`
- `prtr sync --write claude,gemini,codex`

Requirements:

- diff-first
- preview before overwrite
- vendor-specific comment markers
- generated file provenance
- partial target writes
- dry-run support

Non-goals in v1:

- syncing full vendor settings files
- syncing hooks, plugins, or extensions end to end
- syncing secrets or auth state

### R3. Delivery

Delivery is split into three layers.

#### `open-copy`

The shipped default.
Optimized for beginner activation and visible terminal handoff.

#### `exec`

Headless subprocess execution.
This is the first real background engine.

#### `server`

Long-running session and orchestration surface.

Requirements:

- one internal run model across all delivery types
- shared prompt rendering across delivery modes
- shared result storage format
- `open-copy` fallback when possible
- background run ids
- `collect`, `tail`, and `resume` support

### R4. JSON event log

Every run should leave an event log.

Storage:

- `.prtr/runs/<run-id>.jsonl`
- `.prtr/runs/<run-id>.meta.json`

Event types:

- `run.started`
- `prompt.rendered`
- `delivery.started`
- `delivery.completed`
- `agent.output`
- `repair.suggested`
- `run.completed`
- `run.failed`

Stdout policy:

- default output is human-readable
- `--json` enables machine-readable output
- stderr stays concise

Acceptance criteria:

- one run should make sense to a human
- the same run should be machine-parsable
- there should be a shortcut for the final answer only

### R5. Doctor

`doctor` is a conversion tool, not just a diagnostics screen.

Checks:

- Claude, Gemini, and Codex CLI presence
- auth state
- clipboard provider
- launcher availability
- paste backend
- trust, sandbox, and approval mismatches
- sync drift
- history integrity

Fix actions:

- missing dependency guidance
- safe fallback delivery recommendation
- invalid config reset
- generated file rebuild
- auth rerun shortcut
- OS-specific paste fallback change

Acceptance criteria:

- fix advice appears before raw error detail
- false positives are tracked as a KPI
- destructive mutations require preview

### R6. Trust profiles

Reduce vendor-specific permission language into:

- `safe`
- `work`
- `yolo`

Mapping intent:

- `safe` for read, summarize, and review
- `work` for workspace edits with limited auto-approval
- `yolo` for full automation in isolated environments

Acceptance criteria:

- users should not need vendor jargon
- internal mapping should still be logged clearly
- `yolo` should never turn on silently

### R7. Fanout and collect

`swap` remains the comparison UX.
Parallel execution is separated into `fanout`.

Commands:

- `prtr fanout review "..." --to claude,gemini,codex --delivery exec`
- `prtr collect --latest`
- `prtr tail <run-id|app>`

Requirements:

- parallel dispatch
- per-target status tracking
- partial failure tolerance
- fastest-first collection
- final summary and raw outputs preserved together

### R8. Start

`start` should become the main activation entry.

Flow:

1. detect default app and language
2. run setup
3. run doctor
4. do a sample send
5. confirm success
6. suggest the next action

Acceptance criteria:

- a README-only user can finish it
- failure falls directly into `doctor --fix`
- target median time to first success is under three minutes

## UX Flows

### First run

`install -> start -> go`

### Daily use

`go -> swap -> take -> again -> learn`

### Failure recovery

`go fails -> doctor --fix -> fallback delivery -> go again`

### Background execution

`fanout --delivery exec -> collect`

## Investor Logic

This is not “another AI CLI.”

### Wedge

Eliminate activation friction for beginners.

### Product moat

- mode-first command vocabulary
- cross-agent continuity
- sync layer
- repair data
- reusable packs, recipes, and event logs

### Expansion path

- personal CLI
- team pack, policy, and sync layer
- CI and automation layer
- agent-to-agent orchestration surface

## Risks

**Risk 1. Overpromising**
Mitigation: split README copy into Available, Planned, and Alpha.

**Risk 2. OS-level paste reliability**
Mitigation: document `open-copy` honestly and ship `exec` quickly.

**Risk 3. Vendor drift**
Mitigation: keep sync and render layers thin and strengthen doctor.

**Risk 4. Scope explosion**
Mitigation: do not become a full terminal or a full agent runtime.

## Release Gate

Before release, all seven answers should be yes.

1. Can a person start from the README alone?
2. Can a user succeed from mode choice without knowing the app first?
3. Does doctor provide real recovery help?
4. Is the next action easy after the first answer?
5. Are expert capabilities still intact?
6. Are OS-specific limits shown honestly?
7. Can the product be explained in one sentence?

## Final Definition

> `prtr` is the command layer that turns a beginner's intent into the next useful AI action as quickly as possible.
> It should stay easy for people, scripts, CI, and other agents to call.
