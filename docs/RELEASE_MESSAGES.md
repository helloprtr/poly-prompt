# Release Messages

Short PR and announcement copy for the current release train.

## v0.6.2

**PR title**
`feat: clarify open-copy handoff and add selectable submit on macOS`

**PR description**
This release improves the part of `prtr` people touch every day after the first route is chosen: the open-copy handoff.

What changed:

- `prtr doctor` and `prtr platform` now print a short open-copy summary that explains whether launch, paste, and submit are ready
- compact `go` and `swap` status output now explains what happened to the prompt after routing
- macOS now supports `--submit auto` alongside `--submit confirm`
- `PRTR_TERMINAL_APP` is now surfaced more clearly in diagnostics, with explicit guidance when a selected terminal is unsupported for open-copy launch
- docs and site copy now reflect the current macOS handoff surface more honestly

Why it matters:

- users can tell whether `prtr` will open, paste, or wait for manual send before trial and error
- macOS users can choose between manual, confirm, and auto send behavior
- terminal preference is now visible and debuggable instead of hidden behind defaults
- the release surface better matches the actual day-to-day user experience

**Announcement**
`v0.6.2` makes the open-copy handoff much easier to read and control. `doctor` and `platform` now explain launch, paste, and submit readiness in plain language, macOS supports `--submit auto`, and `PRTR_TERMINAL_APP` is now visible in diagnostics instead of hidden behind hardcoded assumptions.

**Announcement (long)**
`prtr v0.6.2` is the release where the handoff layer gets easier to trust.

Instead of only showing raw diagnostics, `doctor` and `platform` now explain what will actually happen during open-copy:

- will the target app open?
- will the compiled prompt paste automatically?
- do you still need to press Enter yourself?

This release also adds selectable submit behavior on macOS:

- manual send remains the default
- `--submit confirm` stays available
- `--submit auto` is now supported

Finally, `PRTR_TERMINAL_APP` now shows up clearly in diagnostics, so terminal preference no longer feels like hidden magic.

**Announcement (KR)**
`prtr v0.6.2`는 open-copy handoff를 실제 사용 기준으로 더 신뢰할 수 있게 만든 릴리스입니다.

이제 `doctor`와 `platform`은 단순 체크리스트를 넘어서,

- 앱이 실제로 열리는지
- 프롬프트가 자동으로 붙여넣기 되는지
- 마지막 Enter를 사용자가 직접 눌러야 하는지

를 더 직접적으로 설명합니다.

또한 macOS에서는 submit 동작을 사용자가 직접 선택할 수 있게 됐습니다.

- 기본은 manual
- `--submit confirm` 계속 지원
- `--submit auto` 새로 지원

그리고 `PRTR_TERMINAL_APP`도 진단 표면에 더 명확하게 드러나서, 예전처럼 하드코딩된 기본값 뒤에 숨어 있는 느낌이 줄어듭니다.

## v0.6.1

**PR title**
`ci: harden release workflow for v0.6.1`

**PR description**
This release hardens the `prtr` release path without changing the public product loop.

What changed:

- moved the release workflow to Node 24 compatible GitHub Action versions
- replaced the GoReleaser action step with a pinned GoReleaser CLI install
- pinned GoReleaser to a Go 1.24 compatible version so release tags stay reproducible on the current runner image
- kept the `v0.6.0` product surface intact while making future release tags safer to ship

Why it matters:

- fewer release-time warnings
- more predictable GitHub Release runs
- lower risk of Homebrew publication drift
- easier future maintenance when GitHub Actions runtime defaults change

**Announcement**
`v0.6.1` hardens the release path around Node 24 compatible GitHub Actions, replaces the GoReleaser action with a pinned GoReleaser CLI install, and makes future release tags safer to ship.

**Announcement (long)**
`prtr v0.6.1` is a release-operations safety cut.

It does not change the public `go -> swap -> take -> learn` loop, but it does make the release pipeline much safer:

- Node 24 compatible GitHub Action versions
- pinned GoReleaser CLI install instead of the action wrapper
- Go 1.24 compatible GoReleaser pin for more predictable release runs

Result: future GitHub Releases and Homebrew updates should be quieter, more stable, and easier to trust.

**Announcement (KR)**
`prtr v0.6.1`은 기능 추가 릴리스가 아니라 배포 안정화 릴리스입니다.

사용자 입장에서 보이는 `go -> swap -> take -> learn` 루프는 그대로 유지하면서, 실제 릴리스 파이프라인을 더 안전하게 정리했습니다.

- Node 24 호환 GitHub Actions 버전으로 정리
- GoReleaser GitHub Action 대신 고정 버전 CLI 설치 방식으로 전환
- 현재 runner 환경인 Go 1.24와 호환되는 GoReleaser 버전으로 고정

결과적으로 앞으로 GitHub Release와 Homebrew 배포가 더 예측 가능하고, 릴리스 때 생기던 불필요한 경고도 줄어듭니다.

## v0.5.1

**PR title**
`docs: align public surfaces around the AI command layer loop`

**Announcement**
Unified README, docs, site, and CLI help around one message: `prtr` turns what you mean into the next AI action, with the public loop centered on `go`, `swap`, `take`, and `learn`.

## v0.6.0

**PR title**
`feat: strengthen take/learn continuity around the next action`

**Announcement**
Start fast. Swap instantly. Take action. Learn the project. `v0.6.0` adds mode-aware next-step suggestions after `go`, expands `learn` to update repo memory alongside the termbook, and ships the demo asset kit plus launch checklist for a cleaner public rollout.

## v0.4.1

**PR title**
`feat: make start the beginner-first entry`

**Announcement**
Introduced `prtr start`, moved `setup` into the advanced compatibility path, and made the first-run flow easier to explain and complete.

## v0.4.2

**PR title**
`feat: add doctor --fix and repair-ready diagnostics`

**Announcement**
Added `doctor --fix`, platform matrix visibility, and clearer repair guidance for clipboard, launcher, paste, and auth failures.

## v0.4.3

**PR title**
`feat: add sync, routing metadata, and repo memory`

**Announcement**
Added canonical `.prtr` sync, vendor guide generation, richer routing metadata, and repo-local memory for more consistent follow-up runs.

## v0.4.5

**PR title**
`feat: add platform surface and shared matrix reporting`

**Announcement**
Added `prtr platform` plus shared platform matrix reporting so supported surfaces and delivery constraints are visible before users get stuck.

## v0.5.0

**PR title**
`feat: add exec/server and reposition prtr around the next action`

**Announcement**
Added `exec` and `server` alpha surfaces, aligned the docs and site around the next-action command layer, and tightened first-run readiness across `start`, `doctor`, and `sync`.
