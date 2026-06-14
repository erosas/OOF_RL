# Update Checker

The update checker is host-core (`internal/update`), not a plugin: it reports
on the host binary itself and its routes are host-reserved.

The app **never downloads release artifacts**. The release manifest is
unsigned, so an in-app download verified against a SHA256 from that same
manifest would prove transport integrity only, not who built the binary. The
app only checks versions; the user downloads the release zip in their browser
and updates with the `install.ps1` shipped inside it.

## Behavior

- Checks fetch a manifest and compare its version against the running version
  (`config.AppVersion`, set by release builds via
  `-ldflags "-X OOF_RL/internal/config.AppVersion=v1.2.3"`; local dev builds
  report `dev`). Which manifest is fetched depends on the channel — see
  [Release channels](#release-channels) below.
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

## Release channels

Two channels form a hierarchy, not parallel tracks: **regular users see the
newest stable release; dev-mode users see the newest build of any kind (dev or
stable).** This is driven by two manifests plus GitHub's prerelease semantics —
`releases/latest` never returns a prerelease.

| Channel | Manifest URL | Refreshed on | Read by |
|---|---|---|---|
| stable | `releases/latest/download/update-manifest.json` | stable releases only | regular users |
| dev | `releases/download/dev/update-manifest.json` | every release (dev **and** stable) | dev-mode users |

- **Stable releases** publish a normal GitHub release (tag `vX.Y.Z`), which
  becomes `releases/latest`.
- **Dev releases** publish a GitHub *prerelease* (semver prerelease tag, e.g.
  `vX.Y.Z-dev.1`), so they stay out of `releases/latest` and regular users
  never see them.
- The **dev channel is a single rolling `dev` prerelease** whose
  `update-manifest.json` asset is overwritten on every release. Because a stable
  release refreshes it too, dev-mode users are offered stable builds as well —
  hence "newest of either kind". The manifest's download links resolve to the
  versioned release; the `dev` release only carries the pointer.
- The client (`update.Checker`) picks the URL from the live `dev_mode` setting
  on every check (`main.go` passes `func() bool { return srv.Config().DevMode }`),
  so toggling dev mode in Settings switches channels without a restart.
- `IsNewer` uses semver-2 precedence so a prerelease ranks below its release
  (`vX.Y.Z-dev.1 < vX.Y.Z`) and later dev builds rank above earlier ones — a dev
  user never gets a phantom downgrade prompt, and rolls forward onto stable when
  it ships.

## Link allowlist

Because the manifest is unsigned, a tampered manifest could put an arbitrary
link in a dialog the user is primed to click. `SafeReleaseURL` parses each of
`notes_url`/`artifact_url` and only lets it through to the UI when the scheme
is `https`, the host is exactly `github.com`, there is no userinfo, and the
path starts with `/erosas/OOF_RL/` with no dot segments or backslashes in
either the raw or percent-decoded path (so `%2e%2e`/`%5c` can't smuggle
traversal). Anything else is dropped from the status.

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
`dist/update-manifest.json` alongside the zip (stamping `channel` from the tag:
a `-` prerelease suffix → `dev`, else `stable`). The release job in
`.github/workflows/ci.yml` attaches it to the GitHub release — that is what
makes the `releases/latest/download/update-manifest.json` URL resolve — and
also clobbers it onto the rolling `dev` release so the dev-channel URL resolves
too.

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
