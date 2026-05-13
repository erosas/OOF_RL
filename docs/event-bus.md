# OOF Event Bus — Design Document

## Motivation

Plugins currently receive raw `events.Envelope` values from the RL WebSocket directly via `HandleEvent`. This creates tight coupling: any change to the RL API requires updates across every plugin that touches the affected field. It also prevents plugins from generating their own events or reacting to other plugins' work without hard-wiring imports.

This document designs an **OOF Event Bus** — a stable, typed event stream that sits between the RL client and the rest of the application.

Goals:

1. Decouple plugins from the RL API surface.
2. Let plugins emit their own events (derived, heuristic, or informational).
3. Give plugins a pub/sub mechanism for loose cooperation.
4. Surface the event stream to browser widgets asynchronously.
5. Model event certainty explicitly so consumers know when to trust data.

---

## Event Model

### The `OOFEvent` Interface

Every event in the system, whether translated from the RL API or emitted by a plugin, satisfies this interface:

```go
type OOFEvent interface {
    Type()        string    // stable string key, e.g. "goal_scored"
    OccurredAt()  time.Time
    Certainty()   Certainty
    Source()      Source
    MatchGUID()   string    // "" when not match-scoped
}
```

The `Type()` string is the primary dispatch key for subscriptions. It must be stable across versions. Convention: `snake_case`, namespaced for plugin events (`forfeit.detected`, `party.detected`).

### Certainty

```go
type Certainty int

const (
    // Authoritative: directly from the RL API with no inference.
    // The game said so; we just translated it.
    Authoritative Certainty = iota

    // Inferred: derived from data we trust, or corroborated by multiple
    // independent sources agreeing on the same conclusion.
    Inferred

    // Signal: a single heuristic data point suggesting something may be true.
    // One plugin's opinion. Not confirmed. May be wrong.
    Signal
)
```

**Why three levels, not a float confidence?**

A float (0.0–1.0) sounds precise but is hard to reason about in practice: what does 0.73 mean? The three-tier enum maps cleanly to how consumers should behave:

| Certainty | Consumer behavior |
|---|---|
| Authoritative | Act immediately. Display as fact. |
| Inferred | Display as fact with optional footnote. Log to DB. |
| Signal | Accumulate. Don't display until corroborated or intentionally shown as speculation. |

**Who can emit what?**

- Any plugin or the RL translator may emit `Authoritative` events when they have direct, unambiguous knowledge of a fact — the RL API explicitly stated it, or the plugin directly observes a definitive state change with no inference required.
- Plugins may emit `Inferred` for deterministic derivations from Authoritative data (e.g. detecting overtime from clock state transitions).
- Plugins may emit `Signal` for heuristic conclusions.

### Source

```go
type Source struct {
    PluginID string // "" for RL API translations
    Label    string // human-readable, e.g. "forfeit-detector"
}
```

### Base Struct

Concrete event types embed a base:

```go
type Base struct {
    EventType  string
    At         time.Time
    EventCert  Certainty
    Src        Source
    GUID       string
}

func (b Base) Type()       string    { return b.EventType }
func (b Base) OccurredAt() time.Time { return b.At }
func (b Base) Certainty()  Certainty { return b.EventCert }
func (b Base) Source()     Source    { return b.Src }
func (b Base) MatchGUID()  string    { return b.GUID }
```

---

## Event Taxonomy

### Authoritative (RL API translations)

These are 1:1 mappings of RL API events. The translator owns them. Plugin authors can rely on their fields never changing without a major version bump of OOF itself.

| OOF Event Type | RL API Source | Notes |
|---|---|---|
| `match.started` | `game:match_guid` | First GUID assignment for a match |
| `match.ended` | `game:end` | Carries winner team num |
| `goal.scored` | `game:goal_scored` | Scorer, assister, speed, time |
| `stat.feed` | `game:statfeedEvent` | Goal, Save, Assist, etc. |
| `clock.updated` | `game:clock_updated_seconds` | Rate-limited; not every tick |
| `crossbar.hit` | `game:crossbar_hit` | Ball hit the post/crossbar |
| `ball.hit` | `game:ballHit` | High frequency — off by default |
| `state.updated` | `game:update_state` | Full game snapshot; rate-limited |

