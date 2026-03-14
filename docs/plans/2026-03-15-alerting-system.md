# Alerting System Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Event-driven alerting system that evaluates price/PnL/risk/trade/system/custom alerts and dispatches via Telegram + WebSocket, with SQLite persistence and agent chat management.

**Architecture:** New `internal/alert/` package with AlertManager subscribing to EventBus. Alerts stored in SQLite. Three new agent tools (create_alert, list_alerts, delete_alert). Dispatcher sends to Telegram + publishes `alert.triggered` events. Daily briefing goroutine sends portfolio summary via Telegram.

**Tech Stack:** Go, existing EventBus/TelegramBot/WebSocket hub, SQLite, expression evaluator from backtest package.

---

### Task 1: Alert Model & Database Migration

**Files:**
- Create: `internal/alert/model.go`
- Modify: `internal/database/migrations.go:6-87`
- Create: `internal/alert/model_test.go`

**Step 1: Write the failing test**

Create `internal/alert/model_test.go`:

```go
package alert

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func testDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		type TEXT NOT NULL,
		symbol TEXT,
		condition TEXT NOT NULL,
		threshold REAL,
		expression TEXT,
		message TEXT,
		enabled BOOLEAN DEFAULT 1,
		one_shot BOOLEAN DEFAULT 0,
		last_triggered_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE alert_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		alert_id INTEGER NOT NULL,
		event_type TEXT NOT NULL,
		value REAL,
		message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAlertStore_CreateAndList(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{
		Type:      AlertTypePrice,
		Symbol:    "BTC/USDT",
		Condition: CondAbove,
		Threshold: 70000,
		Message:   "BTC above 70k",
		Enabled:   true,
		OneShot:   false,
	}

	id, err := store.Create(a)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if id == 0 {
		t.Fatal("expected non-zero ID")
	}

	alerts, err := store.ListEnabled()
	if err != nil {
		t.Fatalf("ListEnabled: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 alert, got %d", len(alerts))
	}
	if alerts[0].Symbol != "BTC/USDT" {
		t.Errorf("expected BTC/USDT, got %s", alerts[0].Symbol)
	}
}

func TestAlertStore_Delete(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "ETH/USDT", Condition: CondBelow, Threshold: 3000, Enabled: true}
	id, _ := store.Create(a)

	if err := store.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	alerts, _ := store.ListEnabled()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 alerts after delete, got %d", len(alerts))
	}
}

func TestAlertStore_Disable(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "ETH/USDT", Condition: CondAbove, Threshold: 5000, Enabled: true, OneShot: true}
	id, _ := store.Create(a)

	if err := store.Disable(id); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	alerts, _ := store.ListEnabled()
	if len(alerts) != 0 {
		t.Fatalf("expected 0 enabled alerts, got %d", len(alerts))
	}
}

func TestAlertStore_LogTrigger(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true}
	id, _ := store.Create(a)

	if err := store.LogTrigger(id, "price.update", 71000, "BTC crossed 70k"); err != nil {
		t.Fatalf("LogTrigger: %v", err)
	}

	history, err := store.TodayHistory()
	if err != nil {
		t.Fatalf("TodayHistory: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected 1 history entry, got %d", len(history))
	}
}

func TestAlertStore_UpdateLastTriggered(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	a := &Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true}
	id, _ := store.Create(a)

	now := time.Now()
	if err := store.UpdateLastTriggered(id, now); err != nil {
		t.Fatalf("UpdateLastTriggered: %v", err)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -v`
Expected: FAIL — package doesn't exist yet.

**Step 3: Write minimal implementation**

Create `internal/alert/model.go`:

```go
package alert

import (
	"database/sql"
	"time"
)

// Alert types
const (
	AlertTypePrice  = "price"
	AlertTypePnL    = "pnl"
	AlertTypeRisk   = "risk"
	AlertTypeTrade  = "trade"
	AlertTypeSystem = "system"
	AlertTypeCustom = "custom"
)

// Alert conditions
const (
	CondAbove      = "above"
	CondBelow      = "below"
	CondCross      = "cross"
	CondExpression = "expression"
)

// Alert represents an alert rule.
type Alert struct {
	ID              int64
	Type            string
	Symbol          string
	Condition       string
	Threshold       float64
	Expression      string
	Message         string
	Enabled         bool
	OneShot         bool
	LastTriggeredAt *time.Time
	CreatedAt       time.Time
}

// HistoryEntry represents a triggered alert log entry.
type HistoryEntry struct {
	ID        int64
	AlertID   int64
	EventType string
	Value     float64
	Message   string
	CreatedAt time.Time
}

// Store handles alert persistence in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore creates a new alert store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Create inserts a new alert and returns its ID.
func (s *Store) Create(a *Alert) (int64, error) {
	res, err := s.db.Exec(
		`INSERT INTO alerts (type, symbol, condition, threshold, expression, message, enabled, one_shot) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Type, a.Symbol, a.Condition, a.Threshold, a.Expression, a.Message, a.Enabled, a.OneShot,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// ListEnabled returns all enabled alerts.
