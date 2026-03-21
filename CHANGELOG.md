# Changelog

All notable product-facing changes to `prtr` are documented in this file.

## [1.0.0] — 2026-03-22

### Added
- Session as first-class citizen: `prtr` auto-creates and maintains work sessions per git repo
- `prtr review [files]`, `prtr edit [files]`, `prtr fix [desc]`, `prtr design [topic]` — mode-specific session starters
- `prtr @gemini`, `prtr @codex` — handoff current session to another model
- `prtr checkpoint "note"` — save progress memo for better handoff quality
- `prtr done` — mark session complete
- `prtr sessions` — list all sessions (active and completed)
- `prtr status` now shows current session state alongside Work Capsule drift info
- `prtr doctor` now checks AI binary availability for claude, gemini, codex

### Changed
- `prtr` (bare) shows active session and offers to continue or start new
- `prtr @model` requires an active session; exits with clear error if none

### Deprecated (hidden, still functional)
- `prtr go`, `swap`, `take`, `again`, `start`, `learn`, `resume` — use session commands instead

## v0.7.0 - 2026-03-17

### Highlights

- Introduced `prtr take --deep` (alias `--dip`): a five-worker AI pipeline that turns clipboard answers into structured, evidence-backed execution runs.
- Added `--llm=claude|gemini|codex` flag to generate provider-optimized delivery prompts (Claude XML tags, Gemini Markdown headers, Codex numbered instructions).
- Expanded deep actions: `take patch`, `take test`, `take debug`, `take refactor` all run the full pipeline.
- Source-driven risk detection and test generation: workers analyze the source text and produce targeted risks and test cases instead of boilerplate.
- New `llm_provider` config field and `PRTR_LLM_PROVIDER` env var for persistent provider selection.
- Full documentation rewrite: English and Korean user manuals, command reference, updated README and site.

### Why This Release Matters

`v0.7.0` moves `prtr take` from a simple clipboard-to-prompt converter into a typed execution layer. The five-worker pipeline (planner → patcher → critic → tester → reconciler) produces a plan, a patch bundle, a risk report, and a test plan as local artifacts every time `--deep` runs. The `--llm` flag then formats the final delivery prompt to match how each AI tool processes instructions best — XML for Claude, structured Markdown for Gemini, numbered lists for Codex — so the handoff from prtr to the AI is as clear as possible regardless of which tool you use.

### What Changed

**Deep execution engine (`take --deep` / `--dip`)**
- Five-worker pipeline: planner, patcher, critic, tester, reconciler
- Artifacts written to `.prtr/runs/<id>/`: `manifest.json`, `plan.json`, `events.jsonl`, `result/patch_bundle.json`, `result/patch.diff`, `result/tests.md`, `result/summary.md`
- `take test`, `take debug`, `take refactor` are valid deep actions alongside `take patch`
- `planSummaryFor` and `resultTypeFor` return action-specific plan summaries and bundle types

**Provider-aware delivery prompts (`--llm`)**
- `--llm=claude` → XML semantic tags: `<role>`, `<context>`, `<task>`, `<validation>`
- `--llm=gemini` → `##` Markdown section headers
- `--llm=codex` → numbered instructions with ` ```diff ``` ` code blocks
- No `--llm` → improved universal Markdown (works without any API key)
- Config: `llm_provider = "claude"` in `~/.config/prtr/config.toml`
- Env: `PRTR_LLM_PROVIDER=claude`

**Source-driven analysis**
- `detectSourceRisks`: maps source keywords to targeted risk items (auth, migration, API contract, concurrency, destructive ops, config/secret exposure, cache invalidation)
- `buildTestCases`, `buildEdgeCases`, `buildVerificationSteps`: generate test cases based on source keywords (nil/panic, error paths, timeouts, auth, loops, race conditions)

**Documentation**
- `README.md`: full rewrite around the command-layer story
- `README.ko.md`: new Korean README
- `docs/guide.md`: complete English user manual (installation → daily use → deep mode → config)
- `docs/guide.ko.md`: complete Korean user manual (한국어 전체 사용 설명서)
- `docs/reference.md`: command and flag reference tables
- GitHub Pages: `--deep` loop card, provider-aware routing section, Korean docs hub (`/docs-ko/`)

### Product Value

- Faster, more structured handoff to AI tools
- Consistent prompt quality regardless of which AI tool receives it
- Local artifact trail for every deep run (plan, diff, risks, tests)
- Zero API key required to use the deep pipeline; `--llm` is an optional enhancement

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

The format is intentionally written so each section can be reused as a GitHub Release body with minimal editing.

## v0.4.0 - 2026-03-14

### Launch Copy

Think in your language. Move the whole loop forward.

The multilingual prompt router for repeatable AI work.

From multilingual intent to routed prompt to next action.

### Highlights

- Adopted a Cobra-based command tree for the public CLI surface.
- Clarified the primary workflow around `go`, `again`, `swap`, `take`, `learn`, and `inspect`.
- Improved the long-term maintainability of the command surface and future feature expansion.

### Why This Release Matters

`v0.4.0` strengthens the foundation behind the same product promise: a multilingual prompt router for repeatable AI work. The CLI now has a clearer command structure, which improves help output, keeps command behavior more consistent, and makes future releases easier to grow without adding surface confusion.

### Product Value

- Better command discoverability for new users
- Cleaner internal structure for future workflow expansion
- A more stable base for release and documentation quality going forward

## v0.3.6 - 2026-03-14

### Highlights

- Polished docs and product-facing copy across the main user guides.
- Improved how the multilingual prompt router story is explained.
- Tightened the onboarding narrative for installation, setup, and daily use.

