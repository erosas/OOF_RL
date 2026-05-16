package momentum

import (
	"math"
)

const epsilon = 0.000001

type EventPressureEngine struct {
	cfg          Config
	state        MomentumState
	events       []NormalizedGameEvent
	reasons      []string
	weights      []WeightApplied
	debugReasons []string
	debugWeights []WeightApplied
	currentEvent EventType
	eventCounts  map[EventType]int
	sourceCounts map[string]int
	lastStrong   *NormalizedGameEvent
}

func NewEventPressureEngine(cfg Config) *EventPressureEngine {
	if cfg.MaxEvents <= 0 {
		cfg = DefaultConfig()
	}
	return &EventPressureEngine{
		cfg: cfg,
		state: MomentumState{
			CurrentState: StateNeutral,
		},
	}
}

func (e *EventPressureEngine) Reset() {
	e.state = MomentumState{CurrentState: StateNeutral}
	e.events = nil
	e.reasons = nil
	e.weights = nil
	e.debugReasons = nil
	e.debugWeights = nil
	e.currentEvent = ""
	e.eventCounts = nil
	e.sourceCounts = nil
	e.lastStrong = nil
}

func (e *EventPressureEngine) ProcessEvent(ev NormalizedGameEvent) MomentumFlowOutput {
	e.reasons = nil
	e.weights = nil
	e.currentEvent = ev.Type
	defer func() { e.currentEvent = "" }()
	e.decayTo(ev.Time)
	e.appendEvent(ev)
	e.state.LastEventTime = &ev.Time

	switch ev.Type {
	case EventBallHit:
		e.applyBallHit(ev)
	case EventShot:
		e.applyShot(ev)
	case EventSave:
		e.applySave(ev)
	case EventGoal:
		e.applyGoal(ev)
	case EventAssist:
		e.applyAssist(ev)
	case EventDemo:
		e.applyDemo(ev)
	}

	e.state.Confidence = clamp01(e.state.Confidence)
	e.classify()
	e.debugReasons = append([]string(nil), e.reasons...)
	e.debugWeights = append([]WeightApplied(nil), e.weights...)
	return e.Output()
}

func (e *EventPressureEngine) Tick(now int64) MomentumFlowOutput {
	e.reasons = nil
	e.weights = nil
	e.decayTo(now)
	e.expirePulse(now)
	e.classify()
	return e.Output()
}

func (e *EventPressureEngine) Output() MomentumFlowOutput {
	blueControlShare := share(e.state.Blue.Control, e.state.Blue.Control+e.state.Orange.Control)
	orangeControlShare := share(e.state.Orange.Control, e.state.Blue.Control+e.state.Orange.Control)
	bluePressureShare := share(e.state.Blue.Pressure, e.state.Blue.Pressure+e.state.Orange.Pressure)
	orangePressureShare := share(e.state.Orange.Pressure, e.state.Blue.Pressure+e.state.Orange.Pressure)

	blueTotal := e.state.Blue.Control + e.state.Blue.Pressure
	orangeTotal := e.state.Orange.Control + e.state.Orange.Pressure
	blueMomentum := share(blueTotal, blueTotal+orangeTotal)
	orangeMomentum := 1.0 - blueMomentum
	if blueTotal+orangeTotal <= epsilon {
		blueMomentum = 0.5
		orangeMomentum = 0.5
	}

	var dominant *Team
	var glowTeam *Team
	var clockTeam *Team
	switch e.state.CurrentState {
	case StateBlueControl, StateBluePressure:
		t := TeamBlue
		dominant = &t
		glowTeam = &t
		clockTeam = &t
	case StateOrangeControl, StateOrangePressure:
		t := TeamOrange
		dominant = &t
		glowTeam = &t
		clockTeam = &t
	}

	intensity := clamp01(math.Max(math.Abs(blueMomentum-orangeMomentum), math.Max(e.state.Blue.Pressure, e.state.Orange.Pressure)/10.0) * e.state.Confidence)
	clockIntensity := intensity
	if e.state.CurrentState == StateBlueControl || e.state.CurrentState == StateOrangeControl {
		clockIntensity *= 0.65
	}
	if e.state.CurrentState == StateNeutral {
		intensity = 0
		clockIntensity = 0
	}

	return MomentumFlowOutput{
		State: e.state.CurrentState,
		Blue: TeamOutput{
			Control:       round(e.state.Blue.Control),
			Pressure:      round(e.state.Blue.Pressure),
			ControlShare:  round(blueControlShare),
			PressureShare: round(bluePressureShare),
		},
		Orange: TeamOutput{
			Control:       round(e.state.Orange.Control),
			Pressure:      round(e.state.Orange.Pressure),
			ControlShare:  round(orangeControlShare),
			PressureShare: round(orangePressureShare),
		},
		Confidence:   round(e.state.Confidence),
		Volatility:   round(e.state.Volatility),
		DominantTeam: dominant,
		Overlay: OverlayOutput{
			ScoreboardGlowTeam:       glowTeam,
			ScoreboardGlowIntensity:  round(intensity),
			ClockRingTeam:            clockTeam,
			ClockRingIntensity:       round(clockIntensity),
			MomentumBarBluePercent:   round(blueMomentum * 100),
			MomentumBarOrangePercent: round(orangeMomentum * 100),
			Pulse:                    e.state.LastPulse,
			PulseTeam:                e.state.LastPulseTeam,
		},
		Debug: &DebugOutput{
			LastEvents:      e.lastEvents(10),
			LastStrongEvent: e.copyLastStrong(),
			EventCounts:     copyEventCounts(e.eventCounts),
			SourceCounts:    copySourceCounts(e.sourceCounts),
			Reasons:         append([]string(nil), e.debugReasons...),
			WeightsApplied:  append([]WeightApplied(nil), e.debugWeights...),
		},
	}
}