### Inferred (derived, high confidence)

Emitted by the RL translator or a core analyzer when a conclusion is certain from the data even though the RL API doesn't explicitly say it:

| OOF Event Type | Derivation |
|---|---|
| `overtime.started` | `bOvertime` transition false→true within an active match |
| `match.restarted` | New GUID seen for a match already in progress (reconnect) |

### Plugin-Contributed

Plugins declare their event types at registration. Examples:

| OOF Event Type | Plugin | Certainty | Notes |
|---|---|---|---|
| `forfeit.signal` | forfeit-detector | Signal | Heuristic; needs corroboration |
| `forfeit.detected` | bus (corroborated) | Inferred | Promoted when N signals agree |
| `party.signal` | party-detector | Signal | |
| `party.detected` | bus | Inferred | |
| `gamemode.identified` | gamemode-analyzer | Inferred | From playlist field |
| `comeback.signal` | comeback-tracker | Signal | |

---

## The Event Bus

### Interfaces

Two distinct publish surfaces enforce the Authoritative restriction structurally — not via convention or runtime checks.

**`PluginBus`** is the surface all plugins and the RL translator receive in `Init`. It exposes all three certainty levels as separate methods:

```go
// PluginBus is the bus surface exposed to plugins and the RL translator.
// Separate publish methods enforce certainty at the call site — wrong
// certainty is a compile error, not a runtime panic.
type PluginBus interface {
    // PublishAuthoritative emits a direct, unambiguous fact.
    // Panics if e.Certainty() != Authoritative.
    PublishAuthoritative(e OOFEvent)

    // PublishInferred emits a high-confidence derived event.
    // Panics if e.Certainty() != Inferred.
    PublishInferred(e OOFEvent)

    // PublishSignal emits a heuristic event.
    // Panics if e.Certainty() != Signal.
    PublishSignal(e OOFEvent)

    // Subscribe registers a handler for a specific event type.
    // Returns a Subscription that must be cancelled on plugin shutdown.
    Subscribe(eventType string, fn func(OOFEvent)) Subscription

    // SubscribeAll registers a handler for every event.
    SubscribeAll(fn func(OOFEvent)) Subscription

    // SubscribeMinCertainty filters by minimum certainty level.
    SubscribeMinCertainty(eventType string, min Certainty, fn func(OOFEvent)) Subscription
}
```

**`Bus`** extends `PluginBus` with lifecycle methods used by application bootstrap. It is never passed to plugins:

```go
// Bus is the internal event bus. Extends PluginBus with lifecycle
// methods used by the application — not exposed to plugins.
type Bus interface {
    PluginBus
    Start() error
    Stop()
    // ResetMatch clears corroboration state when a match ends.
    ResetMatch(guid string)
}
```

**Why separate methods (`PublishAuthoritative` / `PublishInferred` / `PublishSignal`) instead of a single `Publish(OOFEvent) error`?**

A single method with a runtime certainty check is a runtime surprise: the bug compiles and ships, failing only when the code path executes. Separate methods make the wrong certainty unrepresentable at the call site — a plugin author calling the wrong one gets a compile error instead.

### Source Stamping

Each plugin and the RL translator receives a `PluginBus` instance pre-configured with its ID. When any `Publish*` method is called, the bus automatically stamps `Source.PluginID` on the event before dispatching — callers never set the source themselves. This prevents misattribution and keeps event provenance consistent.

```go
// The bus wraps each publish call:
func (pb *pluginBus) PublishAuthoritative(e OOFEvent) {
    stampSource(e, pb.pluginID) // sets Source.PluginID
    pb.inner.dispatch(e)
}
```

Convention: the RL translator receives a bus instance configured with `pluginID = ""`, so `Source.PluginID == ""` continues to mean "directly from the RL API."

