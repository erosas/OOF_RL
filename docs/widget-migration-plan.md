# Widget Migration Plan

## Context

The ballchasing plugin was migrated to the composable widget concept, establishing the pattern:
each plugin can register widgets into a global `window.widgetRegistry` that the Dashboard tab
renders into a user-configurable GridStack grid. The goal is to extend this to all remaining
plugins so users can compose a dashboard from any combination of plugin views.

All plugins **keep their full-page nav tabs for now**. Widget factories are added alongside
the existing tab UI — no tab is removed in this migration.

---

## Widget system contract

A widget definition registered via `window.registerWidget(def)` has this shape:

```js
{
  id:       string,   // globally unique, e.g. 'live-scoreboard'
  pluginId: string,   // matches plugin nav tab ID, e.g. 'live'
  title:    string,   // shown in picker + widget header
  defaultW: number,   // grid columns (12-column grid)
  defaultH: number,   // grid rows (1 row = 60px)
  minW:     number,
  minH:     number,
  factory:  (container: HTMLElement) => { refresh(), destroy?() }
}
```

**Factory rules:**
- Must use `container.querySelector` — never `document.getElementById`
- Must return `{ refresh() }` at minimum; `destroy()` is called when the widget is removed from the grid
- May maintain module-level instance arrays for WS event routing; `destroy()` removes the instance

**Dashboard change required (Phase 0):** call `ctrl?.destroy?.()` in `_dashAddWidget`'s remove
button handler so factory instances can deregister from WS routing arrays.

---

## Migration order & plan

### Phase 0 — Dashboard: widget destroy lifecycle
**File:** `internal/plugins/dashboard/view.js`

In `_dashAddWidget`, call `ctrl?.destroy?.()` **before** `_dashGrid.removeWidget(itemEl)` in the remove button handler, and store `ctrl` on `itemEl._widgetCtrl` so that bulk removals (e.g. `_dashCancel` calling `removeAll`) can also iterate items and call `destroy()` before clearing the grid.
Enables all subsequent phases to implement clean WS deregistration.

Effort: ~3 lines.

---

### Phase 1 — Ranks
**Files:** `internal/plugins/ranks/view.js`, `view.html`

**Keep:** full-page tab (current UI unchanged).

**Add 1 widget:**

| id | title | defaultW | defaultH | minW | minH |
|----|-------|----------|----------|------|------|
| `ranks-display` | Player Ranks | 4 | 8 | 3 | 4 |

Factory renders both team rank panels into `container`. Maintains a module-level
`_ranksInstances` array. `handleRanksUpdate` and `handleRanksClear` push to all
active instances. `destroy()` splices from the array.

`refresh()` re-renders from cached `_rankPlayers` (already module-level state).

> **Future note:** Ranks is a pure monitoring widget — it has no data to browse and is only
> useful during a live match. Once widget-only mode is viable, Ranks is the first candidate to
> lose its nav tab. The full-page view adds no value over a dashboard widget.

Effort: low. WS routing is simple, render logic is already clean.

---

### Phase 2 — Live
**Files:** `internal/plugins/live/view.js`, `view.html`

**Keep:** full-page tab (current UI unchanged).

**Add 2 widgets:**

| id | title | defaultW | defaultH | minW | minH |
|----|-------|----------|----------|------|------|
| `live-scoreboard` | Live Score | 6 | 5 | 4 | 4 |
| `live-players` | Live Players | 12 | 10 | 6 | 6 |

`live-scoreboard` — arena name, clock, overtime/replay badges, team names + scores,
possession bar. Compact, works at small sizes.

`live-players` — full player table with boost bars, status chips, rank rows. Needs width.

