# Community Growth and Safety Playbook

This guide is for opening `prtr` to more contributors without turning the repository into an untriageable or unsafe inbox.

## What to put in place first

### 1. Tighten the intake

- Keep blank issues disabled so all public bug reports and feature requests go through forms.
- Route open-ended ideas to Discussions instead of Issues.
- Require version, OS, terminal, command, and reproduction steps for bugs.
- Add `needs-triage` automatically so new reports do not get lost.

### 2. Protect `main`

In GitHub Settings -> Branches -> Add rule, enable:

- Require a pull request before merging
- Require approvals
- Require status checks to pass before merging
- Dismiss stale approvals when new commits are pushed
- Restrict force pushes and deletions

Recommended required checks:

- `ci / Go checks`
- `ci / Docs and site build`
- `codeql / analyze (go)`
- `codeql / analyze (javascript-typescript)`

### 3. Review with a safety lens

Ask these questions on outside PRs:

- Does this add or replace a dependency?
- Does it expand shell execution, automation, clipboard, network, or filesystem access?
- Does the test actually prove the new behavior?
- Does the docs change match the runtime change?
- Could this leak secrets, tokens, or local project data?

When a PR touches execution, launch automation, filesystem writes, or release workflows, label it for extra review even if the change looks small.

### 4. Keep secrets out of public review

- Never ask contributors to paste API keys or `.env` files into an issue.
- Use GitHub Private Vulnerability Reporting for security reports.
- Avoid `pull_request_target` for untrusted outside code unless you have a very specific reason.
- Treat screenshots and logs as sensitive until checked.

## Label strategy

Use a small label set consistently:

- `bug`
- `enhancement`
- `good first issue`
- `help wanted`
- `needs-triage`
- `dependencies`
- `docs`
- `security`

Good first issues should always include:

- clear goal
- exact files to look at
- acceptance criteria
- local verification command

That makes the issue promotable as well as approachable.

## How to promote the new contribution flow

### Show that the repo is safe and active

People are more likely to contribute when they see structure.

Add visible entry points in the README and repo sidebar:

- Contribution guide
- Code of Conduct
- Security policy
- Discussions
- Good first issue search

Also pin one Discussion that says:

`If you want to help, start with a templated issue or a good first issue. Small, tested PRs move fastest.`

### Lead with a specific invitation

Generic messages like "contributions welcome" do not pull people in.

Use narrow asks such as:

- Help improve multilingual error triage flows
- Test `prtr` on Windows terminals and report rough edges
- Pick up a `good first issue` if you want a first CLI contribution
- Suggest the next reusable `go/review/fix/design` workflow

### Package each starter issue like a mini brief

For each issue you want people to click:

- use a concrete title
- explain the user pain in 2 to 4 sentences
- include one screenshot or terminal transcript
- define done criteria
- mention the command to verify locally

This increases both trust and click-through.

### Use launch posts that mix proof with safety

Good hooks for this project:

- one command turns Korean intent plus logs into the next AI action
- same repo context, different AI app, without prompt rebuilding
- contribution flow now has templates, checks, and safer review gates

Post channels to use:

- GitHub Discussions announcement
- X post with a terminal clip or promo card
- release notes
- developer communities that like CLI workflows

## Suggested announcement copy

### Korean

`prtr 저장소의 기여 흐름을 정리했습니다. 이제 버그 리포트와 기능 제안은 템플릿으로 받고, PR은 테스트와 빌드 체크를 통과해야 하며, 보안 이슈는 비공개로 제보할 수 있습니다. 기여를 해보고 싶다면 good first issue 또는 Discussion부터 시작해 주세요.`

### English

`We tightened the contribution flow for prtr: structured issue forms, safer PR checks, private security reporting, and clearer contribution rules. If you want to help, start with a Discussion or a good first issue and send a small tested PR.`

## Attention hooks that are safe

Use these angles to attract interest without inviting low-quality spam:

- show a real workflow before asking for help
- ask for validation on one platform or one command loop
- publish 3 to 5 curated starter issues instead of saying "anything helps"
- explain the review bar up front so serious contributors self-select in

## Maintainer operating rhythm

To keep momentum after promotion:

- triage new issues at least once a week
- answer feature requests with accept, defer, or discuss
- keep `good first issue` count small and current
- close incomplete reports kindly and point back to the template
- thank first-time contributors in release notes or Discussions

## Safe defaults to keep

- outside contributions go through pull requests
- `main` stays protected
- secrets never appear in issues or PRs
- security reports stay private
- automation and execution paths get extra scrutiny

