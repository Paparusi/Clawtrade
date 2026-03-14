# Bybit Adapter Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the stub Bybit adapter with a full implementation supporting spot + futures trading via Bybit V5 Unified API.

**Architecture:** Mirror the existing Binance adapter pattern — REST API for market data and trading, WebSocket for real-time price streaming, HMAC-SHA256 authentication, testnet support.

**Tech Stack:** Go, Bybit V5 API, gorilla/websocket

---

### Task 1: HTTP helpers and authentication

**Files:**
- Modify: `internal/adapter/bybit/bybit.go`

**Step 1: Write the failing test**

Add test for symbol conversion and signing:

```go
func TestToBybitSymbol(t *testing.T) {
    tests := []struct{ input, want string }{
        {"BTC/USDT", "BTCUSDT"},
        {"ETH/BTC", "ETHBTC"},
        {"btc/usdt", "BTCUSDT"},
    }
    for _, tt := range tests {
        got := toBybitSymbol(tt.input)
        if got != tt.want {
            t.Errorf("toBybitSymbol(%q) = %q, want %q", tt.input, got, tt.want)
        }
    }
}

func TestFromBybitSymbol(t *testing.T) {
    tests := []struct{ input, want string }{
        {"BTCUSDT", "BTC/USDT"},
        {"ETHUSDC", "ETH/USDC"},
    }
    for _, tt := range tests {
        got := fromBybitSymbol(tt.input)
        if got != tt.want {
            t.Errorf("fromBybitSymbol(%q) = %q, want %q", tt.input, got, tt.want)
        }
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/adapter/bybit/ -run TestToBybitSymbol -v`
Expected: FAIL

**Step 3: Implement Adapter struct, HTTP helpers, auth signing**

- Adapter struct with apiKey, apiSecret, testnet, sync.RWMutex, wsConn, prices map
- `baseURL()` — returns `https://api-testnet.bybit.com` or `https://api.bybit.com`
- `wsBaseURL()` — returns `wss://stream-testnet.bybit.com` or `wss://stream.bybit.com`
- `publicGet(ctx, path, params)` — unsigned GET request
- `signedRequest(ctx, method, path, params)` — HMAC-SHA256 signed request using headers:
  - `X-BAPI-API-KEY`
  - `X-BAPI-SIGN`
  - `X-BAPI-TIMESTAMP`
  - `X-BAPI-RECV-WINDOW`
- `signedGet`, `signedPost` — convenience wrappers
- `sign(timestamp, params)` — HMAC-SHA256 of `timestamp + apiKey + recvWindow + queryString`
- `toBybitSymbol`, `fromBybitSymbol` — symbol format conversion
- `parseBybitResponse(body)` — unwrap `{"retCode":0, "result":{...}}` wrapper
- `mapOrderStatus(s)` — map Bybit status strings to adapter.OrderStatus

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapter/bybit/ -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/adapter/bybit/bybit.go internal/adapter/bybit/bybit_test.go
git commit -m "feat(bybit): add HTTP helpers and auth signing"
```

---

### Task 2: Market data endpoints (GetPrice, GetCandles, GetOrderBook)

**Files:**
- Modify: `internal/adapter/bybit/bybit.go`

**Step 1: Write failing tests**

```go
func TestAdapter_Name(t *testing.T) {
    a := New("key", "secret")
    if a.Name() != "bybit" {
        t.Error("expected name 'bybit'")
    }
}

