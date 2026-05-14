package rl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"OOF_RL/internal/config"
	"OOF_RL/internal/events"
	"OOF_RL/internal/hub"
	"OOF_RL/internal/mmr"
)

type Client struct {
	cfg      *config.Config
	hub      *hub.Hub
	kick     chan struct{}
	dispatch func(events.Envelope)
	// matchGuid and lastPlayers are tracked solely for raw-packet-capture metadata.
	matchGuid   string
	lastPlayers map[string]events.Player

	rawPacketCapture    []capturedPacket
	rawPacketGuid       string
	rawPacketStart      time.Time
	rawPacketEnd        time.Time
	rawPacketReason     string
	rawPacketClosedGuid string
	rawPacketClosedAt   time.Time
}

type capturedPacket struct {
	wire       []byte
	normalized []byte
}

type capturePlayerMeta struct {
	Name      string `json:"name"`
	PrimaryID string `json:"primary_id"`
	Platform  string `json:"platform"`
	Team      int    `json:"team"`
	TRNURL    string `json:"trn_url,omitempty"`
}

type captureMeta struct {
	MetaVersion         int                 `json:"meta_version"`
	MatchGuid           string              `json:"match_guid"`
	StartedAtUTC        string              `json:"started_at_utc"`
	EndedAtUTC          string              `json:"ended_at_utc"`
	EndReason           string              `json:"end_reason"`
	DurationMs          int64               `json:"duration_ms"`
	PacketCount         int                 `json:"packet_count"`
	ChunkSize           int                 `json:"chunk_size"`
	ChunkCount          int                 `json:"chunk_count"`
	NormalizedFiles     []string            `json:"normalized_files"`
	WireFiles           []string            `json:"wire_files"`
	CaptureDirectory    string              `json:"capture_directory"`
	CaptureDirectoryUTC string              `json:"capture_directory_utc"`
	Players             []capturePlayerMeta `json:"players,omitempty"`
}

// trnSlug returns the tracker.gg URL path segment for a given raw platform string.
// Xbox uses "xbl" in tracker.gg URLs even though the canonical Platform value is "xbox".
func trnSlug(raw string) string {
	p := mmr.NormalizePlatform(raw)
	if p == mmr.PlatformXbox {
		return "xbl"
	}
	return string(p)
}

// trnProfileURL builds a tracker.gg profile URL from a RL primary ID and display name.
func trnProfileURL(primaryID, name string) string {
	sep := strings.IndexAny(primaryID, "|:_")
	if sep < 1 {
		return ""
	}
	plat := trnSlug(strings.ToLower(primaryID[:sep]))
	rest := primaryID[sep+1:]
	if end := strings.IndexAny(rest, "|:_"); end >= 0 {
		rest = rest[:end]
	}
	// All asterisks = masked Switch identity — no profile to link.
	masked := len(rest) > 0
	for _, c := range rest {
		if c != '*' {
			masked = false
			break
		}
	}
	if masked {
		return ""
	}
	lookup := rest
	if plat != "steam" && name != "" {
		lookup = name
	}
	return "https://tracker.gg/rocket-league/profile/" +
		url.PathEscape(plat) + "/" + url.PathEscape(lookup)
}

type captureIndex struct {
	Version     int            `json:"version"`
	MatchGuid   string         `json:"match_guid"`
	PacketCount int            `json:"packet_count"`
	Markers     []indexMarker  `json:"markers"`
	EventCounts map[string]int `json:"event_counts"`
}

type indexMarker struct {
	PacketIndex int     `json:"packet_index"`
	Event       string  `json:"event"`
	TimeSeconds float64 `json:"time_seconds,omitempty"`
	GoalTime    float64 `json:"goal_time,omitempty"`
}

func New(cfg *config.Config, h *hub.Hub) *Client {
	return &Client{cfg: cfg, hub: h, kick: make(chan struct{}, 1)}
}

func (c *Client) SetDispatch(fn func(events.Envelope)) { c.dispatch = fn }

func (c *Client) Reconnect() {
	select {
	case c.kick <- struct{}{}:
	default:
	}
}

