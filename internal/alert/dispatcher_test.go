package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/bot"
	"github.com/clawtrade/clawtrade/internal/engine"
)

func TestDispatcher_SendsTelegram(t *testing.T) {
	var received []string
	var mu sync.Mutex

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req bot.SendMessageRequest
		json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		received = append(received, req.Text)
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	tgBot := bot.NewTelegramBot("test-token")
	tgBot.SetBaseURL(ts.URL)

	db := testDB(t)
	defer db.Close()
	bus := engine.NewEventBus()

	d := NewDispatcher(tgBot, 12345, bus)

	a := Alert{ID: 1, Type: AlertTypePrice, Symbol: "BTC/USDT"}
	d.Dispatch(a, "BTC above 70k!")

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if len(received) != 1 {
		t.Fatalf("expected 1 Telegram message, got %d", len(received))
	}
	if received[0] == "" {
		t.Fatal("empty message")
	}
	mu.Unlock()
}

func TestDispatcher_PublishesEvent(t *testing.T) {
	bus := engine.NewEventBus()

	var triggered bool
	var mu sync.Mutex
	bus.Subscribe("alert.triggered", func(e engine.Event) {
		mu.Lock()
		triggered = true
		mu.Unlock()
	})

	d := NewDispatcher(nil, 0, bus)
	a := Alert{ID: 1, Type: AlertTypePrice, Symbol: "BTC/USDT"}
	d.Dispatch(a, "test message")

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if !triggered {
		t.Fatal("expected alert.triggered event")
	}
	mu.Unlock()
}
