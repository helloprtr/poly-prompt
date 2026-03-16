# prtr v0.6.1 Product Report

## 1. Executive Summary

`prtr v0.6.1` is the point where the product can be described clearly, demonstrated clearly, shipped clearly, and tested clearly.

From `v0.4.0` to `v0.6.1`, `prtr` moved through four layers of maturity:

1. command foundation
2. first-run success and recovery
3. repo-aware continuity
4. public product clarity and release reliability

The result is not "a prompt translation CLI."

The result is:

`prtr turns what you mean into the next AI action.`

That means:

- write in your language
- choose the right mode
- route to Claude, Codex, or Gemini
- keep the next action close
- remember the repo for the next run

In practical terms, the product now centers on:

- `go` = start
- `swap` = compare
- `take` = execute
- `learn` = remember

Everything else exists to make that loop more trustworthy, easier to start, and easier to repeat.

## 2. Product Evolution: v0.4.0 -> v0.6.1

### v0.4.0: Command foundation

`v0.4.0` was the command-tree foundation release.

Based on the repository history, this is the point where the Cobra command surface became the structural base for the CLI. It matters because later releases depend on that shape for discoverability, help output, and product surface expansion.

User impact:

- commands become easier to discover
- help output becomes more consistent
- the CLI is now ready for a real public surface instead of ad hoc flows

### v0.4.1: First-run front door

This is where `prtr start` became the beginner-first entry.

What changed:

- `start` became the front door
- `setup` moved to the advanced compatibility path
- release packaging became more structured

User impact:

- lower activation friction
- clearer "what do I run first?" answer
- better separation between beginner flow and advanced configuration

### v0.4.2: Repair and recovery

This is where `doctor` became a real recovery surface.

What changed:

- `doctor --fix`
- platform matrix summary
- better diagnostics for clipboard, launcher, paste, and translation failures

User impact:

- less silent failure
- faster recovery from broken environment state
- better trust because readiness is visible

### v0.4.3: Repo memory and sync

This is where `prtr` stopped being only about one-off prompts and started becoming repo-aware.

What changed:

- `prtr sync init`
- `prtr sync status`
- `prtr sync`
- `.prtr/memory.toml`
- richer routing metadata and history

User impact:

- repo-specific guidance becomes reusable
- follow-up runs get more stable
- app-specific guide files can be generated from one canonical source

### v0.4.5: Platform visibility

This release made environment capability easier to inspect before failure.

What changed:

- `prtr platform`
- shared matrix logic across `platform` and `doctor`

User impact:

- faster readiness checks
- clearer support boundaries across macOS, Linux, and Windows

### v0.5.0: Headless and automation surface

This is where `prtr` became more than a desktop handoff tool.

What changed:

- `prtr exec`
- `prtr server` alpha
- stronger first-run readiness
- stronger next-action framing in docs and site

User impact:

- scriptable execution path exists
- orchestration path exists in alpha form
- the product becomes easier to automate and easier to explain

### v0.5.1: Public product story alignment

This release is where the product finally started saying the same thing everywhere.

What changed:

- README, docs, site, and help output aligned
- public loop became `go -> swap -> take -> learn`
- `take` action list standardized
- `swap`, `take`, and `learn` reframed in value language

User impact:

- less confusion on first contact
- stronger install motivation
- better understanding of why people should keep using it

### v0.6.0: Continuity becomes visible

This is where the internal loop value became more explicit in both behavior and launch assets.

What changed:

- `go` now suggests next steps by mode
- `learn` updates both `.prtr/termbook.toml` and `.prtr/memory.toml`
- `learn --dry-run` previews both
- `learn` save output is richer
- launch GIFs, demo scripts, and release checklist were added

User impact:

- better momentum after the first answer
- stronger repo memory behavior
- easier self-onboarding and product demonstration

### v0.6.1: Release reliability hardening

This is an operations safety release.

What changed:

- release workflow moved to Node 24 compatible action versions
- GoReleaser action wrapper was replaced with direct CLI install and execution
- GoReleaser was pinned to a Go 1.24 compatible version

User impact:

- no product loop change
- more predictable GitHub Release and Homebrew publication
- lower operational risk for future releases

## 3. What prtr Is in v0.6.1

`prtr` is a command layer for multilingual AI work.

It is not just:

- translation
- prompt formatting
- clipboard automation

It is the layer that connects:

- user intent
- mode selection
- app selection
- repo context
- next action
- repo memory

The current public identity is:

`prtr turns what you mean into the next AI action.`

## 4. The Core Product Surface

### `go`

Purpose:

- start fast
- pick a mode
- route to the right app
- include evidence and repo context

User experience:

- "I can begin in Korean."
- "I do not have to pre-optimize the prompt."
- "The tool helps me reach the first useful send quickly."

### `swap`

Purpose:

- compare Claude, Codex, and Gemini without rebuilding context

User experience:

- "I can test another app without rewriting the whole request."
- "Comparison cost is low enough to become normal."

### `take`

Purpose:

