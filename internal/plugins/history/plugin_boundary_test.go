package history

import (
	"path/filepath"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
	"OOF_RL/internal/oofevents"
)

func newTestPlugin(t *testing.T) *Plugin {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	cfg := config.Defaults()
	return New(&cfg, database)
}

// translateUpdateState converts a raw RL UpdateStateData to a StateUpdatedEvent
// so tests can call plugin handlers directly (synchronous, no bus needed).
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

func matchByGUID(t *testing.T, p *Plugin, guid string) Match {
	t.Helper()
	matches, err := p.store.matches("")
	if err != nil {
		t.Fatalf("matches: %v", err)
	}
	for _, m := range matches {
		if m.MatchGUID == guid {
			return m
		}
	}
	t.Fatalf("match %q not found", guid)
	return Match{}
}

func playerIDsForMatch(t *testing.T, p *Plugin, matchID int64) map[string]bool {
	t.Helper()
	players, err := p.store.matchPlayers(matchID)
	if err != nil {
		t.Fatalf("matchPlayers: %v", err)
	}
	out := map[string]bool{}
	for _, pl := range players {
		out[pl.PrimaryID] = true
	}
	return out
}

func matchCount(t *testing.T, p *Plugin) int {
	t.Helper()
	matches, err := p.store.matches("")
	if err != nil {
		t.Fatalf("matches: %v", err)
	}
	return len(matches)
}

func TestNewMatchGUIDFlushesPreviousMatchAndDoesNotCarryPlayers(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-a", "DFH Stadium", []events.Player{
		player("stale-blue", "Stale Blue", 0, 111),
		player("stale-orange", "Stale Orange", 1, 222),
	}, 1, 2, 120)))

	p.onStateUpdated(translateUpdateState(updateState("guid-b", "Paname_Dusk_P", []events.Player{
		player("current-blue", "Current Blue", 0, 333),
		player("current-orange", "Current Orange", 1, 444),
	}, 3, 4, 0)))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-b", 1))

	oldMatch := matchByGUID(t, p, "guid-a")
	if !oldMatch.Incomplete {
		t.Fatalf("old match should be marked incomplete when a new guid appears")
	}
	oldPlayers := playerIDsForMatch(t, p, oldMatch.ID)
	if !oldPlayers["stale-blue"] || !oldPlayers["stale-orange"] {
		t.Fatalf("old match should keep its own players, got %v", oldPlayers)
	}

	newMatch := matchByGUID(t, p, "guid-b")
	if newMatch.Incomplete {
		t.Fatalf("new match should complete normally")
	}
	newPlayers := playerIDsForMatch(t, p, newMatch.ID)
	if len(newPlayers) != 2 || !newPlayers["current-blue"] || !newPlayers["current-orange"] {
		t.Fatalf("new match should contain only current players, got %v", newPlayers)
	}
	if newPlayers["stale-blue"] || newPlayers["stale-orange"] {
		t.Fatalf("new match contains stale players: %v", newPlayers)
	}
}

func TestStaleMatchEndedDoesNotFlushActiveMatch(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-a", "DFH Stadium", []events.Player{
		player("a-player", "A Player", 0, 100),
	}, 1, 0, 10)))
	p.onStateUpdated(translateUpdateState(updateState("guid-b", "Mannfield", []events.Player{
		player("b-player", "B Player", 1, 200),
	}, 0, 1, 0)))

	p.onMatchEnded(oofevents.NewMatchEnded("guid-a", 0))

	if p.matchGuid != "guid-b" || p.matchID == 0 {
		t.Fatalf("stale MatchEnded should not reset active match, guid=%q id=%d", p.matchGuid, p.matchID)
	}

	p.onMatchEnded(oofevents.NewMatchEnded("guid-b", 1))
	newMatch := matchByGUID(t, p, "guid-b")
	if newMatch.Incomplete || newMatch.WinnerTeamNum != 1 {
		t.Fatalf("active match should complete after its own MatchEnded: %+v", newMatch)
	}
}

func TestUpdateStateReplacesCurrentRosterSnapshot(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 100),
		player("real-orange", "Real Orange", 1, 200),
		player("stale-blue", "Stale Blue", 0, 300),
		player("stale-orange", "Stale Orange", 1, 400),
	}, 1, 1, 120)))

	p.onStateUpdated(translateUpdateState(updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 500),
		player("real-orange", "Real Orange", 1, 600),
	}, 2, 3, 0)))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-roster", 1))

	match := matchByGUID(t, p, "guid-roster")
	players := playerIDsForMatch(t, p, match.ID)
	if len(players) != 2 || !players["real-blue"] || !players["real-orange"] {
		t.Fatalf("match should contain only the latest roster snapshot, got %v", players)
	}
	if players["stale-blue"] || players["stale-orange"] {
		t.Fatalf("stale players should be removed by latest roster snapshot, got %v", players)
	}
}

