# Debug Assistant Plugin

## Purpose

The Debug Assistant is a developer/tester workflow plugin for structured OOF RL regression testing. It helps record pass/fail/N/A checks, link played matches to specific verification items, preserve notes/evidence, and export reviewable reports for PRs.

The plugin is intended to support testing and documentation only. It does not replace core History, Session, Live, or database source-of-truth behavior.

## Scope

- Track A: App Match Regression
- Track B: Debug Assistant verification
- Track C: Commit fix verification
- Track D: Debug Match Linking Bug Watch
- Pass/fail/N/A checklist state
- Scenario notes and screenshot filename references
- Manual JSON state import
- Markdown, HTML, and JSON report export
- Debug match linking metadata
- Inline linked match evidence inside checklist items
- Reset behavior for Debug Assistant state and observed debug events
- Floating link action buttons for confirming, clearing, and deselecting debug link targets

## Safety Boundary

- Debug Assistant state is metadata layered on top of existing app data.
- The plugin must not mutate History, Session, Live, or match source-of-truth data.
- No database schema is required for debug links in this implementation.
- Resetting Debug Assistant state must clear only Debug Assistant metadata, report output, export state, selected link batches, and observed debug-event buffers.
- Core Rocket League event handling must remain non-blocking.

## App-Shell View Scroll Isolation

The per-page scroll isolation added with this work lives in the shared app shell, not inside the Debug Assistant plugin itself.

Each enabled app page/plugin renders as a `.view` section. The app shell stores and restores scroll state per active `.view`, so scrolling History does not change Debug's saved position, and scrolling Debug does not change History's saved position.

This behavior is UI-only:

- It does not affect Rocket League event processing.
- It does not affect WebSocket handling.
- It does not affect backend match/session/history logic.
- It does not write scroll state to the database or app data directory.
- If a plugin/page is disabled, that page may still be injected by the app shell, but it is hidden from navigation and not user-reachable as an active view for scroll-state purposes.

## Acceptance Criteria

- Track A/B/C/D appear as top-level Debug Assistant tracks.
- Each track exposes grouped checks with pass/fail/N/A state.
- Notes wrap correctly and remain readable.
- Reports include metadata, track summaries, notes, failures, and linked evidence.
- HTML report preserves Session Notes formatting.
- JSON import requires manual file selection and confirmation.
- Reset clears Debug Assistant metadata, reports, export output, selected links, and observed event buffer.
- Confirm Links workflow supports one or more selected issue checks.
- One match can link to multiple checks.
- Linked evidence appears inline under each relevant check.
- No duplicate large standalone linked match panels appear.
- Removing evidence affects only Debug Assistant metadata.
- Enabled app pages/plugins preserve independent frontend scroll positions during navigation.

## Known Deferrals

- Full Selenium/capture replay automation.
- Persistent database-backed debug links.
- Screenshot embedding expansion.
- Reusable future-app Debug Assistant framework documentation.
- Custom track template export/import for adding tester-defined tracks without replacing built-in Track A/B/C/D.
- External replay metadata lookup.
- Core History/Session heuristic changes.
