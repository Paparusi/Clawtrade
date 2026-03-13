// internal/adapter/dex/uniswap.go
package dex

import (
	"context"
	"fmt"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

var ErrNotImplemented = fmt.Errorf("not implemented")

// UniswapAdapter implements the TradingAdapter interface for Uniswap DEX.
type UniswapAdapter struct {
	rpcURL    string
	chainID   int
	connected bool
}

// NewUniswap creates a new Uniswap adapter.
func NewUniswap(rpcURL string, chainID int) *UniswapAdapter {
	return &UniswapAdapter{
		rpcURL:  rpcURL,
		chainID: chainID,
	}
}

func (a *UniswapAdapter) Name() string {
	return "uniswap"
}

func (a *UniswapAdapter) Capabilities() adapter.AdapterCaps {
	return adapter.AdapterCaps{
		Name:      "Uniswap",
		WebSocket: false,
		Margin:    false,
		Futures:   false,
		OrderTypes: []adapter.OrderType{
			adapter.OrderTypeMarket,
		},
	}
}

func (a *UniswapAdapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) GetOrderBook(ctx context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) PlaceOrder(ctx context.Context, order adapter.Order) (*adapter.Order, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) CancelOrder(ctx context.Context, orderID string) error {
	return ErrNotImplemented
}

func (a *UniswapAdapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) GetBalances(ctx context.Context) ([]adapter.Balance, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) GetPositions(ctx context.Context) ([]adapter.Position, error) {
	return nil, ErrNotImplemented
}

func (a *UniswapAdapter) Connect(ctx context.Context) error {
	a.connected = true
	return nil
}

func (a *UniswapAdapter) Disconnect() error {
	a.connected = false
	return nil
}

func (a *UniswapAdapter) IsConnected() bool {
	return a.connected
}
