package overlayhud

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js momentum-flow-bar.css momentum-control-wheel.css casey-bar-down111.png
var viewFS embed.FS

const overlayPerfFrontendReportTTLMS = int64(15000)
const overlayEventQueueSize = 512

// Plugin is a read-only overlay design lab. It renders HUD components and
// exposes deterministic event-pressure state without mutating core match data.
type Plugin struct {
	plugin.BasePlugin
	mu              sync.Mutex
	engine          *momentum.EventPressureEngine
	updateDeltas    *momentum.UpdateStateDeltaNormalizer
	explicitEvents  []recentMomentumEvent
	subs            []oofevents.Subscription
	playerCacheGUID string
	playerRefs      map[string]events.PlayerRef
	replayActive    bool
	replayFileMode  bool
	replayChanged   int64
	momentumResetAt int64
	goalResetDue    int64
	goalReplaySeen  bool
	prefs           overlayPrefs
	perf            overlayPerfCounters
	nativeHUD       nativeHUDState
	eventQueue      chan oofevents.OOFEvent
	eventStop       chan struct{}
	eventDone       chan struct{}
	eventQueueDrops atomic.Uint64
}

func New() *Plugin {
	return &Plugin{
		engine:       momentum.NewEventPressureEngine(momentum.DefaultConfig()),
		updateDeltas: momentum.NewUpdateStateDeltaNormalizer(),
		playerRefs:   make(map[string]events.PlayerRef),
	}
}

func (p *Plugin) ID() string         { return "overlayhud" }
func (p *Plugin) DBPrefix() string   { return "" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "overlay", Label: "Overlay Lab", Order: 88}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/overlay/momentum", p.handleMomentum)
	mux.HandleFunc("/api/overlay/momentum/reset", p.handleMomentumReset)
	mux.HandleFunc("/api/overlay/prefs", p.handlePrefs)
	mux.HandleFunc("/api/overlay/perf", p.handlePerf)
	mux.HandleFunc("/api/overlay/perf/frontend", p.handlePerfFrontend)
	mux.HandleFunc("/api/overlay/hud/native-visibility", p.handleNativeHUDVisibility)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.eventQueue = make(chan oofevents.OOFEvent, overlayEventQueueSize)
	p.eventStop = make(chan struct{})
	p.eventDone = make(chan struct{})
	go p.runEventWorker()
	p.subs = []oofevents.Subscription{
		bus.Subscribe(oofevents.TypeMatchStarted, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeMatchRestarted, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeStateUpdated, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeGoalScored, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeStatFeed, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeClockUpdated, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeBallHit, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeMatchEnded, p.enqueueOOFEvent),
		bus.Subscribe(oofevents.TypeMatchDestroyed, p.enqueueOOFEvent),
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	for _, sub := range p.subs {
		sub.Cancel()
	}
	p.subs = nil
	if p.eventStop != nil {
		close(p.eventStop)
		p.eventStop = nil
	}
	if p.eventDone != nil {
		<-p.eventDone
		p.eventDone = nil
	}
	p.eventQueue = nil
	return nil
}

func (p *Plugin) enqueueOOFEvent(e oofevents.OOFEvent) {
	if p.eventQueue == nil || p.eventStop == nil {
		return
	}
	select {
	case p.eventQueue <- e:
	case <-p.eventStop:
	default:
		p.eventQueueDrops.Add(1)
		// Keep the event-bus dispatch path non-blocking under overload.
	}
}

func (p *Plugin) runEventWorker() {
	defer close(p.eventDone)
	for {
		select {
		case e := <-p.eventQueue:
			p.HandleOOFEvent(e)
		case <-p.eventStop:
			return
		}
	}
}

func (p *Plugin) HandleEvent(env events.Envelope) {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handleEnvelopeLocked(env, now)
}

func (p *Plugin) HandleOOFEvent(e oofevents.OOFEvent) {
	now := time.Now()
	p.mu.Lock()
	defer p.mu.Unlock()
	p.handleOOFEventLocked(e, now)
}

func (p *Plugin) handleOOFEventLocked(e oofevents.OOFEvent, now time.Time) {
	nowMs := now.UnixMilli()
	p.perfCountLocked(nowMs, "oofevents."+e.Type(), 1)
	switch ev := oofevents.Unwrap(e).(type) {
	case oofevents.MatchStartedEvent:
		p.handleTypedMatchCreatedLocked(ev.MatchGUID(), now)
	case oofevents.MatchRestartedEvent:
		p.handleTypedMatchCreatedLocked(ev.MatchGUID(), now)
	case oofevents.MatchEndedEvent:
		p.handleTypedLifecycleLocked("MatchEnded", now)
	case oofevents.MatchDestroyedEvent:
		p.playerCacheGUID = ""
		p.playerRefs = make(map[string]events.PlayerRef)
		p.handleTypedLifecycleLocked("MatchDestroyed", now)
	case oofevents.ClockUpdatedEvent:
		p.perfCountLocked(nowMs, "envelope.ClockUpdatedSeconds", 1)
		p.applyGoalFallback(nowMs)
	case oofevents.StateUpdatedEvent:
		// Phase 1 keeps state snapshots on the legacy UpdateState path so replay
		// state and stat-delta baselines remain byte-for-byte behavior compatible.
		env, ok := p.envelopeFromOOFEventLocked(e)
		if !ok {
			p.perfCountLocked(nowMs, "adapter.drop."+e.Type(), 1)
			return
		}
		p.handleEnvelopeLocked(env, now)
	case oofevents.BallHitEvent:
		normalized, ok := p.normalizedBallHitFromOOFLocked(ev, now)
		if !ok {
			p.perfCountLocked(nowMs, "adapter.drop."+e.Type(), 1)
			return
		}
		p.handleTypedExplicitEventsLocked("BallHit", normalized, now)
	case oofevents.GoalScoredEvent:
		p.handleTypedExplicitEventsLocked("GoalScored", p.normalizedGoalScoredFromOOFLocked(ev, now), now)
	case oofevents.StatFeedEvent:
		p.handleTypedExplicitEventsLocked("StatfeedEvent", p.normalizedStatFeedFromOOFLocked(ev, now), now)
	default:
		p.perfCountLocked(nowMs, "adapter.drop."+e.Type(), 1)
	}
}

