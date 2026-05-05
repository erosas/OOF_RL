package history

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/events"
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

func emitHistoryEvent(t *testing.T, p *Plugin, name string, data any) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal %s: %v", name, err)
	}
	p.HandleEvent(events.Envelope{Event: name, Data: raw})
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

func TestNewMatchGUIDFlushesPreviousMatchAndDoesNotCarryPlayers(t *testing.T) {
	p := newTestPlugin(t)

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-a", "DFH Stadium", []events.Player{
		player("stale-blue", "Stale Blue", 0, 111),
		player("stale-orange", "Stale Orange", 1, 222),
	}, 1, 2, 120))

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-b", "Paname_Dusk_P", []events.Player{
		player("current-blue", "Current Blue", 0, 333),
		player("current-orange", "Current Orange", 1, 444),
	}, 3, 4, 0))
	emitHistoryEvent(t, p, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-b", WinnerTeamNum: 1})

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

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-a", "DFH Stadium", []events.Player{
		player("a-player", "A Player", 0, 100),
	}, 1, 0, 10))
	emitHistoryEvent(t, p, "UpdateState", updateState("guid-b", "Mannfield", []events.Player{
		player("b-player", "B Player", 1, 200),
	}, 0, 1, 0))

	emitHistoryEvent(t, p, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-a", WinnerTeamNum: 0})

	if p.matchGuid != "guid-b" || p.matchID == 0 {
		t.Fatalf("stale MatchEnded should not reset active match, guid=%q id=%d", p.matchGuid, p.matchID)
	}

	emitHistoryEvent(t, p, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-b", WinnerTeamNum: 1})
	newMatch := matchByGUID(t, p, "guid-b")
	if newMatch.Incomplete || newMatch.WinnerTeamNum != 1 {
		t.Fatalf("active match should complete after its own MatchEnded: %+v", newMatch)
	}
}

func TestUpdateStateReplacesCurrentRosterSnapshot(t *testing.T) {
	p := newTestPlugin(t)

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 100),
		player("real-orange", "Real Orange", 1, 200),
		player("stale-blue", "Stale Blue", 0, 300),
		player("stale-orange", "Stale Orange", 1, 400),
	}, 1, 1, 120))

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-roster", "Utopia Coliseum", []events.Player{
		player("real-blue", "Real Blue", 0, 500),
		player("real-orange", "Real Orange", 1, 600),
	}, 2, 3, 0))
	emitHistoryEvent(t, p, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-roster", WinnerTeamNum: 1})

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

	emitHistoryEvent(t, p, "UpdateState", updateState("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 100),
		player("blue-two", "Blue Two", 0, 200),
		player("orange-one", "Orange One", 1, 300),
		player("orange-two", "Orange Two", 1, 400),
	}, 4, 3, 0))

	emitHistoryEvent(t, p, "UpdateState", updateStateWithReplay("guid-overtime", "Utopia Coliseum", []events.Player{
		player("blue-one", "Blue One", 0, 500),
		player("blue-two", "Blue Two", 0, 600),
	}, 4, 3, 0, true))
	emitHistoryEvent(t, p, "MatchEnded", events.MatchEndedData{MatchGuid: "guid-overtime", WinnerTeamNum: 0})

	match := matchByGUID(t, p, "guid-overtime")
	players := playerIDsForMatch(t, p, match.ID)
	if len(players) != 4 {
		t.Fatalf("late partial roster should not shrink full match roster, got %v", players)
	}
}