func (s *Store) ListEnabled() ([]Alert, error) {
	rows, err := s.db.Query(`SELECT id, type, symbol, condition, threshold, expression, message, enabled, one_shot, last_triggered_at, created_at FROM alerts WHERE enabled = 1`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var lastTriggered sql.NullTime
		if err := rows.Scan(&a.ID, &a.Type, &a.Symbol, &a.Condition, &a.Threshold, &a.Expression, &a.Message, &a.Enabled, &a.OneShot, &lastTriggered, &a.CreatedAt); err != nil {
			return nil, err
		}
		if lastTriggered.Valid {
			a.LastTriggeredAt = &lastTriggered.Time
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// ListAll returns all alerts (including disabled).
func (s *Store) ListAll() ([]Alert, error) {
	rows, err := s.db.Query(`SELECT id, type, symbol, condition, threshold, expression, message, enabled, one_shot, last_triggered_at, created_at FROM alerts`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		var lastTriggered sql.NullTime
		if err := rows.Scan(&a.ID, &a.Type, &a.Symbol, &a.Condition, &a.Threshold, &a.Expression, &a.Message, &a.Enabled, &a.OneShot, &lastTriggered, &a.CreatedAt); err != nil {
			return nil, err
		}
		if lastTriggered.Valid {
			a.LastTriggeredAt = &lastTriggered.Time
		}
		alerts = append(alerts, a)
	}
	return alerts, nil
}

// Delete removes an alert by ID.
func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM alerts WHERE id = ?`, id)
	return err
}

// Disable sets an alert's enabled flag to false.
func (s *Store) Disable(id int64) error {
	_, err := s.db.Exec(`UPDATE alerts SET enabled = 0 WHERE id = ?`, id)
	return err
}

// UpdateLastTriggered updates the last_triggered_at timestamp.
func (s *Store) UpdateLastTriggered(id int64, at time.Time) error {
	_, err := s.db.Exec(`UPDATE alerts SET last_triggered_at = ? WHERE id = ?`, at, id)
	return err
}

// LogTrigger records a triggered alert in alert_history.
func (s *Store) LogTrigger(alertID int64, eventType string, value float64, message string) error {
	_, err := s.db.Exec(
		`INSERT INTO alert_history (alert_id, event_type, value, message) VALUES (?, ?, ?, ?)`,
		alertID, eventType, value, message,
	)
	return err
}

// TodayHistory returns alert history entries from today.
func (s *Store) TodayHistory() ([]HistoryEntry, error) {
	rows, err := s.db.Query(`SELECT id, alert_id, event_type, value, message, created_at FROM alert_history WHERE date(created_at) = date('now')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []HistoryEntry
	for rows.Next() {
		var e HistoryEntry
		if err := rows.Scan(&e.ID, &e.AlertID, &e.EventType, &e.Value, &e.Message, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}
```

**Step 4: Run test to verify it passes**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -v`
Expected: PASS — all 5 tests pass.

**Step 5: Add database migration**

Modify `internal/database/migrations.go` — add these two table definitions to the `migrations` slice (after the `candle_cache` entry):

```go
`CREATE TABLE IF NOT EXISTS alerts (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	type TEXT NOT NULL,
	symbol TEXT,
	condition TEXT NOT NULL,
	threshold REAL,
	expression TEXT,
	message TEXT,
	enabled BOOLEAN DEFAULT 1,
	one_shot BOOLEAN DEFAULT 0,
	last_triggered_at DATETIME,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
`CREATE TABLE IF NOT EXISTS alert_history (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	alert_id INTEGER NOT NULL,
	event_type TEXT NOT NULL,
	value REAL,
	message TEXT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)`,
```

**Step 6: Commit**

```bash
git add internal/alert/model.go internal/alert/model_test.go internal/database/migrations.go
git commit -m "feat(alert): add alert model, store, and database migration"
```

---

### Task 2: AlertManager Core — Event Evaluation

**Files:**
- Create: `internal/alert/manager.go`
- Create: `internal/alert/manager_test.go`

**Step 1: Write the failing test**

Create `internal/alert/manager_test.go`:

```go
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

	// First trigger
	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 71000.0},
	})
	time.Sleep(10 * time.Millisecond)

	// Second trigger — should be rate-limited
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

	// First trigger
	mgr.Evaluate(engine.Event{
		Type: "price.update",
		Data: map[string]any{"symbol": "BTC/USDT", "last": 71000.0},
	})
	time.Sleep(10 * time.Millisecond)

	// Second trigger — one-shot should be disabled
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
```

**Step 2: Run test to verify it fails**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestManager -v`
Expected: FAIL — `NewManager` not defined.

**Step 3: Write minimal implementation**

Create `internal/alert/manager.go`:

```go
package alert

import (
	"fmt"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/engine"
)

// TriggerHandler is called when an alert fires.
type TriggerHandler func(alert Alert, message string)

// ManagerConfig holds AlertManager configuration.
type ManagerConfig struct {
	RateLimitMinutes int
}

// Manager evaluates events against active alerts and dispatches notifications.
type Manager struct {
	mu       sync.RWMutex
	store    *Store
	bus      *engine.EventBus
	db       interface{} // reserved for custom expression candle lookups
	alerts   []Alert
	handlers []TriggerHandler
	config   ManagerConfig

	// Track last trigger time per alert ID for rate limiting
	lastTrigger map[int64]time.Time
}

// NewManager creates a new AlertManager.
func NewManager(store *Store, bus *engine.EventBus, db interface{}, cfg ManagerConfig) *Manager {
	return &Manager{
		store:       store,
		bus:         bus,
		db:          db,
		config:      cfg,
		lastTrigger: make(map[int64]time.Time),
	}
}

// OnTrigger registers a callback for when an alert fires.
func (m *Manager) OnTrigger(handler TriggerHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers = append(m.handlers, handler)
}

// LoadAlerts loads enabled alerts from the database into memory.
func (m *Manager) LoadAlerts() error {
	alerts, err := m.store.ListEnabled()
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.alerts = alerts
	m.mu.Unlock()
	return nil
}

// AddAlert creates a new alert in DB and reloads in-memory list.
func (m *Manager) AddAlert(a *Alert) (int64, error) {
	id, err := m.store.Create(a)
	if err != nil {
		return 0, err
	}
	m.LoadAlerts()
	return id, nil
}

// RemoveAlert deletes an alert and reloads in-memory list.
func (m *Manager) RemoveAlert(id int64) error {
	if err := m.store.Delete(id); err != nil {
		return err
	}
	m.LoadAlerts()
	return nil
}

// Start subscribes to EventBus and begins evaluating alerts.
func (m *Manager) Start() {
	m.LoadAlerts()
	m.bus.Subscribe("price.*", func(e engine.Event) { m.Evaluate(e) })
	m.bus.Subscribe("trade.*", func(e engine.Event) { m.Evaluate(e) })
	m.bus.Subscribe("risk.*", func(e engine.Event) { m.Evaluate(e) })
	m.bus.Subscribe("portfolio.*", func(e engine.Event) { m.Evaluate(e) })
	m.bus.Subscribe("system.*", func(e engine.Event) { m.Evaluate(e) })
}

// Evaluate checks all active alerts against an incoming event.
func (m *Manager) Evaluate(e engine.Event) {
	m.mu.RLock()
	alerts := make([]Alert, len(m.alerts))
	copy(alerts, m.alerts)
	m.mu.RUnlock()

	for _, a := range alerts {
		if !a.Enabled {
			continue
		}
		if triggered, msg := m.checkAlert(a, e); triggered {
			m.fireAlert(a, msg, e)
		}
	}
}

// checkAlert determines if an event triggers a specific alert.
func (m *Manager) checkAlert(a Alert, e engine.Event) (bool, string) {
	switch a.Type {
	case AlertTypePrice:
		return m.checkPriceAlert(a, e)
	case AlertTypePnL:
		return m.checkPnLAlert(a, e)
	case AlertTypeRisk:
		return m.checkEventTypeMatch(a, e, "risk.")
	case AlertTypeTrade:
		return m.checkEventTypeMatch(a, e, "trade.")
	case AlertTypeSystem:
		return m.checkEventTypeMatch(a, e, "system.")
	case AlertTypeCustom:
		return m.checkCustomAlert(a, e)
	}
	return false, ""
}

func (m *Manager) checkPriceAlert(a Alert, e engine.Event) (bool, string) {
	if e.Type != "price.update" {
		return false, ""
	}
	symbol, _ := e.Data["symbol"].(string)
	if a.Symbol != "" && symbol != a.Symbol {
		return false, ""
	}
	price, ok := e.Data["last"].(float64)
	if !ok {
		return false, ""
	}

	switch a.Condition {
	case CondAbove:
		if price > a.Threshold {
			return true, fmt.Sprintf("%s price $%.2f crossed above $%.2f", symbol, price, a.Threshold)
		}
	case CondBelow:
		if price < a.Threshold {
			return true, fmt.Sprintf("%s price $%.2f dropped below $%.2f", symbol, price, a.Threshold)
		}
	}
	return false, ""
}

func (m *Manager) checkPnLAlert(a Alert, e engine.Event) (bool, string) {
	if e.Type != "portfolio.update" {
		return false, ""
	}
	pnl, ok := e.Data["total_pnl"].(float64)
	if !ok {
		return false, ""
	}

	switch a.Condition {
	case CondAbove:
		if pnl > a.Threshold {
			return true, fmt.Sprintf("Portfolio PnL $%.2f crossed above $%.2f", pnl, a.Threshold)
		}
	case CondBelow:
		if pnl < a.Threshold {
			return true, fmt.Sprintf("Portfolio PnL $%.2f dropped below $%.2f", pnl, a.Threshold)
		}
	}
	return false, ""
}

func (m *Manager) checkEventTypeMatch(a Alert, e engine.Event, prefix string) (bool, string) {
	if len(e.Type) > len(prefix) && e.Type[:len(prefix)] == prefix {
		msg := a.Message
		if msg == "" {
			msg = fmt.Sprintf("Alert: %s event — %s", e.Type, formatEventData(e.Data))
		}
		return true, msg
	}
	return false, ""
}

func (m *Manager) checkCustomAlert(a Alert, e engine.Event) (bool, string) {
	// Custom expression alerts are evaluated in Task 3
	return false, ""
}

// fireAlert handles rate limiting, logging, dispatching.
func (m *Manager) fireAlert(a Alert, msg string, e engine.Event) {
	now := time.Now()

	// Rate limiting
	m.mu.RLock()
	lastTime, hasLast := m.lastTrigger[a.ID]
	m.mu.RUnlock()

	if hasLast && m.config.RateLimitMinutes > 0 {
		if now.Sub(lastTime) < time.Duration(m.config.RateLimitMinutes)*time.Minute {
			return
		}
	}

	// Update rate limit tracker
	m.mu.Lock()
	m.lastTrigger[a.ID] = now
	m.mu.Unlock()

	// Log to history
	value := 0.0
	if v, ok := e.Data["last"].(float64); ok {
		value = v
	} else if v, ok := e.Data["total_pnl"].(float64); ok {
		value = v
	}
	m.store.LogTrigger(a.ID, e.Type, value, msg)
	m.store.UpdateLastTriggered(a.ID, now)

	// One-shot: disable after first trigger
	if a.OneShot {
		m.store.Disable(a.ID)
		m.mu.Lock()
		for i := range m.alerts {
			if m.alerts[i].ID == a.ID {
				m.alerts[i].Enabled = false
				break
			}
		}
		m.mu.Unlock()
	}

	// Publish alert.triggered event
	if m.bus != nil {
		m.bus.Publish(engine.Event{
			Type: "alert.triggered",
			Data: map[string]any{
				"alert_id": a.ID,
				"type":     a.Type,
				"symbol":   a.Symbol,
				"message":  msg,
			},
		})
	}

	// Call registered handlers
	m.mu.RLock()
	handlers := make([]TriggerHandler, len(m.handlers))
	copy(handlers, m.handlers)
	m.mu.RUnlock()

	for _, h := range handlers {
		h(a, msg)
	}
}

func formatEventData(data map[string]any) string {
	if symbol, ok := data["symbol"].(string); ok {
		return symbol
	}
	return "event data"
}
```

**Step 4: Run test to verify it passes**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestManager -v`
Expected: PASS — all 5 tests pass.

**Step 5: Commit**

```bash
git add internal/alert/manager.go internal/alert/manager_test.go
git commit -m "feat(alert): add AlertManager with event evaluation and rate limiting"
```

---

### Task 3: Custom Expression Alerts

**Files:**
- Modify: `internal/alert/manager.go`
- Modify: `internal/alert/manager_test.go`

**Step 1: Write the failing test**

Add to `internal/alert/manager_test.go`:

```go
func TestManager_CustomExpressionAlert(t *testing.T) {
	db := testDB(t)
	defer db.Close()

	// Also create candle_cache table for expression evaluation
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

	// Insert 50 candles for RSI calculation (trending down to make RSI < 30)
	now := time.Now().Unix()
	for i := 0; i < 50; i++ {
		price := 100.0 - float64(i)*1.5 // downtrend
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
```

**Step 2: Run test to verify it fails**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestManager_CustomExpression -v`
Expected: FAIL — `checkCustomAlert` returns false always.

**Step 3: Implement custom expression evaluation**

Modify `internal/alert/manager.go` — update the imports to add `database/sql` and the backtest package, update the `db` field type, and implement `checkCustomAlert`:

Update the `Manager` struct's `db` field from `interface{}` to `*sql.DB`:

```go
import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/backtest"
	"github.com/clawtrade/clawtrade/internal/engine"
)
```

Change `db interface{}` to `db *sql.DB` in the Manager struct. Update `NewManager` parameter similarly.

Replace `checkCustomAlert`:

```go
func (m *Manager) checkCustomAlert(a Alert, e engine.Event) (bool, string) {
	if e.Type != "price.update" {
		return false, ""
	}
	symbol, _ := e.Data["symbol"].(string)
	if a.Symbol != "" && symbol != a.Symbol {
		return false, ""
	}
	if a.Expression == "" {
		return false, ""
	}
	if m.db == nil {
		return false, ""
	}

	// Load recent candles from cache for indicator computation
	candles := m.loadCachedCandles(symbol, "1h", 200)
	if len(candles) < 26 {
		// Not enough data for indicator computation — use price-only evaluation
		price, ok := e.Data["last"].(float64)
		if !ok {
			return false, ""
		}
		// Simple expression with just close/price
		indicators := map[string]float64{"close": price}
		if backtest.EvalExprPublic(a.Expression, indicators) {
			msg := a.Message
			if msg == "" {
				msg = fmt.Sprintf("Custom alert: %s (%s = %.2f)", a.Expression, symbol, price)
			}
			return true, msg
		}
		return false, ""
	}

	indicators := backtest.ComputeIndicatorsPublic(candles)
	// Override close with live price
	if price, ok := e.Data["last"].(float64); ok {
		indicators["close"] = price
	}

	if backtest.EvalExprPublic(a.Expression, indicators) {
		msg := a.Message
		if msg == "" {
			msg = fmt.Sprintf("Custom alert: %s triggered for %s", a.Expression, symbol)
		}
		return true, msg
	}
	return false, ""
}

func (m *Manager) loadCachedCandles(symbol, timeframe string, limit int) []adapter.Candle {
	rows, err := m.db.Query(
		`SELECT timestamp, open, high, low, close, volume FROM candle_cache WHERE symbol = ? AND timeframe = ? ORDER BY timestamp DESC LIMIT ?`,
		symbol, timeframe, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var candles []adapter.Candle
	for rows.Next() {
		var ts int64
		var c adapter.Candle
		if err := rows.Scan(&ts, &c.Open, &c.High, &c.Low, &c.Close, &c.Volume); err != nil {
			continue
		}
		c.Timestamp = time.Unix(ts, 0)
		c.Symbol = symbol
		candles = append(candles, c)
	}

	// Reverse to chronological order
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}
	return candles
}
```

**Step 4: Export expression helpers from backtest package**

Modify `internal/backtest/strategy.go` — add two public wrapper functions at the bottom of the file:

```go
// EvalExprPublic evaluates an expression string against indicator values.
// Exported for use by the alerting system.
func EvalExprPublic(expr string, indicators map[string]float64) bool {
	return evalExpr(expr, indicators)
}

// ComputeIndicatorsPublic computes technical indicators from candles.
// Exported for use by the alerting system.
func ComputeIndicatorsPublic(candles []adapter.Candle) map[string]float64 {
	return computeIndicators(candles)
}
```

**Step 5: Run test to verify it passes**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestManager_CustomExpression -v`
Expected: PASS.

**Step 6: Commit**

```bash
git add internal/alert/manager.go internal/alert/manager_test.go internal/backtest/strategy.go
git commit -m "feat(alert): add custom expression alert evaluation using backtest indicators"
```

---

### Task 4: Telegram Dispatcher

**Files:**
- Create: `internal/alert/dispatcher.go`
- Create: `internal/alert/dispatcher_test.go`

**Step 1: Write the failing test**

Create `internal/alert/dispatcher_test.go`:

```go
package alert

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/clawtrade/clawtrade/internal/bot"
	"github.com/clawtrade/clawtrade/internal/engine"
)

func TestDispatcher_SendsTelegram(t *testing.T) {
	var received []string
	var mu sync.Mutex

	// Mock Telegram API
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req bot.SendMessageRequest
		json.NewDecoder(r.Body).Decode(&req)
		mu.Lock()
		received = append(received, req.Text)
		mu.Unlock()
		w.WriteHeader(200)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer ts.Close()

	tgBot := bot.NewTelegramBot("test-token")
	tgBot.SetBaseURL(ts.URL)

	db := testDB(t)
	defer db.Close()
	store := NewStore(db)
	bus := engine.NewEventBus()

	d := NewDispatcher(tgBot, 12345, bus)

	a := Alert{ID: 1, Type: AlertTypePrice, Symbol: "BTC/USDT"}
	d.Dispatch(a, "BTC above 70k!")

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if len(received) != 1 {
		t.Fatalf("expected 1 Telegram message, got %d", len(received))
	}
	if received[0] == "" {
		t.Fatal("empty message")
	}
	mu.Unlock()
}

func TestDispatcher_PublishesEvent(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	bus := engine.NewEventBus()

	var triggered bool
	var mu sync.Mutex
	bus.Subscribe("alert.triggered", func(e engine.Event) {
		mu.Lock()
		triggered = true
		mu.Unlock()
	})

	// No Telegram bot — should still publish event
	d := NewDispatcher(nil, 0, bus)
	a := Alert{ID: 1, Type: AlertTypePrice, Symbol: "BTC/USDT"}
	d.Dispatch(a, "test message")

	time.Sleep(50 * time.Millisecond)
	mu.Lock()
	if !triggered {
		t.Fatal("expected alert.triggered event")
	}
	mu.Unlock()
}
```

**Step 2: Run test to verify it fails**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestDispatcher -v`
Expected: FAIL — `NewDispatcher` not defined.

**Step 3: Write implementation**

Create `internal/alert/dispatcher.go`:

```go
package alert

import (
	"fmt"
	"strconv"

	"github.com/clawtrade/clawtrade/internal/bot"
	"github.com/clawtrade/clawtrade/internal/engine"
)

// Dispatcher sends alert notifications via Telegram and EventBus.
type Dispatcher struct {
	tgBot  *bot.TelegramBot
	chatID int64
	bus    *engine.EventBus
}

// NewDispatcher creates a new alert dispatcher.
func NewDispatcher(tgBot *bot.TelegramBot, chatID int64, bus *engine.EventBus) *Dispatcher {
	return &Dispatcher{
		tgBot:  tgBot,
		chatID: chatID,
		bus:    bus,
	}
}

// Dispatch sends an alert notification via all configured channels.
func (d *Dispatcher) Dispatch(a Alert, message string) {
	// Format the message
	formatted := d.formatMessage(a, message)

	// Send via Telegram
	if d.tgBot != nil && d.chatID != 0 {
		go func() {
			d.tgBot.SendMessage(bot.SendMessageRequest{
				ChatID: d.chatID,
				Text:   formatted,
			})
		}()
	}

	// Publish alert.triggered event to bus (for WebSocket clients)
	if d.bus != nil {
		d.bus.Publish(engine.Event{
			Type: "alert.triggered",
			Data: map[string]any{
				"alert_id": a.ID,
				"type":     a.Type,
				"symbol":   a.Symbol,
				"message":  message,
			},
		})
	}
}

func (d *Dispatcher) formatMessage(a Alert, message string) string {
	icon := alertIcon(a.Type)
	return fmt.Sprintf("%s %s\n%s", icon, alertTitle(a.Type), message)
}

func alertIcon(alertType string) string {
	switch alertType {
	case AlertTypePrice:
		return "\xf0\x9f\x93\x88" // chart_with_upwards_trend
	case AlertTypePnL:
		return "\xf0\x9f\x92\xb0" // money_bag
	case AlertTypeRisk:
		return "\xe2\x9a\xa0\xef\xb8\x8f" // warning
	case AlertTypeTrade:
		return "\xf0\x9f\x94\x84" // arrows_counterclockwise
	case AlertTypeSystem:
		return "\xf0\x9f\x94\xa7" // wrench
	case AlertTypeCustom:
		return "\xf0\x9f\x94\x94" // bell
	default:
		return "\xf0\x9f\x94\x94" // bell
	}
}

func alertTitle(alertType string) string {
	switch alertType {
	case AlertTypePrice:
		return "Price Alert"
	case AlertTypePnL:
		return "PnL Alert"
	case AlertTypeRisk:
		return "Risk Alert"
	case AlertTypeTrade:
		return "Trade Alert"
	case AlertTypeSystem:
		return "System Alert"
	case AlertTypeCustom:
		return "Custom Alert"
	default:
		return "Alert"
	}
}

// ParseChatID converts a string chat ID to int64.
func ParseChatID(s string) int64 {
	id, _ := strconv.ParseInt(s, 10, 64)
	return id
}
```

**Step 4: Run test to verify it passes**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestDispatcher -v`
Expected: PASS.

**Step 5: Commit**

```bash
git add internal/alert/dispatcher.go internal/alert/dispatcher_test.go
git commit -m "feat(alert): add Telegram dispatcher for alert notifications"
```

---

### Task 5: Agent Tools (create_alert, list_alerts, delete_alert)

**Files:**
- Modify: `internal/agent/tools.go`

**Step 1: Add alert manager to ToolRegistry**

Modify `internal/agent/tools.go`:

Add `alertMgr` field to `ToolRegistry` struct:

```go
type ToolRegistry struct {
	adapters   map[string]adapter.TradingAdapter
	riskEngine *risk.Engine
	mcpBridge  MCPBridge
	bus        *engine.EventBus
	db         *sql.DB
	alertMgr   AlertManager
}
```

Add the `AlertManager` interface (above `ToolRegistry`):

```go
// AlertManager is the interface for alert management used by tools.
type AlertManager interface {
	AddAlert(a *alert.Alert) (int64, error)
	RemoveAlert(id int64) error
	ListAlerts() ([]alert.Alert, error)
}
```

Wait — to avoid import cycles, use an interface approach. Add this interface to `tools.go`:

```go
// AlertService is the interface for managing alerts from agent tools.
type AlertService interface {
	CreateAlert(alertType, symbol, condition string, threshold float64, expression, message string, oneShot bool) (int64, error)
	DeleteAlert(id int64) error
	ListAlerts() ([]map[string]any, error)
}
```

Add to `ToolRegistry`:

```go
alertSvc AlertService
```

Add `SetAlertService` method:

```go
func (tr *ToolRegistry) SetAlertService(svc AlertService) {
	tr.alertSvc = svc
}
```

**Step 2: Add tool definitions**

Add 3 new tool definitions to the `defs` slice in `Definitions()`:

```go
{
	Name:        "create_alert",
	Description: "Create a new alert. Supports price alerts (above/below threshold), PnL alerts, and custom expression rules (e.g. 'rsi < 30 AND close > sma_50').",
	InputSchema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"alert_type": map[string]any{"type": "string", "enum": []string{"price", "pnl", "risk", "trade", "system", "custom"}, "description": "Type of alert"},
			"symbol":     map[string]any{"type": "string", "description": "Trading pair (e.g. BTC/USDT). Required for price/custom alerts."},
			"condition":  map[string]any{"type": "string", "enum": []string{"above", "below", "expression"}, "description": "Alert condition", "default": "above"},
			"threshold":  map[string]any{"type": "number", "description": "Price or PnL threshold value"},
			"expression": map[string]any{"type": "string", "description": "Custom rule expression (e.g. 'rsi < 30 AND close > sma_50'). Only for custom type."},
			"message":    map[string]any{"type": "string", "description": "Custom alert message"},
			"one_shot":   map[string]any{"type": "boolean", "description": "Auto-disable after first trigger (default: false)", "default": false},
		},
		"required": []string{"alert_type"},
	},
},
{
	Name:        "list_alerts",
	Description: "List all active alerts with their status, conditions, and last trigger time.",
	InputSchema: map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	},
},
{
	Name:        "delete_alert",
	Description: "Delete an alert by its ID.",
	InputSchema: map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id": map[string]any{"type": "integer", "description": "Alert ID to delete"},
		},
		"required": []string{"id"},
	},
},
```

**Step 3: Add to builtinTools map**

```go
var builtinTools = map[string]bool{
	"get_price": true, "get_candles": true, "analyze_market": true,
	"get_balances": true, "get_positions": true, "risk_check": true,
	"calculate_position_size": true, "place_order": true,
	"cancel_order": true, "get_open_orders": true, "backtest": true,
	"create_alert": true, "list_alerts": true, "delete_alert": true,
}
```

**Step 4: Add Execute cases**

Add to the `switch` in `Execute()`:

```go
case "create_alert":
	return tr.execCreateAlert(ctx, call)
