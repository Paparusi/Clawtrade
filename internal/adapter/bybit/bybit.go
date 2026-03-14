// internal/adapter/bybit/bybit.go
package bybit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/gorilla/websocket"
)

const (
	restBaseURL = "https://api.bybit.com"
	recvWindow  = "5000"
)

// PriceCallback is called when a real-time price update arrives via WebSocket.
type PriceCallback func(price adapter.Price)

// Adapter implements the TradingAdapter interface for Bybit.
type Adapter struct {
	apiKey    string
	apiSecret string
	testnet   bool

	mu        sync.RWMutex
	connected bool
	wsConn    *websocket.Conn
	wsCancel  context.CancelFunc
	prices    map[string]adapter.Price

	onPrice PriceCallback
}

// New creates a new Bybit adapter.
func New(apiKey, apiSecret string) *Adapter {
	return &Adapter{
		apiKey:    apiKey,
		apiSecret: apiSecret,
		prices:    make(map[string]adapter.Price),
	}
}

// SetTestnet enables the testnet endpoints.
func (a *Adapter) SetTestnet(enabled bool) {
	a.testnet = enabled
}

// OnPrice registers a callback for real-time price updates.
func (a *Adapter) OnPrice(cb PriceCallback) {
	a.onPrice = cb
}

func (a *Adapter) baseURL() string {
	if a.testnet {
		return "https://api-testnet.bybit.com"
	}
	return restBaseURL
}

func (a *Adapter) wsURL() string {
	if a.testnet {
		return "wss://stream-testnet.bybit.com/v5/public/spot"
	}
	return "wss://stream.bybit.com/v5/public/spot"
}

func (a *Adapter) Name() string {
	return "bybit"
}

func (a *Adapter) Capabilities() adapter.AdapterCaps {
	return adapter.AdapterCaps{
		Name:      "bybit",
		WebSocket: true,
		Margin:    true,
		Futures:   true,
		OrderTypes: []adapter.OrderType{
			adapter.OrderTypeMarket,
			adapter.OrderTypeLimit,
			adapter.OrderTypeStop,
		},
	}
}

// ─── REST API: Market Data ──────────────────────────────────────────

func (a *Adapter) GetPrice(ctx context.Context, symbol string) (*adapter.Price, error) {
	sym := toBybitSymbol(symbol)

	params := url.Values{
		"category": {"spot"},
		"symbol":   {sym},
	}

	resp, err := a.publicGet(ctx, "/v5/market/tickers", params)
	if err != nil {
		return nil, fmt.Errorf("get price: %w", err)
	}

	var result struct {
		List []struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
			Bid1Price string `json:"bid1Price"`
			Ask1Price string `json:"ask1Price"`
			Volume24h string `json:"volume24h"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse ticker: %w", err)
	}

	if len(result.List) == 0 {
		return nil, fmt.Errorf("no ticker data for %s", symbol)
	}

	t := result.List[0]
	bid, _ := strconv.ParseFloat(t.Bid1Price, 64)
	ask, _ := strconv.ParseFloat(t.Ask1Price, 64)
	last, _ := strconv.ParseFloat(t.LastPrice, 64)
	vol, _ := strconv.ParseFloat(t.Volume24h, 64)

	return &adapter.Price{
		Symbol:    symbol,
		Bid:       bid,
		Ask:       ask,
		Last:      last,
		Volume24h: vol,
		Timestamp: time.Now(),
	}, nil
}

func (a *Adapter) GetCandles(ctx context.Context, symbol, timeframe string, limit int) ([]adapter.Candle, error) {
	sym := toBybitSymbol(symbol)
	interval := mapTimeframe(timeframe)

	params := url.Values{
		"category": {"spot"},
		"symbol":   {sym},
		"interval": {interval},
		"limit":    {strconv.Itoa(limit)},
	}

	resp, err := a.publicGet(ctx, "/v5/market/kline", params)
	if err != nil {
		return nil, fmt.Errorf("get candles: %w", err)
	}

	var result struct {
		List [][]string `json:"list"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse klines: %w", err)
	}

	// Bybit returns newest first — reverse for chronological order
	candles := make([]adapter.Candle, 0, len(result.List))
	for i := len(result.List) - 1; i >= 0; i-- {
		k := result.List[i]
		if len(k) < 6 {
			continue
		}

		tsMs, _ := strconv.ParseInt(k[0], 10, 64)
		o, _ := strconv.ParseFloat(k[1], 64)
		h, _ := strconv.ParseFloat(k[2], 64)
		l, _ := strconv.ParseFloat(k[3], 64)
		c, _ := strconv.ParseFloat(k[4], 64)
		v, _ := strconv.ParseFloat(k[5], 64)

		candles = append(candles, adapter.Candle{
			Open:      o,
			High:      h,
			Low:       l,
			Close:     c,
			Volume:    v,
			Timestamp: time.UnixMilli(tsMs),
		})
	}

	return candles, nil
}