func (p *Plugin) handleEnvelopeLocked(env events.Envelope, now time.Time) {
	nowMs := now.UnixMilli()
	p.perfCountLocked(nowMs, "envelope."+env.Event, 1)
	p.rememberPlayersFromEnvelopeLocked(env)
	normalized := momentum.NormalizeEnvelope(env, now)
	normalized = p.filterDuplicateExplicitEvents(normalized, nowMs)
	p.updateReplayState(env, normalized, nowMs)
	p.applyGoalFallback(nowMs)
	for _, ev := range normalized {
		p.updateDeltas.MarkExplicit(ev)
	}
	deltaEvents := p.updateDeltas.NormalizeEnvelope(env, now)
	allEvents := append(normalized, deltaEvents...)
	p.noteGoalEvents(allEvents, nowMs)
	if len(allEvents) == 0 {
		return
	}
	p.processMomentumEventsLocked(allEvents, nowMs)
}

func (p *Plugin) handleTypedMatchCreatedLocked(matchGUID string, now time.Time) {
	nowMs := now.UnixMilli()
	p.perfCountLocked(nowMs, "envelope.MatchCreated", 1)
	if matchGUID != "" && matchGUID != p.playerCacheGUID {
		p.playerCacheGUID = matchGUID
		p.playerRefs = make(map[string]events.PlayerRef)
		p.perfCountLocked(nowMs, "cache.rebuild.match", 1)
	}
	p.updateReplayStateFromParts("MatchCreated", nil, nil, nowMs)
	p.applyGoalFallback(nowMs)
}

func (p *Plugin) handleTypedLifecycleLocked(eventName string, now time.Time) {
	nowMs := now.UnixMilli()
	p.perfCountLocked(nowMs, "envelope."+eventName, 1)
	p.updateReplayStateFromParts(eventName, nil, nil, nowMs)
	p.applyGoalFallback(nowMs)
}

func (p *Plugin) handleTypedExplicitEventsLocked(eventName string, normalized []momentum.NormalizedGameEvent, now time.Time) {
	nowMs := now.UnixMilli()
	p.perfCountLocked(nowMs, "envelope."+eventName, 1)
	normalized = p.filterDuplicateExplicitEvents(normalized, nowMs)
	p.updateReplayStateFromParts(eventName, nil, normalized, nowMs)
	p.applyGoalFallback(nowMs)
	for _, ev := range normalized {
		p.updateDeltas.MarkExplicit(ev)
	}
	p.noteGoalEvents(normalized, nowMs)
	if len(normalized) == 0 {
		return
	}
	p.processMomentumEventsLocked(normalized, nowMs)
}

func (p *Plugin) processMomentumEventsLocked(events []momentum.NormalizedGameEvent, nowMs int64) {
	for _, ev := range events {
		p.perfCountLocked(nowMs, "normalized."+string(ev.Type), 1)
		if ev.SourceEvent != "" {
			p.perfCountLocked(nowMs, "normalizedSource."+ev.SourceEvent, 1)
		}
		p.engine.ProcessEvent(ev)
	}
}

func (p *Plugin) envelopeFromOOFEventLocked(e oofevents.OOFEvent) (events.Envelope, bool) {
	switch ev := oofevents.Unwrap(e).(type) {
	case oofevents.MatchStartedEvent:
		return envelopeFromData("MatchCreated", events.MatchGuidData{MatchGuid: ev.MatchGUID()})
	case oofevents.MatchRestartedEvent:
		return envelopeFromData("MatchCreated", events.MatchGuidData{MatchGuid: ev.MatchGUID()})
	case oofevents.MatchEndedEvent:
		return envelopeFromData("MatchEnded", events.MatchEndedData{
			MatchGuid:     ev.MatchGUID(),
			WinnerTeamNum: ev.WinnerTeamNum,
		})
	case oofevents.MatchDestroyedEvent:
		return envelopeFromData("MatchDestroyed", events.MatchGuidData{MatchGuid: ev.MatchGUID()})
	case oofevents.ClockUpdatedEvent:
		return envelopeFromData("ClockUpdatedSeconds", events.ClockData{
			MatchGuid:   ev.MatchGUID(),
			TimeSeconds: ev.TimeSeconds,
			BOvertime:   ev.IsOvertime,
		})
	case oofevents.StateUpdatedEvent:
		return envelopeFromData("UpdateState", updateStateDataFromOOF(ev))
	case oofevents.BallHitEvent:
		ref, ok := p.playerRefFromOOFBallHitLocked(ev)
		if !ok {
			return events.Envelope{}, false
		}
		return envelopeFromData("BallHit", events.BallHitData{
			MatchGuid: ev.MatchGUID(),
			Players:   []events.PlayerRef{ref},
			Ball: events.BallHitBall{
				PreHitSpeed:  ev.PreHitSpeed,
				PostHitSpeed: ev.PostHitSpeed,
				Location: events.Vec3{
					X: ev.LocX,
					Y: ev.LocY,
					Z: ev.LocZ,
				},
			},
		})
	case oofevents.GoalScoredEvent:
		scorer := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.ScorerShortcut, ev.Scorer, ev.TeamNum)
		assister := p.optionalPlayerRefFromShortcutLocked(ev.MatchGUID(), ev.AssisterShortcut, ev.Assister, ev.TeamNum)
		lastTouch := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.LastTouchShortcut, "", ev.TeamNum)
		return envelopeFromData("GoalScored", events.GoalScoredData{
			MatchGuid: ev.MatchGUID(),
			GoalSpeed: ev.GoalSpeed,
			GoalTime:  ev.GoalTime,
			ImpactLocation: events.Vec3{
				X: ev.ImpactX,
				Y: ev.ImpactY,
				Z: ev.ImpactZ,
			},
			Scorer:        scorer,
			Assister:      assister,
			BallLastTouch: events.LastTouch{Player: lastTouch},
		})
	case oofevents.StatFeedEvent:
		main := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.MainTargetShortcut, ev.MainTarget, ev.MainTargetTeamNum)
		secondary := p.optionalPlayerRefFromShortcutLocked(ev.MatchGUID(), ev.SecondaryTargetShortcut, ev.SecondaryTarget, -1)
		return envelopeFromData("StatfeedEvent", events.StatfeedEventData{
			MatchGuid:       ev.MatchGUID(),
			EventName:       ev.EventName,
			MainTarget:      main,
			SecondaryTarget: secondary,
		})
	default:
		return events.Envelope{}, false
	}
}

