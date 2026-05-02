package rl

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"OOF_RL/internal/config"
	"OOF_RL/internal/events"
)

func TestHandleMatchStartBeginsRawCaptureWithoutMatchMetadata(t *testing.T) {
	cfg := config.Defaults()
	cfg.Storage.MatchMetadata = false
	cfg.Storage.RawPackets = true
	cfg.Storage.RawPacketsDir = t.TempDir()

	c := New(&cfg, nil, nil)
	data, _ := json.Marshal(events.MatchGuidData{MatchGuid: "guid-abc"})
	c.handleMatchStart(events.Envelope{Event: "MatchInitialized", Data: data})

	if c.rawPacketGuid != "guid-abc" {
		t.Fatalf("rawPacketGuid: got %q, want guid-abc", c.rawPacketGuid)
	}
}

func TestFlushRawPacketCaptureWritesNDJSON(t *testing.T) {
	cfg := config.Defaults()
	cfg.Storage.RawPackets = true
	cfg.Storage.RawPacketsDir = t.TempDir()

	c := New(&cfg, nil, nil)
	c.startRawPacketCapture("guid:one")
	c.captureRawPacketIfActive([]byte(`{"Event":"A","Data":"{\"v\":1}"}`), []byte(`{"Event":"A","Data":{"v":1}}`))
	c.captureRawPacketIfActive([]byte(`{"Event":"B","Data":"{\"v\":2}"}`), []byte(`{"Event":"B","Data":{"v":2}}`))
	c.flushRawPacketCapture()

	entries, err := os.ReadDir(cfg.Storage.RawPacketsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("expected one match capture directory, got %d entries", len(entries))
	}
	if !strings.Contains(entries[0].Name(), "guid_one_") {
		t.Fatalf("capture directory should contain sanitized match guid, got %q", entries[0].Name())
	}

	normPath := filepath.Join(cfg.Storage.RawPacketsDir, entries[0].Name(), "packets_normalized_001.ndjson")
	data, err := os.ReadFile(normPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 packets, got %d", len(lines))
	}

	wirePath := filepath.Join(cfg.Storage.RawPacketsDir, entries[0].Name(), "packets_wire_001.ndjson")
	data, err = os.ReadFile(wirePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	lines = strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 wire packets, got %d", len(lines))
	}

	metaPath := filepath.Join(cfg.Storage.RawPacketsDir, entries[0].Name(), "capture_meta.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("ReadFile capture_meta.json: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("Unmarshal capture_meta.json: %v", err)
	}
	if packetCount, _ := meta["packet_count"].(float64); int(packetCount) != 2 {
		t.Fatalf("meta packet_count: got %v, want 2", meta["packet_count"])
	}
	if chunkCount, _ := meta["chunk_count"].(float64); int(chunkCount) != 1 {
		t.Fatalf("meta chunk_count: got %v, want 1", meta["chunk_count"])
	}

	idxPath := filepath.Join(cfg.Storage.RawPacketsDir, entries[0].Name(), "capture_index.json")
	idxBytes, err := os.ReadFile(idxPath)
	if err != nil {
		t.Fatalf("ReadFile capture_index.json: %v", err)
	}
	var idx map[string]any
	if err := json.Unmarshal(idxBytes, &idx); err != nil {
		t.Fatalf("Unmarshal capture_index.json: %v", err)
	}
	if packetCount, _ := idx["packet_count"].(float64); int(packetCount) != 2 {
		t.Fatalf("index packet_count: got %v, want 2", idx["packet_count"])
	}
}

func TestFlushRawPacketCaptureChunksLargeOutput(t *testing.T) {
	cfg := config.Defaults()
	cfg.Storage.RawPackets = true
	cfg.Storage.RawPacketsDir = t.TempDir()

	c := New(&cfg, nil, nil)
	c.startRawPacketCapture("guid-chunks")
	for i := 0; i < 10001; i++ {
		c.captureRawPacketIfActive([]byte(`{"Event":"UpdateState","Data":"{}"}`), []byte(`{"Event":"UpdateState","Data":{}}`))
	}
	c.flushRawPacketCapture()

	entries, err := os.ReadDir(cfg.Storage.RawPacketsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 || !entries[0].IsDir() {
		t.Fatalf("expected one match capture directory, got %d entries", len(entries))
	}

	matchDir := filepath.Join(cfg.Storage.RawPacketsDir, entries[0].Name())
	if _, err := os.Stat(filepath.Join(matchDir, "packets_normalized_001.ndjson")); err != nil {
		t.Fatalf("packets_normalized_001.ndjson missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(matchDir, "packets_normalized_002.ndjson")); err != nil {
		t.Fatalf("packets_normalized_002.ndjson missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(matchDir, "packets_wire_001.ndjson")); err != nil {
		t.Fatalf("packets_wire_001.ndjson missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(matchDir, "packets_wire_002.ndjson")); err != nil {
		t.Fatalf("packets_wire_002.ndjson missing: %v", err)
	}

	metaPath := filepath.Join(matchDir, "capture_meta.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("capture_meta.json missing: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		t.Fatalf("Unmarshal capture_meta.json: %v", err)
	}
	if packetCount, _ := meta["packet_count"].(float64); int(packetCount) != 10001 {
		t.Fatalf("meta packet_count: got %v, want 10001", meta["packet_count"])
	}
	if chunkCount, _ := meta["chunk_count"].(float64); int(chunkCount) != 2 {
		t.Fatalf("meta chunk_count: got %v, want 2", meta["chunk_count"])
	}
}

func TestStartRawPacketCaptureSuppressesRecentlyClosedMatch(t *testing.T) {
	cfg := config.Defaults()
	cfg.Storage.RawPackets = true
	cfg.Storage.RawPacketsDir = t.TempDir()

	c := New(&cfg, nil, nil)
	c.startRawPacketCapture("guid-dup")
	c.captureRawPacketIfActive([]byte(`{"Event":"A","Data":"{}"}`), []byte(`{"Event":"A","Data":{}}`))
	c.closeRawPacketCapture("MatchEnded")

	entries, err := os.ReadDir(cfg.Storage.RawPacketsDir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one flushed capture dir, got %d", len(entries))
	}

	c.startRawPacketCapture("guid-dup")
	if c.rawPacketGuid != "" {
		t.Fatalf("expected duplicate guid capture to be suppressed, got %q", c.rawPacketGuid)
	}

	c.startRawPacketCapture("guid-new")
	if c.rawPacketGuid != "guid-new" {
		t.Fatalf("expected new guid capture to start, got %q", c.rawPacketGuid)
	}
}