Both maintain entries in `_liveInstances`. `handleUpdateState` routes to all instances.
`flashGoal` stays global (it's a body-level flash, not scoped to a container).
`refresh()` fetches `/api/live/state` and renders if active.

> **Future note:** Like Ranks, Live is monitoring-only. The full-page tab is redundant once
> users have the dashboard. Candidate for widget-only after the migration settles.

Effort: medium. The render functions (`teamPanel`, `updatePossessionBar`) need to be
refactored to accept a `root` element instead of calling `document.getElementById`.

---

### Phase 3 — Session
**Files:** `internal/plugins/session/view.js`, `view.html`

**Keep:** full-page tab with all controls (player picker, start/reset/new session buttons,
session history list with edit/delete). These are control surfaces — not appropriate for
a dashboard widget.

**Add 2 widgets:**

| id | title | defaultW | defaultH | minW | minH |
|----|-------|----------|----------|------|------|
| `session-summary` | Session Stats | 6 | 6 | 4 | 4 |
| `session-live-game` | Live Game Stats | 4 | 4 | 3 | 3 |

`session-summary` — W/L/G/A/Sv/Sh/Dm stat pills + elapsed time. Reads from
`/api/session/stats` using the module-level `_sessionPlayerID` and `_sessionSince`.
`refresh()` re-fetches. `handleSessionMatchStart` triggers refresh on all instances.

`session-live-game` — the in-game stats row for the tracked player (goals/assists/saves
in the current match). Updates from `handleSessionUpdate`. Shows placeholder when no
player selected or no match active.

Both read module-level session state (`_sessionPlayerID`, `_sessionSince`, `_liveStats`)
already maintained by the tab init — no duplication of state ownership.

Effort: medium. Factories are straightforward but need careful scoping since session state
is owned by the tab init path. If the session tab has never been opened, `_sessionPlayerID`
may be empty — widgets should handle that gracefully with a "Select a player in the Session
tab" placeholder.

---

### Phase 4 — History
**Files:** `internal/plugins/history/view.js`, `view.html`

**Keep:** full-page tab (paginated, filterable match list + inline detail panels).
The full browsing experience lives here.

**Add 1 widget:**

| id | title | defaultW | defaultH | minW | minH |
|----|-------|----------|----------|------|------|
| `history-recent` | Recent Matches | 6 | 10 | 4 | 6 |

Shows the last 10 matches (no filter, no pagination) in compact card form.
Click-to-expand inline detail using the shared `renderMatchDetailPanel` utility.
`refresh()` fetches `/api/matches` and re-renders. Triggered by `handleSessionMatchStart`
(a match completed, history may have a new entry).

This widget intentionally omits the player filter — it's a summary view. For filtering,
use the History tab.

Effort: medium. `renderMatchDetailPanel` is already a shared utility in `app.js`, so the
inline detail composes cleanly. The main work is making the match list render into
`container` rather than `#matches-list`.

---

### Phase 5 — Ballchasing (already done, verify)

Three widgets are already registered: `bc-match-replays`, `bc-uploaded-replays`, `bc-groups`.

Verify that `destroy()` is wired up correctly once Phase 0 is in place. The widget
controllers do not currently maintain a module-level instances array (they are stateless
fetch-on-refresh) so no WS deregistration is needed — `handleBCUploaded` directly calls
`_uploadedWidget` and `_matchWidget` which are the tab-view instances, not dashboard
widget instances. This is a gap to close: uploaded WS events should also refresh any
dashboard instances of those widgets.

Effort: small cleanup.

---

## Summary table

| Plugin | Tab kept? | Widgets | Effort | Phase |
|--------|-----------|---------|--------|-------|
| Dashboard | — | lifecycle fix | tiny | 0 |
| Ranks | yes (widget-only candidate) | `ranks-display` | low | 1 |
| Live | yes (widget-only candidate) | `live-scoreboard`, `live-players` | medium | 2 |
| Session | yes (control surface) | `session-summary`, `session-live-game` | medium | 3 |
| History | yes (browsing surface) | `history-recent` | medium | 4 |
| Ballchasing | yes (action surface) | already done — cleanup only | small | 5 |

**Total new widgets:** 6 (+ 1 lifecycle fix + 1 BC cleanup)

---

## Technical notes

### WS event routing pattern

```js
// module level (in the plugin's view.js)
const _myInstances = [];

function myWidget(container) {
  function render(data) { /* use container.querySelector, not document.getElementById */ }
  function refresh() { fetch('/api/my/endpoint').then(r => r.json()).then(render); }
  function destroy() { const i = _myInstances.indexOf(entry); if (i >= 0) _myInstances.splice(i, 1); }

  const entry = { render, refresh };
  _myInstances.push(entry);
  refresh();
  return { refresh, destroy };
}

// existing WS handler extended to also push to instances:
window.handleMyEvent = function(data) {
  updateTabView(data);                    // existing tab logic unchanged
  _myInstances.forEach(w => w.render(data));  // also push to dashboard widgets
};
```

### State ownership

Module-level state (e.g. `_rankPlayers`, `_sessionPlayerID`, `_liveStats`) stays owned
by the tab's `pluginInit_*` function. Widget factories read from it — they do not
re-initialise it. If the tab has never been opened, factories must handle the empty/null
state gracefully.

### Container-scoped rendering

Replace all `document.getElementById('foo')` with `container.querySelector('#foo')`
inside factory render functions. Tab-level code (`pluginInit_*`) continues to use
`document.getElementById` for the static tab DOM.