func (p *Plugin) normalizedBallHitFromOOFLocked(ev oofevents.BallHitEvent, now time.Time) ([]momentum.NormalizedGameEvent, bool) {
	ref, ok := p.playerRefFromOOFBallHitLocked(ev)
	if !ok {
		return nil, false
	}
	p.rememberPlayerRefLocked(ev.MatchGUID(), ref)
	team, ok := momentum.TeamFromNum(ref.TeamNum)
	if !ok {
		return nil, false
	}
	return []momentum.NormalizedGameEvent{{
		Type:        momentum.EventBallHit,
		Team:        team,
		PlayerID:    ref.PrimaryId,
		PlayerName:  ref.Name,
		Time:        now.UnixMilli(),
		MatchGUID:   ev.MatchGUID(),
		SourceEvent: "BallHit",
	}}, true
}

func (p *Plugin) normalizedGoalScoredFromOOFLocked(ev oofevents.GoalScoredEvent, now time.Time) []momentum.NormalizedGameEvent {
	scorer := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.ScorerShortcut, ev.Scorer, ev.TeamNum)
	assister := p.optionalPlayerRefFromShortcutLocked(ev.MatchGUID(), ev.AssisterShortcut, ev.Assister, ev.TeamNum)
	lastTouch := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.LastTouchShortcut, "", ev.TeamNum)
	p.rememberPlayerRefLocked(ev.MatchGUID(), scorer)
	if assister != nil {
		p.rememberPlayerRefLocked(ev.MatchGUID(), *assister)
	}
	p.rememberPlayerRefLocked(ev.MatchGUID(), lastTouch)

	if scorer.Name == "" && scorer.PrimaryId == "" && scorer.Shortcut == 0 {
		return nil
	}
	team, ok := momentum.TeamFromNum(scorer.TeamNum)
	if !ok {
		return nil
	}
	matchClock := int(ev.GoalTime)
	out := []momentum.NormalizedGameEvent{{
		Type:        momentum.EventGoal,
		Team:        team,
		PlayerID:    scorer.PrimaryId,
		PlayerName:  scorer.Name,
		Time:        now.UnixMilli(),
		MatchClock:  &matchClock,
		MatchGUID:   ev.MatchGUID(),
		SourceEvent: "GoalScored",
	}}
	if assister != nil {
		out[0].AssisterID = assister.PrimaryId
		out = append(out, momentum.NormalizedGameEvent{
			Type:        momentum.EventAssist,
			Team:        team,
			PlayerID:    assister.PrimaryId,
			PlayerName:  assister.Name,
			Time:        now.UnixMilli(),
			MatchClock:  &matchClock,
			MatchGUID:   ev.MatchGUID(),
			SourceEvent: "GoalScored",
		})
	}
	return out
}

func (p *Plugin) normalizedStatFeedFromOOFLocked(ev oofevents.StatFeedEvent, now time.Time) []momentum.NormalizedGameEvent {
	main := p.playerRefFromShortcutLocked(ev.MatchGUID(), ev.MainTargetShortcut, ev.MainTarget, ev.MainTargetTeamNum)
	secondary := p.optionalPlayerRefFromShortcutLocked(ev.MatchGUID(), ev.SecondaryTargetShortcut, ev.SecondaryTarget, -1)
	p.rememberPlayerRefLocked(ev.MatchGUID(), main)
	if secondary != nil {
		p.rememberPlayerRefLocked(ev.MatchGUID(), *secondary)
	}
	eventType, ok := statFeedMomentumType(ev.EventName)
	if !ok {
		return nil
	}
	team, ok := momentum.TeamFromNum(main.TeamNum)
	if !ok {
		return nil
	}
	out := momentum.NormalizedGameEvent{
		Type:        eventType,
		Team:        team,
		PlayerID:    main.PrimaryId,
		PlayerName:  main.Name,
		Time:        now.UnixMilli(),
		MatchGUID:   ev.MatchGUID(),
		SourceEvent: "StatfeedEvent",
	}
	if secondary != nil {
		out.VictimID = secondary.PrimaryId
	}
	return []momentum.NormalizedGameEvent{out}
}

func statFeedMomentumType(name string) (momentum.EventType, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "shot":
		return momentum.EventShot, true
	case "save", "epicsave":
		return momentum.EventSave, true
	case "assist":
		return momentum.EventAssist, true
	case "demolish":
		return momentum.EventDemo, true
	case "goal", "owngoal":
		return momentum.EventGoal, true
	default:
		return "", false
	}
}

