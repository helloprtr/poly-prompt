# Patch Bundle

## Summary
Root cause: repo context is not actually empty; it only looks sparse because the collector currently stores git status lines only. If you want richer context, update internal/repoctx/repoctx.go to include a short git ...

## Touched Files
- internal/repoctx/repoctx.go
- internal/repoctx/repoctx_test.go

## Risks
- [HIGH] Behavior Drift: The follow-up may change behavior beyond the user's requested fix if the source answer mixed diagnosis and implementation.
- [MEDIUM] Local Context Risk: Review the existing logic in internal/repoctx/repoctx.go before applying broad changes.

## Test Plan
- Add or update one regression test that covers the primary failure path described in the source material.
- Verify the happy path still passes after the change.
- Add a test that cancels context early and verifies clean exit.
- Add a regression test in internal/repoctx/repoctx_test.go that covers the primary failure path.
