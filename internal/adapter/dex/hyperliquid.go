// internal/adapter/dex/hyperliquid.go
package dex

import (
	"context"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

// HyperliquidAdapter implements the TradingAdapter interface for Hyperliquid (on-chain perp DEX).
type HyperliquidAdapter struct {
	apiURL    string
	connected bool
}

// NewHyperliquid creates a new Hyperliquid adapter.
func NewHyperliquid(apiURL string) *HyperliquidAdapter {
	return &HyperliquidAdapter{
		apiURL: apiURL,
	}
}

func (a *HyperliquidAdapter) Name() string {
	return "hyperliquid"
}

func (a *HyperliquidAdapter) Capabilities() adapter.AdapterCaps {
	return adapter.AdapterCaps{
		Name:      "Hyperliquid",
		WebSocket: true,
		Margin:    true,
		Futures:   true,
		OrderTypes: []adapter.OrderType{
			adapter.OrderTypeMarket,
			adapter.OrderTypeLimit,
		},
	}
}

func (a *HyperliquidAdapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) GetOrderBook(ctx context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) PlaceOrder(ctx context.Context, order adapter.Order) (*adapter.Order, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) CancelOrder(ctx context.Context, orderID string) error {
	return ErrNotImplemented
}

func (a *HyperliquidAdapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) GetBalances(ctx context.Context) ([]adapter.Balance, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) GetPositions(ctx context.Context) ([]adapter.Position, error) {
	return nil, ErrNotImplemented
}

func (a *HyperliquidAdapter) Connect(ctx context.Context) error {
	a.connected = true
	return nil
}

func (a *HyperliquidAdapter) Disconnect() error {
	a.connected = false
	return nil
}

func (a *HyperliquidAdapter) IsConnected() bool {
	return a.connected
}
