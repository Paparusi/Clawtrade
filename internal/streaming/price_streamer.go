package streaming

import (
	"context"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/engine"
)

// PriceStreamerConfig holds configuration for the PriceStreamer.
type PriceStreamerConfig struct {
	Adapters     map[string]adapter.TradingAdapter
	Bus          *engine.EventBus
	Symbols      []string
	PollInterval time.Duration
}

// PriceStreamer polls connected adapters for price data and publishes
// price update events to the engine event bus.
type PriceStreamer struct {
	config     PriceStreamerConfig
	prevPrices map[string]float64
	mu         sync.RWMutex
}

// NewPriceStreamer creates a new PriceStreamer with the given configuration.
func NewPriceStreamer(cfg PriceStreamerConfig) *PriceStreamer {
	if cfg.PollInterval == 0 {
		cfg.PollInterval = time.Second
	}
	return &PriceStreamer{
		config:     cfg,
		prevPrices: make(map[string]float64),
	}
}

// Start begins polling all connected adapters for price updates.
// It blocks until the context is cancelled.
func (ps *PriceStreamer) Start(ctx context.Context) {
	var wg sync.WaitGroup
	for _, adp := range ps.config.Adapters {
		if !adp.IsConnected() {
			continue
		}
		wg.Add(1)
		go func(a adapter.TradingAdapter) {
			defer wg.Done()
			ps.pollLoop(ctx, a, ps.config.Symbols)
		}(adp)
	}
	wg.Wait()
}

// pollLoop polls an adapter for prices at the configured interval.
func (ps *PriceStreamer) pollLoop(ctx context.Context, adp adapter.TradingAdapter, symbols []string) {
	ticker := time.NewTicker(ps.config.PollInterval)
	defer ticker.Stop()

	// Poll immediately on start.
	ps.pollOnce(ctx, adp, symbols)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ps.pollOnce(ctx, adp, symbols)
		}
	}
}

func (ps *PriceStreamer) pollOnce(ctx context.Context, adp adapter.TradingAdapter, symbols []string) {
	for _, sym := range symbols {
		price, err := adp.GetPrice(ctx, sym)
		if err != nil {
			continue
		}
		ps.publishPrice(sym, price)
	}
}

// publishPrice creates a price.update event and publishes it to the bus.
func (ps *PriceStreamer) publishPrice(symbol string, price *adapter.Price) {
	changePct := ps.calcChangePct(symbol, price.Last)

	ev := engine.Event{
		Type: "price.update",
		Data: map[string]any{
			"symbol":     symbol,
			"last":       price.Last,
			"bid":        price.Bid,
			"ask":        price.Ask,
			"volume_24h": price.Volume24h,
			"change_pct": changePct,
		},
	}
	ps.config.Bus.Publish(ev)
}

// calcChangePct computes the percentage change from the previous price
// and stores the current price for the next calculation.
func (ps *PriceStreamer) calcChangePct(symbol string, currentPrice float64) float64 {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	prev, ok := ps.prevPrices[symbol]
	ps.prevPrices[symbol] = currentPrice

	if !ok || prev == 0 {
		return 0
	}
	return (currentPrice - prev) / prev * 100
}
