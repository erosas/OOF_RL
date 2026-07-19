package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/erosas/oof-plugin-sdk"
)

const (
	statusNoActiveMatch       = "no_active_match"
	statusNoSelectedPlayer    = "no_session_selected_player"
	statusSelectedNotInRoster = "selected_not_in_roster"
	statusSelectedUnstableID  = "selected_no_stable_id"
	statusSelectedInvalidTeam = "selected_team_unclassified"
	statusOK                  = "ok"
	statusHistoryError        = "history_error"
	historyQueryFailedMessage = "history query failed"
)

var (
	mu           sync.RWMutex
	currentState *liveState
	cache        historyCache

	dbQuery = sdk.DBQuery
)

type liveState struct {
	MatchGUID string       `json:"match_guid"`
	Players   []livePlayer `json:"players"`
}

type livePlayer struct {
	PrimaryID string `json:"primary_id,omitempty"`
	Name      string `json:"name"`
	TeamNum   int    `json:"team_num"`
	Shortcut  int    `json:"-"`
}

type stateUpdatedPayload struct {
	GUID    string `json:"GUID"`
	Players []struct {
		PrimaryID string `json:"PrimaryId"`
		Name      string `json:"Name"`
		TeamNum   int    `json:"TeamNum"`
		Shortcut  int    `json:"Shortcut"`
	} `json:"Players"`
}

type recallResponse struct {
	Status     string         `json:"status"`
	MatchGUID  string         `json:"match_guid,omitempty"`
	SelectedID string         `json:"selected_id,omitempty"`
	Selected   *selectedState `json:"selected,omitempty"`
	Players    []recallPlayer `json:"players,omitempty"`
	Error      string         `json:"error,omitempty"`
	Retryable  bool           `json:"retryable,omitempty"`
}

type selectedState struct {
	PrimaryID string `json:"primary_id"`
	Name      string `json:"name"`
	TeamNum   int    `json:"team_num"`
}

type recallPlayer struct {
	RowID            string `json:"row_id"`
	PrimaryID        string `json:"primary_id,omitempty"`
	Name             string `json:"name"`
	TeamNum          int    `json:"team_num"`
	CurrentSide      string `json:"current_side"`
	StableID         bool   `json:"stable_id"`
	HistoryAvailable bool   `json:"history_available"`
	PriorCount       int    `json:"prior_count"`
	WithCount        int    `json:"with_count"`
	AgainstCount     int    `json:"against_count"`
	WithWins         int    `json:"with_wins"`
	WithLosses       int    `json:"with_losses"`
	AgainstWins      int    `json:"against_wins"`
	AgainstLosses    int    `json:"against_losses"`
	WithNoResult     int    `json:"with_no_result"`
	AgainstNoResult  int    `json:"against_no_result"`
	LastSeen         string `json:"last_seen,omitempty"`
}

type historyCache struct {
	Ready      bool
	Key        string
	Aggregates map[string]historyAgg
	Error      string
}

type historyAgg struct {
	PriorCount      int
	WithCount       int
	AgainstCount    int
	WithWins        int
	WithLosses      int
	AgainstWins     int
	AgainstLosses   int
	WithNoResult    int
	AgainstNoResult int
	LastSeen        string
	lastSeenTime    time.Time
	seenMatchIDs    map[int64]struct{}
}

func onEvent(eventType string, payload []byte) {
	switch eventType {
	case "state.updated":
		var ev stateUpdatedPayload
		if err := json.Unmarshal(payload, &ev); err != nil {
			sdk.Log("dejavu: parse state.updated: " + err.Error())
			return
		}
		next := liveState{MatchGUID: strings.TrimSpace(ev.GUID)}
		if next.MatchGUID != "" {
			next.Players = make([]livePlayer, 0, len(ev.Players))
			for _, p := range ev.Players {
				next.Players = append(next.Players, livePlayer{
					PrimaryID: normalizePrimaryID(p.PrimaryID),
					Name:      p.Name,
					TeamNum:   p.TeamNum,
					Shortcut:  p.Shortcut,
				})
			}
		}
		mu.Lock()
		if next.MatchGUID == "" || len(next.Players) == 0 {
			resetCurrentMatchLocked()
		} else {
			currentState = &next
		}
		mu.Unlock()
	case "match.ended", "match.destroyed":
		resetCurrentMatch()
	}
}

