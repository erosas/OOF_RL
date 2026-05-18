package histstore

import (
	"net/http"
	"strconv"

	"OOF_RL/internal/httputil"
)

func (s *Store) HandlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := s.AllPlayers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	httputil.WriteJSON(w, players)
}

func (s *Store) HandleMatches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	matches, err := s.Matches(playerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	teamGoals, _ := s.AllTeamGoals()
	playerCounts, _ := s.MatchPlayerCounts()
	botCounts, _ := s.MatchBotCounts()

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

func (s *Store) HandleMatchDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/matches/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	match, err := s.MatchByID(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	players, err := s.MatchPlayers(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	goals, err := s.MatchGoals(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	statfeedEvents, err := s.MatchStatfeedEvents(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if statfeedEvents == nil {
		statfeedEvents = []StatfeedEvent{}
	}
	httputil.WriteJSON(w, map[string]any{"match": match, "players": players, "goals": goals, "events": statfeedEvents})
}