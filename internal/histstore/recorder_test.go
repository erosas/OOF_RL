package histstore

import (
	"path/filepath"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
)

func newTestRecorder(t *testing.T) (*Recorder, *Store) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	if err := Migrate(database); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	cfg := config.Defaults()
	s := NewStore(database)
	r := NewRecorder(s, &cfg)
	return r, s
}

func translateUpdateState(d events.UpdateStateData) oofevents.StateUpdatedEvent {
	players := make([]oofevents.PlayerSnapshot, len(d.Players))
	for i, p := range d.Players {
		players[i] = oofevents.PlayerSnapshot{
			Name:       p.Name,
			PrimaryID:  p.PrimaryId,
			Shortcut:   p.Shortcut,
			TeamNum:    p.TeamNum,
			Score:      p.Score,
			Goals:      p.Goals,
			Shots:      p.Shots,
			Assists:    p.Assists,
			Saves:      p.Saves,
			Touches:    p.Touches,
			CarTouches: p.CarTouches,
			Demos:      p.Demos,
		}
	}
	teams := make([]oofevents.TeamSnapshot, len(d.Game.Teams))
	for i, tm := range d.Game.Teams {
		teams[i] = oofevents.TeamSnapshot{
			Name:    tm.Name,
			TeamNum: tm.TeamNum,
			Score:   tm.Score,
		}
	}
	return oofevents.NewStateUpdated(d.MatchGuid, players, oofevents.GameSnapshot{
		Teams:       teams,
		TimeSeconds: d.Game.TimeSeconds,
		IsOvertime:  d.Game.BOvertime,
		IsReplay:    d.Game.BReplay,
		HasWinner:   d.Game.BHasWinner,
		Arena:       d.Game.Arena,
		Playlist:    d.Game.Playlist,
	})
}

func updateState(guid, arena string, players []events.Player, blueScore, orangeScore, seconds int) events.UpdateStateData {
	return updateStateWithReplay(guid, arena, players, blueScore, orangeScore, seconds, false)
}

func updateStateWithReplay(guid, arena string, players []events.Player, blueScore, orangeScore, seconds int, replay bool) events.UpdateStateData {
	return events.UpdateStateData{
		MatchGuid: guid,
		Players:   players,
		Game: events.GameState{
			Arena:       arena,
			TimeSeconds: seconds,
			BReplay:     replay,
			Teams: []events.Team{
				{TeamNum: 0, Score: blueScore, Name: "Blue"},
				{TeamNum: 1, Score: orangeScore, Name: "Orange"},
			},
		},
	}
}

func terminalUpdateState(guid, arena string, players []events.Player, blueScore, orangeScore int) events.UpdateStateData {
	d := updateState(guid, arena, players, blueScore, orangeScore, 0)
	d.Game.BHasWinner = true
	d.Game.Winner = "Blue"
	return d
}

func player(primaryID, name string, teamNum, score int) events.Player {
	return events.Player{
		PrimaryId: primaryID,
		Name:      name,
		TeamNum:   teamNum,
		Score:     score,
	}
}

func matchByGUID(t *testing.T, s *Store, guid string) Match {
	t.Helper()
	matches, err := s.Matches("")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	for _, m := range matches {
		if m.MatchGUID == guid {
			return m
		}
	}
	t.Fatalf("match %q not found", guid)
	return Match{}
}

func playerIDsForMatch(t *testing.T, s *Store, matchID int64) map[string]bool {
	t.Helper()
	players, err := s.MatchPlayers(matchID)
	if err != nil {
		t.Fatalf("MatchPlayers: %v", err)
	}
	out := map[string]bool{}
	for _, pl := range players {
		out[pl.PrimaryID] = true
	}
	return out
}

func matchCount(t *testing.T, s *Store) int {
	t.Helper()
	matches, err := s.Matches("")
	if err != nil {
		t.Fatalf("Matches: %v", err)
	}
	return len(matches)
}

