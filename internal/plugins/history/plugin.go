package history

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/httputil"
	"OOF_RL/internal/oofevents"
	"OOF_RL/internal/plugin"
)

//go:embed view.html view.js
var viewFS embed.FS

const historySchema = `
DROP TABLE IF EXISTS ball_hit_events;
DROP TABLE IF EXISTS tick_snapshots;
DROP TABLE IF EXISTS goal_events;
DROP TABLE IF EXISTS player_match_stats;
DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS players;

CREATE TABLE IF NOT EXISTS hist_players (
	primary_id TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	last_seen  DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_matches (
	id              INTEGER PRIMARY KEY AUTOINCREMENT,
	match_guid      TEXT UNIQUE,
	arena           TEXT,
	started_at      DATETIME NOT NULL,
	ended_at        DATETIME,
	winner_team_num INTEGER,
	overtime        BOOLEAN DEFAULT 0,
	incomplete      BOOLEAN DEFAULT 0,
	forfeit         BOOLEAN DEFAULT 0,
	playlist_type   INTEGER,
	team_score_0    INTEGER,
	team_score_1    INTEGER
);

CREATE TABLE IF NOT EXISTS hist_player_match_stats (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	primary_id  TEXT    NOT NULL REFERENCES hist_players(primary_id),
	team_num    INTEGER,
	score       INTEGER DEFAULT 0,
	goals       INTEGER DEFAULT 0,
	shots       INTEGER DEFAULT 0,
	assists     INTEGER DEFAULT 0,
	saves       INTEGER DEFAULT 0,
	touches     INTEGER DEFAULT 0,
	car_touches INTEGER DEFAULT 0,
	demos       INTEGER DEFAULT 0,
	UNIQUE(match_id, primary_id)
);

CREATE TABLE IF NOT EXISTS hist_goal_events (
	id                   INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id             INTEGER NOT NULL REFERENCES hist_matches(id),
	scorer_id            TEXT REFERENCES hist_players(primary_id),
	scorer_name          TEXT NOT NULL DEFAULT '',
	assister_id          TEXT REFERENCES hist_players(primary_id),
	assister_name        TEXT NOT NULL DEFAULT '',
	ball_last_touch_id   TEXT REFERENCES hist_players(primary_id),
	goal_speed           REAL,
	goal_time            REAL,
	impact_x             REAL,
	impact_y             REAL,
	impact_z             REAL,
	scored_at            DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_ball_hit_events (
	id             INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id       INTEGER NOT NULL REFERENCES hist_matches(id),
	player_id      TEXT REFERENCES hist_players(primary_id),
	pre_hit_speed  REAL,
	post_hit_speed REAL,
	loc_x          REAL,
	loc_y          REAL,
	loc_z          REAL,
	hit_at         DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_tick_snapshots (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id    INTEGER NOT NULL REFERENCES hist_matches(id),
	captured_at DATETIME NOT NULL,
	raw_json    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS hist_statfeed_events (
	id           INTEGER PRIMARY KEY AUTOINCREMENT,
	match_id     INTEGER NOT NULL REFERENCES hist_matches(id),
	player_id    TEXT NOT NULL DEFAULT '',
	player_name  TEXT NOT NULL DEFAULT '',
	team_num     INTEGER NOT NULL DEFAULT 0,
	event_type   TEXT NOT NULL,
	target_id    TEXT NOT NULL DEFAULT '',
	target_name  TEXT NOT NULL DEFAULT '',
	occurred_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_hist_pms_primary_id ON hist_player_match_stats(primary_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_scorer     ON hist_goal_events(scorer_id);
CREATE INDEX IF NOT EXISTS idx_hist_goal_match      ON hist_goal_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_bh_player       ON hist_ball_hit_events(player_id);
CREATE INDEX IF NOT EXISTS idx_hist_tick_match      ON hist_tick_snapshots(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_matches_started_at ON hist_matches(started_at);
CREATE INDEX IF NOT EXISTS idx_hist_sf_match  ON hist_statfeed_events(match_id);
CREATE INDEX IF NOT EXISTS idx_hist_sf_player ON hist_statfeed_events(player_id);
`

