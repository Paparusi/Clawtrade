package subagent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Episode represents a completed trade episode for reflection analysis.
// Defined locally to avoid circular dependencies with internal/memory.
type Episode struct {
	ID         int64     `json:"id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	PnL        float64   `json:"pnl"`
	Reasoning  string    `json:"reasoning"`
	Strategy   string    `json:"strategy"`
	OpenedAt   time.Time `json:"opened_at"`
	ClosedAt   time.Time `json:"closed_at"`
}

// ReflectionConfig holds configuration for the Reflection sub-agent.
type ReflectionConfig struct {
	LLM          *LLMCaller
	Bus          *EventBus
	ScanInterval time.Duration
}

// ReflectionAgent is a sub-agent that reviews closed trade episodes, calling
// an LLM to provide psychological and strategic analysis. It publishes
// "reflection" events with the analysis results.
type ReflectionAgent struct {
	cfg             ReflectionConfig
	running         bool
	lastRun         time.Time
	runCount        int
	errorCount      int
	lastError       string
	cancel          context.CancelFunc
	lastProcessedID int64
	pendingEpisodes []Episode
	mu              sync.RWMutex
}

// NewReflectionAgent creates a new ReflectionAgent with the given configuration.
func NewReflectionAgent(cfg ReflectionConfig) *ReflectionAgent {
	if cfg.ScanInterval == 0 {
		cfg.ScanInterval = 5 * time.Minute
	}
	return &ReflectionAgent{
		cfg: cfg,
	}
}

// Name returns the sub-agent name.
func (ra *ReflectionAgent) Name() string {
	return "reflection"
}

// Start begins the reflection ticker loop. Each tick processes any pending
// episodes that have been added via AddEpisode.
func (ra *ReflectionAgent) Start(ctx context.Context) error {
	childCtx, cancel := context.WithCancel(ctx)

	ra.mu.Lock()
	ra.cancel = cancel
	ra.running = true
	ra.mu.Unlock()

	defer func() {
		ra.mu.Lock()
		ra.running = false
		ra.mu.Unlock()
	}()

	ticker := time.NewTicker(ra.cfg.ScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-childCtx.Done():
			return nil
		case <-ticker.C:
			ra.processPending(childCtx)
		}
	}
}

// Stop cancels the ticker loop.
func (ra *ReflectionAgent) Stop() error {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	if ra.cancel != nil {
		ra.cancel()
	}
	ra.running = false
	return nil
}

// Status returns the current status of the Reflection sub-agent.
func (ra *ReflectionAgent) Status() SubAgentStatus {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return SubAgentStatus{
		Name:       "reflection",
		Running:    ra.running,
		LastRun:    ra.lastRun,
		RunCount:   ra.runCount,
		ErrorCount: ra.errorCount,
		LastError:  ra.lastError,
	}
}

// AddEpisode adds a completed trade episode to the pending queue for reflection.
// This method is thread-safe.
func (ra *ReflectionAgent) AddEpisode(ep Episode) {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.pendingEpisodes = append(ra.pendingEpisodes, ep)
}

// processPending drains the pending episodes queue and processes each one.
func (ra *ReflectionAgent) processPending(ctx context.Context) {
	ra.mu.Lock()
	episodes := ra.pendingEpisodes
	ra.pendingEpisodes = nil
	ra.mu.Unlock()

	for _, ep := range episodes {
		ra.processEpisode(ctx, ep)
	}
}

// processEpisode calls the LLM with a reflection prompt for the given episode
// and publishes a "reflection" event with the analysis.
func (ra *ReflectionAgent) processEpisode(ctx context.Context, ep Episode) {
	ra.mu.Lock()
	ra.runCount++
	ra.lastRun = time.Now()
	ra.mu.Unlock()

	if ra.cfg.LLM == nil {
		ra.mu.Lock()
		ra.errorCount++
		ra.lastError = "no LLM caller configured"
		ra.mu.Unlock()
		return
	}

	// Gather recent episodes for context (none currently in queue since we drained)
	prompt := ra.buildReflectionPrompt(ep, nil)

	response, err := ra.cfg.LLM.Call(ctx, prompt, fmt.Sprintf("Analyze this %s %s trade on %s", ep.Side, ep.Strategy, ep.Symbol))
	if err != nil {
		ra.mu.Lock()
		ra.errorCount++
		ra.lastError = fmt.Sprintf("LLM call failed: %v", err)
		ra.mu.Unlock()
		return
	}

	if ep.ID > ra.lastProcessedID {
		ra.mu.Lock()
		ra.lastProcessedID = ep.ID
		ra.mu.Unlock()
	}

	if ra.cfg.Bus != nil {
		ra.cfg.Bus.Publish(Event{
			Type:   "reflection",
			Source: "reflection",
			Symbol: ep.Symbol,
			Data: map[string]any{
				"analysis":    response,
				"episode_id":  ep.ID,
				"pnl":         ep.PnL,
				"side":        ep.Side,
				"entry_price": ep.EntryPrice,
				"exit_price":  ep.ExitPrice,
			},
			Time: time.Now(),
		})
	}
}

// buildReflectionPrompt constructs the LLM prompt for reflecting on a trade episode.
func (ra *ReflectionAgent) buildReflectionPrompt(ep Episode, recentEpisodes []Episode) string {
	var b strings.Builder

	b.WriteString("You are a Trading Psychologist and Performance Analyst.\n\n")

	b.WriteString("## The Trade\n")
	fmt.Fprintf(&b, "Symbol: %s\n", ep.Symbol)
	fmt.Fprintf(&b, "Side: %s\n", ep.Side)
	fmt.Fprintf(&b, "Entry: $%.2f\n", ep.EntryPrice)
	fmt.Fprintf(&b, "Exit: $%.2f\n", ep.ExitPrice)
	fmt.Fprintf(&b, "PnL: $%.2f\n", ep.PnL)
	fmt.Fprintf(&b, "Strategy: %s\n", ep.Strategy)
	fmt.Fprintf(&b, "Original reasoning: %s\n", ep.Reasoning)

	b.WriteString("\n## Recent Trade History\n")
	if len(recentEpisodes) == 0 {
		b.WriteString("No recent trades available.\n")
	} else {
		for _, r := range recentEpisodes {
			fmt.Fprintf(&b, "- %s %s PnL: $%.2f\n", r.Symbol, r.Side, r.PnL)
		}
	}

	b.WriteString(`
## Your Analysis
1. Was the entry thesis correct? Why or why not?
2. Was the exit optimal?
3. Did the trader follow their own rules?
4. Detect behavioral patterns:
   - Revenge trading (loss -> immediate re-entry)
   - FOMO (entered after move already happened)
   - Overtrading (too many trades in short period)
   - Emotional exits (panic sell / greed hold)
5. Strategy effectiveness: is this strategy working?
6. Generate a learned rule if pattern confidence > 70%

## Output as JSON
{"grade":"A|B|C|D|F", "lesson":"...", "behavioral_flags":["..."], "rule_suggestion":{"content":"...", "confidence":0.85}, "strategy_adjustment":"..."}`)

	return b.String()
}
