# Update Checker

The manual update checker is host-core (`internal/update`), not a plugin: the
end state of an updater — replace the running exe and restart — can never live
inside the WASM sandbox, so the check/download flow lives beside it in the
host.

## Milestone 1 Behavior

- No checks at startup. The Settings page has a manual **Check for updates**
  button.
- Check fetches
  `https://github.com/erosas/OOF_RL/releases/latest/download/update-manifest.json`
  and compares its version against the running version
  (`config.AppVersion`, set by release builds via
  `-ldflags "-X OOF_RL/internal/config.AppVersion=v1.2.3"`; dev builds report
  `dev`).
- Download streams the release zip to `%LOCALAPPDATA%\OOF_RL\updates\` on a
  background goroutine; the UI polls progress. The file is kept only when its
  SHA256 matches the manifest.
- Nothing is installed, extracted, replaced, or restarted.

## Routes

| Route | Method | Behavior |
|---|---|---|
| `/api/update/status` | GET | Current `update.Status` snapshot (poll target) |
| `/api/update/check` | POST | Fetch + validate manifest; 502 with status body on upstream failure |
| `/api/update/download` | POST | Start background download; 409 when no checked update is available |

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

`version`, `artifact_url`, `artifact_name`, and `artifact_sha256` are
required; the manifest is rejected otherwise.

`scripts/package-release.ps1 -Version vX.Y.Z` generates
`dist/update-manifest.json` alongside the zip, and the release job in
`.github/workflows/ci.yml` attaches it to the GitHub release — that is what
makes the `releases/latest/download/update-manifest.json` URL resolve.

## Trust Boundary

Milestone 1 is SHA256-only. The hash and the artifact URL come from the same
unsigned manifest, so verification proves transport integrity, not who
authored the release. Signed manifests (Ed25519 detached signature with a
public key pinned in the app) are a prerequisite for any milestone that
installs anything.