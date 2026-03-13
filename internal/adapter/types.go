// internal/adapter/types.go
package adapter

import "time"

type Side string

const (
	SideBuy  Side = "BUY"
	SideSell Side = "SELL"
)

type OrderType string

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
	OrderTypeStop   OrderType = "STOP"
)

type OrderStatus string

const (
	OrderStatusPending  OrderStatus = "PENDING"
	OrderStatusFilled   OrderStatus = "FILLED"
	OrderStatusCanceled OrderStatus = "CANCELED"
	OrderStatusFailed   OrderStatus = "FAILED"
)

type Price struct {
	Symbol    string    `json:"symbol"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Last      float64   `json:"last"`
	Volume24h float64   `json:"volume_24h"`
	Timestamp time.Time `json:"timestamp"`
}

type Candle struct {
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

type OrderBookEntry struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

type OrderBook struct {
	Symbol string           `json:"symbol"`
	Bids   []OrderBookEntry `json:"bids"`
	Asks   []OrderBookEntry `json:"asks"`
}

type Order struct {
	ID        string      `json:"id"`
	Symbol    string      `json:"symbol"`
	Side      Side        `json:"side"`
	Type      OrderType   `json:"type"`
	Price     float64     `json:"price,omitempty"`
	Size      float64     `json:"size"`
	Status    OrderStatus `json:"status"`
	Exchange  string      `json:"exchange"`
	FilledAt  float64     `json:"filled_at,omitempty"`
	CreatedAt time.Time   `json:"created_at"`
}

type Position struct {
	Symbol       string    `json:"symbol"`
	Side         Side      `json:"side"`
	Size         float64   `json:"size"`
	EntryPrice   float64   `json:"entry_price"`
	CurrentPrice float64   `json:"current_price"`
	PnL          float64   `json:"pnl"`
	Exchange     string    `json:"exchange"`
	OpenedAt     time.Time `json:"opened_at"`
}

type Balance struct {
	Asset  string  `json:"asset"`
	Free   float64 `json:"free"`
	Locked float64 `json:"locked"`
	Total  float64 `json:"total"`
}

type AdapterCaps struct {
	Name       string      `json:"name"`
	WebSocket  bool        `json:"websocket"`
	Margin     bool        `json:"margin"`
	Futures    bool        `json:"futures"`
	OrderTypes []OrderType `json:"order_types"`
}