case "list_alerts":
	return tr.execListAlerts(ctx, call)
case "delete_alert":
	return tr.execDeleteAlert(ctx, call)
```

**Step 5: Implement tool execution methods**

Add to `tools.go`:

```go
func (tr *ToolRegistry) execCreateAlert(ctx context.Context, call ToolCall) ToolResult {
	if tr.alertSvc == nil {
		return ToolResult{ID: call.ID, Content: "alert service not configured", IsError: true}
	}

	alertType := getString(call.Input, "alert_type", "price")
	symbol := getString(call.Input, "symbol", "")
	condition := getString(call.Input, "condition", "above")
	threshold := getFloat(call.Input, "threshold", 0)
	expression := getString(call.Input, "expression", "")
	message := getString(call.Input, "message", "")
	oneShot := false
	if v, ok := call.Input["one_shot"].(bool); ok {
		oneShot = v
	}

	if alertType == "custom" {
		condition = "expression"
	}

	id, err := tr.alertSvc.CreateAlert(alertType, symbol, condition, threshold, expression, message, oneShot)
	if err != nil {
		return ToolResult{ID: call.ID, Content: fmt.Sprintf("failed to create alert: %v", err), IsError: true}
	}

	result := map[string]any{
		"id":      id,
		"type":    alertType,
		"symbol":  symbol,
		"status":  "created",
		"message": fmt.Sprintf("Alert #%d created successfully", id),
	}
	data, _ := json.Marshal(result)
	return ToolResult{ID: call.ID, Content: string(data)}
}

