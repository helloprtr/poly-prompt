# Release Messages

Short PR and announcement copy for the current release train.

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
