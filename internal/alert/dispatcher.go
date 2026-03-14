package alert

import (
	"fmt"
	"strconv"

	"github.com/clawtrade/clawtrade/internal/bot"
	"github.com/clawtrade/clawtrade/internal/engine"
)

// Dispatcher sends alert notifications via Telegram and EventBus.
type Dispatcher struct {
	tgBot  *bot.TelegramBot
	chatID int64
	bus    *engine.EventBus
}

// NewDispatcher creates a new alert dispatcher.
func NewDispatcher(tgBot *bot.TelegramBot, chatID int64, bus *engine.EventBus) *Dispatcher {
	return &Dispatcher{
		tgBot:  tgBot,
		chatID: chatID,
		bus:    bus,
	}
}

// Dispatch sends an alert notification via all configured channels.
func (d *Dispatcher) Dispatch(a Alert, message string) {
	formatted := d.formatMessage(a, message)

	// Send via Telegram
	if d.tgBot != nil && d.chatID != 0 {
		go func() {
			d.tgBot.SendMessage(bot.SendMessageRequest{
				ChatID: d.chatID,
				Text:   formatted,
			})
		}()
	}

	// Publish alert.triggered event to bus (for WebSocket clients)
	if d.bus != nil {
		d.bus.Publish(engine.Event{
			Type: "alert.triggered",
			Data: map[string]any{
				"alert_id": a.ID,
				"type":     a.Type,
				"symbol":   a.Symbol,
				"message":  message,
			},
		})
	}
}

func (d *Dispatcher) formatMessage(a Alert, message string) string {
	icon := alertIcon(a.Type)
	return fmt.Sprintf("%s %s\n%s", icon, alertTitle(a.Type), message)
}

func alertIcon(alertType string) string {
	switch alertType {
	case AlertTypePrice:
		return "\xf0\x9f\x93\x88"
	case AlertTypePnL:
		return "\xf0\x9f\x92\xb0"
	case AlertTypeRisk:
		return "\xe2\x9a\xa0\xef\xb8\x8f"
	case AlertTypeTrade:
		return "\xf0\x9f\x94\x84"
	case AlertTypeSystem:
		return "\xf0\x9f\x94\xa7"
	case AlertTypeCustom:
		return "\xf0\x9f\x94\x94"
	default:
		return "\xf0\x9f\x94\x94"
	}
}

func alertTitle(alertType string) string {
	switch alertType {
	case AlertTypePrice:
		return "Price Alert"
	case AlertTypePnL:
		return "PnL Alert"
	case AlertTypeRisk:
		return "Risk Alert"
	case AlertTypeTrade:
		return "Trade Alert"
	case AlertTypeSystem:
		return "System Alert"
	case AlertTypeCustom:
		return "Custom Alert"
	default:
		return "Alert"
	}
}

// ParseChatID converts a string chat ID to int64.
func ParseChatID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