func (tr *ToolRegistry) execListAlerts(ctx context.Context, call ToolCall) ToolResult {
	if tr.alertSvc == nil {
		return ToolResult{ID: call.ID, Content: "alert service not configured", IsError: true}
	}

	alerts, err := tr.alertSvc.ListAlerts()
	if err != nil {
		return ToolResult{ID: call.ID, Content: fmt.Sprintf("failed to list alerts: %v", err), IsError: true}
	}

	if len(alerts) == 0 {
		return ToolResult{ID: call.ID, Content: "No active alerts."}
	}

	// Format as text + JSON
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Active Alerts (%d):\n", len(alerts)))
	for _, a := range alerts {
		sb.WriteString(fmt.Sprintf("  #%v [%v] %v %v %v",
			a["id"], a["type"], a["symbol"], a["condition"], a["threshold"]))
		if expr, ok := a["expression"].(string); ok && expr != "" {
			sb.WriteString(fmt.Sprintf(" expr=%s", expr))
		}
		if msg, ok := a["message"].(string); ok && msg != "" {
			sb.WriteString(fmt.Sprintf(" — %s", msg))
		}
		sb.WriteString("\n")
	}
	data, _ := json.Marshal(alerts)
	sb.WriteString("\n")
	sb.Write(data)

	return ToolResult{ID: call.ID, Content: sb.String()}
}

