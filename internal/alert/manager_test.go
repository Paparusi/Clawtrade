package alert

import (
	"sync"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/engine"
)

func TestManager_EvaluatePriceAbove(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, nil, ManagerConfig{RateLimitMinutes: 0})
	mgr.LoadAlerts()

	store.Create(&Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Message: "BTC above 70k", Enabled: true})
	mgr.LoadAlerts()

	var triggered []string
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		triggered = append(triggered, msg)
		mu.Unlock()
	})

	// Price below threshold — should not trigger
	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 69000.0},
	})
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	if len(triggered) != 0 {
		t.Fatalf("should not trigger, got %d", len(triggered))
	}
	mu.Unlock()

	// Price above threshold — should trigger
	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 71000.0},
	})
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	if len(triggered) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggered))
	}
	mu.Unlock()
}

func TestManager_EvaluatePriceBelow(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, nil, ManagerConfig{RateLimitMinutes: 0})

	store.Create(&Alert{Type: AlertTypePrice, Symbol: "ETH/USDT", Condition: CondBelow, Threshold: 3000, Message: "ETH below 3k", Enabled: true})
	mgr.LoadAlerts()

	var triggered []string
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		triggered = append(triggered, msg)
		mu.Unlock()
	})

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "ETH/USDT", "last": 2900.0},
	})
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	if len(triggered) != 1 {
		t.Fatalf("expected 1 trigger, got %d", len(triggered))
	}
	mu.Unlock()
}

func TestManager_RateLimit(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, nil, ManagerConfig{RateLimitMinutes: 60})

	store.Create(&Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true})
	mgr.LoadAlerts()

	var count int
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 71000.0},
	})
	time.Sleep(10 * time.Millisecond)

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 72000.0},
	})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Fatalf("expected 1 trigger (rate-limited), got %d", count)
	}
	mu.Unlock()
}

func TestManager_OneShot(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, nil, ManagerConfig{RateLimitMinutes: 0})

	store.Create(&Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true, OneShot: true})
	mgr.LoadAlerts()

	var count int
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		count++
		mu.Unlock()
	})

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 71000.0},
	})
	time.Sleep(10 * time.Millisecond)

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 72000.0},
	})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if count != 1 {
		t.Fatalf("expected 1 trigger (one-shot), got %d", count)
	}
	mu.Unlock()
}

func TestManager_CustomExpressionAlert(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Create candle_cache table for expression evaluation
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS candle_cache (
		symbol TEXT NOT NULL,
		timeframe TEXT NOT NULL,
		timestamp INTEGER NOT NULL,
		open REAL NOT NULL,
		high REAL NOT NULL,
		low REAL NOT NULL,
		close REAL NOT NULL,
		volume REAL NOT NULL,
		PRIMARY KEY (symbol, timeframe, timestamp)
	)`)
	if err != nil {
		t.Fatal(err)
	}

	// Insert 50 candles (downtrend for RSI < 30)
	now := time.Now().Unix()
	for i := 0; i < 50; i++ {
		price := 100.0 - float64(i)*1.5
		ts := now - int64((50-i)*3600)
		db.Exec(`INSERT INTO candle_cache (symbol, timeframe, timestamp, open, high, low, close, volume) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			"BTC/USDT", "1h", ts, price+1, price+2, price-1, price, 1000)
	}

	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, db, ManagerConfig{RateLimitMinutes: 0})

	store.Create(&Alert{
		Type:       AlertTypeCustom,
		Symbol:     "BTC/USDT",
		Condition:  CondExpression,
		Expression: "close < 30",
		Message:    "Custom rule triggered",
		Enabled:    true,
	})
	mgr.LoadAlerts()

	var triggered []string
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		triggered = append(triggered, msg)
		mu.Unlock()
	})

	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 25.0},
	})
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	if len(triggered) != 1 {
		t.Fatalf("expected 1 custom trigger, got %d", len(triggered))
	}
	mu.Unlock()
}

func TestManager_PnLAlert(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	mgr := NewManager(store, bus, nil, ManagerConfig{RateLimitMinutes: 0})

	store.Create(&Alert{Type: AlertTypePnL, Condition: CondBelow, Threshold: -500, Message: "PnL below -$500", Enabled: true})
	mgr.LoadAlerts()

	var triggered []string
	var mu sync.Mutex
	mgr.OnTrigger(func(a Alert, msg string) {
		mu.Lock()
		triggered = append(triggered, msg)
		mu.Unlock()
	})

	mgr.Evaluate(engine.Event{
		Type: "portfolio.update",
		Data: map[string]any{"total_pnl": -600.0},
	})
	time.Sleep(10 * time.Millisecond)
	mu.Lock()
	if len(triggered) != 1 {
		t.Fatalf("expected 1 PnL trigger, got %d", len(triggered))
	}
	mu.Unlock()
}
