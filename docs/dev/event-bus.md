# OOF Event Bus

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

The `Type()` string is the primary dispatch key for subscriptions. It must be stable across versions. Convention: `dot.separated` lowercase, namespaced for plugin events (`forfeit.detected`, `party.detected`).

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

Trade-off accepted: a slow handler blocks all subsequent events. Mitigation: handlers must not do I/O or long computation inline — they should enqueue to their own internal channel and process asynchronously. Panics in handlers are recovered and logged so one bad handler cannot stop the bus.

This guarantees:
- Events arrive at subscribers in publication order.
- No subscriber sees an event before the one published before it.
- Match state derived from events is always consistent.

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
