package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// Telegram Bot API types

// Update represents an incoming update from Telegram.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
	Date      int64  `json:"date"`
}

// Chat represents a Telegram chat.
type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	FirstName string `json:"first_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// CallbackQuery represents a callback query from an inline keyboard button.
type CallbackQuery struct {
	ID      string   `json:"id"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

// SendMessageRequest represents a request to send a message.
type SendMessageRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// InlineKeyboardButton represents a button in an inline keyboard.
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
}

// CommandHandler handles a bot command.
type CommandHandler func(chat Chat, args string) (string, *InlineKeyboardMarkup)

// TelegramBot manages the Telegram bot.
type TelegramBot struct {
	mu           sync.RWMutex
	token        string
	commands     map[string]CommandHandler
	allowedChats map[int64]bool
	baseURL      string
	httpClient   *http.Client
}

// NewTelegramBot creates a new bot with the given token.
func NewTelegramBot(token string) *TelegramBot {
	return &TelegramBot{
		token:        token,
		commands:     make(map[string]CommandHandler),
		allowedChats: make(map[int64]bool),
		baseURL:      "https://api.telegram.org",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SetBaseURL allows overriding the API URL for testing.
func (b *TelegramBot) SetBaseURL(url string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.baseURL = url
}

// AllowChat adds a chat ID to the whitelist.
func (b *TelegramBot) AllowChat(chatID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.allowedChats[chatID] = true
}

// IsAllowed checks if a chat is whitelisted.
func (b *TelegramBot) IsAllowed(chatID int64) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.allowedChats[chatID]
}

// RegisterCommand registers a command handler.
func (b *TelegramBot) RegisterCommand(command string, handler CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.commands[command] = handler
}

// HandleUpdate processes a single update.
func (b *TelegramBot) HandleUpdate(update Update) error {
	// Handle callback queries for inline buttons
	if update.CallbackQuery != nil {
		return b.handleCallback(update.CallbackQuery)
	}

	if update.Message == nil {
		return nil
	}

	chatID := update.Message.Chat.ID

	// Check if chat is allowed
	if !b.IsAllowed(chatID) {
		return nil
	}

	text := strings.TrimSpace(update.Message.Text)
	if text == "" || text[0] != '/' {
		return nil
	}

	// Parse command and args
	text = text[1:] // strip leading /
	// Strip @botname suffix if present (e.g. /start@MyBot)
	if atIdx := strings.Index(text, "@"); atIdx != -1 {
		// Only strip if before any space
		spaceIdx := strings.Index(text, " ")
		if spaceIdx == -1 || atIdx < spaceIdx {
			text = text[:atIdx] + text[spaceIdx:]
			if spaceIdx == -1 {
				text = text[:atIdx]
			}
		}
	}

	parts := strings.SplitN(text, " ", 2)
	command := strings.ToLower(parts[0])
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	b.mu.RLock()
	handler, ok := b.commands[command]
	b.mu.RUnlock()

	if !ok {
		return b.SendMessage(SendMessageRequest{
			ChatID: chatID,
			Text:   fmt.Sprintf("Unknown command: /%s\nType /help for available commands.", command),
		})
	}

	responseText, keyboard := handler(update.Message.Chat, args)
	return b.SendMessage(SendMessageRequest{
		ChatID:      chatID,
		Text:        responseText,
		ReplyMarkup: keyboard,
	})
}

// handleCallback processes a callback query from an inline button press.
func (b *TelegramBot) handleCallback(cq *CallbackQuery) error {
	if cq.Message == nil {
		return nil
	}

	chatID := cq.Message.Chat.ID

	if !b.IsAllowed(chatID) {
		return nil
	}

	// Answer the callback query to dismiss the loading indicator
	_ = b.answerCallbackQuery(cq.ID)

	return b.SendMessage(SendMessageRequest{
		ChatID: chatID,
		Text:   fmt.Sprintf("Action: %s", cq.Data),
	})
}

// answerCallbackQuery acknowledges a callback query.
func (b *TelegramBot) answerCallbackQuery(queryID string) error {
	b.mu.RLock()
	baseURL := b.baseURL
	token := b.token
	b.mu.RUnlock()

	url := fmt.Sprintf("%s/bot%s/answerCallbackQuery", baseURL, token)

	payload := map[string]string{"callback_query_id": queryID}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal answer callback: %w", err)
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}
	defer resp.Body.Close()
	return nil
}

// SendMessage sends a message to a chat.
func (b *TelegramBot) SendMessage(req SendMessageRequest) error {
	b.mu.RLock()
	baseURL := b.baseURL
	token := b.token
	b.mu.RUnlock()

	url := fmt.Sprintf("%s/bot%s/sendMessage", baseURL, token)

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal send message: %w", err)
	}

	resp, err := b.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// HandleWebhook returns an http.HandlerFunc for webhook mode.
func (b *TelegramBot) HandleWebhook() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var update Update
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		if err := b.HandleUpdate(update); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
	}
}

// RegisterDefaultCommands registers built-in trading commands.
func (b *TelegramBot) RegisterDefaultCommands() {
	b.RegisterCommand("start", func(chat Chat, _ string) (string, *InlineKeyboardMarkup) {
		return fmt.Sprintf("Welcome to Clawtrade, %s! \xf0\x9f\xa4\x96\nUse /help for available commands.", chat.FirstName), nil
	})

	b.RegisterCommand("help", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
		return "Available commands:\n/portfolio - View portfolio\n/positions - View positions\n/price <symbol> - Get price\n/analyze <symbol> - AI analysis\n/risk - Risk status\n/briefing - Daily briefing", nil
	})

	b.RegisterCommand("portfolio", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
		kb := &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{{Text: "Refresh", CallbackData: "portfolio_refresh"}, {Text: "Details", CallbackData: "portfolio_details"}},
			},
		}
		return "\xf0\x9f\x93\x8a Portfolio Summary\nBalance: $10,000.00\nUnrealized PnL: +$0.00\nToday PnL: $0.00", kb
	})

	b.RegisterCommand("positions", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
		return "\xf0\x9f\x93\x8b Open Positions\nNo open positions.", nil
	})

	b.RegisterCommand("price", func(_ Chat, args string) (string, *InlineKeyboardMarkup) {
		if args == "" {
			return "Usage: /price <symbol>", nil
		}
		kb := &InlineKeyboardMarkup{
			InlineKeyboard: [][]InlineKeyboardButton{
				{{Text: "Buy", CallbackData: "buy_" + args}, {Text: "Sell", CallbackData: "sell_" + args}, {Text: "Analyze", CallbackData: "analyze_" + args}},
			},
		}
		return fmt.Sprintf("\xf0\x9f\x92\xb0 %s\nPrice data coming soon...", args), kb
	})

	b.RegisterCommand("briefing", func(_ Chat, _ string) (string, *InlineKeyboardMarkup) {
		return fmt.Sprintf("\xf0\x9f\x93\xb0 Daily Briefing - %s\n\nMarket Overview: Coming soon...\nYour Performance: Coming soon...\nAI Insights: Coming soon...", time.Now().Format("2006-01-02")), nil
	})
}

// ListCommands returns registered command names in sorted order.
func (b *TelegramBot) ListCommands() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	cmds := make([]string, 0, len(b.commands))
	for cmd := range b.commands {
		cmds = append(cmds, cmd)
	}
	sort.Strings(cmds)
	return cmds
}
