package hub_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"OOF_RL/internal/hub"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// dialHub starts a test HTTP server that registers each connection with h,
// dials it, and blocks until registration is confirmed before returning.
func dialHub(t *testing.T, h *hub.Hub) *websocket.Conn {
	t.Helper()
	registered := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(conn)
		defer h.Unregister(conn)
		registered <- struct{}{}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })

	select {
	case <-registered:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for hub registration")
	}
	return client
}

func TestHubBroadcastToSingleClient(t *testing.T) {
	h := hub.New()
	client := dialHub(t, h)

	msg := []byte(`{"Event":"test"}`)
	h.Broadcast(msg)

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got, err := client.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage: %v", err)
	}
	if string(got) != string(msg) {
		t.Errorf("got %s, want %s", got, msg)
	}
}

func TestHubBroadcastToMultipleClients(t *testing.T) {
	h := hub.New()
	c1 := dialHub(t, h)
	c2 := dialHub(t, h)

	msg := []byte(`{"Event":"multi"}`)
	h.Broadcast(msg)

	deadline := time.Now().Add(2 * time.Second)
	for i, c := range []*websocket.Conn{c1, c2} {
		c.SetReadDeadline(deadline)
		_, got, err := c.ReadMessage()
		if err != nil {
			t.Fatalf("client %d ReadMessage: %v", i+1, err)
		}
		if string(got) != string(msg) {
			t.Errorf("client %d: got %s, want %s", i+1, got, msg)
		}
	}
}

func TestHubBroadcastWithNoClients(t *testing.T) {
	h := hub.New()
	// Must not panic when there are no registered clients.
	h.Broadcast([]byte(`{"Event":"empty"}`))
}

func TestHubUnregisterStopsDelivery(t *testing.T) {
	h := hub.New()
	unregistered := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(conn)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				h.Unregister(conn)
				close(unregistered)
				return
			}
		}
	}))
	defer srv.Close()

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}

	time.Sleep(20 * time.Millisecond) // let Register run
	client.Close()

	select {
	case <-unregistered:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Unregister")
	}

	// Broadcast after Unregister should not panic.
	h.Broadcast([]byte(`{"Event":"post-unregister"}`))
}

func TestHubBroadcastDropsDeadClient(t *testing.T) {
	h := hub.New()
	alive := dialHub(t, h)

	// Register a connection whose server-side conn we can close directly to
	// simulate a dead client that is still in the hub map.
	serverConnCh := make(chan *websocket.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		h.Register(conn)
		serverConnCh <- conn
		// No Unregister defer — let Broadcast detect and clean up the dead conn.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)

	deadURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	deadClient, _, err := websocket.DefaultDialer.Dial(deadURL, nil)
	if err != nil {
		t.Fatalf("dial dead client: %v", err)
	}
	t.Cleanup(func() { deadClient.Close() })

	serverConn := <-serverConnCh
	serverConn.Close() // make the server-side conn fail on next write

	msg := []byte(`{"Event":"after-dead"}`)
	done := make(chan struct{})
	go func() {
		h.Broadcast(msg)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Broadcast blocked on dead client")
	}

	// Alive client must still receive the message.
	alive.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, got, err := alive.ReadMessage()
	if err != nil {
		t.Fatalf("alive client ReadMessage: %v", err)
	}
	if string(got) != string(msg) {
		t.Errorf("got %s, want %s", got, msg)
	}
}

func TestHubMultipleBroadcasts(t *testing.T) {
	h := hub.New()
	client := dialHub(t, h)

	messages := []string{
		`{"Event":"first"}`,
		`{"Event":"second"}`,
		`{"Event":"third"}`,
	}
	for _, m := range messages {
		h.Broadcast([]byte(m))
	}

	client.SetReadDeadline(time.Now().Add(2 * time.Second))
	for _, want := range messages {
		_, got, err := client.ReadMessage()
		if err != nil {
			t.Fatalf("ReadMessage: %v", err)
		}
		if string(got) != want {
			t.Errorf("got %s, want %s", got, want)
		}
	}
}