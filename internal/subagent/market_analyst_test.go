package subagent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

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
			"1h": {{Open: 64000, High: 64500, Low: 63800, Close: 64200, Volume: 100, Timestamp: time.Date(2026, 3, 14, 0, 0, 0, 0, time.UTC)}},
		},
		Price: &adapter.Price{
			Symbol:    "BTC/USDT",
			Last:      64720.50,
			Bid:       64710,
			Ask:       64730,
			Volume24h: 12345.67,
		},
		OrderBook: &adapter.OrderBook{
			Symbol: "BTC/USDT",
			Bids: []adapter.OrderBookEntry{
				{Price: 64700, Amount: 12.5},
				{Price: 64690, Amount: 8.3},
			},
			Asks: []adapter.OrderBookEntry{
				{Price: 64710, Amount: 5.2},
				{Price: 64720, Amount: 15.8},
			},
		},
		Correlated: map[string]*adapter.Price{
			"ETH/USDT": {Symbol: "ETH/USDT", Last: 3450},
		},
	}
	text := ma.formatForLLM(snap)
	if text == "" {
		t.Error("expected non-empty formatted text")
	}
	if !strings.Contains(text, "BTC/USDT") {
		t.Error("should contain symbol")
	}
	if !strings.Contains(text, "64720.50") {
		t.Error("should contain current price")
	}
	if !strings.Contains(text, "1h Candles") {
		t.Error("should contain timeframe header")
	}
	if !strings.Contains(text, "64700.00") {
		t.Error("should contain orderbook bid price")
	}
	if !strings.Contains(text, "ETH/USDT") {
		t.Error("should contain correlated asset")
	}
}

func TestMarketAnalyst_Status(t *testing.T) {
	ma := NewMarketAnalyst(MarketAnalystConfig{})
	status := ma.Status()
	if status.Running {
		t.Error("should not be running before start")
	}
	if status.Name != "market-analyst" {
		t.Errorf("expected status name 'market-analyst', got %q", status.Name)
	}
}

func TestMarketAnalyst_StartStop(t *testing.T) {
	bus := NewEventBus()
	ma := NewMarketAnalyst(MarketAnalystConfig{
		ScanInterval: 50 * time.Millisecond,
		Bus:          bus,
		Watchlist:    []string{},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- ma.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(20 * time.Millisecond)

	status := ma.Status()
	if !status.Running {
		t.Error("should be running after start")
	}

	if err := ma.Stop(); err != nil {
		t.Errorf("unexpected error from Stop: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Start did not return after Stop")
	}

	status = ma.Status()
	if status.Running {
		t.Error("should not be running after stop")
	}
}

// mockAdapter implements adapter.TradingAdapter for testing.
type mockAdapter struct {
	connected bool
	mu        sync.Mutex
}

func (m *mockAdapter) Name() string                    { return "mock" }
func (m *mockAdapter) Capabilities() adapter.AdapterCaps { return adapter.AdapterCaps{Name: "mock"} }
func (m *mockAdapter) GetPrice(_ context.Context, symbol string) (*adapter.Price, error) {
	return &adapter.Price{Symbol: symbol, Last: 64720.50, Bid: 64710, Ask: 64730, Volume24h: 12345.67, Timestamp: time.Now()}, nil
}
func (m *mockAdapter) GetCandles(_ context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	candles := make([]adapter.Candle, 0, limit)
	for i := 0; i < limit; i++ {
		candles = append(candles, adapter.Candle{
			Open: 64000, High: 64500, Low: 63800, Close: 64200, Volume: 100,
			Timestamp: time.Now().Add(-time.Duration(i) * time.Hour),
		})
	}
	return candles, nil
}
func (m *mockAdapter) GetOrderBook(_ context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	return &adapter.OrderBook{
		Symbol: symbol,
		Bids:   []adapter.OrderBookEntry{{Price: 64700, Amount: 12.5}},
		Asks:   []adapter.OrderBookEntry{{Price: 64710, Amount: 5.2}},
	}, nil
}
func (m *mockAdapter) PlaceOrder(_ context.Context, order adapter.Order) (*adapter.Order, error) {
	return &order, nil
}
func (m *mockAdapter) CancelOrder(_ context.Context, _ string) error     { return nil }
func (m *mockAdapter) GetOpenOrders(_ context.Context) ([]adapter.Order, error) { return nil, nil }
func (m *mockAdapter) GetBalances(_ context.Context) ([]adapter.Balance, error) { return nil, nil }
func (m *mockAdapter) GetPositions(_ context.Context) ([]adapter.Position, error) { return nil, nil }
func (m *mockAdapter) Connect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}
func (m *mockAdapter) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}
func (m *mockAdapter) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

