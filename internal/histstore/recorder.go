package histstore

import (
	"log"
	"strconv"
	"strings"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/oofevents"
)

// Recorder subscribes to the event bus and persists match data to the Store.
type Recorder struct {
	store *Store
	cfg   *config.Config
	subs  []oofevents.Subscription

	// per-match state, reset on MatchDestroyed
	matchID         int64
	matchGuid       string
	overtime        bool
	playlistType    *int
	lastPlayers     map[string]oofevents.PlayerSnapshot
	lastTeams       []oofevents.TeamSnapshot
	lastTimeSeconds int
}

func NewRecorder(s *Store, cfg *config.Config) *Recorder {
	return &Recorder{store: s, cfg: cfg}
}

// Subscribe wires all event handlers onto the bus. Call after bus.Start().
func (r *Recorder) Subscribe(bus oofevents.PluginBus) {
	r.subs = append(r.subs,
		bus.Subscribe(oofevents.TypeMatchStarted, r.onMatchStarted),
		bus.Subscribe(oofevents.TypeStateUpdated, r.onStateUpdated),
		bus.Subscribe(oofevents.TypeGoalScored, r.onGoalScored),
		bus.Subscribe(oofevents.TypeBallHit, r.onBallHit),
		bus.Subscribe(oofevents.TypeClockUpdated, r.onClockUpdated),
		bus.Subscribe(oofevents.TypeStatFeed, r.onStatFeed),
		bus.Subscribe(oofevents.TypeMatchEnded, r.onMatchEnded),
		bus.Subscribe(oofevents.TypeMatchDestroyed, r.onMatchDestroyed),
	)
}

// Stop cancels all bus subscriptions.
func (r *Recorder) Stop() {
	for _, s := range r.subs {
		s.Cancel()
	}
	r.subs = nil
}

func (r *Recorder) onMatchStarted(e oofevents.OOFEvent) {
	if e.MatchGUID() == "" {
		return
	}
	r.switchMatch(e.MatchGUID())
}

func (r *Recorder) onStateUpdated(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.StateUpdatedEvent)
	if !ok {
		return
	}
	if r.matchID == 0 && ev.Game.HasWinner {
		return
	}
	if ev.MatchGUID() != "" {
		r.switchMatch(ev.MatchGUID())
	}
	r.overtime = ev.Game.IsOvertime
	r.lastTimeSeconds = ev.Game.TimeSeconds

	if r.matchID == 0 && r.matchGuid != "" {
		id, err := r.store.UpsertMatch(r.matchGuid, ev.Game.Arena, time.Now())
		if err == nil {
			r.matchID = id
		}
	}

	if len(ev.Players) > 0 {
		currentPlayers := make(map[string]oofevents.PlayerSnapshot, len(ev.Players))
		for _, pl := range ev.Players {
			primaryID := resolvePlayerID(ev.MatchGUID(), pl.PrimaryID, pl.Shortcut, pl.Name)
			if primaryID != "" {
				pl.PrimaryID = primaryID
				currentPlayers[primaryID] = pl
			}
		}
		if len(currentPlayers) >= len(r.lastPlayers) || !ev.Game.IsReplay {
			r.lastPlayers = currentPlayers
		}
	}

	if len(ev.Game.Teams) > 0 {
		r.lastTeams = ev.Game.Teams
	}

	if r.matchID != 0 && ev.Game.Playlist != nil && r.playlistType == nil {
		r.playlistType = ev.Game.Playlist
		if err := r.store.UpdateMatchPlaylist(r.matchID, *ev.Game.Playlist); err != nil {
			log.Printf("[histstore] UpdateMatchPlaylist %d: %v", r.matchID, err)
		}
	}
}

func (r *Recorder) onGoalScored(e oofevents.OOFEvent) {
	if r.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.GoalScoredEvent)
	if !ok || !r.isActiveMatch(ev.MatchGUID()) {
		return
	}
	// GoalScored fires twice per goal: once with scorer info, once as a
	// replay-end packet with an empty scorer name. Filter the duplicate.
	if ev.Scorer == "" {
		return
	}
	scorerID := r.findPlayerByShortcut(ev.ScorerShortcut)
	assisterID, assisterName := "", ""
	if ev.Assister != "" {
		assisterID = r.findPlayerByShortcut(ev.AssisterShortcut)
		assisterName = ev.Assister
	}
	lastTouchID := r.findPlayerByShortcut(ev.LastTouchShortcut)
	if err := r.store.InsertGoal(r.matchID,
		scorerID, ev.Scorer, assisterID, assisterName, lastTouchID,
		ev.GoalSpeed, ev.GoalTime,
		ev.ImpactX, ev.ImpactY, ev.ImpactZ); err != nil {
		log.Printf("[histstore] InsertGoal match %d: %v", r.matchID, err)
	}
}

func (r *Recorder) onBallHit(e oofevents.OOFEvent) {
	if !r.cfg.Storage.BallHitEvents || r.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.BallHitEvent)
	if !ok || !r.isActiveMatch(ev.MatchGUID()) {
		return
	}
	playerID := resolvePlayerID(ev.MatchGUID(), ev.PlayerPrimaryID, ev.PlayerShortcut, ev.PlayerName)
	if err := r.store.InsertBallHit(r.matchID, playerID,
		ev.PreHitSpeed, ev.PostHitSpeed,
		ev.LocX, ev.LocY, ev.LocZ); err != nil {
		log.Printf("[histstore] InsertBallHit match %d: %v", r.matchID, err)
	}
}

func (r *Recorder) onClockUpdated(e oofevents.OOFEvent) {
	if r.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.ClockUpdatedEvent)
	if !ok || !r.isActiveMatch(ev.MatchGUID()) {
		return
	}
	r.overtime = ev.IsOvertime
	r.lastTimeSeconds = ev.TimeSeconds
}