func (p *Plugin) handleMomentum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p.mu.Lock()
	now := time.Now().UnixMilli()
	p.applyGoalFallback(now)
	out := p.engine.Tick(now)
	resp := momentumResponse{
		MomentumFlowOutput: out,
		Display: momentumDisplayContext{
			ReplayActive:    p.replayActive,
			ReplayFileMode:  p.replayFileMode,
			ReplayChanged:   p.replayChanged,
			MomentumResetAt: p.momentumResetAt,
		},
		Prefs: copyOverlayPrefs(p.prefs),
		Perf:  p.perf.Enabled,
	}
	p.mu.Unlock()
	httputil.WriteJSON(w, resp)
}

func (p *Plugin) handleMomentumReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	p.mu.Lock()
	p.engine.Reset()
	p.updateDeltas.Reset()
	p.explicitEvents = nil
	p.replayActive = false
	p.replayFileMode = false
	p.replayChanged = 0
	p.momentumResetAt = time.Now().UnixMilli()
	p.goalResetDue = 0
	p.goalReplaySeen = false
	out := p.engine.Output()
	p.mu.Unlock()
	httputil.WriteJSON(w, out)
}

func (p *Plugin) handlePrefs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		p.mu.Lock()
		prefs := copyOverlayPrefs(p.prefs)
		p.mu.Unlock()
		httputil.WriteJSON(w, prefs)
	case http.MethodPost:
		var prefs overlayPrefs
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
		if err := decoder.Decode(&prefs); err != nil {
			http.Error(w, "invalid overlay prefs", http.StatusBadRequest)
			return
		}
		prefs.UpdatedAt = time.Now().UnixMilli()
		p.mu.Lock()
		p.prefs = copyOverlayPrefs(prefs)
		stored := copyOverlayPrefs(p.prefs)
		p.mu.Unlock()
		httputil.WriteJSON(w, stored)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (p *Plugin) handlePerf(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	p.mu.Lock()
	now := time.Now().UnixMilli()
	query := r.URL.Query()
	if query.Get("enable") == "1" || query.Get("enabled") == "1" {
		p.perfResetLocked(now, true)
	}
	if query.Get("reset") == "1" {
		p.perfResetLocked(now, p.perf.Enabled)
	}
	if query.Get("disable") == "1" || query.Get("enabled") == "0" {
		p.perfResetLocked(now, false)
	}
	snapshot := p.perfSnapshotLocked(now)
	p.mu.Unlock()

	httputil.WriteJSON(w, snapshot)
}

func (p *Plugin) handlePerfFrontend(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var report overlayPerfFrontendReport
	if r.Method == http.MethodDelete {
		report.ClientID = r.URL.Query().Get("clientId")
		report.Unregister = true
		if report.ClientID == "" && r.Body != nil {
			decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4*1024))
			_ = decoder.Decode(&report)
			report.Unregister = true
		}
	} else {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64*1024))
		if err := decoder.Decode(&report); err != nil {
			http.Error(w, "invalid overlay perf report", http.StatusBadRequest)
			return
		}
	}
	if report.ClientID == "" {
		if report.Unregister {
			http.Error(w, "missing overlay perf client id", http.StatusBadRequest)
			return
		}
		report.ClientID = "unknown"
	}
	report.At = time.Now().UnixMilli()

	p.mu.Lock()
	if p.perf.Enabled {
		p.pruneFrontendPerfReportsLocked(report.At)
		if p.perf.Frontend == nil {
			p.perf.Frontend = make(map[string]overlayPerfFrontendReport)
		}
		if report.Unregister {
			delete(p.perf.Frontend, report.ClientID)
		} else {
			p.perf.Frontend[report.ClientID] = sanitizeFrontendPerfReport(report)
		}
	}
	enabled := p.perf.Enabled
	p.mu.Unlock()

	httputil.WriteJSON(w, map[string]any{"enabled": enabled, "unregistered": report.Unregister})
}

