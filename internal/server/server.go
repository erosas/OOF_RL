package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"OOF_RL/internal/config"
	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// trnLimiter spaces out live TRN API calls so they look organic rather than
// machine-like. Each caller reserves the next available slot under a mutex,
// then sleeps outside it — so multiple goroutines queue cleanly without
// one holding the lock during a sleep.
type trnLimiter struct {
	mu        sync.Mutex
	nextSlot  time.Time
	baseDelay time.Duration // minimum gap between requests
	maxJitter time.Duration // random extra delay added on top
}

func newTRNLimiter() *trnLimiter {
	return &trnLimiter{
		baseDelay: 3000 * time.Millisecond,
		maxJitter: 1500 * time.Millisecond, // total gap: 3–4.5 s
	}
}

// isAllAsterisks reports whether s is non-empty and consists solely of '*'
// characters — the mask RL uses for cross-platform Switch player names.
func isAllAsterisks(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if c != '*' {
			return false
		}
	}
	return true
}

func (l *trnLimiter) Wait() {
	l.mu.Lock()
	now := time.Now()
	slot := l.nextSlot
	if slot.Before(now) {
		slot = now
	}
	jitter := time.Duration(rand.Int63n(int64(l.maxJitter) + 1))
	l.nextSlot = slot.Add(l.baseDelay + jitter)
	l.mu.Unlock()
	if wait := slot.Sub(now); wait > 0 {
		time.Sleep(wait)
	}
}

type Server struct {
	cfg       *config.Config
	db        *db.DB
	hub       *hub.Hub
	fs        http.Handler
	reconnect func() // triggers RL client reconnect
	trn       *trnLimiter
}

func New(cfg *config.Config, database *db.DB, h *hub.Hub, static http.Handler, reconnect func()) *Server {
	return &Server{cfg: cfg, db: database, hub: h, fs: static, reconnect: reconnect, trn: newTRNLimiter()}
}

func (s *Server) Register(mux *http.ServeMux) {
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/api/players", s.handlePlayers)
	mux.HandleFunc("/api/matches", s.handleMatches)
	mux.HandleFunc("/api/matches/", s.handleMatchDetail)
	mux.HandleFunc("/api/players/", s.handlePlayerDetail)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/config/ini", s.handleINI)
	mux.HandleFunc("/api/replays", s.handleReplays)
	mux.HandleFunc("/api/captures", s.handleCaptures)
	mux.HandleFunc("/api/captures/", s.handleCaptureDetail)
	mux.HandleFunc("/api/tracker/profile", s.handleTrackerProfile)
	mux.HandleFunc("/api/ballchasing/replays", s.handleBCReplays)
	mux.HandleFunc("/api/ballchasing/groups", s.handleBCGroups)
	mux.HandleFunc("/api/ballchasing/upload", s.handleBCUpload)
	mux.HandleFunc("/api/ballchasing/uploads", s.handleBCUploads)
	mux.HandleFunc("/api/ballchasing/ping", s.handleBCPing)
	mux.Handle("/", s.fs)
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.hub.Register(conn)
	defer func() {
		s.hub.Unregister(conn)
		conn.Close()
	}()
	// keep alive — drain any pings
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (s *Server) handlePlayers(w http.ResponseWriter, r *http.Request) {
	players, err := s.db.AllPlayers()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, players)
}

func (s *Server) handleMatches(w http.ResponseWriter, r *http.Request) {
	playerID := r.URL.Query().Get("player")
	matches, err := s.db.Matches(playerID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	teamGoals, _ := s.db.AllTeamGoals()
	playerCounts, _ := s.db.MatchPlayerCounts()

	type matchRow struct {
		db.Match
		Team0Goals  int `json:"team0_goals"`
		Team1Goals  int `json:"team1_goals"`
		PlayerCount int `json:"player_count"`
	}
	out := make([]matchRow, len(matches))
	for i, m := range matches {
		goals := teamGoals[m.ID]
		out[i] = matchRow{
			Match:       m,
			Team0Goals:  goals[0],
			Team1Goals:  goals[1],
			PlayerCount: playerCounts[m.ID],
		}
	}
	writeJSON(w, out)
}

func (s *Server) handleMatchDetail(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/api/matches/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "bad id", 400)
		return
	}
	players, err := s.db.MatchPlayers(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	goals, err := s.db.MatchGoals(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"players": players, "goals": goals})
}

func (s *Server) handlePlayerDetail(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/players/"):]
	agg, err := s.db.PlayerAggregate(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	matches, err := s.db.Matches(id)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	writeJSON(w, map[string]any{"aggregate": agg, "matches": matches})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		writeJSON(w, s.cfg)
		return
	}
	if r.Method == http.MethodPost {
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		*s.cfg = incoming
		if err := config.Save("config.toml", *s.cfg); err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		s.reconnect()
		writeJSON(w, s.cfg)
		return
	}
	http.Error(w, "method not allowed", 405)
}