func handleHTTP(req sdk.HTTPRequest) sdk.HTTPResponse {
	switch req.Path {
	case "/api/dejavu/recall":
		if req.Method != "" && req.Method != "GET" {
			return sdk.JSONError(405, "method not allowed")
		}
		return handleRecall(req)
	default:
		return sdk.JSONError(404, "not found")
	}
}

func handleRecall(req sdk.HTTPRequest) sdk.HTTPResponse {
	selectedID := normalizePrimaryID(sdk.QueryParam(req.Query, "player"))
	if selectedID == "" {
		selectedID = normalizePrimaryID(sdk.QueryParam(req.Query, "anchor"))
	}
	retryHistory := sdk.QueryParam(req.Query, "retry") == "1"

	st, ok := snapshotCurrentState()
	if !ok {
		return jsonRecall(recallResponse{Status: statusNoActiveMatch})
	}
	if selectedID == "" {
		return jsonRecall(recallResponse{
			Status:    statusNoSelectedPlayer,
			MatchGUID: st.MatchGUID,
		})
	}

	selected, found := findSelectedPlayer(st.Players, selectedID)
	if !found {
		return jsonRecall(recallResponse{
			Status:     statusSelectedNotInRoster,
			MatchGUID:  st.MatchGUID,
			SelectedID: selectedID,
		})
	}
	resp := recallResponse{
		Status:     statusOK,
		MatchGUID:  st.MatchGUID,
		SelectedID: selectedID,
		Selected: &selectedState{
			PrimaryID: selected.PrimaryID,
			Name:      selected.Name,
			TeamNum:   selected.TeamNum,
		},
	}
	if !isUsableStableID(selectedID) {
		resp.Status = statusSelectedUnstableID
		return jsonRecall(resp)
	}
	if !validTeam(selected.TeamNum) {
		resp.Status = statusSelectedInvalidTeam
		return jsonRecall(resp)
	}

	baseRows, stableTargetIDs := buildRosterRows(st, selectedID, selected.TeamNum)
	sort.Strings(stableTargetIDs)
	key := historyKey(st.MatchGUID, selectedID, stableTargetIDs)
	if retryHistory {
		invalidateCachedHistoryError(key)
	}
	aggs, errText := cachedHistory(key, selectedID, st.MatchGUID, stableTargetIDs)
	for i := range baseRows {
		row := &baseRows[i]
		if row.StableID {
			if agg, ok := aggs[row.PrimaryID]; ok {
				applyAgg(row, agg)
			}
		}
	}
	resp.Players = baseRows
	if errText != "" {
		resp.Status = statusHistoryError
		resp.Error = errText
		resp.Retryable = true
	}
	return jsonRecall(resp)
}

func resetCurrentMatch() {
	mu.Lock()
	resetCurrentMatchLocked()
	mu.Unlock()
}

func resetCurrentMatchLocked() {
	currentState = nil
	cache = historyCache{}
}

func snapshotCurrentState() (liveState, bool) {
	mu.RLock()
	defer mu.RUnlock()
	if currentState == nil || currentState.MatchGUID == "" {
		return liveState{}, false
	}
	out := *currentState
	out.Players = append([]livePlayer(nil), currentState.Players...)
	return out, true
}

func findSelectedPlayer(players []livePlayer, selectedID string) (livePlayer, bool) {
	for _, p := range players {
		if normalizePrimaryID(p.PrimaryID) == selectedID {
			return p, true
		}
	}
	return livePlayer{}, false
}

