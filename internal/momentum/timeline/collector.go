package timeline

import (
	"sync"
	"time"

	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
)

const defaultMaxEntries = 256

// Config controls the runtime-only Timeline collector.
type Config struct {
	MaxEntries int
}

// Collector stores a bounded, in-memory Timeline view for the current match.
// It is display/review enrichment only and does not write to source-of-truth
// match, session, history, replay, or database state.
type Collector struct {
	mu          sync.RWMutex
	provider    momentum.SnapshotProvider
	maxEntries  int
	entries     []TimelineEntry
	matchGUID   string
	nextIndex   int
	matchEnded  bool
	endedReason string
}

// TimelineSnapshot is a copied read-only view of the collector state.
type TimelineSnapshot struct {
	MatchGUID   string
	NextIndex   int
	MatchEnded  bool
	EndedReason string
	Entries     []TimelineEntry
}

// TimelineEntry is a runtime-only marker enriched with sampled Momentum state.
type TimelineEntry struct {
	Index            int
	MatchGUID        string
	OccurredAt       time.Time
	Action           oofevents.ActionKind
	ActorTeam        oofevents.Team
	ImpactTeam       oofevents.Team
	PlayerID         string
	PlayerName       string
	VictimID         string
	IsOwnGoal        bool
	IsEpicSave       bool
	MomentumSequence int
	Blue             SignalSample
	Orange           SignalSample
	Category         string
}

// SignalSample captures display-safe Momentum values at an event boundary.
type SignalSample struct {
	Pressure            float64
	MomentumInfluence   float64
	ContestInvolvement  float64
	EventDerivedControl float64
	Confidence          float64
	Volatility          float64
}

// NewCollector creates a runtime-only Timeline collector.
func NewCollector(provider momentum.SnapshotProvider, config Config) *Collector {
	maxEntries := config.MaxEntries
	if maxEntries <= 0 {
		maxEntries = defaultMaxEntries
	}
	return &Collector{
		provider:   provider,
		maxEntries: maxEntries,
		entries:    make([]TimelineEntry, 0, maxEntries),
	}
}

// HandleGameAction records supported typed game actions and samples current
// Momentum state. It returns false when the event is unsupported or unsafe.
func (c *Collector) HandleGameAction(event oofevents.GameActionEvent) bool {
	if !supportedAction(event.Action) || !supportedTeam(event.Team) {
		return false
	}

	state := momentum.MomentumState{}
	if c.provider != nil {
		state = c.provider.Snapshot()
	}
	entry := entryFromEvent(event, state)

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.matchGUID == "" {
		c.matchGUID = entry.MatchGUID
	}
	if entry.MatchGUID != "" && c.matchGUID != "" && entry.MatchGUID != c.matchGUID {
		c.resetLocked(entry.MatchGUID)
	}
	if c.matchEnded {
		return false
	}
	entry.Index = c.nextIndex
	c.nextIndex++
	c.entries = append(c.entries, entry)
	if overflow := len(c.entries) - c.maxEntries; overflow > 0 {
		c.entries = append([]TimelineEntry(nil), c.entries[overflow:]...)
	}
	return true
}

// Snapshot returns a copied view so callers cannot mutate collector state.
func (c *Collector) Snapshot() TimelineSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.snapshotLocked()
}

// Reset clears entries for a new match lifecycle boundary.
func (c *Collector) Reset(matchGUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resetLocked(matchGUID)
}

// MarkMatchEnded freezes lifecycle status while preserving collected entries.
func (c *Collector) MarkMatchEnded(matchGUID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if matchGUID != "" {
		c.matchGUID = matchGUID
	}
	c.matchEnded = true
	c.endedReason = "match.ended"
}

// Clear removes all runtime Timeline state.
func (c *Collector) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.resetLocked("")
}

func (c *Collector) resetLocked(matchGUID string) {
	c.entries = c.entries[:0]
	c.matchGUID = matchGUID
	c.nextIndex = 0
	c.matchEnded = false
	c.endedReason = ""
}

func (c *Collector) snapshotLocked() TimelineSnapshot {
	entries := append([]TimelineEntry(nil), c.entries...)
	return TimelineSnapshot{
		MatchGUID:   c.matchGUID,
		NextIndex:   c.nextIndex,
		MatchEnded:  c.matchEnded,
		EndedReason: c.endedReason,
		Entries:     entries,
	}
}

func entryFromEvent(event oofevents.GameActionEvent, state momentum.MomentumState) TimelineEntry {
	matchGUID := event.MatchGUID()
	if matchGUID == "" {
		matchGUID = state.MatchGUID
	}
	impactTeam := event.Team
	if event.Action == oofevents.ActionGoal && event.IsOwnGoal {
		impactTeam = oofevents.Opponent(event.Team)
	}
	if state.LastEvent.Action == event.Action &&
		state.LastEvent.OccurredAt.Equal(event.OccurredAt()) &&
		state.LastEvent.MatchGUID == event.MatchGUID() {
		impactTeam = state.LastEvent.ImpactTeam
	}

	return TimelineEntry{
		MatchGUID:        matchGUID,
		OccurredAt:       event.OccurredAt(),
		Action:           event.Action,
		ActorTeam:        event.Team,
		ImpactTeam:       impactTeam,
		PlayerID:         event.PlayerID,
		PlayerName:       event.PlayerName,
		VictimID:         event.VictimID,
		IsOwnGoal:        event.IsOwnGoal,
		IsEpicSave:       event.IsEpicSave,
		MomentumSequence: state.Sequence,
		Blue:             sampleSignal(state.Teams[oofevents.TeamBlue]),
		Orange:           sampleSignal(state.Teams[oofevents.TeamOrange]),
		Category:         categoryFor(event),
	}
}

func sampleSignal(signal momentum.TeamSignal) SignalSample {
	return SignalSample{
		Pressure:            signal.Pressure,
		MomentumInfluence:   signal.MomentumInfluence,
		ContestInvolvement:  signal.ContestInvolvement,
		EventDerivedControl: signal.EventDerivedControl,
		Confidence:          signal.Confidence,
		Volatility:          signal.Volatility,
	}
}

func categoryFor(event oofevents.GameActionEvent) string {
	switch event.Action {
	case oofevents.ActionBallHit:
		return "event impact"
	case oofevents.ActionShot, oofevents.ActionGoal, oofevents.ActionAssist:
		return "pressure"
	case oofevents.ActionSave:
		return "contest"
	case oofevents.ActionDemo:
		return "volatility"
	default:
		return ""
	}
}

func supportedAction(action oofevents.ActionKind) bool {
	switch action {
	case oofevents.ActionBallHit,
		oofevents.ActionShot,
		oofevents.ActionSave,
		oofevents.ActionGoal,
		oofevents.ActionAssist,
		oofevents.ActionDemo:
		return true
	default:
		return false
	}
}

func supportedTeam(team oofevents.Team) bool {
	return team == oofevents.TeamBlue || team == oofevents.TeamOrange
}
