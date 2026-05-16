package momentum

import "OOF_RL/internal/oofevents"

// Engine consumes typed game action events and maintains runtime-only momentum
// signals for future display/enrichment consumers.
type Engine struct {
	config Config
	state  MomentumState
}

// NewEngine creates an Engine with safe default values filled in.
func NewEngine(config Config) *Engine {
	if config.Decay <= 0 || config.Decay > 1 {
		config.Decay = DefaultConfig().Decay
	}

	e := &Engine{config: config}
	e.Reset()
	return e
}

// Reset clears runtime state without touching any external source of truth.
func (e *Engine) Reset() {
	e.state = MomentumState{
		Teams: map[oofevents.Team]TeamSignal{
			oofevents.TeamBlue:   {},
			oofevents.TeamOrange: {},
		},
	}
}

// Snapshot returns a copy of the current runtime-only state.
func (e *Engine) Snapshot() MomentumState {
	return cloneState(e.state)
}

// ApplyGameAction applies one typed game action and returns the updated state.
// Unknown actions or teams are ignored safely.
func (e *Engine) ApplyGameAction(event oofevents.GameActionEvent) MomentumState {
	impactTeam, delta, ok := e.deltaFor(event)
	if !ok {
		return e.Snapshot()
	}

	if guid := event.MatchGUID(); guid != "" && e.state.MatchGUID != "" && guid != e.state.MatchGUID {
		e.Reset()
	}
	if e.state.MatchGUID == "" {
		e.state.MatchGUID = event.MatchGUID()
	}

	e.decay()
	e.applyDelta(impactTeam, delta)
	e.state.Sequence++
	e.state.LastEvent = EventSignal{
		Action:     event.Action,
		ActorTeam:  event.Team,
		ImpactTeam: impactTeam,
		PlayerID:   event.PlayerID,
		PlayerName: event.PlayerName,
		VictimID:   event.VictimID,
		IsOwnGoal:  event.IsOwnGoal,
		IsEpicSave: event.IsEpicSave,
		OccurredAt: event.OccurredAt(),
		MatchGUID:  event.MatchGUID(),
	}

	return e.Snapshot()
}

func (e *Engine) deltaFor(event oofevents.GameActionEvent) (oofevents.Team, TeamSignal, bool) {
	if event.Team != oofevents.TeamBlue && event.Team != oofevents.TeamOrange {
		return "", TeamSignal{}, false
	}

	switch event.Action {
	case oofevents.ActionBallHit:
		return event.Team, TeamSignal{
			Pressure:            0.05,
			MomentumInfluence:   0.06,
			ContestInvolvement:  0.05,
			EventDerivedControl: 0.05,
			Confidence:          confidenceFor(event),
			Volatility:          0.02,
		}, true
	case oofevents.ActionShot:
		return event.Team, TeamSignal{
			Pressure:            0.22,
			MomentumInfluence:   0.20,
			ContestInvolvement:  0.12,
			EventDerivedControl: 0.14,
			Confidence:          confidenceFor(event) + 0.03,
			Volatility:          0.08,
		}, true
	case oofevents.ActionSave:
		delta := TeamSignal{
			Pressure:            0.08,
			MomentumInfluence:   0.14,
			ContestInvolvement:  0.20,
			EventDerivedControl: 0.10,
			Confidence:          confidenceFor(event) + 0.04,
			Volatility:          0.10,
		}
		if event.IsEpicSave {
			delta.MomentumInfluence += 0.06
			delta.ContestInvolvement += 0.05
			delta.Confidence += 0.02
			delta.Volatility += 0.04
		}
		return event.Team, delta, true
	case oofevents.ActionGoal:
		impactTeam := event.Team
		if event.IsOwnGoal {
			impactTeam = oofevents.Opponent(event.Team)
			if impactTeam == "" {
				return "", TeamSignal{}, false
			}
		}
		return impactTeam, TeamSignal{
			Pressure:            0.34,
			MomentumInfluence:   0.36,
			ContestInvolvement:  0.10,
			EventDerivedControl: 0.18,
			Confidence:          confidenceFor(event) + 0.06,
			Volatility:          0.20,
		}, true
	case oofevents.ActionAssist:
		return event.Team, TeamSignal{
			Pressure:            0.12,
			MomentumInfluence:   0.14,
			ContestInvolvement:  0.10,
			EventDerivedControl: 0.12,
			Confidence:          confidenceFor(event) + 0.02,
			Volatility:          0.04,
		}, true
	case oofevents.ActionDemo:
		return event.Team, TeamSignal{
			Pressure:            0.10,
			MomentumInfluence:   0.12,
			ContestInvolvement:  0.16,
			EventDerivedControl: 0.08,
			Confidence:          confidenceFor(event),
			Volatility:          0.14,
		}, true
	default:
		return "", TeamSignal{}, false
	}
}

func (e *Engine) decay() {
	for team, signal := range e.state.Teams {
		e.state.Teams[team] = TeamSignal{
			Pressure:            clamp01(signal.Pressure * e.config.Decay),
			MomentumInfluence:   clamp01(signal.MomentumInfluence * e.config.Decay),
			ContestInvolvement:  clamp01(signal.ContestInvolvement * e.config.Decay),
			EventDerivedControl: clamp01(signal.EventDerivedControl * e.config.Decay),
			Confidence:          clamp01(signal.Confidence * e.config.Decay),
			Volatility:          clamp01(signal.Volatility * e.config.Decay),
		}
	}
}

func (e *Engine) applyDelta(team oofevents.Team, delta TeamSignal) {
	signal := e.state.Teams[team]
	e.state.Teams[team] = TeamSignal{
		Pressure:            clamp01(signal.Pressure + delta.Pressure),
		MomentumInfluence:   clamp01(signal.MomentumInfluence + delta.MomentumInfluence),
		ContestInvolvement:  clamp01(signal.ContestInvolvement + delta.ContestInvolvement),
		EventDerivedControl: clamp01(signal.EventDerivedControl + delta.EventDerivedControl),
		Confidence:          clamp01(signal.Confidence + delta.Confidence),
		Volatility:          clamp01(signal.Volatility + delta.Volatility),
	}
}

func confidenceFor(event oofevents.GameActionEvent) float64 {
	confidence := 0.08
	if event.PlayerID != "" {
		confidence += 0.04
	}
	if event.PlayerName != "" {
		confidence += 0.02
	}
	if event.Action == oofevents.ActionDemo && event.VictimID != "" {
		confidence += 0.02
	}
	return confidence
}

func cloneState(state MomentumState) MomentumState {
	clone := state
	clone.Teams = make(map[oofevents.Team]TeamSignal, len(state.Teams))
	for team, signal := range state.Teams {
		clone.Teams[team] = signal
	}
	return clone
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
