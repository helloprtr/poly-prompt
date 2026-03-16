# Contributing to prtr

Thanks for your interest in improving `prtr`.

This project accepts outside contributions, but we keep the contribution flow intentionally structured so bugs are reproducible, reviews stay safe, and maintainers can move quickly.

## Start in the right place

- Bug reports: use the Bug Report issue form and include version, platform, command, and reproduction steps.
- Feature requests: use the Feature Request form when you can describe the workflow pain and desired outcome clearly.
- Product ideas or early discussion: use GitHub Discussions first.
- Security problems: do not open a public issue. Follow [SECURITY.md](SECURITY.md) and use private vulnerability reporting.

## Before you open a PR

- Open or link an issue first for any non-trivial change.
- Keep changes focused. Small, reviewable PRs are much easier to merge safely.
- Avoid bundling refactors with bug fixes unless the refactor is required.
- If you add a dependency, explain why an existing standard library or current package is not enough.

## Local checks

Run the relevant checks before pushing:

```bash
gofmt -w $(git ls-files '*.go')
go test ./...
cd site && npm run build
```

The site build is required when you change docs, promotion pages, or the Astro site.

## PR expectations

Every pull request should:

- explain the user-facing change in plain language
- link the related issue
- include tests or a clear manual verification note
- mention any network, clipboard, shell execution, filesystem, or automation behavior changes
- keep secrets, `.env` files, private logs, and local machine details out of the diff

If your change affects security-sensitive paths, expect a slower review:

- command execution
- app launch and paste automation
- clipboard handling
- filesystem writes
- release workflows
- GitHub Actions

## Review and merge standards

The `main` branch should be protected in GitHub settings with:

- pull requests required before merge
- status checks required before merge
- at least one maintainer approval
- dismissal of stale approvals after new commits

Repo files can define templates and CI, but branch protection itself must still be enabled manually in GitHub.

## Labels used for triage

- `bug`: confirmed or likely broken behavior
- `enhancement`: new capability or improvement
- `good first issue`: intentionally scoped for a first contribution
- `help wanted`: maintainers are open to outside implementation help
- `needs-triage`: newly opened issue that still needs maintainer review
- `dependencies`: dependency or toolchain update
- `security`: private or maintainer-handled security work

## Community expectations

By participating, you agree to follow [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md).

If you are unsure whether something is ready, open a Discussion or draft PR. Clear context is always more helpful than a larger patch with missing rationale.

