# prtr v0.6.1 End-to-End Test Guide

## 1. Purpose

This guide is for validating `prtr v0.6.1` from A to Z.

If you want a ready-to-use checkbox sheet for the hands-on eval repo, use `docs/V0_6_1_FIXTURE_CHECKLIST.md` together with this guide.

The goal is not only to confirm that commands run.

The goal is to confirm that:

- the product story is true
- the core loop works
- the support surfaces are trustworthy
- repo memory behaves correctly
- release and install paths are healthy

## 2. Test Outcome Standard

A test pass means all of the following are true:

- the command exits successfully when success is expected
- the command fails clearly when failure is expected
- the output matches the documented product story
- no confusing mismatch exists between docs, help text, and behavior
- the result is useful to a real user, not only technically correct

## 3. Recommended Test Environments

Minimum recommended environments:

1. macOS with Homebrew
2. macOS inside a Git repo with clipboard available
3. one clean shell with no prior `prtr` config
4. one existing repo with `README.md` and `docs/`

Nice-to-have environments:

1. Linux shell
2. Windows shell
3. shell without DeepL key
4. shell with launcher or automation limitations

## 4. Prerequisites

Before testing:

1. note the version under test
2. choose whether you are testing installed binary or local build
3. prepare one disposable repo for `learn` and `sync`
4. decide whether to test real delivery or `--dry-run`

Recommended prep commands:

```bash
prtr version
prtr --help
prtr platform
prtr doctor
```

## 4.1 Fast hands-on fixture

If you want one compact scenario that exercises the core loop, use the built-in eval fixture.

Setup:

```bash
./scripts/eval/setup_v0_6_1_fixture.sh
cd /tmp/prtr-v0.6.1-eval
```

What it contains:

- a small Node project with a failing checkout test
- repo-local README and docs content for `learn`
- a sample AI answer for `take patch`
- a dirty working tree so repo context is visible during testing

