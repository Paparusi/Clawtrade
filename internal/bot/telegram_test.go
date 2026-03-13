package bot

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNewTelegramBot(t *testing.T) {
	bot := NewTelegramBot("test-token-123")
	if bot == nil {
		t.Fatal("expected non-nil bot")
	}
	if bot.token != "test-token-123" {
		t.Errorf("expected token 'test-token-123', got %q", bot.token)
	}
	if bot.commands == nil {
		t.Error("expected commands map to be initialized")
	}
	if bot.allowedChats == nil {
		t.Error("expected allowedChats map to be initialized")
	}
	if bot.httpClient == nil {
		t.Error("expected httpClient to be initialized")
	}
}

func TestRegisterCommand(t *testing.T) {
	bot := NewTelegramBot("token")
	bot.RegisterCommand("test", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
		return "test response", nil
	})

	bot.mu.RLock()
	_, ok := bot.commands["test"]
	bot.mu.RUnlock()
	if !ok {
		t.Error("expected command 'test' to be registered")
	}
}

func TestListCommands(t *testing.T) {
	bot := NewTelegramBot("token")
	bot.RegisterCommand("beta", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) { return "", nil })
	bot.RegisterCommand("alpha", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) { return "", nil })
	bot.RegisterCommand("gamma", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) { return "", nil })

	cmds := bot.ListCommands()
	if len(cmds) != 3 {
		t.Fatalf("expected 3 commands, got %d", len(cmds))
	}
	if cmds[0] != "alpha" || cmds[1] != "beta" || cmds[2] != "gamma" {
		t.Errorf("expected sorted commands [alpha, beta, gamma], got %v", cmds)
	}
}

func TestAllowChatAndIsAllowed(t *testing.T) {
	bot := NewTelegramBot("token")

	if bot.IsAllowed(123) {
		t.Error("expected chat 123 to not be allowed initially")
	}

	bot.AllowChat(123)

	if !bot.IsAllowed(123) {
		t.Error("expected chat 123 to be allowed after AllowChat")
	}

	if bot.IsAllowed(456) {
		t.Error("expected chat 456 to not be allowed")
	}
}

func TestHandleUpdateStartCommand(t *testing.T) {
	var sentReq SendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	bot.AllowChat(100)
	bot.RegisterDefaultCommands()

	update := Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 1,
			Chat:      Chat{ID: 100, FirstName: "Alice"},
			Text:      "/start",
		},
	}

	err := bot.HandleUpdate(update)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sentReq.ChatID != 100 {
		t.Errorf("expected chat_id 100, got %d", sentReq.ChatID)
	}
	if sentReq.Text == "" {
		t.Error("expected non-empty response text")
	}
	if !containsString(sentReq.Text, "Alice") {
		t.Errorf("expected response to contain 'Alice', got %q", sentReq.Text)
	}
}