func (tr *ToolRegistry) execDeleteAlert(ctx context.Context, call ToolCall) ToolResult {
	if tr.alertSvc == nil {
		return ToolResult{ID: call.ID, Content: "alert service not configured", IsError: true}
	}

	id := int64(getInt(call.Input, "id", 0))
	if id == 0 {
		return ToolResult{ID: call.ID, Content: "alert ID is required", IsError: true}
	}

	if err := tr.alertSvc.DeleteAlert(id); err != nil {
		return ToolResult{ID: call.ID, Content: fmt.Sprintf("failed to delete alert: %v", err), IsError: true}
	}

	return ToolResult{ID: call.ID, Content: fmt.Sprintf(`{"status":"deleted","id":%d}`, id)}
}
```

**Step 6: Add `create_alert` to ChatPanel tool labels**

Modify `web/src/components/ChatPanel.tsx` — add to `TOOL_LABELS`:

```typescript
create_alert: 'Creating alert',
list_alerts: 'Listing alerts',
delete_alert: 'Deleting alert',
```

**Step 7: Commit**

```bash
git add internal/agent/tools.go web/src/components/ChatPanel.tsx
git commit -m "feat(alert): add create_alert, list_alerts, delete_alert agent tools"
```

---

### Task 6: AlertService Adapter & Wire-up in serve()

**Files:**
- Create: `internal/alert/service.go`
- Modify: `internal/config/config.go`
- Modify: `cmd/clawtrade/main.go`
- Modify: `internal/api/server.go`

**Step 1: Create AlertService adapter**

Create `internal/alert/service.go` — implements the `AlertService` interface from tools.go:

```go
package alert

