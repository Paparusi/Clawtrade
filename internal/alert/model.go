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
