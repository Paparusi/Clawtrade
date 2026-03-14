# AI Sub-Agents Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build 5 AI-powered sub-agents (Market Analyst, Devil's Advocate, Narrative, Reflection, Correlation) running as background goroutines with EventBus communication, user-configurable strategy files, and multi-provider LLM support.

**Architecture:** EventBus (Go channels) + AgentManager + SubAgent interface. Each sub-agent runs in its own goroutine, calls LLM independently, publishes results to EventBus.

**Tech Stack:** Go, existing LLM providers (Anthropic/OpenAI/Google), gorilla/websocket (for future dashboard push), existing memory store

---

### Task 1: EventBus and core types

**Files:**
- Create: `internal/subagent/types.go`
- Create: `internal/subagent/bus.go`
- Create: `internal/subagent/bus_test.go`

**Step 1: Write the failing test**

```go
// internal/subagent/bus_test.go
package subagent

import (
    "testing"
    "time"
)

func TestEventBus_PublishSubscribe(t *testing.T) {
    bus := NewEventBus()
    ch := bus.Subscribe("analysis")

    go func() {
        bus.Publish(Event{
            Type:   "analysis",
            Source: "test",
            Symbol: "BTC/USDT",
            Data:   map[string]any{"bias": "bullish"},
        })
    }()

    select {
    case ev := <-ch:
        if ev.Type != "analysis" {
            t.Errorf("expected type 'analysis', got %q", ev.Type)
        }
        if ev.Symbol != "BTC/USDT" {
            t.Errorf("expected symbol 'BTC/USDT', got %q", ev.Symbol)
        }
    case <-time.After(time.Second):
        t.Fatal("timeout waiting for event")
    }
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
    bus := NewEventBus()
    ch1 := bus.Subscribe("alert")
    ch2 := bus.Subscribe("alert")

    bus.Publish(Event{Type: "alert", Source: "test"})

    for _, ch := range []<-chan Event{ch1, ch2} {
        select {
        case ev := <-ch:
            if ev.Type != "alert" {
                t.Error("wrong event type")
            }
        case <-time.After(time.Second):
            t.Fatal("timeout")
        }
    }
}

func TestEventBus_Unsubscribe(t *testing.T) {
    bus := NewEventBus()
    ch := bus.Subscribe("analysis")
    bus.Unsubscribe("analysis", ch)

    bus.Publish(Event{Type: "analysis", Source: "test"})

    select {
    case <-ch:
        t.Fatal("should not receive after unsubscribe")
    case <-time.After(100 * time.Millisecond):
        // expected
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestEventBus -v`
Expected: FAIL (package doesn't exist)

**Step 3: Implement types and EventBus**

`internal/subagent/types.go`:
```go
package subagent

import (
    "context"
    "time"
)

type Event struct {
    Type     string         `json:"type"`
    Source   string         `json:"source"`
    Symbol   string         `json:"symbol,omitempty"`
    Data     map[string]any `json:"data,omitempty"`
    Priority int            `json:"priority"`
    Time     time.Time      `json:"time"`
}

type SubAgentStatus struct {
    Name      string    `json:"name"`
    Running   bool      `json:"running"`
    LastRun   time.Time `json:"last_run"`
    RunCount  int       `json:"run_count"`
    ErrorCount int      `json:"error_count"`
    LastError string    `json:"last_error,omitempty"`
}

type SubAgent interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Status() SubAgentStatus
}
```

`internal/subagent/bus.go`:
```go
package subagent

import (
    "sync"
    "time"
)

type EventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
}

func NewEventBus() *EventBus {
    return &EventBus{
        subscribers: make(map[string][]chan Event),
    }
}

func (b *EventBus) Subscribe(eventType string) <-chan Event {
    b.mu.Lock()
    defer b.mu.Unlock()
    ch := make(chan Event, 64)
    b.subscribers[eventType] = append(b.subscribers[eventType], ch)
    return ch
}

func (b *EventBus) Unsubscribe(eventType string, ch <-chan Event) {
    b.mu.Lock()
    defer b.mu.Unlock()
    subs := b.subscribers[eventType]
    for i, s := range subs {
        if s == ch {
            b.subscribers[eventType] = append(subs[:i], subs[i+1:]...)
            close(s)
            return
        }
    }
}

func (b *EventBus) Publish(ev Event) {
    if ev.Time.IsZero() {
        ev.Time = time.Now()
    }
    b.mu.RLock()
    defer b.mu.RUnlock()
    for _, ch := range b.subscribers[ev.Type] {
        select {
        case ch <- ev:
        default:
            // drop if subscriber is full
        }
    }
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/subagent/types.go internal/subagent/bus.go internal/subagent/bus_test.go
git commit -m "feat(subagent): add EventBus and core types"
```

---

### Task 2: Strategy loader (markdown file parsing)

**Files:**
- Create: `internal/subagent/strategy.go`
- Create: `internal/subagent/strategy_test.go`
- Create: `strategies/price_action.md`
- Create: `strategies/smc.md`
- Create: `strategies/ict.md`
- Create: `strategies/volume_profile.md`

**Step 1: Write the failing test**

```go
// internal/subagent/strategy_test.go
package subagent

import (
    "os"
    "path/filepath"
    "testing"
)

func TestParseStrategy(t *testing.T) {
    content := `---
name: Test Strategy
description: A test strategy
author: test
version: "1.0"
default_timeframes: ["1h", "4h"]
requires_data: ["candles", "volume"]
---

You are an expert test analyst.

## Analysis Steps
1. Check the trend
2. Find key levels
`
    s, err := ParseStrategy(content)
    if err != nil {
        t.Fatalf("parse error: %v", err)
    }
    if s.Name != "Test Strategy" {
        t.Errorf("expected name 'Test Strategy', got %q", s.Name)
    }
    if len(s.Timeframes) != 2 {
        t.Errorf("expected 2 timeframes, got %d", len(s.Timeframes))
    }
    if s.Prompt == "" {
        t.Error("expected non-empty prompt")
    }
    if s.Prompt[0] == '-' {
        t.Error("prompt should not include frontmatter")
    }
}

func TestLoadStrategies(t *testing.T) {
    dir := t.TempDir()

    // Write test strategy files
    s1 := `---
name: Strategy A
description: Test A
default_timeframes: ["1h"]
requires_data: ["candles"]
---

Analyze using method A.
`
    s2 := `---
name: Strategy B
description: Test B
default_timeframes: ["4h"]
requires_data: ["candles", "volume"]
---

Analyze using method B.
`
    os.WriteFile(filepath.Join(dir, "strat_a.md"), []byte(s1), 0644)
    os.WriteFile(filepath.Join(dir, "strat_b.md"), []byte(s2), 0644)
    os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("not a strategy"), 0644)

    strategies, err := LoadStrategies(dir)
    if err != nil {
        t.Fatalf("load error: %v", err)
    }
    if len(strategies) != 2 {
        t.Errorf("expected 2 strategies, got %d", len(strategies))
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestParseStrategy -v`
Expected: FAIL

**Step 3: Implement strategy loader**

`internal/subagent/strategy.go` — parse YAML frontmatter from markdown, load all `.md` files from directory. Use `gopkg.in/yaml.v3` (already a dependency).

Key functions:
- `ParseStrategy(content string) (*Strategy, error)` — split frontmatter/body, parse YAML
- `LoadStrategies(dir string) ([]Strategy, error)` — glob `*.md`, parse each
- Strategy struct: Name, Description, Author, Version, Timeframes, Requires, Prompt, Slug (filename without .md)

**Step 4: Create built-in strategy files**

Create 4 strategy markdown files in `strategies/` directory:

`strategies/price_action.md` — Market structure, candlestick patterns, S/R zones, trendlines, multi-TF alignment

`strategies/smc.md` — Order blocks, FVG, liquidity pools, premium/discount, breaker blocks

`strategies/ict.md` — Kill zones, OTE, displacement, MSS, daily/weekly bias

`strategies/volume_profile.md` — POC, VAH/VAL, volume delta, naked POCs, HVN/LVN

Each file follows the frontmatter format from the design doc.

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/subagent/strategy.go internal/subagent/strategy_test.go strategies/
git commit -m "feat(subagent): add strategy loader and built-in strategy files"
```

---

### Task 3: Shared LLM caller

**Files:**
- Create: `internal/subagent/llm.go`
- Create: `internal/subagent/llm_test.go`

**Step 1: Write the failing test**

```go
// internal/subagent/llm_test.go
package subagent

import (
    "testing"
)

func TestBuildAnthropicRequest(t *testing.T) {
    caller := &LLMCaller{
        Provider:  "anthropic",
        Model:     "claude-haiku-4-5-20251001",
        APIKey:    "test-key",
        MaxTokens: 2048,
    }
    body := caller.buildAnthropicBody("system prompt", "user message")
    if body["model"] != "claude-haiku-4-5-20251001" {
        t.Error("wrong model")
    }
    if body["system"] != "system prompt" {
        t.Error("wrong system prompt")
    }
}

func TestBuildOpenAIRequest(t *testing.T) {
    caller := &LLMCaller{
        Provider:  "openai",
        Model:     "gpt-4o-mini",
        APIKey:    "test-key",
        MaxTokens: 2048,
    }
    body := caller.buildOpenAIBody("system prompt", "user message")
    if body["model"] != "gpt-4o-mini" {
        t.Error("wrong model")
    }
}

func TestNewLLMCallerFromConfig(t *testing.T) {
    caller := NewLLMCaller("anthropic/claude-haiku-4-5-20251001", "test-key", 2048)
    if caller.Provider != "anthropic" {
        t.Errorf("expected provider 'anthropic', got %q", caller.Provider)
    }
    if caller.Model != "claude-haiku-4-5-20251001" {
        t.Errorf("expected model 'claude-haiku-4-5-20251001', got %q", caller.Model)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestBuild -v`
Expected: FAIL

**Step 3: Implement shared LLM caller**

`internal/subagent/llm.go`:
- `LLMCaller` struct: Provider, Model, APIKey, MaxTokens, BaseURL
- `NewLLMCaller(modelString, apiKey string, maxTokens int) *LLMCaller` — parse "provider/model" format
- `Call(ctx context.Context, systemPrompt, userMessage string) (string, error)` — make HTTP request to appropriate provider API, return text response
- Support: Anthropic, OpenAI-compatible (OpenAI, DeepSeek, OpenRouter, Ollama), Google
- No tool calling needed — sub-agents use LLM for analysis only, not tool execution
- `buildAnthropicBody`, `buildOpenAIBody`, `buildGoogleBody` — request builders

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/subagent/llm.go internal/subagent/llm_test.go
git commit -m "feat(subagent): add shared multi-provider LLM caller"
```

---

### Task 4: AgentManager (lifecycle management)

**Files:**
- Create: `internal/subagent/manager.go`
- Create: `internal/subagent/manager_test.go`

**Step 1: Write the failing test**

```go
// internal/subagent/manager_test.go
package subagent

import (
    "context"
    "testing"
    "time"
)

type mockSubAgent struct {
    name    string
    started bool
    stopped bool
}

func (m *mockSubAgent) Name() string { return m.name }
func (m *mockSubAgent) Start(ctx context.Context) error {
    m.started = true
    <-ctx.Done()
    return nil
}
func (m *mockSubAgent) Stop() error {
    m.stopped = true
    return nil
}
func (m *mockSubAgent) Status() SubAgentStatus {
    return SubAgentStatus{Name: m.name, Running: m.started && !m.stopped}
}

func TestManager_RegisterAndStart(t *testing.T) {
    mgr := NewAgentManager(NewEventBus())
    agent := &mockSubAgent{name: "test-agent"}
    mgr.Register(agent)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    mgr.StartAll(ctx)
    time.Sleep(50 * time.Millisecond)

    statuses := mgr.Statuses()
    if len(statuses) != 1 {
        t.Fatalf("expected 1 status, got %d", len(statuses))
    }
    if !agent.started {
        t.Error("agent should be started")
    }

    mgr.StopAll()
    time.Sleep(50 * time.Millisecond)
}

func TestManager_GetEventBus(t *testing.T) {
    bus := NewEventBus()
    mgr := NewAgentManager(bus)
    if mgr.Bus() != bus {
        t.Error("expected same bus")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestManager -v`
Expected: FAIL

**Step 3: Implement AgentManager**

`internal/subagent/manager.go`:
- `AgentManager` struct: agents map, bus, cancel func, wg sync.WaitGroup
- `NewAgentManager(bus *EventBus) *AgentManager`
- `Register(agent SubAgent)` — add to agents map
- `StartAll(ctx context.Context)` — launch each agent in a goroutine
- `StopAll()` — cancel context, wait for all goroutines
- `Statuses() []SubAgentStatus` — collect status from all agents
- `Bus() *EventBus` — expose bus for wiring

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/subagent/manager.go internal/subagent/manager_test.go
git commit -m "feat(subagent): add AgentManager lifecycle management"
```

---

### Task 5: Market Analyst sub-agent

**Files:**
- Create: `internal/subagent/market_analyst.go`
- Create: `internal/subagent/market_analyst_test.go`

**Step 1: Write the failing test**

```go
func TestMarketAnalyst_Name(t *testing.T) {
    ma := NewMarketAnalyst(MarketAnalystConfig{})
    if ma.Name() != "market-analyst" {
        t.Errorf("expected 'market-analyst', got %q", ma.Name())
    }
}

func TestMarketAnalyst_FormatSnapshot(t *testing.T) {
    ma := NewMarketAnalyst(MarketAnalystConfig{})
    snap := &MarketSnapshot{
        Symbol: "BTC/USDT",
        Candles: map[string][]adapter.Candle{
            "1h": {{Open: 64000, High: 64500, Low: 63800, Close: 64200, Volume: 100}},
        },
    }
    text := ma.formatForLLM(snap)
    if text == "" {
        t.Error("expected non-empty formatted text")
    }
    if !strings.Contains(text, "BTC/USDT") {
        t.Error("should contain symbol")
    }
}

func TestMarketAnalyst_Status(t *testing.T) {
    ma := NewMarketAnalyst(MarketAnalystConfig{})
    status := ma.Status()
    if status.Running {
        t.Error("should not be running before start")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestMarketAnalyst -v`
Expected: FAIL

**Step 3: Implement Market Analyst**

`internal/subagent/market_analyst.go`:
- `MarketAnalystConfig` struct: Strategies []Strategy, ActiveStrategies []string, Weights map[string]float64, ScanInterval, Timeframes, ExpertModel, SynthesisModel, MinConfluence, Adapters, Bus, Memory
- `MarketSnapshot` struct: Symbol, Candles map[string][]Candle, OrderBook, Price, Correlated
- `NewMarketAnalyst(cfg MarketAnalystConfig) *MarketAnalyst`
- `Start(ctx)` — ticker loop at scan_interval
- `runScan(ctx)` — for each watchlist symbol:
  1. `collectSnapshot` — fetch OHLCV for each timeframe, orderbook, price, correlated assets
  2. `formatForLLM` — format raw data as structured text
  3. `runExperts` — parallel LLM calls with each active strategy prompt
  4. `synthesize` — LLM call combining expert outputs, finding confluence
  5. Publish `Event{Type: "analysis"}` to bus
- `collectSnapshot(ctx, symbol)` — use adapters to gather data
- `formatForLLM(snap)` — structured text with candles, orderbook, correlations
- `runExperts(ctx, formatted, strategies)` — concurrent LLM calls, collect results
- `synthesize(ctx, expertResults)` — final LLM call to combine and score

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/subagent/market_analyst.go internal/subagent/market_analyst_test.go
git commit -m "feat(subagent): add Market Analyst with multi-expert pipeline"
```

---

### Task 6: Devil's Advocate sub-agent

**Files:**
- Create: `internal/subagent/devils_advocate.go`
- Create: `internal/subagent/devils_advocate_test.go`

**Step 1: Write the failing test**

```go
func TestDevilsAdvocate_Name(t *testing.T) {
    da := NewDevilsAdvocate(DevilsAdvocateConfig{})
    if da.Name() != "devils-advocate" {
        t.Errorf("expected 'devils-advocate', got %q", da.Name())
    }
}

func TestDevilsAdvocate_BuildPrompt(t *testing.T) {
    da := NewDevilsAdvocate(DevilsAdvocateConfig{})
    thesis := "BTC is bullish because of strong 4h structure"
    prompt := da.buildCounterPrompt(thesis)
    if !strings.Contains(prompt, "WRONG") {
        t.Error("prompt should instruct to find reasons thesis is wrong")
    }
    if !strings.Contains(prompt, thesis) {
        t.Error("prompt should contain the original thesis")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/subagent/ -run TestDevilsAdvocate -v`
Expected: FAIL

**Step 3: Implement Devil's Advocate**

`internal/subagent/devils_advocate.go`:
- Subscribes to `"analysis"` events from EventBus
- On each analysis event, extracts thesis from event data
- Calls LLM with counter-argument system prompt
- Publishes `Event{Type: "counter_analysis"}` with counter-confidence, risks, verdict
- `buildCounterPrompt(thesis string) string` — the Devil's Advocate system prompt

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/subagent/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/subagent/devils_advocate.go internal/subagent/devils_advocate_test.go
git commit -m "feat(subagent): add Devil's Advocate counter-analysis agent"
```

---

### Task 7: Narrative Agent sub-agent

**Files:**
- Create: `internal/subagent/narrative.go`
- Create: `internal/subagent/narrative_test.go`

**Step 1: Write the failing test**

```go
func TestNarrativeAgent_Name(t *testing.T) {
    na := NewNarrativeAgent(NarrativeConfig{})
    if na.Name() != "narrative" {
        t.Errorf("expected 'narrative', got %q", na.Name())
    }
}

func TestNarrativeAgent_FormatMarketData(t *testing.T) {
    na := NewNarrativeAgent(NarrativeConfig{})
    prices := map[string]float64{"BTC/USDT": 64000, "ETH/USDT": 3400}
    changes := map[string]float64{"BTC/USDT": 2.5, "ETH/USDT": -1.2}
    text := na.formatMarketData(prices, changes)
    if !strings.Contains(text, "BTC/USDT") {
        t.Error("should contain symbol")
    }
}
```

**Step 2: Implement Narrative Agent**

`internal/subagent/narrative.go`:
- Runs on scan_interval (longer than Market Analyst, e.g. 15 min)
- Collects watchlist price changes and volume
- Queries memory store for recent trades
- Calls LLM to identify narratives, lifecycle phases, sector rotations
- Publishes `Event{Type: "narrative"}` with narrative assessments
- Stores narrative history in memory for tracking accuracy over time

**Step 3: Run tests, commit**

Run: `go test ./internal/subagent/ -v`

```bash
git add internal/subagent/narrative.go internal/subagent/narrative_test.go
git commit -m "feat(subagent): add Narrative Agent for market story analysis"
```

---

### Task 8: Reflection Agent sub-agent

**Files:**
- Create: `internal/subagent/reflection.go`
- Create: `internal/subagent/reflection_test.go`

**Step 1: Write the failing test**

```go
func TestReflectionAgent_Name(t *testing.T) {
    ra := NewReflectionAgent(ReflectionConfig{})
    if ra.Name() != "reflection" {
        t.Errorf("expected 'reflection', got %q", ra.Name())
    }
}

func TestReflectionAgent_BuildReflectionPrompt(t *testing.T) {
    ra := NewReflectionAgent(ReflectionConfig{})
    ep := memory.Episode{
        Symbol: "BTC/USDT", Side: "BUY",
        EntryPrice: 64000, ExitPrice: 63000,
        PnL: -100, Reasoning: "Strong support at 64k",
    }
    prompt := ra.buildReflectionPrompt(ep, nil)
    if !strings.Contains(prompt, "BTC/USDT") {
        t.Error("should contain symbol")
    }
    if !strings.Contains(prompt, "-100") {
        t.Error("should contain PnL")
    }
}
```

**Step 2: Implement Reflection Agent**

`internal/subagent/reflection.go`:
- Triggers on new closed episodes in memory store (polls periodically)
- For each new closed trade, calls LLM with trade details + recent history
- LLM grades the trade, detects behavioral patterns, suggests rules
- If rule confidence > 70%, creates Rule in memory store
- Tracks strategy performance, adjusts weights
- Publishes `Event{Type: "reflection"}` with insights

**Step 3: Run tests, commit**

Run: `go test ./internal/subagent/ -v`

```bash
git add internal/subagent/reflection.go internal/subagent/reflection_test.go
git commit -m "feat(subagent): add Reflection Agent for post-trade analysis"
```

---

### Task 9: Correlation Agent sub-agent

**Files:**
- Create: `internal/subagent/correlation.go`
- Create: `internal/subagent/correlation_test.go`

**Step 1: Write the failing test**

```go
func TestCorrelationAgent_Name(t *testing.T) {
    ca := NewCorrelationAgent(CorrelationConfig{})
    if ca.Name() != "correlation" {
        t.Errorf("expected 'correlation', got %q", ca.Name())
    }
}

func TestCorrelationAgent_CalcCorrelation(t *testing.T) {
    // Perfect positive correlation
    a := []float64{1, 2, 3, 4, 5}
    b := []float64{2, 4, 6, 8, 10}
    corr := calcCorrelation(a, b)
    if corr < 0.99 {
        t.Errorf("expected ~1.0, got %f", corr)
    }

    // Perfect negative correlation
    c := []float64{5, 4, 3, 2, 1}
    corr2 := calcCorrelation(a, c)
    if corr2 > -0.99 {
        t.Errorf("expected ~-1.0, got %f", corr2)
    }
}
```

**Step 2: Implement Correlation Agent**

`internal/subagent/correlation.go`:
- Runs on scan_interval (e.g. 10 min)
- Calculates rolling correlations between watchlist pairs
- Compares 30d vs 7d correlations to detect breakdowns
- Calls LLM with correlation data to interpret meaning
- Detects regime changes (trending, ranging, volatile, transitioning)
- Publishes `Event{Type: "correlation"}` with regime and correlation analysis
- `calcCorrelation(a, b []float64) float64` — Pearson correlation coefficient

**Step 3: Run tests, commit**

Run: `go test ./internal/subagent/ -v`

```bash
git add internal/subagent/correlation.go internal/subagent/correlation_test.go
git commit -m "feat(subagent): add Correlation Agent for cross-market analysis"
```

---

### Task 10: Wire into main agent and config

**Files:**
- Modify: `internal/config/config.go` — add SubAgentConfig type
- Modify: `internal/agent/context.go` — include sub-agent insights in system prompt
- Modify: `cmd/clawtrade/main.go` — initialize AgentManager, start sub-agents

**Step 1: Update config**

Add to `config.go`:
```go
type SubAgentEntry struct {
    Name         string `yaml:"name"`
    Enabled      bool   `yaml:"enabled"`
    ScanInterval int    `yaml:"scan_interval"`
    Model        string `yaml:"model"`
}

type AnalysisConfig struct {
    StrategiesDir    string             `yaml:"strategies_dir"`
    ActiveStrategies []string           `yaml:"active_strategies"`
    Weights          map[string]float64 `yaml:"weights"`
    MinConfluence    int                `yaml:"min_confluence"`
    Timeframes       []string           `yaml:"timeframes"`
    ExpertModel      string             `yaml:"expert_model"`
    SynthesisModel   string             `yaml:"synthesis_model"`
}
```

Update AgentConfig to include `SubAgentEntries []SubAgentEntry` and `Analysis AnalysisConfig`.

**Step 2: Update context builder**

Add method to `ContextBuilder` that reads latest sub-agent events and includes them in the system prompt:
- Latest analysis results from Market Analyst
- Any active warnings from Devil's Advocate
- Current narrative assessment
- Recent reflection insights
- Market regime from Correlation Agent

**Step 3: Wire in main.go**

Initialize `AgentManager`, create sub-agents based on config, start them.

**Step 4: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

**Step 5: Commit**

```bash
git add internal/config/config.go internal/agent/context.go cmd/clawtrade/main.go
git commit -m "feat(subagent): wire sub-agents into main agent and config"
```

---

### Task 11: API endpoint for sub-agent status

**Files:**
- Modify: `internal/api/routes.go` — add `/api/agents` endpoint

**Step 1: Add endpoint**

- `GET /api/agents` — returns statuses of all sub-agents
- `GET /api/agents/events` — returns recent events from EventBus (last 50)
- `POST /api/agents/{name}/toggle` — enable/disable a sub-agent

**Step 2: Run tests**

Run: `go test ./... -count=1`
Expected: All PASS

**Step 3: Commit**

```bash
git add internal/api/routes.go
git commit -m "feat(api): add sub-agent status and control endpoints"
```
