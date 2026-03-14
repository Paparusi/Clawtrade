# AI Sub-Agents Design

**Goal:** Add background AI sub-agents that run continuously, each powered by LLM, performing advanced analysis that human traders cannot replicate manually.

**Core Principle:** Every sub-agent MUST require LLM intelligence. If it can be done with simple rules/thresholds, it's not a sub-agent — it's a utility function.

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                    AgentManager                           │
│                                                          │
│  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐      │
│  │ Market  │ │ Devil's │ │Narrative│ │Reflection│      │
│  │ Analyst │ │Advocate │ │  Agent  │ │  Agent   │      │
│  └────┬────┘ └────┬────┘ └────┬────┘ └────┬─────┘      │
│       │           │           │            │             │
│       └───────────┴─────┬─────┴────────────┘             │
│                         │                                │
│                    EventBus                               │
│                    (channels)                             │
│                         │                                │
│              ┌──────────┴──────────┐                     │
│              │   Main Agent        │                     │
│              │   (user-facing)     │                     │
│              └─────────────────────┘                     │
└──────────────────────────────────────────────────────────┘
```

### EventBus

Simple Go channel-based pub/sub:

```go
type Event struct {
    Type      string         // "analysis", "alert", "reflection", "narrative"
    Source    string         // sub-agent name
    Symbol   string         // trading pair (if applicable)
    Data     map[string]any // structured payload
    Priority int            // 0=info, 1=warning, 2=urgent
    Time     time.Time
}

type EventBus struct {
    subscribers map[string][]chan Event
    mu          sync.RWMutex
}
```

### AgentManager

Lifecycle management for sub-agents:

```go
type SubAgent interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Status() SubAgentStatus
}

type AgentManager struct {
    agents   map[string]SubAgent
    bus      *EventBus
    cfg      *config.Config
    adapters map[string]adapter.TradingAdapter
}
```

- Starts/stops sub-agents based on config
- Monitors health, restarts on crash
- Provides shared resources (adapters, config, memory, LLM access)

---

## Sub-Agent 1: Market Analyst

**Purpose:** Multi-expert LLM analysis using configurable methodology prompts.

### LLM-Native Analysis Pipeline

```
Raw OHLCV + Volume + Orderbook + Correlated pairs
        │
        ├── LLM Pass 1: Expert A (e.g. Price Action)
        ├── LLM Pass 2: Expert B (e.g. SMC)
        ├── LLM Pass 3: Expert C (e.g. ICT)
        │        (parallel LLM calls)
        │
        ▼
   Synthesis LLM Call
   - Combine expert analyses
   - Find confluence zones
   - Rate conviction
        │
        ▼
   Memory Comparison
   - "Last time this pattern appeared..."
   - Historical accuracy of each expert
        │
        ▼
   Final: Trade Thesis + Confidence Score
```

### User-Configurable Strategy Files

Strategies are markdown files in `~/.clawtrade/strategies/`:

```markdown
# ~/.clawtrade/strategies/smc.md

---
name: Smart Money Concepts
description: Analyze market using Order Blocks, FVG, Liquidity concepts
author: user
version: 1.0
default_timeframes: ["15m", "1h", "4h"]
requires_data: ["candles", "volume"]
---

You are an expert in Smart Money Concepts (SMC) trading methodology.
Given the raw OHLCV data below, perform the following analysis:

## 1. Market Structure
- Identify HH, HL, LH, LL on each timeframe
- Detect Break of Structure (BOS) and Change of Character (CHoCH)
- Determine current trend

## 2. Order Blocks
- Find the last opposing candle before impulsive moves
- Bullish OB: last bearish candle before strong bullish move
- Bearish OB: last bullish candle before strong bearish move
- Mark if mitigated or unmitigated

## 3. Fair Value Gaps (FVG)
- Identify 3-candle imbalances
- Note which FVGs are filled vs unfilled
- Rate fill probability

## 4. Liquidity
- Equal highs/lows (stop hunt targets)
- Trendline liquidity
- Session highs/lows

## Output Format
- **Bias**: bullish / bearish / neutral
- **Confidence**: 0-100%
- **Key Levels**: price levels with labels (OB, FVG, liquidity)
- **Setups**: actionable setups with entry/SL/TP
- **Reasoning**: step-by-step logic
```

### Strategy Loading

```go
type Strategy struct {
    Name       string   `yaml:"name"`
    Desc       string   `yaml:"description"`
    Author     string   `yaml:"author"`
    Version    string   `yaml:"version"`
    Timeframes []string `yaml:"default_timeframes"`
    Requires   []string `yaml:"requires_data"`
    Prompt     string   // markdown body after frontmatter
}