// Service implements the AlertService interface for agent tools.
type Service struct {
	mgr *Manager
}

// NewService wraps a Manager as an AlertService.
func NewService(mgr *Manager) *Service {
	return &Service{mgr: mgr}
}

// CreateAlert creates a new alert via the manager.
func (s *Service) CreateAlert(alertType, symbol, condition string, threshold float64, expression, message string, oneShot bool) (int64, error) {
	a := &Alert{
		Type:       alertType,
		Symbol:     symbol,
		Condition:  condition,
		Threshold:  threshold,
		Expression: expression,
		Message:    message,
		Enabled:    true,
		OneShot:    oneShot,
	}
	return s.mgr.AddAlert(a)
}

// DeleteAlert removes an alert via the manager.
func (s *Service) DeleteAlert(id int64) error {
	return s.mgr.RemoveAlert(id)
}

// ListAlerts returns all alerts as maps for the agent tool.
func (s *Service) ListAlerts() ([]map[string]any, error) {
	alerts, err := s.mgr.store.ListAll()
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for _, a := range alerts {
		m := map[string]any{
			"id":        a.ID,
			"type":      a.Type,
			"symbol":    a.Symbol,
			"condition": a.Condition,
			"threshold": a.Threshold,
			"enabled":   a.Enabled,
			"one_shot":  a.OneShot,
		}
		if a.Expression != "" {
			m["expression"] = a.Expression
		}
		if a.Message != "" {
			m["message"] = a.Message
		}
		if a.LastTriggeredAt != nil {
			m["last_triggered"] = a.LastTriggeredAt.Format("2006-01-02 15:04:05")
		}
		result = append(result, m)
	}
	return result, nil
}
```

**Step 2: Extend AlertsConfig**

Modify `internal/config/config.go` — add new fields to `AlertsConfig`:

```go
type AlertsConfig struct {
	TradeExecuted    bool `yaml:"trade_executed"`
	RiskAlert        bool `yaml:"risk_alert"`
	PnlUpdate        bool `yaml:"pnl_update"`
	SystemAlert      bool `yaml:"system_alert"`
	RateLimitMinutes int  `yaml:"rate_limit_minutes"`
	DailyBriefing    bool `yaml:"daily_briefing"`
	BriefingHourUTC  int  `yaml:"briefing_hour_utc"`
}
```

Update `defaultConfig()` — add defaults in the `AlertsConfig`:

```go
Alerts: AlertsConfig{
	TradeExecuted:    true,
	RiskAlert:        true,
	PnlUpdate:        false,
	SystemAlert:      true,
	RateLimitMinutes: 5,
	DailyBriefing:    false,
	BriefingHourUTC:  8,
},
```

Add new fields to `setNotificationField` under `case "alerts"`:

```go
case "rate_limit_minutes":
	v, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("must be an integer: %w", err)
	}
	c.Notifications.Alerts.RateLimitMinutes = v
