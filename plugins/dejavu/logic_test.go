//go:build !wasip1

package main

import (
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"testing"

	sdk "github.com/erosas/oof-plugin-sdk"
)

type testPlayer struct {
	ID       string
	Name     string
	TeamNum  int
	Shortcut int
}

func resetPlugin(t *testing.T) {
	t.Helper()
	mu.Lock()
	currentState = nil
	cache = historyCache{}
	mu.Unlock()
	oldDBQuery := dbQuery
	dbQuery = func(string, []string) []map[string]any { return []map[string]any{} }
	t.Cleanup(func() {
		mu.Lock()
		currentState = nil
		cache = historyCache{}
		mu.Unlock()
		dbQuery = oldDBQuery
	})
}

func stubHistoryQuery(t *testing.T, fn func(string, []string) []map[string]any) {
	t.Helper()
	dbQuery = fn
}

func pushState(t *testing.T, guid string, players ...testPlayer) {
	t.Helper()
	payloadPlayers := make([]map[string]any, 0, len(players))
	for _, p := range players {
		payloadPlayers = append(payloadPlayers, map[string]any{
			"PrimaryId": p.ID,
			"Name":      p.Name,
			"TeamNum":   p.TeamNum,
			"Shortcut":  p.Shortcut,
		})
	}
	b, err := json.Marshal(map[string]any{
		"GUID":    guid,
		"Players": payloadPlayers,
	})
	if err != nil {
		t.Fatalf("marshal state: %v", err)
	}
	onEvent("state.updated", b)
}

func recall(t *testing.T, playerID string) recallResponse {
	t.Helper()
	return recallQuery(t, "player="+url.QueryEscape(playerID))
}

func recallRetry(t *testing.T, playerID string) recallResponse {
	t.Helper()
	return recallQuery(t, "player="+url.QueryEscape(playerID)+"&retry=1")
}

func recallAnchorRetry(t *testing.T, playerID string) recallResponse {
	t.Helper()
	return recallQuery(t, "anchor="+url.QueryEscape(playerID)+"&retry=1")
}

func recallQuery(t *testing.T, query string) recallResponse {
	t.Helper()
	req := sdk.HTTPRequest{
		Method: "GET",
		Path:   "/api/dejavu/recall",
		Query:  query,
	}
	resp := handleHTTP(req)
	if resp.Status != 200 {
		t.Fatalf("HTTP status: got %d body=%s", resp.Status, resp.Body)
	}
	var out recallResponse
	if err := json.Unmarshal([]byte(resp.Body), &out); err != nil {
		t.Fatalf("decode response: %v\n%s", err, resp.Body)
	}
	return out
}

func encounter(targetID string, matchID int64, matchGUID string, targetTeam, anchorTeam, winnerTeam int, incomplete bool, startedAt string) map[string]any {
	incompleteInt := 0
	if incomplete {
		incompleteInt = 1
	}
	return map[string]any{
		"target_id":       targetID,
		"match_id":        matchID,
		"match_guid":      matchGUID,
		"target_team_num": targetTeam,
		"anchor_team_num": anchorTeam,
		"winner_team_num": winnerTeam,
		"incomplete":      incompleteInt,
		"started_at":      startedAt,
	}
}

func byID(rows []recallPlayer, id string) recallPlayer {
	for _, row := range rows {
		if row.PrimaryID == id {
			return row
		}
	}
	return recallPlayer{}
}

func TestRecallStateGuards(t *testing.T) {
	resetPlugin(t)

	if got := recall(t, "steam|selected").Status; got != statusNoActiveMatch {
		t.Fatalf("no active status: got %q", got)
	}

	pushState(t, "g1", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0})
	if got := recall(t, "").Status; got != statusNoSelectedPlayer {
		t.Fatalf("missing selected status: got %q", got)
	}
	if got := recall(t, "steam|stale").Status; got != statusSelectedNotInRoster {
		t.Fatalf("stale selected status: got %q", got)
	}

	pushState(t, "g1", testPlayer{ID: "Unknown|0|0", Name: "Tracked", TeamNum: 0})
	if got := recall(t, "Unknown|0|0").Status; got != statusSelectedUnstableID {
		t.Fatalf("unstable selected status: got %q", got)
	}

	pushState(t, "g1", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 3})
	if got := recall(t, "steam|selected").Status; got != statusSelectedInvalidTeam {
		t.Fatalf("unclassified team status: got %q", got)
	}
}