func TestHandleUpdateHelpCommand(t *testing.T) {
	var sentReq SendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	bot.AllowChat(100)
	bot.RegisterDefaultCommands()

	err := bot.HandleUpdate(Update{
		UpdateID: 2,
		Message: &Message{
			MessageID: 2,
			Chat:      Chat{ID: 100},
			Text:      "/help",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsString(sentReq.Text, "/portfolio") {
		t.Errorf("expected help text to contain '/portfolio', got %q", sentReq.Text)
	}
}

func TestHandleUpdatePriceWithArgs(t *testing.T) {
	var sentReq SendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	bot.AllowChat(100)
	bot.RegisterDefaultCommands()

	err := bot.HandleUpdate(Update{
		UpdateID: 3,
		Message: &Message{
			MessageID: 3,
			Chat:      Chat{ID: 100},
			Text:      "/price BTC",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsString(sentReq.Text, "BTC") {
		t.Errorf("expected response to contain 'BTC', got %q", sentReq.Text)
	}
	if sentReq.ReplyMarkup == nil {
		t.Fatal("expected inline keyboard markup")
	}
	if len(sentReq.ReplyMarkup.InlineKeyboard) == 0 {
		t.Fatal("expected at least one keyboard row")
	}
	row := sentReq.ReplyMarkup.InlineKeyboard[0]
	if len(row) != 3 {
		t.Fatalf("expected 3 buttons, got %d", len(row))
	}
	if row[0].CallbackData != "buy_BTC" {
		t.Errorf("expected callback data 'buy_BTC', got %q", row[0].CallbackData)
	}
}

func TestHandleUpdateIgnoresUnallowedChat(t *testing.T) {
	called := false
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	// Do NOT allow chat 999
	bot.RegisterDefaultCommands()

	err := bot.HandleUpdate(Update{
		UpdateID: 4,
		Message: &Message{
			MessageID: 4,
			Chat:      Chat{ID: 999},
			Text:      "/start",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("expected no message to be sent for unallowed chat")
	}
}

func TestHandleUpdateUnknownCommand(t *testing.T) {
	var sentReq SendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	bot.AllowChat(100)

	err := bot.HandleUpdate(Update{
		UpdateID: 5,
		Message: &Message{
			MessageID: 5,
			Chat:      Chat{ID: 100},
			Text:      "/nonexistent",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !containsString(sentReq.Text, "Unknown command") {
		t.Errorf("expected unknown command response, got %q", sentReq.Text)
	}
}

func TestHandleUpdateCallbackQuery(t *testing.T) {
	var sentReqs []SendMessageRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req SendMessageRequest
		json.NewDecoder(r.Body).Decode(&req)
		if req.ChatID != 0 {
			sentReqs = append(sentReqs, req)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(ts.URL)
	bot.AllowChat(100)

	err := bot.HandleUpdate(Update{
		UpdateID: 6,
		CallbackQuery: &CallbackQuery{
			ID:   "cb-1",
			Data: "portfolio_refresh",
			Message: &Message{
				MessageID: 10,
				Chat:      Chat{ID: 100},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sentReqs) == 0 {
		t.Fatal("expected a message to be sent for callback query")
	}
	if !containsString(sentReqs[0].Text, "portfolio_refresh") {
		t.Errorf("expected callback action in response, got %q", sentReqs[0].Text)
	}
}

func TestRegisterDefaultCommandsCount(t *testing.T) {
	bot := NewTelegramBot("token")
	bot.RegisterDefaultCommands()

	cmds := bot.ListCommands()
	if len(cmds) != 6 {
		t.Errorf("expected 6 default commands, got %d: %v", len(cmds), cmds)
	}
}

func TestHandleWebhookReturnsHandler(t *testing.T) {
	bot := NewTelegramBot("token")
	handler := bot.HandleWebhook()
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandleWebhookProcessesUpdate(t *testing.T) {
	var sentReq SendMessageRequest
	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&sentReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer apiServer.Close()

	bot := NewTelegramBot("token")
	bot.SetBaseURL(apiServer.URL)
	bot.AllowChat(100)
	bot.RegisterDefaultCommands()

	handler := bot.HandleWebhook()

	update := Update{
		UpdateID: 1,
		Message: &Message{
			MessageID: 1,
			Chat:      Chat{ID: 100, FirstName: "Bob"},
			Text:      "/start",
		},
	}
	body, _ := json.Marshal(update)
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if sentReq.ChatID != 100 {
		t.Errorf("expected chat_id 100, got %d", sentReq.ChatID)
	}
}

func TestHandleWebhookRejectsGet(t *testing.T) {
	bot := NewTelegramBot("token")
	handler := bot.HandleWebhook()

	req := httptest.NewRequest(http.MethodGet, "/webhook", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405 for GET, got %d", w.Code)
	}
}

func TestSendMessageBuildsCorrectRequest(t *testing.T) {
	var receivedReq SendMessageRequest
	var receivedPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		json.NewDecoder(r.Body).Decode(&receivedReq)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	bot := NewTelegramBot("my-bot-token")
	bot.SetBaseURL(ts.URL)

	kb := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "OK", CallbackData: "ok"}},
		},
	}

	err := bot.SendMessage(SendMessageRequest{
		ChatID:      42,
		Text:        "Hello!",
		ParseMode:   "HTML",
		ReplyMarkup: kb,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedPath != "/botmy-bot-token/sendMessage" {
		t.Errorf("expected path '/botmy-bot-token/sendMessage', got %q", receivedPath)
	}
	if receivedReq.ChatID != 42 {
		t.Errorf("expected chat_id 42, got %d", receivedReq.ChatID)
	}
	if receivedReq.Text != "Hello!" {
		t.Errorf("expected text 'Hello!', got %q", receivedReq.Text)
	}
	if receivedReq.ParseMode != "HTML" {
		t.Errorf("expected parse_mode 'HTML', got %q", receivedReq.ParseMode)
	}
	if receivedReq.ReplyMarkup == nil {
		t.Fatal("expected reply_markup to be set")
	}
}

func TestConcurrentAccess(t *testing.T) {
	bot := NewTelegramBot("token")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()
	bot.SetBaseURL(ts.URL)

	var wg sync.WaitGroup

	// Concurrent AllowChat
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			bot.AllowChat(id)
		}(int64(i))
	}

	// Concurrent RegisterCommand
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			name := "cmd" + string(rune('a'+idx%26))
			bot.RegisterCommand(name, func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
				return "ok", nil
			})
		}(i)
	}

	// Concurrent IsAllowed
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			bot.IsAllowed(id)
		}(int64(i))
	}

	// Concurrent ListCommands
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			bot.ListCommands()
		}()
	}

	// Concurrent HandleUpdate
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(id int64) {
			defer wg.Done()
			bot.AllowChat(id)
			bot.RegisterCommand("test", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
				return "ok", nil
			})
			bot.HandleUpdate(Update{
				UpdateID: id,
				Message: &Message{
					MessageID: id,
					Chat:      Chat{ID: id},
					Text:      "/test",
				},
			})
		}(int64(i + 1000))
	}

	wg.Wait()
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