// LoadStrategies reads all .md files from strategies directory
func LoadStrategies(dir string) ([]Strategy, error)
```

- Hot-reload: watch directory for changes, reload without restart
- Built-in defaults: ship with `price_action.md`, `smc.md`, `ict.md`, `volume_profile.md`
- User creates new strategy: just add a `.md` file

### Config

```yaml
agent:
  analysis:
    strategies_dir: "~/.clawtrade/strategies"
    active_strategies: ["price_action", "smc", "ict"]
    weights:
      price_action: 0.4
      smc: 0.35
      ict: 0.25
    scan_interval: 300          # seconds between full scans
    expert_model: "anthropic/claude-haiku-4-5-20251001"
    synthesis_model: "anthropic/claude-sonnet-4-6"
    min_confluence: 2           # need N experts to agree
    timeframes: ["15m", "1h", "4h"]
```

### Data Collector

Code collects raw data, formats for LLM consumption:

```go
type MarketSnapshot struct {
    Symbol     string
    Candles    map[string][]adapter.Candle  // timeframe -> candles
    OrderBook  *adapter.OrderBook
    Price      *adapter.Price
    Correlated map[string]*adapter.Price    // BTC, ETH, DXY prices
}

func (ma *MarketAnalyst) collectSnapshot(ctx context.Context, symbol string) (*MarketSnapshot, error)
func (ma *MarketAnalyst) formatForLLM(snap *MarketSnapshot) string
```

The `formatForLLM` function converts raw data into a structured text format:
```
## BTC/USDT Raw Data

### 4h Candles (last 50)
Time | Open | High | Low | Close | Volume
2026-03-14 00:00 | 64250.00 | 64890.00 | 63800.00 | 64720.50 | 1234.56
...

### 1h Candles (last 100)
...

### Current Orderbook (20 levels)
Bids: 64700.00 (12.5), 64690.00 (8.3), ...
Asks: 64710.00 (5.2), 64720.00 (15.8), ...

### Correlated Assets
BTC/USDT: $64,720  ETH/USDT: $3,450  DXY: 104.2
```

---

## Sub-Agent 2: Devil's Advocate

**Purpose:** Automatically counter-argue every trade thesis to prevent confirmation bias.

### How It Works

1. Receives `analysis` events from Market Analyst via EventBus
2. Takes the thesis and **argues the opposite** with a dedicated LLM call
3. Produces a counter-analysis with specific risks and failure scenarios
4. Synthesis agent weighs both sides

### LLM Prompt Structure

```
You are a Devil's Advocate trader. Your ONLY job is to find reasons
why the following trade thesis is WRONG.

## The Thesis
{market_analyst_output}

## Your Task
1. Find every reason this trade could fail
2. Identify risks the analyst missed
3. Check for confirmation bias in the analysis
4. Look for contradicting evidence in the data
5. Rate your counter-argument strength: 0-100%

## Rules
- You MUST argue against the thesis, even if you agree with it
- Be specific: cite price levels, patterns, data points
- Consider: macro conditions, funding rates, liquidation levels,
  historical failure rate of similar setups
- If the thesis is very strong, say so but still present risks

## Output Format
- **Counter-bias**: the opposite direction
- **Counter-confidence**: 0-100%
- **Risks**: specific risk factors
- **Failure scenarios**: what could go wrong
- **Verdict**: "weak thesis" / "moderate thesis" / "strong thesis despite risks"
```

### Confluence Adjustment

```go
type SynthesisResult struct {
    OriginalConfidence float64
    CounterConfidence  float64
    AdjustedConfidence float64  // weighted combination
    Risks             []string
    Verdict           string   // "proceed", "reduce_size", "skip"
}
```

If Devil's Advocate counter-confidence > 60%, the system:
- Reduces position size suggestion by 50%
- Adds risk warnings to the thesis
- Requires user confirmation even in auto-trade mode

---

## Sub-Agent 3: Narrative Agent

**Purpose:** Understand market narratives and story arcs, not just price/sentiment.

### What It Tracks

- **Narrative lifecycle**: forming → growing → peaking → fading → dead
- **Sector rotations**: capital flow between narratives (DeFi → AI → L2 → Meme)
- **Macro narratives**: Fed policy, regulations, ETF flows, halving cycles
- **Micro narratives**: project-specific catalysts, partnerships, launches

### Data Sources

- Watchlist price changes + volume (already available via adapters)
- User-provided context (through chat)
- Trade history patterns (from memory store)
- Cross-asset correlation changes

### LLM Prompt (runs every scan_interval)

```
You are a Crypto Narrative Analyst. Analyze the current market data
to identify active narratives and their lifecycle phase.

