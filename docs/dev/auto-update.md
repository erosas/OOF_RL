# Auto Update Plugin

Milestone 1 adds a manual update check and verified download flow through the
`autoupdate` WASM plugin.

## Milestone 1 Behavior

- The plugin does not check for updates at startup.
- The Settings page exposes a manual **Check for updates** action.
- The plugin fetches `https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json`.
- If a newer version is found, the user can manually download the release zip.
- The downloaded zip is accepted only when its SHA256 matches the manifest.
- The plugin does not install, extract, replace files, restart the app, or mutate plugin loading.

State is stored in `%LOCALAPPDATA%\OOF_RL\plugin_data\autoupdate\state.json`.
Downloads are stored under `%LOCALAPPDATA%\OOF_RL\plugin_data\autoupdate\downloads\`.

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

## Trust Boundary

Milestone 1 is SHA256-only. SHA256 verifies that the downloaded artifact matches
the manifest, but it does not prove who authored the manifest. Signed manifests
remain a follow-up requirement, preferably with an Ed25519 detached signature and
a pinned public key in the app.
