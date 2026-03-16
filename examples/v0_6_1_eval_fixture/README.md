# prtr v0.6.1 Eval Fixture

This fixture exists to test the `prtr` loop with a small, realistic workflow.

Scenario:

- a checkout utility has a failing test
- the repo has lightweight product and architecture context
- the tester can run `go`, `swap`, `take`, `again`, `learn`, and `inspect`

The intended bug:

- fixed coupons are being applied once per item entry instead of once per order

Suggested flow:

```bash
npm test 2>&1 | prtr go fix "왜 깨지는지 정확한 원인만 찾아줘" --dry-run
prtr swap gemini --dry-run
cat fixtures/sample_ai_answer.md | pbcopy
prtr take patch --dry-run
prtr again --edit
prtr learn README.md docs
prtr inspect --json "show the route"
```

What good looks like:

- `go` identifies the fixed-coupon bug quickly
- `swap` reuses the same intent without rewriting
- `take patch` converts the clipboard answer into an implementation prompt
- `learn` stores checkout-specific terms and guidance
- `inspect` explains the compiled route in structured form
