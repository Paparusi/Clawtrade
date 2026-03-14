package subagent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

// NarrativeConfig holds the configuration for the Narrative Agent sub-agent.
type NarrativeConfig struct {
	LLM          *LLMCaller
	Bus          *EventBus
	Adapters     map[string]adapter.TradingAdapter
	Watchlist    []string
	ScanInterval time.Duration
}

// NarrativeAgent is a sub-agent that periodically scans market prices,
// calculates percentage changes, and uses an LLM to identify active
// crypto narratives and their lifecycle phases. It publishes "narrative"
// events with the analysis results.
type NarrativeAgent struct {
	cfg        NarrativeConfig
	running    bool
	lastRun    time.Time
	runCount   int
	errorCount int
	lastError  string
	cancel     context.CancelFunc
	prevPrices map[string]float64
	mu         sync.RWMutex
}

// NewNarrativeAgent creates a new NarrativeAgent with the given configuration.
func NewNarrativeAgent(cfg NarrativeConfig) *NarrativeAgent {
	if cfg.ScanInterval == 0 {
		cfg.ScanInterval = 15 * time.Minute
	}
	return &NarrativeAgent{
		cfg:        cfg,
		prevPrices: make(map[string]float64),
	}
}

// Name returns the sub-agent name.
func (na *NarrativeAgent) Name() string {
	return "narrative"
}

// Start begins the narrative agent scan loop. It blocks until the context
// is canceled or Stop is called.
func (na *NarrativeAgent) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)

	na.mu.Lock()
	na.cancel = cancel
	na.running = true
	na.mu.Unlock()

	defer func() {
		na.mu.Lock()
		na.running = false
		na.mu.Unlock()
	}()

	ticker := time.NewTicker(na.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-childCtx.Done():
			return nil
		case <-ticker.C:
			na.runScan(childCtx)
		}
	}
}

// Stop cancels the scan loop and marks the agent as not running.
func (na *NarrativeAgent) Stop() error {
	na.mu.Lock()
	defer na.mu.Unlock()
	if na.cancel != nil {
		na.cancel()
	}
	na.running = false
	return nil
}

// Status returns the current status of the narrative agent.
func (na *NarrativeAgent) Status() SubAgentStatus {
	na.mu.RLock()
	defer na.mu.RUnlock()
	return SubAgentStatus{
		Name:       "narrative",
		Running:    na.running,
		LastRun:    na.lastRun,
		RunCount:   na.runCount,
		ErrorCount: na.errorCount,
		LastError:  na.lastError,
	}
}

// runScan fetches current prices for all watchlist symbols, calculates
// percentage changes vs previous scan, formats the data, optionally calls
// the LLM for narrative analysis, and publishes a "narrative" event.
func (na *NarrativeAgent) runScan(ctx context.Context) {
	na.mu.Lock()
	na.runCount++
	na.lastRun = time.Now()
	na.mu.Unlock()

	// Find a connected adapter
	adp := na.findConnectedAdapter()
	if adp == nil {
		na.mu.Lock()
		na.errorCount++
		na.lastError = "no connected adapter available"
		na.mu.Unlock()
		return
	}

	// Fetch current prices for all watchlist symbols
	prices := make(map[string]float64, len(na.cfg.Watchlist))
	for _, symbol := range na.cfg.Watchlist {
		select {
		case <-ctx.Done():
			return
		default:
		}

		price, err := adp.GetPrice(ctx, symbol)
		if err != nil {
			na.mu.Lock()
			na.errorCount++
			na.lastError = fmt.Sprintf("get price %s: %v", symbol, err)
			na.mu.Unlock()
			continue
		}
		prices[symbol] = price.Last
	}

	// Calculate % changes vs previous scan's prices
	na.mu.RLock()
	changes := make(map[string]float64, len(prices))
	for symbol, price := range prices {
		if prev, ok := na.prevPrices[symbol]; ok && prev > 0 {
			changes[symbol] = ((price - prev) / prev) * 100
		}
	}
	na.mu.RUnlock()

	// Format market data
	formatted := na.formatMarketData(prices, changes)

	// Call LLM with narrative analysis prompt (if configured)
	var analysis string
	if na.cfg.LLM != nil {
		resp, err := na.cfg.LLM.Call(ctx, narrativeSystemPrompt, formatted)
		if err != nil {
			na.mu.Lock()
			na.errorCount++
			na.lastError = fmt.Sprintf("LLM call failed: %v", err)
			na.mu.Unlock()
		} else {
			analysis = resp
		}
	} else {
		// Without LLM, use the formatted data as the analysis
		analysis = formatted
	}

	// Store current prices for next comparison
	na.mu.Lock()
	for symbol, price := range prices {
		na.prevPrices[symbol] = price
	}
	na.mu.Unlock()

	// Publish narrative event
	if na.cfg.Bus != nil {
		na.cfg.Bus.Publish(Event{
			Type:   "narrative",
			Source: "narrative",
			Data: map[string]any{
				"analysis": analysis,
				"prices":   prices,
				"changes":  changes,
			},
			Time: time.Now(),
		})
	}
}

// findConnectedAdapter returns the first connected adapter from the config.
func (na *NarrativeAgent) findConnectedAdapter() adapter.TradingAdapter {
	for _, adp := range na.cfg.Adapters {
		if adp.IsConnected() {
			return adp
		}
	}
	return nil
}

// formatMarketData formats prices and percentage changes into a structured
// text representation suitable for LLM analysis.
func (na *NarrativeAgent) formatMarketData(prices map[string]float64, changes map[string]float64) string {
	var b strings.Builder

	b.WriteString("## Current Market Overview\n")

	// Sort symbols for deterministic output
	symbols := make([]string, 0, len(prices))
	for s := range prices {
		symbols = append(symbols, s)
	}
	sort.Strings(symbols)

	for _, symbol := range symbols {
		price := prices[symbol]
		change := changes[symbol] // defaults to 0 if not present

		sign := "+"
		if change < 0 {
			sign = ""
		}

		fmt.Fprintf(&b, "- %s: %s (change: %s%.1f%%)\n",
			symbol, formatPrice(price), sign, change)
	}

	return b.String()
}

// formatPrice formats a float64 price with comma separators and 2 decimal places.
func formatPrice(price float64) string {
	// Format with 2 decimal places
	raw := fmt.Sprintf("%.2f", price)

	// Split into integer and decimal parts
	parts := strings.SplitN(raw, ".", 2)
	intPart := parts[0]
	decPart := parts[1]

	// Add commas to integer part
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	var result strings.Builder
	for i, digit := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteByte(',')
		}
		result.WriteRune(digit)
	}

	formatted := "$" + result.String() + "." + decPart
	if negative {
		formatted = "-" + formatted
	}
	return formatted
}

const narrativeSystemPrompt = `You are a Crypto Narrative Analyst. Analyze the current market data to identify active narratives and their lifecycle phase.

## Your Analysis
1. Identify active narratives from price/volume patterns
2. Rate each narrative's lifecycle phase: forming / growing / peaking / fading / dead
3. Detect sector rotation signals
4. Flag narratives that are peaking (danger) or forming (opportunity)

## Output as JSON
{"narratives":[{"name":"...","phase":"...","tokens":["..."],"evidence":"...","action":"opportunity|caution|avoid"}],"sector_rotation":"...","overall_sentiment":"..."}`
