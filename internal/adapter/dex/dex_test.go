package dex

import (
	"context"
	"errors"
	"testing"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

// Compile-time interface checks.
var (
	_ adapter.TradingAdapter = (*UniswapAdapter)(nil)
	_ adapter.TradingAdapter = (*JupiterAdapter)(nil)
	_ adapter.TradingAdapter = (*HyperliquidAdapter)(nil)
)

func TestNewUniswap(t *testing.T) {
	a := NewUniswap("https://mainnet.infura.io/v3/key", 1)
	if a == nil {
		t.Fatal("NewUniswap returned nil")
	}
	if a.rpcURL != "https://mainnet.infura.io/v3/key" {
		t.Errorf("rpcURL = %q, want %q", a.rpcURL, "https://mainnet.infura.io/v3/key")
	}
	if a.chainID != 1 {
		t.Errorf("chainID = %d, want 1", a.chainID)
	}
}

func TestNewJupiter(t *testing.T) {
	a := NewJupiter("https://api.mainnet-beta.solana.com")
	if a == nil {
		t.Fatal("NewJupiter returned nil")
	}
	if a.rpcURL != "https://api.mainnet-beta.solana.com" {
		t.Errorf("rpcURL = %q, want %q", a.rpcURL, "https://api.mainnet-beta.solana.com")
	}
}

func TestNewHyperliquid(t *testing.T) {
	a := NewHyperliquid("https://api.hyperliquid.xyz")
	if a == nil {
		t.Fatal("NewHyperliquid returned nil")
	}
	if a.apiURL != "https://api.hyperliquid.xyz" {
		t.Errorf("apiURL = %q, want %q", a.apiURL, "https://api.hyperliquid.xyz")
	}
}

func TestUniswapName(t *testing.T) {
	a := NewUniswap("http://localhost:8545", 1)
	if got := a.Name(); got != "uniswap" {
		t.Errorf("Name() = %q, want %q", got, "uniswap")
	}
}

func TestJupiterName(t *testing.T) {
	a := NewJupiter("http://localhost:8899")
	if got := a.Name(); got != "jupiter" {
		t.Errorf("Name() = %q, want %q", got, "jupiter")
	}
}

func TestHyperliquidName(t *testing.T) {
	a := NewHyperliquid("https://api.hyperliquid.xyz")
	if got := a.Name(); got != "hyperliquid" {
		t.Errorf("Name() = %q, want %q", got, "hyperliquid")
	}
}

func TestUniswapCapabilities(t *testing.T) {
	a := NewUniswap("http://localhost:8545", 1)
	caps := a.Capabilities()
	if caps.Name != "Uniswap" {
		t.Errorf("caps.Name = %q, want %q", caps.Name, "Uniswap")
	}
	if caps.WebSocket {
		t.Error("caps.WebSocket = true, want false")
	}
	if caps.Margin {
		t.Error("caps.Margin = true, want false")
	}
	if caps.Futures {
		t.Error("caps.Futures = true, want false")
	}
	if len(caps.OrderTypes) != 1 || caps.OrderTypes[0] != adapter.OrderTypeMarket {
		t.Errorf("caps.OrderTypes = %v, want [MARKET]", caps.OrderTypes)
	}
}

func TestJupiterCapabilities(t *testing.T) {
	a := NewJupiter("http://localhost:8899")
	caps := a.Capabilities()
	if caps.Name != "Jupiter" {
		t.Errorf("caps.Name = %q, want %q", caps.Name, "Jupiter")
	}
	if caps.WebSocket {
		t.Error("caps.WebSocket = true, want false")
	}
	if caps.Margin {
		t.Error("caps.Margin = true, want false")
	}
	if caps.Futures {
		t.Error("caps.Futures = true, want false")
	}
	if len(caps.OrderTypes) != 1 || caps.OrderTypes[0] != adapter.OrderTypeMarket {
		t.Errorf("caps.OrderTypes = %v, want [MARKET]", caps.OrderTypes)
	}
}

func TestHyperliquidCapabilities(t *testing.T) {
	a := NewHyperliquid("https://api.hyperliquid.xyz")
	caps := a.Capabilities()
	if caps.Name != "Hyperliquid" {
		t.Errorf("caps.Name = %q, want %q", caps.Name, "Hyperliquid")
	}
	if !caps.WebSocket {
		t.Error("caps.WebSocket = false, want true")
	}
	if !caps.Margin {
		t.Error("caps.Margin = false, want true")
	}
	if !caps.Futures {
		t.Error("caps.Futures = false, want true")
	}
	if len(caps.OrderTypes) != 2 {
		t.Fatalf("caps.OrderTypes length = %d, want 2", len(caps.OrderTypes))
	}
	if caps.OrderTypes[0] != adapter.OrderTypeMarket {
		t.Errorf("caps.OrderTypes[0] = %q, want %q", caps.OrderTypes[0], adapter.OrderTypeMarket)
	}
	if caps.OrderTypes[1] != adapter.OrderTypeLimit {
		t.Errorf("caps.OrderTypes[1] = %q, want %q", caps.OrderTypes[1], adapter.OrderTypeLimit)
	}
}

func TestConnectDisconnectLifecycle(t *testing.T) {
	adapters := []adapter.TradingAdapter{
		NewUniswap("http://localhost:8545", 1),
		NewJupiter("http://localhost:8899"),
		NewHyperliquid("https://api.hyperliquid.xyz"),
	}

	for _, a := range adapters {
		t.Run(a.Name(), func(t *testing.T) {
			if a.IsConnected() {
				t.Error("IsConnected() = true before Connect, want false")
			}

			if err := a.Connect(context.Background()); err != nil {
				t.Fatalf("Connect() error = %v", err)
			}
			if !a.IsConnected() {
				t.Error("IsConnected() = false after Connect, want true")
			}

			if err := a.Disconnect(); err != nil {
				t.Fatalf("Disconnect() error = %v", err)
			}
			if a.IsConnected() {
				t.Error("IsConnected() = true after Disconnect, want false")
			}
		})
	}
}

func TestUnimplementedMethods(t *testing.T) {
	adapters := []adapter.TradingAdapter{
		NewUniswap("http://localhost:8545", 1),
		NewJupiter("http://localhost:8899"),
		NewHyperliquid("https://api.hyperliquid.xyz"),
	}
	ctx := context.Background()

	for _, a := range adapters {
		t.Run(a.Name(), func(t *testing.T) {
			_, err := a.GetPrice(ctx, "ETH/USDT")
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetPrice() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.GetCandles(ctx, "ETH/USDT", "1h", 100)
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetCandles() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.GetOrderBook(ctx, "ETH/USDT", 10)
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetOrderBook() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.PlaceOrder(ctx, adapter.Order{})
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("PlaceOrder() error = %v, want ErrNotImplemented", err)
			}

			err = a.CancelOrder(ctx, "order-123")
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("CancelOrder() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.GetOpenOrders(ctx)
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetOpenOrders() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.GetBalances(ctx)
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetBalances() error = %v, want ErrNotImplemented", err)
			}

			_, err = a.GetPositions(ctx)
			if !errors.Is(err, ErrNotImplemented) {
				t.Errorf("GetPositions() error = %v, want ErrNotImplemented", err)
			}
		})
	}
}