func (e *EventPressureEngine) applyBallHit(ev NormalizedGameEvent) {
	teamState := e.teamState(ev.Team)
	if teamState == nil {
		return
	}

	controlDelta := e.cfg.BallHitControl
	pressureDelta := e.cfg.BallHitPressure
	if e.state.LastTouchTeam == ev.Team && teamState.LastTouchTime != nil && ev.Time-*teamState.LastTouchTime <= e.cfg.TouchChainWindowMs {
		teamState.TouchChain++
		chainBonus := math.Min(e.cfg.MaxChainBonus, float64(teamState.TouchChain-1)*e.cfg.SameTeamTouchControlBonus)
		controlDelta += chainBonus
		pressureDelta += math.Min(e.cfg.MaxChainBonus, float64(teamState.TouchChain-1)*e.cfg.SameTeamTouchPressureBonus)
		e.addReason("same-team touch chain increased event control")
		e.bumpConfidence(0.12 + math.Min(0.16, float64(teamState.TouchChain)*0.03))
	} else {
		if e.state.LastTouchTeam != "" && e.state.LastTouchTeam != ev.Team {
			prev := e.teamState(e.state.LastTouchTeam)
			if prev != nil {
				e.addControl(e.state.LastTouchTeam, e.cfg.OpponentTouchPreviousPenalty, "opponent touch after chain reduces previous control")
			}
			controlDelta += e.cfg.OpponentTouchNewControl
			if e.anyRecentTouch(e.cfg.AlternatingTouchWindowMs, ev.Team, ev.Time) {
				e.addVolatility(e.cfg.AlternatingTouchVolatilityBonus, "rapid alternating touches")
				e.setPulse(PulseVolatileContest, "", ev.Time)
			}
			e.addReason("opponent touch contested previous chain")
		}
		teamState.TouchChain = 1
		e.bumpConfidence(0.08)
	}

	now := ev.Time
	teamState.LastTouchTime = &now
	e.state.LastTouchTeam = ev.Team
	e.addControl(ev.Team, controlDelta, "ball hit event control")
	e.addPressure(ev.Team, pressureDelta, "ball hit light pressure")
}

func (e *EventPressureEngine) applyShot(ev NormalizedGameEvent) {
	e.addControl(ev.Team, e.cfg.ShotControl, "shot control bump")
	e.addPressure(ev.Team, e.cfg.ShotPressure, "shot pressure")
	if e.recentDemoByTeam(ev.Team, ev.Time, e.cfg.DemoBeforeShotWindowMs) {
		e.addPressure(ev.Team, e.cfg.DemoBeforeShotPressureBonus, "same-team demo before shot pressure sequence")
		e.setPulse(PulseDemoPressure, ev.Team, ev.Time)
	} else {
		e.setPulse(PulseShot, ev.Team, ev.Time)
	}
	e.markStrong(ev.Team, ev.Time)
	e.bumpConfidence(0.18)
}

func (e *EventPressureEngine) applySave(ev NormalizedGameEvent) {
	attackingTeam := Opponent(ev.Team)
	e.addControl(ev.Team, e.cfg.SaveDefendingControl, "save defensive control")
	if attackingTeam != "" {
		e.addPressure(attackingTeam, e.cfg.SaveForcedAttackingPressure, "save forced by attacking pressure")
	}
	e.setPulse(PulseSaveForced, attackingTeam, ev.Time)
	e.markStrong(ev.Team, ev.Time)
	e.bumpConfidence(0.14)
}