func (a *Adapter) GetOrderBook(ctx context.Context, symbol string, depth int) (*adapter.OrderBook, error) {
	sym := toBybitSymbol(symbol)
	if depth <= 0 {
		depth = 20
	}

	params := url.Values{
		"category": {"spot"},
		"symbol":   {sym},
		"limit":    {strconv.Itoa(depth)},
	}

	resp, err := a.publicGet(ctx, "/v5/market/orderbook", params)
	if err != nil {
		return nil, fmt.Errorf("get order book: %w", err)
	}

	var raw struct {
		B [][]string `json:"b"`
		A [][]string `json:"a"`
	}
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, fmt.Errorf("parse depth: %w", err)
	}

	ob := &adapter.OrderBook{Symbol: symbol}
	for _, b := range raw.B {
		if len(b) >= 2 {
			p, _ := strconv.ParseFloat(b[0], 64)
			amt, _ := strconv.ParseFloat(b[1], 64)
			ob.Bids = append(ob.Bids, adapter.OrderBookEntry{Price: p, Amount: amt})
		}
	}
	for _, ask := range raw.A {
		if len(ask) >= 2 {
			p, _ := strconv.ParseFloat(ask[0], 64)
			amt, _ := strconv.ParseFloat(ask[1], 64)
			ob.Asks = append(ob.Asks, adapter.OrderBookEntry{Price: p, Amount: amt})
		}
	}

	return ob, nil
}

// ─── REST API: Trading ──────────────────────────────────────────────