case "daily_briefing":
	c.Notifications.Alerts.DailyBriefing = value == "true" || value == "1"
case "briefing_hour_utc":
	v, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("must be an integer: %w", err)
	}
	c.Notifications.Alerts.BriefingHourUTC = v
```

**Step 3: Wire up in serve()**

Modify `cmd/clawtrade/main.go` — add alert import and initialization in `serve()`.

Add import:

```go
"github.com/clawtrade/clawtrade/internal/alert"
"github.com/clawtrade/clawtrade/internal/bot"
```

After the portfolio poller start block (around line 396), add:

```go
// Start alert manager
alertStore := alert.NewStore(db)
alertMgr := alert.NewManager(alertStore, bus, db, alert.ManagerConfig{
	RateLimitMinutes: cfg.Notifications.Alerts.RateLimitMinutes,
})

// Set up Telegram dispatcher
var tgBot *bot.TelegramBot
if cfg.Notifications.Telegram.Enabled && cfg.Notifications.Telegram.Token != "" {
	tgBot = bot.NewTelegramBot(cfg.Notifications.Telegram.Token)
	tgBot.RegisterDefaultCommands()
	chatID := alert.ParseChatID(cfg.Notifications.Telegram.ChatID)
	if chatID != 0 {
		tgBot.AllowChat(chatID)
	}
	dispatcher := alert.NewDispatcher(tgBot, chatID, bus)
	alertMgr.OnTrigger(func(a alert.Alert, msg string) {
		dispatcher.Dispatch(a, msg)
	})
	fmt.Println("Alerts: Telegram dispatcher enabled")
}

alertMgr.Start()

// Set alert service on agent tools
alertSvc := alert.NewService(alertMgr)
```

Then pass `alertSvc` to the server. Modify `api.NewServer` to accept the alert service, or set it after creation. Easiest approach: add `SetAlertService` to the server which passes it through to the agent's tool registry.

In `internal/api/server.go`, add import for alert service. In `NewServer`, after creating the agent `ag`, add:

After `ag := agent.New(...)` and before `llm := NewLLMHandler(ag)`:

The agent needs a `SetAlertService` method. Add to `internal/agent/agent.go`:

```go
func (a *Agent) SetAlertService(svc interface{}) {
	if s, ok := svc.(AlertService); ok {
		a.tools.SetAlertService(s)
	}
}
```

Then in `cmd/clawtrade/main.go` after creating alertSvc, before `srv.Start()`:

Actually, it's simpler to restructure. Let's have the server accept the agent and set it up:

In `cmd/clawtrade/main.go`, after `alertSvc := alert.NewService(alertMgr)`:

```go
srv.SetAlertService(alertSvc)
```

Add `SetAlertService` to `Server` in `internal/api/server.go`:

```go
func (s *Server) SetAlertService(svc interface{}) {
	// This will be called after construction, passing through to the agent
}
```

Actually, the cleanest approach: since `agent.New()` is called inside `NewServer`, we need to either pass alertSvc to NewServer or set it after. Let's set it after via the agent.

Add to `internal/api/server.go`:

```go
// agent field in Server struct
agent *agent.Agent
```

Store the agent reference in NewServer: `s.agent = ag` (before returning).

Add method:

```go
func (s *Server) SetAlertService(svc interface{}) {
	s.agent.SetAlertService(svc)
}
```

In `cmd/clawtrade/main.go`:

```go
srv.SetAlertService(alertSvc)
```

**Step 4: Add `alert.*` to WebSocket subscriptions**

Modify `internal/api/server.go` — add `"alert.*"` to the hub subscription list:

```go
hub.SubscribeToEvents(bus, []string{
	"market.*",
	"trade.*",
	"risk.*",
	"system.*",
	"price.*",
	"agent.*",
	"portfolio.*",
	"backtest.*",
	"alert.*",
})
```

**Step 5: Build and verify**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go build ./...`
Expected: Build succeeds.

**Step 6: Commit**

```bash
git add internal/alert/service.go internal/config/config.go internal/api/server.go internal/agent/agent.go internal/agent/tools.go cmd/clawtrade/main.go
git commit -m "feat(alert): wire up AlertManager, Telegram dispatcher, and agent tools in serve()"
```

---

### Task 7: Daily Briefing

**Files:**
- Create: `internal/alert/briefing.go`
- Create: `internal/alert/briefing_test.go`
- Modify: `cmd/clawtrade/main.go`

**Step 1: Write the failing test**

