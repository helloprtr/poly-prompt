# prtr v0.6.1 Fixture Checklist

Use this checklist with the evaluation fixture under `examples/v0_6_1_eval_fixture`.

Recommended setup:

```bash
cd /Users/koo/dev/translateCLI-brew
./scripts/eval/setup_v0_6_1_fixture.sh
cd /tmp/prtr-v0.6.1-eval
```

Recommended command flow:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘" --dry-run
prtr swap gemini --dry-run
cat fixtures/sample_ai_answer.md | pbcopy
prtr take patch --dry-run
prtr again --edit
prtr learn README.md docs
prtr inspect --json "show the route"
```

## 1. Test session info

- [ ] Tester name recorded
- [ ] Test date recorded
- [ ] OS and shell recorded
- [ ] `prtr version` recorded
- [ ] Testing installed binary or local build noted

Session notes:

```text
Tester:
Date:
OS:
Shell:
prtr version:
Binary source:
```

## 2. Fixture setup

- [ ] `./scripts/eval/setup_v0_6_1_fixture.sh` runs successfully
- [ ] fixture directory exists at `/tmp/prtr-v0.6.1-eval`
- [ ] Git repo is initialized in the fixture
- [ ] working tree is intentionally dirty
- [ ] `README.md`, `docs/architecture.md`, and `fixtures/sample_ai_answer.md` exist

Observed result:

```text
```

## 3. Baseline failure is real

Run:

```bash
cd /tmp/prtr-v0.6.1-eval
npm test
```

- [ ] test run exits with a failure
- [ ] one test passes and one test fails
- [ ] failure clearly points to fixed coupon behavior
- [ ] failure output is readable enough to be useful as `go fix` evidence

Observed result:

```text
```

## 4. `go fix` quality

Run:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘" --dry-run
```

- [ ] command succeeds
- [ ] the Korean request is accepted naturally
- [ ] piped test output is reflected in the preview
- [ ] repo context appears in the preview or route
- [ ] chosen mode is clearly `fix`
- [ ] chosen app feels reasonable for `fix`
- [ ] the prompt aims at root cause, not generic advice
- [ ] next-step suggestions are visible after `go`
- [ ] suggested next steps fit `fix` mode

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 5. `swap gemini` continuity

Run:

```bash
prtr swap gemini --dry-run
```

- [ ] command succeeds
- [ ] original intent is preserved
- [ ] mode is preserved from the previous `go`
- [ ] only the destination app changes
- [ ] no manual rewrite is needed
- [ ] comparison feels faster than starting over

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 6. `take patch` answer-to-action quality

Run:

```bash
cat fixtures/sample_ai_answer.md | pbcopy
prtr take patch --dry-run
```

- [ ] command succeeds
- [ ] clipboard content is used as source material
- [ ] output is clearly implementation-oriented
- [ ] output names the likely file or file area
- [ ] output preserves the checkout domain language
- [ ] output is meaningfully different from a generic summary
- [ ] output feels ready to send, not just informative

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 7. `again --edit` iteration cost

Run:

```bash
prtr again --edit
```

- [ ] command succeeds
- [ ] previous flow is reused
- [ ] edit step opens or previews correctly
- [ ] iterating feels cheaper than composing a new prompt
- [ ] context from the prior run is still intact

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 8. `learn` repo memory quality

Run:

```bash
prtr learn README.md docs
ls -l .prtr/termbook.toml .prtr/memory.toml
```

- [ ] command succeeds
- [ ] `.prtr/termbook.toml` is created
- [ ] `.prtr/memory.toml` is created
- [ ] output includes protected term count
- [ ] output includes repo summary
- [ ] output includes guidance lines
- [ ] saved guidance matches the checkout fixture
- [ ] saved names feel stable and useful for future runs

Inspect files if needed:

```bash
sed -n '1,200p' .prtr/termbook.toml
sed -n '1,240p' .prtr/memory.toml
```

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 9. `inspect --json` route visibility

Run:

```bash
prtr inspect --json "show the route"
```

- [ ] command succeeds
- [ ] JSON output is valid-looking and structured
- [ ] route explanation is understandable
- [ ] selected app/mode/template are visible
- [ ] route output is useful for debugging or trust
- [ ] this output explains why the loop behaved the way it did

Rate the result:

- [ ] 5 = excellent
- [ ] 4 = good
- [ ] 3 = usable
- [ ] 2 = weak
- [ ] 1 = poor

Why:

```text
```

## 10. End-to-end product judgment

Answer each item honestly:

- [ ] `go` reduced the time to first useful action
- [ ] `swap` made model comparison cheap enough to be normal
- [ ] `take` made the answer feel actionable
- [ ] `again` made iteration cheaper than restart
- [ ] `learn` made the repo feel remembered
- [ ] `inspect` made the system easier to trust
- [ ] the full loop felt coherent
- [ ] docs/help/behavior matched each other

Overall scores:

- Speed: `__ / 5`
- Clarity: `__ / 5`
- Continuity: `__ / 5`
- Trust: `__ / 5`
- Overall product value: `__ / 5`

## 11. Final verdict

- [ ] Ship-ready
- [ ] Strong but needs polish
- [ ] Usable but confusing
- [ ] Not ready for wider testing

Top strengths:

```text
1.
2.
3.
```

Top weaknesses:

```text
1.
2.
3.
```

Most convincing moment in the loop:

```text
```

Most disappointing moment in the loop:

```text
```

Would you keep using `prtr` after this test?

- [ ] Yes
- [ ] Maybe
- [ ] No

Why:

```text
```
