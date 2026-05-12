package overlayhud

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"OOF_RL/internal/events"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/momentum"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js momentum-flow-bar.css casey-bar-down111.png
var viewFS embed.FS

// Plugin is a read-only overlay design lab. It renders HUD components and
// exposes deterministic event-pressure state without mutating core match data.
type Plugin struct {
	mu              sync.Mutex
	engine          *momentum.EventPressureEngine
	updateDeltas    *momentum.UpdateStateDeltaNormalizer
	explicitEvents  []recentMomentumEvent
	replayActive    bool
	replayFileMode  bool
	replayChanged   int64
	momentumResetAt int64
	goalResetDue    int64
	goalReplaySeen  bool
}

func New() *Plugin {
	return &Plugin{
		engine:       momentum.NewEventPressureEngine(momentum.DefaultConfig()),
		updateDeltas: momentum.NewUpdateStateDeltaNormalizer(),
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
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) HandleEvent(env events.Envelope) {
	now := time.Now()
	nowMs := now.UnixMilli()
	normalized := momentum.NormalizeEnvelope(env, now)
	p.mu.Lock()
	defer p.mu.Unlock()
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
	for _, ev := range allEvents {
		p.engine.ProcessEvent(ev)
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

type momentumResponse struct {
	momentum.MomentumFlowOutput
	Display momentumDisplayContext `json:"display"`
}

type momentumDisplayContext struct {
	ReplayActive    bool  `json:"replayActive"`
	ReplayFileMode  bool  `json:"replayFileMode,omitempty"`
	ReplayChanged   int64 `json:"replayChanged,omitempty"`
	MomentumResetAt int64 `json:"momentumResetAt,omitempty"`
}

type recentMomentumEvent struct {
	eventType momentum.EventType
	team      momentum.Team
	playerKey string
	matchGUID string
	time      int64
}

const explicitEventDedupeWindowMs int64 = 1500

func (p *Plugin) updateReplayState(env events.Envelope, normalized []momentum.NormalizedGameEvent, now int64) {
	if p.replayActive && env.Event != "UpdateState" && containsLiveBallHit(normalized) {
		if p.replayFileMode {
			return
		}
		p.replayActive = false
		p.replayFileMode = false
		p.replayChanged = now
		p.resetMomentumForKickoff(now)
		return
	}
	if env.Event == "RoundStarted" {
		if p.replayActive {
			p.replayChanged = now
		}
		p.replayActive = false
		p.replayFileMode = false
		p.resetMomentumForKickoff(now)
		return
	}
	if env.Event == "MatchEnded" || env.Event == "MatchDestroyed" || env.Event == "MatchCreated" || env.Event == "MatchInitialized" {
		if p.replayActive {
			p.replayActive = false
			p.replayChanged = now
		}
		p.replayFileMode = false
		p.resetMomentumForKickoff(now)
		return
	}
	if env.Event != "UpdateState" {
		return
	}
	var d events.UpdateStateData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return
	}
	if d.Game.BReplay != p.replayActive {
		p.replayActive = d.Game.BReplay
		p.replayChanged = now
		if d.Game.BReplay {
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