Create `internal/alert/briefing_test.go`:

```go
package alert

import (
	"testing"

	"github.com/clawtrade/clawtrade/internal/adapter"
)

type mockBriefingAdapter struct {
	balances  []adapter.Balance
	positions []adapter.Position
}

func (m *mockBriefingAdapter) GetBalances(ctx interface{}) ([]adapter.Balance, error) {
	return m.balances, nil
}

func (m *mockBriefingAdapter) GetPositions(ctx interface{}) ([]adapter.Position, error) {
	return m.positions, nil
}

func TestFormatBriefing(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewStore(db)

	// Log some alerts today
	store.Create(&Alert{Type: AlertTypePrice, Symbol: "BTC/USDT", Condition: CondAbove, Threshold: 70000, Enabled: true})
	store.LogTrigger(1, "price.update", 71000, "BTC above 70k")

	history, _ := store.TodayHistory()

	msg := FormatBriefing(10500.0, 3, len(history))
	if msg == "" {
		t.Fatal("expected non-empty briefing")
	}
	if !containsStr(msg, "10,500") {
		t.Errorf("expected portfolio value in briefing, got: %s", msg)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

**Step 2: Run test to verify it fails**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestFormatBriefing -v`
Expected: FAIL — `FormatBriefing` not defined.

**Step 3: Write implementation**

Create `internal/alert/briefing.go`:

```go
package alert

import (
	"context"
	"fmt"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/bot"
)

// BriefingConfig holds daily briefing configuration.
type BriefingConfig struct {
	HourUTC  int
	TgBot    *bot.TelegramBot
	ChatID   int64
	Adapters map[string]adapter.TradingAdapter
	Store    *Store
}

// StartDailyBriefing launches a goroutine that sends a daily briefing.
func StartDailyBriefing(ctx context.Context, cfg BriefingConfig) {
	go func() {
		for {
			now := time.Now().UTC()
			next := time.Date(now.Year(), now.Month(), now.Day(), cfg.HourUTC, 0, 0, 0, time.UTC)
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			wait := next.Sub(now)

			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
				sendBriefing(ctx, cfg)
			}
		}
	}()
}

func sendBriefing(ctx context.Context, cfg BriefingConfig) {
	// Aggregate portfolio value
	totalValue := 0.0
	activeAlerts := 0

	for _, adp := range cfg.Adapters {
		if !adp.IsConnected() {
			continue
		}
		balances, err := adp.GetBalances(ctx)
		if err != nil {
			continue
		}
		for _, b := range balances {
			totalValue += b.Total
		}
	}

	// Count active alerts
	if alerts, err := cfg.Store.ListEnabled(); err == nil {
		activeAlerts = len(alerts)
	}

	// Count today's triggered alerts
	triggeredToday := 0
	if history, err := cfg.Store.TodayHistory(); err == nil {
		triggeredToday = len(history)
	}

	msg := FormatBriefing(totalValue, activeAlerts, triggeredToday)

	if cfg.TgBot != nil && cfg.ChatID != 0 {
		cfg.TgBot.SendMessage(bot.SendMessageRequest{
			ChatID: cfg.ChatID,
			Text:   msg,
		})
	}
}

// FormatBriefing creates the daily briefing message text.
func FormatBriefing(portfolioValue float64, activeAlerts, triggeredToday int) string {
	date := time.Now().UTC().Format("2006-01-02")
	return fmt.Sprintf(
		"\xf0\x9f\x93\xb0 Daily Briefing — %s\n\n"+
			"\xf0\x9f\x92\xbc Portfolio: $%s\n"+
			"\xf0\x9f\x94\x94 Active Alerts: %d\n"+
			"\xe2\x9a\xa1 Triggered Today: %d",
		date,
		formatMoney(portfolioValue),
		activeAlerts,
		triggeredToday,
	)
}

func formatMoney(v float64) string {
	if v >= 1000000 {
		return fmt.Sprintf("%.2fM", v/1000000)
	}
	if v >= 1000 {
		whole := int(v)
		frac := int((v - float64(whole)) * 100)
		// Simple thousands separator
		s := fmt.Sprintf("%d", whole)
		if len(s) > 3 {
			s = s[:len(s)-3] + "," + s[len(s)-3:]
		}
		return fmt.Sprintf("%s.%02d", s, frac)
	}
	return fmt.Sprintf("%.2f", v)
}
```

**Step 4: Run test to verify it passes**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -run TestFormatBriefing -v`
Expected: PASS.

**Step 5: Wire up in serve()**

Modify `cmd/clawtrade/main.go` — after `alertMgr.Start()`, add:

```go
// Start daily briefing if configured
if cfg.Notifications.Alerts.DailyBriefing && tgBot != nil {
	chatID := alert.ParseChatID(cfg.Notifications.Telegram.ChatID)
	alert.StartDailyBriefing(ctx, alert.BriefingConfig{
		HourUTC:  cfg.Notifications.Alerts.BriefingHourUTC,
		TgBot:    tgBot,
		ChatID:   chatID,
		Adapters: adapters,
		Store:    alertStore,
	})
	fmt.Printf("Daily briefing: enabled at %02d:00 UTC\n", cfg.Notifications.Alerts.BriefingHourUTC)
}
```

**Step 6: Commit**

```bash
git add internal/alert/briefing.go internal/alert/briefing_test.go cmd/clawtrade/main.go
git commit -m "feat(alert): add daily briefing via Telegram"
```

---

### Task 8: Build, Test All, Final Verification

**Files:**
- All files from Tasks 1-7

**Step 1: Run all alert package tests**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/alert/ -v`
Expected: All tests PASS.

**Step 2: Run full project build**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go build ./...`
Expected: Build succeeds.

**Step 3: Run existing tests to ensure no regressions**

Run: `export PATH="/c/go/bin:$PATH" && cd /d/Clawtrade && go test ./internal/... -v`
Expected: All tests PASS.

**Step 4: Build frontend**

Run: `cd /d/Clawtrade/web && npm run build`
Expected: Build succeeds.

**Step 5: Final commit (if any fixes needed)**

```bash
git add -A
git commit -m "feat(alert): alerting system complete — price, PnL, risk, custom rules, Telegram, daily briefing"
```