func (p *Plugin) handleNativeHUDVisibility(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UnixMilli()
	switch r.Method {
	case http.MethodGet:
		p.mu.Lock()
		state := p.nativeHUD
		if !state.Known {
			state.Visible = true
		}
		p.mu.Unlock()
		httputil.WriteJSON(w, state)
	case http.MethodPost:
		visible, ok := parseBoolQuery(r.URL.Query().Get("visible"))
		if !ok {
			http.Error(w, "missing or invalid visible", http.StatusBadRequest)
			return
		}
		p.mu.Lock()
		p.nativeHUD = nativeHUDState{Visible: visible, Known: true, UpdatedAt: now}
		state := p.nativeHUD
		p.mu.Unlock()
		httputil.WriteJSON(w, state)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseBoolQuery(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}

type momentumResponse struct {
	momentum.MomentumFlowOutput
	Display momentumDisplayContext `json:"display"`
	Prefs   overlayPrefs           `json:"prefs,omitempty"`
	Perf    bool                   `json:"perfEnabled,omitempty"`
}

type momentumDisplayContext struct {
	ReplayActive    bool  `json:"replayActive"`
	ReplayFileMode  bool  `json:"replayFileMode,omitempty"`
	ReplayChanged   int64 `json:"replayChanged,omitempty"`
	MomentumResetAt int64 `json:"momentumResetAt,omitempty"`
}

type nativeHUDState struct {
	Visible   bool  `json:"visible"`
	Known     bool  `json:"known"`
	UpdatedAt int64 `json:"updatedAt,omitempty"`
}

type recentMomentumEvent struct {
	eventType momentum.EventType
	team      momentum.Team
	playerKey string
	matchGUID string
	time      int64
}

type overlayPrefs struct {
	Widget    map[string]any `json:"widget,omitempty"`
	Host      map[string]any `json:"host,omitempty"`
	UpdatedAt int64          `json:"updatedAt,omitempty"`
}

type overlayPerfCounters struct {
	Enabled     bool
	StartedAt   int64
	WindowStart int64
	Current     map[string]int
	Previous    map[string]int
	Totals      map[string]int
	Frontend    map[string]overlayPerfFrontendReport
}

type overlayPerfSnapshot struct {
	Enabled            bool                                 `json:"enabled"`
	StartedAt          int64                                `json:"startedAt,omitempty"`
	WindowStart        int64                                `json:"windowStart,omitempty"`
	CurrentSecond      map[string]int                       `json:"currentSecond,omitempty"`
	PreviousSecond     map[string]int                       `json:"previousSecond,omitempty"`
	Totals             map[string]int                       `json:"totals,omitempty"`
	Frontend           map[string]overlayPerfFrontendReport `json:"frontend,omitempty"`
	PlayerCacheGUID    string                               `json:"playerCacheGuid,omitempty"`
	PlayerCacheEntries int                                  `json:"playerCacheEntries"`
	EventQueueDrops    uint64                               `json:"eventQueueDrops"`
}

type overlayPerfFrontendReport struct {
	ClientID          string         `json:"clientId"`
	SchemaVersion     int            `json:"perfSchemaVersion,omitempty"`
	At                int64          `json:"at,omitempty"`
	IsHUD             bool           `json:"isHud"`
	PreviewPaused     bool           `json:"previewPaused"`
	NativeHUDVisible  bool           `json:"nativeHudVisible"`
	NativeHUDKnown    bool           `json:"nativeHudVisibilityKnown"`
	NativeHUDDormant  bool           `json:"nativeHudDormant"`
	DocumentHidden    bool           `json:"documentHidden"`
	VisibilityState   string         `json:"visibilityState,omitempty"`
	WindowFocused     bool           `json:"windowFocused"`
	ViewActive        bool           `json:"viewActive"`
	RenderActive      bool           `json:"renderActive"`
	HUDVisibleGuess   bool           `json:"hudVisibleGuess"`
	F9WindowVisible   bool           `json:"f9WindowVisible"`
	F9VisibilityKnown bool           `json:"f9WindowVisibilityKnown"`
	PerfRole          string         `json:"perfRole,omitempty"`
	PerfStatus        string         `json:"perfStatus,omitempty"`
	VisibilitySource  string         `json:"visibilitySource,omitempty"`
	NativeHUD         bool           `json:"nativeHud,omitempty"`
	AssetVersion      string         `json:"assetVersion,omitempty"`
	ClientClass       string         `json:"clientClass,omitempty"`
	Visual            string         `json:"visual,omitempty"`
	Variant           string         `json:"variant,omitempty"`
	Performance       bool           `json:"performanceMode,omitempty"`
	ReducedMotion     bool           `json:"reducedMotion,omitempty"`
	BarHidden         bool           `json:"barHidden"`
	WheelHidden       bool           `json:"wheelHidden"`
	BarDisplay        string         `json:"barDisplay,omitempty"`
	WheelDisplay      string         `json:"wheelDisplay,omitempty"`
	BarNodes          int            `json:"barNodes,omitempty"`
	WheelNodes        int            `json:"wheelNodes,omitempty"`
	CurrentSecond     map[string]int `json:"currentSecond,omitempty"`
	PreviousSecond    map[string]int `json:"previousSecond,omitempty"`
	Totals            map[string]int `json:"totals,omitempty"`
	LastSignalKey     string         `json:"lastSignalKey,omitempty"`
	NodeCount         int            `json:"nodeCount,omitempty"`
	URL               string         `json:"url,omitempty"`
	Unregister        bool           `json:"unregister,omitempty"`
}

func copyOverlayPrefs(in overlayPrefs) overlayPrefs {
	out := overlayPrefs{UpdatedAt: in.UpdatedAt}
	if in.Widget != nil {
		out.Widget = make(map[string]any, len(in.Widget))
		for key, value := range in.Widget {
			out.Widget[key] = value
		}
	}
	if in.Host != nil {
		out.Host = make(map[string]any, len(in.Host))
		for key, value := range in.Host {
			out.Host[key] = value
		}
	}
	return out
}

func (p *Plugin) perfResetLocked(now int64, enabled bool) {
	p.perf = overlayPerfCounters{
		Enabled:     enabled,
		StartedAt:   now,
		WindowStart: now / 1000 * 1000,
		Current:     make(map[string]int),
		Previous:    make(map[string]int),
		Totals:      make(map[string]int),
		Frontend:    make(map[string]overlayPerfFrontendReport),
	}
}

func (p *Plugin) perfCountLocked(now int64, key string, amount int) {
	if !p.perf.Enabled || key == "" || amount <= 0 {
		return
	}
	if p.perf.Current == nil || p.perf.Totals == nil {
		p.perfResetLocked(now, true)
	}
	p.perfRotateLocked(now)
	p.perf.Current[key] += amount
	p.perf.Totals[key] += amount
}

func (p *Plugin) perfRotateLocked(now int64) {
	window := now / 1000 * 1000
	if p.perf.WindowStart == 0 {
		p.perf.WindowStart = window
	}
	if window == p.perf.WindowStart {
		return
	}
	if window-p.perf.WindowStart == 1000 {
		p.perf.Previous = copyIntMap(p.perf.Current)
	} else {
		p.perf.Previous = nil
	}
	p.perf.Current = make(map[string]int)
	p.perf.WindowStart = window
}

func (p *Plugin) perfSnapshotLocked(now int64) overlayPerfSnapshot {
	if p.perf.Enabled {
		p.perfRotateLocked(now)
		p.pruneFrontendPerfReportsLocked(now)
	}
	return overlayPerfSnapshot{
		Enabled:            p.perf.Enabled,
		StartedAt:          p.perf.StartedAt,
		WindowStart:        p.perf.WindowStart,
		CurrentSecond:      copyIntMap(p.perf.Current),
		PreviousSecond:     copyIntMap(p.perf.Previous),
		Totals:             copyIntMap(p.perf.Totals),
		Frontend:           copyFrontendPerfReports(p.perf.Frontend, now),
		PlayerCacheGUID:    p.playerCacheGUID,
		PlayerCacheEntries: len(p.playerRefs),
		EventQueueDrops:    p.eventQueueDrops.Load(),
	}
}

func (p *Plugin) pruneFrontendPerfReportsLocked(now int64) {
	for key, report := range p.perf.Frontend {
		if frontendPerfReportExpired(report, now) {
			delete(p.perf.Frontend, key)
		}
	}
}

func frontendPerfReportExpired(report overlayPerfFrontendReport, now int64) bool {
	return report.At > 0 && now-report.At > overlayPerfFrontendReportTTLMS
}

func sanitizeFrontendPerfReport(in overlayPerfFrontendReport) overlayPerfFrontendReport {
	in.CurrentSecond = copyIntMapLimited(in.CurrentSecond, 64)
	in.PreviousSecond = copyIntMapLimited(in.PreviousSecond, 64)
	in.Totals = copyIntMapLimited(in.Totals, 128)
	in.ClientClass = classifyFrontendPerfReport(in)
	if len(in.URL) > 240 {
		in.URL = in.URL[:240]
	}
	if len(in.LastSignalKey) > 240 {
		in.LastSignalKey = in.LastSignalKey[:240]
	}
	if len(in.VisibilityState) > 64 {
		in.VisibilityState = in.VisibilityState[:64]
	}
	if len(in.PerfRole) > 80 {
		in.PerfRole = in.PerfRole[:80]
	}
	if len(in.PerfStatus) > 80 {
		in.PerfStatus = in.PerfStatus[:80]
	}
	if len(in.VisibilitySource) > 80 {
		in.VisibilitySource = in.VisibilitySource[:80]
	}
	if len(in.AssetVersion) > 80 {
		in.AssetVersion = in.AssetVersion[:80]
	}
	if len(in.ClientClass) > 80 {
		in.ClientClass = in.ClientClass[:80]
	}
	if len(in.BarDisplay) > 80 {
		in.BarDisplay = in.BarDisplay[:80]
	}
	if len(in.WheelDisplay) > 80 {
		in.WheelDisplay = in.WheelDisplay[:80]
	}
	return in
}

func classifyFrontendPerfReport(report overlayPerfFrontendReport) string {
	if report.IsHUD {
		if report.NativeHUD && report.SchemaVersion >= 2 && report.AssetVersion != "" {
			return "native-f9-hud"
		}
		if report.SchemaVersion == 0 {
			return "legacy-hud-client"
		}
		return "manual-hud-url"
	}
	if report.PerfRole == "overlay-lab-preview" || !report.IsHUD {
		return "overlay-lab-preview"
	}
	return "unknown-client"
}

func copyFrontendPerfReports(in map[string]overlayPerfFrontendReport, now int64) map[string]overlayPerfFrontendReport {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]overlayPerfFrontendReport, len(in))
	for key, report := range in {
		if frontendPerfReportExpired(report, now) {
			continue
		}
		out[key] = sanitizeFrontendPerfReport(report)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func copyIntMap(in map[string]int) map[string]int {
	return copyIntMapLimited(in, 0)
}

func copyIntMapLimited(in map[string]int, limit int) map[string]int {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]int, len(in))
	count := 0
	for key, value := range in {
		if limit > 0 && count >= limit {
			break
		}
		out[key] = value
		count++
	}
	return out
}

