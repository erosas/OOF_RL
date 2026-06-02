# Release Readiness

This page is a maintainer checklist for pre-tag review. Keep it factual and update it whenever a blocker or known risk is resolved.

## Current Pre-Tag Status

Status as of the 2026-06-01 pre-tag review: not tag-ready.

The open tag blocker is release artifact/plugin availability. The release workflow currently publishes `oof_rl.exe` and `oof_rl.exe.sha256`, while WASM plugins are loaded at startup from `%LOCALAPPDATA%\OOF_RL\plugins`. Before tagging, a fresh-user release must make the public plugin pages available without requiring dev-only build steps.

Do not fix this from a docs-only cleanup branch. The release/distribution decision belongs in a separate packaging/runtime branch.

## Blocker To Resolve Before Tagging

- Decide the release format: true single `.exe`, portable zip with sidecar plugins, or another explicit distribution model.
- Decide how bundled plugins are discovered or installed.
- Decide load priority if both bundled plugins and `%LOCALAPPDATA%\OOF_RL\plugins` plugins exist.
- Decide whether missing expected plugins should be log-only, visible in the UI, or both.
- Run a fresh-user smoke test that verifies the public plugin pages appear: Live, Ranks, Session, Ballchasing, and Dashboard.

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
- Release artifact/plugin availability blocker is resolved or the release format is explicitly changed and documented.
- Fresh-user app data smoke test is run against the actual release artifact, not only `go run .` or an HTML preview.
- Live, Ranks, Session, Ballchasing, Dashboard, History, and Settings can be opened from the release artifact.
- Ballchasing copy does not promise automatic upload unless auto-upload has been implemented, reviewed, and tested.
- MMR docs match current code and do not mention nonexistent config fields.
- Known risks above are either resolved with evidence or carried forward honestly in release notes.

## Out Of Scope For This Cleanup

- Release packaging or plugin-loading behavior changes.
- MMR timeout/runtime behavior changes.
- Live replay touch-counting fixes.
- Session time-source fixes.
- Forfeit heuristic changes.
- SQLite schema, saved-match data, replay capture, or app startup changes.
