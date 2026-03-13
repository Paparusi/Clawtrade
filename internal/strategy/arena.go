package strategy

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// Signal represents a trading signal from a strategy.
type Signal struct {
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"`      // "buy" or "sell"
	Strength  float64   `json:"strength"`  // 0-1
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

// StrategyFunc is a function that produces signals from market data.
type StrategyFunc func(symbol string, prices []float64) *Signal

// StrategyEntry tracks a strategy and its performance.
type StrategyEntry struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Strategy    StrategyFunc `json:"-"`
	Stats       StrategyStats `json:"stats"`
	Active      bool         `json:"active"`
	CreatedAt   time.Time    `json:"created_at"`
}

// StrategyStats tracks strategy performance metrics.
type StrategyStats struct {
	TotalSignals int     `json:"total_signals"`
	WinCount     int     `json:"win_count"`
	LossCount    int     `json:"loss_count"`
	TotalPnL     float64 `json:"total_pnl"`
	MaxDrawdown  float64 `json:"max_drawdown"`
	SharpeRatio  float64 `json:"sharpe_ratio"`
	WinRate      float64 `json:"win_rate"`
	AvgWin       float64 `json:"avg_win"`
	AvgLoss      float64 `json:"avg_loss"`
	ProfitFactor float64 `json:"profit_factor"` // gross profit / gross loss

	// internal tracking fields (not exported to JSON)
	grossProfit float64
	grossLoss   float64
	peakPnL     float64
	returns     []float64
}

// Comparison holds a head-to-head comparison of two strategies.
type Comparison struct {
	Strategy1 string        `json:"strategy1"`
	Strategy2 string        `json:"strategy2"`
	Stats1    StrategyStats `json:"stats1"`
	Stats2    StrategyStats `json:"stats2"`
	Winner    string        `json:"winner"`  // based on PnL
	Verdict   string        `json:"verdict"` // human-readable summary
}

// Arena manages multiple strategies and compares their performance.
type Arena struct {
	mu         sync.RWMutex
	strategies map[string]*StrategyEntry
}

// NewArena creates a new strategy arena.
func NewArena() *Arena {
	return &Arena{
		strategies: make(map[string]*StrategyEntry),
	}
}

// Register adds a strategy to the arena.
func (a *Arena) Register(name, description string, strategy StrategyFunc) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.strategies[name]; exists {
		return fmt.Errorf("strategy %q already registered", name)
	}

	a.strategies[name] = &StrategyEntry{
		Name:        name,
		Description: description,
		Strategy:    strategy,
		Active:      true,
		CreatedAt:   time.Now(),
	}
	return nil
}

// Unregister removes a strategy from the arena.
func (a *Arena) Unregister(name string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.strategies[name]; !exists {
		return fmt.Errorf("strategy %q not found", name)
	}

	delete(a.strategies, name)
	return nil
}

// SetActive activates or deactivates a strategy.
func (a *Arena) SetActive(name string, active bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, exists := a.strategies[name]
	if !exists {
		return fmt.Errorf("strategy %q not found", name)
	}

	entry.Active = active
	return nil
}

// RunSignal generates signals from all active strategies for given data.
func (a *Arena) RunSignal(symbol string, prices []float64) map[string]*Signal {
	a.mu.RLock()
	defer a.mu.RUnlock()

	results := make(map[string]*Signal)
	for name, entry := range a.strategies {
		if !entry.Active {
			continue
		}
		if sig := entry.Strategy(symbol, prices); sig != nil {
			results[name] = sig
		}
	}
	return results
}