func (c *Client) Run() {
	for {
		if err := c.connect(); err != nil {
			log.Printf("[rl] disconnected: %v — retrying in 5s", err)
		}
		select {
		case <-time.After(5 * time.Second):
		case <-c.kick:
			log.Printf("[rl] reconnecting due to config change")
		}
	}
}

func (c *Client) resolvePort() int {
	ini, err := config.ReadINI(c.cfg.RLInstallPath)
	if err == nil && ini.Port > 0 {
		return ini.Port
	}
	return 49123
}

func (c *Client) connect() error {
	port := c.resolvePort()
	addr := fmt.Sprintf("localhost:%d", port)
	log.Printf("[rl] connecting to %s", addr)

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()
	defer c.abortRawPacketCapture()
	log.Printf("[rl] connected")

	c.broadcastStatus(true)
	defer c.broadcastStatus(false)

	go func() {
		<-c.kick
		conn.Close()
		select {
		case c.kick <- struct{}{}:
		default:
		}
	}()

	dec := json.NewDecoder(conn)
	for {
		// RL sends concatenated JSON objects over raw TCP
		var wire struct {
			Event string          `json:"Event"`
			Data  json.RawMessage `json:"Data"`
		}
		if err := dec.Decode(&wire); err != nil {
			return err
		}
		wireEnv := events.Envelope{Event: wire.Event, Data: wire.Data}
		rawWire, _ := json.Marshal(wireEnv)

		// Data arrives as a JSON-encoded string; unwrap it to a raw object
		data := wire.Data
		var dataStr string
		if json.Unmarshal(wire.Data, &dataStr) == nil {
			data = json.RawMessage(dataStr)
		}

		env := events.Envelope{Event: wire.Event, Data: data}

		// Re-encode normalised envelope for browser clients
		normalized, _ := json.Marshal(env)
		c.handle(env, rawWire, normalized)
		c.hub.Broadcast(normalized)
	}
}

func (c *Client) broadcastStatus(connected bool) {
	b, _ := json.Marshal(map[string]any{
		"Event": "_Status",
		"Data":  map[string]any{"connected": connected},
	})
	c.hub.Broadcast(b)
}

func (c *Client) handle(env events.Envelope, rawWire, normalized []byte) {
	capturedBefore := c.captureRawPacketIfActive(rawWire, normalized)

	switch env.Event {
	case "MatchCreated", "MatchInitialized":
		c.handleMatchStart(env)
		if !capturedBefore {
			c.captureRawPacketIfActive(rawWire, normalized)
		}
	case "UpdateState":
		c.handleUpdateState(env)
		if !capturedBefore {
			c.captureRawPacketIfActive(rawWire, normalized)
		}
	case "MatchEnded":
		c.handleMatchEnded(env)
	case "MatchDestroyed":
		c.closeRawPacketCapture("MatchDestroyed")
		c.lastPlayers = nil
		c.matchGuid = ""
	}
	if c.dispatch != nil {
		c.dispatch(env)
	}
}

func (c *Client) handleMatchStart(env events.Envelope) {
	var d events.MatchGuidData
	if err := json.Unmarshal(env.Data, &d); err == nil {
		c.startRawPacketCapture(d.MatchGuid)
		if d.MatchGuid != "" {
			c.matchGuid = d.MatchGuid
		}
	}
}

func (c *Client) handleUpdateState(env events.Envelope) {
	var d events.UpdateStateData
	if err := json.Unmarshal(env.Data, &d); err != nil {
		return
	}
	c.startRawPacketCapture(d.MatchGuid)
	if d.MatchGuid != "" {
		c.matchGuid = d.MatchGuid
	}

	// Accumulate player state for raw-packet-capture metadata only.
	if len(d.Players) > 0 {
		if c.lastPlayers == nil {
			c.lastPlayers = make(map[string]events.Player)
		}
		for _, p := range d.Players {
			if p.PrimaryId != "" {
				c.lastPlayers[p.PrimaryId] = p
			}
		}
	}
}

func (c *Client) handleMatchEnded(_ events.Envelope) {
	c.closeRawPacketCapture("MatchEnded")
	c.lastPlayers = nil
	c.matchGuid = ""
}