func buildRosterRows(st liveState, selectedID string, selectedTeam int) ([]recallPlayer, []string) {
	rows := make([]recallPlayer, 0, len(st.Players))
	stableIDs := make([]string, 0, len(st.Players))
	seenStable := map[string]struct{}{}
	for i, p := range st.Players {
		pid := normalizePrimaryID(p.PrimaryID)
		if pid == selectedID {
			continue
		}
		stable := isUsableStableID(pid)
		if stable {
			if _, seen := seenStable[pid]; seen {
				continue
			}
			seenStable[pid] = struct{}{}
			stableIDs = append(stableIDs, pid)
		}
		rows = append(rows, recallPlayer{
			RowID:            liveRowID(st.MatchGUID, p, i, stable),
			PrimaryID:        stableIDForResponse(pid, stable),
			Name:             p.Name,
			TeamNum:          p.TeamNum,
			CurrentSide:      currentSide(p.TeamNum, selectedTeam),
			StableID:         stable,
			HistoryAvailable: stable,
		})
	}
	return rows, stableIDs
}

func stableIDForResponse(primaryID string, stable bool) string {
	if !stable {
		return ""
	}
	return primaryID
}

func liveRowID(matchGUID string, p livePlayer, index int, stable bool) string {
	if stable {
		return normalizePrimaryID(p.PrimaryID)
	}
	return fmt.Sprintf("live:%s:%d:%d:%s:%d", matchGUID, p.TeamNum, p.Shortcut, p.Name, index)
}

func currentSide(teamNum, selectedTeam int) string {
	if !validTeam(teamNum) {
		return "unclassified"
	}
	if teamNum == selectedTeam {
		return "teammate"
	}
	return "opponent"
}

func validTeam(teamNum int) bool {
	return teamNum == 0 || teamNum == 1
}

func isUsableStableID(primaryID string) bool {
	id := normalizePrimaryID(primaryID)
	if id == "" {
		return false
	}
	lower := strings.ToLower(id)
	return !strings.HasPrefix(lower, "unknown|") && !strings.HasPrefix(lower, "bot:")
}

func normalizePrimaryID(primaryID string) string {
	return strings.TrimSpace(primaryID)
}

func historyKey(matchGUID, selectedID string, targetIDs []string) string {
	return matchGUID + "\x00" + selectedID + "\x00" + strings.Join(targetIDs, "\x00")
}

func cachedHistory(key, selectedID, matchGUID string, targetIDs []string) (map[string]historyAgg, string) {
	mu.RLock()
	if cache.Ready && cache.Key == key {
		aggs := cloneAggs(cache.Aggregates)
		errText := cache.Error
		mu.RUnlock()
		return aggs, errText
	}
	mu.RUnlock()

	aggs := map[string]historyAgg{}
	errText := ""
	if len(targetIDs) > 0 {
		rows := dbQuery(encounterSQL(len(targetIDs)), encounterArgs(selectedID, matchGUID, targetIDs))
		if rows == nil {
			errText = historyQueryFailedMessage
		} else {
			aggs = aggregateEncounterRows(rows, matchGUID, targetIDs)
		}
	}

	mu.Lock()
	cache = historyCache{
		Ready:      true,
		Key:        key,
		Aggregates: cloneAggs(aggs),
		Error:      errText,
	}
	mu.Unlock()

	return aggs, errText
}

func invalidateCachedHistoryError(key string) bool {
	mu.Lock()
	defer mu.Unlock()
	if cache.Ready && cache.Key == key && cache.Error != "" {
		cache = historyCache{}
		return true
	}
	return false
}

func encounterSQL(targetCount int) string {
	placeholders := make([]string, targetCount)
	for i := range placeholders {
		placeholders[i] = "?"
	}
	return fmt.Sprintf(`
		SELECT target.primary_id AS target_id,
		       m.id AS match_id,
		       COALESCE(m.match_guid,'') AS match_guid,
		       target.team_num AS target_team_num,
		       anchor.team_num AS anchor_team_num,
		       COALESCE(m.winner_team_num,-1) AS winner_team_num,
		       COALESCE(m.incomplete,0) AS incomplete,
		       m.started_at AS started_at
		FROM hist_player_match_stats target
		JOIN hist_player_match_stats anchor
		  ON anchor.match_id = target.match_id
		 AND anchor.primary_id = ?
		JOIN hist_matches m ON m.id = target.match_id
		WHERE target.primary_id IN (%s)
		  AND target.primary_id != ?
		  AND target.team_num IN (0,1)
		  AND anchor.team_num IN (0,1)
		  AND (m.match_guid IS NULL OR m.match_guid != ?)`,
		strings.Join(placeholders, ","))
}