## Current Market Data
{prices and volume changes for watchlist symbols}

## Recent Trade History
{from memory store}

## Your Analysis
1. Identify active narratives from price/volume patterns
2. Rate each narrative's lifecycle phase
3. Detect sector rotation signals
4. Flag narratives that are peaking (danger) or forming (opportunity)

## Output
For each narrative:
- **Name**: short label
- **Phase**: forming / growing / peaking / fading / dead
- **Evidence**: price/volume data supporting this assessment
- **Tokens**: which tokens belong to this narrative
- **Action**: opportunity / caution / avoid
```

### Memory Integration

The Narrative Agent stores its assessments in the memory store:
- Track narrative accuracy over time
- Learn which narrative signals are predictive
- Build a "narrative history" the main agent can reference

---

## Sub-Agent 4: Reflection Agent

**Purpose:** Post-trade analysis and behavioral pattern detection. The AI trading psychologist.

### Triggers

- After every closed trade (win or loss)
- Daily summary (end of trading day)
- Weekly performance review

### Post-Trade Reflection

```
You are a Trading Psychologist and Performance Analyst.

## The Trade
Symbol: {symbol}
Side: {side}
Entry: ${entry} at {time}
Exit: ${exit} at {time}
PnL: ${pnl} ({pnl_pct}%)
Original reasoning: {reasoning from episode}
Strategy used: {strategy}
Market conditions at entry: {snapshot}

## Recent Trade History
{last 20 trades with outcomes}

## Your Analysis
1. Was the entry thesis correct? Why or why not?
2. Was the exit optimal? Too early? Too late?
3. Did the trader follow their own rules?
4. Detect behavioral patterns:
   - Revenge trading (loss → immediate re-entry)
   - FOMO (entered after move already happened)
   - Overtrading (too many trades in short period)
   - Emotional exits (panic sell / greed hold)
5. Strategy effectiveness: is this strategy working?
6. Generate a learned rule if pattern confidence > 70%

## Output
- **Trade grade**: A/B/C/D/F
- **Key lesson**: one sentence
- **Behavioral flags**: any detected patterns
- **Rule suggestion**: if warranted (with confidence %)
- **Strategy adjustment**: should strategy weight change?
```

### Rule Generation

When the Reflection Agent detects a pattern with high confidence, it creates a new Rule in the memory store:

```go
// Auto-generated rule example:
Rule{
    Content:       "Avoid counter-trend longs when 4h structure is bearish (3/3 losses in similar setups)",
    Category:      "entry_filter",
    Confidence:    0.85,
    EvidenceCount: 3,
    Source:        "reflection_agent",
}
```

These rules are loaded into the main agent's system prompt via `addMemoryContext()`, creating a **feedback loop** where the AI learns from its own mistakes.

### Strategy Weight Evolution

```go
type StrategyPerformance struct {
    Name       string
    WinRate    float64
    AvgPnL     float64
    SampleSize int
    Weight     float64  // auto-adjusted
}
```

The Reflection Agent tracks which strategies produce winning trades and adjusts weights accordingly. If "SMC" has 70% win rate but "ICT" has 40%, the system increases SMC weight in the synthesis pass.

---

## Sub-Agent 5: Correlation Agent

**Purpose:** Discover non-obvious relationships between assets and detect regime changes.

### What It Analyzes

- **Cross-asset correlations**: BTC vs ETH, BTC vs DXY, BTC vs Gold, sector correlations
- **Correlation breakdowns**: when historically correlated assets diverge (signal!)
- **Lead-lag relationships**: which asset moves first?
- **Regime detection**: trending → ranging → volatile → quiet

### LLM Analysis

```
You are a Cross-Market Correlation Analyst.

## Price Data (24h changes)
{all watchlist symbols + BTC + ETH with % changes and volume}

## Historical Correlations (30d rolling)
BTC/ETH: 0.85, BTC/SOL: 0.72, ETH/SOL: 0.68, ...

