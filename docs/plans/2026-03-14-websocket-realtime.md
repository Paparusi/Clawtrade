# WebSocket Real-Time Dashboard Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Stream live prices, sub-agent insights, trades, and portfolio updates to the dashboard via the existing WebSocket Hub.

**Architecture:** Three new background goroutines (PriceStreamer, SubAgent Bridge, Portfolio Poller) publish events to `engine.EventBus`, which the existing Hub forwards to WebSocket clients. PriceStreamer uses exchange WebSocket when available, falls back to REST polling.

**Tech Stack:** Go, gorilla/websocket (existing), engine.EventBus (existing), adapter.TradingAdapter (existing)

---

### Task 1: SubAgent Bridge (subagent.EventBus → engine.EventBus)

**Files:**
- Create: `internal/streaming/bridge.go`
- Create: `internal/streaming/bridge_test.go`

**Step 1: Write the failing test**

```go
// internal/streaming/bridge_test.go
package streaming

import (
    "testing"
    "time"

    "github.com/clawtrade/clawtrade/internal/engine"
    "github.com/clawtrade/clawtrade/internal/subagent"
)

func TestBridge_ForwardsAnalysisEvent(t *testing.T) {
    saBus := subagent.NewEventBus()
    engBus := engine.NewEventBus()

    b := NewBridge(saBus, engBus)
    b.Start()
    defer b.Stop()

    received := make(chan engine.Event, 1)
    engBus.Subscribe("agent.analysis", func(e engine.Event) {
        received <- e
    })

    saBus.Publish(subagent.Event{
        Type:   "analysis",
        Source: "market-analyst",
        Symbol: "BTC/USDT",
        Data:   map[string]any{"synthesis": "bullish 78%"},
    })

    select {
    case ev := <-received:
        if ev.Type != "agent.analysis" {
            t.Errorf("expected type 'agent.analysis', got %q", ev.Type)
        }
        data := ev.Data
        if data["source"] != "market-analyst" {
            t.Errorf("expected source 'market-analyst', got %v", data["source"])
        }
        if data["symbol"] != "BTC/USDT" {
            t.Errorf("expected symbol 'BTC/USDT', got %v", data["symbol"])
        }
        if _, ok := data["summary"]; !ok {
            t.Error("expected summary field")
        }
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for bridged event")
    }
}

func TestBridge_MapsEventTypes(t *testing.T) {
    tests := []struct {
        saType  string
        engType string
    }{
        {"analysis", "agent.analysis"},
        {"counter_analysis", "agent.counter"},
        {"narrative", "agent.narrative"},
        {"reflection", "agent.reflection"},
        {"correlation", "agent.correlation"},
    }
    for _, tt := range tests {
        got := mapEventType(tt.saType)
        if got != tt.engType {
            t.Errorf("mapEventType(%q) = %q, want %q", tt.saType, got, tt.engType)
        }
    }
}

func TestBridge_GeneratesSummary(t *testing.T) {
    ev := subagent.Event{
        Source: "market-analyst",
        Symbol: "BTC/USDT",
        Data:   map[string]any{"synthesis": "bullish with 78% confidence"},
    }
    summary := generateSummary(ev)
    if summary == "" {
        t.Error("expected non-empty summary")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/streaming/ -run TestBridge -v`