func (s *Server) handleINI(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		ini, err := config.ReadINI(s.cfg.RLInstallPath)
		if err != nil {
			note := "DefaultStatsAPI.ini not found — it will be created when you save."
			if s.cfg.RLInstallPath == "" {
				note = "RL install path not set. Configure it above and save first."
			}
			writeJSON(w, map[string]any{
				"PacketSendRate": 0.0,
				"Port":           49123,
				"note":           note,
				"error":          true,
			})
			return
		}
		writeJSON(w, ini)
		return
	}
	if r.Method == http.MethodPost {
		var ini config.INISettings
		if err := json.NewDecoder(r.Body).Decode(&ini); err != nil {
			jsonError(w, 400, err.Error())
			return
		}
		if err := config.WriteINI(s.cfg.RLInstallPath, ini); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		s.reconnect()
		writeJSON(w, map[string]string{"status": "ok", "note": "INI saved. Restart Rocket League for changes to take effect."})
		return
	}
	http.Error(w, "method not allowed", 405)
}

func (s *Server) handleTrackerProfile(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	playerName := r.URL.Query().Get("name") // display name, required for non-Steam platforms
	if id == "" {
		http.Error(w, "missing id", 400)
		return
	}
	sep := strings.IndexAny(id, "|:_")
	if sep < 1 {
		http.Error(w, "invalid id format, expected platform|id", 400)
		return
	}
	platform := strings.ToLower(id[:sep])
	rest := id[sep+1:]
	if end := strings.IndexAny(rest, "|:_"); end >= 0 {
		rest = rest[:end]
	}
	// Normalize RL Stats API platform slugs to TRN platform slugs.
	switch platform {
	case "ps4", "ps5", "playstation":
		platform = "psn"
	case "xboxone", "xbox":
		platform = "xbl"
	case "epicgames":
		platform = "epic"
	case "nintendo":
		platform = "switch"
	}
	// Steam uses its numeric Steam64 ID; all other platforms use the display name.
	lookup := rest
	if platform != "steam" && playerName != "" {
		lookup = playerName
	}

	// Nintendo Switch (and some other platforms) masks player identities with
	// asterisks when cross-platform privacy is on. Skip these — TRN will 403.
	if isAllAsterisks(lookup) {
		jsonError(w, 400, "masked player name")
		return
	}

	playerID := url.PathEscape(lookup)
	platform = url.PathEscape(platform)

	// Check DB cache first.
	ttl := time.Duration(s.cfg.TrackerCacheTTLMinutes) * time.Minute
	if ttl > 0 {
		dataJSON, fetchedAt, found, err := s.db.GetTrackerCache(id)
		if err == nil && found && time.Since(fetchedAt) < ttl {
			log.Printf("[tracker] %s — cache hit (age %s)", id, time.Since(fetchedAt).Round(time.Second))
			writeTrackerResponse(w, true, fetchedAt, json.RawMessage(dataJSON))
			return
		}
	}

	trnURL := fmt.Sprintf("https://api.tracker.gg/api/v2/rocket-league/standard/profile/%s/%s", platform, playerID)
	httpClient := &http.Client{Timeout: 12 * time.Second}

	// Up to 3 attempts (1 initial + 2 retries). Each attempt goes through the
	// rate limiter so retries are just as human-paced as first tries.
	// Only 403/429 are worth retrying — those are rate-limit signals from TRN.
	// 404, 4xx are permanent failures; network errors abort immediately.
	const maxAttempts = 3
	var body []byte
	var finalStatus int
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Exponential backoff before re-queueing: 15 s, then 30 s.
			backoff := time.Duration(attempt) * 15 * time.Second
			log.Printf("[tracker] %s → %d — retry %d/%d in %s", trnURL, finalStatus, attempt, maxAttempts-1, backoff)
			select {
			case <-r.Context().Done():
				jsonError(w, 503, "request cancelled")
				return
			case <-time.After(backoff):
			}
		}

		s.trn.Wait() // every attempt, including retries, queues through the limiter

		req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, trnURL, nil)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

		resp, err := httpClient.Do(req)
		if err != nil {
			jsonError(w, 502, err.Error())
			return
		}
		body, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
		finalStatus = resp.StatusCode
		retry := ""
		if attempt > 0 {
			retry = fmt.Sprintf(" [retry %d]", attempt)
		}
		log.Printf("[tracker] %s → %d (%d bytes)%s", trnURL, finalStatus, len(body), retry)

		if (finalStatus == 403 || finalStatus == 429) && attempt < maxAttempts-1 {
			continue // rate-limited — back off and retry
		}
		break
	}

	if finalStatus != http.StatusOK {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(finalStatus)
		_, _ = w.Write(body)
		return
	}

	var trnResp struct {
		Data json.RawMessage `json:"data"`
	}
	if jsonErr := json.Unmarshal(body, &trnResp); jsonErr != nil || trnResp.Data == nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
		return
	}

	now := time.Now().UTC()
	_ = s.db.UpsertTrackerCache(id, string(trnResp.Data))
	writeTrackerResponse(w, false, now, trnResp.Data)
}

func writeTrackerResponse(w http.ResponseWriter, cached bool, fetchedAt time.Time, data json.RawMessage) {
	w.Header().Set("Content-Type", "application/json")
	cachedStr := "false"
	if cached {
		cachedStr = "true"
	}
	fmt.Fprintf(w, `{"cached":%s,"fetched_at":%q,"data":%s}`, cachedStr, fetchedAt.UTC().Format(time.RFC3339), data)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("writeJSON: %v", err)
	}
}
