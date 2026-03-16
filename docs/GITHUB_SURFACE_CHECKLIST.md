# GitHub Surface Checklist

These items are not fully controlled by repo files, so keep this checklist next to the code.

Status on 2026-03-16:

- About updated
- topics updated
- Discussions enabled
- social preview uploaded

## About

Use:

`The command layer for AI work. Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.`

CLI:

```bash
gh api repos/helloprtr/poly-prompt --method PATCH \
  --field description='The command layer for AI work. Turn logs, diffs, and intent into the next AI action across Claude, Codex, and Gemini.' \
  --field homepage='https://helloprtr.github.io/poly-prompt/'
```

## Topics

Use this topic set as the default:

- `ai-workflow`
- `developer-tools`
- `terminal`
- `claude`
- `codex`
- `gemini`
- `agent-workflow`

CLI:

```bash
gh api repos/helloprtr/poly-prompt/topics --method PUT \
  -H 'Accept: application/vnd.github+json' \
  --input - <<'JSON'
{"names":["ai-workflow","developer-tools","terminal","claude","codex","gemini","agent-workflow"]}
JSON
```

## Social preview

Use the same proof-loop asset and wording as the README hero.

Preferred message:

`prtr is the command layer for AI work.`

Uploaded asset:

- `/Users/koo/dev/translateCLI-brew/images/github-social-preview-v062.png`

## Discussions

Enable Discussions, then post the seeds from:

- `docs/DISCUSSION_SEEDS.md`

Pin both threads after publishing them.

CLI:

```bash
gh api graphql -f query='mutation($id:ID!){updateRepository(input:{repositoryId:$id,hasDiscussionsEnabled:true}){repository{hasDiscussionsEnabled}}}' -f id='R_kgDORlaCjg'
```