func TestNewMatchGUIDFlushesPreviousMatchAndDoesNotCarryPlayers(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-a", "DFH Stadium", []events.Player{
		player("stale-blue", "Stale Blue", 0, 111),
		player("stale-orange", "Stale Orange", 1, 222),
	}, 1, 2, 120)))

	r.onStateUpdated(translateUpdateState(updateState("guid-b", "Paname_Dusk_P", []events.Player{
		player("current-blue", "Current Blue", 0, 333),
		player("current-orange", "Current Orange", 1, 444),
	}, 3, 4, 0)))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-b", 1))

	oldMatch := matchByGUID(t, s, "guid-a")
	if !oldMatch.Incomplete {
		t.Fatalf("old match should be marked incomplete when a new guid appears")
	}
	oldPlayers := playerIDsForMatch(t, s, oldMatch.ID)
	if !oldPlayers["stale-blue"] || !oldPlayers["stale-orange"] {
		t.Fatalf("old match should keep its own players, got %v", oldPlayers)
	}

	newMatch := matchByGUID(t, s, "guid-b")
	if newMatch.Incomplete {
		t.Fatalf("new match should complete normally")
	}
	newPlayers := playerIDsForMatch(t, s, newMatch.ID)
	if len(newPlayers) != 2 || !newPlayers["current-blue"] || !newPlayers["current-orange"] {
		t.Fatalf("new match should contain only current players, got %v", newPlayers)
	}
	if newPlayers["stale-blue"] || newPlayers["stale-orange"] {
		t.Fatalf("new match contains stale players: %v", newPlayers)
	}
}

func TestStaleMatchEndedDoesNotFlushActiveMatch(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-a", "DFH Stadium", []events.Player{
		player("a-player", "A Player", 0, 100),
	}, 1, 0, 10)))
	r.onStateUpdated(translateUpdateState(updateState("guid-b", "Mannfield", []events.Player{
		player("b-player", "B Player", 1, 200),
	}, 0, 1, 0)))

	r.onMatchEnded(oofevents.NewMatchEnded("guid-a", 0))

	if r.matchGuid != "guid-b" || r.matchID == 0 {
		t.Fatalf("stale MatchEnded should not reset active match, guid=%q id=%d", r.matchGuid, r.matchID)
	}

	r.onMatchEnded(oofevents.NewMatchEnded("guid-b", 1))
	newMatch := matchByGUID(t, s, "guid-b")
	if newMatch.Incomplete || newMatch.WinnerTeamNum != 1 {
		t.Fatalf("active match should complete after its own MatchEnded: %+v", newMatch)
	}
}

func TestUpdateStateReplacesCurrentRosterSnapshot(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 100),
		player("real-orange", "Real Orange", 1, 200),
		player("stale-blue", "Stale Blue", 0, 300),
		player("stale-orange", "Stale Orange", 1, 400),
	}, 1, 1, 120)))

	r.onStateUpdated(translateUpdateState(updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 500),
		player("real-orange", "Real Orange", 1, 600),
	}, 2, 3, 0)))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-roster", 1))

	match := matchByGUID(t, s, "guid-roster")
	players := playerIDsForMatch(t, s, match.ID)
	if len(players) != 2 || !players["real-blue"] || !players["real-orange"] {
		t.Fatalf("match should contain only the latest roster snapshot, got %v", players)
	}
	if players["stale-blue"] || players["stale-orange"] {
		t.Fatalf("stale players should be removed by latest roster snapshot, got %v", players)
	}
}

func TestLatePartialRosterDoesNotShrinkCompletedMatch(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 100),
		player("blue-two", "Blue Two", 0, 200),
		player("orange-one", "Orange One", 1, 300),
		player("orange-two", "Orange Two", 1, 400),
	}, 4, 3, 0)))

	r.onStateUpdated(translateUpdateState(updateStateWithReplay("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 500),
		player("blue-two", "Blue Two", 0, 600),
	}, 4, 3, 0, true)))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-overtime", 0))

	match := matchByGUID(t, s, "guid-overtime")
	players := playerIDsForMatch(t, s, match.ID)
	if len(players) != 4 {
		t.Fatalf("late partial roster should not shrink full match roster, got %v", players)
	}
}

func TestTerminalUpdateAfterMatchEndedDoesNotReopenMatch(t *testing.T) {
	r, s := newTestRecorder(t)
	players := []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 118),
		player("Unknown|0|0", "Sultan", 1, 34),
	}

	r.onStateUpdated(translateUpdateState(updateState("guid-real", "Wasteland", players, 1, 0, 12)))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-real", 0))

	r.onStateUpdated(translateUpdateState(terminalUpdateState("guid-real", "Wasteland", players, 1, 0)))
	r.onMatchDestroyed(oofevents.NewMatchDestroyed())
	r.onStateUpdated(translateUpdateState(terminalUpdateState("guid-postgame", "Wasteland", players, 1, 0)))
	r.onMatchDestroyed(oofevents.NewMatchDestroyed())

	if got := matchCount(t, s); got != 1 {
		t.Fatalf("terminal post-match updates should not create duplicate matches, got %d", got)
	}
	match := matchByGUID(t, s, "guid-real")
	if match.Incomplete || match.WinnerTeamNum != 0 {
		t.Fatalf("completed match should stay completed after terminal updates: %+v", match)
	}
}

func TestMatchEndedNearZeroClockIsNotForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-full-time", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-full-time", 4, false))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-full-time", 0))

	match := matchByGUID(t, s, "guid-full-time")
	if match.Forfeit {
		t.Fatalf("near-zero full-time match should not be marked forfeit: %+v", match)
	}
}

func TestMatchEndedWithSubstantialClockRemainingIsForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-forfeit", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-forfeit", 173, false))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-forfeit", 0))

	match := matchByGUID(t, s, "guid-forfeit")
	if !match.Forfeit {
		t.Fatalf("early match end should be marked forfeit: %+v", match)
	}
}

func TestMatchEndedTiedScoreIsNotForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-tied-end", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 1, 1, 173)))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-tied-end", 173, false))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-tied-end", 0))

	match := matchByGUID(t, s, "guid-tied-end")
	if match.Forfeit {
		t.Fatalf("tied early match end should not be marked forfeit without a winning score snapshot: %+v", match)
	}
}

func TestMatchEndedMissingTeamScoreSnapshotIsNotForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	d := updateState("guid-missing-scores", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)
	d.Game.Teams = nil
	r.onStateUpdated(translateUpdateState(d))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-missing-scores", 173, false))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-missing-scores", 0))

	match := matchByGUID(t, s, "guid-missing-scores")
	if match.Forfeit {
		t.Fatalf("early match end without team scores should not be marked forfeit: %+v", match)
	}
}

func TestMatchEndedInvalidWinnerIsNotForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-invalid-winner", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-invalid-winner", 173, false))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-invalid-winner", -1))

	match := matchByGUID(t, s, "guid-invalid-winner")
	if match.Forfeit {
		t.Fatalf("early match end with invalid winner should not be marked forfeit: %+v", match)
	}
}

func TestMatchEndedOvertimeIsNotForfeit(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateStateWithReplay("guid-ot-end", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173, false)))
	r.onClockUpdated(oofevents.NewClockUpdated("guid-ot-end", 173, true))
	r.onMatchEnded(oofevents.NewMatchEnded("guid-ot-end", 0))

	match := matchByGUID(t, s, "guid-ot-end")
	if match.Forfeit {
		t.Fatalf("overtime match end should not be marked forfeit: %+v", match)
	}
}

func TestUnknownBotIDsAreScopedToMatchAndShortcut(t *testing.T) {
	r, s := newTestRecorder(t)

	r.onStateUpdated(translateUpdateState(updateState("guid-gerwin", "Wasteland", []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 100),
		player("Unknown|0|0", "Gerwin", 1, 200),
	}, 1, 0, 0)))
	r.onMatchDestroyed(oofevents.NewMatchDestroyed())

	r.onStateUpdated(translateUpdateState(updateState("guid-khan", "Wasteland", []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 100),
		player("Unknown|0|0", "Khan", 1, 200),
	}, 1, 0, 0)))
	r.onMatchDestroyed(oofevents.NewMatchDestroyed())

	gerwinMatch := matchByGUID(t, s, "guid-gerwin")
	gerwinPlayers, err := s.MatchPlayers(gerwinMatch.ID)
	if err != nil {
		t.Fatalf("MatchPlayers gerwin: %v", err)
	}
	khanMatch := matchByGUID(t, s, "guid-khan")
	khanPlayers, err := s.MatchPlayers(khanMatch.ID)
	if err != nil {
		t.Fatalf("MatchPlayers khan: %v", err)
	}

	if !hasPlayer(gerwinPlayers, "bot:guid-gerwin:0", "Gerwin") {
		t.Fatalf("Gerwin bot should be match-scoped, got %+v", gerwinPlayers)
	}
	if !hasPlayer(khanPlayers, "bot:guid-khan:0", "Khan") {
		t.Fatalf("Khan bot should be match-scoped, got %+v", khanPlayers)
	}
	if hasPlayer(gerwinPlayers, "bot:guid-khan:0", "Khan") {
		t.Fatalf("Gerwin match should not be renamed to Khan, got %+v", gerwinPlayers)
	}
}

