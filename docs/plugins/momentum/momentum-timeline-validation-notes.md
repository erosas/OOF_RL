# Momentum Timeline Validation Notes

Status: validation notes for the runtime-only Momentum Timeline stack

Scope: documentation only. These notes describe how to validate the hidden
runtime Timeline snapshot endpoint before adding Timeline UI, persistence,
plugin-facing access, or production routes.

## Current Candidate Behavior

The current Momentum Timeline runtime stack is:

```text
typed GameActionEvent
-> Momentum service update
-> Timeline collector sample
-> core-internal Timeline provider
-> hidden read-only snapshot endpoint
```

The hidden validation endpoint is:

```text
GET /internal/momentum-timeline-snapshot
```

It returns the current runtime-only `TimelineSnapshot` from
`core.Server.Timeline()`.

This endpoint is intended for developer/playtest validation only. It is not a
production Timeline UI and should not be treated as a persisted match review
surface.

## Safety Boundary

The validation endpoint is read-only.

It must not:

- write SQLite rows,
- change SQLite schema,
- mutate Session, History, Live, saved match, or replay data,
- change Momentum Engine math,
- change Overlay HUD behavior,
- adopt Timeline design assets,
- expose Timeline through `plugin.Registry`,
- add frontend dependencies or framework code.

Data read:

- runtime Timeline snapshot memory.

Data written:

- none.

## How To Inspect

Start the app with the Timeline runtime stack enabled, then open:

```text
http://localhost:<app-port>/internal/momentum-timeline-snapshot
```

Expected empty-state shape before a recorded action:

```json
{
  "MatchGUID": "",
  "NextIndex": 0,
  "MatchEnded": false,
  "EndedReason": "",
  "Entries": []
}
```

Field names currently reflect Go JSON defaults. Do not treat that casing as a
final user-facing API contract.

## Fields To Validate

At the snapshot level:

- `MatchGUID`
- `NextIndex`
- `MatchEnded`
- `EndedReason`
- `Entries`

At the entry level:

- `Index`
- `MatchGUID`
- `OccurredAt`
- `Action`
- `ActorTeam`
- `ImpactTeam`
- `PlayerID`
- `PlayerName`
- `VictimID`
- `IsOwnGoal`
- `IsEpicSave`
- `MomentumSequence`
- `Blue`
- `Orange`
- `Category`

At the team sample level:

- `Pressure`
- `MomentumInfluence`
- `ContestInvolvement`
- `EventDerivedControl`
- `Confidence`
- `Volatility`

## Manual Validation Scenarios

### Empty Snapshot

Scenario:
Start the app before entering a match or before receiving a supported
`GameActionEvent`.

Expected:

- `Entries` is empty.
- `NextIndex` is `0`.
- `MatchEnded` is `false`.
- No DB, History, Session, Live, saved match, or replay data changes occur.

### Shot

Scenario:
Record or play through a clear shot event.

Expected:

- A new Timeline entry appears.
- `Action` is `shot`.
- `ActorTeam` matches the shooter team.
- `PlayerName` and `PlayerID` match available event data.
- `MomentumSequence` is greater than `0`.
- The actor team's sampled pressure/control values reflect the Momentum
  snapshot after the event.

### Goal

Scenario:
Record or play through a goal event.

Expected:

- A new Timeline entry appears.
- `Action` is `goal`.
- `ImpactTeam` is the scoring team.
- `Category` remains a safe display category, not win prediction or possession.
- Sampled Momentum values are non-empty for the impacted team.

### Own Goal

Scenario:
Record or play through an own goal if available.

Expected:

- `Action` is `goal`.
- `IsOwnGoal` is `true`.
- `ActorTeam` is the player team that caused the own goal.
- `ImpactTeam` is the opposing team.

Known limitation:

- Own-goal availability depends on typed event normalization and the source
  event stream.

### Save And Epic Save

Scenario:
Record or play through a save and, if possible, an epic save.

Expected:

- `Action` is `save`.
- `IsEpicSave` is `false` for regular saves.
- `IsEpicSave` is `true` for epic saves.
- `Category` is `contest`.
- Defensive and attacking sampled values are plausible for the event.

### Assist

Scenario:
Record or play through an assist event.

Expected:

- `Action` is `assist`.
- `ActorTeam` matches the assisting player's team.
- `Category` is `pressure`.

Known limitation:

- Assist timing depends on the typed stat-feed event source.

### Demo

Scenario:
Record or play through a demolition event.

Expected:

- `Action` is `demo`.
- `VictimID` is present when the source event provides it.
- `Category` is `volatility`.

### Match Restart

Scenario:
Trigger or observe a new match GUID during the same app session.

Expected:

- Timeline resets for the new match.
- `MatchGUID` updates.
- `Entries` clears.
- `NextIndex` returns to `0`.

### Match End

Scenario:
Finish a match and inspect the endpoint before match destroy/reset.

Expected:

- `MatchEnded` is `true`.
- `EndedReason` is `match.ended`.
- Existing entries remain visible for runtime inspection.
- Same-match actions after end are not appended.

### Match Destroy

Scenario:
Leave or tear down the match session.

Expected:

- Runtime Timeline state clears.
- `MatchGUID` is empty.
- `Entries` is empty.
- `MatchEnded` is `false`.

## Validation Log Template

```text
Branch:
Commit SHA:
EXE path:
Date/time:
App data directory:
Tester:
Rocket League version/mode:

Endpoint:
Scenario:
Player names and scores:
Observed snapshot notes:
Ideas or bugs with EST timestamps:
End-of-match stats:
- goals
- assists
- saves
- shots
- demos
- touches
- score
```

## Known Limitations Before UI

- The endpoint is hidden and developer-facing.
- JSON field casing is not final.
- There is no Timeline UI or route-level presentation model.
- There is no persistence or History integration.
- There is no plugin-facing Timeline provider.
- Timeline entries use runtime sampled Momentum values only.
- Event availability depends on typed event normalization.
- The endpoint does not include match clock display data yet.
- Timeline design assets remain reference-only and are not production UI.

## Acceptance Criteria Before UI Work

Before building a Timeline UI slice, validate:

- event ordering is stable,
- supported actions appear once,
- sampled Momentum values are taken after Momentum service update,
- match restart/end/destroy lifecycle behavior is correct,
- own-goal and epic-save flags are accurate when source data provides them,
- buffer behavior is acceptable during a full match,
- category labels remain safe and do not imply possession, rotations, tactical
  certainty, or win prediction.

## Recommended Next Implementation Slice

Next PR:

```text
feature/momentum-timeline-internal-preview
```

Recommended scope:

- add a hidden/internal read-only Timeline preview surface,
- consume `/internal/momentum-timeline-snapshot` or `core.Server.Timeline()`,
- use vanilla HTML/CSS/JS only,
- render runtime entries for developer validation,
- keep design assets reference-only unless explicitly approved,
- no persistence,
- no DB/schema changes,
- no Session/History/Live/replay mutation,
- no Overlay HUD behavior changes,
- no plugin registry Timeline exposure.