func TestAdapter_Capabilities(t *testing.T) {
    a := New("key", "secret")
    caps := a.Capabilities()
    if !caps.Futures || !caps.WebSocket {
        t.Error("expected futures and websocket capabilities")
    }
}
```

**Step 2: Implement market data methods**

- `GetPrice` — `GET /v5/market/tickers?category=spot&symbol=BTCUSDT`
  - Parse: `result.list[0]` → `lastPrice`, `bid1Price`, `ask1Price`, `volume24h`
- `GetCandles` — `GET /v5/market/kline?category=spot&symbol=BTCUSDT&interval=60&limit=100`
  - Map timeframes: "1m"→"1", "5m"→"5", "15m"→"15", "1h"→"60", "4h"→"240", "1d"→"D"
  - Parse: `result.list` array of `[timestamp, open, high, low, close, volume]`
  - Note: Bybit returns newest first, reverse for chronological order
- `GetOrderBook` — `GET /v5/market/orderbook?category=spot&symbol=BTCUSDT&limit=20`
  - Parse: `result.b` (bids) and `result.a` (asks)

**Step 3: Run tests**

Run: `go test ./internal/adapter/bybit/ -v`
Expected: PASS

**Step 4: Commit**

```bash
git add internal/adapter/bybit/bybit.go internal/adapter/bybit/bybit_test.go
git commit -m "feat(bybit): add market data endpoints"
```

---

### Task 3: Trading endpoints (PlaceOrder, CancelOrder, GetOpenOrders)

**Files:**
- Modify: `internal/adapter/bybit/bybit.go`

**Step 1: Implement trading methods**

- `PlaceOrder` — `POST /v5/order/create`
  - Body JSON: `{"category":"spot","symbol":"BTCUSDT","side":"Buy","orderType":"Market","qty":"0.01"}`
  - Map order types: MARKET→"Market", LIMIT→"Limit", STOP→"Market" + triggerPrice
  - Parse response: `result.orderId`
- `CancelOrder` — `POST /v5/order/cancel`
  - Body JSON: `{"category":"spot","symbol":"BTCUSDT","orderId":"xxx"}`
  - OrderID format: `symbol:orderId` (same as Binance convention)
- `GetOpenOrders` — `GET /v5/order/realtime?category=spot`
  - Parse: `result.list` array

**Step 2: Run tests**

Run: `go test ./internal/adapter/bybit/ -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/adapter/bybit/bybit.go internal/adapter/bybit/bybit_test.go
git commit -m "feat(bybit): add trading endpoints"
```

---

### Task 4: Account endpoints (GetBalances, GetPositions)

**Files:**
- Modify: `internal/adapter/bybit/bybit.go`

**Step 1: Implement account methods**

- `GetBalances` — `GET /v5/account/wallet-balance?accountType=UNIFIED`
  - Parse: `result.list[0].coin[]` → asset, free (walletBalance - locked), locked, total
- `GetPositions` — `GET /v5/position/list?category=linear&settleCoin=USDT`
  - Parse: `result.list[]` → symbol, side, size, avgPrice, markPrice, unrealisedPnl

**Step 2: Run tests**

Run: `go test ./internal/adapter/bybit/ -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/adapter/bybit/bybit.go internal/adapter/bybit/bybit_test.go
git commit -m "feat(bybit): add account endpoints"
```

---

### Task 5: WebSocket real-time price stream

**Files:**
- Modify: `internal/adapter/bybit/bybit.go`

**Step 1: Implement WebSocket methods**

- `Connect` — set connected flag
- `SubscribePrices(ctx, symbols)` — connect to `wss://stream.bybit.com/v5/public/spot`
  - Send subscribe message: `{"op":"subscribe","args":["tickers.BTCUSDT","tickers.ETHUSDT"]}`
  - Read loop: parse `{"topic":"tickers.BTCUSDT","data":{...}}` → update prices map
  - Handle ping/pong keepalive
- `GetCachedPrice(symbol)` — return from prices map
- `Disconnect` — close WebSocket, cancel context
- `readWsLoop(ctx)` — goroutine reading WebSocket messages

**Step 2: Run all tests**

Run: `go test ./internal/adapter/bybit/ -v`
Expected: PASS

**Step 3: Commit**

```bash
git add internal/adapter/bybit/bybit.go internal/adapter/bybit/bybit_test.go
git commit -m "feat(bybit): add WebSocket real-time price stream"
```

---

### Task 6: Integration — wire into adapter manager

**Files:**
- Modify: `internal/adapter/bybit/bybit.go` (add SetTestnet)
- Check: `cmd/clawtrade/main.go` (ensure bybit case exists in adapter creation)

**Step 1: Verify bybit adapter creation in main.go**

Check that the switch case for "bybit" creates a real `bybit.New()` adapter.

**Step 2: Run full test suite**

Run: `go test ./... -count=1`
Expected: All PASS

**Step 3: Commit**

```bash
git add -A
git commit -m "feat(bybit): complete Bybit V5 adapter with spot + futures"
```
