# Changelog

All notable product-facing changes to `prtr` are documented in this file.

## v0.5.0 - 2026-03-15

### Highlights

- Added `prtr exec` for headless target execution.
- Added `prtr server` as the alpha orchestration surface.
- Promoted the README and Astro site to the next-action command layer positioning.

## v0.4.5 - 2026-03-15

### Highlights

- Added `prtr platform` with optional JSON output.
- Strengthened platform surface detection across macOS, Linux, and Windows sessions.
- Reused the same platform matrix logic in both `doctor` and `platform`.

## v0.4.3 - 2026-03-15

### Highlights

- Added `prtr sync init`, `prtr sync status`, and `prtr sync`.
- Introduced repo memory loading from `.prtr/memory.toml`.
- Added deterministic mode-aware routing metadata and richer history fields.

### Why This Release Matters

`v0.4.3` gives `prtr` a shared repo-native guidance layer. Instead of relying only on ad hoc prompts, you can now maintain canonical guidance under `.prtr/` and sync it into vendor-facing files for Claude, Gemini, and Codex.

## v0.4.2 - 2026-03-15

### Highlights

- Added `prtr doctor --fix` for safe automatic repair when possible.
- Introduced a platform matrix summary for doctor output.
- Improved recovery guidance for clipboard, launcher, paste, and translation failures.

### Why This Release Matters

`v0.4.2` moves `doctor` from a pure diagnostics screen toward an actual recovery surface. It still stays honest about what can and cannot be repaired automatically, but it now helps users get unstuck faster.

### Product Value

- Better repair guidance
- Clearer platform readiness visibility
- Faster recovery from setup and delivery failures

## v0.4.1 - 2026-03-15

### Highlights

- Added `prtr start` as the beginner-first entry for onboarding and the first real send.
- Repositioned `setup` as the advanced compatibility flow for full defaults configuration.
- Hardened release automation with full git history and structured GoReleaser release notes.

### Why This Release Matters

`v0.4.1` makes the first-run path easier to explain and easier to succeed with. New users now have one front door for setup, doctor, and the first send, while advanced users can still use `setup` when they want deeper control over defaults.

### Product Value

- Clearer first-run activation
- Better separation between beginner and advanced configuration
- Stronger release packaging and changelog generation
