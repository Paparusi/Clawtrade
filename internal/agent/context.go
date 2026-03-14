// internal/agent/context.go
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/config"
	"github.com/clawtrade/clawtrade/internal/memory"
	"github.com/clawtrade/clawtrade/internal/risk"
)

// ContextBuilder gathers real-time data and builds the system prompt.
type ContextBuilder struct {
	cfg              *config.Config
	adapters         map[string]adapter.TradingAdapter
	riskEngine       *risk.Engine
	memory           *memory.Store
	subAgentInsights map[string]string // eventType -> latest insight text
	insightsMu       sync.RWMutex
}

// NewContextBuilder creates a context builder with access to all data sources.
func NewContextBuilder(cfg *config.Config, adapters map[string]adapter.TradingAdapter, riskEngine *risk.Engine, mem *memory.Store) *ContextBuilder {
	return &ContextBuilder{
		cfg:              cfg,
		adapters:         adapters,
		riskEngine:       riskEngine,
		memory:           mem,
		subAgentInsights: make(map[string]string),
	}
}

// SetSubAgentInsight stores or updates a sub-agent insight keyed by event type.
// These insights are included in the system prompt built by BuildSystemPrompt.
func (cb *ContextBuilder) SetSubAgentInsight(eventType, insight string) {
	cb.insightsMu.Lock()
	defer cb.insightsMu.Unlock()
	cb.subAgentInsights[eventType] = insight
}

// BuildSystemPrompt creates a rich system prompt with real-time trading context.
func (cb *ContextBuilder) BuildSystemPrompt(ctx context.Context) string {
	var b strings.Builder

	b.WriteString(`You are Clawtrade AI, an expert AI trading agent. You have REAL access to exchange data and can execute trades.

## Your Capabilities
- Fetch real-time prices, candles, order books from connected exchanges
- Run technical analysis (RSI, SMA, EMA, MACD, Bollinger Bands)
- Check risk limits before placing trades
- Calculate optimal position sizes
- Place, cancel, and monitor orders
- View account balances and open positions

## Rules
1. ALWAYS use get_price or analyze_market to check current data before giving trading advice
2. ALWAYS run risk_check before placing any order
3. Be specific with numbers — don't guess prices, fetch them
4. When analyzing a market, use analyze_market for technical indicators
5. Warn the user about risks. Never guarantee profits
6. If the user asks "how is BTC doing", fetch the price and run analysis — don't respond with generic text
7. For order placement: confirm with the user before executing unless they explicitly say "execute" or "place"
8. Present data in a clear, structured format

`)

	// Add connected exchanges
	b.WriteString("## Connected Exchanges\n")
	if len(cb.adapters) == 0 {
		b.WriteString("No exchanges configured. User needs to run: clawtrade exchange add binance\n")
	} else {
		for name, adp := range cb.adapters {
			status := "disconnected"
			if adp.IsConnected() {
				status = "connected"
			}
			caps := adp.Capabilities()
			b.WriteString(fmt.Sprintf("- **%s** (%s) — websocket:%v, futures:%v, margin:%v\n",
				name, status, caps.WebSocket, caps.Futures, caps.Margin))
		}
	}
	b.WriteString("\n")

	// Add risk settings
	if cb.riskEngine != nil {
		limits := cb.riskEngine.GetLimits()
		b.WriteString(fmt.Sprintf(`## Risk Settings
- Max position size: %.0f%% of balance
- Max risk per trade: %.1f%%
- Max daily loss: %.0f%%
- Max open positions: %d
- Max total exposure: %.0f%%
- Max order size: $%.0f

`, limits.MaxPositionSizePct*100, limits.MaxRiskPerTradePct*100,
			limits.MaxDailyLossPct*100, limits.MaxOpenPositions,
			limits.MaxTotalExposurePct*100, limits.MaxOrderSize))
	}

	// Add live portfolio snapshot (quick, non-blocking)
	cb.addPortfolioSnapshot(ctx, &b)

	// Add watchlist
	if len(cb.cfg.Agent.Watchlist) > 0 {
		b.WriteString("## Watchlist\n")
		b.WriteString(strings.Join(cb.cfg.Agent.Watchlist, ", "))
		b.WriteString("\n\n")
	}

	// Add learned rules from memory
	cb.addMemoryContext(&b)

	// Add sub-agent insights
	cb.insightsMu.RLock()
	if len(cb.subAgentInsights) > 0 {
		b.WriteString("## AI Sub-Agent Insights\n")
		for eventType, insight := range cb.subAgentInsights {
			b.WriteString(fmt.Sprintf("### %s\n%s\n\n", eventType, insight))
		}
	}
	cb.insightsMu.RUnlock()

	b.WriteString(fmt.Sprintf("## Current Time\n%s UTC\n", time.Now().UTC().Format("2006-01-02 15:04:05")))

	return b.String()
}