// RecordResult records the outcome of a signal for a strategy.
func (a *Arena) RecordResult(strategyName string, pnl float64) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	entry, exists := a.strategies[strategyName]
	if !exists {
		return fmt.Errorf("strategy %q not found", strategyName)
	}

	s := &entry.Stats
	s.TotalSignals++
	s.TotalPnL += pnl
	s.returns = append(s.returns, pnl)

	if pnl >= 0 {
		s.WinCount++
		s.grossProfit += pnl
	} else {
		s.LossCount++
		s.grossLoss += math.Abs(pnl)
	}

	total := s.WinCount + s.LossCount
	if total > 0 {
		s.WinRate = float64(s.WinCount) / float64(total)
	}

	if s.WinCount > 0 {
		s.AvgWin = s.grossProfit / float64(s.WinCount)
	}
	if s.LossCount > 0 {
		s.AvgLoss = s.grossLoss / float64(s.LossCount)
	}

	if s.grossLoss > 0 {
		s.ProfitFactor = s.grossProfit / s.grossLoss
	}

	// Track max drawdown
	if s.TotalPnL > s.peakPnL {
		s.peakPnL = s.TotalPnL
	}
	drawdown := s.peakPnL - s.TotalPnL
	if drawdown > s.MaxDrawdown {
		s.MaxDrawdown = drawdown
	}

	// Update Sharpe ratio (using returns, assuming risk-free rate = 0)
	s.SharpeRatio = computeSharpe(s.returns)

	return nil
}

// computeSharpe calculates an annualised Sharpe ratio from a slice of returns.
func computeSharpe(returns []float64) float64 {
	n := len(returns)
	if n < 2 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(n)

	var variance float64
	for _, r := range returns {
		diff := r - mean
		variance += diff * diff
	}
	variance /= float64(n - 1)

	stddev := math.Sqrt(variance)
	if stddev == 0 {
		return 0
	}

	return mean / stddev
}

// GetRanking returns strategies sorted by the given metric (descending).
// Supported metrics: "win_rate", "pnl", "sharpe", "profit_factor".
func (a *Arena) GetRanking(metric string) []StrategyEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := make([]StrategyEntry, 0, len(a.strategies))
	for _, e := range a.strategies {
		entries = append(entries, *e)
	}

	sort.Slice(entries, func(i, j int) bool {
		switch metric {
		case "win_rate":
			return entries[i].Stats.WinRate > entries[j].Stats.WinRate
		case "pnl":
			return entries[i].Stats.TotalPnL > entries[j].Stats.TotalPnL
		case "sharpe":
			return entries[i].Stats.SharpeRatio > entries[j].Stats.SharpeRatio
		case "profit_factor":
			return entries[i].Stats.ProfitFactor > entries[j].Stats.ProfitFactor
		default:
			return entries[i].Stats.TotalPnL > entries[j].Stats.TotalPnL
		}
	})

	return entries
}

// GetStats returns stats for a specific strategy.
func (a *Arena) GetStats(name string) (*StrategyStats, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entry, exists := a.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy %q not found", name)
	}

	stats := entry.Stats
	return &stats, nil
}

// Compare returns a head-to-head comparison of two strategies.
func (a *Arena) Compare(name1, name2 string) (*Comparison, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	e1, ok1 := a.strategies[name1]
	if !ok1 {
		return nil, fmt.Errorf("strategy %q not found", name1)
	}
	e2, ok2 := a.strategies[name2]
	if !ok2 {
		return nil, fmt.Errorf("strategy %q not found", name2)
	}

	winner := name1
	if e2.Stats.TotalPnL > e1.Stats.TotalPnL {
		winner = name2
	}

	verdict := fmt.Sprintf(
		"%s outperforms %s with PnL %.2f vs %.2f (win rates: %.1f%% vs %.1f%%)",
		winner,
		otherName(winner, name1, name2),
		a.strategies[winner].Stats.TotalPnL,
		a.strategies[otherName(winner, name1, name2)].Stats.TotalPnL,
		a.strategies[winner].Stats.WinRate*100,
		a.strategies[otherName(winner, name1, name2)].Stats.WinRate*100,
	)

	return &Comparison{
		Strategy1: name1,
		Strategy2: name2,
		Stats1:    e1.Stats,
		Stats2:    e2.Stats,
		Winner:    winner,
		Verdict:   verdict,
	}, nil
}

func otherName(winner, name1, name2 string) string {
	if winner == name1 {
		return name2
	}
	return name1
}

// ListStrategies returns all registered strategies.
func (a *Arena) ListStrategies() []StrategyEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	entries := make([]StrategyEntry, 0, len(a.strategies))
	for _, e := range a.strategies {
		entries = append(entries, *e)
	}
	return entries
}
