package wasmhost

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"OOF_RL/internal/db"
	"OOF_RL/internal/hub"
	sdk "github.com/erosas/oof-plugin-sdk"
)

// --- host_db_exec ---

// TestHostDBExec_Success creates a real in-memory DB, executes an INSERT via the
// host import, and verifies rows-affected comes back as 1.
func TestHostDBExec_Success(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	if _, err := database.Conn().Exec(`CREATE TABLE t (x TEXT)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: database}
	mod := newMemModule(t)
	ctx := context.Background()

	sqlStr := `INSERT INTO t(x) VALUES(?)`
	args := `["hello"]`
	if !mod.Memory().Write(0, []byte(sqlStr)) || !mod.Memory().Write(256, []byte(args)) {
		t.Fatal("write to memory")
	}
	outOff := uint32(1024)
	n := p.hostDBExec(ctx, mod, 0, uint32(len(sqlStr)), 256, uint32(len(args)), outOff, 64)
	if n == 0 {
		t.Fatal("hostDBExec returned 0")
	}
	data, _ := mod.Memory().Read(outOff, n)
	var rowsAffected int64
	if err := json.Unmarshal(data, &rowsAffected); err != nil {
		t.Fatalf("unmarshal rows: %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("rows affected: got %d, want 1", rowsAffected)
	}
}

// TestHostDBExec_BadSQL verifies that a malformed SQL statement returns 0.
func TestHostDBExec_BadSQL(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: database}
	mod := newMemModule(t)
	ctx := context.Background()

	bad := `NOT VALID SQL !!!`
	args := `[]`
	mod.Memory().Write(0, []byte(bad))
	mod.Memory().Write(256, []byte(args))
	n := p.hostDBExec(ctx, mod, 0, uint32(len(bad)), 256, uint32(len(args)), 1024, 64)
	if n != 0 {
		t.Errorf("bad SQL should return 0, got %d", n)
	}
}

// --- host_db_query ---

// TestHostDBQuery_Success inserts rows and reads them back via the host import.
func TestHostDBQuery_Success(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	database.Conn().Exec(`CREATE TABLE items (name TEXT, val INTEGER)`)
	database.Conn().Exec(`INSERT INTO items VALUES('alpha', 1), ('beta', 2)`)

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: database}
	mod := newMemModule(t)
	ctx := context.Background()

	sqlStr := `SELECT name, val FROM items ORDER BY val`
	args := `[]`
	mod.Memory().Write(0, []byte(sqlStr))
	mod.Memory().Write(256, []byte(args))
	outOff := uint32(1024)
	n := p.hostDBQuery(ctx, mod, 0, uint32(len(sqlStr)), 256, uint32(len(args)), outOff, 32*1024)
	if n == 0 {
		t.Fatal("hostDBQuery returned 0")
	}
	data, _ := mod.Memory().Read(outOff, n)
	var rows []map[string]any
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("unmarshal rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("row count: got %d, want 2", len(rows))
	}
	if rows[0]["name"] != "alpha" {
		t.Errorf("row[0].name: got %v", rows[0]["name"])
	}
}

// TestHostDBQuery_EmptyResult verifies an empty result set returns an empty JSON array.
func TestHostDBQuery_EmptyResult(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(func() { database.Close() })
	database.Conn().Exec(`CREATE TABLE empty_t (x TEXT)`)

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, database: database}
	mod := newMemModule(t)
	ctx := context.Background()

	sqlStr := `SELECT x FROM empty_t`
	args := `[]`
	mod.Memory().Write(0, []byte(sqlStr))
	mod.Memory().Write(256, []byte(args))
	n := p.hostDBQuery(ctx, mod, 0, uint32(len(sqlStr)), 256, uint32(len(args)), 1024, 32*1024)
	if n == 0 {
		t.Fatal("hostDBQuery returned 0 for empty result")
	}
	data, _ := mod.Memory().Read(1024, n)
	var rows []map[string]any
	json.Unmarshal(data, &rows)
	if len(rows) != 0 {
		t.Errorf("want empty slice, got %d rows", len(rows))
	}
}

// --- host_http_fetch ---

// TestHostHTTPFetch_Success verifies a real HTTP GET request via the host import.
func TestHostHTTPFetch_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, httpClient: &http.Client{Timeout: 5 * time.Second}}
	mod := newMemModule(t)
	ctx := context.Background()

	reqJSON, _ := json.Marshal(sdk.HTTPFetchRequest{Method: "GET", URL: srv.URL})
	mod.Memory().Write(0, reqJSON)
	outOff := uint32(4096)
	n := p.hostHTTPFetch(ctx, mod, 0, uint32(len(reqJSON)), outOff, 32*1024)
	if n == 0 {
		t.Fatal("hostHTTPFetch returned 0")
	}
	data, _ := mod.Memory().Read(outOff, n)
	var result sdk.HTTPFetchResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if result.Status != 200 {
		t.Errorf("status: got %d, want 200", result.Status)
	}
	if result.Error != "" {
		t.Errorf("unexpected error: %s", result.Error)
	}
	if result.Body != `{"ok":true}` {
		t.Errorf("body: got %q", result.Body)
	}
}

// TestHostHTTPFetch_BadURL verifies an unreachable URL returns a non-empty Error field.
func TestHostHTTPFetch_BadURL(t *testing.T) {
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, httpClient: &http.Client{Timeout: 5 * time.Second}}
	mod := newMemModule(t)
	ctx := context.Background()

	reqJSON, _ := json.Marshal(sdk.HTTPFetchRequest{Method: "GET", URL: "http://127.0.0.1:1"})
	mod.Memory().Write(0, reqJSON)
	outOff := uint32(4096)
	n := p.hostHTTPFetch(ctx, mod, 0, uint32(len(reqJSON)), outOff, 32*1024)
	if n == 0 {
		t.Fatal("hostHTTPFetch returned 0 (should write an error result)")
	}
	data, _ := mod.Memory().Read(outOff, n)
	var result sdk.HTTPFetchResult
	json.Unmarshal(data, &result)
	if result.Error == "" {
		t.Error("expected non-empty Error for unreachable URL")
	}
}

// TestHostHTTPFetch_DefaultMethod verifies empty Method defaults to GET.
func TestHostHTTPFetch_DefaultMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "wrong method", 400)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, httpClient: &http.Client{Timeout: 5 * time.Second}}
	mod := newMemModule(t)
	ctx := context.Background()

	reqJSON, _ := json.Marshal(sdk.HTTPFetchRequest{URL: srv.URL}) // no Method
	mod.Memory().Write(0, reqJSON)
	n := p.hostHTTPFetch(ctx, mod, 0, uint32(len(reqJSON)), 4096, 32*1024)
	if n == 0 {
		t.Fatal("hostHTTPFetch returned 0")
	}
	data, _ := mod.Memory().Read(4096, n)
	var result sdk.HTTPFetchResult
	json.Unmarshal(data, &result)
	if result.Status != 200 {
		t.Errorf("status: got %d, want 200", result.Status)
	}
}

// --- host_http_download ---

func TestHostHTTPDownload_Success(t *testing.T) {
	payload := []byte("release zip bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}))
	defer srv.Close()

	dataDir := t.TempDir()
	p := &Plugin{
		meta:       sdk.PluginMeta{ID: "test"},
		httpClient: &http.Client{Timeout: 5 * time.Second},
		mounts:     map[string]string{"/data": dataDir},
	}
	mod := newMemModule(t)
	ctx := context.Background()

	reqJSON, _ := json.Marshal(sdk.HTTPDownloadRequest{
		URL:         srv.URL,
		Destination: "/data/downloads/OOF_RL.zip",
	})
	mod.Memory().Write(0, reqJSON)
	n := p.hostHTTPDownload(ctx, mod, 0, uint32(len(reqJSON)), 4096, 32*1024)
	if n == 0 {
		t.Fatal("hostHTTPDownload returned 0")
	}
	data, _ := mod.Memory().Read(4096, n)
	var result sdk.HTTPDownloadResult
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	wantHash := fmt.Sprintf("%x", sha256.Sum256(payload))
	if result.Status != http.StatusOK || result.SHA256 != wantHash || result.Bytes != int64(len(payload)) {
		t.Fatalf("download result: got %+v", result)
	}
	saved, err := os.ReadFile(filepath.Join(dataDir, "downloads", "OOF_RL.zip"))
	if err != nil {
		t.Fatalf("read downloaded file: %v", err)
	}
	if string(saved) != string(payload) {
		t.Fatalf("downloaded payload mismatch: got %q", saved)
	}
}

func TestHostHTTPDownloadRejectsEscapingDestination(t *testing.T) {
	p := &Plugin{
		meta:       sdk.PluginMeta{ID: "test"},
		httpClient: &http.Client{Timeout: 5 * time.Second},
		mounts:     map[string]string{"/data": t.TempDir()},
	}
	mod := newMemModule(t)
	reqJSON, _ := json.Marshal(sdk.HTTPDownloadRequest{
		URL:         "https://example.test/file.zip",
		Destination: "/other/file.zip",
	})
	mod.Memory().Write(0, reqJSON)
	n := p.hostHTTPDownload(context.Background(), mod, 0, uint32(len(reqJSON)), 4096, 32*1024)
	if n == 0 {
		t.Fatal("hostHTTPDownload should return an error result")
	}
	data, _ := mod.Memory().Read(4096, n)
	var result sdk.HTTPDownloadResult
	json.Unmarshal(data, &result)
	if result.Error == "" {
		t.Fatal("expected path error")
	}
}

// --- host_broadcast_ws ---

// TestHostBroadcastWS_WithHub verifies that hostBroadcastWS does not panic with a real hub.
func TestHostBroadcastWS_WithHub(t *testing.T) {
	h := hub.New()
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, hub: h}
	mod := newMemModule(t)
	msg := []byte(`{"Event":"test"}`)
	mod.Memory().Write(0, msg)
	p.hostBroadcastWS(context.Background(), mod, 0, uint32(len(msg))) // must not panic
}

// TestHostBroadcastWS_MemReadFail verifies zero-length reads are silently ignored.
func TestHostBroadcastWS_MemReadFail(t *testing.T) {
	h := hub.New()
	p := &Plugin{meta: sdk.PluginMeta{ID: "test"}, hub: h}
	mod := newMemModule(t)
	// ptr=0, length=0 → Memory.Read returns ok=false → early return, no panic.
	p.hostBroadcastWS(context.Background(), mod, 0, 0)
}