type Plugin struct {
	plugin.BasePlugin
	cfg   *config.Config
	store *store
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

func New(cfg *config.Config, database *db.DB) *Plugin {
	if err := database.RunMigration(historySchema); err != nil {
		log.Printf("[history] migrate: %v", err)
	}
	for _, col := range [][2]string{
		{"team_score_0", "INTEGER"},
		{"team_score_1", "INTEGER"},
		{"incomplete", "BOOLEAN DEFAULT 0"},
		{"forfeit", "BOOLEAN DEFAULT 0"},
	} {
		if err := database.AddColumnIfNotExists("hist_matches", col[0], col[1]); err != nil {
			log.Printf("[history] migrate hist_matches.%s: %v", col[0], err)
		}
	}
	return &Plugin{cfg: cfg, store: &store{conn: database.Conn()}}
}

func (p *Plugin) ID() string         { return "history" }
func (p *Plugin) DBPrefix() string   { return "hist" }
func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) NavTab() plugin.NavTab {
	return plugin.NavTab{ID: "history", Label: "History", Order: 20}
}

func (p *Plugin) Routes(mux *http.ServeMux) {
	mux.HandleFunc("/api/players", p.handlePlayers)
	mux.HandleFunc("/api/matches", p.handleMatches)
	mux.HandleFunc("/api/matches/", p.handleMatchDetail)
}

func (p *Plugin) SettingsSchema() []plugin.Setting        { return nil }
func (p *Plugin) ApplySettings(_ map[string]string) error { return nil }
func (p *Plugin) Assets() fs.FS                           { return viewFS }

func (p *Plugin) Init(bus oofevents.PluginBus, _ plugin.Registry, _ *db.DB) error {
	p.subs = []oofevents.Subscription{
		bus.Subscribe(oofevents.TypeMatchStarted, p.onMatchStarted),
		bus.Subscribe(oofevents.TypeStateUpdated, p.onStateUpdated),
		bus.Subscribe(oofevents.TypeGoalScored, p.onGoalScored),
		bus.Subscribe(oofevents.TypeBallHit, p.onBallHit),
		bus.Subscribe(oofevents.TypeClockUpdated, p.onClockUpdated),
		bus.Subscribe(oofevents.TypeStatFeed, p.onStatFeed),
		bus.Subscribe(oofevents.TypeMatchEnded, p.onMatchEnded),
		bus.Subscribe(oofevents.TypeMatchDestroyed, p.onMatchDestroyed),
	}
	return nil
}

func (p *Plugin) Shutdown() error {
	for _, s := range p.subs {
		s.Cancel()
	}
	return nil
}

func (p *Plugin) onMatchStarted(e oofevents.OOFEvent) {
	if e.MatchGUID() == "" {
		return
	}
	p.switchMatch(e.MatchGUID())
}

func (p *Plugin) onStateUpdated(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.StateUpdatedEvent)
	if !ok {
		return
	}
	if p.matchID == 0 && ev.Game.HasWinner {
		return
	}
	if ev.MatchGUID() != "" {
		p.switchMatch(ev.MatchGUID())
	}
	p.overtime = ev.Game.IsOvertime
	p.lastTimeSeconds = ev.Game.TimeSeconds

	if p.matchID == 0 && p.matchGuid != "" {
		id, err := p.store.upsertMatch(p.matchGuid, ev.Game.Arena, time.Now())
		if err == nil {
			p.matchID = id
		}
	}

	if len(ev.Players) > 0 {
		currentPlayers := make(map[string]oofevents.PlayerSnapshot, len(ev.Players))
		for _, pl := range ev.Players {
			primaryID := historyPlayerID(ev.MatchGUID(), pl.PrimaryID, pl.Shortcut, pl.Name)
			if primaryID != "" {
				pl.PrimaryID = primaryID
				currentPlayers[primaryID] = pl
			}
		}
		if len(currentPlayers) >= len(p.lastPlayers) || !ev.Game.IsReplay {
			p.lastPlayers = currentPlayers
		}
	}

	if len(ev.Game.Teams) > 0 {
		p.lastTeams = ev.Game.Teams
	}

	if p.matchID != 0 && ev.Game.Playlist != nil && p.playlistType == nil {
		p.playlistType = ev.Game.Playlist
		_ = p.store.updateMatchPlaylist(p.matchID, *ev.Game.Playlist)
	}
}