## Current Correlations (7d rolling)
BTC/ETH: 0.45, BTC/SOL: 0.81, ETH/SOL: 0.33, ...

## Your Analysis
1. Which correlations have broken? What does it mean?
2. Lead-lag detection: which asset is leading?
3. Capital flow direction: where is money going?
4. Regime assessment: what phase is the market in?
5. Anomaly detection: anything unusual?

## Output
- **Regime**: trending_up / trending_down / ranging / volatile / transitioning
- **Correlation breaks**: list with significance
- **Capital flow**: from → to
- **Lead asset**: which asset is leading and why
- **Implications**: what this means for trading decisions
```

---

## Built-in Strategy Files

Ship with these defaults in `strategies/` directory:

### price_action.md
- Market structure (HH/HL/LH/LL, BOS, CHoCH)
- Candlestick patterns (engulfing, pin bar, inside bar, doji)
- Support/resistance zones
- Trend lines and channels
- Multi-timeframe structure alignment

### smc.md
- Order blocks (bullish/bearish, mitigated/unmitigated)
- Fair Value Gaps (FVG)
- Liquidity pools (equal highs/lows, trendline liquidity)
- Premium/discount zones (Fibonacci)
- Breaker blocks and mitigation blocks

### ict.md
- Kill zones (London open, NY open, Asia)
- Optimal Trade Entry (OTE at 62-79% fib)
- Displacement candles and Market Structure Shift (MSS)
- Daily/weekly bias from HTF
- Institutional order flow concepts

### volume_profile.md
- Point of Control (POC), Value Area High/Low
- Volume delta and cumulative delta
- Naked POCs as price magnets
- High volume nodes (support/resistance)
- Low volume nodes (fast-move zones)

---

## Configuration

```yaml
agent:
  sub_agents:
    - name: "market-analyst"
      enabled: true
      scan_interval: 300
      expert_model: "anthropic/claude-haiku-4-5-20251001"
      synthesis_model: "anthropic/claude-sonnet-4-6"
    - name: "devils-advocate"
      enabled: true
      model: "anthropic/claude-haiku-4-5-20251001"
    - name: "narrative"
      enabled: true
      scan_interval: 900
      model: "anthropic/claude-haiku-4-5-20251001"
    - name: "reflection"
      enabled: true
      model: "anthropic/claude-sonnet-4-6"
    - name: "correlation"
      enabled: true
      scan_interval: 600
      model: "anthropic/claude-haiku-4-5-20251001"

  analysis:
    strategies_dir: "~/.clawtrade/strategies"
    active_strategies: ["price_action", "smc"]
    weights:
      price_action: 0.5
      smc: 0.5
    min_confluence: 2
    timeframes: ["15m", "1h", "4h"]
```

---

## File Structure

```
internal/
  subagent/
    manager.go          # AgentManager, lifecycle, health monitoring
    bus.go              # EventBus (channel-based pub/sub)
    types.go            # SubAgent interface, Event, shared types
    strategy.go         # Strategy loading from markdown files
    market_analyst.go   # Market Analyst sub-agent
    devils_advocate.go  # Devil's Advocate sub-agent
    narrative.go        # Narrative Agent sub-agent
    reflection.go       # Reflection Agent sub-agent
    correlation.go      # Correlation Agent sub-agent
    llm.go              # Shared LLM calling helpers (multi-provider)

strategies/             # Built-in strategy files (shipped with binary)
  price_action.md
  smc.md
  ict.md
  volume_profile.md
```

---

## Key Design Decisions

1. **LLM receives raw data, not pre-computed results** — Code only collects and formats data. LLM does the actual analysis. This is what makes it AI, not just automation.

2. **Strategies as markdown files** — Users write natural language methodology descriptions. No code required. Hot-reloadable.

3. **Multi-pass pipeline** — Multiple expert LLM calls in parallel, then synthesis. More thorough than single-pass analysis.

4. **Devil's Advocate is mandatory** — Every thesis gets stress-tested. Prevents confirmation bias.

5. **Reflection creates feedback loop** — Post-trade analysis generates rules that feed back into the main agent's system prompt. The AI learns from its own trades.

6. **Each sub-agent has its own model config** — Use cheap models (Haiku) for frequent scans, expensive models (Sonnet/Opus) for synthesis and reflection.

7. **EventBus decouples sub-agents** — Sub-agents communicate via events, not direct calls. Easy to add/remove sub-agents without changing others.