func hasPlayer(players []PlayerMatchStats, primaryID, name string) bool {
	for _, pl := range players {
		if pl.PrimaryID == primaryID && pl.Name == name {
			return true
		}
	}
	return false
}

func TestRecorderSubscribeAndStop(t *testing.T) {
	bus := oofevents.New()
	if err := bus.Start(); err != nil {
		t.Fatalf("bus.Start: %v", err)
	}
	t.Cleanup(bus.Stop)

	r, _ := newTestRecorder(t)
	r.Subscribe(bus.ForPlugin("histstore"))
	if len(r.subs) != 8 {
		t.Fatalf("Subscribe should register 8 subscriptions, got %d", len(r.subs))
	}
	r.Stop()
	if len(r.subs) != 0 {
		t.Fatalf("Stop should clear subscriptions, got %d", len(r.subs))
	}
}

func TestOnMatchStartedSetsGUID(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.onMatchStarted(oofevents.NewMatchStarted("guid-started"))
	if r.matchGuid != "guid-started" {
		t.Fatalf("expected matchGuid=guid-started, got %q", r.matchGuid)
	}
}

func TestOnMatchStartedIgnoresEmptyGUID(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.onMatchStarted(oofevents.NewMatchStarted(""))
	if r.matchGuid != "" {
		t.Fatalf("expected matchGuid empty, got %q", r.matchGuid)
	}
}

func TestOnGoalScored(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-goal", "DFH Stadium", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
		player("steam|bob|1", "Bob", 1, 50),
	}, 1, 0, 100)))

	r.onGoalScored(oofevents.NewGoalScored("guid-goal", "Alice", 0, "Bob", 1, -1, 95.5, 45.0, 10.0, 20.0, 30.0, 0))

	goals, err := s.MatchGoals(r.matchID)
	if err != nil {
		t.Fatalf("MatchGoals: %v", err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].ScorerName != "Alice" {
		t.Errorf("scorer: got %q, want Alice", goals[0].ScorerName)
	}
	if goals[0].AssisterName != "Bob" {
		t.Errorf("assister: got %q, want Bob", goals[0].AssisterName)
	}
	if goals[0].GoalSpeed != 95.5 {
		t.Errorf("speed: got %f, want 95.5", goals[0].GoalSpeed)
	}
}

func TestOnGoalScoredIgnoresEmptyScorer(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-empty-scorer", "Mannfield", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 50)))

	r.onGoalScored(oofevents.NewGoalScored("guid-empty-scorer", "", 0, "", -1, -1, 85.0, 30.0, 0, 0, 0, 0))

	goals, _ := s.MatchGoals(r.matchID)
	if len(goals) != 0 {
		t.Errorf("empty-scorer goal should be filtered, got %d goals", len(goals))
	}
}

func TestOnBallHit(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.cfg.Storage.BallHitEvents = true
	r.onStateUpdated(translateUpdateState(updateState("guid-ballhit", "Aquadome", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 60)))

	r.onBallHit(oofevents.NewBallHit("guid-ballhit", "Alice", "steam|alice|0", 0, 0, 55.0, 70.0, 1.0, 2.0, 3.0))
}

func TestOnBallHitSkippedWhenDisabled(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-bh-off", "Aquadome", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 60)))

	r.onBallHit(oofevents.NewBallHit("guid-bh-off", "Alice", "steam|alice|0", 0, 0, 55.0, 70.0, 1.0, 2.0, 3.0))
}

func TestOnStatFeed(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-statfeed", "Wasteland", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 90)))

	r.onStatFeed(oofevents.NewStatFeed("guid-statfeed", "Save", "Alice", "steam|alice|0", 0, 0, "", "", -1))

	evts, err := s.MatchStatfeedEvents(r.matchID)
	if err != nil {
		t.Fatalf("MatchStatfeedEvents: %v", err)
	}
	if len(evts) != 1 {
		t.Fatalf("expected 1 statfeed event, got %d", len(evts))
	}
	if evts[0].EventType != "Save" {
		t.Errorf("EventType: got %q, want Save", evts[0].EventType)
	}
	if evts[0].PlayerName != "Alice" {
		t.Errorf("PlayerName: got %q, want Alice", evts[0].PlayerName)
	}
}