func (p *Plugin) onGoalScored(e oofevents.OOFEvent) {
	if p.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.GoalScoredEvent)
	if !ok {
		return
	}
	if !p.isActiveMatch(ev.MatchGUID()) {
		return
	}
	// GoalScored fires twice per goal: first with scorer info, then a replay-end packet
	// with an empty scorer name. Filter the duplicate.
	if ev.Scorer == "" {
		return
	}
	scorerID := p.findPlayerByShortcut(ev.ScorerShortcut)
	assisterID, assisterName := "", ""
	if ev.Assister != "" {
		assisterID = p.findPlayerByShortcut(ev.AssisterShortcut)
		assisterName = ev.Assister
	}
	lastTouchID := p.findPlayerByShortcut(ev.LastTouchShortcut)
	_ = p.store.insertGoal(p.matchID,
		scorerID, ev.Scorer, assisterID, assisterName, lastTouchID,
		ev.GoalSpeed, ev.GoalTime,
		ev.ImpactX, ev.ImpactY, ev.ImpactZ)
}

// findPlayerByShortcut returns the PrimaryID of the player with the given Shortcut,
// or "" if not found in the current lastPlayers snapshot.
func (p *Plugin) findPlayerByShortcut(shortcut int) string {
	for id, pl := range p.lastPlayers {
		if pl.Shortcut == shortcut {
			return id
		}
	}
	return ""
}

func (p *Plugin) onBallHit(e oofevents.OOFEvent) {
	if !p.cfg.Storage.BallHitEvents || p.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.BallHitEvent)
	if !ok {
		return
	}
	if !p.isActiveMatch(ev.MatchGUID()) {
		return
	}
	playerID := historyPlayerID(ev.MatchGUID(), ev.PlayerPrimaryID, ev.PlayerShortcut, ev.PlayerName)
	_ = p.store.insertBallHit(p.matchID, playerID,
		ev.PreHitSpeed, ev.PostHitSpeed,
		ev.LocX, ev.LocY, ev.LocZ)
}

func (p *Plugin) onClockUpdated(e oofevents.OOFEvent) {
	if p.matchID == 0 {
		return
	}
	ev, ok := oofevents.Unwrap(e).(oofevents.ClockUpdatedEvent)
	if !ok {
		return
	}
	if !p.isActiveMatch(ev.MatchGUID()) {
		return
	}
	p.overtime = ev.IsOvertime
	p.lastTimeSeconds = ev.TimeSeconds
}

func (p *Plugin) onStatFeed(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.StatFeedEvent)
	if !ok || ev.EventName == "" {
		return
	}
	if !p.isActiveMatch(ev.MatchGUID()) {
		return
	}
	if ev.MainTarget == "" {
		return
	}

	actorID := p.findPlayerByShortcut(ev.MainTargetShortcut)
	targetID, targetName := "", ""
	if ev.SecondaryTarget != "" {
		targetID = p.findPlayerByShortcut(ev.SecondaryTargetShortcut)
		targetName = ev.SecondaryTarget
	}

	if p.matchID != 0 {
		_ = p.store.insertStatfeedEvent(p.matchID, actorID, ev.MainTarget, ev.MainTargetTeamNum, ev.EventName, targetID, targetName)
	}
}

func (p *Plugin) onMatchEnded(e oofevents.OOFEvent) {
	ev, ok := oofevents.Unwrap(e).(oofevents.MatchEndedEvent)
	if !ok || p.matchID == 0 {
		return
	}
	if !p.isActiveMatch(ev.MatchGUID()) {
		return
	}
	// Forfeit: if any clock time remained when MatchEnded fired, the game didn't run to zero — someone forfeited.
	forfeit := !p.overtime && p.lastTimeSeconds > 5
	p.flushMatch(ev.WinnerTeamNum, false, forfeit)
}

// onMatchDestroyed handles the case where MatchEnded is never sent (private matches,
// disconnects). Any active match is flushed and marked incomplete — winner is unknown.
func (p *Plugin) onMatchDestroyed(_ oofevents.OOFEvent) {
	if p.matchID != 0 {
		p.flushMatch(-1, true, false)
	} else {
		p.resetMatchState()
	}
}

// flushMatch writes end-of-match state to the DB and resets in-memory state.
// incomplete=true when MatchEnded was never received (private match ended early, crash, etc.).
// forfeit=true when MatchEnded fired with significant clock time remaining (opponent surrendered).
func (p *Plugin) flushMatch(winnerTeamNum int, incomplete, forfeit bool) {
	_ = p.store.endMatch(p.matchID, winnerTeamNum, p.overtime, incomplete, forfeit)

	score0, score1 := -1, -1
	for _, t := range p.lastTeams {
		if t.TeamNum == 0 {
			score0 = t.Score
		} else if t.TeamNum == 1 {
			score1 = t.Score
		}
	}
	if score0 >= 0 && score1 >= 0 {
		_ = p.store.updateTeamScores(p.matchID, score0, score1)
	}

	for _, pl := range p.lastPlayers {
		_ = p.store.upsertPlayer(pl.PrimaryID, pl.Name)
		_ = p.store.upsertPlayerMatchStats(p.matchID, pl.PrimaryID,
			pl.TeamNum, pl.Score, pl.Goals, pl.Shots, pl.Assists, pl.Saves,
			pl.Touches, pl.CarTouches, pl.Demos)
	}

	p.resetMatchState()
}