func envelopeFromData(event string, data any) (events.Envelope, bool) {
	b, err := json.Marshal(data)
	if err != nil {
		return events.Envelope{}, false
	}
	return events.Envelope{Event: event, Data: b}, true
}

func updateStateDataFromOOF(ev oofevents.StateUpdatedEvent) events.UpdateStateData {
	players := make([]events.Player, len(ev.Players))
	for i, p := range ev.Players {
		players[i] = events.Player{
			Name:          p.Name,
			PrimaryId:     p.PrimaryID,
			Shortcut:      p.Shortcut,
			TeamNum:       p.TeamNum,
			Score:         p.Score,
			Goals:         p.Goals,
			Shots:         p.Shots,
			Assists:       p.Assists,
			Saves:         p.Saves,
			Touches:       p.Touches,
			CarTouches:    p.CarTouches,
			Demos:         p.Demos,
			Speed:         p.Speed,
			Boost:         p.Boost,
			BBoosting:     p.IsBoosting,
			BOnWall:       p.IsOnWall,
			BPowersliding: p.IsPowersliding,
			BDemolished:   p.IsDemolished,
			BSupersonic:   p.IsSupersonic,
		}
	}

	teams := make([]events.Team, len(ev.Game.Teams))
	for i, t := range ev.Game.Teams {
		teams[i] = events.Team{
			Name:           t.Name,
			TeamNum:        t.TeamNum,
			Score:          t.Score,
			ColorPrimary:   t.ColorPrimary,
			ColorSecondary: t.ColorSecondary,
		}
	}

	return events.UpdateStateData{
		MatchGuid: ev.MatchGUID(),
		Players:   players,
		Game: events.GameState{
			Teams:       teams,
			TimeSeconds: ev.Game.TimeSeconds,
			BOvertime:   ev.Game.IsOvertime,
			Ball:        events.Ball{Speed: ev.Game.Ball.Speed},
			BReplay:     ev.Game.IsReplay,
			BHasWinner:  ev.Game.HasWinner,
			Winner:      ev.Game.Winner,
			Arena:       ev.Game.Arena,
			Playlist:    ev.Game.Playlist,
		},
	}
}

