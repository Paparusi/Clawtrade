// internal/adapter/simulation/simulation.go
package simulation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

type SimAdapter struct {
	mu             sync.RWMutex
	name           string
	connected      bool
	initialBalance float64
	balances       map[string]float64
	positions      []adapter.Position
	orders         []adapter.Order
	prices         map[string]float64
	nextOrderID    int
}

func New(name string, initialUSDT float64) *SimAdapter {
	return &SimAdapter{
		name:           name,
		initialBalance: initialUSDT,
		balances:       map[string]float64{"USDT": initialUSDT},
		prices:         make(map[string]float64),
	}
}

func (s *SimAdapter) Name() string { return s.name }

func (s *SimAdapter) Capabilities() adapter.AdapterCaps {
	return adapter.AdapterCaps{
		Name:       s.name,
		WebSocket:  false,
		Margin:     false,
		Futures:    false,
		OrderTypes: []adapter.OrderType{adapter.OrderTypeMarket, adapter.OrderTypeLimit},
	}
}

func (s *SimAdapter) SetPrice(symbol string, price float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prices[symbol] = price
}

func (s *SimAdapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.prices[symbol]
	if !ok {
		return nil, fmt.Errorf("no price for %s", symbol)
	}
	return &adapter.Price{Symbol: symbol, Bid: p, Ask: p, Last: p, Timestamp: time.Now()}, nil
}

func (s *SimAdapter) GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	return nil, fmt.Errorf("candles not available in simulation mode")
}

func (s *SimAdapter) GetOrderBook(ctx context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.prices[symbol]
	if !ok {
		return nil, fmt.Errorf("no price for %s", symbol)
	}
	return &adapter.OrderBook{
		Symbol: symbol,
		Bids:   []adapter.OrderBookEntry{{Price: p - 1, Amount: 10}},
		Asks:   []adapter.OrderBookEntry{{Price: p + 1, Amount: 10}},
	}, nil
}

func (s *SimAdapter) PlaceOrder(ctx context.Context, order adapter.Order) (*adapter.Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	price, ok := s.prices[order.Symbol]
	if !ok {
		return nil, fmt.Errorf("no price set for %s", order.Symbol)
	}

	cost := price * order.Size

	if order.Side == adapter.SideBuy {
		if s.balances["USDT"] < cost {
			return nil, fmt.Errorf("insufficient balance: need %f, have %f", cost, s.balances["USDT"])
		}
		s.balances["USDT"] -= cost
		s.addPosition(order.Symbol, adapter.SideBuy, order.Size, price)
	} else {
		s.closePosition(order.Symbol, order.Size, price)
		s.balances["USDT"] += cost
	}

	s.nextOrderID++
	filled := order
	filled.ID = fmt.Sprintf("sim-%d", s.nextOrderID)
	filled.Status = adapter.OrderStatusFilled
	filled.FilledAt = price
	filled.Exchange = s.name
	filled.CreatedAt = time.Now()
	s.orders = append(s.orders, filled)

	return &filled, nil
}

func (s *SimAdapter) addPosition(symbol string, side adapter.Side, size, price float64) {
	for i, p := range s.positions {
		if p.Symbol == symbol && p.Side == side {
			totalCost := p.EntryPrice*p.Size + price*size
			totalSize := p.Size + size
			s.positions[i].EntryPrice = totalCost / totalSize
			s.positions[i].Size = totalSize
			return
		}
	}
	s.positions = append(s.positions, adapter.Position{
		Symbol: symbol, Side: side, Size: size, EntryPrice: price,
		CurrentPrice: price, PnL: 0, Exchange: s.name, OpenedAt: time.Now(),
	})
}

func (s *SimAdapter) closePosition(symbol string, size, price float64) {
	for i, p := range s.positions {
		if p.Symbol == symbol {
			s.positions[i].Size -= size
			if s.positions[i].Size <= 0 {
				s.positions = append(s.positions[:i], s.positions[i+1:]...)
			}
			return
		}
	}
}

func (s *SimAdapter) CancelOrder(ctx context.Context, orderID string) error {
	return fmt.Errorf("market orders cannot be cancelled in simulation")
}

func (s *SimAdapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error) {
	return nil, nil
}

func (s *SimAdapter) GetBalances(ctx context.Context) ([]adapter.Balance, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var balances []adapter.Balance
	for asset, amount := range s.balances {
		balances = append(balances, adapter.Balance{Asset: asset, Free: amount, Total: amount})
	}
	return balances, nil
}

func (s *SimAdapter) GetPositions(ctx context.Context) ([]adapter.Position, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]adapter.Position, len(s.positions))
	for i, p := range s.positions {
		result[i] = p
		if price, ok := s.prices[p.Symbol]; ok {
			result[i].CurrentPrice = price
			if p.Side == adapter.SideBuy {
				result[i].PnL = (price - p.EntryPrice) * p.Size
			} else {
				result[i].PnL = (p.EntryPrice - price) * p.Size
			}
		}
	}
	return result, nil
}

func (s *SimAdapter) Connect(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = true
	return nil
}

func (s *SimAdapter) Disconnect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connected = false
	return nil
}

func (s *SimAdapter) IsConnected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.connected
}