### Why This Release Matters

This release improves how `prtr` is understood. The product already had a strong workflow, but `v0.3.6` made the value proposition clearer for new users evaluating the tool through the README, install guide, and usage docs.

### Product Value

- Easier onboarding
- Clearer messaging for demos and promotion
- Better explanation of the repeat-loop workflow

## v0.3.5 - 2026-03-14

### Highlights

- Release tag aligned with the `learn` and termbook workflow work.

### Notes

This tag currently points to the same commit as `v0.3.4`. For practical release communication, it is best treated as a release-management checkpoint rather than a separate feature release.

## v0.3.4 - 2026-03-14

### Highlights

- Added `prtr learn` to build a repo-local termbook.
- Extracts project-specific terms from README files, docs, and code identifiers.
- Stores protected translation terms in `.prtr/termbook.toml`.
- Preserves project names, flags, identifiers, and domain language during translation.

### Why This Release Matters

One of the biggest failure modes in AI-assisted workflows is losing project-specific vocabulary during translation. `v0.3.4` solves that by teaching `prtr` which terms must survive translation unchanged, making multilingual prompting much safer for real repositories.

### Product Value

- More reliable translation for real-world codebases
- Better preservation of project identity and naming
- Less manual correction after prompt generation

## v0.3.3 - 2026-03-14

### Highlights

- Added lightweight repo-aware context to `prtr go`.
- Automatically surfaces repo name, current branch, and changed file summaries when available.
- Reduced the need to manually restate project state in every prompt.

### Why This Release Matters

`prtr` is more useful when it understands where the request is happening. `v0.3.3` brings repository awareness into the prompt-building path so the generated prompt can better reflect the real working context instead of only the user's raw message.

### Product Value

- Better prompt quality with less manual context writing
- More relevant output for active repository work
- Stronger alignment between request and current project state

## v0.3.2 - 2026-03-14

### Highlights

- Added the `take` loop for next-action prompting.
- Turns copied AI output into a new prompt for actions such as `patch`, `test`, `commit`, and `summary`.
- Extends `prtr` from first-send routing into iterative workflow support.

### Why This Release Matters

The first prompt is only part of the workflow. `v0.3.2` makes follow-up actions dramatically faster by converting a model response into the next useful request. This is one of the releases that most clearly shifts `prtr` from a prompt utility into a repeatable AI work loop.

### Product Value

- Faster iteration after the first answer
- Less manual rewriting between steps
- Stronger support for real coding and review loops

## v0.3.1 - 2026-03-14

### Highlights

- Stabilized the delivery surface around the `go` workflow.
- Expanded launcher and paste automation support across platforms.
- Added translation policy behavior and stronger diagnostics.
- Improved the public framing of `prtr` as a multilingual prompt router.

### Why This Release Matters

This release turned the product from a prompt-generation tool into a more complete delivery workflow. Instead of only translating and formatting prompts, `prtr` now more clearly supports the end-to-end path from request to app launch, paste, and local history.

### Product Value

- Stronger cross-platform usability
- Better operational confidence through diagnostics
- A more complete send flow from prompt creation to delivery

## v0.3.0 - 2026-03-13

### Highlights

- Added guided `setup` for first-run configuration.
- Added template presets, reusable profiles, and local history workflows.
- Added rerun-oriented flows for repeated prompt reuse.
- Moved the product toward a repeatable personal workflow rather than one-off prompting.

### Why This Release Matters

`v0.3.0` is a major workflow release. It made `prtr` much more practical for repeated daily use by letting people save defaults, organize reusable prompt behavior, and revisit previous runs without rebuilding everything from scratch.

### Product Value

- Faster onboarding
- More reusable prompt behavior
- Local prompt memory for recurring tasks

## v0.2.2 - 2026-03-13

### Highlights

- Simplified target templates.
- Redesigned role prompts for more consistent prompt shaping.
- Reduced internal complexity while improving output predictability.

### Why This Release Matters

This release refined the structure behind prompt generation. The improvements are less about adding visible commands and more about making the resulting prompt system easier to reason about and easier to trust.

### Product Value

- More consistent prompt output
- Cleaner template behavior
- Stronger base for later workflow releases

## v0.2.1 - 2026-03-13

### Highlights

- Refined role templates.
- Added a dedicated installation guide.
- Improved the first-run experience for new users.

### Why This Release Matters

`v0.2.1` improved two areas that strongly shape adoption: better prompt-role guidance and clearer install documentation. That combination makes the tool easier to understand and easier to start using successfully.

### Product Value

- Better role-driven prompt quality
- Smoother setup experience
- Lower adoption friction

## v0.2.0 - 2026-03-13

### Highlights

- Added interactive prompt editing before delivery.
- Added stronger cross-platform release support.
- Improved clipboard behavior across supported operating systems.

### Why This Release Matters

This release introduced the important idea that users may want an automatic prompt pipeline without losing final human control. Interactive editing made `prtr` more practical for people who want to review, refine, and approve the final prompt before sending it onward.

### Product Value

- Human-in-the-loop prompt review
- Better platform reach
- Safer final prompt delivery

## v0.1.0 - 2026-03-13

### Highlights

- Rebranded the CLI to `prtr`.
- Established the product identity and naming foundation for future releases.

### Why This Release Matters

`v0.1.0` gave the project a tighter and more memorable public identity. While small in surface area, it created the naming consistency that later releases build on.

### Product Value

- Stronger product identity
- Simpler CLI naming
- Better long-term brand recognition
