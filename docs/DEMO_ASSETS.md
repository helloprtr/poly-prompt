# Demo Assets

This document is the source of truth for the three launch GIFs used in the README and site.

## 1. First send

- Output file: `images/prtr-setup-doctor.gif`
- Tape: `scripts/demos/prtr-go-first-send.tape`
- Render command: `HOST=127.0.0.1 vhs scripts/demos/prtr-go-first-send.tape`
- Caption: `Write in Korean. Route to Codex. See the next action immediately.`
- Alt text: `Animated terminal demo showing prtr go turning a Korean fix request into a dry-run Codex prompt with repo context.`

## 2. App comparison

- Output file: `images/prtr-routing-history.gif`
- Tape: `scripts/demos/prtr-swap-compare.tape`
- Render command: `HOST=127.0.0.1 vhs scripts/demos/prtr-swap-compare.tape`
- Caption: `Keep the same intent, then swap apps without rebuilding context.`
- Alt text: `Animated terminal demo showing prtr go review followed by prtr swap claude to compare another app with the same request.`

## 3. Answer to action

- Output file: `images/prtr-delivery-paste.gif`
- Tape: `scripts/demos/prtr-take-learn.tape`
- Render command: `HOST=127.0.0.1 vhs scripts/demos/prtr-take-learn.tape`
- Caption: `Turn the answer into a patch prompt, then save repo memory for the next run.`
- Alt text: `Animated terminal demo showing prtr take patch from clipboard output and prtr learn saving repo memory.`
