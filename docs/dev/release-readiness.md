# Release Readiness

This page is a maintainer checklist for pre-tag review. Keep it factual and update it whenever a blocker or known risk is resolved.

## Current Pre-Tag Status

Status as of the 2026-06-02 packaging branch: release artifact/plugin availability has a candidate fix and still needs artifact inspection plus fresh-user smoke testing before tagging.

The release workflow should publish one normal-user zip with `oof_rl.exe`, bundled public WASM plugins, plugin assets, and a checksum for the zip. On startup, the app seeds known bundled public plugins from the extracted release folder into `%LOCALAPPDATA%\OOF_RL\plugins`, then uses the existing plugin loader. Bundled public plugin files win for `live`, `ranks`, `session`, and `dashboard`; unknown/custom app-data plugins are preserved. Ballchasing is not bundled or auto-seeded by the public release zip, but manually installed/developer-built Ballchasing plugins can still load from app data. Missing or unreadable bundled plugin files remain log-only in this branch.

## Blocker To Resolve Before Tagging

- Inspect the release zip and verify it contains `OOF_RL/oof_rl.exe`, `OOF_RL/plugins/*.wasm`, plugin asset folders, and `OOF_RL/README.txt`.
- Verify the checksum file is for the release zip users download.
- Verify bundled plugins seed into `%LOCALAPPDATA%\OOF_RL\plugins` on first launch.
- Verify bundled plugins update existing public plugin files when the release zip contains newer/different files.
- Verify unknown/custom app-data plugin files and directories are preserved.
- Stale files inside known public plugin asset directories are not deleted by this branch.
- Run a fresh-user smoke test that verifies the bundled public plugin pages appear: Live, Ranks, Session, and Dashboard.

## Known Risks To Track

These risks are not fixed by the pre-tag cleanup branch. Do not present them as resolved in release notes until a specific fix lands and is tested.

| Risk | Current concern | Required follow-up |
|---|---|---|
| Replay touch-counting | Live replay mode can still display touch totals from replay-time events. Confirm whether replay-mode touches can pollute saved History or Session stats in normal release use. | Separate data/stat collection fix or severity decision. |
| Session time source | Session boundaries use wall-clock time in the Session plugin. WASM/app time behavior was discussed as a separate concern. | Separate Session/core review; do not hide it with UI changes. |
| Forfeit heuristic | Saved match forfeits are inferred from match clock state and may have edge cases. | Separate match classification review with replay/playtest evidence. |
| MMR timeout hardening | MMR fallback retries and provider delays may outlive the intended request window. | Decide whether context-aware timeout hardening is required before tag or can wait. |

## Pre-Tag Checklist

- CI is green on `main`.
- Root, SDK, and nested plugin module tests pass.
- `git diff --check` is clean before tagging.
- Release artifact contains one normal-user zip with `oof_rl.exe`, public plugin WASM files, plugin assets, and package notes.
- Release checksum corresponds to the zip users download.
- Bundled public plugins seed into app data before plugin loading.
- Fresh-user app data smoke test is run against the actual release artifact, not only `go run .` or an HTML preview.
- Live, Ranks, Session, Dashboard, History, and Settings can be opened from the release artifact.
- Ballchasing copy does not promise automatic upload unless auto-upload has been implemented, reviewed, and tested.
- MMR docs match current code and do not mention nonexistent config fields.
- Known risks above are either resolved with evidence or carried forward honestly in release notes.

## Out Of Scope For This Packaging Branch

- MMR timeout/runtime behavior changes.
- Live replay touch-counting fixes.
- Session time-source fixes.
- Forfeit heuristic changes.
- SQLite schema, saved-match data, replay capture, or broad app startup changes unrelated to bundled plugin seeding.
- Visible UI warnings for missing bundled plugin files.
