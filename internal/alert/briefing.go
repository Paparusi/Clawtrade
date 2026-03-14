package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/bot"
)

// BriefingConfig holds daily briefing configuration.
type BriefingConfig struct {
	HourUTC  int
	TgBot    *bot.TelegramBot
	ChatID   int64
	Adapters map[string]adapter.TradingAdapter
	Store    *Store
}

// StartDailyBriefing launches a goroutine that sends a daily briefing at the configured hour.
func StartDailyBriefing(ctx context.Context, cfg BriefingConfig) {
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day(), cfg.HourUTC, 0, 0, 0, time.UTC)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			wait := next.Sub(now)

			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				sendBriefing(ctx, cfg)
			}
		}
	}()
}

func sendBriefing(ctx context.Context, cfg BriefingConfig) {
	totalValue := 0.0

	for _, adp := range cfg.Adapters {
		if !adp.IsConnected() {
			continue
		}
		balances, err := adp.GetBalances(ctx)
		if err != nil {
			continue
		}
		for _, b := range balances {
			totalValue += b.Total
		}
	}

	activeAlerts := 0
	if alerts, err := cfg.Store.ListEnabled(); err == nil {
		activeAlerts = len(alerts)
	}

	triggeredToday := 0
	if history, err := cfg.Store.TodayHistory(); err == nil {
		triggeredToday = len(history)
	}

	msg := FormatBriefing(totalValue, activeAlerts, triggeredToday)

	if cfg.TgBot != nil && cfg.ChatID != 0 {
		cfg.TgBot.SendMessage(bot.SendMessageRequest{
			ChatID: cfg.ChatID,
			Text:   msg,
		})
	}
}

// FormatBriefing creates the daily briefing message text.
func FormatBriefing(portfolioValue float64, activeAlerts, triggeredToday int) string {
	date := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"\xf0\x9f\x93\xb0 Daily Briefing \xe2\x80\x94 %s\n\n"+
			"\xf0\x9f\x92\xbc Portfolio: $%s\n"+
			"\xf0\x9f\x94\x94 Active Alerts: %d\n"+
			"\xe2\x9a\xa1 Triggered Today: %d",
		date,
		formatMoney(portfolioValue),
		activeAlerts,
		triggeredToday,
	)
}

func formatMoney(v float64) string {
	if v >= 1000000 {
		return fmt.Sprintf("%.2fM", v/1000000)
	}
	if v >= 1000 {
		whole := int(v)
		frac := int((v - float64(whole)) * 100)
		s := fmt.Sprintf("%d", whole)
		if len(s) > 3 {
			s = s[:len(s)-3] + "," + s[len(s)-3:]
		}
		return fmt.Sprintf("%s.%02d", s, frac)
	}
	return fmt.Sprintf("%.2f", v)
}
