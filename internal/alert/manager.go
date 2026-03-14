package alert

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/clawtrade/clawtrade/internal/adapter"
	"github.com/clawtrade/clawtrade/internal/backtest"
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
	mu          sync.RWMutex
	store       *Store
	bus         *engine.EventBus
	db          *sql.DB
	alerts      []Alert
	handlers    []TriggerHandler
	config      ManagerConfig
	lastTrigger map[int64]time.Time
}

// NewManager creates a new AlertManager.
func NewManager(store *Store, bus *engine.EventBus, db *sql.DB, cfg ManagerConfig) *Manager {
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
		// No DB — use price-only evaluation
		price, ok := e.Data["last"].(float64)
		if !ok {
			return false, ""
		}
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

	// Load recent candles from cache for indicator computation
	candles := m.loadCachedCandles(symbol, "1h", 200)
	if len(candles) < 26 {
		// Not enough data — use price-only
		price, ok := e.Data["last"].(float64)
		if !ok {
			return false, ""
		}
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
		candles = append(candles, c)
	}

	// Reverse to chronological order
	for i, j := 0, len(candles)-1; i < j; i, j = i+1, j-1 {
		candles[i], candles[j] = candles[j], candles[i]
	}
	return candles
}

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
