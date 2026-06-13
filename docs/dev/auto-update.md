# Update Checker

The update checker is host-core (`internal/update`), not a plugin: it reports
on the host binary itself and its routes are host-reserved.

The app **never downloads release artifacts**. The release manifest is
unsigned, so an in-app download verified against a SHA256 from that same
manifest would prove transport integrity only, not who built the binary. The
app only checks versions; the user downloads the release zip in their browser
and updates with the `install.ps1` shipped inside it.

## Behavior

- Checks fetch
  `https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json`
  and compare its version against the running version (`config.AppVersion`,
  set by release builds via
  `-ldflags "-X OOF_RL/internal/config.AppVersion=v1.2.3"`; dev builds report
  `dev`).
- Checks run ~15s after startup, every 24h while running
  (`Checker.RunPeriodic`, started in `main.go`), and on demand from the
  Settings page's **Check for updates** button.
- When a newer version is found, the UI shows an **Update available** dialog
  with links to the GitHub release page (`notes_url`) and the zip
  (`artifact_url`). Dismissing it stores the version in `localStorage`
  (`oof-upd-dismissed`), so the dialog reappears only for the next release; a
  manual check clears the dismissal.
- Nothing is downloaded, installed, extracted, replaced, or restarted by the
  app.

## Link allowlist

Because the manifest is unsigned, a tampered manifest could put an arbitrary
link in a dialog the user is primed to click. `SafeReleaseURL` only lets
`notes_url`/`artifact_url` through to the UI when they start with
`https://github.com/erosas/OOF_RL/` (and contain no `..`/`\`/`@` after the
prefix); anything else is dropped from the status.

## Routes

| Route | Method | Behavior |
|---|---|---|
| `/api/update/status` | GET | Current `update.Status` snapshot (poll target) |
| `/api/update/check` | POST | Fetch + validate manifest; 502 with status body on upstream failure |

The `update` plugin ID and `/api/update/` route namespace are host-reserved;
WASM plugins cannot claim them.

## Manifest Shape

```json
{
  "version": "v1.2.3",
  "channel": "stable",
  "notes_url": "https://github.com/erosas/OOF_RL/releases/tag/v1.2.3",
  "published_at": "2026-06-06T12:00:00Z",
  "artifact_url": "https://github.com/erosas/OOF_RL/releases/download/v1.2.3/OOF_RL-v1.2.3.zip",
  "artifact_name": "OOF_RL-v1.2.3.zip",
  "artifact_sha256": "..."
}
```

Only `version` is required (the checker compares versions and links out; it
never fetches artifacts). `artifact_sha256` is still published so users can
verify a browser download by hand.

`scripts/package-release.ps1 -Version vX.Y.Z` generates
`dist/update-manifest.json` alongside the zip, and the release job in
`.github/workflows/ci.yml` attaches it to the GitHub release — that is what
makes the `releases/latest/download/update-manifest.json` URL resolve.

## install.ps1

`scripts/install.ps1` ships in the root of the release zip
(`package-release.ps1` copies it in). Run from the extracted folder it:

1. Stops a running `oof_rl.exe` (and targets that copy's folder for the
   update, unless `-InstallDir` is given; default
   `%LOCALAPPDATA%\Programs\OOF_RL`).
2. Copies the extracted files over the install folder and clears the
   Mark-of-the-Web (`Unblock-File`).
3. Creates a Start Menu shortcut (skip with `-NoShortcut`) and relaunches the
   app (skip with `-NoLaunch`).

User data (database, logs, settings) lives in `%LOCALAPPDATA%\OOF_RL` and is
never touched.