func (e *EventPressureEngine) applyGoal(ev NormalizedGameEvent) {
	e.addControl(ev.Team, e.cfg.GoalScoringControl, "goal scoring control bump")
	e.addPressure(ev.Team, e.cfg.GoalScoringPressure, "goal pressure burst")
	if e.recentDemoByTeam(ev.Team, ev.Time, e.cfg.DemoBeforeGoalWindowMs) {
		e.addPressure(ev.Team, e.cfg.DemoBeforeGoalPressureBonus, "same-team demo before goal pressure sequence")
	}
	e.setPulse(PulseGoalBurst, ev.Team, ev.Time)
	e.markStrong(ev.Team, ev.Time)
	e.bumpConfidence(0.38)
}

func (e *EventPressureEngine) applyAssist(ev NormalizedGameEvent) {
	e.addPressure(ev.Team, e.cfg.AssistPressure, "assist validates attack chain")
	e.bumpConfidence(e.cfg.AssistConfidenceBonus)
}

func (e *EventPressureEngine) applyDemo(ev NormalizedGameEvent) {
	e.addPressure(ev.Team, e.cfg.DemoPressure, "demo light pressure")
	e.setPulse(PulseDemoPressure, ev.Team, ev.Time)
	e.markStrong(ev.Team, ev.Time)
	e.bumpConfidence(0.06)
}

func (e *EventPressureEngine) classify() {
	bluePressureShare := share(e.state.Blue.Pressure, e.state.Blue.Pressure+e.state.Orange.Pressure)
	orangePressureShare := share(e.state.Orange.Pressure, e.state.Blue.Pressure+e.state.Orange.Pressure)
	blueControlShare := share(e.state.Blue.Control, e.state.Blue.Control+e.state.Orange.Control)
	orangeControlShare := share(e.state.Orange.Control, e.state.Blue.Control+e.state.Orange.Control)
	strongPressureDominance := bluePressureShare > e.cfg.PressureShareThreshold || orangePressureShare > e.cfg.PressureShareThreshold

	switch {
	case e.state.Confidence < e.cfg.ConfidenceThreshold:
		e.state.CurrentState = StateNeutral
	case e.state.Volatility > e.cfg.VolatilityThreshold && !strongPressureDominance:
		e.state.CurrentState = StateVolatile
	case bluePressureShare > e.cfg.PressureShareThreshold && e.state.Blue.Pressure > e.cfg.PressureThreshold:
		e.state.CurrentState = StateBluePressure
	case orangePressureShare > e.cfg.PressureShareThreshold && e.state.Orange.Pressure > e.cfg.PressureThreshold:
		e.state.CurrentState = StateOrangePressure
	case blueControlShare > e.cfg.ControlShareThreshold && e.state.Blue.Control > e.cfg.ControlThreshold:
		e.state.CurrentState = StateBlueControl
	case orangeControlShare > e.cfg.ControlShareThreshold && e.state.Orange.Control > e.cfg.ControlThreshold:
		e.state.CurrentState = StateOrangeControl
	default:
		e.state.CurrentState = StateNeutral
	}
}

func (e *EventPressureEngine) decayTo(now int64) {
	if e.state.LastEventTime == nil || now <= *e.state.LastEventTime {
		return
	}
	seconds := float64(now-*e.state.LastEventTime) / 1000.0
	e.state.Blue.Control *= math.Pow(e.cfg.ControlDecayPerSecond, seconds)
	e.state.Orange.Control *= math.Pow(e.cfg.ControlDecayPerSecond, seconds)
	e.state.Blue.Pressure *= math.Pow(e.cfg.PressureDecayPerSecond, seconds)
	e.state.Orange.Pressure *= math.Pow(e.cfg.PressureDecayPerSecond, seconds)
	e.state.Volatility *= math.Pow(e.cfg.VolatilityDecayPerSecond, seconds)
	e.state.Confidence *= math.Pow(e.cfg.ConfidenceDecayPerSecond, seconds)
	e.pruneEvents(now)
}

func (e *EventPressureEngine) teamState(team Team) *TeamMomentumState {
	switch team {
	case TeamBlue:
		return &e.state.Blue
	case TeamOrange:
		return &e.state.Orange
	default:
		return nil
	}
}

func (e *EventPressureEngine) addControl(team Team, delta float64, reason string) {
	if state := e.teamState(team); state != nil {
		state.Control = math.Max(0, state.Control+delta)
		e.recordWeight(e.weightEventType(), team, &delta, nil, nil, reason)
	}
}