func TestMarketAnalyst_CollectSnapshot(t *testing.T) {
	mock := &mockAdapter{connected: true}
	ma := NewMarketAnalyst(MarketAnalystConfig{
		Timeframes: []string{"1h", "4h"},
		Adapters:   map[string]adapter.TradingAdapter{"mock": mock},
	})

	snap, err := ma.collectSnapshot(context.Background(), "BTC/USDT")
	if err != nil {
		t.Fatalf("collectSnapshot error: %v", err)
	}
	if snap.Symbol != "BTC/USDT" {
		t.Errorf("expected symbol BTC/USDT, got %q", snap.Symbol)
	}
	if snap.Price == nil {
		t.Fatal("expected price to be set")
	}
	if snap.OrderBook == nil {
		t.Fatal("expected orderbook to be set")
	}
	if len(snap.Candles) != 2 {
		t.Errorf("expected 2 timeframes of candles, got %d", len(snap.Candles))
	}
	if _, ok := snap.Candles["1h"]; !ok {
		t.Error("expected 1h candles")
	}
	if _, ok := snap.Candles["4h"]; !ok {
		t.Error("expected 4h candles")
	}
}

func TestMarketAnalyst_RunExperts(t *testing.T) {
	ma := NewMarketAnalyst(MarketAnalystConfig{
		Strategies: []Strategy{
			{Name: "Trend Follower", Slug: "trend", Prompt: "Analyze the trend."},
			{Name: "Mean Reversion", Slug: "mean-reversion", Prompt: "Analyze mean reversion."},
		},
		ActiveStrategies: []string{"trend"},
		ExpertCaller: &LLMCaller{
			Provider: "test",
		},
	})

	// runExperts will fail on LLM call since provider "test" is unsupported,
	// but we can verify the structure. We test with nil caller handled gracefully.
	results := ma.runExperts(context.Background(), "some formatted data")
	// With unsupported provider, all calls fail, so results should be empty or contain errors
	// The important thing is it doesn't panic
	_ = results
}

func TestMarketAnalyst_ExpertResult(t *testing.T) {
	er := ExpertResult{
		Strategy: "trend",
		Response: "bullish signal detected",
	}
	if er.Strategy != "trend" {
		t.Errorf("expected strategy 'trend', got %q", er.Strategy)
	}
	if er.Response != "bullish signal detected" {
		t.Errorf("unexpected response: %q", er.Response)
	}
}

func TestMarketAnalyst_FormatSnapshot_EmptyCandles(t *testing.T) {
	ma := NewMarketAnalyst(MarketAnalystConfig{})
	snap := &MarketSnapshot{
		Symbol:  "ETH/USDT",
		Candles: map[string][]adapter.Candle{},
		Price: &adapter.Price{
			Symbol: "ETH/USDT",
			Last:   3450.00,
			Bid:    3449,
			Ask:    3451,
		},
	}
	text := ma.formatForLLM(snap)
	if !strings.Contains(text, "ETH/USDT") {
		t.Error("should contain symbol even with empty candles")
	}
	if !strings.Contains(text, "3450.00") {
		t.Error("should contain price")
	}
}