func (cb *ContextBuilder) addPortfolioSnapshot(ctx context.Context, b *strings.Builder) {
	for name, adp := range cb.adapters {
		if !adp.IsConnected() {
			continue
		}

		// Balances (with short timeout)
		subCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		balances, err := adp.GetBalances(subCtx)
		cancel()
		if err == nil && len(balances) > 0 {
			b.WriteString(fmt.Sprintf("## %s Balances\n", strings.Title(name)))
			totalUSD := 0.0
			for _, bal := range balances {
				if bal.Total > 0.01 {
					b.WriteString(fmt.Sprintf("- %s: %.4f (free: %.4f, locked: %.4f)\n",
						bal.Asset, bal.Total, bal.Free, bal.Locked))
					if bal.Asset == "USDT" || bal.Asset == "BUSD" || bal.Asset == "USDC" {
						totalUSD += bal.Total
					}
				}
			}
			if totalUSD > 0 {
				b.WriteString(fmt.Sprintf("- **Stablecoin total: $%.2f**\n", totalUSD))
			}
			b.WriteString("\n")
		}

		// Positions
		subCtx2, cancel2 := context.WithTimeout(ctx, 3*time.Second)
		positions, err := adp.GetPositions(subCtx2)
		cancel2()
		if err == nil && len(positions) > 0 {
			b.WriteString(fmt.Sprintf("## %s Open Positions\n", strings.Title(name)))
			totalPnL := 0.0
			for _, pos := range positions {
				b.WriteString(fmt.Sprintf("- %s %s %.4f @ $%.2f (now $%.2f, PnL: $%.2f)\n",
					pos.Symbol, pos.Side, pos.Size, pos.EntryPrice, pos.CurrentPrice, pos.PnL))
				totalPnL += pos.PnL
			}
			b.WriteString(fmt.Sprintf("- **Total unrealized PnL: $%.2f**\n\n", totalPnL))
		}
		break // Only first connected exchange for now
	}
}

func (cb *ContextBuilder) addMemoryContext(b *strings.Builder) {
	if cb.memory == nil {
		return
	}

	// Recent trade history
	episodes, err := cb.memory.QueryEpisodes("", 5)
	if err == nil && len(episodes) > 0 {
		b.WriteString("## Recent Trade History\n")
		for _, ep := range episodes {
			outcome := "win"
			if ep.PnL < 0 {
				outcome = "loss"
			}
			b.WriteString(fmt.Sprintf("- %s %s %s: PnL $%.2f (%s) — %s\n",
				ep.OpenedAt.Format("Jan 02"), ep.Symbol, ep.Side, ep.PnL, outcome, ep.Reasoning))
		}
		b.WriteString("\n")
	}

	// Learned rules
	rules, err := cb.memory.QueryRules("", 5)
	if err == nil && len(rules) > 0 {
		b.WriteString("## Learned Trading Rules\n")
		for _, r := range rules {
			b.WriteString(fmt.Sprintf("- [%.0f%% confidence] %s\n", r.Confidence*100, r.Content))
		}
		b.WriteString("\n")
	}
}
