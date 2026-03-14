# WebSocket Real-Time Dashboard Design

**Goal:** Push live data (prices, sub-agent insights, trades, portfolio) to the dashboard via the existing WebSocket infrastructure.

**Approach:** Hybrid price streaming (exchange WebSocket + polling fallback), bridge sub-agent events to engine.EventBus, poll portfolio for changes.

---

## Data Flow

```
Exchange WS ──→ Adapter ──→ PriceStreamer ──→ engine.EventBus ──→ Hub ──→ Dashboard
                                                    ↑
SubAgent EventBus ──→ Bridge ─────────────────────────┘
                                                    ↑
Trade/Portfolio events ─────────────────────────────┘
```

## Event Types

| Type | Source | Data |
|------|--------|------|
| `price.update` | PriceStreamer | `{symbol, last, bid, ask, volume_24h, change_pct}` |
| `agent.analysis` | SubAgent bridge | `{source, symbol, summary, data}` |
| `agent.counter` | SubAgent bridge | `{source, symbol, summary, data}` |
| `agent.narrative` | SubAgent bridge | `{source, summary, data}` |
| `agent.reflection` | SubAgent bridge | `{source, summary, data}` |
| `agent.correlation` | SubAgent bridge | `{source, summary, data}` |
| `trade.executed` | Order handler | `{symbol, side, size, price, status}` |
| `portfolio.update` | Portfolio poller | `{balances, positions, total_pnl}` |

## Components

### 1. PriceStreamer

Background goroutine that streams live prices to engine.EventBus.

- **Hybrid approach**: use `adapter.SubscribePrices()` if adapter supports WebSocket, fallback to polling every 2 seconds via `adapter.GetPrice()`
- When new price received → publish `price.update` on engine.EventBus
- Track previous prices to calculate `change_pct` (percentage change since last update)
- Configurable symbols via watchlist

### 2. SubAgent Bridge

Connects the `subagent.EventBus` (channel-based) to `engine.EventBus` (callback-based).

- Subscribes to all sub-agent event types: analysis, counter_analysis, narrative, reflection, correlation
- Converts to engine.Event with `agent.*` prefix
- Adds `summary` field (short human-readable text) alongside `data` field (raw sub-agent output)
- Summary format: `"{source}: {symbol} {key_insight}"` e.g. `"Market Analyst: BTC/USDT bullish 78%"`

### 3. Portfolio Poller

Background goroutine polling portfolio state.

- Polls balances + positions from connected adapters every 30 seconds
- Compares with previous state (hash-based change detection)
- Only publishes `portfolio.update` when data actually changed
- Includes: balances array, positions array, total unrealized PnL

### 4. Trade Event Hook

Publishes events when trades are executed.

- Hook into existing `place_order` tool execution path
- On successful order → publish `trade.executed` on engine.EventBus
- Includes: symbol, side, size, price, order type, status, order ID

### 5. Hub Subscription Update

Expand the Hub's EventBus subscriptions to include new event types.

Current: `market.*`, `trade.*`, `risk.*`, `system.*`
Add: `price.*`, `agent.*`, `portfolio.*`

## Client Protocol

Already implemented — clients send subscribe messages:

```json
{"type": "subscribe", "data": "price.update"}
{"type": "subscribe", "data": "agent.analysis"}
```

Server pushes matching events:

```json
{"type": "price.update", "data": {"symbol": "BTC/USDT", "last": 64720.5, "bid": 64710, "ask": 64730, "volume_24h": 12345.67, "change_pct": 2.5}}
{"type": "agent.analysis", "data": {"source": "market-analyst", "symbol": "BTC/USDT", "summary": "Bullish 78% — SMC + PA confluence at 64.2k OB", "data": {...}}}
```

## File Structure

```
internal/
  streaming/
    price_streamer.go        # Hybrid price streaming (WS + polling)
    price_streamer_test.go
    portfolio_poller.go      # Portfolio change detection
    portfolio_poller_test.go
    bridge.go                # SubAgent EventBus → engine.EventBus bridge
    bridge_test.go
```

## Key Decisions

1. **Hybrid streaming** — exchange WebSocket when available, REST polling fallback. Best coverage across all adapters.
2. **Two EventBus systems** — subagent.EventBus (Go channels, internal) stays separate from engine.EventBus (callbacks, external-facing). Bridge connects them.
3. **Change detection for portfolio** — don't flood WebSocket with unchanged data every 30s.
4. **Summary + raw data** — frontend can show quick summary, expand for details.