```go
type Subscription interface {
    Cancel()
}
```

### Dispatch Model

**Single goroutine, ordered dispatch.** The bus runs one internal goroutine reading from a buffered channel. All `Publish` calls enqueue; all handler calls happen sequentially on the dispatcher goroutine.

Trade-off accepted: a slow handler blocks all subsequent events. Mitigation: handlers must not do I/O or long computation inline — they should enqueue to their own internal channel and process asynchronously. The bus enforces a per-handler timeout (e.g. 50ms) and logs warnings.

This guarantees:
- Events arrive at subscribers in publication order.
- No subscriber sees an event before the one published before it.
- Match state derived from events is always consistent.

### Corroboration

When multiple plugins independently emit a `Signal` event of the same type for the same match within a time window, the bus upgrades certainty automatically.

```
Signal₁ (plugin A) ─┐
                     ├─► bus accumulates ─► threshold met? ─► emit Inferred + Corroborated
Signal₂ (plugin B) ─┘
```

Config per event type:

```go
type CorroborationPolicy struct {
    EventType       string
    PromotedType    string        // e.g. "forfeit.detected"
    MinSources      int           // minimum unique plugin sources
    Window          time.Duration // max time between first and last signal
}
```

The corroborated `Inferred` event carries references to all contributing signals. Subscribers to the original `Signal` type still receive each signal as it arrives. Subscribers to the promoted type only receive the corroborated result.

**What if two plugins emit a duplicate `Authoritative`-level event?**

This shouldn't happen — only the RL translator emits `Authoritative`. If it fires twice (e.g. reconnect replays an event), deduplication by `(Type, MatchGUID, timestamp window)` is desirable, but it is not implemented in the bus/translator yet. For now, treat replayed `Authoritative` events as a known gap and future work; once implemented, duplicates should be dropped and logged.

---

## Plugin Lifecycle Changes

### Two Interfaces: Analyzer and Plugin

Not everything needs a UI. We split the interface:

```go
// Analyzer is a background computation unit. No UI, no routes.
// Analyzers subscribe to events, emit events, and may have settings.
type Analyzer interface {
    ID()         string
    DBPrefix()   string
    Requires()   []string
    Init(bus Bus, registry Registry, db *db.DB) error
    Shutdown()   error
    SettingsSchema()    []Setting
    ApplySettings(map[string]string) error
    DeclaredEvents() []EventDeclaration // event types this plugin may emit
}

// Plugin extends Analyzer with a UI tab, HTTP routes, and assets.
type Plugin interface {
    Analyzer
    NavTab()  NavTab
    Routes(mux *http.ServeMux)
    Assets()  fs.FS
}
```

`HandleEvent(env events.Envelope)` is **removed** from both interfaces. Plugins subscribe in `Init` instead.

### Init / Shutdown