func TestLatePartialRosterDoesNotShrinkCompletedMatch(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 100),
		player("blue-two", "Blue Two", 0, 200),
		player("orange-one", "Orange One", 1, 300),
		player("orange-two", "Orange Two", 1, 400),
	}, 4, 3, 0)))

	p.onStateUpdated(translateUpdateState(updateStateWithReplay("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 500),
		player("blue-two", "Blue Two", 0, 600),
	}, 4, 3, 0, true)))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-overtime", 0))

	match := matchByGUID(t, p, "guid-overtime")
	players := playerIDsForMatch(t, p, match.ID)
	if len(players) != 4 {
		t.Fatalf("late partial roster should not shrink full match roster, got %v", players)
	}
}

func TestTerminalUpdateAfterMatchEndedDoesNotReopenMatch(t *testing.T) {
	p := newTestPlugin(t)
	players := []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 118),
		player("Unknown|0|0", "Sultan", 1, 34),
	}

	p.onStateUpdated(translateUpdateState(updateState("guid-real", "Wasteland", players, 1, 0, 12)))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-real", 0))

	p.onStateUpdated(translateUpdateState(terminalUpdateState("guid-real", "Wasteland", players, 1, 0)))
	p.onMatchDestroyed(oofevents.NewMatchDestroyed())
	p.onStateUpdated(translateUpdateState(terminalUpdateState("guid-postgame", "Wasteland", players, 1, 0)))
	p.onMatchDestroyed(oofevents.NewMatchDestroyed())

	if got := matchCount(t, p); got != 1 {
		t.Fatalf("terminal post-match updates should not create duplicate matches, got %d", got)
	}
	match := matchByGUID(t, p, "guid-real")
	if match.Incomplete || match.WinnerTeamNum != 0 {
		t.Fatalf("completed match should stay completed after terminal updates: %+v", match)
	}
}

func TestMatchEndedNearZeroClockIsNotForfeit(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-full-time", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)))
	p.onClockUpdated(oofevents.NewClockUpdated("guid-full-time", 4, false))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-full-time", 0))

	match := matchByGUID(t, p, "guid-full-time")
	if match.Forfeit {
		t.Fatalf("near-zero full-time match should not be marked forfeit: %+v", match)
	}
}

func TestMatchEndedWithSubstantialClockRemainingIsForfeit(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-forfeit", "TrainStation", []events.Player{
		player("blue-one", "Blue One", 0, 200),
		player("orange-one", "Orange One", 1, 100),
	}, 2, 1, 173)))
	p.onClockUpdated(oofevents.NewClockUpdated("guid-forfeit", 173, false))
	p.onMatchEnded(oofevents.NewMatchEnded("guid-forfeit", 0))

	match := matchByGUID(t, p, "guid-forfeit")
	if !match.Forfeit {
		t.Fatalf("early match end should still be marked forfeit: %+v", match)
	}
}

func TestUnknownBotIDsAreScopedToMatchAndShortcut(t *testing.T) {
	p := newTestPlugin(t)

	p.onStateUpdated(translateUpdateState(updateState("guid-gerwin", "Wasteland", []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 100),
		player("Unknown|0|0", "Gerwin", 1, 200),
	}, 1, 0, 0)))
	p.onMatchDestroyed(oofevents.NewMatchDestroyed())

	p.onStateUpdated(translateUpdateState(updateState("guid-khan", "Wasteland", []events.Player{
		player("steam|player|0", "Mr Mung Beans", 0, 100),
		player("Unknown|0|0", "Khan", 1, 200),
	}, 1, 0, 0)))
	p.onMatchDestroyed(oofevents.NewMatchDestroyed())

	gerwinMatch := matchByGUID(t, p, "guid-gerwin")
	gerwinPlayers, err := p.store.matchPlayers(gerwinMatch.ID)
	if err != nil {
		t.Fatalf("matchPlayers gerwin: %v", err)
	}
	khanMatch := matchByGUID(t, p, "guid-khan")
	khanPlayers, err := p.store.matchPlayers(khanMatch.ID)
	if err != nil {
		t.Fatalf("matchPlayers khan: %v", err)
	}

	if !hasHistoryPlayer(gerwinPlayers, "bot:guid-gerwin:0", "Gerwin") {
		t.Fatalf("Gerwin bot should be match-scoped, got %+v", gerwinPlayers)
	}
	if !hasHistoryPlayer(khanPlayers, "bot:guid-khan:0", "Khan") {
		t.Fatalf("Khan bot should be match-scoped, got %+v", khanPlayers)
	}
	if hasHistoryPlayer(gerwinPlayers, "bot:guid-khan:0", "Khan") {
		t.Fatalf("Gerwin match should not be renamed to Khan, got %+v", gerwinPlayers)
	}
}

func hasHistoryPlayer(players []PlayerMatchStats, primaryID, name string) bool {
	for _, pl := range players {
		if pl.PrimaryID == primaryID && pl.Name == name {
			return true
		}
	}
	return false
}