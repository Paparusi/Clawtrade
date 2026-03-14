package bybit

import (
	"testing"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

// Compile-time interface compliance check.
var _ adapter.TradingAdapter = (*Adapter)(nil)

func TestNew(t *testing.T) {
	a := New("key", "secret")
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestName(t *testing.T) {
	a := New("key", "secret")
	if a.Name() != "bybit" {
		t.Fatalf("expected name %q, got %q", "bybit", a.Name())
	}
}

func TestCapabilities(t *testing.T) {
	a := New("key", "secret")
	caps := a.Capabilities()

	if caps.Name != "bybit" {
		t.Fatalf("expected caps name %q, got %q", "bybit", caps.Name)
	}
	if !caps.WebSocket {
		t.Fatal("expected WebSocket to be true")
	}
	if !caps.Margin {
		t.Fatal("expected Margin to be true")
	}
	if !caps.Futures {
		t.Fatal("expected Futures to be true")
	}
	if len(caps.OrderTypes) != 3 {
		t.Fatalf("expected 3 order types, got %d", len(caps.OrderTypes))
	}
}

func TestToBybitSymbol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BTC/USDT", "BTCUSDT"},
		{"ETH/USDT", "ETHUSDT"},
		{"btc/usdt", "BTCUSDT"},
		{"BTCUSDT", "BTCUSDT"},
		{"SOL/USDC", "SOLUSDC"},
	}
	for _, tt := range tests {
		got := toBybitSymbol(tt.input)
		if got != tt.want {
			t.Errorf("toBybitSymbol(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFromBybitSymbol(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BTCUSDT", "BTC/USDT"},
		{"ETHUSDT", "ETH/USDT"},
		{"SOLUSDC", "SOL/USDC"},
		{"ETHBTC", "ETH/BTC"},
	}
	for _, tt := range tests {
		got := fromBybitSymbol(tt.input)
		if got != tt.want {
			t.Errorf("fromBybitSymbol(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapTimeframe(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"1m", "1"},
		{"5m", "5"},
		{"15m", "15"},
		{"30m", "30"},
		{"1h", "60"},
		{"4h", "240"},
		{"1d", "D"},
		{"1w", "W"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := mapTimeframe(tt.input)
		if got != tt.want {
			t.Errorf("mapTimeframe(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapOrderStatus(t *testing.T) {
	tests := []struct {
		input string
		want  adapter.OrderStatus
	}{
		{"New", adapter.OrderStatusPending},
		{"PartiallyFilled", adapter.OrderStatusPending},
		{"Filled", adapter.OrderStatusFilled},
		{"Cancelled", adapter.OrderStatusCanceled},
		{"Rejected", adapter.OrderStatusCanceled},
		{"Deactivated", adapter.OrderStatusCanceled},
	}
	for _, tt := range tests {
		got := mapOrderStatus(tt.input)
		if got != tt.want {
			t.Errorf("mapOrderStatus(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMapSide(t *testing.T) {
	if got := mapSideToBybit(adapter.SideBuy); got != "Buy" {
		t.Errorf("mapSideToBybit(SideBuy) = %q, want %q", got, "Buy")
	}
	if got := mapSideToBybit(adapter.SideSell); got != "Sell" {
		t.Errorf("mapSideToBybit(SideSell) = %q, want %q", got, "Sell")
	}
	if got := mapSideFromBybit("Buy"); got != adapter.SideBuy {
		t.Errorf("mapSideFromBybit(Buy) = %q, want %q", got, adapter.SideBuy)
	}
	if got := mapSideFromBybit("Sell"); got != adapter.SideSell {
		t.Errorf("mapSideFromBybit(Sell) = %q, want %q", got, adapter.SideSell)
	}
}

func TestConnectDisconnect(t *testing.T) {
	a := New("key", "secret")

	if a.IsConnected() {
		t.Fatal("expected not connected initially")
	}

	if err := a.Connect(nil); err != nil {
		t.Fatalf("connect error: %v", err)
	}
	if !a.IsConnected() {
		t.Fatal("expected connected after Connect")
	}

	if err := a.Disconnect(); err != nil {
		t.Fatalf("disconnect error: %v", err)
	}
	if a.IsConnected() {
		t.Fatal("expected not connected after Disconnect")
	}
}

func TestSetTestnet(t *testing.T) {
	a := New("key", "secret")
	a.SetTestnet(true)
	if a.baseURL() != "https://api-testnet.bybit.com" {
		t.Fatalf("expected testnet URL, got %q", a.baseURL())
	}
	a.SetTestnet(false)
	if a.baseURL() != "https://api.bybit.com" {
		t.Fatalf("expected mainnet URL, got %q", a.baseURL())
	}
}

func TestGetCachedPrice(t *testing.T) {
	a := New("key", "secret")

	_, ok := a.GetCachedPrice("BTC/USDT")
	if ok {
		t.Fatal("expected no cached price initially")
	}

	a.mu.Lock()
	a.prices["BTC/USDT"] = adapter.Price{Symbol: "BTC/USDT", Last: 50000}
	a.mu.Unlock()

	p, ok := a.GetCachedPrice("BTC/USDT")
	if !ok {
		t.Fatal("expected cached price after setting")
	}
	if p.Last != 50000 {
		t.Fatalf("expected last price 50000, got %f", p.Last)
	}
}
