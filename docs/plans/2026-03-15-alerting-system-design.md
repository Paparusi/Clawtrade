# Alerting System Design

**Goal:** Event-driven alerting system that evaluates price, PnL, risk, trade, system, and custom rule alerts, dispatching notifications via Telegram and WebSocket with SQLite persistence.

**Architecture:** AlertManager subscribes to EventBus, evaluates active alerts on each event, dispatches via Telegram + WebSocket. Custom rules use the same expression syntax as backtest ConfigStrategy. Daily briefing via scheduled goroutine.

**Tech Stack:** Go (backend), existing EventBus + TelegramBot + WebSocket hub, SQLite persistence, expression evaluator from backtest package.

---

## Components

### 1. Alert Model

SQLite table `alerts`:

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| type | TEXT | price, pnl, risk, trade, system, custom |
| symbol | TEXT | Trading pair (nullable for system/risk alerts) |
| condition | TEXT | above, below, cross, expression |
| threshold | REAL | Target value (for price/pnl alerts) |
| expression | TEXT | Custom rule expression (for custom type) |
| message | TEXT | User-defined alert message |
| enabled | BOOLEAN | Active flag |
| one_shot | BOOLEAN | Auto-disable after first trigger |
| last_triggered_at | DATETIME | Rate limiting |
| created_at | DATETIME | Creation timestamp |

SQLite table `alert_history`:

| Column | Type | Description |
|--------|------|-------------|
| id | INTEGER PK | Auto-increment |
| alert_id | INTEGER | FK to alerts |
| event_type | TEXT | Event that triggered it |
| value | REAL | Value at trigger time |
| message | TEXT | Formatted message sent |
| created_at | DATETIME | Trigger timestamp |

### 2. AlertManager

- Loads all enabled alerts from DB on startup
- Subscribes to EventBus patterns: `price.*`, `trade.*`, `risk.*`, `portfolio.*`, `system.*`
- Evaluation logic per event type:
  - `price.update` → evaluate price alerts (above/below/cross) + custom rule alerts for matching symbol
  - `trade.executed` → send trade notification (if `alerts.trade_executed` config enabled)
  - `risk.*` → send risk alert (drawdown, circuit breaker)
  - `portfolio.update` → evaluate PnL alerts (total PnL crossing threshold)
  - `system.*` → send system alert (adapter connect/disconnect)
- Rate limiting: max 1 notification per alert per N minutes (configurable, default 5)
- Thread-safe with RWMutex for alert list access
- Methods: `AddAlert()`, `RemoveAlert()`, `ListAlerts()`, `Evaluate(event)`

### 3. Dispatcher

- Formats alert into human-readable message with context
- Sends via `TelegramBot.SendMessage()` (if Telegram enabled in config)
- Publishes `alert.triggered` event to EventBus → WebSocket clients receive it
- Logs to `alert_history` table

### 4. Daily Briefing

- Goroutine with timer firing at configurable hour (default 08:00 UTC)
- Collects:
  - Portfolio snapshot (balances/positions from all adapters)
  - Alerts triggered today (from `alert_history`)
  - Active alert count
- Formats as Telegram message and sends

### 5. Agent Tools

Three new tools registered in ToolRegistry:

**`create_alert`** — Create a new alert
- Parameters: type, symbol, condition, threshold, expression, message, one_shot
- Returns: alert ID + confirmation

**`list_alerts`** — List all active alerts
- Parameters: none (optional type filter)
- Returns: formatted table of alerts

**`delete_alert`** — Delete an alert by ID
- Parameters: id
- Returns: confirmation

### 6. Custom Rule Expression

Reuse expression evaluator from `internal/backtest/strategy.go`. Custom alerts evaluate expressions like `"rsi < 30 AND close > sma_50"` against live indicator values computed from recent candles.

When a `price.update` event arrives for a symbol with custom alerts:
1. Fetch recent candles from cache (candle_cache table)
2. Compute indicators (RSI, SMA, EMA, MACD, BB)
3. Evaluate expression
4. If true → trigger alert

## Data Flow

```
User: "alert me when ETH drops below 3000"
  → Agent parses → calls create_alert tool
  → AlertManager.AddAlert() → saves to DB + in-memory list

PriceStreamer polls ETH → EventBus "price.update" {symbol, price}
  → AlertManager.Evaluate()
  → Match: ETH price 2950 < 3000? YES
  → Rate limit: last triggered > 5min ago? YES
  → Dispatcher → Telegram + "alert.triggered" event → WebSocket
  → Log to alert_history
  → one_shot? → disable alert
```

## Config Additions

```yaml
notifications:
  alerts:
    trade_executed: true
    risk_alert: true
    pnl_update: false
    system_alert: true
    rate_limit_minutes: 5
    daily_briefing: true
    briefing_hour_utc: 8
```

## Integration Points

- **EventBus**: AlertManager subscribes to events (no changes to EventBus needed)
- **TelegramBot**: Use existing `SendMessage()` method
- **WebSocket**: New `alert.*` event pattern added to hub subscriptions
- **Database**: New migration for `alerts` + `alert_history` tables
- **Agent**: 3 new tools in ToolRegistry
- **Config**: Extended AlertsConfig struct
- **main.go serve()**: Create AlertManager, pass to server, start briefing goroutine