`Init` is called once at startup after all plugins are registered. The plugin receives:
- `bus` — to subscribe and publish
- `registry` — to query other plugins by ID (soft dependency)
- `db` — for storage (same embedded SQLite, plugin's own prefix)

`Shutdown` is called on graceful exit. Plugins must cancel subscriptions and drain in-flight work.

### DeclaredEvents

```go
type EventDeclaration struct {
    Type        string
    Certainty   Certainty
    Description string
}
```

Plugins declare what they emit. The registry validates this and can expose it via `/api/events/schema` for tooling and the debug assistant. A plugin that emits an undeclared event type gets a logged warning.

### Registry

```go
type Registry interface {
    // Get returns a registered Plugin by ID, false if absent.
    // Plugins should not hard-fail if a dependency is absent.
    Get(id string) (Plugin, bool)
    GetAnalyzer(id string) (Analyzer, bool)
    List() []Plugin
    ListAnalyzers() []Analyzer
}
```

Soft dependency pattern:

```go
func (p *ForfeitPlugin) Init(bus Bus, reg Registry, db *db.DB) error {
    // Works fine without the MMR plugin; just won't annotate forfeits with rank data.
    if mmr, ok := reg.Get("mmr"); ok {
        p.mmr = mmr
    }
    bus.Subscribe("stat.feed", p.onStatFeed)
    return nil
}
```

---

## DB Logging

### Keep the domain DB as-is

Matches, goals, players, uploads — this data is stored by specific plugins as it arrives. No change.

### Add a lightweight Event Journal

A new `events` table for `Authoritative` and `Inferred` events only:

```sql
CREATE TABLE oof_events (
    id          INTEGER PRIMARY KEY,
    type        TEXT NOT NULL,
    match_guid  TEXT,
    occurred_at INTEGER NOT NULL, -- unix ms
    certainty   INTEGER NOT NULL, -- 0=Authoritative, 1=Inferred, 2=Signal
    source_id   TEXT,
    payload     TEXT              -- JSON
);
```

`Signal` events are **not** journaled. They are ephemeral; if we need to re-evaluate forfeit likelihood we recompute from the stored domain data.

**Why journal at all?**

- On browser reconnect, the hub can replay recent `Authoritative` events (last N minutes) to hydrate widget state without waiting for the next RL event.
- The debug assistant can query the journal to understand what happened in a session.
- Future: "why did OOF flag this match as a forfeit?" — walk the journal.

**Who controls journaling?** The bus itself, not individual plugins. Any `Authoritative` or `Inferred` event published to the bus is automatically persisted. Plugins don't need to think about it.

---

## Frontend Subscription

The existing WS hub fan-out changes: instead of forwarding raw RL `Envelope` JSON, it forwards `OOFEvent` JSON.

Browser clients send a subscription control message on connect:

```json
{ "action": "subscribe", "types": ["goal.scored", "forfeit.detected"] }
{ "action": "subscribe", "min_certainty": "inferred" }
{ "action": "subscribe_all" }
```

Widgets that currently react to raw RL events update their WS handler to listen for OOF event types instead. The payload is the OOF event's JSON, which includes `type`, `occurred_at`, `certainty`, `source`, `match_guid`, and event-specific fields.

Widgets can still call REST APIs for initial hydration; the event stream provides live updates.

---

## Open Questions

| Question | Options | Leaning |
|---|---|---|
| Who enforces the 50ms handler timeout? | Bus? Separate watchdog? | Bus, configurable |
| Can plugins publish `Inferred` without going through corroboration? | Yes (for deterministic derivations) or no (all heuristics must go through Signal→corroboration) | Yes — trust the plugin author |
| Event journal in main DB or separate file? | Same `oof_rl.db` under `oof_events` table, or `oof_events.db` | Same DB, simpler |
| Corroboration time window: fixed or match-scoped? | Fixed 60s window vs. "within the same match" | Match-scoped avoids cross-match false positives |
| Should the bus reset corroboration state on `match.ended`? | Yes (clean slate) vs. carry forward for analysis | Yes |
| Can Analyzers-only (no UI) register a settings schema? | Yes — settings page renders a card for them | Yes |
| How do plugins handle events emitted during Init before they finish subscribing? | Buffer during init phase vs. plugins are responsible | Buffer: bus queues events until all Init calls complete |

---

## Migration Plan (Rough)

1. Define `OOFEvent`, `Certainty`, `Source`, `Base`, `Bus`, and `Subscription` in `internal/oofevents`.
2. Write the RL-to-OOF translator in `internal/rlevents`: consumes `events.Envelope`, publishes `Authoritative` OOF events.
3. Add `Init`/`Shutdown`/`DeclaredEvents` to the `Plugin` interface; provide no-op defaults via a `BasePlugin` embed so existing plugins compile.
4. Keep `HandleEvent` temporarily — adapter in server dispatches raw events AND publishes via the new bus. Remove once all plugins migrate.
5. Migrate plugins one at a time: replace `HandleEvent` logic with `bus.Subscribe` in `Init`.
6. Add the event journal table and bus-side persistence.
7. Update hub fan-out to forward OOF events; update JS WS handler.
8. Introduce `Analyzer` interface for pure-computation plugins (forfeit, party, etc.).