func (p *Plugin) rememberPlayersFromEnvelopeLocked(env events.Envelope) {
	switch env.Event {
	case "MatchCreated", "MatchInitialized":
		var d events.MatchGuidData
		if json.Unmarshal(env.Data, &d) == nil && d.MatchGuid != "" && d.MatchGuid != p.playerCacheGUID {
			p.playerCacheGUID = d.MatchGuid
			p.playerRefs = make(map[string]events.PlayerRef)
			p.perfCountLocked(time.Now().UnixMilli(), "cache.rebuild.match", 1)
		}
	case "MatchDestroyed":
		p.playerCacheGUID = ""
		p.playerRefs = make(map[string]events.PlayerRef)
	case "UpdateState":
		var d events.UpdateStateData
		if json.Unmarshal(env.Data, &d) != nil {
			return
		}
		p.rememberPlayersFromUpdateStateLocked(d)
	case "BallHit":
		var d events.BallHitData
		if json.Unmarshal(env.Data, &d) != nil {
			return
		}
		for _, player := range d.Players {
			p.rememberPlayerRefLocked(d.MatchGuid, player)
		}
	case "GoalScored":
		var d events.GoalScoredData
		if json.Unmarshal(env.Data, &d) != nil {
			return
		}
		p.rememberPlayerRefLocked(d.MatchGuid, d.Scorer)
		if d.Assister != nil {
			p.rememberPlayerRefLocked(d.MatchGuid, *d.Assister)
		}
		p.rememberPlayerRefLocked(d.MatchGuid, d.BallLastTouch.Player)
	case "StatfeedEvent":
		var d events.StatfeedEventData
		if json.Unmarshal(env.Data, &d) != nil {
			return
		}
		p.rememberPlayerRefLocked(d.MatchGuid, d.MainTarget)
		if d.SecondaryTarget != nil {
			p.rememberPlayerRefLocked(d.MatchGuid, *d.SecondaryTarget)
		}
	}
}

func (p *Plugin) rememberPlayersFromUpdateStateLocked(d events.UpdateStateData) {
	if d.MatchGuid == "" {
		return
	}
	if d.MatchGuid != p.playerCacheGUID {
		p.playerCacheGUID = d.MatchGuid
		p.playerRefs = make(map[string]events.PlayerRef)
		p.perfCountLocked(time.Now().UnixMilli(), "cache.rebuild.stateUpdated", 1)
	}
	p.perfCountLocked(time.Now().UnixMilli(), "cache.stateUpdated", 1)
	for _, player := range d.Players {
		p.rememberPlayerRefLocked(d.MatchGuid, events.PlayerRef{
			Name:      player.Name,
			PrimaryId: player.PrimaryId,
			Shortcut:  player.Shortcut,
			TeamNum:   player.TeamNum,
		})
	}
}

func (p *Plugin) rememberPlayerRefLocked(matchGUID string, ref events.PlayerRef) {
	if matchGUID == "" || !isKnownTeamNum(ref.TeamNum) {
		return
	}
	if matchGUID != p.playerCacheGUID {
		p.playerCacheGUID = matchGUID
		p.playerRefs = make(map[string]events.PlayerRef)
	}
	if ref.PrimaryId != "" {
		p.playerRefs[playerCacheKey("primary", ref.PrimaryId)] = ref
	}
	if ref.Shortcut != 0 {
		p.playerRefs[playerCacheKey("shortcut", strconv.Itoa(ref.Shortcut))] = ref
	}
	if ref.Name != "" {
		p.playerRefs[playerCacheKey("name", ref.Name)] = ref
	}
}

func (p *Plugin) playerRefFromOOFBallHitLocked(ev oofevents.BallHitEvent) (events.PlayerRef, bool) {
	ref, ok := p.lookupPlayerRefLocked(ev.MatchGUID(), ev.PlayerPrimaryID, ev.PlayerShortcut, ev.PlayerName)
	if !ok || !isKnownTeamNum(ref.TeamNum) {
		p.perfCountLocked(time.Now().UnixMilli(), "cache.miss.ballHit", 1)
		return events.PlayerRef{}, false
	}
	if ref.PrimaryId == "" {
		ref.PrimaryId = ev.PlayerPrimaryID
	}
	if ref.Shortcut == 0 {
		ref.Shortcut = ev.PlayerShortcut
	}
	if ref.Name == "" {
		ref.Name = ev.PlayerName
	}
	return ref, true
}

func (p *Plugin) playerRefFromShortcutLocked(matchGUID string, shortcut int, name string, fallbackTeam int) events.PlayerRef {
	ref, ok := p.lookupPlayerRefLocked(matchGUID, "", shortcut, name)
	if !ok {
		ref = events.PlayerRef{Shortcut: shortcut, Name: name, TeamNum: fallbackTeam}
	}
	if ref.Shortcut == 0 {
		ref.Shortcut = shortcut
	}
	if ref.Name == "" {
		ref.Name = name
	}
	if !isKnownTeamNum(ref.TeamNum) {
		ref.TeamNum = fallbackTeam
	}
	return ref
}

func (p *Plugin) optionalPlayerRefFromShortcutLocked(matchGUID string, shortcut int, name string, fallbackTeam int) *events.PlayerRef {
	if shortcut == 0 && name == "" {
		return nil
	}
	ref := p.playerRefFromShortcutLocked(matchGUID, shortcut, name, fallbackTeam)
	return &ref
}

func (p *Plugin) lookupPlayerRefLocked(matchGUID, primaryID string, shortcut int, name string) (events.PlayerRef, bool) {
	if matchGUID == "" || matchGUID != p.playerCacheGUID {
		return events.PlayerRef{}, false
	}
	if primaryID != "" {
		if ref, ok := p.playerRefs[playerCacheKey("primary", primaryID)]; ok {
			return ref, true
		}
	}
	if shortcut != 0 {
		if ref, ok := p.playerRefs[playerCacheKey("shortcut", strconv.Itoa(shortcut))]; ok {
			return ref, true
		}
	}
	if name != "" {
		if ref, ok := p.playerRefs[playerCacheKey("name", name)]; ok {
			return ref, true
		}
	}
	return events.PlayerRef{}, false
}