func (p *Plugin) resetMatchState() {
	p.matchID = 0
	p.matchGuid = ""
	p.overtime = false
	p.playlistType = nil
	p.lastPlayers = nil
	p.lastTeams = nil
	p.lastTimeSeconds = 0
}

func (p *Plugin) switchMatch(matchGuid string) {
	if matchGuid == "" || matchGuid == p.matchGuid {
		return
	}
	if p.matchID != 0 {
		p.flushMatch(-1, true, false)
	}
	p.resetMatchState()
	p.matchGuid = matchGuid
}

func (p *Plugin) isActiveMatch(matchGuid string) bool {
	return matchGuid == "" || p.matchGuid == "" || matchGuid == p.matchGuid
}

func historyPlayerID(matchGuid, primaryID string, shortcut int, name string) string {
	primaryID = strings.TrimSpace(primaryID)
	if primaryID != "" && !isUnknownHistoryPlayerID(primaryID) {
		return primaryID
	}
	if matchGuid == "" || name == "" {
		return ""
	}
	return "bot:" + matchGuid + ":" + strconv.Itoa(shortcut)
}

func isUnknownHistoryPlayerID(primaryID string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(primaryID)), "unknown|")
}

// -- seeding helpers (used by integration tests in other packages) --

func (p *Plugin) UpsertPlayer(primaryID, name string) error {
	return p.store.upsertPlayer(primaryID, name)
}

func (p *Plugin) UpsertMatch(guid, arena string, t time.Time) (int64, error) {
	return p.store.upsertMatch(guid, arena, t)
}

func (p *Plugin) UpsertPlayerMatchStats(matchID int64, primaryID string, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos int) error {
	return p.store.upsertPlayerMatchStats(matchID, primaryID, teamNum, score, goals, shots, assists, saves, touches, carTouches, demos)
}

func (p *Plugin) InsertGoal(matchID int64, scorerID, scorerName, assisterID, assisterName, lastTouchID string, speed, goalTime, ix, iy, iz float64) error {
	return p.store.insertGoal(matchID, scorerID, scorerName, assisterID, assisterName, lastTouchID, speed, goalTime, ix, iy, iz)
}

// -- handlers --

func (p *Plugin) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := p.store.allPlayers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httputil.WriteJSON(w, players)
}

func (p *Plugin) handleMatches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	matches, err := p.store.matches(playerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	teamGoals, _ := p.store.allTeamGoals()
	playerCounts, _ := p.store.matchPlayerCounts()
	botCounts, _ := p.store.matchBotCounts()

	type matchRow struct {
		Match
		Team0Goals  int `json:"team0_goals"`
		Team1Goals  int `json:"team1_goals"`
		PlayerCount int `json:"player_count"`
		BotCount    int `json:"bot_count"`
	}
	out := make([]matchRow, len(matches))
	for i, m := range matches {
		var t0, t1 int
		if m.TeamScore0 != nil && m.TeamScore1 != nil {
			t0, t1 = *m.TeamScore0, *m.TeamScore1
		} else {
			goals := teamGoals[m.ID]
			t0, t1 = goals[0], goals[1]
		}
		out[i] = matchRow{
			Match:       m,
			Team0Goals:  t0,
			Team1Goals:  t1,
			PlayerCount: playerCounts[m.ID],
			BotCount:    botCounts[m.ID],
		}
	}
	httputil.WriteJSON(w, out)
}

func (p *Plugin) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/matches/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	match, err := p.store.matchByID(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	players, err := p.store.matchPlayers(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	goals, err := p.store.matchGoals(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	statfeedEvents, err := p.store.matchStatfeedEvents(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if statfeedEvents == nil {
		statfeedEvents = []StatfeedEvent{}
	}
	httputil.WriteJSON(w, map[string]any{"match": match, "players": players, "goals": goals, "events": statfeedEvents})
}