func (a *Adapter) PlaceOrder(ctx context.Context, order adapter.Order) (*adapter.Order, error) {
	body := map[string]string{
		"category": "spot",
		"symbol":   toBybitSymbol(order.Symbol),
		"side":     mapSideToBybit(order.Side),
		"qty":      strconv.FormatFloat(order.Size, 'f', -1, 64),
	}

	switch order.Type {
	case adapter.OrderTypeMarket:
		body["orderType"] = "Market"
	case adapter.OrderTypeLimit:
		body["orderType"] = "Limit"
		body["price"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
		body["timeInForce"] = "GTC"
	case adapter.OrderTypeStop:
		body["orderType"] = "Market"
		body["triggerPrice"] = strconv.FormatFloat(order.Price, 'f', -1, 64)
		if order.Side == adapter.SideSell {
			body["triggerDirection"] = "2" // fall below
		} else {
			body["triggerDirection"] = "1" // rise above
		}
	}

	resp, err := a.signedPost(ctx, "/v5/order/create", body)
	if err != nil {
		return nil, fmt.Errorf("place order: %w", err)
	}

	var result struct {
		OrderID     string `json:"orderId"`
		OrderLinkID string `json:"orderLinkId"`
		Symbol      string `json:"symbol"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse order response: %w", err)
	}

	return &adapter.Order{
		ID:        result.OrderID,
		Symbol:    order.Symbol,
		Side:      order.Side,
		Type:      order.Type,
		Price:     order.Price,
		Size:      order.Size,
		Status:    adapter.OrderStatusPending,
		Exchange:  "bybit",
		CreatedAt: time.Now(),
	}, nil
}

func (a *Adapter) CancelOrder(ctx context.Context, orderID string) error {
	parts := strings.SplitN(orderID, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("orderID must be in symbol:orderId format")
	}

	body := map[string]string{
		"category": "spot",
		"symbol":   parts[0],
		"orderId":  parts[1],
	}

	_, err := a.signedPost(ctx, "/v5/order/cancel", body)
	return err
}

func (a *Adapter) GetOpenOrders(ctx context.Context) ([]adapter.Order, error) {
	params := url.Values{
		"category": {"spot"},
	}

	resp, err := a.signedGet(ctx, "/v5/order/realtime", params)
	if err != nil {
		return nil, fmt.Errorf("get open orders: %w", err)
	}

	var result struct {
		List []struct {
			OrderID     string `json:"orderId"`
			Symbol      string `json:"symbol"`
			Side        string `json:"side"`
			OrderType   string `json:"orderType"`
			Price       string `json:"price"`
			Qty         string `json:"qty"`
			OrderStatus string `json:"orderStatus"`
			CreatedTime string `json:"createdTime"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse orders: %w", err)
	}

	orders := make([]adapter.Order, 0, len(result.List))
	for _, r := range result.List {
		price, _ := strconv.ParseFloat(r.Price, 64)
		size, _ := strconv.ParseFloat(r.Qty, 64)
		tsMs, _ := strconv.ParseInt(r.CreatedTime, 10, 64)

		orders = append(orders, adapter.Order{
			ID:        r.OrderID,
			Symbol:    fromBybitSymbol(r.Symbol),
			Side:      mapSideFromBybit(r.Side),
			Type:      mapOrderTypeFromBybit(r.OrderType),
			Price:     price,
			Size:      size,
			Status:    mapOrderStatus(r.OrderStatus),
			Exchange:  "bybit",
			CreatedAt: time.UnixMilli(tsMs),
		})
	}

	return orders, nil
}

func (a *Adapter) GetBalances(ctx context.Context) ([]adapter.Balance, error) {
	params := url.Values{
		"accountType": {"UNIFIED"},
	}

	resp, err := a.signedGet(ctx, "/v5/account/wallet-balance", params)
	if err != nil {
		return nil, fmt.Errorf("get balances: %w", err)
	}

	var result struct {
		List []struct {
			Coin []struct {
				Coin                string `json:"coin"`
				WalletBalance       string `json:"walletBalance"`
				Locked              string `json:"locked"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
			} `json:"coin"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse account: %w", err)
	}

	var balances []adapter.Balance
	for _, account := range result.List {
		for _, c := range account.Coin {
			free, _ := strconv.ParseFloat(c.AvailableToWithdraw, 64)
			locked, _ := strconv.ParseFloat(c.Locked, 64)
			total, _ := strconv.ParseFloat(c.WalletBalance, 64)
			if total > 0 {
				balances = append(balances, adapter.Balance{
					Asset:  c.Coin,
					Free:   free,
					Locked: locked,
					Total:  total,
				})
			}
		}
	}

	return balances, nil
}

func (a *Adapter) GetPositions(ctx context.Context) ([]adapter.Position, error) {
	params := url.Values{
		"category":   {"linear"},
		"settleCoin": {"USDT"},
	}

	resp, err := a.signedGet(ctx, "/v5/position/list", params)
	if err != nil {
		return nil, fmt.Errorf("get positions: %w", err)
	}

	var result struct {
		List []struct {
			Symbol        string `json:"symbol"`
			Side          string `json:"side"`
			Size          string `json:"size"`
			AvgPrice      string `json:"avgPrice"`
			MarkPrice     string `json:"markPrice"`
			UnrealisedPnl string `json:"unrealisedPnl"`
			CreatedTime   string `json:"createdTime"`
		} `json:"list"`
	}
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("parse positions: %w", err)
	}

	var positions []adapter.Position
	for _, r := range result.List {
		size, _ := strconv.ParseFloat(r.Size, 64)
		if size <= 0 {
			continue
		}
		entry, _ := strconv.ParseFloat(r.AvgPrice, 64)
		mark, _ := strconv.ParseFloat(r.MarkPrice, 64)
		pnl, _ := strconv.ParseFloat(r.UnrealisedPnl, 64)
		tsMs, _ := strconv.ParseInt(r.CreatedTime, 10, 64)

		positions = append(positions, adapter.Position{
			Symbol:       fromBybitSymbol(r.Symbol),
			Side:         mapSideFromBybit(r.Side),
			Size:         size,
			EntryPrice:   entry,
			CurrentPrice: mark,
			PnL:          pnl,
			Exchange:     "bybit",
			OpenedAt:     time.UnixMilli(tsMs),
		})
	}

	return positions, nil
}

// ─── WebSocket: Real-time Price Stream ──────────────────────────────

func (a *Adapter) Connect(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.connected {
		return nil
	}

	a.connected = true
	return nil
}

// SubscribePrices connects to Bybit WebSocket and streams tickers for given symbols.
func (a *Adapter) SubscribePrices(ctx context.Context, symbols []string) error {
	if len(symbols) == 0 {
		return nil
	}

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, a.wsURL(), nil)
	if err != nil {
		return fmt.Errorf("websocket connect: %w", err)
	}

	// Build subscription args
	var args []string
	for _, s := range symbols {
		sym := toBybitSymbol(s)
		args = append(args, "tickers."+sym)
	}

	sub := map[string]interface{}{
		"op":   "subscribe",
		"args": args,
	}
	if err := conn.WriteJSON(sub); err != nil {
		conn.Close()
		return fmt.Errorf("websocket subscribe: %w", err)
	}

	a.mu.Lock()
	a.wsConn = conn
	a.connected = true
	wsCtx, cancel := context.WithCancel(ctx)
	a.wsCancel = cancel
	a.mu.Unlock()

	go a.readWsLoop(wsCtx)

	log.Printf("bybit: WebSocket connected, streaming %d symbols", len(symbols))
	return nil
}

func (a *Adapter) readWsLoop(ctx context.Context) {
	defer func() {
		a.mu.Lock()
		a.connected = false
		if a.wsConn != nil {
			a.wsConn.Close()
		}
		a.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_, message, err := a.wsConn.ReadMessage()
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("bybit: websocket read error: %v", err)
			}
			return
		}

		// Handle ping from server
		var ping struct {
			Op string `json:"op"`
		}
		if json.Unmarshal(message, &ping) == nil && ping.Op == "ping" {
			pong := map[string]string{"op": "pong"}
			a.wsConn.WriteJSON(pong)
			continue
		}

		var msg struct {
			Topic string `json:"topic"`
			Type  string `json:"type"`
			Data  struct {
				Symbol    string `json:"symbol"`
				LastPrice string `json:"lastPrice"`
				Bid1Price string `json:"bid1Price"`
				Ask1Price string `json:"ask1Price"`
				Volume24h string `json:"volume24h"`
			} `json:"data"`
		}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		if msg.Topic == "" || !strings.HasPrefix(msg.Topic, "tickers.") {
			continue
		}

		last, _ := strconv.ParseFloat(msg.Data.LastPrice, 64)
		bid, _ := strconv.ParseFloat(msg.Data.Bid1Price, 64)
		ask, _ := strconv.ParseFloat(msg.Data.Ask1Price, 64)
		vol, _ := strconv.ParseFloat(msg.Data.Volume24h, 64)

		price := adapter.Price{
			Symbol:    fromBybitSymbol(msg.Data.Symbol),
			Last:      last,
			Bid:       bid,
			Ask:       ask,
			Volume24h: vol,
			Timestamp: time.Now(),
		}

		a.mu.Lock()
		a.prices[price.Symbol] = price
		a.mu.Unlock()

		if a.onPrice != nil {
			a.onPrice(price)
		}
	}
}