func (c *Client) startRawPacketCapture(matchGuid string) {
	if !c.cfg.Storage.RawPackets || matchGuid == "" {
		return
	}
	if c.rawPacketClosedGuid == matchGuid && time.Since(c.rawPacketClosedAt) < 10*time.Second {
		return
	}
	if c.rawPacketGuid == "" {
		c.rawPacketGuid = matchGuid
		c.rawPacketStart = time.Now()
		c.rawPacketEnd = time.Time{}
		c.rawPacketReason = ""
	}
}

func (c *Client) closeRawPacketCapture(reason string) {
	if c.rawPacketGuid != "" {
		c.rawPacketClosedGuid = c.rawPacketGuid
		c.rawPacketClosedAt = time.Now()
	}
	c.rawPacketEnd = time.Now()
	c.rawPacketReason = reason
	c.flushRawPacketCapture()
}

func (c *Client) captureRawPacketIfActive(rawWire, normalized []byte) bool {
	if !c.cfg.Storage.RawPackets || c.rawPacketGuid == "" {
		return false
	}
	p := capturedPacket{
		wire:       append([]byte(nil), rawWire...),
		normalized: append([]byte(nil), normalized...),
	}
	c.rawPacketCapture = append(c.rawPacketCapture, p)
	return true
}

func (c *Client) flushRawPacketCapture() {
	if len(c.rawPacketCapture) == 0 {
		c.resetRawPacketCaptureState()
		return
	}

	baseDir := c.cfg.CapturesDir()
	matchPart := sanitizePathPart(c.rawPacketGuid)
	if matchPart == "" {
		matchPart = "unknown_match"
	}
	start := c.rawPacketStart
	if start.IsZero() {
		start = time.Now()
	}
	dir := filepath.Join(baseDir, fmt.Sprintf("%s_%s", matchPart, start.UTC().Format("20060102_150405")))
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("[rl] create capture dir: %v", err)
		c.resetRawPacketCaptureState()
		return
	}

	const chunkSize = 10000
	normalizedFiles := make([]string, 0)
	wireFiles := make([]string, 0)
	for i := 0; i < len(c.rawPacketCapture); i += chunkSize {
		end := i + chunkSize
		if end > len(c.rawPacketCapture) {
			end = len(c.rawPacketCapture)
		}

		var buf bytes.Buffer
		for _, pkt := range c.rawPacketCapture[i:end] {
			buf.Write(pkt.normalized)
			buf.WriteByte('\n')
		}

		normalizedName := fmt.Sprintf("packets_normalized_%03d.ndjson", (i/chunkSize)+1)
		filePath := filepath.Join(dir, normalizedName)
		if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			log.Printf("[rl] write normalized capture file: %v", err)
			break
		}
		normalizedFiles = append(normalizedFiles, normalizedName)

		buf.Reset()
		for _, pkt := range c.rawPacketCapture[i:end] {
			buf.Write(pkt.wire)
			buf.WriteByte('\n')
		}

		wireName := fmt.Sprintf("packets_wire_%03d.ndjson", (i/chunkSize)+1)
		filePath = filepath.Join(dir, wireName)
		if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
			log.Printf("[rl] write wire capture file: %v", err)
			break
		}
		wireFiles = append(wireFiles, wireName)
	}

	end := c.rawPacketEnd
	if end.IsZero() {
		end = time.Now()
	}
	duration := end.Sub(start)
	if duration < 0 {
		duration = 0
	}
	reason := c.rawPacketReason
	if reason == "" {
		reason = "flush"
	}
	chunkCount := len(normalizedFiles)
	if len(wireFiles) < chunkCount {
		chunkCount = len(wireFiles)
	}

	var players []capturePlayerMeta
	for _, p := range c.lastPlayers {
		sep := strings.IndexAny(p.PrimaryId, "|:_")
		plat := ""
		if sep >= 1 {
			plat = trnSlug(strings.ToLower(p.PrimaryId[:sep]))
		}
		players = append(players, capturePlayerMeta{
			Name:      p.Name,
			PrimaryID: p.PrimaryId,
			Platform:  plat,
			Team:      p.TeamNum,
			TRNURL:    trnProfileURL(p.PrimaryId, p.Name),
		})
	}

	meta := captureMeta{
		MetaVersion:         1,
		MatchGuid:           c.rawPacketGuid,
		StartedAtUTC:        start.UTC().Format(time.RFC3339Nano),
		EndedAtUTC:          end.UTC().Format(time.RFC3339Nano),
		EndReason:           reason,
		DurationMs:          duration.Milliseconds(),
		PacketCount:         len(c.rawPacketCapture),
		ChunkSize:           chunkSize,
		ChunkCount:          chunkCount,
		NormalizedFiles:     normalizedFiles,
		WireFiles:           wireFiles,
		CaptureDirectory:    dir,
		CaptureDirectoryUTC: start.UTC().Format("20060102_150405"),
		Players:             players,
	}
	if b, err := json.MarshalIndent(meta, "", "  "); err != nil {
		log.Printf("[rl] encode capture meta: %v", err)
	} else if err := os.WriteFile(filepath.Join(dir, "capture_meta.json"), b, 0644); err != nil {
		log.Printf("[rl] write capture meta: %v", err)
	}

	idx := buildCaptureIndex(c.rawPacketGuid, c.rawPacketCapture)
	if b, err := json.MarshalIndent(idx, "", "  "); err != nil {
		log.Printf("[rl] encode capture index: %v", err)
	} else if err := os.WriteFile(filepath.Join(dir, "capture_index.json"), b, 0644); err != nil {
		log.Printf("[rl] write capture index: %v", err)
	}

	log.Printf("[rl] wrote %d raw packets to %s", len(c.rawPacketCapture), dir)
	c.resetRawPacketCaptureState()
}

