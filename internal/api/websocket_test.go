package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func setupTestHub(t *testing.T) (*Hub, *httptest.Server) {
	t.Helper()
	hub := NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	server := httptest.NewServer(mux)
	return hub, server
}

func dialWS(t *testing.T, serverURL string) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket: %v", err)
	}
	return conn
}

func TestClientConnection(t *testing.T) {
	hub, server := setupTestHub(t)
	defer server.Close()
	defer hub.Stop()

	conn := dialWS(t, server.URL)
	defer conn.Close()

	// Give the hub a moment to register the client.
	time.Sleep(50 * time.Millisecond)

	if got := hub.Clients(); got != 1 {
		t.Fatalf("expected 1 client, got %d", got)
	}

	conn.Close()
	time.Sleep(50 * time.Millisecond)

	if got := hub.Clients(); got != 0 {
		t.Fatalf("expected 0 clients after disconnect, got %d", got)
	}
}

func TestBroadcastMessage(t *testing.T) {
	hub, server := setupTestHub(t)
	defer server.Close()
	defer hub.Stop()

	conn1 := dialWS(t, server.URL)
	defer conn1.Close()
	conn2 := dialWS(t, server.URL)
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	msg := WSMessage{Type: "test.event", Data: "hello"}
	hub.Broadcast(msg)

	for i, conn := range []*websocket.Conn{conn1, conn2} {
		conn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, raw, err := conn.ReadMessage()
		if err != nil {
			t.Fatalf("client %d: failed to read message: %v", i, err)
		}
		var got WSMessage
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("client %d: failed to unmarshal: %v", i, err)
		}
		if got.Type != "test.event" {
			t.Fatalf("client %d: expected type test.event, got %s", i, got.Type)
		}
	}
}

func TestHubClientManagement(t *testing.T) {
	hub, server := setupTestHub(t)
	defer server.Close()
	defer hub.Stop()

	conns := make([]*websocket.Conn, 3)
	for i := range conns {
		conns[i] = dialWS(t, server.URL)
	}
	time.Sleep(50 * time.Millisecond)

	if got := hub.Clients(); got != 3 {
		t.Fatalf("expected 3 clients, got %d", got)
	}

	// Close one connection and verify count decreases.
	conns[1].Close()
	time.Sleep(50 * time.Millisecond)

	if got := hub.Clients(); got != 2 {
		t.Fatalf("expected 2 clients after one disconnect, got %d", got)
	}

	// Clean up remaining.
	conns[0].Close()
	conns[2].Close()
	time.Sleep(50 * time.Millisecond)

	if got := hub.Clients(); got != 0 {
		t.Fatalf("expected 0 clients, got %d", got)
	}
}

func TestBroadcastToSubscribers(t *testing.T) {
	hub, server := setupTestHub(t)
	defer server.Close()
	defer hub.Stop()

	conn1 := dialWS(t, server.URL)
	defer conn1.Close()
	conn2 := dialWS(t, server.URL)
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	// conn1 subscribes to "price.update" only
	subMsg, _ := json.Marshal(WSMessage{Type: "subscribe", Data: "price.update"})
	conn1.WriteMessage(websocket.TextMessage, subMsg)
	time.Sleep(50 * time.Millisecond)

	// Broadcast an event that only conn1 is subscribed to.
	hub.BroadcastToSubscribers("price.update", map[string]any{"price": 100})

	conn1.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, raw, err := conn1.ReadMessage()
	if err != nil {
		t.Fatalf("subscribed client should receive message: %v", err)
	}
	var got WSMessage
	json.Unmarshal(raw, &got)
	if got.Type != "price.update" {
		t.Fatalf("expected type price.update, got %s", got.Type)
	}

	// conn2 has no subscriptions so it receives all events.
	conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _, err = conn2.ReadMessage()
	if err != nil {
		t.Fatalf("unsubscribed client (wildcard) should receive message: %v", err)
	}
}