// GetCachedPrice returns the last known price from WebSocket stream.
func (a *Adapter) GetCachedPrice(symbol string) (adapter.Price, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	p, ok := a.prices[symbol]
	return p, ok
}

func (a *Adapter) Disconnect() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.wsCancel != nil {
		a.wsCancel()
	}
	if a.wsConn != nil {
		a.wsConn.Close()
		a.wsConn = nil
	}
	a.connected = false
	return nil
}

func (a *Adapter) IsConnected() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.connected
}

// ─── HTTP helpers ───────────────────────────────────────────────────

// publicGet makes an unauthenticated GET request and returns the "result" field.
func (a *Adapter) publicGet(ctx context.Context, path string, params url.Values) ([]byte, error) {
	u := a.baseURL() + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bybit API error (%d): %s", resp.StatusCode, string(body))
	}

	return a.unwrapResponse(body)
}

// signedGet makes an authenticated GET request.
func (a *Adapter) signedGet(ctx context.Context, path string, params url.Values) ([]byte, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	queryString := params.Encode()

	// Signature payload for GET: timestamp + apiKey + recvWindow + queryString
	payload := timestamp + a.apiKey + recvWindow + queryString
	signature := a.sign(payload)

	u := a.baseURL() + path
	if queryString != "" {
		u += "?" + queryString
	}

	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-BAPI-API-KEY", a.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bybit API error (%d): %s", resp.StatusCode, string(body))
	}

	return a.unwrapResponse(body)
}

