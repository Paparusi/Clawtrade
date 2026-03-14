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

func (m *mockAdapter) Name() string                  { return m.name }
func (m *mockAdapter) IsConnected() bool             { return m.connected }
func (m *mockAdapter) Capabilities() adapter.AdapterCaps { return adapter.AdapterCaps{} }
func (m *mockAdapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
	if p, ok := m.prices[symbol]; ok {
		return p, nil
	}
	return &adapter.Price{Symbol: symbol, Last: 64000}, nil
}
func (m *mockAdapter) GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	return nil, nil
}
func (m *mockAdapter) GetOrderBook(ctx context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	return nil, nil
}
func (m *mockAdapter) PlaceOrder(ctx context.Context, order adapter.Order) (*adapter.Order, error) {
	return nil, nil
}
func (m *mockAdapter) CancelOrder(ctx context.Context, orderID string) error { return nil }
func (m *mockAdapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error) {
	return nil, nil
}
func (m *mockAdapter) GetBalances(ctx context.Context) ([]adapter.Balance, error) {
	return nil, nil
}
func (m *mockAdapter) GetPositions(ctx context.Context) ([]adapter.Position, error) {
	return nil, nil
}
func (m *mockAdapter) Connect(ctx context.Context) error { return nil }
func (m *mockAdapter) Disconnect() error                 { return nil }

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
