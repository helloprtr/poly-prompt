# Multilingual Prompt Router Demo Script

This script is designed for the single proof loop that should anchor the README, GitHub social preview, release post, and live walkthroughs.

Use one short demo instead of three unrelated animations.

## Demo Goal

By the end of the demo, viewers should understand:

- the input can start in Korean
- logs become evidence
- the same request can be compared across apps
- the answer can become the next action

## Environment Assumptions

Recommended setup for the recording:

- terminal width around 100 to 110 columns
- large monospace font
- dark terminal theme with good contrast
- `prtr` already installed
- DeepL key already configured for a real translation recording

If you are creating a repository-safe GIF without bundling credentials, use a deterministic replay of the translated text and keep the command surface identical to the real CLI flow.

## Recommended README GIF Set

### GIF 1: Setup and doctor

Command focus:

```bash
prtr setup
prtr doctor
```

Viewer takeaway:

- setup can keep the API key configured without exposing it on screen
- `doctor` verifies whether translation and delivery dependencies are ready
- paste support is permission-gated and easy to explain visually

What to pause on:

- `DeepL API key [configured]`
- `OK deepl api key`
- `OK translation`
- automation permission guidance for `--paste`

### GIF 2: Routing and history

Command:

```bash
prtr --no-copy --explain -t codex -r be "도커 컨테이너 실행법을 초보자용 단계로 정리해줘"
prtr history
```

Keep the output visible long enough to read the routing block and final prompt:

```text
// Target: codex
Expert Backend Engineer & Tech Lead.
Focus on implementation details, concrete file changes, tests, and safe rollout steps.

Please explain how to run Docker containers in beginner-friendly steps.
```

Viewer takeaway:

- the router is translating and shaping the prompt
- target, role, and template resolution are inspectable
- the original request is stored in history for reuse

Pause on:

- `language route: auto -> en`
- `translation decision: translated`
- the final prompt block
- the history line

### GIF 3: Copy, launch, paste, and submit

Command:

```bash
prtr --paste --submit confirm -t codex "이 변경의 핵심 리스크를 요약해줘"
```

Viewer takeaway:

- clipboard copy happens automatically
- the target CLI session opens
- paste and confirm-submit are the final delivery stage

Pause on:

- `copied prompt for target "codex" to clipboard`
- `opened codex CLI session`
- `pasted prompt into codex terminal session`
- `Submit pasted prompt to codex now? [y/N]: y`
- `submitted prompt to codex`

## Suggested Voiceover Or Caption

Use this if you want a spoken or written narration that matches the proof-loop assets:

`prtr` turns logs, diffs, and intent into the next AI action. It keeps the loop moving by turning one request into a target-aware prompt, making app comparison cheap, and carrying the answer into the next action without rebuilding context.

## README-Friendly Caption

Short caption:

`Turn logs and intent into the next AI action, then keep the loop moving.`

Long caption:

`prtr is the command layer for AI work. It turns logs, diffs, and intent into the next AI action, keeps app switching cheap, and makes the answer easier to hand off into the next step.`

## Suggested Live Demo Pacing

For a live walkthrough, this pacing works well:

1. Show the failing test output.
2. Run `go fix` with a Korean intent.
3. Swap to another app.
4. Take the answer into `patch`.

That structure keeps the demo short while still teaching the whole product story.

## Recommended proof loop

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch
```

Pause on:

- the Korean request
- the evidence block
- the target app change
- the next-action handoff

Keep `setup` and `doctor` out of the hero demo. They belong in docs and onboarding, not in the first public proof.