Core evaluation sequence:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘" --dry-run
prtr swap gemini --dry-run
cat fixtures/sample_ai_answer.md | pbcopy
prtr take patch --dry-run
prtr again --edit
prtr learn README.md docs
prtr inspect --json "show the route"
```

What to judge:

1. did `go` reach a useful first prompt quickly
2. did `swap` preserve intent without rewrite
3. did `take patch` produce an action-ready implementation prompt
4. did `again --edit` make iteration cheaper than starting over
5. did `learn` save terms and guidance that match the checkout domain
6. did `inspect --json` explain the route clearly enough for debugging

Why this fixture is useful:

- it produces real failing test output
- it is small enough to understand in minutes
- it exercises both prompt routing and repo memory
- it gives you a repeatable way to compare app quality and loop continuity

## 5. Test Areas

The full test scope is:

1. install and upgrade
2. first-run activation
3. diagnostics and platform visibility
4. core loop
5. continuity and memory
6. expert and automation surfaces
7. docs and release consistency
8. failure handling

## 6. Test Checklist

### A. Release and install surface

#### A1. GitHub Release exists

Check:

- the `v0.6.1` GitHub Release page exists
- archives exist for macOS, Linux, and Windows
- `checksums.txt` exists
- release body reflects the intended `v0.6.1` message

Expected result:

- release is public
- not draft
- not prerelease

#### A2. Homebrew formula is updated

Check:

- `brew info prtr`
- formula version is `0.6.1`

Expected result:

- Homebrew stable version is `0.6.1`

#### A3. Upgrade path works

Run:

```bash
brew update
brew upgrade prtr
prtr version
```

Expected result:

- upgrade succeeds
- `prtr version` prints `0.6.1`

### B. Global help and product story

#### B1. Root help matches product story

Run:

```bash
prtr --help
```

Check:

- hero line says `prtr turns what you mean into the next AI action.`
- visible loop centers on `go`, `swap`, `take`, `learn`
- expert surfaces are secondary

Expected result:

- help output matches README and site positioning

#### B2. Command help is consistent

Run:

```bash
prtr go --help
prtr swap --help
prtr take --help
prtr learn --help
```

Check:

- `go` is framed as the first send
- `swap` is framed as comparison
- `take` is framed as answer-to-action
- `learn` is framed as repo memory

Expected result:

- wording and intent match docs

### C. First-run activation

#### C1. `start` works for a new user

Run:

```bash
prtr start
```

Check:

- onboarding path is clear
- `doctor` runs before the first send when appropriate
- user is not dropped into unclear setup state

Expected result:

- the first action feels guided

#### C2. `setup` remains available as advanced path

Run:

```bash
prtr setup
```

Expected result:

- advanced configuration flow still exists
- it is clearly secondary to `start`

### D. Diagnostics and readiness

#### D1. `doctor` reports state clearly

Run:

```bash
prtr doctor
```

Check:

- config readiness
- translation readiness
- clipboard readiness
- launcher readiness
- platform matrix visibility

Expected result:

- output is readable and actionable

#### D2. `doctor --fix` is safe

Run:

```bash
prtr doctor --fix
```

Expected result:

- only safe fixes are applied
- unresolved issues still produce fallback guidance

#### D3. `platform` mirrors readiness surface

Run:

```bash
prtr platform
prtr platform --json
```

Expected result:

- output is consistent with `doctor`
- JSON mode is usable for automation

### E. `go` from first request to preview

#### E1. Plain request works

Run:

```bash
prtr go "이 에러 원인 분석해줘" --dry-run
```

Check:

- request is accepted in Korean
- final prompt is visible
- app and mode are reasonable

#### E2. Mode-specific behavior works

Run:

```bash
prtr go ask "이 문서 요약해줘" --dry-run
prtr go review "이 PR에서 위험한 부분만 짚어줘" --dry-run
prtr go fix "왜 테스트가 깨지는지 정확한 원인만 찾아줘" --dry-run
prtr go design "이 기능 구조 설계해줘" --dry-run
```

Expected result:

- wording changes by mode
- routing defaults make sense

#### E3. Piped evidence works

Run:

```bash
printf 'panic: nil pointer dereference\n' | prtr go fix "왜 깨지는지 찾아줘" --dry-run
```

Expected result:

- piped content is treated as evidence

#### E4. Repo context works

Inside a Git repo:

```bash
prtr go fix "이 테스트 왜 실패하는지 찾아줘" --dry-run
```

Expected result:

- repo summary is included
- changed files context is reflected when available

#### E5. `--no-context` really removes auto context

Run:

```bash
prtr go fix "이 테스트 왜 실패하는지 찾아줘" --dry-run --no-context
```

Expected result:

- repo and stdin evidence are not attached automatically

#### E6. Next-step suggestions match mode

Check after `go` output:

- `fix` suggests `take patch` and `take test`
- `review` suggests `swap <other-app>` and `take issue`
- `design` suggests `take plan` and `take patch`

Expected result:

- next-step guidance feels intentional and mode-aware

### F. `swap` comparison flow

Prerequisite:

- run one successful `go` first

#### F1. Compare another app without rewrite

Run:

```bash
prtr swap claude --dry-run
prtr swap codex --dry-run
prtr swap gemini --dry-run
```

Expected result:

- latest request and mode are reused
- destination app changes
- no manual rewrite is needed

#### F2. `swap` with `--edit`

Run:

```bash
prtr swap claude --edit
```

Expected result:

- the reused prompt can still be reviewed before send

### G. `take` answer-to-action flow

Prerequisite:

- copy sample text into clipboard

Example:

```bash
printf 'We found three risky files and need a safe implementation path.' | pbcopy
```

#### G1. All seven actions are visible

Run:

```bash
prtr take --help
```

Expected result:

- `patch`
- `issue`
- `plan`
- `summary`
- `test`
- `commit`
- `clarify`

#### G2. Representative actions work

Run:

```bash
prtr take patch --dry-run
prtr take issue --dry-run
prtr take plan --dry-run
```

Expected result:

- each action produces clearly different follow-up intent

#### G3. Secondary actions work

Run:

```bash
prtr take summary --dry-run
prtr take test --dry-run
prtr take commit --dry-run
prtr take clarify --dry-run
```

Expected result:

- each prompt shape matches the named action

#### G4. Empty clipboard failure is clear

Run with empty clipboard if possible:

```bash
prtr take patch
```

Expected result:

- error is explicit
- user knows what to do next

### H. `learn` repo memory flow

Use a disposable repo with:

- `README.md`
- `docs/`

#### H1. Dry-run previews both outputs

Run:

```bash
prtr learn --dry-run README.md docs
```

Expected result:

- preview contains `.prtr/termbook.toml`
- preview contains `.prtr/memory.toml`

#### H2. Save creates both files

Run:

```bash
prtr learn README.md docs
ls -l .prtr/termbook.toml .prtr/memory.toml
```

Expected result:

- both files exist
- output includes new protected term count
- output includes repo summary
- output includes guidance lines

#### H3. Default source set works

Run in a normal repo:

```bash
prtr learn
```

Expected result:

- default sources are discovered
- command succeeds without explicit paths when the repo has standard files

#### H4. `--reset` rebuilds both layers

Run:

```bash
prtr learn --reset
```

Expected result:

- both termbook and memory are rebuilt rather than merged

### I. `sync` canonical guidance flow

#### I1. Initialize canonical repo guidance

Run:

```bash
prtr sync init
prtr sync status
```

Expected result:

- `.prtr` guidance structure is visible

#### I2. Dry-run vendor file rendering

Run:

```bash
prtr sync --dry-run
```

Expected result:

- user can see what would be rendered

#### I3. Write vendor-facing files

Run:

```bash
prtr sync --write claude,codex
```

Expected result:

- vendor-facing guide files are written from canonical guidance

### J. `again` and `inspect`

#### J1. `again` replays the recent loop

Run:

```bash
prtr again --dry-run
```

Expected result:

- latest run is replayed

#### J2. `inspect` explains the route

Run:

```bash
prtr inspect
```

Expected result:

- user can inspect raw details and composition logic

### K. `exec` and `server`

#### K1. `exec` works as headless path

Run:

```bash
prtr exec fix "Find the real reason this is failing."
prtr exec review "Summarize the risky parts only." --to claude --json
```

Expected result:

- headless execution path works
- JSON output is machine-readable

#### K2. `server` is clearly alpha

Run:

```bash
prtr server --addr 127.0.0.1:8787
```

Expected result:

- server starts or fails clearly
- alpha status is not hidden

### L. Docs and product consistency

#### L1. README matches the product

Check:

- hero message
- current public release
- loop ordering

#### L2. Site matches the product

Check:

- home hero
- GIF demos
- release timeline

#### L3. Docs hub is current

Check:

- release messages link
- report link
- test guide link

Expected result:

- no obvious product-story mismatch

## 7. High-Risk Failure Cases to Test on Purpose

These are worth testing deliberately:

1. no config present
2. invalid config present
3. missing DeepL key
4. empty clipboard before `take`
5. no Git repo for repo-context dependent flows
6. repo without `README.md` or `docs/`
7. launcher unavailable
8. automation unavailable
9. Homebrew update lag after release

## 8. Suggested Sign-Off Criteria

`v0.6.1` should be considered fully healthy when all of the following are true:

1. GitHub Release is public and complete
2. Homebrew installs and upgrades to `0.6.1`
3. root help and command help match the docs
4. `go -> swap -> take -> learn` works end to end
5. `learn` updates both termbook and memory
6. docs, site, and release copy all agree on the current story
7. no critical failure is silent or confusing

## 9. Recommended Test Order

If you want the shortest practical full-pass order, use this:

1. install and version
2. root help
3. `doctor`
4. `platform`
5. `start`
6. `go`
7. `swap`
8. `take`
9. `learn`
10. `sync`
11. `again`
12. `inspect`
13. `exec`
14. site and docs consistency
15. release and Homebrew verification

## 9.1 Fast product evaluation rubric

Use this rubric if your goal is not only correctness, but product judgment.

### Speed

Check:

- time to first useful `go` preview
- time to compare with `swap`
- time to reach an implementation-ready prompt with `take patch`

`prtr` feels strong when the loop moves forward without prompt rewriting or tool-switch friction.

### Clarity

Check:

- whether the help text explains what to do next
- whether next-step suggestions feel intentional
- whether `inspect` makes the route understandable

`prtr` feels strong when a user does not need to guess what the next command should be.

### Continuity

Check:

- whether `swap` preserves the original intent
- whether `again --edit` feels cheaper than rebuilding the flow
- whether `learn` reduces repeated explanation cost

`prtr` feels strong when the second and third steps are easier than the first.

### Trust

Check:

- whether `doctor` and `platform` explain readiness honestly
- whether failure states are readable
- whether docs, help, site, and release story match

`prtr` feels strong when the product is explicit about both its strengths and its boundaries.

## 10. Final Notes for Manual Testers

While testing, do not only ask:

- "Did the command run?"

Also ask:

- "Did the product explain itself well?"
- "Would a real user know what to do next?"
- "Did the output reduce friction?"
- "Did the tool make the loop feel cheaper to continue?"

That is the real standard for `prtr v0.6.1`.