- turn an answer into a patch, issue, plan, summary, test, commit, or clarification prompt

User experience:

- "The answer is not the end."
- "I can keep moving without rethinking the whole next prompt."

### `learn`

Purpose:

- save repo-specific terms, names, guidance, and preferred language into local memory

User experience:

- "I do not need to reteach the project every time."
- "The repo starts to feel known."

## 5. Support Surfaces That Make the Loop Trustworthy

These surfaces are not the main story, but they are critical to user confidence:

- `start`
- `doctor`
- `platform`
- `sync`
- `again`
- `inspect`
- `exec`
- `server` alpha

Their role is simple:

- make activation easier
- make failure states clearer
- make power use possible

## 6. The User Experience prtr Now Delivers

### First-time user

The first-time user gets:

- a clear entry point with `start`
- diagnostics and repair through `doctor`
- a visible core loop
- a short mental model

### Repeating user

The repeating user gets:

- cheap app switching through `swap`
- cheap next-step prompting through `take`
- better continuity through `learn`

### Repo owner or team lead

The repo owner gets:

- canonical guidance via `.prtr`
- repeatable vendor guide generation with `sync`
- more stable wording across follow-up runs

### Power user or automation-focused user

The power user gets:

- `exec`
- `server` alpha
- `inspect`
- `platform`

## 7. What Is Most Important in v0.6.1

The most important thing is not a single command.

The most important thing is that the loop is now coherent:

- start is easier
- recovery is clearer
- repo memory is real
- the product message is aligned
- the release pipeline is safer

That combination gives `prtr` three strong properties:

1. it is easier to try
2. it is easier to trust
3. it is easier to keep using

## 8. Current Boundaries

`prtr v0.6.1` is stronger, but it is still intentionally honest about its boundaries.

Current boundaries:

- history is local-only
- full auto submit is not supported
- `--submit confirm` is still macOS-only
- `server` remains alpha
- `fanout` and `collect` are planned, not public

These boundaries are important because the product gains trust when it is explicit about what is real now.

## 9. Why This Matters Competitively

Many AI tools compete on model access.

`prtr` competes on workflow compression.

Its strongest differentiators are:

- multilingual intent entry
- mode-first routing
- cheap cross-app comparison
- answer-to-action continuity
- repo-local memory
- honest operator surfaces for install, repair, and release

The deeper message is:

`prtr` does not just help you ask.
`prtr` helps you keep moving.`

## 10. Recommended Positioning Lines

### Hero

`prtr turns what you mean into the next AI action.`

### Subcopy

`Write in your language. Route to Claude, Codex, or Gemini. Keep the loop moving.`

### Short one-liners

- `One intent, many AI apps, zero prompt babysitting.`
- `Start fast. Swap instantly. Take action. Learn the project.`
- `Write in Korean. Send to Codex. Swap to Claude. Take the patch.`
- `The command layer for multilingual AI work.`

### Korean one-liners

- `한국어로 시작해서, 가장 맞는 AI 앱의 다음 액션으로 바로 연결합니다.`
- `질문을 잘 쓰게 해주는 툴이 아니라, 일을 앞으로 밀어주는 커맨드 레이어입니다.`
- `go로 시작하고, swap으로 비교하고, take로 실행하고, learn으로 기억합니다.`

## 11. Ready-to-Use Promotion Copy

### Launch post: English

`prtr v0.6.1` is out.

This release does not change the public `go -> swap -> take -> learn` loop. Instead, it hardens the release path behind it: Node 24 compatible GitHub Actions, a pinned GoReleaser CLI install, and a Go 1.24 compatible release toolchain pin.

The result is simpler: future GitHub Releases and Homebrew updates should be quieter, safer, and easier to trust.

### Launch post: Korean

`prtr v0.6.1` 배포 완료.

이번 버전은 사용자 기능을 늘리는 릴리스가 아니라, 배포 안정성을 높이는 릴리스입니다. 사용자가 보는 `go -> swap -> take -> learn` 루프는 그대로 유지하면서, GitHub Release와 Homebrew 배포 파이프라인을 더 안전하고 예측 가능하게 정리했습니다.

앞으로 릴리스 경고는 줄고, 배포는 더 안정적으로 반복될 수 있습니다.

### Product post: English

`prtr` is not a prompt toy.

It is a command layer that turns what you mean into the next AI action:

- start with `go`
- compare with `swap`
- execute with `take`
- remember with `learn`

### Product post: Korean

`prtr`는 번역 프롬프트 툴이 아닙니다.

사용자의 의도를 가장 적절한 AI 앱의 다음 액션으로 빠르게 연결하는 커맨드 레이어입니다.

- `go` = 시작
- `swap` = 비교
- `take` = 실행
- `learn` = 기억

## 12. Final Assessment

`v0.6.1` is not just "the latest version."

It is the first version where:

- the loop is clear
- the story is clear
- the demos are ready
- the repo memory model is visible
- the release path is reliable enough to support repeated launches

That makes `v0.6.1` the strongest public base so far for user testing, promotion, and deeper product validation.