Expected: FAIL (package doesn't exist)

**Step 3: Implement Bridge**

`internal/streaming/bridge.go`:

```go
package streaming

import (
    "context"
    "fmt"
    "strings"

    "github.com/clawtrade/clawtrade/internal/engine"
    "github.com/clawtrade/clawtrade/internal/subagent"
)

// Bridge connects subagent.EventBus to engine.EventBus, forwarding
// sub-agent events with "agent.*" prefix for WebSocket broadcast.
type Bridge struct {
    saBus  *subagent.EventBus
    engBus *engine.EventBus
    cancel context.CancelFunc
}

func NewBridge(saBus *subagent.EventBus, engBus *engine.EventBus) *Bridge {
    return &Bridge{saBus: saBus, engBus: engBus}
}

var bridgedTypes = []string{"analysis", "counter_analysis", "narrative", "reflection", "correlation"}

func (b *Bridge) Start() {
    ctx, cancel := context.WithCancel(context.Background())
    b.cancel = cancel

    for _, et := range bridgedTypes {
        ch := b.saBus.Subscribe(et)
        go b.forwardLoop(ctx, ch)
    }
}

func (b *Bridge) Stop() {
    if b.cancel != nil {
        b.cancel()
    }
}

func (b *Bridge) forwardLoop(ctx context.Context, ch <-chan subagent.Event) {
    for {
        select {
        case ev := <-ch:
            engEvent := engine.Event{
                Type: mapEventType(ev.Type),
                Data: map[string]any{
                    "source":  ev.Source,
                    "symbol":  ev.Symbol,
                    "summary": generateSummary(ev),
                    "data":    ev.Data,
                },
            }
            b.engBus.Publish(engEvent)
        case <-ctx.Done():
            return
        }
    }
}

func mapEventType(saType string) string {
    switch saType {
    case "analysis":
        return "agent.analysis"
    case "counter_analysis":
        return "agent.counter"
    case "narrative":
        return "agent.narrative"
    case "reflection":
        return "agent.reflection"
    case "correlation":
        return "agent.correlation"
    default:
        return "agent." + saType
    }
}

func generateSummary(ev subagent.Event) string {
    source := ev.Source
    symbol := ev.Symbol

    // Try to extract key insight from data
    if ev.Data != nil {
        if synthesis, ok := ev.Data["synthesis"].(string); ok {
            if symbol != "" {
                return fmt.Sprintf("%s: %s — %s", source, symbol, truncate(synthesis, 80))
            }
            return fmt.Sprintf("%s: %s", source, truncate(synthesis, 80))
        }
        if counter, ok := ev.Data["counter"].(string); ok {
            return fmt.Sprintf("%s: %s — %s", source, symbol, truncate(counter, 80))
        }
    }

    if symbol != "" {
        return fmt.Sprintf("%s: %s update", source, symbol)
    }
    return fmt.Sprintf("%s: update", source)
}

func truncate(s string, max int) string {
    s = strings.ReplaceAll(s, "\n", " ")
    if len(s) <= max {
        return s
    }
    return s[:max] + "..."
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/streaming/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/streaming/bridge.go internal/streaming/bridge_test.go
git commit -m "feat(streaming): add SubAgent-to-EventBus bridge"
```

---

### Task 2: PriceStreamer (hybrid exchange WS + polling)

**Files:**
- Create: `internal/streaming/price_streamer.go`
- Create: `internal/streaming/price_streamer_test.go`

**Step 1: Write the failing test**

```go
// internal/streaming/price_streamer_test.go
package streaming

import (
    "context"
    "testing"
    "time"

    "github.com/clawtrade/clawtrade/internal/adapter"
    "github.com/clawtrade/clawtrade/internal/engine"
)

type mockAdapter struct {
    name      string
    connected bool
    prices    map[string]*adapter.Price
}

func (m *mockAdapter) Name() string                    { return m.name }
func (m *mockAdapter) IsConnected() bool               { return m.connected }
func (m *mockAdapter) Capabilities() adapter.AdapterCaps { return adapter.AdapterCaps{} }
func (m *mockAdapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
    if p, ok := m.prices[symbol]; ok {
        return p, nil
    }
    return &adapter.Price{Symbol: symbol, Last: 64000}, nil
}
func (m *mockAdapter) GetCandles(ctx context.Context, s, tf string, l int) ([]adapter.Candle, error) { return nil, nil }
func (m *mockAdapter) GetOrderBook(ctx context.Context, s string, d int) (*adapter.OrderBook, error) { return nil, nil }
func (m *mockAdapter) PlaceOrder(ctx context.Context, o adapter.Order) (*adapter.Order, error)       { return nil, nil }
func (m *mockAdapter) CancelOrder(ctx context.Context, id string) error                              { return nil }
func (m *mockAdapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error)                    { return nil, nil }
func (m *mockAdapter) GetBalances(ctx context.Context) ([]adapter.Balance, error)                    { return nil, nil }
func (m *mockAdapter) GetPositions(ctx context.Context) ([]adapter.Position, error)                  { return nil, nil }
func (m *mockAdapter) Connect(ctx context.Context) error                                             { return nil }
func (m *mockAdapter) Disconnect() error                                                             { return nil }

func TestPriceStreamer_PublishesPriceUpdate(t *testing.T) {
    bus := engine.NewEventBus()
    adp := &mockAdapter{name: "test", connected: true}

    ps := NewPriceStreamer(PriceStreamerConfig{
        Adapters:     map[string]adapter.TradingAdapter{"test": adp},
        Bus:          bus,
        Symbols:      []string{"BTC/USDT"},
        PollInterval: 100 * time.Millisecond,
    })

    received := make(chan engine.Event, 1)
    bus.Subscribe("price.update", func(e engine.Event) {
        received <- e
    })

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go ps.Start(ctx)

    select {
    case ev := <-received:
        if ev.Data["symbol"] != "BTC/USDT" {
            t.Errorf("expected symbol BTC/USDT, got %v", ev.Data["symbol"])
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for price update")
    }
}

func TestPriceStreamer_CalculatesChangePct(t *testing.T) {
    ps := &PriceStreamer{prevPrices: make(map[string]float64)}
    ps.prevPrices["BTC/USDT"] = 60000

    pct := ps.calcChangePct("BTC/USDT", 63000)
    if pct != 5.0 {
        t.Errorf("expected 5.0%%, got %f%%", pct)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/streaming/ -run TestPriceStreamer -v`
Expected: FAIL

**Step 3: Implement PriceStreamer**

`internal/streaming/price_streamer.go`:

- `PriceStreamerConfig`: Adapters, Bus, Symbols, PollInterval
- `PriceStreamer` struct: config, prevPrices map, mu sync.RWMutex
- `NewPriceStreamer(cfg) *PriceStreamer`
- `Start(ctx)` — for each connected adapter:
  - Try to type-assert to concrete Binance/Bybit adapter that has `OnPrice`
  - If has `OnPrice`, register callback that publishes to engine.EventBus
  - Call `SubscribePrices(ctx, symbols)` for WebSocket streaming
  - Otherwise, start polling goroutine: every PollInterval, call `GetPrice` for each symbol
- `pollLoop(ctx, adp, symbols)` — polling fallback
- `publishPrice(price adapter.Price)` — creates engine.Event with symbol, last, bid, ask, volume_24h, change_pct and publishes to bus
- `calcChangePct(symbol string, currentPrice float64) float64` — `(current - prev) / prev * 100`

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/streaming/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/streaming/price_streamer.go internal/streaming/price_streamer_test.go
git commit -m "feat(streaming): add hybrid PriceStreamer with WS and polling"
```

---

### Task 3: Portfolio Poller (change-detected portfolio updates)

**Files:**
- Create: `internal/streaming/portfolio_poller.go`
- Create: `internal/streaming/portfolio_poller_test.go`

**Step 1: Write the failing test**

```go
func TestPortfolioPoller_PublishesOnChange(t *testing.T) {
    bus := engine.NewEventBus()
    adp := &mockAdapter{name: "test", connected: true}

    pp := NewPortfolioPoller(PortfolioPollerConfig{
        Adapters:     map[string]adapter.TradingAdapter{"test": adp},
        Bus:          bus,
        PollInterval: 100 * time.Millisecond,
    })

    received := make(chan engine.Event, 1)
    bus.Subscribe("portfolio.update", func(e engine.Event) {
        received <- e
    })

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    go pp.Start(ctx)

    select {
    case ev := <-received:
        if ev.Type != "portfolio.update" {
            t.Errorf("expected type 'portfolio.update', got %q", ev.Type)
        }
    case <-time.After(2 * time.Second):
        t.Fatal("timeout waiting for portfolio update")
    }
}

func TestPortfolioPoller_HashChanges(t *testing.T) {
    pp := &PortfolioPoller{}
    h1 := pp.hashState([]adapter.Balance{{Asset: "USDT", Total: 1000}}, nil)
    h2 := pp.hashState([]adapter.Balance{{Asset: "USDT", Total: 1000}}, nil)
    h3 := pp.hashState([]adapter.Balance{{Asset: "USDT", Total: 1001}}, nil)

    if h1 != h2 {
        t.Error("same state should produce same hash")
    }
    if h1 == h3 {
        t.Error("different state should produce different hash")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/streaming/ -run TestPortfolioPoller -v`
Expected: FAIL

**Step 3: Implement PortfolioPoller**

`internal/streaming/portfolio_poller.go`:

- `PortfolioPollerConfig`: Adapters, Bus, PollInterval (default 30s)
- `PortfolioPoller` struct: config, lastHash string
- `NewPortfolioPoller(cfg) *PortfolioPoller`
- `Start(ctx)` — ticker loop at PollInterval, calls `poll(ctx)` each tick
- `poll(ctx)` — for first connected adapter:
  1. Get balances and positions
  2. Compute hash of state
  3. If hash != lastHash, publish `portfolio.update` event
  4. Update lastHash
- `hashState(balances, positions) string` — JSON marshal + hash for change detection
- Event data: `{balances: [...], positions: [...], total_pnl: float64}`

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/streaming/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/streaming/portfolio_poller.go internal/streaming/portfolio_poller_test.go
git commit -m "feat(streaming): add PortfolioPoller with change detection"
```

---

### Task 4: Trade event hook

**Files:**
- Modify: `internal/agent/tools.go` — add trade event publishing in execPlaceOrder

**Step 1: Add engine.EventBus to ToolRegistry**

In `internal/agent/tools.go`, add a `bus` field to `ToolRegistry`:

```go
type ToolRegistry struct {
    adapters   map[string]adapter.TradingAdapter
    riskEngine *risk.Engine
    mcpBridge  MCPBridge
    bus        *engine.EventBus  // NEW
}
```

Update `NewToolRegistry` to accept and store the bus:

```go
func NewToolRegistry(adapters map[string]adapter.TradingAdapter, riskEngine *risk.Engine, bus *engine.EventBus) *ToolRegistry {
    return &ToolRegistry{
        adapters:   adapters,
        riskEngine: riskEngine,
        bus:        bus,
    }
}
```

**Step 2: Publish trade.executed in execPlaceOrder**

After successful `adp.PlaceOrder(ctx, order)`, add:

```go
if s.bus != nil {
    s.bus.Publish(engine.Event{
        Type: "trade.executed",
        Data: map[string]any{
            "symbol":   result.Symbol,
            "side":     string(result.Side),
            "size":     result.Size,
            "price":    result.FilledAt,
            "type":     string(result.Type),
            "status":   string(result.Status),
            "order_id": result.ID,
            "exchange": result.Exchange,
        },
    })
}
```

**Step 3: Update callers of NewToolRegistry**

In `internal/agent/agent.go`, update the `New` function to pass bus:

```go
func New(cfg *config.Config, adapters map[string]adapter.TradingAdapter, riskEngine *risk.Engine, mem *memory.Store, bus *engine.EventBus) *Agent {
    return &Agent{
        cfg:     cfg,
        tools:   NewToolRegistry(adapters, riskEngine, bus),
        context: NewContextBuilder(cfg, adapters, riskEngine, mem),
        memory:  mem,
    }
}
```

Update `internal/api/server.go` where `agent.New` is called — pass the `bus` parameter.

**Step 4: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/agent/tools.go internal/agent/agent.go internal/api/server.go
git commit -m "feat(agent): publish trade.executed events on order placement"
```

---

### Task 5: Wire streaming into main.go and expand Hub subscriptions

**Files:**
- Modify: `cmd/clawtrade/main.go` — start PriceStreamer, Bridge, PortfolioPoller
- Modify: `internal/api/server.go` — add `price.*`, `agent.*`, `portfolio.*` to Hub subscriptions

**Step 1: Update Hub event subscriptions in server.go**

Change the `SubscribeToEvents` call to include new event types:

```go
hub.SubscribeToEvents(bus, []string{
    "market.*",
    "trade.*",
    "risk.*",
    "system.*",
    "price.*",      // NEW: live price updates
    "agent.*",      // NEW: sub-agent insights
    "portfolio.*",  // NEW: portfolio changes
})
```

**Step 2: Start streaming services in main.go**

After server creation and agent manager setup, add:

```go
// Start price streamer
priceStreamer := streaming.NewPriceStreamer(streaming.PriceStreamerConfig{
    Adapters:     adapters,
    Bus:          bus,
    Symbols:      cfg.Agent.Watchlist,
    PollInterval: 2 * time.Second,
})
go priceStreamer.Start(ctx)

// Start SubAgent-to-EventBus bridge
if agentMgr != nil {
    bridge := streaming.NewBridge(agentMgr.Bus(), bus)
    bridge.Start()
    defer bridge.Stop()
}

// Start portfolio poller
portfolioPoller := streaming.NewPortfolioPoller(streaming.PortfolioPollerConfig{
    Adapters:     adapters,
    Bus:          bus,
    PollInterval: 30 * time.Second,
})
go portfolioPoller.Start(ctx)
```

Add import: `"github.com/clawtrade/clawtrade/internal/streaming"`

**Step 3: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

**Step 4: Commit**

```bash
git add cmd/clawtrade/main.go internal/api/server.go
git commit -m "feat: wire streaming services into main and expand Hub subscriptions"
```