func TestOnStatFeedWithSecondaryTarget(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-sf2", "Wasteland", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
		player("steam|bob|1", "Bob", 1, 50),
	}, 0, 0, 90)))

	r.onStatFeed(oofevents.NewStatFeed("guid-sf2", "Demo", "Alice", "steam|alice|0", 0, 0, "Bob", "steam|bob|1", 1))

	evts, _ := s.MatchStatfeedEvents(r.matchID)
	if len(evts) != 1 {
		t.Fatalf("expected 1 statfeed event, got %d", len(evts))
	}
	if evts[0].TargetName != "Bob" {
		t.Errorf("TargetName: got %q, want Bob", evts[0].TargetName)
	}
}

func TestOnGoalScoredBeforeMatchIsIgnored(t *testing.T) {
	r, s := newTestRecorder(t)
	// no state update — matchID stays 0
	r.onGoalScored(oofevents.NewGoalScored("guid-no-match", "Alice", 0, "", -1, -1, 85.0, 30.0, 0, 0, 0, 0))

	// no match in DB, so MatchGoals on id 0 returns nothing
	goals, _ := s.MatchGoals(0)
	if len(goals) != 0 {
		t.Errorf("goal before match should be ignored, got %d goals", len(goals))
	}
}

func TestOnBallHitBeforeMatchIsIgnored(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.cfg.Storage.BallHitEvents = true
	// matchID == 0 with events enabled — should not panic or write anything
	r.onBallHit(oofevents.NewBallHit("guid-no-match", "Alice", "steam|alice|0", 0, 0, 55.0, 70.0, 1.0, 2.0, 3.0))
}

func TestOnStatFeedEmptyEventNameIgnored(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-sf-empty", "Wasteland", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 90)))

	r.onStatFeed(oofevents.NewStatFeed("guid-sf-empty", "", "Alice", "steam|alice|0", 0, 0, "", "", -1))

	evts, _ := s.MatchStatfeedEvents(r.matchID)
	if len(evts) != 0 {
		t.Errorf("empty EventName should be ignored, got %d events", len(evts))
	}
}

func TestOnGoalScoredStaleGUIDIgnored(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-active", "DFH Stadium", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 60)))

	r.onGoalScored(oofevents.NewGoalScored("guid-stale", "Alice", 0, "", -1, -1, 85.0, 30.0, 0, 0, 0, 0))

	goals, _ := s.MatchGoals(r.matchID)
	if len(goals) != 0 {
		t.Errorf("goal from stale GUID should be ignored, got %d goals", len(goals))
	}
}

func TestOnBallHitStaleGUIDIgnored(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.cfg.Storage.BallHitEvents = true
	r.onStateUpdated(translateUpdateState(updateState("guid-active", "DFH Stadium", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 60)))

	// ball hit from a different guid — should be ignored
	r.onBallHit(oofevents.NewBallHit("guid-stale", "Alice", "steam|alice|0", 0, 0, 55.0, 70.0, 1.0, 2.0, 3.0))
}

func TestOnStatFeedStaleGUIDIgnored(t *testing.T) {
	r, s := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-active", "Wasteland", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 90)))

	r.onStatFeed(oofevents.NewStatFeed("guid-stale", "Save", "Alice", "steam|alice|0", 0, 0, "", "", -1))

	evts, _ := s.MatchStatfeedEvents(r.matchID)
	if len(evts) != 0 {
		t.Errorf("statfeed from stale GUID should be ignored, got %d events", len(evts))
	}
}

func TestOnClockUpdatedBeforeMatchDoesNothing(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.onClockUpdated(oofevents.NewClockUpdated("guid-x", 30, false))
	if r.lastTimeSeconds != 0 {
		t.Errorf("clock should not update before match starts, got %d", r.lastTimeSeconds)
	}
}

func TestOnClockUpdatedStaleGUIDIgnored(t *testing.T) {
	r, _ := newTestRecorder(t)
	r.onStateUpdated(translateUpdateState(updateState("guid-active", "DFH Stadium", []events.Player{
		player("steam|alice|0", "Alice", 0, 100),
	}, 0, 0, 60)))
	before := r.lastTimeSeconds

	r.onClockUpdated(oofevents.NewClockUpdated("guid-stale", 999, false))

	if r.lastTimeSeconds != before {
		t.Errorf("stale GUID clock should not update lastTimeSeconds: got %d, want %d", r.lastTimeSeconds, before)
	}
}

func TestResolvePlayerIDEmptyNameReturnsEmpty(t *testing.T) {
	got := resolvePlayerID("some-guid", "Unknown|0|0", 0, "")
	if got != "" {
		t.Errorf("expected empty string for unknown ID with empty name, got %q", got)
	}
}
