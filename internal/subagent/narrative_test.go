package subagent

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

// mockTradingAdapter implements adapter.TradingAdapter with configurable prices for narrative tests.
type mockTradingAdapter struct {
	connected bool
	prices    map[string]*adapter.Price
	mu        sync.Mutex
}

func (m *mockTradingAdapter) Name() string                    { return "mock" }
func (m *mockTradingAdapter) Capabilities() adapter.AdapterCaps { return adapter.AdapterCaps{Name: "mock"} }
func (m *mockTradingAdapter) GetPrice(_ context.Context, symbol string) (*adapter.Price, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.prices[symbol]; ok {
		return p, nil
	}
	return &adapter.Price{Symbol: symbol, Last: 0}, nil
}
func (m *mockTradingAdapter) GetCandles(_ context.Context, _ string, _ string, _ int) ([]adapter.Candle, error) {
	return nil, nil
}
func (m *mockTradingAdapter) GetOrderBook(_ context.Context, _ string, _ int) (*adapter.OrderBook, error) {
	return nil, nil
}
func (m *mockTradingAdapter) PlaceOrder(_ context.Context, order adapter.Order) (*adapter.Order, error) {
	return &order, nil
}
func (m *mockTradingAdapter) CancelOrder(_ context.Context, _ string) error     { return nil }
func (m *mockTradingAdapter) GetOpenOrders(_ context.Context) ([]adapter.Order, error) { return nil, nil }
func (m *mockTradingAdapter) GetBalances(_ context.Context) ([]adapter.Balance, error) { return nil, nil }
func (m *mockTradingAdapter) GetPositions(_ context.Context) ([]adapter.Position, error) { return nil, nil }
func (m *mockTradingAdapter) Connect(_ context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = true
	return nil
}
func (m *mockTradingAdapter) Disconnect() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connected = false
	return nil
}
func (m *mockTradingAdapter) IsConnected() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.connected
}

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
	if !strings.Contains(text, "ETH/USDT") {
		t.Error("should contain ETH/USDT symbol")
	}
	if !strings.Contains(text, "Current Market Overview") {
		t.Error("should contain header")
	}
	if !strings.Contains(text, "+2.5%") {
		t.Error("should contain positive change with + prefix")
	}
	if !strings.Contains(text, "-1.2%") {
		t.Error("should contain negative change")
	}
	if !strings.Contains(text, "$64,000.00") {
		t.Error("should contain formatted BTC price")
	}
	if !strings.Contains(text, "$3,400.00") {
		t.Error("should contain formatted ETH price")
	}
}

func TestNarrativeAgent_FormatMarketData_NoChanges(t *testing.T) {
	na := NewNarrativeAgent(NarrativeConfig{})
	prices := map[string]float64{"SOL/USDT": 150.5}
	changes := map[string]float64{}
	text := na.formatMarketData(prices, changes)
	if !strings.Contains(text, "SOL/USDT") {
		t.Error("should contain SOL/USDT")
	}
	if !strings.Contains(text, "+0.0%") {
		t.Error("should show 0% change when no previous data")
	}
}

func TestNarrativeAgent_Status(t *testing.T) {
	na := NewNarrativeAgent(NarrativeConfig{})
	status := na.Status()
	if status.Name != "narrative" {
		t.Errorf("expected name 'narrative', got %q", status.Name)
	}
	if status.Running {
		t.Error("should not be running initially")
	}
	if status.RunCount != 0 {
		t.Error("run count should be 0 initially")
	}
}

func TestNarrativeAgent_StartStop(t *testing.T) {
	na := NewNarrativeAgent(NarrativeConfig{
		ScanInterval: 50 * time.Millisecond,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- na.Start(ctx)
	}()

	time.Sleep(20 * time.Millisecond)

	status := na.Status()
	if !status.Running {
		t.Error("should be running after Start")
	}

	if err := na.Stop(); err != nil {
		t.Fatalf("Stop() error: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Start returned error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Stop")
	}

	status = na.Status()
	if status.Running {
		t.Error("should not be running after Stop")
	}
}

func TestNarrativeAgent_RunScan_PublishesEvent(t *testing.T) {
	bus := NewEventBus()
	ch := bus.Subscribe("narrative")

	mockAdapter := &mockTradingAdapter{
		connected: true,
		prices: map[string]*adapter.Price{
			"BTC/USDT": {Symbol: "BTC/USDT", Last: 64000},
			"ETH/USDT": {Symbol: "ETH/USDT", Last: 3400},
		},
	}

	na := NewNarrativeAgent(NarrativeConfig{
		Bus:          bus,
		Adapters:     map[string]adapter.TradingAdapter{"mock": mockAdapter},
		Watchlist:    []string{"BTC/USDT", "ETH/USDT"},
		ScanInterval: time.Minute,
	})

	ctx := context.Background()
	na.runScan(ctx)

	select {
	case ev := <-ch:
		if ev.Type != "narrative" {
			t.Errorf("expected event type 'narrative', got %q", ev.Type)
		}
		if ev.Source != "narrative" {
			t.Errorf("expected source 'narrative', got %q", ev.Source)
		}
		if _, ok := ev.Data["analysis"]; !ok {
			t.Error("event data should contain 'analysis' key")
		}
	case <-time.After(time.Second):
		t.Fatal("did not receive narrative event")
	}
}

func TestNarrativeAgent_RunScan_TracksChanges(t *testing.T) {
	bus := NewEventBus()
	_ = bus.Subscribe("narrative")

	mockAdapter := &mockTradingAdapter{
		connected: true,
		prices: map[string]*adapter.Price{
			"BTC/USDT": {Symbol: "BTC/USDT", Last: 60000},
		},
	}

	na := NewNarrativeAgent(NarrativeConfig{
		Bus:          bus,
		Adapters:     map[string]adapter.TradingAdapter{"mock": mockAdapter},
		Watchlist:    []string{"BTC/USDT"},
		ScanInterval: time.Minute,
	})

	ctx := context.Background()

	// First scan establishes baseline
	na.runScan(ctx)

	// Update price
	mockAdapter.prices["BTC/USDT"] = &adapter.Price{Symbol: "BTC/USDT", Last: 63000}

	// Second scan should detect the change
	na.runScan(ctx)

	status := na.Status()
	if status.RunCount != 2 {
		t.Errorf("expected 2 runs, got %d", status.RunCount)
	}
}

func TestNarrativeAgent_RunScan_NoAdapter(t *testing.T) {
	na := NewNarrativeAgent(NarrativeConfig{
		Watchlist: []string{"BTC/USDT"},
	})

	ctx := context.Background()
	na.runScan(ctx)

	status := na.Status()
	if status.ErrorCount == 0 {
		t.Error("expected errors when no adapter available")
	}
}

func TestNarrativeAgent_DefaultScanInterval(t *testing.T) {
	na := NewNarrativeAgent(NarrativeConfig{})
	if na.cfg.ScanInterval != 15*time.Minute {
		t.Errorf("expected default scan interval of 15m, got %v", na.cfg.ScanInterval)
	}
}