// signedPost makes an authenticated POST request with a JSON body.
func (a *Adapter) signedPost(ctx context.Context, path string, bodyMap map[string]string) ([]byte, error) {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)

	bodyJSON, err := json.Marshal(bodyMap)
	if err != nil {
		return nil, err
	}

	// Signature payload for POST: timestamp + apiKey + recvWindow + bodyJSON
	payload := timestamp + a.apiKey + recvWindow + string(bodyJSON)
	signature := a.sign(payload)

	u := a.baseURL() + path

	req, err := http.NewRequestWithContext(ctx, "POST", u, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", a.apiKey)
	req.Header.Set("X-BAPI-SIGN", signature)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bybit API error (%d): %s", resp.StatusCode, string(body))
	}

	return a.unwrapResponse(body)
}

// unwrapResponse extracts the "result" field from Bybit's standard response envelope.
func (a *Adapter) unwrapResponse(body []byte) ([]byte, error) {
	var envelope struct {
		RetCode int             `json:"retCode"`
		RetMsg  string          `json:"retMsg"`
		Result  json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("parse response envelope: %w", err)
	}

	if envelope.RetCode != 0 {
		return nil, fmt.Errorf("bybit API error (%d): %s", envelope.RetCode, envelope.RetMsg)
	}

	return envelope.Result, nil
}

func (a *Adapter) sign(payload string) string {
	mac := hmac.New(sha256.New, []byte(a.apiSecret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// ─── Symbol conversion ─────────────────────────────────────────────

// toBybitSymbol converts "BTC/USDT" → "BTCUSDT".
func toBybitSymbol(symbol string) string {
	return strings.ReplaceAll(strings.ToUpper(symbol), "/", "")
}

// fromBybitSymbol converts "BTCUSDT" → "BTC/USDT".
func fromBybitSymbol(symbol string) string {
	stables := []string{"USDT", "USDC", "BTC", "ETH"}
	upper := strings.ToUpper(symbol)
	for _, quote := range stables {
		if strings.HasSuffix(upper, quote) {
			base := upper[:len(upper)-len(quote)]
			if base != "" {
				return base + "/" + quote
			}
		}
	}
	return symbol
}

// ─── Mapping helpers ────────────────────────────────────────────────

// mapTimeframe converts standard timeframe strings to Bybit interval values.
func mapTimeframe(tf string) string {
	switch tf {
	case "1m":
		return "1"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h":
		return "60"
	case "4h":
		return "240"
	case "1d":
		return "D"
	case "1w":
		return "W"
	default:
		return tf
	}
}

func mapSideToBybit(side adapter.Side) string {
	switch side {
	case adapter.SideBuy:
		return "Buy"
	case adapter.SideSell:
		return "Sell"
	default:
		return string(side)
	}
}

func mapSideFromBybit(side string) adapter.Side {
	switch side {
	case "Buy":
		return adapter.SideBuy
	case "Sell":
		return adapter.SideSell
	default:
		return adapter.Side(side)
	}
}

func mapOrderTypeFromBybit(ot string) adapter.OrderType {
	switch ot {
	case "Market":
		return adapter.OrderTypeMarket
	case "Limit":
		return adapter.OrderTypeLimit
	default:
		return adapter.OrderType(ot)
	}
}

func mapOrderStatus(s string) adapter.OrderStatus {
	switch s {
	case "New", "PartiallyFilled":
		return adapter.OrderStatusPending
	case "Filled":
		return adapter.OrderStatusFilled
	case "Cancelled", "Rejected", "Deactivated":
		return adapter.OrderStatusCanceled
	default:
		return adapter.OrderStatusPending
	}
}
