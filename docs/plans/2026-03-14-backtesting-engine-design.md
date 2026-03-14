# Backtesting Engine Design

**Goal:** Add a backtesting engine that replays historical candles through strategies, simulates portfolio state, and outputs performance metrics — accessible from CLI, AI agent tools, and WebSocket-streaming dashboard.

**Architecture:** Monolithic `internal/backtest/` package with 4 core files (engine, data, strategy, portfolio). Reuses existing adapter infrastructure for historical data, analysis/indicators for signals, strategy/arena for registered strategies, and engine/eventbus for real-time streaming.

**Tech Stack:** Go, SQLite (candle cache), existing adapters (Binance/Bybit), existing indicators (SMA/EMA/RSI/MACD/BB/Ichimoku).

---

## Package Structure

```
internal/backtest/
├── engine.go        # Core time-loop, orchestrates data→strategy→portfolio→metrics
├── data.go          # Historical data loader with hybrid cache (exchange API + SQLite)
├── strategy.go      # Strategy executor (code/config/AI modes)
├── portfolio.go     # Portfolio simulator (balance, positions, fees, slippage)
└── engine_test.go   # Tests
```

## Data Flow

```
1. User/Agent triggers backtest (CLI command or agent tool call)
2. DataLoader fetches candles (SQLite cache first, fallback to exchange API)
3. Engine loops through candles chronologically:
   a. Feed candle + indicator window to strategy → Signal (buy/sell/hold)
   b. Execute signal on PortfolioSimulator → update positions/balance
   c. Check stop-loss / take-profit on open positions
   d. Record equity point + trade details
   e. Emit backtest.progress event via EventBus (for WebSocket streaming)
4. After loop: calculate metrics, output results (CLI table + JSON + WS event)
```

## Components

### DataLoader (`data.go`)

- SQLite cache table: `candles(symbol, timeframe, timestamp, open, high, low, close, volume)`
- `LoadCandles(symbol, timeframe, from, to)` — check cache first, fetch missing ranges from exchange via adapter `GetCandles()`, auto-cache fetched data
- Auto-pagination: exchange APIs limit ~200 candles/request, loader paginates transparently for larger date ranges
- Uses existing `adapter.Manager` to get connected exchange adapter

### Engine (`engine.go`)

- `BacktestConfig` struct:
  - Symbol, Timeframe, From/To dates
  - InitialCapital (default $10,000)
  - MakerFee, TakerFee (default 0.1%)
  - Slippage (default 0.05%)
  - StrategyMode: "code" | "config" | "ai"
  - StrategyName (for code mode) or StrategyConfig (for config/ai mode)
- `Engine.Run(ctx, config) → BacktestResult`
- Time loop iterates candles, maintains rolling window (last N candles for indicator calculation)
- Emits `backtest.progress` events via EventBus: `{candle_index, total_candles, equity, current_price, open_positions}`
- Emits `backtest.complete` with full `BacktestResult`

### StrategyExecutor (`strategy.go`)

Three modes via `StrategyRunner` interface:

1. **CodeStrategy** — wraps existing `strategy.Arena.RunSignal()`, reuses registered Go strategies
2. **ConfigStrategy** — parses simple YAML/JSON rule expressions:
   - Format: `{buy_when: "rsi < 30 AND close > sma_50", sell_when: "rsi > 70"}`
   - Simple expression evaluator over pre-computed indicator values
3. **AIStrategy** — sends candle data + indicators to AI agent, agent returns buy/sell/hold signal
   - Reuses existing `analyze_market` tool flow
   - Slower (LLM call per candle), useful for testing AI judgment

Signal output: `{Action: Buy|Sell|Hold, Size: float64, StopLoss: float64, TakeProfit: float64}`

### PortfolioSimulator (`portfolio.go`)

- Tracks: cash balance, open positions (with entry price/time), closed trade history
- Applies maker/taker fees and slippage on each trade execution
- Position sizing modes: fixed dollar amount, fixed % of equity, or Kelly criterion
- Auto-executes stop-loss / take-profit checks on each candle tick
- Records equity curve point after each candle

### Results & Metrics (`BacktestResult`)

Reuse existing arena.go metric calculations where possible:

- **Core**: Total PnL, Win Rate, Total Trades, Profit Factor
- **Risk**: Max Drawdown, Sharpe Ratio, Sortino Ratio, Calmar Ratio
- **Detail**: Equity curve ([]float64 with timestamps), trade log (entry/exit/pnl per trade), avg holding period
- **Output formats**: CLI pretty-printed table, JSON file export, WebSocket event stream

## Integration Points

### CLI Command
```
clawtrade backtest --symbol BTC/USDT --timeframe 1d --from 2025-01-01 --to 2025-12-31 --strategy momentum --capital 10000
clawtrade backtest --symbol ETH/USDT --timeframe 4h --config rules.yaml
```
New `backtest` subcommand in `cmd/clawtrade/main.go`.

### Agent Tool
New `backtest` tool in `internal/agent/tools.go`:
- Input: symbol, timeframe, period, strategy description
- Output: summary metrics + trade count + equity change
- Agent can backtest before deciding to trade live

### API Endpoint
`POST /api/v1/backtest` — triggers backtest, returns result as JSON. For long backtests, returns job ID and streams progress via WebSocket.

### WebSocket Events
- `backtest.progress` — emitted per candle (throttled to ~10/sec for performance): `{index, total, equity, price}`
- `backtest.complete` — full results when done: `{metrics, trades, equity_curve}`

### Optimizer Integration
Existing `strategy/optimizer.go` GridSearch/RandomSearch/HillClimb can use `Engine.Run()` as the evaluation function — test parameter combinations automatically.

## What We Reuse

| Existing Code | How It's Reused |
|---|---|
| `adapter.Manager` + Binance/Bybit | Fetch historical candles |
| `adapter.Candle` type | Data structure throughout |
| `analysis.indicators` | SMA, EMA, RSI, MACD, BB, Ichimoku |
| `strategy.Arena` | CodeStrategy mode runs registered strategies |
| `strategy.Optimizer` | Parameter optimization over backtest results |
| `engine.EventBus` | Stream progress/results to WebSocket |
| `api.WebSocket` hub | Deliver events to frontend |
| Simulation adapter patterns | Portfolio state management inspiration |