func encounterArgs(selectedID, matchGUID string, targetIDs []string) []string {
	args := make([]string, 0, len(targetIDs)+3)
	args = append(args, selectedID)
	args = append(args, targetIDs...)
	args = append(args, selectedID, matchGUID)
	return args
}

func aggregateEncounterRows(rows []map[string]any, matchGUID string, targetIDs []string) map[string]historyAgg {
	targetSet := make(map[string]struct{}, len(targetIDs))
	for _, id := range targetIDs {
		targetSet[id] = struct{}{}
	}
	aggs := map[string]historyAgg{}
	for _, row := range rows {
		targetID := normalizePrimaryID(rowStr(row, "target_id"))
		if _, ok := targetSet[targetID]; !ok || !isUsableStableID(targetID) {
			continue
		}
		if rowStr(row, "match_guid") == matchGUID {
			continue
		}
		targetTeam := int(rowInt(row, "target_team_num"))
		anchorTeam := int(rowInt(row, "anchor_team_num"))
		if !validTeam(targetTeam) || !validTeam(anchorTeam) {
			continue
		}
		matchID := rowInt(row, "match_id")
		if matchID <= 0 {
			continue
		}

		agg := aggs[targetID]
		if agg.seenMatchIDs == nil {
			agg.seenMatchIDs = map[int64]struct{}{}
		}
		if _, seen := agg.seenMatchIDs[matchID]; seen {
			aggs[targetID] = agg
			continue
		}
		agg.seenMatchIDs[matchID] = struct{}{}

		with := targetTeam == anchorTeam
		if with {
			agg.WithCount++
		} else {
			agg.AgainstCount++
		}
		agg.PriorCount = agg.WithCount + agg.AgainstCount

		winnerTeam := int(rowInt(row, "winner_team_num"))
		resultEligible := !rowBool(row, "incomplete") && winnerTeam >= 0
		if resultEligible {
			win := anchorTeam == winnerTeam
			if with && win {
				agg.WithWins++
			} else if with {
				agg.WithLosses++
			} else if win {
				agg.AgainstWins++
			} else {
				agg.AgainstLosses++
			}
		} else if with {
			agg.WithNoResult++
		} else {
			agg.AgainstNoResult++
		}

		if startedAt := rowStr(row, "started_at"); startedAt != "" {
			if t := sdk.ParseTime(startedAt); !t.IsZero() && t.After(agg.lastSeenTime) {
				agg.lastSeenTime = t
				agg.LastSeen = startedAt
			}
		}
		aggs[targetID] = agg
	}
	return aggs
}

func applyAgg(row *recallPlayer, agg historyAgg) {
	row.PriorCount = agg.PriorCount
	row.WithCount = agg.WithCount
	row.AgainstCount = agg.AgainstCount
	row.WithWins = agg.WithWins
	row.WithLosses = agg.WithLosses
	row.AgainstWins = agg.AgainstWins
	row.AgainstLosses = agg.AgainstLosses
	row.WithNoResult = agg.WithNoResult
	row.AgainstNoResult = agg.AgainstNoResult
	row.LastSeen = agg.LastSeen
}

func cloneAggs(in map[string]historyAgg) map[string]historyAgg {
	out := make(map[string]historyAgg, len(in))
	for k, v := range in {
		v.seenMatchIDs = nil
		v.lastSeenTime = time.Time{}
		out[k] = v
	}
	return out
}

func jsonRecall(resp recallResponse) sdk.HTTPResponse {
	b, _ := json.Marshal(resp)
	return sdk.JSONResponse(b)
}

func rowStr(row map[string]any, key string) string {
	switch v := row[key].(type) {
	case string:
		return v
	case []byte:
		return string(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func rowInt(row map[string]any, key string) int64 {
	switch v := row[key].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case int32:
		return int64(v)
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	default:
		return 0
	}
}

func rowBool(row map[string]any, key string) bool {
	switch v := row[key].(type) {
	case bool:
		return v
	case int:
		return v != 0
	case int64:
		return v != 0
	case float64:
		return v != 0
	case string:
		return v == "1" || strings.EqualFold(v, "true")
	default:
		return false
	}
}