func (r *Recorder) onStatFeed(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.StatFeedEvent)
	if !ok || ev.EventName == "" || ev.MainTarget == "" {
		return
	}
	if !r.isActiveMatch(ev.MatchGUID()) {
		return
	}
	actorID := r.findPlayerByShortcut(ev.MainTargetShortcut)
	targetID, targetName := "", ""
	if ev.SecondaryTarget != "" {
		targetID = r.findPlayerByShortcut(ev.SecondaryTargetShortcut)
		targetName = ev.SecondaryTarget
	}
	if r.matchID != 0 {
		if err := r.store.InsertStatfeedEvent(r.matchID, actorID, ev.MainTarget, ev.MainTargetTeamNum, ev.EventName, targetID, targetName); err != nil {
			log.Printf("[histstore] InsertStatfeedEvent match %d: %v", r.matchID, err)
		}
	}
}

func (r *Recorder) onMatchEnded(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.MatchEndedEvent)
	if !ok || r.matchID == 0 || !r.isActiveMatch(ev.MatchGUID()) {
		return
	}
	forfeit := r.isLikelyForfeit(ev.WinnerTeamNum)
	r.flushMatch(ev.WinnerTeamNum, false, forfeit)
}

func (r *Recorder) isLikelyForfeit(winnerTeamNum int) bool {
	if r.overtime || winnerTeamNum < 0 || r.lastTimeSeconds <= 5 {
		return false
	}
	winnerScore, opponentScore, ok := r.teamScoreLead(winnerTeamNum)
	return ok && winnerScore > opponentScore
}

func (r *Recorder) teamScoreLead(winnerTeamNum int) (int, int, bool) {
	winnerScore := 0
	opponentScore := 0
	winnerFound := false
	opponentFound := false
	for _, t := range r.lastTeams {
		if t.TeamNum == winnerTeamNum {
			winnerScore = t.Score
			winnerFound = true
			continue
		}
		if !opponentFound || t.Score > opponentScore {
			opponentScore = t.Score
			opponentFound = true
		}
	}
	return winnerScore, opponentScore, winnerFound && opponentFound
}

// onMatchDestroyed handles the case where MatchEnded is never sent (private
// matches, disconnects). Any active match is flushed and marked incomplete.
func (r *Recorder) onMatchDestroyed(_ oofevents.OOFEvent) {
	if r.matchID != 0 {
		r.flushMatch(-1, true, false)
	} else {
		r.resetMatchState()
	}
}

// flushMatch writes end-of-match state to the DB and resets in-memory state.
func (r *Recorder) flushMatch(winnerTeamNum int, incomplete, forfeit bool) {
	if err := r.store.EndMatch(r.matchID, winnerTeamNum, r.overtime, incomplete, forfeit); err != nil {
		log.Printf("[histstore] EndMatch %d: %v", r.matchID, err)
	}

	score0, score1 := -1, -1
	for _, t := range r.lastTeams {
		if t.TeamNum == 0 {
			score0 = t.Score
		} else if t.TeamNum == 1 {
			score1 = t.Score
		}
	}
	if score0 >= 0 && score1 >= 0 {
		if err := r.store.UpdateTeamScores(r.matchID, score0, score1); err != nil {
			log.Printf("[histstore] UpdateTeamScores %d: %v", r.matchID, err)
		}
	}

	for _, pl := range r.lastPlayers {
		if err := r.store.UpsertPlayer(pl.PrimaryID, pl.Name); err != nil {
			log.Printf("[histstore] UpsertPlayer %s: %v", pl.PrimaryID, err)
		}
		if err := r.store.UpsertPlayerMatchStatsWithName(r.matchID, pl.PrimaryID, pl.Name,
			pl.TeamNum, pl.Score, pl.Goals, pl.Shots, pl.Assists, pl.Saves,
			pl.Touches, pl.CarTouches, pl.Demos); err != nil {
			log.Printf("[histstore] UpsertPlayerMatchStats %s: %v", pl.PrimaryID, err)
		}
	}

	r.resetMatchState()
}

func (r *Recorder) resetMatchState() {
	r.matchID = 0
	r.matchGuid = ""
	r.overtime = false
	r.playlistType = nil
	r.lastPlayers = nil
	r.lastTeams = nil
	r.lastTimeSeconds = 0
}

func (r *Recorder) switchMatch(matchGuid string) {
	if matchGuid == "" || matchGuid == r.matchGuid {
		return
	}
	if r.matchID != 0 {
		r.flushMatch(-1, true, false)
	}
	r.resetMatchState()
	r.matchGuid = matchGuid
}

func (r *Recorder) isActiveMatch(matchGuid string) bool {
	return matchGuid == "" || r.matchGuid == "" || matchGuid == r.matchGuid
}

func (r *Recorder) findPlayerByShortcut(shortcut int) string {
	for id, pl := range r.lastPlayers {
		if pl.Shortcut == shortcut {
			return id
		}
	}
	return ""
}

// resolvePlayerID returns a stable player ID for history purposes.
// Unknown/bot players are scoped by match GUID and shortcut slot.
func resolvePlayerID(matchGuid, primaryID string, shortcut int, name string) string {
	primaryID = strings.TrimSpace(primaryID)
	if primaryID != "" && !isUnknownID(primaryID) {
		return primaryID
	}
	if matchGuid == "" || name == "" {
		return ""
	}
	return "bot:" + matchGuid + ":" + strconv.Itoa(shortcut)
}

func isUnknownID(primaryID string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(primaryID)), "unknown|")
}