func playerCacheKey(kind, value string) string {
	return kind + ":" + value
}

func isKnownTeamNum(teamNum int) bool {
	return teamNum == 0 || teamNum == 1
}

const explicitEventDedupeWindowMs int64 = 1500

func (p *Plugin) updateReplayState(env events.Envelope, normalized []momentum.NormalizedGameEvent, now int64) {
	if env.Event != "UpdateState" {
		p.updateReplayStateFromParts(env.Event, nil, normalized, now)
		return
	}
	var d events.UpdateStateData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return
	}
	replayActive := d.Game.BReplay
	p.updateReplayStateFromParts(env.Event, &replayActive, normalized, now)
}

func (p *Plugin) updateReplayStateFromParts(eventName string, replayActive *bool, normalized []momentum.NormalizedGameEvent, now int64) {
	if p.replayActive && eventName != "UpdateState" && containsLiveBallHit(normalized) {
		if p.replayFileMode {
			return
		}
		p.replayActive = false
		p.replayFileMode = false
		p.replayChanged = now
		p.resetMomentumForKickoff(now)
		return
	}
	if eventName == "RoundStarted" {
		if p.replayActive {
			p.replayChanged = now
		}
		p.replayActive = false
		p.replayFileMode = false
		p.resetMomentumForKickoff(now)
		return
	}
	if eventName == "MatchEnded" || eventName == "MatchDestroyed" || eventName == "MatchCreated" || eventName == "MatchInitialized" {
		if p.replayActive {
			p.replayActive = false
			p.replayChanged = now
		}
		p.replayFileMode = false
		p.resetMomentumForKickoff(now)
		return
	}
	if eventName != "UpdateState" || replayActive == nil {
		return
	}
	if *replayActive != p.replayActive {
		p.replayActive = *replayActive
		p.replayChanged = now
		if *replayActive {
			p.goalReplaySeen = true
			p.replayFileMode = p.goalResetDue == 0
		} else {
			p.replayFileMode = false
			p.resetMomentumForKickoff(now)
		}
	}
}

func (p *Plugin) noteGoalEvents(events []momentum.NormalizedGameEvent, now int64) {
	for _, ev := range events {
		if ev.Type != momentum.EventGoal {
			continue
		}
		p.goalResetDue = now + 3750
		p.goalReplaySeen = p.replayActive
	}
}

func (p *Plugin) applyGoalFallback(now int64) {
	if p.goalResetDue == 0 || p.goalReplaySeen || p.replayActive || now < p.goalResetDue {
		return
	}
	p.resetMomentumForKickoff(now)
}

func (p *Plugin) resetMomentumForKickoff(now int64) {
	p.engine.Reset()
	p.updateDeltas.Reset()
	p.explicitEvents = nil
	p.momentumResetAt = now
	p.goalResetDue = 0
	p.goalReplaySeen = false
}

func (p *Plugin) filterDuplicateExplicitEvents(events []momentum.NormalizedGameEvent, now int64) []momentum.NormalizedGameEvent {
	if len(events) == 0 {
		p.trimExplicitEvents(now)
		return events
	}
	p.trimExplicitEvents(now)
	out := events[:0]
	for _, ev := range events {
		if isDuplicateEligible(ev.Type) && p.seenRecentExplicitEvent(ev, now) {
			continue
		}
		out = append(out, ev)
		if isDuplicateEligible(ev.Type) {
			p.explicitEvents = append(p.explicitEvents, recentMomentumEvent{
				eventType: ev.Type,
				team:      ev.Team,
				playerKey: eventPlayerKey(ev),
				matchGUID: ev.MatchGUID,
				time:      now,
			})
		}
	}
	return out
}

func (p *Plugin) seenRecentExplicitEvent(ev momentum.NormalizedGameEvent, now int64) bool {
	playerKeys := eventPlayerKeys(ev)
	for _, recent := range p.explicitEvents {
		if now-recent.time > explicitEventDedupeWindowMs {
			continue
		}
		if recent.eventType == ev.Type && recent.team == ev.Team && playerKeys[recent.playerKey] && recent.matchGUID == ev.MatchGUID {
			p.perfCountLocked(now, "dedupe.explicit."+string(ev.Type), 1)
			return true
		}
	}
	return false
}

func (p *Plugin) trimExplicitEvents(now int64) {
	keep := p.explicitEvents[:0]
	for _, recent := range p.explicitEvents {
		if now-recent.time <= explicitEventDedupeWindowMs {
			keep = append(keep, recent)
		}
	}
	p.explicitEvents = keep
}

func isDuplicateEligible(eventType momentum.EventType) bool {
	switch eventType {
	case momentum.EventGoal, momentum.EventShot, momentum.EventSave, momentum.EventAssist, momentum.EventDemo:
		return true
	default:
		return false
	}
}

func eventPlayerKey(ev momentum.NormalizedGameEvent) string {
	if ev.PlayerID != "" {
		return ev.PlayerID
	}
	if ev.PlayerName != "" {
		return "name:" + ev.PlayerName
	}
	return ""
}

func eventPlayerKeys(ev momentum.NormalizedGameEvent) map[string]bool {
	keys := map[string]bool{}
	if ev.PlayerID != "" {
		keys[ev.PlayerID] = true
	}
	if ev.PlayerName != "" {
		keys["name:"+ev.PlayerName] = true
	}
	if len(keys) == 0 {
		keys[""] = true
	}
	return keys
}

func containsLiveBallHit(events []momentum.NormalizedGameEvent) bool {
	for _, ev := range events {
		if ev.Type == momentum.EventBallHit {
			return true
		}
	}
	return false
}