func TestExactIDSameNameDifferentPlayers(t *testing.T) {
	resetPlugin(t)
	var gotArgs []string
	stubHistoryQuery(t, func(sql string, args []string) []map[string]any {
		if !strings.Contains(sql, "target.primary_id IN (?,?)") {
			t.Fatalf("expected two target placeholders, got SQL: %s", sql)
		}
		gotArgs = append([]string(nil), args...)
		return []map[string]any{
			encounter("steam|a", 1, "prior-a", 1, 0, 0, false, "2024-01-01T12:00:00Z"),
			encounter("steam|b", 2, "prior-b", 0, 0, 0, false, "2024-01-02T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|b", Name: "Same Name", TeamNum: 0},
		testPlayer{ID: "steam|a", Name: "Same Name", TeamNum: 1})

	resp := recall(t, "steam|selected")
	if resp.Status != statusOK {
		t.Fatalf("status: got %q", resp.Status)
	}
	if len(resp.Players) != 2 {
		t.Fatalf("players: got %d", len(resp.Players))
	}
	if byID(resp.Players, "steam|a").PriorCount != 1 || byID(resp.Players, "steam|b").PriorCount != 1 {
		t.Fatalf("expected separate exact-ID rows: %+v", resp.Players)
	}
	if len(gotArgs) != 5 || gotArgs[0] != "steam|selected" || gotArgs[3] != "steam|selected" || gotArgs[4] != "live-g" {
		t.Fatalf("unexpected query args: %v", gotArgs)
	}
	targetArgs := append([]string(nil), gotArgs[1:3]...)
	sort.Strings(targetArgs)
	if strings.Join(targetArgs, ",") != "steam|a,steam|b" {
		t.Fatalf("target args should be exact stable IDs, got %v", gotArgs)
	}
}

func TestRenamedPlayerSameIDUsesLiveName(t *testing.T) {
	resetPlugin(t)
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		return []map[string]any{
			encounter("steam|target", 1, "prior", 1, 0, 1, false, "2024-01-01T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "New Name", TeamNum: 1})

	row := byID(recall(t, "steam|selected").Players, "steam|target")
	if row.Name != "New Name" || row.PriorCount != 1 {
		t.Fatalf("renamed row: %+v", row)
	}
}

func TestActiveMatchRowsAreExcludedAndLastSeenUsesPriorStartedAt(t *testing.T) {
	resetPlugin(t)
	stubHistoryQuery(t, func(sql string, args []string) []map[string]any {
		if !strings.Contains(sql, "m.match_guid != ?") {
			t.Fatalf("SQL should exclude active match_guid: %s", sql)
		}
		if args[len(args)-1] != "live-g" {
			t.Fatalf("last arg should be live match guid, got %v", args)
		}
		return []map[string]any{
			encounter("steam|target", 1, "live-g", 0, 0, 0, false, "2024-01-05T12:00:00Z"),
			encounter("steam|target", 2, "prior", 1, 0, 1, false, "2024-01-03T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	row := byID(recall(t, "steam|selected").Players, "steam|target")
	if row.PriorCount != 1 || row.LastSeen != "2024-01-03T12:00:00Z" {
		t.Fatalf("active match should be excluded: %+v", row)
	}
}

func TestFirstEncounterReportsZeroPrior(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|new", Name: "New Player", TeamNum: 1})

	row := byID(recall(t, "steam|selected").Players, "steam|new")
	if queries != 1 || !row.HistoryAvailable || row.PriorCount != 0 {
		t.Fatalf("first encounter row=%+v queries=%d", row, queries)
	}
}

func TestHistoricalRelationshipSwitchingAndSessionParityWL(t *testing.T) {
	resetPlugin(t)
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		return []map[string]any{
			encounter("steam|target", 1, "m1", 0, 0, 0, false, "2024-01-01T12:00:00Z"),
			encounter("steam|target", 2, "m2", 1, 0, 1, false, "2024-01-02T12:00:00Z"),
			encounter("steam|target", 3, "m3", 1, 0, -1, true, "2024-01-03T12:00:00Z"),
			encounter("steam|target", 4, "m4", 0, 0, 0, false, "2024-01-04T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	row := byID(recall(t, "steam|selected").Players, "steam|target")
	if row.PriorCount != 4 || row.WithCount != 2 || row.AgainstCount != 2 {
		t.Fatalf("relationship counts: %+v", row)
	}
	if row.WithWins != 2 || row.WithLosses != 0 || row.AgainstWins != 0 || row.AgainstLosses != 1 || row.AgainstNoResult != 1 {
		t.Fatalf("W/L parity counts: %+v", row)
	}
	if row.PriorCount != row.WithCount+row.AgainstCount {
		t.Fatalf("prior invariant failed: %+v", row)
	}
}

func TestDuplicateEncounterRowsCountOnce(t *testing.T) {
	resetPlugin(t)
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		return []map[string]any{
			encounter("steam|target", 1, "m1", 0, 0, 0, false, "2024-01-01T12:00:00Z"),
			encounter("steam|target", 1, "m1", 0, 0, 0, false, "2024-01-01T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 0})

	row := byID(recall(t, "steam|selected").Players, "steam|target")
	if row.PriorCount != 1 || row.WithWins != 1 {
		t.Fatalf("duplicate match counted more than once: %+v", row)
	}
}

func TestUnknownLiveIDsAndGeneratedIDsAreVisibleButNotQueried(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{encounter("bot:history:2", 1, "m1", 1, 0, 0, false, "2024-01-01T12:00:00Z")}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "Unknown|0|0", Name: "Unknown Live", TeamNum: 1, Shortcut: 3},
		testPlayer{ID: "bot:live-g:4", Name: "Generated", TeamNum: 1, Shortcut: 4})

	resp := recall(t, "steam|selected")
	if queries != 0 {
		t.Fatalf("unknown/generated live IDs should not query history, got %d queries", queries)
	}
	if len(resp.Players) != 2 {
		t.Fatalf("expected visible unavailable rows, got %+v", resp.Players)
	}
	for _, row := range resp.Players {
		if row.HistoryAvailable || row.PrimaryID != "" || row.PriorCount != 0 {
			t.Fatalf("unknown row should be unavailable: %+v", row)
		}
	}
}

func TestQueryCacheCadenceAndInvalidation(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{}
	})
	pushState(t, "g1", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	_ = recall(t, "steam|selected")
	_ = recall(t, "steam|selected")
	pushState(t, "g1", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target Renamed", TeamNum: 1})
	_ = recall(t, "steam|selected")
	if queries != 1 {
		t.Fatalf("same match/id set should reuse cache, queries=%d", queries)
	}

	pushState(t, "g2", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})
	_ = recall(t, "steam|selected")
	if queries != 2 {
		t.Fatalf("new match should invalidate cache, queries=%d", queries)
	}

	pushState(t, "g2", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|other-selected", Name: "Other", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})
	_ = recall(t, "steam|other-selected")
	if queries != 3 {
		t.Fatalf("anchor selection change should invalidate cache, queries=%d", queries)
	}
}

func TestDBFailureIsPluginLocalAndCached(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return nil
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	resp := recall(t, "steam|selected")
	if resp.Status != statusHistoryError || resp.Error != historyQueryFailedMessage || !resp.Retryable || len(resp.Players) != 1 {
		t.Fatalf("expected fail-safe history error response: %+v", resp)
	}
	resp = recall(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable {
		t.Fatalf("cached history error should remain retryable: %+v", resp)
	}
	if queries != 1 {
		t.Fatalf("history error should be cached for same key, queries=%d", queries)
	}
}

func TestHistoryRetryRequeriesAndReplacesErrorCache(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		if queries == 1 {
			return nil
		}
		return []map[string]any{
			encounter("steam|target", 1, "prior", 1, 0, 0, false, "2024-01-01T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	resp := recall(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable || queries != 1 {
		t.Fatalf("first request should fail once and be retryable: resp=%+v queries=%d", resp, queries)
	}

	resp = recall(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable || queries != 1 {
		t.Fatalf("same-key normal request should reuse cached failure: resp=%+v queries=%d", resp, queries)
	}

	resp = recallRetry(t, "steam|selected")
	row := byID(resp.Players, "steam|target")
	if resp.Status != statusOK || resp.Retryable || row.PriorCount != 1 || queries != 2 {
		t.Fatalf("retry should requery once and replace error cache: resp=%+v row=%+v queries=%d", resp, row, queries)
	}

	resp = recall(t, "steam|selected")
	row = byID(resp.Players, "steam|target")
	if resp.Status != statusOK || row.PriorCount != 1 || queries != 2 {
		t.Fatalf("normal request should reuse successful retry cache: resp=%+v row=%+v queries=%d", resp, row, queries)
	}
}

func TestHistoryRetryFailureRemainsRetryable(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return nil
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	resp := recall(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable || queries != 1 {
		t.Fatalf("first request should fail retryably: resp=%+v queries=%d", resp, queries)
	}

	resp = recallRetry(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable || queries != 2 {
		t.Fatalf("failed retry should remain retryable: resp=%+v queries=%d", resp, queries)
	}

	resp = recall(t, "steam|selected")
	if resp.Status != statusHistoryError || !resp.Retryable || queries != 2 {
		t.Fatalf("post-retry normal request should reuse failed retry cache: resp=%+v queries=%d", resp, queries)
	}
}

func TestHistoryRetryDoesNotInvalidateSuccessfulCache(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{
			encounter("steam|target", 1, "prior", 1, 0, 0, false, "2024-01-01T12:00:00Z"),
		}
	})
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	resp := recall(t, "steam|selected")
	if resp.Status != statusOK || byID(resp.Players, "steam|target").PriorCount != 1 || queries != 1 {
		t.Fatalf("initial success should query once: resp=%+v queries=%d", resp, queries)
	}

	resp = recallRetry(t, "steam|selected")
	if resp.Status != statusOK || byID(resp.Players, "steam|target").PriorCount != 1 || queries != 1 {
		t.Fatalf("retry should not invalidate successful cache: resp=%+v queries=%d", resp, queries)
	}
}

func TestHistoryRetryInvalidAnchorOrNoActiveMatchDoesNotQuery(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{}
	})

	if got := recallAnchorRetry(t, "steam|missing").Status; got != statusNoActiveMatch {
		t.Fatalf("no-active retry status: got %q", got)
	}
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})
	if got := recallAnchorRetry(t, "steam|missing").Status; got != statusSelectedNotInRoster {
		t.Fatalf("invalid-anchor retry status: got %q", got)
	}
	if queries != 0 {
		t.Fatalf("retry should not query without a valid current anchor, got %d queries", queries)
	}
}

func TestMatchEndedClearsActiveStateAndNewGUIDResumes(t *testing.T) {
	resetPlugin(t)
	queries := 0
	stubHistoryQuery(t, func(string, []string) []map[string]any {
		queries++
		return []map[string]any{
			encounter("steam|target", int64(queries), "prior", 1, 0, 0, false, "2024-01-01T12:00:00Z"),
		}
	})
	pushState(t, "g1", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})

	if got := recall(t, "steam|selected"); got.Status != statusOK || len(got.Players) != 1 || queries != 1 {
		t.Fatalf("initial recall: resp=%+v queries=%d", got, queries)
	}
	onEvent("match.ended", nil)
	if got := recall(t, "steam|selected").Status; got != statusNoActiveMatch {
		t.Fatalf("match ended status: got %q", got)
	}
	if queries != 1 {
		t.Fatalf("recall after match.ended should not query history, got %d queries", queries)
	}

	pushState(t, "g2", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0},
		testPlayer{ID: "steam|target", Name: "Target", TeamNum: 1})
	if got := recall(t, "steam|selected"); got.Status != statusOK || len(got.Players) != 1 || queries != 2 {
		t.Fatalf("new guid recall should resume normally: resp=%+v queries=%d", got, queries)
	}
}

func TestMatchDestroyedClearsActiveState(t *testing.T) {
	resetPlugin(t)
	pushState(t, "live-g", testPlayer{ID: "steam|selected", Name: "Tracked", TeamNum: 0})
	onEvent("match.destroyed", nil)
	if got := recall(t, "steam|selected").Status; got != statusNoActiveMatch {
		t.Fatalf("match destroyed status: got %q", got)
	}
}