func (e *EventPressureEngine) addPressure(team Team, delta float64, reason string) {
	if state := e.teamState(team); state != nil {
		state.Pressure = math.Max(0, state.Pressure+delta)
		e.recordWeight(e.weightEventType(), team, nil, &delta, nil, reason)
	}
}

func (e *EventPressureEngine) addVolatility(delta float64, reason string) {
	e.state.Volatility = math.Max(0, e.state.Volatility+delta)
	e.recordWeight(e.weightEventType(), "", nil, nil, &delta, reason)
}

func (e *EventPressureEngine) weightEventType() EventType {
	if e.currentEvent != "" {
		return e.currentEvent
	}
	return EventBallHit
}

func (e *EventPressureEngine) markStrong(team Team, now int64) {
	if state := e.teamState(team); state != nil {
		state.LastStrongEventTime = &now
	}
	if len(e.events) > 0 {
		ev := e.events[len(e.events)-1]
		e.lastStrong = &ev
	}
}

func (e *EventPressureEngine) bumpConfidence(delta float64) {
	e.state.Confidence = clamp01(e.state.Confidence + delta)
}

func (e *EventPressureEngine) appendEvent(ev NormalizedGameEvent) {
	e.events = append(e.events, ev)
	if len(e.events) > e.cfg.MaxEvents {
		e.events = e.events[len(e.events)-e.cfg.MaxEvents:]
	}
	if e.eventCounts == nil {
		e.eventCounts = make(map[EventType]int)
	}
	if e.sourceCounts == nil {
		e.sourceCounts = make(map[string]int)
	}
	e.eventCounts[ev.Type]++
	source := ev.SourceEvent
	if source == "" {
		source = "unknown"
	}
	e.sourceCounts[source]++
}

func (e *EventPressureEngine) pruneEvents(now int64) {
	cutoff := now - e.cfg.LongWindowMs
	first := 0
	for first < len(e.events) && e.events[first].Time < cutoff {
		first++
	}
	if first > 0 {
		e.events = append([]NormalizedGameEvent(nil), e.events[first:]...)
	}
}

func (e *EventPressureEngine) lastEvents(n int) []NormalizedGameEvent {
	if len(e.events) <= n {
		return append([]NormalizedGameEvent(nil), e.events...)
	}
	return append([]NormalizedGameEvent(nil), e.events[len(e.events)-n:]...)
}

func (e *EventPressureEngine) anyRecentTouch(windowMs int64, currentTeam Team, now int64) bool {
	for i := len(e.events) - 1; i >= 0; i-- {
		ev := e.events[i]
		if now-ev.Time > windowMs {
			break
		}
		if ev.Type == EventBallHit && ev.Team != "" && ev.Team != currentTeam {
			return true
		}
	}
	return false
}

func (e *EventPressureEngine) recentDemoByTeam(team Team, now, windowMs int64) bool {
	for i := len(e.events) - 1; i >= 0; i-- {
		ev := e.events[i]
		if now-ev.Time > windowMs {
			break
		}
		if ev.Type == EventDemo && ev.Team == team {
			return true
		}
	}
	return false
}

func (e *EventPressureEngine) addReason(reason string) {
	e.reasons = append(e.reasons, reason)
}

func (e *EventPressureEngine) recordWeight(eventType EventType, team Team, control, pressure, volatility *float64, reason string) {
	e.weights = append(e.weights, WeightApplied{
		EventType:       eventType,
		Team:            team,
		ControlDelta:    control,
		PressureDelta:   pressure,
		VolatilityDelta: volatility,
		Reason:          reason,
	})
	e.addReason(reason)
}

func (e *EventPressureEngine) setPulse(pulse OverlayPulse, team Team, now int64) {
	e.state.LastPulse = pulse
	e.state.LastPulseTeam = team
	t := now
	e.state.LastPulseTime = &t
}

func (e *EventPressureEngine) expirePulse(now int64) {
	if e.state.LastPulseTime == nil || e.state.LastPulse == "" {
		return
	}
	if now-*e.state.LastPulseTime <= e.cfg.PulseHoldMs {
		return
	}
	e.state.LastPulse = ""
	e.state.LastPulseTeam = ""
	e.state.LastPulseTime = nil
}

func (e *EventPressureEngine) copyLastStrong() *NormalizedGameEvent {
	if e.lastStrong == nil {
		return nil
	}
	cp := *e.lastStrong
	return &cp
}

func copyEventCounts(in map[EventType]int) map[EventType]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[EventType]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copySourceCounts(in map[string]int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func share(value, total float64) float64 {
	if total <= epsilon {
		return 0.5
	}
	return clamp01(value / total)
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func round(v float64) float64 {
	return math.Round(v*1000) / 1000
}
