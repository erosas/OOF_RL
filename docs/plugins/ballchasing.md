# Ballchasing Plugin

The Ballchasing plugin integrates OOF RL with [ballchasing.com](https://ballchasing.com) so users can upload Rocket League replay files, view their recent Ballchasing uploads, and connect local match history to replay upload status.

This document is intended for developers and maintainers working on the plugin.

## Responsibilities

The plugin currently owns four main workflows:

1. API-key based connectivity to Ballchasing.
2. Manual upload of local `.replay` files.
3. Automatic upload after match completion.
4. Replay status display for matches recorded in OOF RL history.

The plugin is registered with:

```go
func (p *Plugin) ID() string       { return "ballchasing" }
func (p *Plugin) DBPrefix() string { return "bc" }
```

It exposes a nav tab with ID `bc` and label `Ballchasing`.

## Settings

The plugin contributes these settings through `SettingsSchema()`:

| Key | Type | Purpose |
| --- | --- | --- |
| `ballchasing_api_key` | password | Ballchasing API key. Required for ping, upload, replay listing, sync, and groups. |
| `ballchasing_delete_after_upload` | checkbox | Deletes a local replay file after a successful upload. Defaults to false. |

The API key is stored in config but omitted from JSON responses via `json:"-"`.

## Database

The plugin creates one table:

```sql
CREATE TABLE IF NOT EXISTS bc_uploads (
  replay_name    TEXT PRIMARY KEY,
  ballchasing_id TEXT NOT NULL
);
```

`bc_uploads` maps local replay names, or normalized replay GUID keys, to Ballchasing replay IDs.

Examples:

```text
SomeReplay.replay -> ballchasing replay ID
024690394AE0B6BB20BBD1A3EFB2DA1E.replay -> ballchasing replay ID
```

The second form is used by sync-from-Ballchasing, where the remote replay is matched by Rocket League ID rather than by local filename.

## Runtime Flow

### Connectivity Check

`GET /api/ballchasing/ping`

The backend calls:

```text
GET https://ballchasing.com/api/
```

If the API key is valid, the endpoint returns the connected account name when available. The frontend displays this in the Ballchasing status bar.

### Match Replay Listing

`GET /api/ballchasing/matches`

The backend:

1. Reads recent OOF RL history matches that started after plugin startup.
2. Scans the user's Rocket League replay folder.
3. Filters replay files to files modified after plugin startup.
4. Assigns replay files to matches with a one-to-one greedy matching algorithm.
5. Combines local replay status with known Ballchasing upload status from `bc_uploads`.

The frontend displays each match as one of:

- uploadable local replay found
- already uploaded
- no replay saved

### Manual Upload

`POST /api/ballchasing/upload`

Request body:

```json
{
  "replay_name": "Example.replay",
  "visibility": "unlisted"
}
```

The backend validates:

- `replay_name` is present.
- `replay_name` is a basename and cannot contain a path.
- `visibility` is one of `public`, `unlisted`, or `private`.

If `visibility` is omitted, it defaults to `unlisted`.

The backend uploads to:

```text
POST https://ballchasing.com/api/v2/upload?visibility={visibility}
```

After a `200` or `201` response, the plugin records the returned Ballchasing ID in `bc_uploads`. If `ballchasing_delete_after_upload` is enabled, the local replay file is deleted after the successful upload.

### Automatic Upload

Automatic upload is event-driven.

On `MatchEnded`, the plugin broadcasts:

```json
{
  "Event": "bc:save-replay-reminder"
}
```

The frontend uses that event to remind the user to save the replay while the post-match screen is still available.

On `MatchDestroyed`, the plugin:

1. Requires a configured Ballchasing API key.
2. Uses an `uploadPending` guard to avoid overlapping upload scans.
3. Waits 12 seconds to give Rocket League time to finish writing replay files.
4. Scans the replay folder for `.replay` files written after plugin startup.
5. Skips replay names that are already present in `bc_uploads`.
6. Uploads new replay files as `unlisted`.
7. Broadcasts `bc:uploaded` when at least one replay uploads successfully.

The frontend uses `bc:uploaded` to update the uploaded replay list and refresh the match replay status.

### Sync From Ballchasing

`POST /api/ballchasing/sync`

The backend fetches recent Ballchasing uploads:

```text
GET https://ballchasing.com/api/replays?uploader=me&count=200&sort-by=replay-date&sort-dir=desc
```

For each replay with both an `id` and `rocket_league_id`, the plugin normalizes the Rocket League ID and stores it in `bc_uploads` as:

```text
{NORMALIZED_GUID}.replay -> {ballchasing_id}
```

This allows the match replay list to show uploaded status even when the local filename is not known.

### Delete Uploaded Replays

`POST /api/ballchasing/local-replays/purge`

The backend deletes local replay files whose names appear in `bc_uploads`.

This is intended to clean up replay files after they are safely uploaded to Ballchasing. It only increments the deleted count when `os.Remove` succeeds.

## Replay Folder Detection

The plugin looks for Rocket League replay files in:

```text
{Documents}\My Games\Rocket League\TAGame\Demos
```

Candidate base directories include:

- `%USERPROFILE%\OneDrive\Documents`
- `%USERPROFILE%\Documents`
- `%OneDriveConsumer%\Documents`
- `%OneDrive%\Documents`

If no replay directory is found, the backend logs all checked candidate paths.

## API Routes

| Route | Method | Purpose |
| --- | --- | --- |
| `/api/ballchasing/ping` | GET | Validate API key and return connected account metadata. |
| `/api/ballchasing/matches` | GET | Return OOF RL session matches with replay/upload status. |
| `/api/ballchasing/replays` | GET | Proxy recent Ballchasing replays for the current uploader. |
| `/api/ballchasing/groups` | GET | Proxy Ballchasing groups for the current creator. |
| `/api/ballchasing/upload` | POST | Upload a specific local replay file. |
| `/api/ballchasing/sync` | POST | Backfill upload records from Ballchasing replay history. |
| `/api/ballchasing/local-replays/purge` | POST | Delete local replay files that are known as uploaded. |

## Frontend Behavior

The frontend implementation lives in:

```text
internal/plugins/ballchasing/view.html
internal/plugins/ballchasing/view.js
```

On page load, `loadBC()` concurrently fetches:

- match replay status
- uploaded Ballchasing replays
- Ballchasing groups

The UI has three sections:

1. Match Replays
2. Your Replays on Ballchasing
3. Groups

The Match Replays section supports:

- manual upload
- upload status badges
- view links to Ballchasing
- pagination
- sync-from-Ballchasing
- delete-uploaded cleanup

## Usage Examples

### Check Ballchasing Connection

```bash
curl http://localhost:8080/api/ballchasing/ping
```

### Upload a Replay

```bash
curl -X POST http://localhost:8080/api/ballchasing/upload \
  -H "Content-Type: application/json" \
  -d '{"replay_name":"Example.replay","visibility":"unlisted"}'
```

Valid visibility values are:

- `public`
- `unlisted`
- `private`

### Sync Uploaded Replays From Ballchasing

```bash
curl -X POST http://localhost:8080/api/ballchasing/sync
```

### Delete Uploaded Local Replays

```bash
curl -X POST http://localhost:8080/api/ballchasing/local-replays/purge
```

Use this endpoint carefully. It deletes local files from the Rocket League replay folder.

## Current Limitations and Known Issues

- Match replay status is scoped to matches and replay files created after plugin startup. After restarting the app, older local matches may not appear in the Match Replays section.
- Sync-from-Ballchasing stores each returned `rocket_league_id` in `bc_uploads` without first verifying that the replay exists in local OOF RL match history.
- Delete Uploaded deletes by the full `bc_uploads` table, not by an explicit list of replay names selected by the current UI action.
- Automatic upload uses a fixed `unlisted` visibility. There is not currently a persisted or runtime UI selector for upload privacy.
- Automatic upload only treats HTTP `200` and `201` as success. Duplicate upload responses, such as `409`, may need dedicated handling if Ballchasing returns a usable replay ID in the response body.
- Replay folder detection is Windows-focused and assumes standard Documents or OneDrive Rocket League paths.
- External Ballchasing links currently rely on WebView2/default anchor behavior. The app does not yet explicitly hand external links to the system default browser.
- The plugin stores only the Ballchasing replay ID in `bc_uploads`; URL and upload timestamp can be derived or added later if needed.

## Technology Decisions

- The plugin uses the existing OOF RL plugin system for nav, settings, routes, assets, and event handling.
- Ballchasing API requests are made server-side so the API key is not exposed to frontend JavaScript.
- Plugin UI assets are embedded with Go `embed` and served through the plugin asset system.
- Local replay file matching is based on replay file modification time and match start time instead of reading replay metadata directly.
- Upload state is stored in SQLite using `bc_uploads` so the UI can persist uploaded status across page reloads.
- Manual replay upload validates filenames with `filepath.Base` to prevent path traversal.

## Technical Debt and Follow-Up Ideas

- Restrict purge to replay names explicitly confirmed by the current UI action.
- Restrict sync to Ballchasing replays that match known local OOF RL history rows.
- Add a Ballchasing upload privacy selector with `unlisted` as the default.
- Add first-class handling for duplicate upload responses.
- Consider reading replay metadata directly instead of relying only on file modification time.
- Consider storing `uploaded_at` and derived Ballchasing URLs in `bc_uploads` for easier auditing.
- Move external Ballchasing links through an explicit "open in default browser" backend route or WebView2 integration.
- Add tests for `handleUpload`, `handleSync`, purge behavior, and replay directory matching edge cases.

