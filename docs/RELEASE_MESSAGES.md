# Release Message Templates

Use the same structure for every launch-facing release note:

1. One-line value proposition
2. One GIF or preview image
3. One copy-paste command

## Release 1: demo + setup-free fast path

Value line:

`prtr is the command layer for AI work. Try the loop now without a key.`

Copy-paste command:

```bash
prtr demo
prtr go "explain this error" --dry-run
```

GIF or image:

- use the same proof-loop asset as the README hero

Body:

`prtr` now has a setup-free path.

Start with `prtr demo`, then try an English `--dry-run` before you ever touch `setup`.

When you are ready for multilingual routing, add DeepL later with `prtr setup`.

## Release 2: go -> swap -> take proof loop

Value line:

`Start in Korean, feed logs as evidence, compare another app, and hand the answer to the next action.`

Copy-paste command:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘"
prtr swap gemini
prtr take patch
```

GIF or image:

- use the same proof-loop asset as the README hero

Body:

The public story for `prtr` is now one loop:

- `go` turns intent and evidence into the first useful prompt
- `swap` compares another app without rebuilding context
- `take` turns the answer into the next action

## Release checklist

- hero copy matches README and site
- value line is one sentence
- command block is copy-paste ready
- only one asset is highlighted
- release title and social post use the same wording
