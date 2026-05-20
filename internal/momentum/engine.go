package momentum

import (
	"math"
	"time"

	"OOF_RL/internal/oofevents"
)

// Engine consumes typed game action events and maintains runtime-only momentum
// signals for future display/enrichment consumers.
type Engine struct {
	config         Config
	state          MomentumState
	recent         []recentAction
	lastTouchTeam  oofevents.Team
	touchChains    map[oofevents.Team]int
	lastTouchTimes map[oofevents.Team]time.Time
}

type recentAction struct {
	action oofevents.ActionKind
	team   oofevents.Team
	at     time.Time
}

type teamDelta struct {
	team   oofevents.Team
	signal TeamSignal
}

// NewEngine creates an Engine with safe default values filled in.
func NewEngine(config Config) *Engine {
	config = normalizeConfig(config)

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
	e.recent = nil
	e.lastTouchTeam = ""
	e.touchChains = map[oofevents.Team]int{
		oofevents.TeamBlue:   0,
		oofevents.TeamOrange: 0,
	}
	e.lastTouchTimes = make(map[oofevents.Team]time.Time)
}

// Snapshot returns a copy of the current runtime-only state.
func (e *Engine) Snapshot() MomentumState {
	return cloneState(e.state)
}

// ApplyGameAction applies one typed game action and returns the updated state.
// Unknown actions or teams are ignored safely.
func (e *Engine) ApplyGameAction(event oofevents.GameActionEvent) MomentumState {
	impactTeam, deltas, ok := e.deltasFor(event)
	if !ok {
		return e.Snapshot()
	}

	if guid := event.MatchGUID(); guid != "" && e.state.MatchGUID != "" && guid != e.state.MatchGUID {
		e.Reset()
	}
	if e.state.MatchGUID == "" {
		e.state.MatchGUID = event.MatchGUID()
	}

	e.decayTo(event.OccurredAt())
	for _, delta := range deltas {
		e.applyDelta(delta.team, delta.signal)
	}
	e.rememberRecent(event)
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

// Tick applies elapsed-time decay without requiring a new game action. This
// mirrors the old PR #47 polling path while keeping the new typed-event stack.
func (e *Engine) Tick(now time.Time) MomentumState {
	e.decayTo(now)
	return e.Snapshot()
}

func (e *Engine) deltasFor(event oofevents.GameActionEvent) (oofevents.Team, []teamDelta, bool) {
	if event.Team != oofevents.TeamBlue && event.Team != oofevents.TeamOrange {
		return "", nil, false
	}

	switch event.Action {
	case oofevents.ActionBallHit:
		return event.Team, e.ballHitDeltas(event), true
	case oofevents.ActionShot:
		pressure := e.config.ShotPressure
		if e.recentDemoByTeam(event.Team, event.OccurredAt(), e.config.DemoBeforeShotWindow) {
			pressure += e.config.DemoBeforeShotPressureBonus
		}
		return event.Team, []teamDelta{{
			team: event.Team,
			signal: signalFromWeights(
				e.config.ShotControl,
				pressure,
				0.12,
				0.18,
				0.08,
			),
		}}, true
	case oofevents.ActionSave:
		defendingControl := e.config.SaveDefendingControl
		contest := 0.20
		confidence := 0.14
		volatility := 0.10
		if event.IsEpicSave {
			defendingControl += e.config.EpicSaveControlBonus
			contest += e.config.EpicSaveContestBonus
			confidence += e.config.EpicSaveConfidenceBonus
			volatility += e.config.EpicSaveVolatilityBonus
		}
		deltas := []teamDelta{{
			team: event.Team,
			signal: signalFromWeights(
				defendingControl,
				0,
				contest,
				confidence,
				volatility,
			),
		}}
		if attacking := oofevents.Opponent(event.Team); attacking != "" {
			deltas = append(deltas, teamDelta{
				team: attacking,
				signal: signalFromWeights(
					0,
					e.config.SaveForcedAttackingPressure,
					0.08,
					0,
					0.04,
				),
			})
		}
		return event.Team, deltas, true
	case oofevents.ActionGoal:
		impactTeam := event.Team
		if event.IsOwnGoal {
			impactTeam = oofevents.Opponent(event.Team)
			if impactTeam == "" {
				return "", nil, false
			}
		}
		pressure := e.config.GoalScoringPressure
		if e.recentDemoByTeam(impactTeam, event.OccurredAt(), e.config.DemoBeforeGoalWindow) {
			pressure += e.config.DemoBeforeGoalPressureBonus
		}
		return impactTeam, []teamDelta{{
			team: impactTeam,
			signal: signalFromWeights(
				e.config.GoalScoringControl,
				pressure,
				0.10,
				0.38,
				0.20,
			),
		}}, true
	case oofevents.ActionAssist:
		return event.Team, []teamDelta{{
			team: event.Team,
			signal: signalFromWeights(
				0.04,
				e.config.AssistPressure,
				0.10,
				e.config.AssistConfidenceBonus,
				0.04,
			),
		}}, true
	case oofevents.ActionDemo:
		return event.Team, []teamDelta{{
			team: event.Team,
			signal: signalFromWeights(
				0.02,
				e.config.DemoPressure,
				0.16,
				0.06,
				0.14,
			),
		}}, true
	default:
		return "", nil, false
	}
}

func (e *Engine) ballHitDeltas(event oofevents.GameActionEvent) []teamDelta {
	control := e.config.BallHitControl
	pressure := e.config.BallHitPressure
	volatility := 0.02
	deltas := make([]teamDelta, 0, 2)
	now := event.OccurredAt()

	if e.lastTouchTeam == event.Team && !e.lastTouchTimes[event.Team].IsZero() && within(now, e.lastTouchTimes[event.Team], e.config.TouchChainWindow) {
		e.touchChains[event.Team]++
		chainBonus := math.Min(e.config.MaxTouchChainBonus, float64(e.touchChains[event.Team]-1)*e.config.SameTeamTouchControlBonus)
		control += chainBonus
		pressure += math.Min(e.config.MaxTouchChainBonus, float64(e.touchChains[event.Team]-1)*e.config.SameTeamTouchPressureBonus)
		confidence := 0.12 + math.Min(0.16, float64(e.touchChains[event.Team])*0.03)
		deltas = append(deltas, teamDelta{
			team: event.Team,
			signal: TeamSignal{
				Confidence: confidence,
			},
		})
	} else {
		if e.lastTouchTeam != "" && e.lastTouchTeam != event.Team {
			deltas = append(deltas, teamDelta{
				team: e.lastTouchTeam,
				signal: TeamSignal{
					MomentumInfluence:   e.config.OpponentTouchPreviousPenalty,
					EventDerivedControl: e.config.OpponentTouchPreviousPenalty,
				},
			})
			control += e.config.OpponentTouchNewControl
			if e.anyRecentTouchByOpponent(event.Team, now, e.config.AlternatingTouchWindow) {
				volatility += e.config.AlternatingTouchVolatilityBonus
			}
		}
		e.touchChains[event.Team] = 1
		deltas = append(deltas, teamDelta{
			team: event.Team,
			signal: TeamSignal{
				Confidence: 0.08,
			},
		})
	}

	e.lastTouchTeam = event.Team
	e.lastTouchTimes[event.Team] = now
	deltas = append(deltas, teamDelta{
		team: event.Team,
		signal: signalFromWeights(
			control,
			pressure,
			0.05,
			0,
			volatility,
		),
	})
	return deltas
}

func (e *Engine) decayTo(now time.Time) {
	if e.state.LastEvent.OccurredAt.IsZero() || !now.After(e.state.LastEvent.OccurredAt) {
		return
	}
	seconds := now.Sub(e.state.LastEvent.OccurredAt).Seconds()
	for team, signal := range e.state.Teams {
		e.state.Teams[team] = TeamSignal{
			Pressure:            max0(signal.Pressure * math.Pow(e.config.PressureDecayPerSecond, seconds)),
			MomentumInfluence:   max0(signal.MomentumInfluence * math.Pow(e.config.ControlDecayPerSecond, seconds)),
			ContestInvolvement:  max0(signal.ContestInvolvement * math.Pow(e.config.VolatilityDecayPerSecond, seconds)),
			EventDerivedControl: max0(signal.EventDerivedControl * math.Pow(e.config.ControlDecayPerSecond, seconds)),
			Confidence:          clamp01(signal.Confidence * math.Pow(e.config.ConfidenceDecayPerSecond, seconds)),
			Volatility:          max0(signal.Volatility * math.Pow(e.config.VolatilityDecayPerSecond, seconds)),
		}
	}
}

func (e *Engine) applyDelta(team oofevents.Team, delta TeamSignal) {
	signal := e.state.Teams[team]
	e.state.Teams[team] = TeamSignal{
		Pressure:            max0(signal.Pressure + delta.Pressure),
		MomentumInfluence:   max0(signal.MomentumInfluence + delta.MomentumInfluence),
		ContestInvolvement:  max0(signal.ContestInvolvement + delta.ContestInvolvement),
		EventDerivedControl: max0(signal.EventDerivedControl + delta.EventDerivedControl),
		Confidence:          clamp01(signal.Confidence + delta.Confidence),
		Volatility:          max0(signal.Volatility + delta.Volatility),
	}
}

func confidenceFor(event oofevents.GameActionEvent, config Config) float64 {
	confidence := config.ConfidenceBase
	if event.PlayerID != "" {
		confidence += config.ConfidencePlayerID
	}
	if event.PlayerName != "" {
		confidence += config.ConfidencePlayerName
	}
	if event.Action == oofevents.ActionDemo && event.VictimID != "" {
		confidence += config.ConfidenceDemoVictim
	}
	return confidence
}

func signalFromWeights(control, pressure, contest, confidence, volatility float64) TeamSignal {
	return TeamSignal{
		Pressure:            max0(pressure),
		MomentumInfluence:   max0(control + pressure),
		ContestInvolvement:  max0(contest),
		EventDerivedControl: max0(control),
		Confidence:          clamp01(confidence),
		Volatility:          max0(volatility),
	}
}

func (e *Engine) rememberRecent(event oofevents.GameActionEvent) {
	e.recent = append(e.recent, recentAction{
		action: event.Action,
		team:   event.Team,
		at:     event.OccurredAt(),
	})
	cutoff := event.OccurredAt().Add(-e.config.DemoBeforeGoalWindow)
	first := 0
	for first < len(e.recent) && e.recent[first].at.Before(cutoff) {
		first++
	}
	if first > 0 {
		e.recent = append([]recentAction(nil), e.recent[first:]...)
	}
}

func (e *Engine) recentDemoByTeam(team oofevents.Team, now time.Time, window time.Duration) bool {
	for i := len(e.recent) - 1; i >= 0; i-- {
		action := e.recent[i]
		if !within(now, action.at, window) {
			break
		}
		if action.action == oofevents.ActionDemo && action.team == team {
			return true
		}
	}
	return false
}

func (e *Engine) anyRecentTouchByOpponent(team oofevents.Team, now time.Time, window time.Duration) bool {
	for i := len(e.recent) - 1; i >= 0; i-- {
		action := e.recent[i]
		if !within(now, action.at, window) {
			break
		}
		if action.action == oofevents.ActionBallHit && action.team != "" && action.team != team {
			return true
		}
	}
	return false
}

func within(now, then time.Time, window time.Duration) bool {
	if then.IsZero() || window <= 0 {
		return false
	}
	diff := now.Sub(then)
	return diff >= 0 && diff <= window
}

func normalizeConfig(config Config) Config {
	defaults := DefaultConfig()
	if config.Decay < 0 || config.Decay > 1 {
		config.Decay = 0
	}
	decayFallback := config.Decay
	if decayFallback == 0 {
		decayFallback = defaults.ControlDecayPerSecond
	}
	fillFloat(&config.ControlDecayPerSecond, decayFallback)
	if config.Decay > 0 {
		fillFloat(&config.PressureDecayPerSecond, config.Decay)
		fillFloat(&config.VolatilityDecayPerSecond, config.Decay)
		fillFloat(&config.ConfidenceDecayPerSecond, config.Decay)
	} else {
		fillFloat(&config.PressureDecayPerSecond, defaults.PressureDecayPerSecond)
		fillFloat(&config.VolatilityDecayPerSecond, defaults.VolatilityDecayPerSecond)
		fillFloat(&config.ConfidenceDecayPerSecond, defaults.ConfidenceDecayPerSecond)
	}
	fillFloat(&config.ControlThreshold, defaults.ControlThreshold)
	fillFloat(&config.PressureThreshold, defaults.PressureThreshold)
	fillFloat(&config.ConfidenceThreshold, defaults.ConfidenceThreshold)
	fillFloat(&config.VolatilityThreshold, defaults.VolatilityThreshold)
	fillFloat(&config.PressureShareThreshold, defaults.PressureShareThreshold)
	fillFloat(&config.ControlShareThreshold, defaults.ControlShareThreshold)
	if config.TouchChainWindow <= 0 {
		config.TouchChainWindow = defaults.TouchChainWindow
	}
	if config.AlternatingTouchWindow <= 0 {
		config.AlternatingTouchWindow = defaults.AlternatingTouchWindow
	}
	if config.DemoBeforeShotWindow <= 0 {
		config.DemoBeforeShotWindow = defaults.DemoBeforeShotWindow
	}
	if config.DemoBeforeGoalWindow <= 0 {
		config.DemoBeforeGoalWindow = defaults.DemoBeforeGoalWindow
	}
	fillFloat(&config.BallHitControl, defaults.BallHitControl)
	fillFloat(&config.BallHitPressure, defaults.BallHitPressure)
	fillFloat(&config.SameTeamTouchControlBonus, defaults.SameTeamTouchControlBonus)
	fillFloat(&config.SameTeamTouchPressureBonus, defaults.SameTeamTouchPressureBonus)
	fillFloat(&config.MaxTouchChainBonus, defaults.MaxTouchChainBonus)
	fillFloat(&config.OpponentTouchNewControl, defaults.OpponentTouchNewControl)
	if config.OpponentTouchPreviousPenalty == 0 {
		config.OpponentTouchPreviousPenalty = defaults.OpponentTouchPreviousPenalty
	}
	fillFloat(&config.ShotControl, defaults.ShotControl)
	fillFloat(&config.ShotPressure, defaults.ShotPressure)
	fillFloat(&config.SaveDefendingControl, defaults.SaveDefendingControl)
	fillFloat(&config.SaveForcedAttackingPressure, defaults.SaveForcedAttackingPressure)
	fillFloat(&config.EpicSaveControlBonus, defaults.EpicSaveControlBonus)
	fillFloat(&config.EpicSaveContestBonus, defaults.EpicSaveContestBonus)
	fillFloat(&config.EpicSaveConfidenceBonus, defaults.EpicSaveConfidenceBonus)
	fillFloat(&config.EpicSaveVolatilityBonus, defaults.EpicSaveVolatilityBonus)
	fillFloat(&config.GoalScoringControl, defaults.GoalScoringControl)
	fillFloat(&config.GoalScoringPressure, defaults.GoalScoringPressure)
	fillFloat(&config.AssistPressure, defaults.AssistPressure)
	fillFloat(&config.AssistConfidenceBonus, defaults.AssistConfidenceBonus)
	fillFloat(&config.DemoPressure, defaults.DemoPressure)
	fillFloat(&config.DemoBeforeShotPressureBonus, defaults.DemoBeforeShotPressureBonus)
	fillFloat(&config.DemoBeforeGoalPressureBonus, defaults.DemoBeforeGoalPressureBonus)
	fillFloat(&config.AlternatingTouchVolatilityBonus, defaults.AlternatingTouchVolatilityBonus)
	fillFloat(&config.ConfidenceBase, defaults.ConfidenceBase)
	fillFloat(&config.ConfidencePlayerID, defaults.ConfidencePlayerID)
	fillFloat(&config.ConfidencePlayerName, defaults.ConfidencePlayerName)
	fillFloat(&config.ConfidenceDemoVictim, defaults.ConfidenceDemoVictim)
	return config
}

func fillFloat(value *float64, fallback float64) {
	if *value == 0 {
		*value = fallback
	}
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

func max0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}