func (c *Client) abortRawPacketCapture() {
	if len(c.rawPacketCapture) > 0 {
		log.Printf("[rl] dropping %d buffered raw packets (match did not end cleanly)", len(c.rawPacketCapture))
	}
	c.resetRawPacketCaptureState()
}

func (c *Client) resetRawPacketCaptureState() {
	c.rawPacketCapture = nil
	c.rawPacketGuid = ""
	c.rawPacketStart = time.Time{}
	c.rawPacketEnd = time.Time{}
	c.rawPacketReason = ""
}

func buildCaptureIndex(matchGuid string, packets []capturedPacket) captureIndex {
	idx := captureIndex{
		Version:     1,
		MatchGuid:   matchGuid,
		PacketCount: len(packets),
		Markers:     make([]indexMarker, 0),
		EventCounts: make(map[string]int),
	}

	for i, pkt := range packets {
		var env struct {
			Event string          `json:"Event"`
			Data  json.RawMessage `json:"Data"`
		}
		if err := json.Unmarshal(pkt.normalized, &env); err != nil {
			continue
		}
		idx.EventCounts[env.Event]++

		switch env.Event {
		case "MatchCreated", "MatchInitialized", "MatchEnded", "MatchDestroyed":
			idx.Markers = append(idx.Markers, indexMarker{PacketIndex: i, Event: env.Event})
		case "GoalScored":
			var d struct {
				GoalTime float64 `json:"GoalTime"`
			}
			_ = json.Unmarshal(env.Data, &d)
			idx.Markers = append(idx.Markers, indexMarker{PacketIndex: i, Event: env.Event, GoalTime: d.GoalTime})
		case "UpdateState":
			var d struct {
				Game struct {
					TimeSeconds int `json:"TimeSeconds"`
				} `json:"Game"`
			}
			if json.Unmarshal(env.Data, &d) == nil && (i == 0 || i%300 == 0) {
				idx.Markers = append(idx.Markers, indexMarker{PacketIndex: i, Event: env.Event, TimeSeconds: float64(d.Game.TimeSeconds)})
			}
		}
	}

	return idx
}

func sanitizePathPart(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"<", "_", ">", "_", ":", "_", `"`, "_",
		"/", "_", "\\", "_", "|", "_", "?", "_", "*", "_", " ", "_",
	)
	return replacer.Replace(s)
}
