# Launch Checklist

Use this checklist for `v0.5.1` and `v0.6.0` public releases.

## Release order

1. Ship `v0.5.1` for public surface sync.
2. Ship `v0.6.0` for continuity upgrades and demo-backed rollout.

## Before tagging

- Run `go test ./...`
- Run `npm run build` from `site/`
- Verify README, site, `CHANGELOG.md`, and `docs/RELEASE_MESSAGES.md` use the same release message
- Verify the GIF filenames used by the site still exist under `images/`
- Verify demo captions and alt text in `docs/DEMO_ASSETS.md`
- Confirm the correct release section has been moved out of `Unreleased`
- Confirm `README.md` points to the latest public release
- If the release includes demo changes, rerender with the commands listed in `docs/DEMO_ASSETS.md`

## Tag and release

- Create an annotated tag for the release version
- Push the release branch commit first if it is not on `origin`
- Push the tag to `origin`
- Watch `.github/workflows/release.yml`
- Confirm GitHub Release artifacts, checksums, and body text
- Confirm Homebrew tap formula update landed

### Suggested commands

- `git tag -a v0.5.1 -m "v0.5.1"`
- `git push origin v0.5.1`
- `git tag -a v0.6.0 -m "v0.6.0"`
- `git push origin v0.6.0`
- `gh run list --workflow release.yml --limit 5`
- `gh release view <tag>`

## After release

- Run a brew install or upgrade smoke test
- Confirm the site reflects the correct hero, GIFs, and release timeline
- Publish the release post using the matching entry in `docs/RELEASE_MESSAGES.md`
- Record any workflow failures or manual follow-up in the release notes thread

### Verification checklist

- `brew info prtr` shows the new stable version
- `prtr version` reports the new release
- GitHub Release includes archives for macOS, Linux, and Windows plus `checksums.txt`
- Homebrew formula under `helloprtr/homebrew-tap` points at the new version and checksums
- The site still shows the three loop demos and the latest release at the top of the timeline

## Rollback notes

- If GoReleaser fails before publishing assets, fix the workflow issue and re-push the tag
- If GitHub Release assets are incomplete, regenerate from the release workflow before announcing
- If the Homebrew tap update fails, hold the install announcement until the formula is corrected
- If the site deploy is stale, hold the public post until GitHub Pages reflects the latest hero and demo assets
