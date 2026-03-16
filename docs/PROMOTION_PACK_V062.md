# v0.6.2 Promotion Pack

## Message Spine

- Primary line: `prtr is the command layer for AI work.`
- Support line: `Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.`
- Product framing: `The command layer for AI work.`
- Repeat loop: `go -> swap -> take -> learn`
- Setup-free CTA: `Try prtr now with \`prtr demo\` or an English \`--dry-run\` before setup.`

## Asset Map

- X card 1: `/Users/koo/dev/translateCLI-brew/images/x-card-loop-v062.png`
- X card 2: `/Users/koo/dev/translateCLI-brew/images/x-card-compare-v062.png`
- Show HN thumbnail: `/Users/koo/dev/translateCLI-brew/images/show-hn-thumb-v062.png`
- GitHub social preview: `/Users/koo/dev/translateCLI-brew/images/github-social-preview-v062.png`

## X Post Set

### Post 1

`prtr` is the command layer for AI work.

Logs in. Korean intent in. Codex first. Gemini next. Patch after.

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch
```

That is the whole product story.

GitHub: https://github.com/helloprtr/poly-prompt

Recommended image:

- `/Users/koo/dev/translateCLI-brew/images/x-card-loop-v062.png`

### Post 2

The real cost in AI work is not writing the first prompt.

It is turning logs into evidence, comparing another app, and moving from the answer to the next action without losing momentum.

`prtr` is my attempt to turn that mess into one command loop:

`go -> swap -> take -> learn`

Try it now without a key:

```bash
prtr demo
prtr go "explain this error" --dry-run
```

GitHub: https://github.com/helloprtr/poly-prompt

Recommended image:

- `/Users/koo/dev/translateCLI-brew/images/x-card-compare-v062.png`

## Show HN Set

### Title options

- `Show HN: prtr – the command layer for AI work`
- `Show HN: prtr – turn logs and intent into the next AI action`
- `Show HN: prtr – compare Claude, Codex, and Gemini without rebuilding context`

### Body

I built `prtr` because the annoying part of AI work was not writing the first prompt.

It was everything around it:

- copying logs into another app
- rebuilding the same request for Claude, Codex, or Gemini
- turning one answer into the next action
- keeping repo terms and context stable while doing all of that

`prtr` is the command layer I wanted in that gap.

The core loop is:

- `go`: build and send the first useful prompt
- `swap`: compare another AI app without rebuilding context
- `take`: turn the answer into the next action
- `learn`: save repo-local memory for future runs

The clearest demo is:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch
```

You can also try it without setup first:

```bash
prtr demo
prtr go "explain this error" --dry-run
```

GitHub: https://github.com/helloprtr/poly-prompt

Install:

```bash
brew tap helloprtr/homebrew-tap
brew install prtr
```

Recommended thumbnail:

- `/Users/koo/dev/translateCLI-brew/images/show-hn-thumb-v062.png`

### First comment

Happy to answer questions about:

- why I framed this as a command layer instead of a prompt helper
- how `go`, `swap`, `take`, and `learn` fit together
- what is already real in `v0.6.2` versus what is still just a roadmap idea
