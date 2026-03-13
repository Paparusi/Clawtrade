// internal/adapter/adapter.go
package adapter

import "context"

type TradingAdapter interface {
	Name() string
	Capabilities() AdapterCaps
	GetPrice(ctx context.Context, symbol string) (*Price, error)
	GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]Candle, error)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
	PlaceOrder(ctx context.Context, order Order) (*Order, error)
	CancelOrder(ctx context.Context, orderID string) error
	GetOpenOrders(ctx context.Context) ([]Order, error)
	GetBalances(ctx context.Context) ([]Balance, error)
	GetPositions(ctx context.Context) ([]Position, error)
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
}

type DataAdapter interface {
	Name() string
	GetData(ctx context.Context, query string) (any, error)
}

type SignalAdapter interface {
	Name() string
	OnSignal(ctx context.Context, handler func(signal any)) error
}
