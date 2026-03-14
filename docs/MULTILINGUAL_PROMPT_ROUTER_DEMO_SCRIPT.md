# Multilingual Prompt Router Demo Script

This script is designed for README demo GIFs or a 60 to 90 second live walkthrough.
For the README, the clearest structure is three short demos instead of one long animation:

1. setup and doctor
2. multilingual routing and history
3. copy, launch, paste, and submit

## Demo Goal

By the end of the demo, viewers should understand:

- the input can start in Korean, Japanese, or another source language
- the output prompt is shaped for a specific target such as Codex or Claude
- `--explain` makes the routing decision visible
- successful runs are searchable and reusable from history

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

Use this if you want a spoken or written narration that matches the GIF set:

`prtr` starts with guided setup, validates the environment, translates multilingual input when needed, shapes the request for the selected target and role, and then delivers that prompt through copy, launch, and optional paste workflows. Every successful run is also stored in local history, so the flow becomes reusable instead of one-off.

## README-Friendly Caption

Short caption:

`Setup once, route multilingual prompts clearly, then deliver them through copy and paste workflows.`

Long caption:

`The multilingual prompt router guides first-run setup, validates the environment with doctor, translates the user request when needed, applies target-aware prompt templates and role guidance, and supports local history plus launch and paste delivery flows.`

## Suggested Live Demo Pacing

For a live walkthrough, this pacing works well:

1. Show `setup` and `doctor` in one short pass.
2. Show one translated routing example and pause on `--explain`.
3. Show the final prompt and history line.
4. Finish with the copy and paste delivery sequence.

That structure keeps the demo short while still teaching the full workflow.

## Optional Alternate Flow

If you want a shortcut-based demo instead of a fully explicit routing command:

```bash
prtr review --no-copy "이 변경의 핵심 리스크를 찾아줘"
```

Use that version when you want to emphasize the day-to-day workflow rather than the lower-level routing knobs.
